package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const (
	// FlagDockerImages specifies a to newline seperated list of docker images to save
	FlagDockerImages = "docker-images"
	// The directory to save archives into
	FlagArchiveDir = "archive-dir"
)

var (
	// RootCmd represents the base command when called without any subcommands
	RootCmd = &cobra.Command{
		Use:   "artefactor",
		Short: "artefactor saves things to files",
		Long:  "artefactor saves docker containers, git repos and web files to, err, files",
		RunE: func(c *cobra.Command, args []string) error {
			if c.Flags().Changed("version") {
				printVersion()
				return nil
			}
			return c.Usage()
		},
	}
)

func init() {
	// Local flags
	RootCmd.Flags().BoolP("help", "h", false, "Help message")
	RootCmd.Flags().BoolP("version", "v", false, "Print version")
	RootCmd.PersistentFlags().String(
		FlagDockerImages,
		os.Getenv(strings.ToUpper(FlagDockerImages)),
		"A whitespace seperated list of docker images")
	RootCmd.PersistentFlags().String(
		FlagArchiveDir,
		defaultValue(FlagArchiveDir, "downloads"),
		"Where to save all archives")
}

func defaultValue(flagName string, defaultValue string) string {
	envValue := os.Getenv(strings.ToUpper(flagName))
	if len(envValue) > 0 {
		return envValue
	}
	return defaultValue
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}
