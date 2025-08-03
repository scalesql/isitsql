package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/kardianos/osext"
	"github.com/pkg/errors"
	"github.com/scalesql/isitsql/internal/crlf"
	"github.com/scalesql/isitsql/internal/failure"
	"github.com/sirupsen/logrus"
)

func configureFileLogging() {
	var wd string
	var err error
	logrus.SetFormatter(&logrus.TextFormatter{})

	wd, err = osext.ExecutableFolder()
	if err != nil {
		failure.WriteFile("logging", err.Error())
		logrus.Fatal(errors.Wrap(err, "osext.executablefolder"))
	}

	logdir := filepath.Join(wd, "log")
	if _, err = os.Stat(logdir); os.IsNotExist(err) {
		err = os.Mkdir(logdir, 0644)
		if err != nil {
			failure.WriteFile("logging", err.Error())
			logrus.Fatal(errors.Wrap(err, "os.mkdir"))
		}
	}

	if err != nil {
		failure.WriteFile("logging", err.Error())
		logrus.Fatal(errors.Wrap(err, "os.stat"))
	}

	logfile := fmt.Sprintf("isitsql_%s.log", time.Now().Format("20060102_150405"))
	fullfile := filepath.Join(wd, "log", logfile)

	file, err := os.Create(fullfile)
	if err != nil {
		failure.WriteFile("logging", err.Error())
		logrus.Fatal(errors.Wrap(err, "os.create"))
	}

	var fileWriter io.Writer
	// if we are in Windows, we will writer CRLF terminated log lines
	// to be notepad friendly
	if runtime.GOOS == "windows" {
		fileWriter = crlf.NewWriter(file)
	} else {
		fileWriter = file
	}
	multi := io.MultiWriter(fileWriter, os.Stdout)
	logrus.SetOutput(multi)
}

// WinLogln writes a log line
func WinLogln(a ...interface{}) {
	var str string
	for _, v := range a {
		s := fmt.Sprintf("%v", v)
		if len(s) > 0 && len(str) > 0 {
			str += " "
		}
		str += s
	}
	//log.Println(str, "\r")
	logrus.Info(str)
	GLOBAL_RINGLOG.Enqueue(str)
}

// WinLogf logs an error
func WinLogErr(err error) {
	logrus.Error(err.Error())
	GLOBAL_RINGLOG.Enqueue("ERROR: " + err.Error())
}

// WinLogf logs a formatted line
func WinLogf(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	WinLogln(msg)
}
