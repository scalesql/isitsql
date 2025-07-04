package app

import (
	"runtime"
	"time"

	"github.com/scalesql/isitsql/internal/failure"
	"github.com/scalesql/isitsql/settings"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func launchPProfLogger() {
	defer failure.HandlePanic()
	logrus.Trace("Launch PProf Logger...")
	s, err := settings.ReadConfig()
	if err != nil {
		logrus.Error(errors.Wrap(err, "launchpprof:readconfig"))
		return
	}
	if s.PProfLogMB == 0 {
		return
	}
	logrus.Debugf("launchPProfLogger: pprof_log_mb: %d", s.PProfLogMB)

	quit := make(chan struct{})
	batchTicker := time.NewTicker(10 * time.Second)
	go func(mb int) {
		defer failure.HandlePanic()
		for {
			select {
			case <-batchTicker.C:
				var mem runtime.MemStats
				runtime.ReadMemStats(&mem)
				if mem.HeapSys/(1024*1024) > uint64(mb) {
					logrus.WithFields(logrus.Fields{
						"heap_alloc_mb": mem.HeapAlloc / (1024 * 1024),
						//"heap_inuse_mb":  mem.HeapInuse / (1024 * 1024),
						"heap_sys_mb": mem.HeapSys / (1024 * 1024),
						//"stack_inuse_mb": mem.StackInuse / (1024 * 1024),
						//"stack_sys_mb":   mem.StackSys / (1024 * 1024),
						"pprof_log_mb": mb,
					}).Infof("up=%v", durationToShortString(globalStats.StartTime, time.Now()))
					logMemory()
					logPProf()
					mb = mb * 2
				}
			case <-quit:
				batchTicker.Stop()
				return
			}
		}
	}(s.PProfLogMB)
}

func logPProf() {
	failure.WritePProf()
}

// launchMemoryLogger writes a log record of the memory every 24 hours
func launchMemoryLogger() {
	defer failure.HandlePanic()
	logrus.Debug("Launch Memory Logger...")

	quit := make(chan struct{})
	batchTicker := time.NewTicker(time.Duration(24) * time.Hour)
	time.Sleep(60 * time.Second)
	logMemory()
	time.Sleep(60 * time.Minute)
	logMemory()
	go func() {
		defer failure.HandlePanic()
		for {
			select {
			case <-batchTicker.C:
				logMemory()
			case <-quit:
				batchTicker.Stop()
				return
			}
		}
	}()
}

func logMemory() {
	// https://lemire.me/blog/2024/03/17/measuring-your-systems-performance-using-software-go-edition/
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	logrus.WithFields(logrus.Fields{
		"heap_alloc_mb":  mem.HeapAlloc / (1024 * 1024),
		"heap_inuse_mb":  mem.HeapInuse / (1024 * 1024),
		"heap_sys_mb":    mem.HeapSys / (1024 * 1024),
		"stack_inuse_mb": mem.StackInuse / (1024 * 1024),
		"stack_sys_mb":   mem.StackSys / (1024 * 1024),
	}).Infof("up=%v", durationToShortString(globalStats.StartTime, time.Now()))
}

// func launchGCLogger() {
// 	defer failure.HandlePanic()
// 	logrus.Debug("Launch GC Logger...")

// 	quit := make(chan struct{})
// 	batchTicker := time.NewTicker(time.Duration(10) * time.Second)
// 	t := time.Now()
// 	var lastTotalAlloc uint64
// 	go func() {
// 		defer failure.HandlePanic()
// 		for {
// 			select {
// 			case <-batchTicker.C:
// 				var stats runtime.MemStats
// 				runtime.ReadMemStats(&stats)
// 				//newt := time.Now()
// 				logrus.Debugf("Allocations/sec: %s", humanize.Bytes(uint64(float64((stats.TotalAlloc-lastTotalAlloc))/time.Since(t).Seconds())))
// 				t = time.Now()
// 				lastTotalAlloc = stats.TotalAlloc
// 			case <-quit:
// 				batchTicker.Stop()
// 				return
// 			}
// 		}
// 	}()
// }
