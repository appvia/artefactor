package version

var (
	gitSha     string
	gitVersion string
)

// Version represents version data.
type Version struct {
	Version string
	Sha     string
}

// Get returns the overall codebase version.
func Get() Version {
	return Version{
		Version: gitVersion,
		Sha:     gitSha,
	}
}
