package docker

import (
	"testing"
)

func TestImageToFilePath(t *testing.T) {
	type Image struct {
		name         string
		expectedfile string
		dir          string
	}

	images := []Image{
		{"alpine", "alpine.tar", ""},
		{"alpine:latest", "alpine~~latest.tar", ""},
		{"registry/alpine:latest", "registry~alpine~~latest.tar", ""},
		{"dns.registry/alpine:latest", "dns.registry~alpine~~latest.tar", ""},
		{"dns.registry/alpine:latest", "dns.registry~alpine~~latest.tar", ""},
		{"alpine", "dir/alpine.tar", "dir"},
	}

	for _, image := range images {
		fileName, err := ImageToFilePath(image.name, image.dir)
		if err != nil {
			t.Fatal(err)
		}

		// Ensure the filename dir IS the same as the download dir
		if fileName != "" {
			if fileName != image.expectedfile {
				t.Errorf("Expecting %v but got %v", image.expectedfile, fileName)
			}
		} else {
			t.Errorf("Expecting %v but got %v", image.expectedfile, fileName)
		}
	}
}
