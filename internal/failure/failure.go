// Package failure offers functions to support panics.
package failure

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"runtime/pprof"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var BuildGit string = "undefined"
var BuildDate string = "undefined"

// HandlePanic runs a recover and writes any panic to
// stdout and writes a text file with the details.
//
// GO routines should "defer failure.HandlePanic()" at startup.
func HandlePanic() {
	r := recover()
	if r != nil {
		stack := string(debug.Stack())
		msg := "===================================\nPANIC\n"
		msg += "-----------------------------------\n"
		msg += fmt.Sprintf("Build: %s\n", BuildGit)
		msg += fmt.Sprintf("Date:  %s\n", BuildDate)
		msg += "-----------------------------------\n"
		msg += fmt.Sprintf("%v\n", r)
		msg += "===================================\n"
		msg += fmt.Sprintf("\n===================================\nSTACK\n-----------------------------------\n%s", stack)
		msg += "\n===================================\n"
		fmt.Fprintln(os.Stderr, msg)
		WriteFile("panic", msg)
	}
}

// WriteFile generates a time stamped file in the EXE folder.
// It is primarily used to handle panics.
// It tries to write in the EXE folder but falls back to the current folder (which may be the same).
// prefix is used to build the file name: isitsql_prefix_ymd_hms.txt.
func WriteFile(prefix, body string) {
	if prefix == "" {
		prefix = "unknown"
	}
	ts := time.Now().Format("20060102_030405")
	syspaniclog := fmt.Sprintf("isitsql_%s_%s.txt", prefix, ts)

	ex, err := os.Executable()
	if err != nil {
		// if error isn't nil, just write where ever we can (system32 probably)
		_ = os.WriteFile(syspaniclog, []byte(body), 0644)
		return
	}
	// write to the EXE directory
	apppaniclog := filepath.Join(filepath.Dir(ex), syspaniclog)
	_ = os.WriteFile(apppaniclog, []byte(body), 0644)
}

// WritePProf writes various PPROF profiles.
func WritePProf() {
	ts := time.Now().Format("20060102_030405")
	var heapFile string
	format := 0
	heapFile = fmt.Sprintf("isitsql_heap_%s.pprof.pb.gz", ts)

	ex, err := os.Executable()
	if err != nil {
		// if error isn't nil, just write where ever we can (system32 probably)
		//_ = os.WriteFile(syspaniclog, []byte(body), 0644)
		f, err := os.Create(heapFile)
		if err != nil {
			logrus.Error(errors.Wrap(err, "writepprof: os.create"))
			return
		}
		defer f.Close()
		logrus.Infof("pprof: %s", heapFile)
		err = pprof.Lookup("heap").WriteTo(f, format)
		if err != nil {
			logrus.Error(errors.Wrap(err, "pprof.writeto"))
		}
		return
	}
	// write to the EXE directory
	heapFile = filepath.Join(filepath.Dir(ex), heapFile)
	f, err := os.Create(heapFile)
	if err != nil {
		logrus.Error(errors.Wrap(err, "writepprof: os.create"))
		return
	}
	defer f.Close()
	logrus.Infof("pprof: %s", heapFile)
	err = pprof.Lookup("heap").WriteTo(f, format)
	if err != nil {
		logrus.Error(errors.Wrap(err, "pprof.writeto"))
	}
}
