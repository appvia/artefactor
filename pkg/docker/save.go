package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/appvia/artefactor/pkg/hashcache"
	"github.com/appvia/artefactor/pkg/util"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type SaveEvent struct {
	Status         string `json:"status"`
	Error          string `json:"error"`
	Progress       string `json:"progress"`
	ProgressDetail struct {
		Current int `json:"current"`
		Total   int `json:"total"`
	} `json:"progressDetail"`
	Id string `json:"id"`
}

// Save will save a docker image
func Save(c *hashcache.CheckSumCache, image string, dir string, creds *util.Creds) error {

	archiveFile, err := ImageToFilePath(image, dir)
	if err != nil {
		return fmt.Errorf("error getting image name from %s and %s:%s\n",
			image,
			dir,
			err)
	}
	if _, err := os.Stat(archiveFile); err == nil {
		// docker tar exists, just check the previous checksum exists / correct
		if c.IsCachedMatchingFile(archiveFile) {
			fmt.Printf("file already downloaded and matching checksum:%+v\n", archiveFile)
			c.Keep(archiveFile)
			return nil
		}
	}

	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return (err)
	}
	// Load auth details from .docker config
	var ipo types.ImagePullOptions
	if creds != nil {
		if auth, err := GetAuthString(
			image, creds.Username, creds.Password); err != nil {
			return fmt.Errorf("error with credentials provided:%s", err)
		} else {
			ipo.RegistryAuth = auth
		}
	} else {
		ipo.RegistryAuth = GetAuth(image)
	}
	events, err := cli.ImagePull(ctx, image, ipo)
	if err != nil {
		return (err)
	}
	d := json.NewDecoder(events)
	em := make(map[string]*SaveEvent)
	var event *SaveEvent
	lastStatus := ""
	for {
		if err := d.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}

			return err
		}
		em[event.Status] = event
		if event.Status != lastStatus {
			fmt.Printf("%+v (%s)\n", event.Status, event.Id)
		}
		lastStatus = event.Status
	}
	ior, err := cli.ImageSave(ctx, []string{image})
	if err != nil {
		return err
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0744); err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	fmt.Printf("Saving to archive:%+v\n", archiveFile)
	outFile, err := os.Create(archiveFile)
	// handle err
	defer outFile.Close()
	if err != nil {
		return err
	}
	_, err = io.Copy(outFile, ior)
	if err != nil {
		return err
	}
	// Update the cache with checksum
	_, err = c.Update(archiveFile)
	return err
}
