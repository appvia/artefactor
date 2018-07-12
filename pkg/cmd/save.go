package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/appvia/artefactor/pkg/docker"
	"github.com/appvia/artefactor/pkg/git"
	"github.com/appvia/artefactor/pkg/hashcache"
	"github.com/appvia/artefactor/pkg/util"
	"github.com/appvia/artefactor/pkg/version"
	"github.com/appvia/artefactor/pkg/web"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// SaveCommand is the sub command syntax
const (
	SaveCommand           string = "save"
	SaveDirMetaFile       string = "saveDir.meta"
	ArtefactorBinaryName  string = "artefactor"
	ArtefactorPublishRoot string = "https://github.com/appvia/artefactor/releases/download/%s/"
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

	addFlagWithEnvDefault(
		saveCmd,
		FlagTargetPlatform,
		DefaultTargetPlatform,
		"the target platform in format [platform]_[arch]")

	addFlagWithEnvDefault(
		saveCmd,
		FlagImageVars,
		"",
		"the whitelist separated list of variables specifying orininal image names")

	RootCmd.AddCommand(saveCmd)
}

func save(c *cobra.Command) error {
	common(c)
	// Record where the archives should be stored
	saveDir := c.Flag(FlagArchiveDir).Value.String()

	type webfile struct {
		url      string
		fileName string
		sha      string
		bin      bool
	}

	// Pre-flight checks:
	webFiles := []webfile{}
	for _, webFile := range strings.Fields(c.Flag(FlagWebFiles).Value.String()) {
		parts := strings.Split(webFile, ",")
		if len(parts) < 3 {
			return errors.Errorf("expecting a web file CSV with url,filename,sha256[,true|false]")
		}
		binFile := false
		if len(parts) == 4 {
			if strings.ToLower(parts[3]) == "true" {
				binFile = true
			}
		}
		w := webfile{
			url:      parts[0],
			fileName: parts[1],
			sha:      parts[2],
			bin:      binFile,
		}
		webFiles = append(webFiles, w)
	}

	// Now make changes
	fmt.Println("Saving meta-data and me")
	if err := saveSavedPath(saveDir); err != nil {
		return fmt.Errorf(
			"problem saving meta data file to record archive path %s:%s",
			saveDir,
			err)
	}

	// Save the binary for the target platform
	platform := c.Flag(FlagTargetPlatform).Value.String()
	if err := saveMe(saveDir, platform); err != nil {
		return err
	}

	// save any git repos
	gitRepos := strings.Fields(c.Flag(FlagGitRepos).Value.String())
	for _, repo := range gitRepos {
		fmt.Printf("\nSaving git repos\n")
		if err := git.Archive(repo, saveDir); err != nil {
			return fmt.Errorf(
				"problem saving git repository %s to directory %s:%s",
				repo,
				saveDir,
				err)
		}
	}

	// save docker images
	images := getImages(c)
	for _, image := range images {
		fmt.Printf("\nSaving docker images\n")
		if err := docker.Save(image, saveDir); err != nil {
			return fmt.Errorf(
				"problem saving docker image %s to directory %s:%s",
				image,
				saveDir,
				err)
		}
	}

	// Now save Web files
	for _, webFile := range webFiles {
		fmt.Printf("\nSaving web files\n")
		if err := web.Save(webFile.url, webFile.fileName, saveDir, webFile.sha, webFile.bin); err != nil {
			return fmt.Errorf(
				"problem saving url:%s to filename %s/%s:%s",
				webFile.url,
				saveDir,
				webFile.fileName,
				err)
		}
	}
	fmt.Printf("all artefacts correct and present\n")
	return nil
}

// getImages gets the images from either docker-images or image-vars
func getImages(c *cobra.Command) []string {
	images := strings.Fields(c.Flag(FlagDockerImages).Value.String())
	imageVars := strings.Fields(c.Flag(FlagImageVars).Value.String())
	for _, varName := range imageVars {
		newImage := os.Getenv(varName)
		if !contains(images, newImage) {
			images = append(images, newImage)
		}
	}
	return images
}

func contains(ary []string, item string) bool {
	for _, s := range ary {
		if s == item {
			return true
		}
	}
	return false
}

// saveMe saves a copy of the target binary in the save dir
func saveMe(saveDir, platform string) error {
	binaryDst := filepath.Join(saveDir, ArtefactorBinaryName)
	// detect if the binary we are saving with matches target platform...
	if fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH) == platform {
		me, _ := os.Executable()
		if err := copyBin(me, saveDir); err != nil {
			return fmt.Errorf(
				"problem trying to save %s as %s:%s",
				me,
				binaryDst,
				err)
		}
	} else {
		platformBin := ArtefactorBinaryName + "_" + platform
		// We need to download the correct binary
		// TODO: maybe implement a local download cache in users home (with cleanup?)
		url := fmt.Sprintf(ArtefactorPublishRoot, version.Get().Version) +
			"/" + platformBin
		checkSumsUrl := fmt.Sprintf(ArtefactorPublishRoot, version.Get().Version) +
			"/" + hashcache.DefaultCheckSumFileName

		tmpDir, err := ioutil.TempDir("", "artefactor_downloads")
		if err != nil {
			fmt.Sprintf("problem creating temp dir for artefactor downloads")
		}

		defer os.RemoveAll(tmpDir) // clean up

		// download checksums file:
		checkSumFile := filepath.Join(tmpDir, hashcache.DefaultCheckSumFileName)
		if err := web.SaveNoCheck(checkSumsUrl, checkSumFile, false); err != nil {
			return fmt.Errorf(
				"problem trying to download artefactor checksums from %s",
				checkSumsUrl)
		}
		tmpBinPath := filepath.Join(tmpDir, platformBin)
		if err := web.SaveNoCheck(url, tmpBinPath, true); err != nil {
			return fmt.Errorf("problem trying to download artefactor from %s", url)
		}
		// Verify the download:
		binChksum, err := hashcache.GetCachedChecksum(tmpBinPath)
		if err != nil {
			return fmt.Errorf(
				"problem getting checksum for from %s:%s",
				tmpBinPath,
				err)
		}
		if calcBinChkSum, err := hashcache.CalcChecksum(tmpBinPath); err != nil {
			if calcBinChkSum != binChksum {
				return fmt.Errorf(
					"download %s had unexpected checksum %s, expecting %s (from %s)",
					url,
					calcBinChkSum,
					binChksum,
					checkSumsUrl)
			}
		}

		// Finaly move the file to the correct download path:
		if err := util.Mv(tmpBinPath, binaryDst); err != nil {
			return fmt.Errorf(
				"unable to move from %s to %s:%s",
				tmpBinPath,
				binaryDst,
				err)
		}

		if _, err := hashcache.UpdateCache(binaryDst); err != nil {
			return fmt.Errorf("unable to update hash for %s:%s", binaryDst, err)
		}
		if err := util.BinMark(binaryDst); err != nil {
			return fmt.Errorf(
				"problem creating meta data file for %s:%s", binaryDst, err)
		}
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
