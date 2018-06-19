package cmd

import (
	"fmt"
	"os"

	"github.com/appvia/artefactor/pkg/docker"
	"github.com/spf13/cobra"
)

const (
	FlagDockerRegistry string = "docker-registry"
	// PublishCommand is the sub command syntax
	PublishCommand string = "publish"
)

// publishCmd represents the version command
var publishCmd = &cobra.Command{
	Use:   PublishCommand,
	Short: "publishes artefact(s)",
	Long:  "will publish artefact(s) to correct registries / locations.",
	RunE: func(c *cobra.Command, args []string) error {
		return publish(c)
	},
}

func init() {
	addFlagWithEnvDefault(
		publishCmd,
		FlagArchiveDir,
		DefaultArchiveDir,
		"a directory where artefacts exist to publish from")

	addFlagWithEnvDefault(
		publishCmd,
		FlagDockerRegistry,
		"",
		"where to publish images e.g. private-registry.local")

	RootCmd.AddCommand(publishCmd)
}

func publish(c *cobra.Command) error {
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
	if len(images) > 0 {
		// Complain if we've been asked to publish any containers
		if len(registry) < 1 {
			return fmt.Errorf("must specify registry for publish")
		}
	}
	for _, image := range images {
		fmt.Printf("Loading image from %s.\n", image.FileName)
		if err := docker.Load(image.FileName); err != nil {
			return fmt.Errorf("load image problem for %s:%s", image.FileName, err)
		}
		fmt.Printf("ReTagging image as %s.\n", image.NewImageName)
		if err := docker.ReTag(image); err != nil {
			return fmt.Errorf(
				"problem retagging %s to %s:%s",
				image.ImageName,
				image.NewImageName, err)
		}
		fmt.Printf("pushing image %s\n", image.NewImageName)
		if err := docker.Push(image.NewImageName); err != nil {
			return fmt.Errorf(
				"problem pushing image %s to registry:%s",
				image.NewImageName,
				err)
		}
		fmt.Printf("  Pushed.\n")
	}
	return nil
}
