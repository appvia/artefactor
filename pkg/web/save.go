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
	c *hashcache.CheckSumCache,
	url string,
	fileName string,
	dir string,
	sha256 string,
	binFile bool) error {

	download := fmt.Sprintf("%s/%s", dir, fileName)
	// Check checksum cache first...
	if c.IsCachedMatched(download, sha256) {
		fmt.Printf("file %q in cache and matching checksum %s\n", download, sha256)
		// Make sure we tell cache to keep this item:
		c.Keep(download)
		if binFile {
			util.BinMark(c, download)
		}
		return nil
	} else {
		if c.IsCached(download) {
			fmt.Printf("file %q is in cache but does NOT match checksum %s\n", download, sha256)
			// need to delete file for now and manage partial recovery logic in lib...
			fmt.Printf("deleting file %q\n", download)
			os.Remove(download)
		} // else not cached...
	}

	if err := SaveNoCheck(url, download, binFile); err != nil {
		return fmt.Errorf("download problem:%s", err)
	}

	// Save the file mode meta data if it's a binary
	if binFile {
		if err := util.BinMark(c, download); err != nil {
			return err
		}
	}

	// Now the file is updated - update the checksum...
	hash, err := c.Update(download)
	if err != nil {
		return err
	}
	if !c.IsCachedMatched(download, sha256) {
		log.Printf("invalid checksum (%s) for %s, expecting %q", sha256, download, hash)
		return errors.Errorf("invalid checksum for %s", download)
	}
	fmt.Printf("File checksum ok for %q\n", download)
	return nil
}

// SaveNoCheck will download a file without verifying checksums
func SaveNoCheck(
	url string,
	download string,
	binFile bool,
) error {
	tmpDownload := download + ".download"
	client := grab.NewClient()
	req, _ := grab.NewRequest(tmpDownload, url)

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
	if _, err := os.Stat(download); err == nil {
		if rmErr := os.Remove(download); rmErr != nil {
			return fmt.Errorf(
				"can not remove %q. Trying to update from %q",
				download,
				resp.Filename)
		}
		if util.Mv(resp.Filename, download); err != nil {
			return err
		}
	}
	fmt.Printf("Download saved to %v \n", download)
	if binFile {
		// Update the executable mode:
		if err := os.Chmod(download, 0777); err != nil {
			return errors.Errorf("can't set executable permissions")
		}
	}
	return nil
}
