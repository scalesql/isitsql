// Package logonce logs each message once.  This can be called multiple times
// in a loop and it will only write each unique message once.
// It only writes to the log file.
package logonce

import (
	"sync"

	"github.com/sirupsen/logrus"
)

var msgmap map[string]bool
var mux sync.RWMutex

func init() {
	msgmap = make(map[string]bool)
}

func Info(s string) {
	mux.Lock()
	defer mux.Unlock()
	_, ok := msgmap[s]
	if ok {
		return
	}
	logrus.Info(s)
}

func Warn(s string) {
	mux.Lock()
	defer mux.Unlock()
	_, ok := msgmap[s]
	if ok {
		return
	}
	logrus.Warn(s)
}

func Error(s string) {
	mux.Lock()
	defer mux.Unlock()
	_, ok := msgmap[s]
	if ok {
		return
	}
	logrus.Error(s)
}
