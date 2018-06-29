package web

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/appvia/artefactor/pkg/hashcache"
	"github.com/appvia/artefactor/pkg/util"
	"github.com/cavaliercoder/grab"
	"github.com/pkg/errors"
)

// Save will save a file from the web and optionaly set executable mode
func Save(
	url string,
	fileName string,
	dir string,
	sha256 string,
	binFile bool) error {

	download := fmt.Sprintf("%s/%s", dir, fileName)
	// Check checksum cache first...
	if hashcache.IsCachedMatch(download, sha256) {
		fmt.Printf("file %q in cache and matching checksum %s\n", download, sha256)
		if binFile {
			util.BinMark(download)
		}
		return nil
	} else {
		if hashcache.IsCached(download) {
			fmt.Printf("file %q is in cache but does NOT match checksum %s", download, sha256)
			// need to delete file for now and manage partial recovery logic in lib...
			fmt.Printf("deleting file %q", download)
			os.Remove(download)
		} // else not cached...
	}

	client := grab.NewClient()
	req, _ := grab.NewRequest(download, url)

	// start download
	fmt.Printf("Downloading %q...\n", req.URL())
	resp := client.Do(req)
	fmt.Printf("  %v\n", resp.HTTPResponse.Status)

	// start UI loop
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()

Loop:
	for {
		select {
		case <-t.C:
			fmt.Printf("  transferred %v / %v bytes (%.2f%%)\n",
				resp.BytesComplete(),
				resp.Size,
				100*resp.Progress())

		case <-resp.Done:
			// download is complete
			break Loop
		}
	}
	// check for errors
	if err := resp.Err(); err != nil {
		return err
	}
	fmt.Printf("Download saved to %v \n", resp.Filename)
	if binFile {
		// Update the executable mode:
		if err := os.Chmod(download, 0777); err != nil {
			return errors.Errorf("can't set executable permissions")
		}
	}
	// Now the file is updated - update the checksum...
	hash, err := hashcache.UpdateCache(download)
	if err != nil {
		return err
	}
	if !hashcache.IsCachedMatch(download, sha256) {
		log.Printf("invalid checksum (%s) for %s, expecting %q", sha256, download, hash)
		return errors.Errorf("invalid checksum for %s", download)
	} else {
		fmt.Printf("File checksum ok for %q", download)
	}
	if binFile {
		if err := util.BinMark(download); err != nil {
			return err
		}
	}
	return nil
}
