package bucket

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// BucketWriter writes cached ServerEvent entries
type BucketWriter struct {
	fileStart    time.Time
	fs           afero.Fs
	clock        clock.Clock
	file         *afero.File
	prefix       string
	path         string
	fileDuration time.Duration
	retain       time.Duration
	sync.RWMutex
}

// Start a BucketWriter
func (bw *BucketWriter) Start(path, prefix string) error {
	bw.Lock()
	defer bw.Unlock()
	if bw.fs == nil {
		bw.fs = afero.NewOsFs()
	}
	if bw.clock == nil {
		bw.clock = clock.New()
	}
	bw.prefix = prefix
	bw.path = path
	bw.fileStart = time.Date(1960, 1, 1, 0, 0, 0, 0, time.Local)

	// default rollover and retention
	// retention is retain + 2 * fileDuration
	bw.fileDuration = 10 * time.Minute
	bw.retain = 60 * time.Minute

	// open a new file
	err := bw.open()
	if err != nil {
		return errors.Wrap(err, "open")
	}
	return nil
}

// Write an object into a BucketWriter
func (bw *BucketWriter) Write(mapkey string, v interface{}) error {
	var err error
	valueBytes, err := json.Marshal(v)
	if err != nil {
		return errors.Wrap(err, "v.marshall")
	}
	se := ServerEvent{MapKey: mapkey, Payload: valueBytes}
	bb, err := json.Marshal(se)
	if err != nil {
		return errors.Wrap(err, "se.marshal")
	}

	bw.Lock()
	defer bw.Unlock()
	if bw.clock == nil {
		bw.clock = clock.New()
	}

	// rollover if needed
	if bw.clock.Now().After(bw.fileStart.Add(bw.fileDuration)) || bw.file == nil {
		err = bw.rollover()
		if err != nil {
			return errors.Wrap(err, "bw.rolloverfile")
		}
	}
	if bw.file == nil {
		return nil
	}
	f := *bw.file
	_, err = f.Write(bb)
	if err != nil {
		return errors.Wrap(err, "write")
	}
	_, err = f.WriteString("\n")
	if err != nil {
		return errors.Wrap(err, "writestring")
	}

	// write to the server specific file ==============================================
	// start := time.Now()

	// // bw.w2.key.timestamp.ndjson
	// // bw.waits.key.timestamp.ndjson

	// sFileName := fmt.Sprintf("%s/bw.%s.%s.%s.ndjson", bw.path, bw.prefix, mapkey, bw.clock.Now().UTC().Round(10*time.Minute).Format("20060102_150400"))
	// sf, err := bw.fs.OpenFile(sFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	// if err != nil {
	// 	return errors.Wrap(err, "bw.fs.openfile")
	// }
	// defer sf.Close()
	// n, err := sf.Write(bb)
	// if err != nil {
	// 	return errors.Wrap(err, "sf.write")
	// }
	// _, err = sf.WriteString("\n")
	// if err != nil {
	// 	return errors.Wrap(err, "sf.writestring")
	// }
	// dur := time.Since(start)
	// bw.metrics.writes++
	// bw.metrics.writeDuration += dur
	// bw.metrics.writeBytes += n + 2
	// if dur > bw.metrics.writeSlowest {
	// 	bw.metrics.writeSlowest = dur
	// }
	// //fmt.Printf("write: %s: %s (%d bytes)\n", mapkey, dur, n+2)
	// // if bw.metrics.writes > 0 {
	// // 	fmt.Printf("bw.total (%-5s): %s: dur_µs=%-10d total_µs=%-10d avg_µs=%-6d writes=%-4d (%4d kb) slowest=%-12s\n", bw.prefix, mapkey, dur.Microseconds(), bw.metrics.writeDuration.Microseconds(), (bw.metrics.writeDuration / time.Duration(time.Duration(bw.metrics.writes))).Microseconds(), bw.metrics.writes, bw.metrics.writeBytes/1024, bw.metrics.writeSlowest)
	// // }

	// // purge files more than X minutes old
	// if bw.lastPurge.Add(10 * time.Minute).Before(bw.clock.Now()) {
	// 	//logrus.Debugf("purging: key=%s (%s)  last=%s", mapkey, bw.prefix, bw.lastPurge)
	// 	err = bw.purgeServerFiles(mapkey, 85*time.Minute)
	// 	if err != nil {
	// 		return errors.Wrap(err, "purgeserverfiles")
	// 	}
	// 	bw.lastPurge = bw.clock.Now()
	// }

	return nil
}

// close the bucket file
func (bw *BucketWriter) close() error {
	if bw.file == nil {
		return nil
	}
	f := *bw.file
	err := f.Close()
	bw.file = nil
	return err
}

// open a bucket file
func (bw *BucketWriter) open() error {
	var err error
	// create the path if needed
	if bw.path != "" && bw.path != "." {
		err = bw.fs.MkdirAll(bw.path, 0660)
		if err != nil {
			return errors.Wrap(err, "mkdirall")
		}
	}
	name := fmt.Sprintf("%s/%s_%s.ndjson", bw.path, bw.prefix, bw.clock.Now().UTC().Format("20060102_150405"))
	f, err := bw.fs.OpenFile(name, os.O_CREATE|os.O_WRONLY, 0660)
	if err != nil {
		return errors.Wrap(err, "openfile")
	}
	bw.file = &f
	bw.fileStart = bw.clock.Now()
	return nil
}

// rollover closes the cache file and opens another
func (bw *BucketWriter) rollover() error {
	var err error

	err = bw.close()
	if err != nil {
		return errors.Wrap(err, "close")
	}

	err = bw.open()
	if err != nil {
		return errors.Wrap(err, "open")
	}

	err = purgeFilesPath(bw.fs, clock.New(), bw.path, bw.prefix, "ndjson", 85*time.Minute)
	if err != nil {
		return errors.Wrap(err, "purgeFilesPath")
	}
	return nil
}
