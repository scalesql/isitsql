package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/scalesql/isitsql/internal/c2"
	"github.com/radovskyb/watcher"
	"github.com/shiena/ansicolor"
	log "github.com/sirupsen/logrus"
)

func main() {

	var debug, trace bool
	var watch, tick time.Duration
	var path string
	flag.BoolVar(&debug, "debug", false, "debug")
	flag.BoolVar(&trace, "trace", false, "trace")
	flag.DurationVar(&watch, "watch", 1*time.Second, "watch frequency - watch for file changes this often")
	flag.DurationVar(&tick, "tick", 60*time.Second, "tick frequency - always lint at least this often")
	flag.StringVar(&path, "path", "servers", "path to watch for changes")
	flag.Parse()

	log.SetFormatter(&log.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "15:04:05",
	})
	log.SetOutput(ansicolor.NewAnsiColorWriter(os.Stdout))
	if debug {
		log.SetLevel(log.DebugLevel)
	}
	if trace {
		log.SetLevel(log.TraceLevel)
	}
	msg := fmt.Sprintf("linter: path: '%s', poll: %s  tick: %s", path, watch, tick)
	if debug || trace {
		msg += fmt.Sprintf(" (%s)", strings.ToUpper(log.GetLevel().String()))
	}
	log.Info(msg)
	lintFiles()
	w := watcher.New()

	// If SetMaxEvents is not set, the default is to send all events.
	w.SetMaxEvents(1)

	r := regexp.MustCompile(`^.*\.hcl$`)
	w.AddFilterHook(watcher.RegexFilterHook(r, false))

	go func(tick time.Duration) {
		ticker := time.NewTicker(tick)
		for {
			select {
			case t := <-ticker.C:
				log.Trace("Tick at", t)
				lintFiles()
			case event := <-w.Event:
				log.Debug(event) // Print the event's info.
				lintFiles()
			case err := <-w.Error:
				log.Fatalln(err)
			case <-w.Closed:
				return
			}
		}
	}(tick)

	log.Debugf("add watch path: %s", path)
	if err := w.AddRecursive(path); err != nil {
		log.Fatalln(err)
	}
	for path, f := range w.WatchedFiles() {
		log.Tracef("watching: %s: %s\n", path, f.Name())
	}

	// Waitfor control-c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		log.Trace("GO: wait for signal...")
		<-c
		os.Exit(1)
	}()

	// Start the watching process
	go func(freq time.Duration) {
		log.Trace("GO: w.Start()...")
		if err := w.Start(freq); err != nil {
			log.Fatalln(err)
		}
	}(watch)

	for {
		time.Sleep(100 * time.Millisecond)
		runtime.Gosched()
	}
}

func lintFiles() {
	fc, msgs, err := c2.GetHCLFiles()
	if err != nil {
		log.Error(err)
	}
	for _, msg := range msgs {
		log.Error(msg)
	}
	if err != nil || len(msgs) > 0 {
		log.Errorf("instances: %d  ag: %d  (files: %d)", len(fc.Connections), len(fc.AGs), len(fc.Files))
	} else {
		log.Infof("instances: %d  ag: %d  (files: %d)", len(fc.Connections), len(fc.AGs), len(fc.Files))
	}
}
