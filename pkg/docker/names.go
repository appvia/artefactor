package docker

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	safePathSep  string = "~"
	safeVerSep   string = "~~"
	safeShaIdent string = "~~~sha256~"
	Ext          string = ".docker.tar"
)

// ImageToFileName provides an archived name from a docker image name
func ImageToFilePath(imageName string, dir string) (fileName string, err error) {
	imageName = strings.Replace(imageName, "@sha256:", safeShaIdent, 1)
	imageName = strings.Replace(imageName, string(os.PathSeparator), safePathSep, -1)
	imageName = strings.Replace(imageName, ":", safeVerSep, 1)

	if len(dir) > 0 {
		imageName = dir + string(os.PathSeparator) + imageName + Ext
	} else {
		imageName = imageName + Ext
	}
	return imageName, nil
}

// FileNameToImageName converts an archived file name back to the origonal docker image name
func FilePathToImageName(fileName string) (imageName string, err error) {
	// obtain the image name portion of the path
	imageName = filepath.Base(fileName)
	// Decode addressable content sha identifiers
	imageName = strings.Replace(imageName, safeShaIdent, "@sha256:", -1)
	// Decode version seperator
	imageName = strings.Replace(imageName, safeVerSep, ":", 1)
	// Decode registry path
	imageName = strings.Replace(imageName, safePathSep, string(os.PathSeparator), -1)
	// Remove extension
	imageName = strings.Replace(imageName, Ext, "", 1)
	return imageName, nil
}
