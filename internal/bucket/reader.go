package bucket

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/scalesql/isitsql/internal/failure"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

type BucketReader struct {
	fs      afero.Fs
	Err     error
	Results chan string
	prefix  string
	path    string
}

/*

1. create the reader
	a. sort the files
2. start a GO routine passing a channel
	a. for each file
		a. for each row (bufio.ReadString('\n'))
			a. if len(str) > 0, send the string on the channel
	b. close the channel

After launching the GO routine {
	for str := range the_channel {
		add the wait
	} // closing the channel will exit the for loop
}

we are done!

*/

// NewReader is used to initialize a BucketReader
func NewReader(prefix, path string) (BucketReader, error) {
	br := BucketReader{
		prefix:  prefix,
		path:    path,
		fs:      afero.NewOsFs(),
		Results: make(chan string),
	}
	if prefix == "" {
		return br, errors.New("prefix is required")
	}

	// get the directory
	_, err := br.fs.Stat(path)
	if err != nil {
		return br, errors.Wrap(err, "fs.stat")
	}

	return br, nil
}

// StartReader goes through the files and returns the rows
// to a channel.  It should be called in a GO routine
func (br *BucketReader) StartReader() {
	defer failure.HandlePanic()
	defer func() {
		logrus.Debug("startreader: closing channel")
		close(br.Results)
	}()

	// purge any old files
	err := purgeFilesPath(br.fs, clock.New(), br.path, br.prefix, "ndjson", 85*time.Minute)
	if err != nil {
		logrus.Error(errors.Wrap(err, "purgefilespath"))
	}

	// list the files
	pattern := filepath.Join(br.path, fmt.Sprintf("%s_*.ndjson", br.prefix))
	files, err := afero.Glob(br.fs, pattern)
	if err != nil {
		br.Err = errors.Wrap(err, "filepath.glob")
		logrus.Error(errors.Wrap(err, "filepath.glob"))
		return
	}
	sort.Strings(files)

	for _, file := range files {
		logrus.Tracef("startreader: reader: file: %s", file)
		// open
		fi, err := br.fs.Open(file)
		if err != nil {
			logrus.Error(errors.Wrap(err, "startreader: br.fs.open"))
			br.Err = err
			return
		}
		rdr := bufio.NewReader(fi)
		var str string

		// read the file and send
		for err == nil {
			str, err = rdr.ReadString('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				logrus.Error(errors.Wrap(err, "startreader: rdr.readstring"))
				br.Err = err
				return
			}
			br.Results <- str
			logrus.Tracef("sent bytes: %d", len(str))
		}

		// close
		logrus.Tracef("startreader: reader: closing: %s", file)
		err = fi.Close()
		if err != nil {
			logrus.Error(errors.Wrap(err, "startreader: fi.close"))
			br.Err = err
			return
		}
	}
}
