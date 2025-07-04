package app

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"runtime"
	"strings"

	"github.com/scalesql/isitsql/internal/build"
	"github.com/scalesql/isitsql/internal/failure"
	"github.com/pkg/errors"
	"github.com/shiena/ansicolor"
	"github.com/sirupsen/logrus"
	"go.uber.org/automaxprocs/maxprocs"
	"golang.org/x/sys/windows/svc"
)

func Run(strGit string, strDate string) {
	failure.BuildGit = strGit
	failure.BuildDate = strDate
	buildGit = strGit
	buildDate = strDate
	defer failure.HandlePanic()
	var fileLogging = false

	args := []string{}
	for _, flag := range os.Args {
		switch flag {
		case "-debug":
			if logrus.DebugLevel > logrus.GetLevel() {
				logrus.SetLevel(logrus.DebugLevel)
			}
		case "-trace":
			if logrus.TraceLevel > logrus.GetLevel() {
				logrus.SetLevel(logrus.TraceLevel)
			}
		case "-panic":
			panic("-panic flag on the command line")
		case "-log":
			fileLogging = true
		case "-version":
			fmt.Printf("main.version:  %s (%s)\n", strGit, strDate)
			fmt.Printf("build.version: %s (%s) [%s]\n", build.Version(), build.Commit(), build.Built())
			return
		default:
			args = append(args, flag)
		}
	}

	// Uncomment to do memory profiling
	// go func(){
	//     http.ListenAndServe(":8080", http.DefaultServeMux)
	// }()

	const svcName = "IsItSql"
	version = buildGit

	// IsWindowsService has a bug
	// https://github.com/golang/go/issues/44921
	// https://github.com/golang/go/issues/45966
	//lint:ignore SA1019 IsWindowsService has a bug
	isIntSess, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatal(errors.Wrap(err, "svc.isaninteractiveshell"))
	}

	// Set interactive flag
	if !isIntSess {
		IsInteractive = false
	}

	if !isIntSess || fileLogging {
		configureFileLogging()
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			ForceColors: true,
		})
		logrus.SetOutput(ansicolor.NewAnsiColorWriter(os.Stdout))
		//configureFileLogging()
	}

	WinLogln("Copyright (c) 2024 ScaleOut Consulting, LLC. All Rights Reserved.")
	versionLine := fmt.Sprintf("Version: %s from %s at %s", build.Version(), buildGit, buildDate)
	WinLogln(versionLine)
	var userName string
	u, err := user.Current()
	if err != nil {
		logrus.Error(errors.Wrap(err, "user.current"))
		userName = "unknown"
	} else {
		userName = u.Username
	}
	processLine := fmt.Sprintf("pid=%d  user=\"%s\"", os.Getpid(), userName)
	WinLogln(processLine)
	logrus.Debug("debug enabled")
	logrus.Trace("trace enabled")

	// Set GOMAXPROCS if we are running in a container
	cores := runtime.GOMAXPROCS(0)
	undo, err := maxprocs.Set()
	defer undo()
	if err != nil {
		WinLogln(errors.Wrap(err, "maxprocs"))
		logrus.Error(errors.Wrap(err, "maxprocs").Error())
	}
	newcores := runtime.GOMAXPROCS(0)
	if cores != newcores {
		msg := fmt.Sprintf("gomaxprocs: %d -> %d", cores, newcores)
		WinLogln(msg)
		logrus.Info(msg)
	}

	if !isIntSess {
		runService(svcName, false)
		return
	}

	if len(args) < 2 {
		// usage("no command specified")
		runService(svcName, true)
		return
	}

	cmd := strings.ToLower(args[1])
	switch cmd {
	case "/?":
		usage("Please choose a command")
	case "-?":
		usage("Please choose a command")
	case "debug":
		runService(svcName, true)
		return
	case "install":
		fmt.Println("Installing...")
		err = installService(svcName, "Is It SQL")
		if err != nil {
			fmt.Println("Installation Error: ", err)
			break
		}
		fmt.Println("Service installed")
	case "remove":
		err = removeService(svcName)
		if err != nil {
			fmt.Println("Removal error: ", err)
			break
		}
		fmt.Print("Service removed")
	case "start":
		err = startService(svcName)
	case "stop":
		log.Println("Stopping service...")
		err = controlService(svcName, svc.Stop, svc.Stopped)
	case "pause":
		err = controlService(svcName, svc.Pause, svc.Paused)
	case "continue":
		err = controlService(svcName, svc.Continue, svc.Running)
	default:
		usage(fmt.Sprintf("invalid command %s", cmd))
	}
	if err != nil {
		log.Fatalf("failed to %s %s: %v", cmd, svcName, err)
	}
}

func usage(errmsg string) {
	fmt.Fprintf(os.Stderr,
		"%s\n\n"+
			"usage: %s <command>\n"+
			"       where <command> is one of\n"+
			"       install, remove, debug, start, stop, pause or continue.\n",
		errmsg, "isitsql")
	os.Exit(2)
}
