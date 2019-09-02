package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/appvia/artefactor/pkg/hashcache"
	"github.com/spf13/cobra"
)

const (
	// CleanCmd is the sub command syntax
	CleanCmd string = "clean"
)

// cleanCmd represents the command to delete old files
var cleanCmd = &cobra.Command{
	Use:   CleanCmd,
	Short: "removes files not in checksums file",
	Long:  "will delete any files not in the checksums file",
	RunE: func(c *cobra.Command, args []string) error {
		return clean(c)
	},
}

func init() {
	addFlagWithEnvDefault(
		cleanCmd,
		FlagArchiveDir,
		DefaultArchiveDir,
		"a directory where artefacts exist")

	RootCmd.AddCommand(cleanCmd)
}

func clean(c *cobra.Command) error {
	common(c)
	saveDir := c.Flag(FlagArchiveDir).Value.String()

	if _, err := os.Stat(saveDir); os.IsNotExist(err) {
		fmt.Printf("no files to clean, all done")
	}

	// Create a new CheckSumCache:
	hc, err := hashcache.NewFromDir(saveDir, false)
	if err != nil {
		return fmt.Errorf("cant create cache for dir %s:%s", saveDir, err)
	}

	allFiles := []string{}
	// find all the files...
	err = filepath.Walk(
		saveDir,
		func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("access denied accessing a path %q: %v\n", path, err)
				return err
			}
			allFiles = append(allFiles, path)
			return nil
		})
	if err != nil {
		return err
	}

	for _, file := range allFiles {
		if filepath.Clean(file) == filepath.Clean(saveDir) {
			continue
		}
		if filepath.Clean(file) == hc.CheckSumFile {
			continue
		}
		if _, present := hc.CheckSumsByFilePath[file]; !present {
			// Clean up the old file
			fmt.Printf("removing file %s\n", file)
			if err := os.Remove(file); err != nil {
				return fmt.Errorf("problem removing old file %s:%s", file, err)
			}
		}
	}
	return nil
}
