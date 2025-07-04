package files

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

var fs = afero.NewOsFs()

// GetFirst returns the first file found.  It expects "config/file1.txt" format.
// It returns the fully qualified file and any error
// If the file doesn't exist, it returns os.ErrNotExist
func GetFirst(files ...string) (string, error) {
	var err error
	ex, err := os.Executable()
	if err != nil {
		return "", errors.Wrap(err, "os.executable")
	}
	wd := filepath.Dir(ex)
	return first(wd, files...)

}

func first(wd string, files ...string) (string, error) {
	if wd == "" {
		wd = "."
	}
	for _, f := range files {
		fp := filepath.Join(wd, f)
		_, err := fs.Stat(fp)
		if err == nil {
			return fp, nil
		}
	}
	return "", os.ErrNotExist
}
