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
		restorePath, err := getRelativeSavedPath(src)
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

	// Get the list of files from the checksum file (hashcache) in the source
	// directory.
	repoPath := filepath.Join(filepath.Clean(dst), repoName)
	dstDir := filepath.Join(repoPath, filepath.Clean(savedDir))
	refresh := false
	log.Printf("dst:%s\nRepo:%s\nSavedDir:%s\n==>%s", dst, repoName, savedDir, dstDir)
	if _, err := os.Stat(dstDir); err == nil {
		refresh = true
	}

	var missingFiles []string
	// Verify if we have everything we need BEFORE moving files
	// Check we have all files in source OR destination BEFORE we start to copy...
	srcChk, err := hashcache.NewFromDir(src, true)
	if err != nil {
		return fmt.Errorf("problem with checksum file in folder %s:%s", src, err)
	}
	log.Printf("items in cache %v", len(srcChk.CheckSumsByFilePath))
	for _, item := range srcChk.CheckSumsByFilePath {
		// Only worry if the file refered from the checksum file doesn't exist
		if _, err := os.Stat(item.FilePath); err == nil {
			log.Printf("file present in cache and disk %s", item.FilePath)
		} else {
			log.Printf("file present in cache and missing on disk %s", item.FilePath)
			// if refreshing then check if file exists in destination...
			if !refresh {
				// Not refreshing files so all files have to be present!
				log.Printf("not refreshing files so file is missing %s", item.FilePath)
				missingFiles = append(missingFiles, item.FilePath)
			} else {
				// Refreshing only some files so check if file in destination already...
				dstFile := filepath.Join(dstDir, item.FileName)
				if _, err := os.Stat(dstFile); err != nil {
					// File not in source or destination!
					missingFiles = append(missingFiles, item.FilePath)
					log.Printf("refreshing, and file missing from destination %s", dstFile)
				} else {
					// File only in destination so we need to check it's the right one:
					fmt.Printf(
						"Checking existing file (no update provided) %s\n",
						dstFile)
					calcAndCheckSum(dstFile)
				}
			}
		}
	}
	if len(missingFiles) > 0 {
		fmt.Printf("Missing files:\n")
		for _, file := range missingFiles {
			fmt.Printf("  %s\n", file)
		}
		return fmt.Errorf(
			"files in checksum file %s not present in source %s or destination %s\n",
			srcChk.CheckSumFile,
			src,
			dstDir)
	}
	// Add check for clean destination git repo...
	if refresh {
		clean, err := git.IsClean(repoPath)
		if err != nil {
			return fmt.Errorf("can't read git repo at %s:%s", repoPath, err)
		}
		if !clean {
			return fmt.Errorf(
				"destination repo is NOT clean, please clean then restore (%s)",
				repoPath)
		}
	}

	// Pre-flight checks OK...
	fmt.Printf(
		"Restoring git files from %s to %s\n",
		gitRepoFile,
		repoPath)
	if err := git.Restore(gitRepoFile, dst, repoName, savedDir); err != nil {
		return err
	}
	// This may not exist even after an update due to the way files are moved
	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		log.Printf("the dir doesn't exist %s so creating it...", dstDir)
		if err := os.MkdirAll(dstDir, 0755); err != nil {
			return fmt.Errorf(
				"problem creating destination directory structure %s:%s",
				dstDir,
				err)
		}
	}
	// Ensure we have a checksum file in the destination first...
	if err != nil {
		return fmt.Errorf("cannot read checksum file from %s:%s", dstDir, err)
	}
	// Now move all the files, checking checksums as we go...
	for srcFile, chkItem := range srcChk.CheckSumsByFilePath {
		dstFile := filepath.Join(dstDir, chkItem.FileName)

		// Support incremental copies (error if destination not already present)
		if _, err := os.Stat(srcFile); err == nil {
			fmt.Printf("Moving file %q to %q\n", srcFile, dstDir)
			if err := util.Mv(srcFile, dstFile); err != nil {
				return err
			}
			calcAndCheckSum(dstFile)
		}
	}
	dstChkFile := filepath.Join(dstDir, hashcache.DefaultCheckSumFileName)
	// Finally move the checksums file...
	if err := util.Mv(
		srcChk.CheckSumFile,
		dstChkFile); err != nil {
		return fmt.Errorf(
			"cannot move checksum file (%s) from:%s to %s:%s",
			srcChk.CheckSumFile,
			src,
			dstDir,
			err)
	}
	fmt.Printf("All artefacts restored and checked\n")
	return nil
}

// calcAndCheckSum will display the results of verifying a checksum
func calcAndCheckSum(file string) error {
	file = filepath.Clean(file)
	fmt.Printf("  Checksum:")
	calcHash, err := hashcache.CalcChecksum(file)
	if err != nil {
		return err
	}
	// Get a reference to a checksum file but don't create the entry
	dstChk, err := hashcache.NewFromExistingFile(file, false)
	if err != nil {
		return err
	}
	log.Printf(calcHash)
	if dstChk.IsCachedMatched(file, calcHash) {
		fmt.Printf("OK\n")
	} else {
		expectedHash, ok := dstChk.CheckSumsByFilePath[file]
		if !ok {
			return fmt.Errorf(
				"missing checksum for %s from %s", file, dstChk.CheckSumFile)
		} else {
			return fmt.Errorf(
				"failed for %s, expecting %s but got %s\n",
				file,
				expectedHash,
				calcHash)
		}
	}
	return nil
}
