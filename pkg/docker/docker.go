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

// Load a container from archive and return the image name
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
			// if this is an image being restored from a RepoDigest address, we need
			// to supplement the image object with the imageID docker has imported it as.
			// if no repoDigest is offered, response body does not contain the imageID
			re := regexp.MustCompile(`sha256:([0-9a-f]{64})`)
			image.ImageID = re.FindString(string(b))
			log.Printf("Added %s to image object with RepoDigest %s", image.ImageID, image.RepoDigest)
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
	imageRef := ""
	if image.RepoDigest != "" {
		// a sha256 RepoDigest will have loaded a blank image repo name and blank tag
		// so we must refer to the image with the ImageID
		imageRef = image.ImageID
	}
	if image.ImageTag != "" {
		if image.RepoDigest != "" {
			// also tag image as RepoDigest reference with prefix
			log.Printf("Setting %s as additional tag for image %s to reference source RepoDigest", image.RepoDigest, imageRef)
			if err := cli.ImageTag(ctx, imageRef, image.NewImageName+":repoDigest-"+image.RepoDigest); err != nil {
				return err
			}
		} else {
			imageRef = image.ImageName + ":" + image.ImageTag
		}
	} else {
		// implicitly the archive can only exist in the case where there is a repoDigest.
		// just set Newtag to RepoDigest reference with prefix
		image.ImageTag = "repoDigest-" + image.RepoDigest
		// if we have a RepoDigest, we tag the image with a reference to the RepoDigest
		// so that there is a reference to the registry digest of the image's source
		log.Printf("Setting ImageTag as %s for ImageID %s as image saved only by RepoDigest reference", image.RepoDigest, image.ImageID)
	}
	log.Printf("Re-tagging image ref: %s => %s", imageRef, image.NewImageName+":"+image.ImageTag)
	err = cli.ImageTag(ctx, imageRef, image.NewImageName+":"+image.ImageTag)
	return err
}

// Check if the docker recorded RepoDigest matches Image struct RepoDigest
func ValidatePublishedRepoDigestMatchesHashcache(image Image) (bool, error) {
	// find recorded RepoDigest
	repoDigests, err := GetClientRepoDigests(image.ImageID)
	if err != nil {
		return false, err
	}
	for i, n := range repoDigests {
		if image.NewImageName+"@sha256:"+image.RepoDigest == n {
			log.Printf("RepoDigest %s matches digest [#%s]: %v", image.RepoDigest, n, i)
			return true, nil
		}
	}
	log.Printf("RepoDigest %s did not match any digests in %#v\n",
		image.RepoDigest,
		repoDigests)
	return false, nil
}
func GetClientRepoDigestsByRegistry(imageID string, registry string) ([]string, error) {
	var newRepoDigests []string
	digests, err := GetClientRepoDigests(imageID)
	if err != nil {
		return nil, err
	}
	log.Printf("Processing RepoDigests for image %s", imageID)
	for _, digest := range digests {
		if strings.HasPrefix(digest, registry+"/") {
			newRepoDigests = append(newRepoDigests, digest)
			log.Printf("RepoDigest '%s' match for requested registry '%s'", digest, registry)
		} else {
			log.Printf("Discarding RepoDigest '%s', does not match requested registry '%s'", digest, registry)
		}
	}
	return newRepoDigests, nil
}

func GetClientRepoDigests(imageID string) ([]string, error) {
	// find recorded RepoDigest
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	ii, rawresp, err := cli.ImageInspectWithRaw(ctx, imageID)
	if err != nil {

		log.Printf("imageInspectWithRaw response: %s, error: %#v", string(rawresp), err)
		return nil, err
	}
	return ii.RepoDigests, nil
}

// GetImages retrieves an image struct array
func GetImages(files []string, registry string) ([]Image, error) {
	images := []Image{}
	for _, file := range files {
		if strings.HasSuffix(file, Ext) {
			image, err := NewImageFromFilePath(file, registry)
			if err != nil {
				fmt.Printf("Error processing docker image from file %s, skipping: %s\n", file, err)
			} else {
				log.Printf("Appending image: %#v", image)
				images = append(images, image)
			}
		}
	}
	return images, nil
}

func NewImageFromFilePath(file string, registry string) (Image, error) {
	imageName, err := FilePathToImageName(file)
	if err != nil {
		return Image{}, err
	}
	fullImgName := StripRepoDigest(imageName)
	bareImageName := StripImageTag(fullImgName)

	image := Image{
		FileName:     file,
		ImageName:    bareImageName,
		ImageTag:     GetImageTag(fullImgName),
		NewImageName: GetNewImageName(bareImageName, registry),
		RepoDigest:   GetRepoDigest(imageName),
	}
	return image, nil
}

func GetNewImageName(image string, registry string) string {
	if registry != "" {
		image = fmt.Sprintf("%s/%s", registry, filepath.Base(image))
	}
	return image
}

func GetImageTag(image string) string {
	simage := strings.Split(image, "/")
	// colons can exist in registry names
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
func IsClientErrNotFound(err error) bool {
	return client.IsErrNotFound(err)
}
