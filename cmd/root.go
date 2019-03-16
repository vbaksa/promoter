package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"os"

	"github.com/vbaksa/promoter/image"
	"github.com/vbaksa/promoter/tags"

	"errors"

	"github.com/spf13/cobra"
)

var (
	version = "DEV"
)

// RootCmd provides CLI handler for the application
var RootCmd = &cobra.Command{
	Use:   "promoter",
	Short: "Promotes Docker images",
	Long: `Promotes Docker images from one Registry into another.
                Optimizes network traffic by inspecting existing image data.`,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(1)
	},
}

func init() {
	//optional parameters
	var srcUsername string
	var srcPassword string
	var destUsername string
	var destPassword string
	var debug bool
	var srcInsecure bool
	var destInsecure bool
	var srcHTTP bool
	var destHTTP bool
	var tagRegexp string

	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Long:  `Print the version number`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}
	var promoteCmd = &cobra.Command{
		Use:   "push [registry/image/tag] [registry/image/tag]",
		Short: "Push image",
		Long:  `Push image from one Registry into another one`,
		Run: func(cmd *cobra.Command, args []string) {

			if len(args) < 2 {
				fmt.Println("Missing command arguments, usage: push [registry/image/tag] [registry/image/tag]")
				os.Exit(1)
			}
			srcRegistry, srcImage, srcImageTag, err := ImageNameAndRegistryAndTag(args[0])
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			destRegistry, destImage, destImageTag, err := ImageNameAndRegistryAndTag(args[1])
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			replaceRegistryName(&srcRegistry)
			replaceRegistryName(&destRegistry)
			if srcHTTP {
				addRegistryProtocol(&srcRegistry, false)
			} else {
				addRegistryProtocol(&srcRegistry, true)
			}
			if destHTTP {
				addRegistryProtocol(&destRegistry, false)
			} else {
				addRegistryProtocol(&destRegistry, true)
			}

			prom := &image.Promote{
				SrcRegistry:  srcRegistry,
				SrcImage:     srcImage,
				SrcImageTag:  srcImageTag,
				SrcUsername:  srcUsername,
				SrcPassword:  srcPassword,
				SrcInsecure:  srcInsecure,
				DestRegistry: destRegistry,
				DestImage:    destImage,
				DestImageTag: destImageTag,
				DestUsername: destUsername,
				DestPassword: destPassword,
				DestInsecure: destInsecure,
				Debug:        debug,
			}
			prom.PromoteImage()

		},
	}

	var tagsCmd = &cobra.Command{
		Use:   "tags [registry/image] [registry/image]",
		Short: "Push image tags",
		Long:  `Push all image tags from one Registry into another one`,
		Run: func(cmd *cobra.Command, args []string) {

			if len(args) < 2 {
				fmt.Println("Missing command arguments, usage: tags [registry/image] [registry/image]")
				os.Exit(1)
			}
			srcRegistry, srcImage, err := ImageNameAndRegistry(args[0])
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			destRegistry, destImage, err := ImageNameAndRegistry(args[1])
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			replaceRegistryName(&srcRegistry)
			replaceRegistryName(&destRegistry)
			if srcHTTP {
				addRegistryProtocol(&srcRegistry, false)
			} else {
				addRegistryProtocol(&srcRegistry, true)
			}
			if destHTTP {
				addRegistryProtocol(&destRegistry, false)
			} else {
				addRegistryProtocol(&destRegistry, true)
			}
			if len(tagRegexp) > 0 {
				_, err = regexp.Compile(tagRegexp)
				if err != nil {
					fmt.Printf("Image Tag Regexp does not compile. Error: %q \n", err)
				}
			}

			prom := &tags.TagPush{
				SrcRegistry:  srcRegistry,
				SrcImage:     srcImage,
				SrcUsername:  srcUsername,
				SrcPassword:  srcPassword,
				SrcInsecure:  srcInsecure,
				DestRegistry: destRegistry,
				DestImage:    destImage,
				DestUsername: destUsername,
				DestPassword: destPassword,
				DestInsecure: destInsecure,
				TagRegexp:    tagRegexp,
				Debug:        debug,
			}
			prom.PushTags()

		},
	}

	RootCmd.AddCommand(versionCmd)
	RootCmd.AddCommand(promoteCmd)
	RootCmd.AddCommand(tagsCmd)

	promoteCmd.Flags().StringVar(&srcUsername, "src-username", "", "Source username")
	promoteCmd.Flags().StringVar(&srcPassword, "src-password", "", "Source password")
	promoteCmd.Flags().StringVar(&destUsername, "dest-username", "", "Destination username")
	promoteCmd.Flags().StringVar(&destPassword, "dest-password", "", "Destination password")
	promoteCmd.Flags().BoolVar(&srcHTTP, "src-http", false, "Use http when connecting to Source Registry")
	promoteCmd.Flags().BoolVar(&destHTTP, "dest-http", false, "Use http when connecting to Source Registry")
	promoteCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Debug")
	promoteCmd.Flags().BoolVar(&srcInsecure, "src-insecure", false, "Accept all certificates when connecting to Source Registry")
	promoteCmd.Flags().BoolVar(&destInsecure, "dest-insecure", false, "Accept all certificates when connecting to Destination Registry")
	tagsCmd.Flags().StringVar(&srcUsername, "src-username", "", "Source username")
	tagsCmd.Flags().StringVar(&srcPassword, "src-password", "", "Source password")
	tagsCmd.Flags().StringVar(&destUsername, "dest-username", "", "Destination username")
	tagsCmd.Flags().StringVar(&destPassword, "dest-password", "", "Destination password")
	tagsCmd.Flags().BoolVar(&srcHTTP, "src-http", false, "Use http when connecting to Source Registry")
	tagsCmd.Flags().BoolVar(&destHTTP, "dest-http", false, "Use http when connecting to Source Registry")

	tagsCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Debug")

	tagsCmd.Flags().BoolVar(&srcInsecure, "src-insecure", false, "Accept all certificates when connecting to Source Registry")
	tagsCmd.Flags().BoolVar(&destInsecure, "dest-insecure", false, "Accept all certificates when connecting to Destination Registry")
	tagsCmd.Flags().StringVar(&tagRegexp, "tag-regexp", "", "Filter image tags by specified regexp")
}

