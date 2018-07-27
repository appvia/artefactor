package util

import (
	"fmt"
	"io"
	"os"

	"github.com/appvia/artefactor/pkg/hashcache"
)

// Cp copies a file
func Cp(src string, dst string) error {
	from, err := os.Open(src)
	if err != nil {
		return err
	}
	defer from.Close()
	fi, _ := from.Stat()

	to, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE, fi.Mode().Perm())
	if err != nil {
		return err
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		return err
	}
	return nil
}

// Mv wraps the os.rename and will copy on error
func Mv(src string, dst string) error {
	if err := os.Rename(src, dst); err != nil {
		// If the volumes are different
		if cpErr := Cp(src, dst); cpErr != nil {
			return fmt.Errorf(
				"problem with copying file when trying to move %q to %q:%s",
				src,
				dst,
				cpErr)
		}
		// Now delete the src...
		if rmErr := os.Remove(src); rmErr != nil {
			return fmt.Errorf(
				"can not remove source file %q when moving file to %q:%s",
				src,
				dst,
				rmErr)
		}
	}
	return nil
}

// BinMark marks a file as binary and update meta data checksum
func BinMark(c *hashcache.CheckSumCache, download string) error {
	binMark := download + ".binmark.meta"
	os.Create(binMark)
	_, err := c.Update(binMark)
	return err
}
