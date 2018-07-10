package hashcache

import (
	"testing"
)

type testSha256 struct {
	sha     string
	file    string
	cached  bool
	goodSha bool
}

var (
	shass = []testSha256{
		{"39f1b4c642f73cf660b1d5e0822a39f260fa9c67f24c896e334a7d85a7aa139a", "./test/test.txt", true, true},
		{"a7c9c23a0e050924abe57b0fb5fb0be67c6763c3aa9aab672b2ecc3086ffe924", "./test/testfile2", true, true},
		{"rubbish", "./test/testfile2", true, false},
		{"39f1b4c642f73cf660b1d5e0822a39f260fa9c67f24c896e334a7d85a7aa139a", "./test/bob", false, false},
	}
)

func TestIsCached(t *testing.T) {
	for _, sha := range shass {
		// new checksum struct and don't create checksum for item
		c, _ := NewFromExistingFile(sha.file, false)
		if c.IsCached(sha.file) != sha.cached {
			t.Errorf("Expecting %v but got %v for entry %q", sha.cached, !sha.cached, sha.file)
		}
	}
}

func TestIsCachedMatched(t *testing.T) {
	for _, sha := range shass {
		// new checksum struct and don't create checksum for item
		c, _ := NewFromExistingFile(sha.file, false)
		if c.IsCachedMatched(sha.file, sha.sha) != sha.goodSha {
			t.Errorf("Expecting %v but got %v for entry %q", sha.goodSha, !sha.goodSha, sha.file)
		}
	}
}
