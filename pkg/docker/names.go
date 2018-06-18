package docker

import (
	"fmt"
	"os"
	"strings"
)

const (
	safePathSep string = "~"
	safeVerSep  string = "~~"
	ext         string = ".tar"
	md5         string = ".md5"
)

// ImageToFileName provides an archived name from a docker image name
func ImageToFilePath(imageName string, dir string) (fileName string, err error) {
	flatImageName := strings.Replace(imageName, string(os.PathSeparator), safePathSep, -1)
	flatImageName = strings.Replace(flatImageName, ":", safeVerSep, 1)
	if len(dir) > 0 {
		flatImageName = dir + string(os.PathSeparator) + flatImageName + ext
	} else {
		flatImageName = flatImageName + ext
	}
	return flatImageName, nil
}

// FileNameToImageName converts an archived file name back to the origonal docker image name
func FilePathToImageName(fileName string, dir string) (imageName string, err error) {

	// Check the dir is expected
	if !strings.HasPrefix(fileName, dir) {
		return "", fmt.Errorf(
			"unexpected path for file %s (must match dir %v)", fileName, dir)
	}

	// Strip off leading dir:
	imageNameEnc := fileName[(len(dir) + 1):len(fileName)]
	// Decode version seperator
	imageName = strings.Replace(imageNameEnc, safeVerSep, ":", 1)
	// Decode registry path
	imageName = strings.Replace(imageName, safePathSep, string(os.PathSeparator), -1) + ext

	return imageName, nil
}
