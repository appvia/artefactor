package hashcache

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	CheckSumFileName = "checksum.txt"
)

var (
	CheckSumFilePath = CheckSumFileName
	CheckSums        = make(map[string]string)
)

// IsCached will check if a file is present on disk and in the checksum file
func IsCached(file string) bool {
	// If the file doesn't exist...
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	relFile := setFilePaths(file)
	// If the checksum file doesn't exist
	if _, err := os.Stat(CheckSumFilePath); os.IsNotExist(err) {
		return false
	}
	readCheckSums()
	if _, ok := CheckSums[relFile]; ok {
		return true
	}
	return false
}

// Will write a new cache entry into checksum file
func UpdateCache(file string) error {
	var checksum string
	var err error
	relFile := setFilePaths(file)
	fmt.Printf("updating checksum for %s\n", file)
	if checksum, err = getChecksum(file); err != nil {
		return err
	}
	CheckSums[relFile] = checksum
	writeCheckSums()
	return nil
}

func readCheckSums() {
	csf, err := os.Open(CheckSumFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer csf.Close()
	// open checksum file
	scanner := bufio.NewScanner(csf)
	// Re-init checksums
	CheckSums = make(map[string]string)
	for scanner.Scan() {
		line := scanner.Text()
		hashEntry := strings.Fields(scanner.Text())
		if len(hashEntry) != 2 {
			fmt.Printf("invalid cache entry %s\n", line)
		} else {
			CheckSums[hashEntry[0]] = hashEntry[1]
		}
	}
}

func writeCheckSums() error {
	// Create the file contents from the checksum cache
	contents := ""
	for f, sum := range CheckSums {
		contents = contents + fmt.Sprintf("%s  %s\n", sum, f)
	}
	// Save the file
	err := ioutil.WriteFile(CheckSumFilePath, []byte(contents), 0644)
	if err != nil {
		return err
	}
	return nil
}

// setFilePaths updates CheckSumFilePath and returns a relative path for a file
func setFilePaths(file string) (relativeFile string) {
	CheckSumFilePath = filepath.Dir(file) +
		string(filepath.Separator) +
		CheckSumFileName
	return filepath.Base(file)
}

func getChecksum(file string) (string, error) {
	var f *os.File
	var err error
	if f, err = os.Open(file); err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	sum := fmt.Sprintf("%x", h.Sum(nil))
	return string(sum), nil
}
