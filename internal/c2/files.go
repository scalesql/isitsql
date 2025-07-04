package c2

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ReadFile reads a file and fixes double-backslashes
func ReadFile(file string) ([]byte, error) {
	bb, err := os.ReadFile(file)
	if err != nil {
		return bb, errors.Wrap(err, "os.readfile")
	}

	bb = FixSlashes(bb)
	return bb, nil
}

func GetHCLFiles() (ConfigMaps, []string, error) {
	// get the names
	names, err := FindHCLFiles()
	if err != nil {
		return ConfigMaps{}, []string{}, errors.Wrap(err, "findhclfiles")
	}
	// get all files
	files := make([]ConnectionFile, 0)
	for _, f := range names {
		logrus.Tracef("c2: read: %s", f)
		bb, err := ReadFile(f)
		if err != nil {
			return ConfigMaps{}, []string{}, errors.Wrap(err, "readfile")
		}
		//log.Printf("%s: %d bytes\n", file, len(bb))
		//str := string(bb)
		//println(str)

		cf := ConnectionFile{}
		err = hclsimple.Decode(f, bb, nil, &cf)
		if err != nil {
			return ConfigMaps{}, []string{}, err
		}
		files = append(files, cf)
	}
	// process all the files to make the map
	fc, msgs := makeMap(names, files)
	return fc, msgs, nil
}

// FindHCLFiles returns a list of HCL files
func FindHCLFiles() ([]string, error) {
	path, err := Path()
	if err != nil {
		return []string{}, errors.Wrap(err, "path")
	}
	return find(path, ".hcl"), nil
}

func find(root, ext string) []string {
	var a []string
	filepath.WalkDir(root, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if filepath.Ext(d.Name()) == ext {
			a = append(a, s)
		}
		return nil
	})
	return a
}
