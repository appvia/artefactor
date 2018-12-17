package tar

// Tar takes a source and variable writers and walks 'source' writing each file
import (
	"archive/tar"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/appvia/artefactor/pkg/util"
)

// Create a tar file from file name and array of paths and files to add
func Create(tarFn string, paths []string) error {

	// ensure the src actually exists before trying to tar it
	if len(paths) < 1 {
		return fmt.Errorf("must supply at least one path to archive")
	}
	// Find the first file and use it's directory as the wd
	src, err := filepath.Abs(paths[0])
	if err != nil {
		return err
	}
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("unable to tar files %v", err.Error())
	}
	// create tar file
	tarfile, err := os.Create(tarFn)
	if err != nil {
		return fmt.Errorf("unable to create tar file %v due to %v", tarFn, err)
	}

	defer tarfile.Close()
	var fw io.WriteCloser = tarfile
	tw := tar.NewWriter(fw)
	defer tw.Close()

	// keep track of tar's relative working dir
	var wd string
	if srcInfo.Mode().IsDir() {
		// Add the prefix for the whole repo...
		log.Printf("prefix = %s", srcInfo.Name())
		wd = srcInfo.Name()
	} else {
		wd = filepath.Base(filepath.Dir(src))
	}
	log.Printf("adding prefix from directoy name: %s", wd)

	// add all the files to the archive
	for _, path := range paths {
		if err := addFile(tw, wd, path); err != nil {
			return fmt.Errorf(
				"problem tryig to add %s to archive %s",
				path,
				tarFn)
		}
	}
	return nil
}

// Extract a tar file to the dst directory
func Extract(tarFn string, dst string) error {

	log.Printf("Opening tar %s", tarFn)
	tarFile, err := os.Open(tarFn)
	if err != nil {
		return err
	}
	tr := tar.NewReader(tarFile)
	for {
		header, err := tr.Next()

		switch {
		// if no more files are found return
		case err == io.EOF:
			return nil
		// return any other error
		case err != nil:
			return err
		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)
		log.Printf("Creating %s", target)

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0775); err != nil {
					return err
				}
			}

		case tar.TypeLink:
			dir := filepath.Dir(target)
			dest := header.Linkname
			log.Printf("link dir %s file name: %s links to %s", dir, target, dest)
			// Check the path exists first
			if _, err := os.Stat(dir); err != nil {
				if err := os.MkdirAll(dir, 0775); err != nil {
					return err
				}
			}
			// Destination may not exist until all of tar is extracted
			// os.Symlink cannot create symlink when destination doesn't exists!!!???
			if err := util.SymLink(target, dest); err != nil {
				return err
			}

		// if it's a file create it
		case tar.TypeReg:
			dir := filepath.Dir(target)
			// Check the path exists first
			if _, err := os.Stat(dir); err != nil {
				if err := os.MkdirAll(dir, 0775); err != nil {
					return err
				}
			}
			tarFile, err := os.OpenFile(
				target,
				os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			defer tarFile.Close()

			// copy over contents
			if _, err := io.Copy(tarFile, tr); err != nil {
				return err
			}
		}
	}
}

// addFile to an archive using a tar.Writer
func addFile(
	tw *tar.Writer,
	prefix string,
	path string) error {

	// Ensure path exists
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	header, err := tar.FileInfoHeader(fi, path)
	if err != nil {
		return err
	}
	if len(header.Linkname) > 0 {
		log.Printf("adding link:%s to %s", header.Linkname, header.Name)
		header.Typeflag = tar.TypeLink
		// Link name isn't right - let's make sure it contains the path
		header.Linkname, _ = os.Readlink(path)
	}
	// update the name to correctly reflect the desired destination when untaring
	if len(prefix) > 0 {
		header.Name = filepath.Join(prefix, path)
	}
	// write the header
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	// return on non-regular files (thanks to [kumo](https://medium.com/@komuw/just-like-you-did-fbdd7df829d3) for this suggested update)
	if !fi.Mode().IsRegular() {
		log.Printf("Skipping irregular file %s", path)
		return nil
	}

	// open files for taring
	srcFile, err := os.Open(path)
	defer srcFile.Close()
	if err != nil {
		return err
	}

	// copy file data into tar writer
	if _, err := io.Copy(tw, srcFile); err != nil {
		return err
	}
	log.Printf("File written %s", path)
	tw.Flush()

	return nil
}
