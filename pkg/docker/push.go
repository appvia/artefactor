package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/appvia/artefactor/pkg/util"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type PushEvent struct {
	Status         string `json:"status"`
	Error          string `json:"error"`
	Progress       string `json:"progress"`
	ProgressDetail struct {
		Current int `json:"current"`
		Total   int `json:"total"`
	} `json:"progressDetail"`
	Id string `json:"id"`
}

// Push will push a docker image
func Push(image string, creds *util.Creds) error {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return (err)
	}
	// Load auth details from .docker config
	var ipo types.ImagePushOptions
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
	events, err := cli.ImagePush(ctx, image, ipo)
	if err != nil {
		return (err)
	}
	d := json.NewDecoder(events)
	em := make(map[string]*PushEvent)
	var event *PushEvent
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
	return nil
}
