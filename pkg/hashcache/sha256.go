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
	// CheckSumFileName is the checksum file name (base with no directories)
	CheckSumFileName = "checksum.txt"
)

var (
	// CheckSumFilePath is the Checksum file name
	CheckSumFilePath = CheckSumFileName
	// CheckSums is the hash that stores checksums keyed on relative path
	CheckSums = make(map[string]string)
)

// IsCachedMatch will verify if a file is in Cache AND matching expected sha256
func IsCachedMatch(file string, sha256 string) bool {
	// Get relative path from directory if set
	relFile := setFilePaths(file)
	inCache := IsCached(file)
	if inCache {
		if CheckSums[relFile] == sha256 {
			return true
		}
	}
	return false
}

// IsCached will check if a file is present on disk and in the checksum file
func IsCached(file string) bool {
	// If the file doesn't exist...
	if _, err := os.Stat(file); os.IsNotExist(err) {
		log.Printf("File %q doesn't exist", file)
		return false
	}
	relFile := setFilePaths(file)
	log.Printf("File relative path (for cache) is %q", relFile)
	// If the checksum file doesn't exist
	if _, err := os.Stat(CheckSumFilePath); os.IsNotExist(err) {
		log.Printf("Checksum file %q doesn't exist", CheckSumFilePath)
		return false
	}
	readCheckSums()
	if _, ok := CheckSums[relFile]; ok {
		log.Printf("Cache hit for %q", relFile)
		return true
	} else {
		log.Printf("Cache MISS for %q", relFile)
	}
	return false
}

// GetCachedChecksum will return previously calculated checksum
func GetCachedChecksum(file string) (string, error) {
	if IsCached(file) {
		relFile := setFilePaths(file)
		sum, present := CheckSums[relFile]
		if present {
			return sum, nil
		}
	}
	return "", fmt.Errorf("no checksum exists for file entry %s", file)
}

// UpdateCache will write a new cache entry into checksum file
func UpdateCache(file string) (string, error) {
	var checksum string
	var err error
	relFile := setFilePaths(file)
	readCheckSums()
	fmt.Printf("updating checksum for %s\n", file)
	if checksum, err = CalcChecksum(file); err != nil {
		return "", err
	}
	CheckSums[relFile] = checksum
	writeCheckSums()

	return checksum, nil
}

// GetFiles will return a list of files that have been transfered
func GetFiles(path string) []string {
	updateCacheDir(path)
	readCheckSums()
	files := make([]string, 0, len(CheckSums))
	for file := range CheckSums {
		files = append(files, file)
	}
	return files
}

// CalcChecksum works out the checksum string
func CalcChecksum(file string) (string, error) {
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

// readCheckSums populates the hashcache from checksum file
func readCheckSums() {
	if _, err := os.Stat(CheckSumFilePath); os.IsNotExist(err) {
		log.Printf("File %q doesn't exist", CheckSumFilePath)
		return
	}
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
		log.Printf("read checksum line:%q", line)
		hashEntry := strings.Fields(scanner.Text())
		if len(hashEntry) != 2 {
			log.Printf("invalid cache entry %s\n", line)
		} else {
			key := hashEntry[1]
			value := hashEntry[0]
			log.Printf("adding cache entry key=%q => value=%q", key, value)
			CheckSums[key] = value
		}
	}
}

// Create the file contents from the checksum cache
func writeCheckSums() error {
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
	updateCacheDir(filepath.Dir(file))
	return filepath.Base(file)
}

func updateCacheDir(dirPath string) {
	CheckSumFilePath = dirPath +
		string(filepath.Separator) +
		CheckSumFileName
}
