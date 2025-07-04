// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows
// +build windows

package app

import (
	"fmt"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"

	"github.com/scalesql/isitsql/internal/failure"
	"github.com/kardianos/osext"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var elog debug.Log

type myservice struct{}

func serviceLauncher() {
	defer failure.HandlePanic()
	SetConfigMode()

	err := SetupWrapper()
	if err != nil {
		WinLogln(errors.Wrap(err, "setupwrapper"))
		logrus.Error(err.Error())
		elog.Error(1, err.Error())
	}

	go launchBatchUpdates()
	go launchWebServer()
	go launchMemoryLogger()
	go launchPProfLogger()
	// go launchGCLogger()
}

func (m *myservice) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	// defer func() {
	// 	err := recover()
	// 	if err != nil {
	// 		WinLogln(err)
	// 		logrus.Error(err.Error())
	// 		elog.Error(1, err.Error())
	// 	}
	// }()
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}
	//fasttick := time.Tick(1500 * time.Millisecond)
	//slowtick := time.Tick(5 * time.Second)
	//tick := fasttick
	//fmt.Println("In execute....")
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	err := setup()
	if err != nil {
		WinLogln("setup failed")
		logrus.Error(err.Error())
		elog.Error(1, err.Error())
		return
	}

	go serviceLauncher()

loop:
	//lint:ignore S1000 copied from GO
	for {
		select {
		// case <-tick:
		// 	beep()
		// 	elog.Info(1, "beep")
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				//elog.Info(1, "Received Interrogate...")
				changes <- c.CurrentStatus
				// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:

				elog.Info(1, "Received Shutdown...")
				logrus.Info("received shutdown")

				// stop all polling and save the details here
				// Commented out until I find a better way to save cache files
				// shutdown()

				break loop
			case svc.Pause:
				WinLogln("Received Pause...")
				elog.Info(1, "Received Pause...")
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
				//tick = slowtick
			case svc.Continue:
				WinLogln("Received Continue...")
				elog.Info(1, "Received Continue...")
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
				//tick = fasttick
			default:
				elog.Error(1, fmt.Sprintf("unexpected control request #%d", c))
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}

func runService(name string, isDebug bool) {
	var err error

	if isDebug {
		elog = debug.New(name)
	} else {
		elog, err = eventlog.Open(name)
		if err != nil {
			return
		}
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("starting %s service", name))

	var wd string
	wd, err = osext.ExecutableFolder()
	if err != nil {
		elog.Error(1, fmt.Sprintf("%s service ExecutableFolder failed: %v", name, err))
		return
	}
	elog.Info(1, fmt.Sprintf("osext working directory: %s", wd))

	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	err = run(name, &myservice{})
	if err != nil {
		elog.Error(1, fmt.Sprintf("%s service failed: %v", name, err))
		return
	}
	elog.Info(1, fmt.Sprintf("%s service stopped", name))
	logrus.Infof("%s service stopped", name)
}

// func handler(w http.ResponseWriter, r *http.Request) {
//     fmt.Fprintf(w, "ZZZ Hi there, I love %s!", r.URL.Path[1:])
// }
