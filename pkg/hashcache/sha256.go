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
	DefaultCheckSumFileName = "checksum.txt"
)

// CheckSumItem represents a single file as present in cache (checksum file)
type CheckSumItem struct {
	// CheckSum is the sha256sum initialy calculated
	CheckSum string
	// FilePath is the full path to a file given a start directory
	FilePath string
	// FileName is a relative file name only
	FileName string
}

// CheckSumCache defines data for a hash cache
type CheckSumCache struct {
	CheckSumsByFilePath map[string]CheckSumItem
	Dir                 string
	CheckSumFile        string
	AddedItems          []CheckSumItem
}

// NewFromExistingFile creates an existing cache (relative from the file name)
// optionally will create the cache item...
func NewFromExistingFile(
	file string,
	create bool) (c *CheckSumCache, err error) {

	dir := filepath.Dir(file)
	// Don't error if we're creating a new checksum cache
	c, err = NewFromCheckSumsFile(
		filepath.Join(dir, DefaultCheckSumFileName), !create)
	if err != nil {
		return c, err
	}
	if create {
		// Add the item
		_, err := c.Update(file)
		if err != nil {
			return c, fmt.Errorf(
				"problem adding %s to checksum file %s",
				file,
				c.CheckSumFile)
		}
	}
	return c, err
}

// NewCacheFromDir instanciates a cache using the default cache file name given
// a directory. Optionally error if checksum file is missing
func NewFromDir(dir string, errIfMissing bool) (c *CheckSumCache, err error) {
	c, err = NewFromCheckSumsFile(
		filepath.Join(dir, DefaultCheckSumFileName), errIfMissing)
	return c, err
}

// NewCacheFromCheckSumsFile creates a cache interface from a specific checksum file
func NewFromCheckSumsFile(file string, errIfMissing bool) (c *CheckSumCache, err error) {
	c = &CheckSumCache{
		CheckSumFile:        file,
		Dir:                 filepath.Dir(file),
		CheckSumsByFilePath: make(map[string]CheckSumItem),
	}
	c.readCheckSumsIfPresent()
	if errIfMissing {
		if _, err := os.Stat(c.CheckSumFile); os.IsNotExist(err) {
			return c, fmt.Errorf("no checksum file %q", c.CheckSumFile)
		}
	}
	return c, nil
}

// Clean will remove any old entries (not added using Update method)
func (c *CheckSumCache) Clean() error {
	removeItems := []string{}
	// Check all the cache file entries
	for key, item := range c.CheckSumsByFilePath {
		keep := false
		// Find the one's we've just added
		for _, addedItem := range c.AddedItems {
			if item.FileName == addedItem.FileName {
				keep = true
				break
			}
		}
		if !keep {
			// Keep track of items not added
			removeItems = append(removeItems, key)
		}
	}
	// Now clean up the old items
	for _, item := range removeItems {
		delete(c.CheckSumsByFilePath, item)
	}
	// Lastly save the checksum file
	err := c.writeCheckSums()
	return err
}

// IsCachedMatched will verify if a file is in Cache AND matching expected sha256
func (c *CheckSumCache) IsCachedMatched(file string, sha256 string) bool {
	file = filepath.Clean(file)
	// Get relative path from directory if set
	if c.IsCached(file) {
		if c.CheckSumsByFilePath[file].CheckSum == sha256 {
			return true
		}
	}
	return false
}

// IsCached will check if a file is present on disk and in the checksum file
func (c *CheckSumCache) IsCached(file string) bool {
	file = filepath.Clean(file)
	// If the file doesn't exist...
	if _, err := os.Stat(file); os.IsNotExist(err) {
		log.Printf("File %q doesn't exist", file)
		return false
	}
	// Make sure we're up to date...
	c.readCheckSumsIfPresent()
	if _, ok := c.CheckSumsByFilePath[filepath.Clean(file)]; ok {
		log.Printf("Cache hit for %q", file)
		return true
	} else {
		log.Printf("Cache MISS for %q", file)
	}
	return false
}

