package bucket

import (
	"encoding/json"
)

var WaitsPrefix string = "waits"

// ServerEvent is a wrapper for the event passed in so we include the MapKey
type ServerEvent struct {
	MapKey  string          `json:"map_key"`
	Payload json.RawMessage `json:"payload"`
}

/*
Parameters
==========

appDir -- if "", then afero.NewOsFs()
dir -- cache, log, etc.
prefix - waits, isitsql (for logs), etc.
-- the implied timestamp yyyymmdd_hhmmss
ext -- log, ndjson
retain -- time.Duration

fs afero.fs
clock

bucket.PurgeTimeStampedFiles() -- sets FS and clock and calls purgeFiles
bucket.purgeFiles() -- accepts above parameters and afero.FS and a clock

*/

// func PurgeFilesTimeStamp(fs afero.Fs, dir string, pattern string, retain time.Duration) error {
// 	if fs == nil {
// 		fs = afero.NewOsFs()
// 	}
// 	// get EXE dir
// 	wd, err := osext.ExecutableFolder()
// 	if err != nil {
// 		return errors.Wrap(err, "osext.executablefolder")
// 	}
// 	ptrn := filepath.Join(wd, dir, pattern)
// 	files, err := afero.Glob(fs, ptrn)
// 	if err != nil {
// 		return errors.Wrap(err, "afero.glob")
// 	}
// 	re1, err := regexp.Compile(`(?P<ts>\d{8}_\d{6})`)
// 	if err != nil {
// 		return errors.Wrap(err, "regexp.compile")
// 	}
// 	// go back retain plus two rollovers
// 	purgeThreshold := time.Now().Add(-1 * retain)

// 	// go through each and delete old ones
// 	for _, name := range files {
// 		matches := re1.FindStringSubmatch(name)
// 		if len(matches) < 2 { // not enough matches
// 			continue
// 		}
// 		str := matches[1]
// 		ts, err := time.Parse("20060102_150405", str)
// 		if err != nil {
// 			return errors.Wrap(err, "time.parse")
// 		}
// 		if ts.Before(purgeThreshold) {
// 			err = fs.Remove(name)
// 			if err != nil {
// 				return errors.Wrap(err, "fs.remove")
// 			}
// 		}
// 	}
// 	return nil
// }
