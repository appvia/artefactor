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
		"the whitelist separated list of variables specifying original image names")

	addFlagWithEnvDefault(
		imageNamesCmd,
		FlagDockerRegistry,
		"",
		"where images have been published e.g. private-registry.local")

	RootCmd.AddCommand(imageNamesCmd)
}

func imageNames(c *cobra.Command) error {
	common(c)
	// get the registry (if specified)
	registry := c.Flag(FlagDockerRegistry).Value.String()
	// Complain if no registry is specified
	if len(registry) < 1 {
		return fmt.Errorf("must specify registry for %s", ImageNamesCommand)
	}
	imageVars := strings.Fields(c.Flag(FlagImageVars).Value.String())
	log.Printf("image vars:%v", imageVars)
	for _, imageVar := range imageVars {
		image := os.Getenv(imageVar)
		newImageName := docker.GetNewImageName(image, registry)
		fmt.Printf("export %s=%s\n", imageVar, newImageName)
	}
	return nil
}
