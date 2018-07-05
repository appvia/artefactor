package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/appvia/artefactor/pkg/git"
	"github.com/appvia/artefactor/pkg/hashcache"
	"github.com/appvia/artefactor/pkg/util"
	"github.com/spf13/cobra"
)

// SaveCommand is the sub command syntax
const (
	RestoreCommand       string = "restore"
	FlagRestoreSourceDir string = "source-dir"
	FlagRestoreDestDir   string = "dest-dir"
)

// cleanupCmd represents the version command
var restoreCmd = &cobra.Command{
	Use:   RestoreCommand,
	Short: "restores artefact(s)",
	Long:  "will restore artefact(s) to correct paths and modes i.e. within a git repo.",
	RunE: func(c *cobra.Command, args []string) error {
		return restore(c)
	},
}

func init() {
	restoreCmd.PersistentFlags().String(
		FlagRestoreSourceDir,
		defaultValue(FlagRestoreSourceDir, "."),
		fmt.Sprintf(
			"a directory with all artefacts (${%s})",
			GetEnvName(FlagRestoreSourceDir)))
	restoreCmd.PersistentFlags().String(
		FlagRestoreDestDir,
		defaultValue(FlagRestoreDestDir, "."),
		fmt.Sprintf(
			"a directory to start the restore process from (${%s})",
			GetEnvName(FlagRestoreDestDir)))

	RootCmd.AddCommand(restoreCmd)
}

func restore(c *cobra.Command) error {
	common(c)

	src := c.Flag(FlagRestoreSourceDir).Value.String()
	dst := c.Flag(FlagRestoreDestDir).Value.String()

	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("missing src directory (not found) %s", src)
	}
	// First re-create the 'home' git repo....
	homeRepo, err := git.GetHomeRepo(src)
	if err != nil {
		return err
	}
	log.Printf("home repo is here %s", homeRepo)

	// Get the home repo if it exists
	if homeRepo == "" {
		log.Printf("no git home repo archive found in %s", src)
	} else {
		restorePath, err := getSavedPath(dst, src)
		if err != nil {
			return fmt.Errorf(
				"problem retrieveing meta data for archive directory from %s:%s",
				src,
				err)
		}
		if err := RestoreHome(homeRepo, dst, restorePath); err != nil {
			return err
		}
	}
	return nil
}

// RestoreHome will restore the current repo and move all other archive files as
// specified in the checksums file
func RestoreHome(gitRepoFile string, dst string, savedDir string) error {

	// Get the source directory from repo:
	src := filepath.Dir(gitRepoFile)
	// Get the git repo name from the file name...
	repoName := strings.TrimSuffix(filepath.Base(gitRepoFile), git.GitFileHomeExt)
	fmt.Printf("Restoring git files from %s to %s\n", src, dst)
	if err := git.Restore(gitRepoFile, dst, repoName); err != nil {
		return err
	}

	// Get the list of files from the checksum file (hashcache) in the source
	// directory.
	dstDir := filepath.Join(dst, repoName, savedDir)
	files := hashcache.GetFiles(src)
	// Create directory structure
	if _, err := os.Stat(dstDir); err != nil {
		if err := os.MkdirAll(dstDir, 0755); err != nil {
			return fmt.Errorf(
				"problem creating destination directory structure %s:%s",
				dstDir,
				err)
		}
	}
	// Ensure we have a checksum file in the destination first...
	srcChecksum := filepath.Join(src, hashcache.CheckSumFileName)
	if err := util.Cp(
		srcChecksum,
		filepath.Join(dstDir, hashcache.CheckSumFileName)); err != nil {
		return fmt.Errorf(
			"cannot copy checksum file (%s) from:%s to %s:%s",
			hashcache.CheckSumFileName,
			src,
			dstDir,
			err)
	}

	for _, file := range files {
		srcFile := filepath.Join(src, file)
		dstFile := filepath.Join(dstDir, file)

		// Support incremental copies (error if destination not already present)
		if _, err := os.Stat(srcFile); err == nil {
			fmt.Printf("Moving file %q to %q\n", file, dstDir)
			if err := util.Mv(srcFile, dstFile); err != nil {
				return err
			}
		}
		if _, err := os.Stat(dstFile); os.IsNotExist(err) {
			// File doesn't exist so incremental copy failed!
			fmt.Printf(
				"File missing from src and destination (%s), please provide",
				file)
		}
		fmt.Printf("  Checksum:")
		calcHash, err := hashcache.CalcChecksum(dstFile)
		if err != nil {
			return err
		}
		log.Printf(calcHash)
		if hashcache.IsCachedMatch(dstFile, calcHash) {
			fmt.Printf("OK\n")
		} else {
			expectedHash, _ := hashcache.GetCachedChecksum(dstFile)
			return fmt.Errorf(
				" failed for %s, expecting %s but got %s\n",
				dstFile,
				expectedHash,
				calcHash)
		}
	}
	os.Remove(srcChecksum)
	fmt.Printf("All artefacts restored and checked\n")
	return nil
}