// Update will update (or add) a file to the cache (checksum file)
func (c *CheckSumCache) Update(file string) (checksum string, err error) {
	file = filepath.Clean(file)
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return "", fmt.Errorf("file %q doesn't exist", file)
	}
	// Ensure we are up to date from disk
	c.readCheckSumsIfPresent()
	fmt.Printf("updating checksum for %s\n", file)
	if checksum, err = CalcChecksum(file); err != nil {
		return "", err
	}
	// Create a new item
	item := CheckSumItem{
		CheckSum: checksum,
		FileName: filepath.Base(file),
		FilePath: file,
	}
	// Replace / create the entry
	c.CheckSumsByFilePath[file] = item
	// Keep a track in memory of items added
	c.AddedItems = append(c.AddedItems, item)
	c.writeCheckSums()
	return checksum, nil
}

// Keep will mark a file (and checksum) so it won't be cleaned with .Clean
func (c *CheckSumCache) Keep(file string) {
	file = filepath.Clean(file)
	if _, ok := c.CheckSumsByFilePath[file]; ok {
		item := c.CheckSumsByFilePath[file]
		c.AddedItems = append(c.AddedItems, item)
	}
}

// readCheckSumsIfPresent populates the hashcache from checksum file (if it exists)
func (c *CheckSumCache) readCheckSumsIfPresent() {
	// Re-init in memory checksums
	c.CheckSumsByFilePath = make(map[string]CheckSumItem)
	if _, err := os.Stat(c.CheckSumFile); err != nil {
		// No checksums created... yet...
		fullpath, _ := filepath.Abs(c.CheckSumFile)
		log.Printf("no checksum file found at:%s (%s)", fullpath, err)
		return
	}
	csf, err := os.Open(c.CheckSumFile)
	if err != nil {
		log.Fatal(err)
	}
	defer csf.Close()
	// open checksum file
	scanner := bufio.NewScanner(csf)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("read checksum line:%q", line)
		hashEntry := strings.Fields(scanner.Text())
		if len(hashEntry) != 2 {
			log.Printf("invalid cache entry %s\n", line)
		} else {
			fileName := hashEntry[1]
			item := CheckSumItem{
				FilePath: filepath.Join(c.Dir, fileName),
				FileName: fileName,
				CheckSum: hashEntry[0],
			}
			log.Printf("adding cache entry key=%q => checksum=%q", item.FilePath, item.CheckSum)
			c.CheckSumsByFilePath[item.FilePath] = item
		}
	}
}

// writeCheckSums over write the file contents from the checksum cache
func (c *CheckSumCache) writeCheckSums() error {
	contents := ""
	for _, item := range c.CheckSumsByFilePath {
		line := fmt.Sprintf("%s  %s\n", item.CheckSum, item.FileName)
		contents = contents + line
	}
	// Save the file
	err := ioutil.WriteFile(c.CheckSumFile, []byte(contents), 0644)
	if err != nil {
		return err
	}
	return nil
}

// GetCachedChecksum will return previously calculated checksum
func GetCachedChecksum(file string) (string, error) {
	file = filepath.Clean(file)
	// Create a new CheckSumCache and error if the checksum file doesn't exist
	c, err := NewFromExistingFile(file, true)
	if err != nil {
		return "", err
	}
	if c.IsCached(file) {
		item, present := c.CheckSumsByFilePath[file]
		if present {
			return item.CheckSum, nil
		}
	}
	return "", fmt.Errorf("no checksum exists for file entry %s", file)
}

// GetFiles will return a list of files from cache
func GetFiles(path string) []string {
	c, err := NewFromDir(path, true)
	if err != nil {
		log.Printf("error opening cache:%s", err)
	}
	files := make([]string, 0, len(c.CheckSumsByFilePath))
	for _, item := range c.CheckSumsByFilePath {
		files = append(files, item.FilePath)
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
