package docker

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"runtime"
	"strings"

	"github.com/docker/docker-credential-helpers/client"
	c "github.com/fsouza/go-dockerclient"
)

// GetAuth loads config for a given registry from the Docker config file
func GetAuth(image string) string {
	registry := strings.Split(image, "/")[0]
	auths, err := c.NewAuthConfigurationsFromDockerCfg()
	if err != nil {
		log.Printf("warning (error getting authentication details: %s)", err)
	}
	if auths != nil {
		// If we got any auth configs back, see if we have the required auth
		for key, value := range auths.Configs {
			log.Printf("auth key %s", key)
			if key == registry {
				log.Printf("found auth for server %s", registry)
				encodedJSON, err := json.Marshal(value)
				if err != nil {
					log.Printf("problem parsing auths")
				}
				authStr := base64.URLEncoding.EncodeToString(encodedJSON)
				return authStr
			}
		}
	}
	if runtime.GOOS == "darwin" {
		log.Printf("OSX detected...")
		// No credentials found thus far now try native OS credential helpers:
		p := client.NewShellProgramFunc("docker-credential-osxkeychain")
		creds, err := client.Get(p, "https://"+registry)
		if err != nil {
			log.Printf(
				"warning, error when trying to get credentials from osxkeychain:%s",
				err)
		}
		if creds == nil {
			log.Printf("no creds returned from keychain")
			return ""
		}
		log.Printf(
			"auth details retrieved from keychain for username:%q",
			creds.Username)
		if authStr, err := GetAuthString(
			registry, creds.Username, creds.Secret); err != nil {
			log.Printf("problem parsing auths %s", err)
			return ""
		} else {
			return authStr
		}
	}
	return ""
}

// GetAuthString will return a valid auth string from credentials
func GetAuthString(image, username, password string) (string, error) {
	registry := strings.Split(image, "/")[0]
	encodedJSON, err := json.Marshal(c.AuthConfiguration{
		Username:      username,
		Password:      password,
		ServerAddress: registry,
	})
	if err != nil {
		return "", err
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)
	log.Printf("auth string:%q", authStr)
	return authStr, nil
}
