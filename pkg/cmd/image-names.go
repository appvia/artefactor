package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/appvia/artefactor/pkg/docker"
	"github.com/spf13/cobra"
)

const (
	// ImageNamesCommand is the sub command syntax
	ImageNamesCommand string = "update-image-vars"
)

// imageNamesCmd represents the command to display image information
var imageNamesCmd = &cobra.Command{
	Use:   ImageNamesCommand,
	Short: "update docker image name variables",
	Long:  "update docker image names from environment with registry name",
	RunE: func(c *cobra.Command, args []string) error {
		return imageNames(c)
	},
}

func init() {
	addFlagWithEnvDefault(
		imageNamesCmd,
		FlagArchiveDir,
		DefaultArchiveDir,
		"a directory where artefacts exist to publish from")

	addFlagWithEnvDefault(
		imageNamesCmd,
		FlagImageVars,
		"",
		"the whitelist separated list of variables specifying orininal image names")

	addFlagWithEnvDefault(
		imageNamesCmd,
		FlagDockerRegistry,
		"",
		"where images have been published e.g. private-registry.local")

	RootCmd.AddCommand(imageNamesCmd)
}

func imageNames(c *cobra.Command) error {
	common(c)
	src := c.Flag(FlagArchiveDir).Value.String()
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("missing archive %s. error: %s", src, err)
	}
	// get the registry (if specified)
	registry := c.Flag(FlagDockerRegistry).Value.String()
	images, err := docker.GetImages(src, registry)
	if err != nil {
		return fmt.Errorf(
			"problem getting a list of images from file names in %s:%s", src, err)
	}
	if len(images) < 1 {
		return fmt.Errorf(
			"No docker image artefacts found, use `save` first or specify flag %s",
			FlagArchiveDir)
	}
	// Complain if no registry is specified
	if len(registry) < 1 {
		return fmt.Errorf("must specify registry for %s", ImageNamesCommand)
	}
	imageVars := strings.Fields(c.Flag(FlagImageVars).Value.String())
	log.Printf("image vars:%v", imageVars)
	for _, image := range images {
		for _, imageVar := range imageVars {
			log.Printf("%s == %s", image.ImageName, os.Getenv(imageVar))
			if image.ImageName == os.Getenv(imageVar) {
				fmt.Printf("export %s=%s\n", imageVar, image.NewImageName)
			}
		}
	}
	return nil
}
