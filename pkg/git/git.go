package git

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/appvia/artefactor/pkg/hashcache"
	"github.com/appvia/artefactor/pkg/tar"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

const (
	GitFileExt     string = ".git.tar"
	GitFileHomeExt string = ".git.home.tar"
)

// Archive will create a git archive from a local path
func Archive(repoPath string, saveDir string) error {
	// TODO: add if local path ! exist try and clone first...

	fmt.Printf("Archiving git repo %s\n", repoPath)

	// Open the current git repo path
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return err
	}

	// Check if clean
	w, err := r.Worktree()
	if err != nil {
		return err
	}
	status, err := w.Status()
	if err != nil {
		return err
	}
	if !status.IsClean() {
		// Not a clean repo, deal with it...
		return errors.New(fmt.Sprintf("Not backing up git directory %v- not clean:\n%s", repoPath, status))
	}

	repoName := getRepoName(r, filepath.Dir(repoPath))
	tarFileName := ""
	// The repo should be named appropriatly so we can use it as a home on restore
	if isHome(repoPath) {
		tarFileName = fmt.Sprintf("%s/%s%s", saveDir, repoName, GitFileHomeExt)
	} else {
		tarFileName = fmt.Sprintf("%s/%s%s", saveDir, repoName, GitFileExt)
	}
	// Now archive this repo...
	// Get the HEAD ref
	ref, err := r.Head()
	if err != nil {
		return err
	}
	// ... retrieving the commit object
	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return err
	}
	// ... retrieve the tree from the commit
	tree, err := commit.Tree()
	if err != nil {
		return err
	}

	// get all the files that need archiving from the repo meta-data
	var archiveFiles []string
	tree.Files().ForEach(func(f *object.File) error {
		archiveFiles = append(archiveFiles, f.Name)
		return nil
	})
	// now add the meta-data files themselves (for a functioning git repo with no
	// extra files from .gitignore etc.)
	gitMetaFolder := filepath.Join(repoPath, ".git")
	filepath.Walk(
		gitMetaFolder,
		func(path string, fi os.FileInfo, err error) error {

			if err != nil {
				fmt.Printf("access denied accessing a path %q: %v\n", path, err)
				return err
			}
			archiveFiles = append(archiveFiles, path)
			return nil
		})

	// Now add the complete set of files to archive:
	if err := tar.Create(tarFileName, archiveFiles); err != nil {
		return err
	}

	// Lastly update the checksums
	if hashcache.UpdateCache(tarFileName); err != nil {
		return err
	}
	return nil
}

// IsClean will report is a repo is clean given a path
func IsClean(repoPath string) (bool, error) {
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return false, err
	}

	// Check if clean
	w, err := r.Worktree()
	if err != nil {
		return false, err
	}
	status, err := w.Status()
	if err != nil {
		return false, err
	}
	return status.IsClean(), nil
}

// Restore will extract a git repository to the correct path under the dst
// directory
func Restore(gitRepoFile string, dst string, repoName string) error {
	if err := os.RemoveAll(filepath.Join(dst, repoName, ".git")); err != nil {
		return err
	}
	// first remove any git meta-data:
	if err := tar.Extract(gitRepoFile, dst); err != nil {
		return err
	}
	return nil
}

// GetHomeRepo will return a 'home' repo
func GetHomeRepo(path string) (string, error) {
	tars, err := filepath.Glob(path + string(filepath.Separator) + "*" + GitFileHomeExt)
	if err != nil {
		return "", err
	}
	switch homes := len(tars); homes {
	case 0:
		return "", nil
	case 1:
		return tars[0], nil
	default:
		return "", fmt.Errorf("Multiple home git repos found in %q", path)
	}
}

// GetOtherRepos will list other git repos saved
func GetOtherRepos(path string) ([]string, error) {
	gitRepos, err := filepath.Glob(path + string(filepath.Separator) + "*" + GitFileExt)
	if err != nil {
		return nil, err
	}
	return gitRepos, nil
}

// isHome will work out if we are asked to archive the current dir
func isHome(path string) bool {
	pwd, _ := os.Getwd()
	fullPath, _ := filepath.Abs(path)
	if pwd == fullPath {
		return true
	}
	return false
}

// getRepoName return the Repository base name
func getRepoName(r *git.Repository, dir string) string {
	cfg, _ := r.Config()
	if cfg == nil {
		return dir
	}
	if len(cfg.Remotes) < 1 {
		return dir
	}
	basename := strings.Replace(
		filepath.Base(cfg.Remotes["origin"].URLs[0]),
		".git",
		"",
		1)
	return basename
}
