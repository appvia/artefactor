package version

var (
	gitVersion   string
	gitCommit    string
	gitTreeState string
)

// Version represents version data.
type Version struct {
	Version      string
	Commit       string
	GitTreeState string
}

// Get returns the overall codebase version.
func Get() Version {
	return Version{
		Version:      gitVersion,
		Commit:       gitCommit,
		GitTreeState: gitTreeState,
	}
}
