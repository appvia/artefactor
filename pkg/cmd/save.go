package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/appvia/artefactor/pkg/docker"
	"github.com/appvia/artefactor/pkg/git"
	"github.com/appvia/artefactor/pkg/hashcache"
	"github.com/appvia/artefactor/pkg/util"
	"github.com/appvia/artefactor/pkg/web"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// SaveCommand is the sub command syntax
const (
	SaveCommand     string = "save"
	SaveDirMetaFile string = "saveDir.meta"
)

// saveCmd represents the version command
var saveCmd = &cobra.Command{
	Use:   SaveCommand,
	Short: "saves artefact(s)",
	Long:  "will save artefact(s) to file(s)",
	RunE: func(c *cobra.Command, args []string) error {
		return save(c)
	},
}

func init() {
	addFlagWithEnvDefault(
		saveCmd,
		FlagArchiveDir,
		DefaultArchiveDir,
		"a location to save artefacts to and publish from")

	RootCmd.AddCommand(saveCmd)
}

func save(c *cobra.Command) error {
	common(c)
	// Record where the archives should be stored
	saveDir := c.Flag(FlagArchiveDir).Value.String()
	if err := saveSavedPath(saveDir); err != nil {
		return fmt.Errorf(
			"problem saving meta data file to record archive path %s:%s",
			saveDir,
			err)
	}

	// First save docker images
	images := strings.Fields(c.Flag(FlagDockerImages).Value.String())
	for _, image := range images {
		if err := docker.Save(image, saveDir); err != nil {
			return fmt.Errorf(
				"problem saving docker image %s to directory %s:%s",
				image,
				saveDir,
				err)
		}
	}

	// Next save any git repos
	gitRepos := strings.Fields(c.Flag(FlagGitRepos).Value.String())
	for _, repo := range gitRepos {
		if err := git.Archive(repo, saveDir); err != nil {
			return fmt.Errorf(
				"problem saving git repository %s to directory %s:%s",
				repo,
				saveDir,
				err)
		}
	}

	// Now save Web files
	webFiles := strings.Fields(c.Flag(FlagWebFiles).Value.String())
	for _, webFile := range webFiles {
		parts := strings.Split(webFile, ",")
		if len(parts) < 3 {
			return errors.Errorf("expecting a web file CSV with url,filename,sha256[,true|false]")
		}
		url := parts[0]
		fileName := parts[1]
		sha256 := parts[2]
		binFile := false
		if len(parts) == 4 {
			binFile = true
		}
		if err := web.Save(url, fileName, saveDir, sha256, binFile); err != nil {
			return fmt.Errorf(
				"problem saving url:%s to filename %s/%s:%s",
				url,
				saveDir,
				fileName,
				err)
		}
	}

	// lastly save this binary...
	me, _ := os.Executable()
	if err := copyBin(me, saveDir); err != nil {
		return fmt.Errorf(
			"problem trying to save %s as %s/%s:%s",
			me,
			saveDir,
			me,
			err)
	}
	return nil
}

// copyBin will save binary meta-data for a local binary to the archive dir
func copyBin(srcBin string, saveDir string) error {
	savedBin := filepath.Join(saveDir, filepath.Base(srcBin))
	if err := util.Cp(srcBin, savedBin); err != nil {
		return err
	}
	if _, err := hashcache.UpdateCache(savedBin); err != nil {
		return err
	}
	err := util.BinMark(savedBin)
	return err
}

// saveSavedPath will record meta-data so files are restored to the same
// relative path
func saveSavedPath(saveDir string) error {
	// Save the archive dir as meta-data file:
	saveDir = filepath.Clean(saveDir)
	saveMetaFilePath := filepath.Join(saveDir, SaveDirMetaFile)
	err := ioutil.WriteFile(
		saveMetaFilePath,
		[]byte(saveDir),
		0644)
	if err != nil {
		return err
	}
	hashcache.UpdateCache(saveMetaFilePath)
	return nil
}

// getSavedPath will retireve the saved path from the meta-data file.
func getSavedPath(dstDir, srcDir string) (string, error) {
	b, err := ioutil.ReadFile(filepath.Join(srcDir, SaveDirMetaFile))
	if err != nil {
		return "", err
	}
	saveDir := filepath.Join(dstDir, string(b))
	return saveDir, nil
}
