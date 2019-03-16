package client

import (
	"io/ioutil"
	"log"

	"fmt"
	"os"

	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/manifest"
	manifestV1 "github.com/docker/distribution/manifest/schema1"
	"github.com/vbaksa/promoter/progressbar"

	"github.com/docker/libtrust"
	"github.com/dustin/go-humanize"
	"github.com/heroku/docker-registry-client/registry"
	"gopkg.in/cheggaaa/pb.v1"
)

//Promote holds single image promotions structure
type Promote struct {
	SrcRegistry  string
	SrcImage     string
	SrcImageTag  string
	SrcUsername  string
	SrcPassword  string
	SrcInsecure  bool
	DestRegistry string
	DestImage    string
	DestImageTag string
	DestUsername string
	DestPassword string
	DestInsecure bool
	Debug        bool
}

//PromoteImage executes single image promotion
func (pr *Promote) PromoteImage() {
	if !pr.Debug {
		log.SetOutput(ioutil.Discard)
	}
	fmt.Println("Connecting to Source registry")
	var srcHub *registry.Registry
	var destHub *registry.Registry
	var err error
	if pr.SrcInsecure {
		srcHub, err = registry.NewInsecure(pr.SrcRegistry, pr.SrcUsername, pr.SrcPassword)

	} else {
		srcHub, err = registry.New(pr.SrcRegistry, pr.SrcUsername, pr.SrcPassword)

	}
	if err != nil {
		fmt.Println("Cannot connect to registry: " + pr.SrcRegistry)
		fmt.Println("Connection error: " + err.Error())
		os.Exit(1)
	} else {
		fmt.Println("Connected to Source registry")
	}
	if pr.DestInsecure {
		destHub, err = registry.NewInsecure(pr.DestRegistry, pr.DestUsername, pr.DestPassword)

	} else {
		destHub, err = registry.New(pr.DestRegistry, pr.DestUsername, pr.DestPassword)
	}
	fmt.Println("Connecting to Destination registry")
	if err != nil {
		fmt.Println("Cannot connect to registry: " + pr.DestRegistry)
		fmt.Println("Connection error: " + err.Error())
		os.Exit(1)
	} else {
		fmt.Println("Connected to Destination registry")
	}

	srcManifest, err := srcHub.Manifest(pr.SrcImage, pr.SrcImageTag)
	if err != nil {
		fmt.Println("Failed to download Source Image manifest. Error: " + err.Error())
		os.Exit(1)
	}

	srcLayers := srcManifest.FSLayers
	fmt.Println("Optimising upload...")
	uploadLayer := pr.layerExists(destHub, srcHub, srcLayers)
	totalDownloadSize := pr.getTotalDownloadSize(srcHub, uploadLayer)
	if len(uploadLayer) > 0 {
		fmt.Println()
		fmt.Printf("Going to upload around %s of layer data. Expected network bandwidth: %s \n", humanize.Bytes(uint64(totalDownloadSize)), humanize.Bytes(uint64(totalDownloadSize*2)))
		fmt.Println()

		fmt.Println()
		fmt.Println("Uploading layers")
		fmt.Println()

		done := make(chan bool)
		var totalReader = make(chan int64)
		for _, layer := range uploadLayer {
			//srcHub.DownloadLayer(src)
			go func(layer digest.Digest) {
				pr.uploadLayer(destHub, srcHub, layer, &totalReader)
				done <- true
			}(layer)
		}
		bar := pb.New64(totalDownloadSize * 2).SetUnits(pb.U_BYTES)
		bar.Start()
		go func() {
			for {
				t := <-totalReader
				bar.Add64(t * 2)
			}
		}()

		for i := 0; i < len(uploadLayer); i++ {
			<-done
		}
		bar.Finish()

		fmt.Println("Finished uploading layers")
	}
	fmt.Println("Generating Signing Key...")
	key, err := libtrust.GenerateECP256PrivateKey()
	if err != nil {
		fmt.Println("Error occurred while generating Image Key")
		fmt.Println("Error: " + err.Error())
		os.Exit(1)
	}
	fmt.Println("Signing Image Manifest...")
	destManifest := srcManifest
	destManifest.Name = pr.DestImage
	destManifest.Tag = pr.DestImageTag

	manifest := &manifestV1.Manifest{
		Versioned: manifest.Versioned{
			SchemaVersion: 1,
		},
		Name:         pr.DestImage,
		Tag:          pr.DestImageTag,
		Architecture: destManifest.Architecture,
		FSLayers:     destManifest.FSLayers,
		History:      destManifest.History,
	}
	signedManifest, err := manifestV1.Sign(manifest, key)
	if err != nil {
		fmt.Println("Error occurred while Signing Image Manifest")
		fmt.Println("Error: " + err.Error())
		os.Exit(1)
	}

	fmt.Println("Submitting Image Manifest")
	err = destHub.PutManifest(pr.DestImage, pr.DestImageTag, signedManifest)

	if err != nil {
		fmt.Println("Manifest update error: " + err.Error())
		os.Exit(1)
	}
	fmt.Println("Push Complete")
	os.Exit(0)
}

