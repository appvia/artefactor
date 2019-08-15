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
		{"alpine", "alpine.docker.tar", ""},
		{"alpine:latest", "alpine~~latest.docker.tar", ""},
		{"registry/alpine:latest", "registry~alpine~~latest.docker.tar", ""},
		{"dns.registry/alpine:latest", "dns.registry~alpine~~latest.docker.tar", ""},
		{"dns.registry/alpine:latest", "dns.registry~alpine~~latest.docker.tar", ""},
		{"alpine", "dir/alpine.docker.tar", "dir"},
		{"busybox@sha256:9f1003c480699be56815db0f8146ad2e22efea85129b5b5983d0e0fb52d9ab70", "busybox~~~sha256~9f1003c480699be56815db0f8146ad2e22efea85129b5b5983d0e0fb52d9ab70.docker.tar", ""},
		{"dns.registry/busybox@sha256:9f1003c480699be56815db0f8146ad2e22efea85129b5b5983d0e0fb52d9ab70", "dns.registry~busybox~~~sha256~9f1003c480699be56815db0f8146ad2e22efea85129b5b5983d0e0fb52d9ab70.docker.tar", ""},
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

func TestFilePathToImageName(t *testing.T) {
	type File struct {
		path          string
		expectedImage string
	}

	files := []File{
		{"alpine.docker.tar", "alpine"},
		{"alpine~~latest.docker.tar", "alpine:latest"},
		{"registry~alpine~~latest.docker.tar", "registry/alpine:latest"},
		{"dns.registry~alpine~~latest.docker.tar", "dns.registry/alpine:latest"},
		{"dns.registry~alpine~~latest.docker.tar", "dns.registry/alpine:latest"},
		{"dir/alpine.docker.tar", "alpine"},
		{"busybox~~~sha256~9f1003c480699be56815db0f8146ad2e22efea85129b5b5983d0e0fb52d9ab70.docker.tar", "busybox@sha256:9f1003c480699be56815db0f8146ad2e22efea85129b5b5983d0e0fb52d9ab70"},
		{"dns.registry~busybox~~~sha256~9f1003c480699be56815db0f8146ad2e22efea85129b5b5983d0e0fb52d9ab70.docker.tar", "dns.registry/busybox@sha256:9f1003c480699be56815db0f8146ad2e22efea85129b5b5983d0e0fb52d9ab70"}}
	for _, file := range files {
		imageName, err := FilePathToImageName(file.path)
		if err != nil {
			t.Fatal(err)
		}

		// Ensure the filename dir IS the same as the download dir
		if imageName != "" {
			if imageName != file.expectedImage {
				t.Errorf("Expecting %v but got %v", file.expectedImage, imageName)
			}
		} else {
			t.Errorf("Expecting %v but got %v", file.expectedImage, imageName)
		}
	}

}
