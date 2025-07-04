package waitmap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/scalesql/isitsql/internal/failure"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type WaitLine struct {
	MapKey  string `json:"map_key"`
	Payload Waits  `json:"payload"`
}

// WriteWaits appends a Waits struct to a file named for
// the server key.  It rolls over files after 10 minutes.
func WriteWaitFile(key string, ww Waits) error {
	wl := WaitLine{
		MapKey:  key,
		Payload: ww,
	}
	start := time.Now()
	bb, err := json.Marshal(wl)
	if err != nil {
		return errors.Wrap(err, "se.marshal")
	}
	exe, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "os.executable")
	}
	waitFileName := filepath.Join(filepath.Dir(exe), "cache", fmt.Sprintf("store.waits.%s.%s.ndjson", key, time.Now().Round(10*time.Minute).Format("20060102_150400")))
	sf, err := os.OpenFile(waitFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errors.Wrap(err, "os.openfile")
	}
	defer sf.Close()
	_, err = sf.Write(bb)
	if err != nil {
		return errors.Wrap(err, "sf.write")
	}
	_, err = sf.WriteString("\n")
	if err != nil {
		return errors.Wrap(err, "sf.writestring")
	}
	if time.Since(start) > time.Duration(100*time.Millisecond) {
		logrus.Debugf("waitmap: write: [%s] bytes=%d (%s)", key, len(bb), time.Since(start))
	}

	return nil
}

// ReadWaitFiles reads the wait files for a key and returns []Waits
func ReadWaitFiles(key string) ([]Waits, error) {
	results := make([]Waits, 0, 240)
	defer failure.HandlePanic()

	// get the files
	start := time.Now()
	// fmt.Sprintf("%s/bw.%s.%s.%s.ndjson", bw.path, bw.prefix, mapkey, bw.clock.Now().UTC().Round(10*time.Minute).Format("20060102_1504"))
	exe, err := os.Executable()
	if err != nil {
		return results, errors.Wrap(err, "os.executable")
	}
	pattern := filepath.Join(filepath.Dir(exe), "cache", fmt.Sprintf("store.waits.%s.*.ndjson", key))
	files, err := filepath.Glob(pattern)
	if err != nil {
		return results, errors.Wrap(err, "filepath.glob")
	}
	sort.Strings(files)

	var totalLines int
	for _, fileName := range files {
		// read the entire file into memory
		bb, err := os.ReadFile(fileName)
		if err != nil {
			return results, errors.Wrap(err, "os.readfile")
		}

		// read each line of the NDJSON
		// the goal is to return an array of waitmap.Waits
		for _, v := range bytes.Split(bb, []byte{'\n'}) {
			line := v
			// it seems to be returning a zero length line at the end
			if len(line) == 0 {
				continue
			}

			totalLines++
			wl := WaitLine{}
			err = json.Unmarshal(line, &wl)
			if err != nil {
				return results, errors.Wrap(err, "json.unmarshal")
			}
			if wl.MapKey == key {
				results = append(results, wl.Payload)
			}
		}
	}
	if time.Since(start) > time.Duration(100*time.Millisecond) {
		logrus.Debugf("waitmap: read: [%s] files=%d  lines=%d  kept=%d  (%s)", key, len(files), totalLines, len(results), time.Since(start))
	}
	return results, nil
}

// PurgeWaitFiles purges wait files more than 85 minutes old
func PurgeWaitFiles(key string) error {
	start := time.Now()
	exe, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "os.executable")
	}
	path := filepath.Join(filepath.Dir(exe), "cache")
	//logrus.Debugf("purge: %s", path)
	// does the path exist
	_, err = os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Wrap(err, "os.stat")
	}

	pattern := filepath.Join(path, fmt.Sprintf("store.waits.%s.*.ndjson", key))
	//logrus.Debugf("purge: pattern: %s", pattern)
	// cache/store.waits.%s.%s.ndjson
	files, err := filepath.Glob(pattern)
	if err != nil {
		return errors.Wrap(err, "filepath.glob")
	}
	//sort.Strings(files)
	purgeThreshold := time.Now().Add(-85 * time.Minute)
	purged := 0
	for _, name := range files {
		fi, err := os.Stat(name)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return errors.Wrap(err, "os.stat")
		}
		ts := fi.ModTime()
		if ts.Before(purgeThreshold) {
			purged++
			logrus.Tracef("purging: %s", name)
			err = os.Remove(name)
			if err != nil {
				// just log any file delete errors and keep going
				logrus.Error(errors.Wrap(err, "os.remove"))
			}
		}
	}
	if time.Since(start) > time.Duration(100*time.Millisecond) {
		logrus.Debugf("waitmap: purge: [%s] files=%d purged=%d (%s)", key, len(files), purged, time.Since(start))
	}
	return nil
}
