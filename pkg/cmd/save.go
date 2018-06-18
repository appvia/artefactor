package cmd

import (
	"strings"

	"github.com/appvia/artefactor/pkg/docker"
	"github.com/spf13/cobra"
)

// SaveCommand is the sub command syntax
const SaveCommand string = "save"

// cleanupCmd represents the version command
var saveCmd = &cobra.Command{
	Use:   SaveCommand,
	Short: "saves artefact(s)",
	Long:  "will save artefact(s) to file(s)",
	Run: func(c *cobra.Command, args []string) {
		save(c)
	},
}

func init() {
	saveCmd.PersistentFlags().String(
		"artifacts-dir",
		"./downloads",
		"a location to save / load all artefacts")

	RootCmd.AddCommand(saveCmd)
}

func save(c *cobra.Command) {
	// First save docker images
	images := strings.Split(c.Flag(FlagDockerImages).Value.String(), "\n")
	for _, image := range images {
		docker.Save(
			image,
			c.Flag(FlagArchiveDir).Value.String())
	}

	// Now save Web files
	//webFiles := strings.Split()
}
