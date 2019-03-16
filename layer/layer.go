package layer

import (
	"fmt"
	"os"

	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	manifestV1 "github.com/docker/distribution/manifest/schema1"
	humanize "github.com/dustin/go-humanize"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/vbaksa/promoter/progressbar"
)

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

//MissingLayers computes list of layers required to be uploaded. Upload is optimized by skipping existing layers
func MissingLayers(destHub *registry.Registry, destImage string, srcLayers []manifestV1.FSLayer) []digest.Digest {

	//Layers array returned by function
	results := make([]digest.Digest, 0)
	//How much space we will save by skipping already existing layers
	var totalSaved int64
	//Temporary result channel
	result := make(chan *layerCheckResult)

	// check each layer on remote hub
	for _, layer := range srcLayers {
		go func(layer manifestV1.FSLayer, result chan *layerCheckResult) {

			layerMetada, err := destHub.LayerMetadata(destImage, layer.BlobSum)
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

//DigestSize returns total upload size
func DigestSize(srcHub *registry.Registry, srcImage string, uploadLayer []digest.Digest) int64 {
	result := make(chan distribution.Descriptor)
	for _, layer := range uploadLayer {
		go func(layer digest.Digest) {
			l, err := srcHub.LayerMetadata(srcImage, layer)
			if err != nil {
				fmt.Println("Error while inspecting layer: " + layer)
				fmt.Println("Error: " + err.Error())
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

//UploadLayer uploads image layer with option to track upload progress
func UploadLayer(destHub *registry.Registry, destImage string, srcHub *registry.Registry, srcImage string, layer digest.Digest) {
	UploadLayerWithProgress(destHub, destImage, srcHub, srcImage, layer, nil)
}

//UploadLayerWithProgress uploads image layer with option to track upload progress
func UploadLayerWithProgress(destHub *registry.Registry, destImage string, srcHub *registry.Registry, srcImage string, layer digest.Digest, totalReader *chan int64) {
	reader, err := srcHub.DownloadLayer(srcImage, layer)
	defer reader.Close()
	if totalReader != nil {
		rd := &progressbar.PassThru{ReadCloser: reader, Total: totalReader}
		destHub.UploadLayer(destImage, layer, rd)
	} else {
		destHub.UploadLayer(destImage, layer, reader)
	}
	if err != nil {
		fmt.Println("Error occurred while uploading layer: " + layer)
		fmt.Println("Error: " + err.Error())
		os.Exit(1)
	}
}
