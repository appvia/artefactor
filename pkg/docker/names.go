package docker

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	safePathSep string = "~"
	safeVerSep  string = "~~"
	Ext         string = ".docker.tar"
)

// ImageToFileName provides an archived name from a docker image name
func ImageToFilePath(imageName string, dir string) (fileName string, err error) {
	flatImageName := strings.Replace(imageName, string(os.PathSeparator), safePathSep, -1)
	flatImageName = strings.Replace(flatImageName, ":", safeVerSep, 1)
	if len(dir) > 0 {
		flatImageName = dir + string(os.PathSeparator) + flatImageName + Ext
	} else {
		flatImageName = flatImageName + Ext
	}
	return flatImageName, nil
}

// FileNameToImageName converts an archived file name back to the origonal docker image name
func FilePathToImageName(fileName string) (imageName string, err error) {
	if strings.Contains(fileName, string(filepath.Separator)) {
		imageDirName := filepath.Dir(fileName)
		dirL := len(imageDirName)
		if dirL > 0 {
			// Strip off leading dir:
			imageName = fileName[(len(imageDirName) + 1):]
		}
	} else {
		imageName = fileName
	}
	// Decode version seperator
	imageName = strings.Replace(imageName, safeVerSep, ":", 1)
	// Decode registry path
	imageName = strings.Replace(imageName, safePathSep, string(os.PathSeparator), -1)
	// Remove extension
	imageName = strings.Replace(imageName, Ext, "", 1)
	return imageName, nil
}
