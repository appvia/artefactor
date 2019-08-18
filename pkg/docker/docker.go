package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/appvia/artefactor/pkg/hashcache"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
)

type Image struct {
	FileName     string
	ImageID      string
	ImageName    string
	ImageTag     string
	NewImageName string
	RepoDigest   string
}

// Load a conatiner from archive and return the image name
func Load(image *Image) error {

	if _, err := os.Stat(image.FileName); err != nil {
		return err
	}

	// Get docker client
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	r, err := os.Open(image.FileName)
	if err != nil {
		return err
	}
	response, err := cli.ImageLoad(ctx, r, true)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.Body != nil && response.JSON {
		b, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("error reading response from docker daemon:%s", err)
		}
		var apiMessage jsonmessage.JSONMessage
		if err := json.Unmarshal(b, &apiMessage); err != nil {
			return fmt.Errorf("error decoding response from docker daemon:%s", err)
		}
		if apiMessage.Error != nil {
			return fmt.Errorf("error loading image: %s", apiMessage.Error.Message)
		}
		if image.RepoDigest != "" {
			// if this is an image being restored from a repodigest address, we need
			// to suppliment the image object with the imageID docker has imported it as.
			// this gives us the basis of a tag to retag it with also.
			re := regexp.MustCompile(`sha256:([0-9a-f]{64})`)
			image.ImageID = re.FindString(string(b))
			log.Printf("Added %s to image object with repodigest %s\n", image.ImageID, image.RepoDigest)
		}
		log.Print(string(b))
	} else {
		return fmt.Errorf("empty/invalid response from docker daemon")
	}
	return nil
}

func ReTag(image *Image) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	if image.RepoDigest != "" && image.ImageTag == "" {
		image.ImageTag = image.ImageID[7:19]
		log.Printf("Setting %s ImageTag to short sha %s from ImageID: %s as image saved by RepoDigest reference", image.NewImageName, image.ImageTag, image.ImageID)
	}
	imageRef := ""
	if image.RepoDigest != "" {
		//a sha256 repodigest will have loaded a blank image repo name and blank tag
		//so we must refer to the image with the ImageID
		imageRef = image.ImageID
	} else {
		imageRef = image.ImageName + ":" + image.ImageTag
	}
	err = cli.ImageTag(ctx, imageRef, image.NewImageName+":"+image.ImageTag)
	return err
}

func ValidatePublishedRepoDigest(image Image) error {
	//find recorded repoDigest
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	ii, rawresp, err := cli.ImageInspectWithRaw(ctx, image.ImageID)
	if err != nil {
		log.Print(string(rawresp))
		return err
	}
	for i, n := range ii.RepoDigests {
		if image.NewImageName+"@sha256:"+image.RepoDigest == n {
			log.Printf("RepoDigest %s matches %v", image.RepoDigest, ii.RepoDigests[i])
			return nil
		}
	}

	return nil
}

// GetImages retireves a list of image structs
func GetImages(path string, registry string) ([]Image, error) {
	images := []Image{}
	files := hashcache.GetFiles(path)
	for _, file := range files {
		if strings.HasSuffix(file, Ext) {
			imageName, err := FilePathToImageName(file)
			fullImgName := StripRepoDigest(imageName)
			bareImageName := StripImageTag(fullImgName)
			if err != nil {
				return nil, err
			}
			tmpimage := Image{
				FileName:     file,
				ImageName:    bareImageName,
				ImageTag:     GetImageTag(fullImgName),
				NewImageName: GetNewImageName(bareImageName, registry),
				RepoDigest:   GetRepoDigest(imageName),
			}
			images = append(images, tmpimage)
		}
	}
	return images, nil
}

func GetNewImageName(image string, registry string) string {
	newImage := fmt.Sprintf("%s/%s", registry, filepath.Base(image))
	return newImage
}

func GetImageTag(image string) string {
	simage := strings.Split(image, "/")
	//colons can exist in registry names
	if strings.Contains(simage[len(simage)-1], ":") {
		return strings.Split(simage[len(simage)-1], ":")[1]
	}
	log.Printf("image %s does not contain a tag", image)
	return ""
}

func StripImageTag(image string) string {
	return strings.Split(image, ":")[0]
}

func StripRepoDigest(image string) string {
	return strings.Split(image, ShaIdent)[0]
}

func GetRepoDigest(image string) string {
	if strings.Contains(image, ShaIdent) {
		return strings.Split(image, ShaIdent)[1]
	}
	return ""
}