//ImageNameAndRegistry returns registry, image from provided fqdn
func ImageNameAndRegistry(url string) (registry string, image string, err error) {
	s := strings.Split(url, "/")
	if len(s) < 3 {
		return "", "", errors.New("invalid image reference. Image format should be following: [registry/repository/image] e.g. myregistry/repository/centos")
	}
	registry = s[0]
	image = s[1] + "/" + s[2]
	return registry, image, nil

}

//ImageNameAndRegistryAndTag returns registry, image and tag from provided fqdn
func ImageNameAndRegistryAndTag(src string) (registry string, image string, tag string, err error) {
	s := strings.Split(src, "/")
	if len(s) < 3 {
		return "", "", "", errors.New("invalid image reference. Image format should be following: [registry/repository/image] e.g. hub.docker.io/library/centos")

	}
	registry = s[0]
	image = s[1] + "/" + s[2]
	imageAndTag := strings.Split(image, ":")
	//Image name and tag specified
	if len(imageAndTag) > 1 {
		image = imageAndTag[0]
		tag = imageAndTag[1]
	} else {
		//No tag specified
		image = s[1] + "/" + s[2]
		tag = "latest"
	}
	return registry, image, tag, nil
}

//Adds HTTP or HTTPS suffix if it's missing
func addRegistryProtocol(registry *string, secure bool) {
	if !strings.HasPrefix(*registry, "http") || !strings.HasPrefix(*registry, "https") {
		if secure {
			*registry = "https://" + *registry
		} else {
			*registry = "http://" + *registry
		}
	}
}

//Replaces some hardcoded registry names
func replaceRegistryName(registry *string) {
	if strings.Contains(*registry, "docker.io") {
		//*registry = "index.docker.io"
		*registry = "registry-1.docker.io"

	}
}
