package docker_test

import (
	"fmt"
	"testing"

	"github.com/appvia/artefactor/pkg/docker"
	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
)

func TestNewImageFromFilePath(t *testing.T) {
	cases := []struct {
		File,
		Registry string
		expImage docker.Image
	}{
		{
			"downloads/circleci~golang~~~sha256~be7f30e6cbaed8d8e2537d857c6507fb57dcbc1ceb24a0de35ebe30cc75dba12.docker.tar",
			"localhost:5000",
			docker.Image{
				FileName:     "downloads/circleci~golang~~~sha256~be7f30e6cbaed8d8e2537d857c6507fb57dcbc1ceb24a0de35ebe30cc75dba12.docker.tar",
				ImageID:      "",
				ImageName:    "circleci/golang",
				ImageTag:     "",
				NewImageName: "localhost:5000/golang",
				RepoDigest:   "be7f30e6cbaed8d8e2537d857c6507fb57dcbc1ceb24a0de35ebe30cc75dba12",
			},
		},
		{
			"alpine~~latest.docker.tar",
			"localhost:5000",
			docker.Image{
				FileName:     "alpine~~latest.docker.tar",
				ImageID:      "",
				ImageName:    "alpine",
				ImageTag:     "latest",
				NewImageName: "localhost:5000/alpine",
				RepoDigest:   "",
			},
		},
		{
			"busybox~~~sha256~9f1003c480699be56815db0f8146ad2e22efea85129b5b5983d0e0fb52d9ab70.docker.tar",
			"",
			docker.Image{
				FileName:     "busybox~~~sha256~9f1003c480699be56815db0f8146ad2e22efea85129b5b5983d0e0fb52d9ab70.docker.tar",
				ImageID:      "",
				ImageName:    "busybox",
				ImageTag:     "",
				NewImageName: "busybox",
				RepoDigest:   "9f1003c480699be56815db0f8146ad2e22efea85129b5b5983d0e0fb52d9ab70",
			},
		},
		{
			"downloads/busybox~~~sha256~9f1003c480699be56815db0f8146ad2e22efea85129b5b5983d0e0fb52d9ab70.docker.tar",
			"",
			docker.Image{
				FileName:     "downloads/busybox~~~sha256~9f1003c480699be56815db0f8146ad2e22efea85129b5b5983d0e0fb52d9ab70.docker.tar",
				ImageID:      "",
				ImageName:    "busybox",
				ImageTag:     "",
				NewImageName: "busybox",
				RepoDigest:   "9f1003c480699be56815db0f8146ad2e22efea85129b5b5983d0e0fb52d9ab70",
			},
		},
		{

			"dns.registry~alpine~~latest.docker.tar",
			"",
			docker.Image{
				FileName:     "dns.registry~alpine~~latest.docker.tar",
				ImageID:      "",
				ImageName:    "dns.registry/alpine",
				ImageTag:     "latest",
				NewImageName: "dns.registry/alpine",
				RepoDigest:   "",
			},
		},
		{
			"dir/alpine.docker.tar",
			"",
			docker.Image{
				FileName:     "dir/alpine.docker.tar",
				ImageID:      "",
				ImageName:    "alpine",
				ImageTag:     "",
				NewImageName: "alpine",
				RepoDigest:   "",
			},
		},
		{
			"downloads/dns.registry~busybox~~~sha256~9f1003c480699be56815db0f8146ad2e22efea85129b5b5983d0e0fb52d9ab70.docker.tar",
			"",
			docker.Image{
				FileName:     "downloads/dns.registry~busybox~~~sha256~9f1003c480699be56815db0f8146ad2e22efea85129b5b5983d0e0fb52d9ab70.docker.tar",
				ImageID:      "",
				ImageName:    "dns.registry/busybox",
				ImageTag:     "",
				NewImageName: "dns.registry/busybox",
				RepoDigest:   "9f1003c480699be56815db0f8146ad2e22efea85129b5b5983d0e0fb52d9ab70",
			},
		},
		{
			"downloads/dns.registry~busybox~~~sha256~9f1003c480699be56815db0f8146ad2e22efea85129b5b5983d0e0fb52d9ab70.docker.tar",
			"localhost:5000",
			docker.Image{
				FileName:     "downloads/dns.registry~busybox~~~sha256~9f1003c480699be56815db0f8146ad2e22efea85129b5b5983d0e0fb52d9ab70.docker.tar",
				ImageID:      "",
				ImageName:    "dns.registry/busybox",
				ImageTag:     "",
				NewImageName: "localhost:5000/busybox",
				RepoDigest:   "9f1003c480699be56815db0f8146ad2e22efea85129b5b5983d0e0fb52d9ab70",
			},
		},
		{
			"downloads/dns.registry~busybox~~1.31-musl.docker.tar",
			"localhost:5000",
			docker.Image{
				FileName:     "downloads/dns.registry~busybox~~1.31-musl.docker.tar",
				ImageID:      "",
				ImageName:    "dns.registry/busybox",
				ImageTag:     "1.31-musl",
				NewImageName: "localhost:5000/busybox",
				RepoDigest:   "",
			},
		},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s %s", tc.File, tc.Registry), func(t *testing.T) {
			actual, err := docker.NewImageFromFilePath(tc.File, tc.Registry)
			if err != nil {
				t.Fatalf("Generating image failed with error: %s", err)
			}
			assert.Assert(t, cmp.Equal(actual, tc.expImage))
		})
	}

	// {"dns.registry~busybox~~~sha256~9f1003c480699be56815db0f8146ad2e22efea85129b5b5983d0e0fb52d9ab70.docker.tar", "dns.registry/busybox@sha256:9f1003c480699be56815db0f8146ad2e22efea85129b5b5983d0e0fb52d9ab70"}
}
