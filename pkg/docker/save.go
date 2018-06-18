package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/appvia/artefactor/pkg/hashcache"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type Event struct {
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
func Save(image string, dir string) error {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return (err)
	}
	events, err := cli.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return (err)
	}
	d := json.NewDecoder(events)
	em := make(map[string]*Event)
	var event *Event
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
			fmt.Printf("%+v\n", event.Status)
		}
		lastStatus = event.Status
	}
	ior, err := cli.ImageSave(ctx, []string{image})
	if err != nil {
		return err
	}
	archiveFile, err := ImageToFilePath(image, dir)
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
	if hashcache.UpdateCache(archiveFile); err != nil {
		return err
	}

	return nil
}
