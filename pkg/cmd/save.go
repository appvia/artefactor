package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
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
		"the whitelist separated list of variables specifying original image names")

	addFlagWithEnvDefault(
		saveCmd,
		FlagDockerUserName,
		"",
		FlagDockerUserNameHelp)

	addFlagWithEnvDefault(
		saveCmd,
		FlagDockerPassword,
		"",
		FlagDockerPasswordHelp)

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

	// validate all git repo's exists and are clean
	gitRepos := strings.Fields(c.Flag(FlagGitRepos).Value.String())
	for _, repo := range gitRepos {
		if isclean, err := git.IsClean(repo); err != nil && !isclean {
			return fmt.Errorf(
				"unable to check git repo %s:%s",
				repo,
				err)
		} else if err != nil {
			return fmt.Errorf(
				"git repo %s is not clean - refusing to continue",
				repo)
		}
	}

	// validate docker images
	images := getImages(c)

	// Now make changes
	if _, err := os.Stat(saveDir); os.IsNotExist(err) {
		// Create the downloads folder
		if err := os.MkdirAll(saveDir, 0744); err != nil {
			return fmt.Errorf(
				"problem creating save dir %s:%s",
				saveDir,
				err)
		}
	}

	// Create a new CheckSumCache:
	hc, err := hashcache.NewFromDir(saveDir, false)
	if err != nil {
		return fmt.Errorf("cant create cache for dir %s:%s", saveDir, err)
	}
	fmt.Println("Saving meta-data and me")
	if err := saveSavedPath(hc, saveDir); err != nil {
		return fmt.Errorf(
			"problem saving meta data file to record archive path %s:%s",
			saveDir,
			err)
	}

	// Save the binary for the target platform
	platform := c.Flag(FlagTargetPlatform).Value.String()
	if err := saveMe(hc, saveDir, platform); err != nil {
		return err
	}

	// save any git repos
	for _, repo := range gitRepos {
		fmt.Printf("\nSaving git repos\n")
		if err := git.Archive(hc, repo, saveDir); err != nil {
			return fmt.Errorf(
				"problem saving git repository %s to directory %s:%s",
				repo,
				saveDir,
				err)
		}
	}

	// save docker images
	for _, image := range images {
		fmt.Printf("\nSaving docker images\n")
		if err := docker.Save(hc, image, saveDir, getCredsFromFlags(c)); err != nil {
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
		if err := web.Save(hc, webFile.url, webFile.fileName, saveDir, webFile.sha, webFile.bin); err != nil {
			return fmt.Errorf(
				"problem saving url:%s to filename %s/%s:%s",
				webFile.url,
				saveDir,
				webFile.fileName,
				err)
		}
	}
	if err := hc.Clean(); err != nil {
		return fmt.Errorf("problem saving new set of files:%s", err)
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
			log.Printf("docker image:'%s'", newImage)
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
func saveMe(c *hashcache.CheckSumCache, saveDir string, platform string) error {
	binaryDst := filepath.Join(saveDir, ArtefactorBinaryName)
	// detect if the binary we are saving with matches target platform...
	if fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH) == platform {
		me, _ := os.Executable()
		if err := copyBin(c, me, saveDir); err != nil {
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
			return fmt.Errorf("problem creating temp dir for artefactor downloads: %s", err)
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

		// Finally move the file to the correct download path:
		if err := util.Mv(tmpBinPath, binaryDst); err != nil {
			return fmt.Errorf(
				"unable to move from %s to %s:%s",
				tmpBinPath,
				binaryDst,
				err)
		}

		if _, err := c.Update(binaryDst); err != nil {
			return fmt.Errorf("unable to update hash for %s:%s", binaryDst, err)
		}
		if err := util.BinMark(c, binaryDst); err != nil {
			return fmt.Errorf(
				"problem creating meta data file for %s:%s", binaryDst, err)
		}
	}
	return nil
}

// copyBin will save binary meta-data for a local binary to the archive dir
func copyBin(c *hashcache.CheckSumCache, srcBin string, saveDir string) error {
	savedBin := filepath.Join(saveDir, filepath.Base(srcBin))
	if err := util.Cp(srcBin, savedBin); err != nil {
		return err
	}
	if _, err := c.Update(savedBin); err != nil {
		return err
	}
	err := util.BinMark(c, savedBin)
	return err
}

// saveSavedPath will record meta-data so files are restored to the same
// relative path
func saveSavedPath(c *hashcache.CheckSumCache, saveDir string) error {
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
	_, err = c.Update(saveMetaFilePath)
	return err
}

// getSavedPath will retrieve the saved path from the meta-data file.
func getRelativeSavedPath(srcDir string) (string, error) {
	b, err := ioutil.ReadFile(filepath.Join(srcDir, SaveDirMetaFile))
	if err != nil {
		return "", err
	}
	return string(b), nil
}
