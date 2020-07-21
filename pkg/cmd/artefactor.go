package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/appvia/artefactor/pkg/util"
	"github.com/spf13/cobra"
)

const (
	EnvPrefix = "ARTEFACTOR_"
	// FlagDockerImages specifies a to newline seperated list of docker images to save
	FlagDockerImages = "docker-images"
	// FlagArchiveDir is the directory to save archives into
	FlagArchiveDir = "archive-dir"
	// FlagGitRepos specifies a newline seperated list of local or remote git repos
	FlagGitRepos = "git-repos"
	// FlagWebFiles specifies a whitespace delimited set of csv's with:
	// url,file,md5,[true|false (executable)]
	FlagWebFiles = "web-files"
	// FlagLogs enabled debug logs
	FlagLogs = "logs"
	// FlagTargetPlatform allows the correct version of artefactor to be saved
	// with files.
	FlagTargetPlatform = "target-platform"
	// FlagImageVars is used to specify a whitelist of variable names to enable
	// "export"
	FlagImageVars = "image-vars"
    // FlagImageNameDryRun is used to output the image names that would be
    // generated during `artefactor publish`
    FlagImageNameDryRun = "dry-run"
	// FlagDockerUserName overrides docker registry configuration
	FlagDockerUserName = "docker-username"
	// FlagDockerPassword overrides docker registry configuration
	FlagDockerPassword = "docker-password"
	// FlagDockerPasswordHelp is displayed when getting help for the flag
	FlagDockerPasswordHelp = "overrides docker registry configuration for password"
	// FlagDockerUserNameHelp is displayed when getting help for the flag
	FlagDockerUserNameHelp = "overrides docker registry configuration for username"
	// DefaultArchiveDir
	DefaultArchiveDir = "downloads"
	// DefaultTargetPlatform is the default binary type to include in downloads
	DefaultTargetPlatform = "linux_amd64"
)

var (
	// RootCmd represents the base command when called without any subcommands
	RootCmd = &cobra.Command{
		Use:   "artefactor",
		Short: "artefactor saves things to files",
		Long:  "artefactor saves docker containers, git repos and web files to, err, files",
		RunE: func(c *cobra.Command, args []string) error {
			common(c)
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
	RootCmd.PersistentFlags().Bool(FlagLogs, false, "Enable debug logs")
	addFlagWithEnvDefault(
		RootCmd,
		FlagDockerImages,
		"",
		"A whitespace seperated list of docker images")
	addFlagWithEnvDefault(
		RootCmd,
		FlagGitRepos,
		"",
		"A whitespace seperated list of local or remote git repos")
	addFlagWithEnvDefault(
		RootCmd,
		FlagWebFiles,
		"",
		"A whitespace seperated list of CSV's: url,filename,sha256,true")
}

// addFlagWithEnvDefault adds a defaultValue
func addFlagWithEnvDefault(c *cobra.Command, flag string, defVal string, help string) {
	c.PersistentFlags().String(
		flag,
		defaultValue(flag, defVal),
		fmt.Sprintf("%s (${%s})", help, GetEnvName(flag)))
}

func common(c *cobra.Command) {
	logs, _ := c.Flags().GetBool(FlagLogs)
	if !logs {
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}
}

func defaultValue(flagName string, defaultValue string) string {
	envValue := os.Getenv(GetEnvName(flagName))
	if len(envValue) > 0 {
		return envValue
	}
	return defaultValue
}

func getCredsFromFlags(c *cobra.Command) *util.Creds {
	username := c.Flag(FlagDockerUserName).Value.String()
	password := c.Flag(FlagDockerPassword).Value.String()

	if len(username) > 0 {
		creds := &util.Creds{
			Username: username,
			Password: password,
		}
		return creds
	}
	return nil
}

func GetEnvName(flagName string) string {
	return (EnvPrefix +
		strings.Replace(strings.ToUpper(flagName), "-", "_", -1))
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(-1)
	}
}
