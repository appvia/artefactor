package docker

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"strings"

	c "github.com/fsouza/go-dockerclient"
)

// GetConfig loads config for a given registry from the Docker config file
func GetAuth(image string) string {
	registry := strings.Split(image, "/")[0]
	auths, _ := c.NewAuthConfigurationsFromDockerCfg()

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
	return ""
}