type layerCheckResult struct {
	Err     error
	Exists  *existingLayer
	Missing *missingLayer
}
type missingLayer struct {
	Blob digest.Digest
}
type existingLayer struct {
	Descriptor distribution.Descriptor
}

func (pr *Promote) getTotalDownloadSize(srcHub *registry.Registry, uploadLayer []digest.Digest) int64 {
	result := make(chan distribution.Descriptor)
	for _, layer := range uploadLayer {
		go func(layer digest.Digest) {
			l, err := srcHub.LayerMetadata(pr.SrcImage, layer)
			if err != nil {
				fmt.Println("Error while inspecting layer: " + layer)
				os.Exit(1)
			}
			result <- l
		}(layer)
	}
	var total int64
	for i := 0; i < len(uploadLayer); i++ {
		r := <-result
		total = total + r.Size
	}
	return total
}

func (pr *Promote) layerExists(destHub *registry.Registry, srcHub *registry.Registry, srcLayers []manifestV1.FSLayer) []digest.Digest {

	//Layers array returned by function
	results := make([]digest.Digest, 0)
	//How much space we will save by skipping already existing layers
	var totalSaved int64
	//Temporary result channel
	result := make(chan *layerCheckResult)

	// check each layer on remote hub
	for _, layer := range srcLayers {
		go func(layer manifestV1.FSLayer, result chan *layerCheckResult) {

			layerMetada, err := destHub.LayerMetadata(pr.DestImage, layer.BlobSum)
			if err != nil {
				// Layer does not exist
				//	fmt.Println("Layer does not exist: " + layer.BlobSum)
				checkResult := &layerCheckResult{
					Err: err,
					Missing: &missingLayer{
						Blob: layer.BlobSum,
					},
				}
				result <- checkResult

			} else {
				// Layer exists
				fmt.Println("Layer already exists on Remote Registry: " + layerMetada.Digest)
				checkResult := &layerCheckResult{
					Err: nil,
					Exists: &existingLayer{
						Descriptor: layerMetada,
					},
				}
				result <- checkResult
			}
		}(layer, result)
	}

	// Wait for result (each layer check)
	for i := 0; i < len(srcLayers); i++ {
		res := <-result
		// If we got result about missing layer on remote registry, else check result about existing layer
		if res.Missing != nil {
			//iterate over existing array and make sure that there is no duplicate layers
			var e = false
			for _, r := range results {
				if r == res.Missing.Blob {
					e = true
				}
			}
			//does not exist on array
			if !e {
				results = append(results, res.Missing.Blob)
			}

		} else {
			totalSaved = totalSaved + res.Exists.Descriptor.Size
		}
	}
	fmt.Println()
	if totalSaved > 100 {
		fmt.Printf("Some layers already exist on Remote Registry. Skipping around %s of layer data. Total network bandwidth saved: %s \n", humanize.Bytes(uint64(totalSaved)), humanize.Bytes(uint64(totalSaved*2)))
	}
	fmt.Println()

	return results
}

func (pr *Promote) uploadLayer(destHub *registry.Registry, srcHub *registry.Registry, layer digest.Digest, totalReader *chan int64) {
	reader, err := srcHub.DownloadLayer(pr.SrcImage, layer)
	defer reader.Close()
	rd := &progressbar.PassThru{ReadCloser: reader, Total: totalReader}
	destHub.UploadLayer(pr.DestImage, layer, rd)
	if err != nil {
		fmt.Println("Error occurred while uploading layer: " + layer)
		os.Exit(1)
	}
}
