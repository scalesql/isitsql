package build

import (
	"time"

	"github.com/Masterminds/semver/v3"
)

var builtFlag = "2020-04-01T00:00:00-00:00"
var commitFlag = "dev"
var versionFlag = "dev"

// Commit returns the GIT Commmit
func Commit() string {
	return commitFlag
}

// Version returns the version from the build
func Version() string {
	return versionFlag
}

// Semver returns the version as a semantic version
func Semver() (*semver.Version, error) {
	return semver.NewVersion(versionFlag)
}

// Build returns the time the application was built
func Built() time.Time {
	// return an empty t (0001-01-02) if we get an error
	// With an error, Built is after Expires
	t, err := time.Parse("2006-01-02T15:04:05-07:00", builtFlag)
	if err != nil {
		return t.Add(48 * time.Hour)
	}
	return t
}
