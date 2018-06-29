package docker

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/appvia/artefactor/pkg/hashcache"
	"github.com/docker/docker/client"
)

type Image struct {
	FileName     string
	ImageName    string
	NewImageName string
}

// Load a conatiner from archive and return the image name
func Load(file string) error {
	if _, err := os.Stat(file); err != nil {
		return err
	}

	// Get docker client
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}
	r, err := os.Open(file)
	if err != nil {
		return err
	}
	response, err := cli.ImageLoad(ctx, r, true)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.Body != nil && response.JSON {
		// Decode messages until complete...?
		b, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("error reading response from docker daemon:%s", err)
		}
		log.Printf(string(b))
	}
	return nil
}

func ReTag(image Image) error {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}
	err = cli.ImageTag(ctx, image.ImageName, image.NewImageName)
	return err
}

// GetImages retireves a list of image structs
func GetImages(path string, registry string) ([]Image, error) {
	images := []Image{}
	files := hashcache.GetFiles(path)
	for _, file := range files {
		if strings.HasSuffix(file, Ext) {
			imageName, err := FilePathToImageName(file)
			if err != nil {
				return nil, err
			}
			images = append(images, Image{
				FileName:     filepath.Join(path, file),
				ImageName:    imageName,
				NewImageName: getNewImageName(imageName, registry),
			})
		}
	}
	return images, nil
}

func getNewImageName(image string, registry string) string {
	newImage := fmt.Sprintf("%s/%s", registry, filepath.Base(image))
	return newImage
}
