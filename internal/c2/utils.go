package c2

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// FixSlashes correctly escapes back slashes in byte arrays
// Designed to support server\instance and server\\instance
// for \ -> \\
// for \\ -> \\\\ -> \\
func FixSlashes(bb []byte) []byte {
	bb = bytes.ReplaceAll(bb, []byte(`\`), []byte(`\\`))
	bb = bytes.ReplaceAll(bb, []byte(`\\\\`), []byte(`\\`))
	return bb
}

// Path returns the path of /servers from the EXE folder
func Path() (string, error) {
	exeFile, err := os.Executable()
	if err != nil {
		return "", errors.Wrap(err, "os.executable")
	}
	exePath := filepath.Dir(exeFile)
	wd := filepath.Join(exePath, "servers")
	return wd, nil
}

func ptr[T any](v T) *T {
	return &v
}
