package bucket

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

// PurgeFiles purges files based on the time stamp in the file name.
// It requires \\EXE dir\dir\prefix_yyyymmdd_hhmmss.ext format
// dir is the subdirectory off the EXE directory
func PurgeFiles(dir, prefix, ext string, retain time.Duration) error {
	fs := afero.NewOsFs()
	clk := clock.New()

	// get the app directory
	exe, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "os.getwd")

	}
	wd := filepath.Dir(exe)
	path := filepath.Join(wd, dir)
	return purgeFilesPath(fs, clk, path, prefix, ext, retain)
}

// purgeServerFiles cleans out the old per server files
// func (bw *BucketWriter) purgeServerFiles(retain time.Duration) error {
// 	// if path == "" {
// 	// 	path = "."
// 	// }
// 	// if prefix == "" {
// 	// 	return errors.New("prefix can't be empty")
// 	// }
// 	// if ext == "" {
// 	// 	return errors.New("extension can't be empty")
// 	// }

// 	// does the path exist
// 	exists, err := afero.Exists(bw.fs, bw.path)
// 	if err != nil {
// 		return errors.Wrap(err, "afero.exists")
// 	}
// 	if !exists {
// 		return fmt.Errorf("path not found: %s", bw.path)
// 	}

// 	pattern := filepath.Join(bw.path, fmt.Sprintf("bw.%s.*.*.ndjson", bw.prefix))
// 	files, err := afero.Glob(bw.fs, pattern)
// 	if err != nil {
// 		return errors.Wrap(err, "afero.glob")
// 	}
// 	sort.Strings(files)

// 	// regexStr := fmt.Sprintf(`\\%s_(?P<ts>\d{8}_\d{6})\.%s$`, prefix, ext)
// 	// rexp, err := regexp.Compile(regexStr)
// 	// if err != nil {
// 	// 	return errors.Wrap(err, "regex.compile")
// 	// }
// 	purgeThreshold := bw.clock.Now().Add(retain * -1)

// 	for _, name := range files {
// 		// matches := rexp.FindStringSubmatch(name)
// 		// if len(matches) < 2 {
// 		// 	continue
// 		// }
// 		// str := matches[1]
// 		// ts, err := time.Parse("20060102_150405", str)
// 		// if err != nil {
// 		// 	return errors.Wrap(err, "time.parse")
// 		// }
// 		fi, err := bw.fs.Stat(name)
// 		if err != nil {
// 			return errors.Wrap(err, "bw.fs.stat")
// 		}
// 		ts := fi.ModTime()
// 		if ts.Before(purgeThreshold) {
// 			logrus.Tracef("purging: %s", name)
// 			err = bw.fs.Remove(name)
// 			if err != nil {
// 				// just log any file delete errors and keep going
// 				logrus.Error(errors.Wrap(err, "bw.fs.remove"))
// 			}
// 		}
// 	}
// 	return nil
// }

func purgeFilesPath(fs afero.Fs, clk clock.Clock, path, prefix, ext string, retain time.Duration) error {
	if path == "" {
		path = "."
	}
	if prefix == "" {
		return errors.New("prefix can't be empty")
	}
	if ext == "" {
		return errors.New("extension can't be empty")
	}

	// does the path exist
	exists, err := afero.Exists(fs, path)
	if err != nil {
		return errors.Wrap(err, "afero.exists")
	}
	if !exists {
		return fmt.Errorf("path not found: %s", path)
	}

	pattern := filepath.Join(path, fmt.Sprintf("%s_*.%s", prefix, ext))
	files, err := afero.Glob(fs, pattern)
	if err != nil {
		return errors.Wrap(err, "afero.glob")
	}
	sort.Strings(files)

	regexStr := fmt.Sprintf(`\\%s_(?P<ts>\d{8}_\d{6})\.%s$`, prefix, ext)
	rexp, err := regexp.Compile(regexStr)
	if err != nil {
		return errors.Wrap(err, "regex.compile")
	}
	purgeThreshold := clk.Now().Add(retain * -1)

	for _, name := range files {
		matches := rexp.FindStringSubmatch(name)
		if len(matches) < 2 {
			continue
		}
		str := matches[1]
		ts, err := time.Parse("20060102_150405", str)
		if err != nil {
			return errors.Wrap(err, "time.parse")
		}
		if ts.Before(purgeThreshold) {
			logrus.Tracef("purgefile: %s", name)
			err = fs.Remove(name)
			if err != nil {
				// just log any file delete errors and keep going
				logrus.Error(errors.Wrap(err, "purgefile"))
			}
		}
	}
	return nil
}
