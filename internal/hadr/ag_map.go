package hadr

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/scalesql/isitsql/internal/failure"
	"github.com/scalesql/isitsql/internal/fileio"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

// AGMap holds a map of all Availability Groups
// The key is the AG GUID
type AGMap struct {
	mux    *sync.RWMutex
	groups map[string]*AG
	names  [][]string
}

// NewAGMap
func NewAGMap() (AGMap, error) {
	agm := AGMap{}
	agm.mux = &sync.RWMutex{}
	agm.groups = make(map[string]*AG)
	// TODO start the file watcher
	//err := agm.StartWatching()
	return agm, nil
}

// Set an AG value for a given GUID
func (agm *AGMap) Set(key string, ag *AG) {
	agm.mux.Lock()
	defer agm.mux.Unlock()
	ag.SetDisplayName(agm.names)
	agm.groups[key] = ag

	// expire AGs that we haven't polled within five minutes
	for k, v := range agm.groups {
		// if v.PollTime is before five minutes ago, remove it
		if v.PollTime.Before(time.Now().Add(-5 * time.Minute)) {
			delete(agm.groups, k)
		}
	}
}

func (agm *AGMap) SetLatencies(key string, latencies []Latency) {
	agm.mux.Lock()
	defer agm.mux.Unlock()
	ag, ok := agm.groups[key]
	if !ok { // don't add if we don't have it yet
		return
	}
	ag.Latencies = latencies
}

// TODO: GetPrimaryDBLatency(group_id, group_database_id): send, redo
func (agm *AGMap) GetPrimaryDBLatency(groupID, databaseID string) (send int, redo int) {
	agm.mux.RLock()
	defer agm.mux.RUnlock()
	ag, ok := agm.groups[groupID]
	if !ok {
		return 0, 0
	}
	// sum the latencies for this database
	for _, l := range ag.Latencies {
		if l.GroupID == groupID && l.GroupDatabaseID == databaseID {
			if l.SendQueueKB > 0 {
				send += l.SendQueueKB
			}
			if l.RedoQueueKB > 0 {
				redo += l.RedoQueueKB
			}
		}
	}
	return send, redo
}

func (agm *AGMap) GetSecondaryDBLatency(groupID, replicaID, databaseID string) (send int, redo int) {
	agm.mux.RLock()
	defer agm.mux.RUnlock()
	ag, ok := agm.groups[groupID]
	if !ok {
		return 0, 0
	}
	// return the latencies for this database
	for _, l := range ag.Latencies {
		if l.GroupID == groupID && l.ReplicaID == replicaID && l.GroupDatabaseID == databaseID {
			return l.SendQueueKB, l.RedoQueueKB
		}
	}
	return 0, 0
}

// Get an AG
func (agm *AGMap) Get(key string) (bool, AG) {
	agm.mux.RLock()
	defer agm.mux.RUnlock()
	ag, ok := agm.groups[key]
	return ok, *ag
}

// GetDisplayName gets the display name for an AG
func (agm *AGMap) GetDisplayName(key string) string {
	agm.mux.RLock()
	defer agm.mux.RUnlock()
	ag, ok := agm.groups[key]
	if !ok {
		return ""
	}
	return ag.DisplayName
}

// Groups returns an array of all AGs
func (agm *AGMap) Groups() []AG {
	list := make([]AG, 0)
	agm.mux.Lock()
	defer agm.mux.Unlock()
	for _, ag := range agm.groups {
		list = append(list, *ag)
	}
	return list
}

// agm.SetAGNames([][]string) error//
// Each array entry needs 3 values: domain, AG name, display name
func (agm *AGMap) SetAGNames(names [][]string) (int, bool, error) {
	agm.mux.Lock()
	defer agm.mux.Unlock()
	dirty := false
	// the HCL file can come back in any order
	// so we sort by the domain and AG name before comparing
	sort.Slice(names, func(i, j int) bool {
		if names[i][0] < names[j][0] {
			return true
		}
		if names[i][1] < names[j][1] {
			return true
		}
		return false
	})
	if len(agm.names) != len(names) {
		dirty = true
	} else {
		for i := range agm.names {
			if !slices.Equal(agm.names[i], names[i]) {
				dirty = true
			}
		}
	}
	if !dirty {
		return 0, dirty, nil
	}
	agm.names = names
	return len(names), true, nil
}

// Names returns the string array with the display names
func (agm *AGMap) Names() [][]string {
	agm.mux.Lock()
	defer agm.mux.Unlock()
	return agm.names
}

// StartWatching
func (agm *AGMap) StartWatching() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		//log.Error(err)
		return errors.Wrap(err, "fsnotify.newwatcher")
	}
	ch := make(chan int)
	go coalesce(watcher.Events, ch)

	go func(agm *AGMap) {
		defer failure.HandlePanic()
		for {
			select {
			case count, ok := <-ch:
				if !ok {
					return
				}
				log.Infof("configuration file changed (events: %d)", count)
				agNames, err := fileio.ReadConfigCSV("ag_names.csv")
				if err != nil {
					log.Error(errors.Wrap(err, "fileio.readconfigcsv"))
				}
				n, dirty, err := agm.SetAGNames(agNames)
				if err != nil {
					log.Error(errors.Wrap(err, "agmap.setagnames"))
				} else {
					if dirty {
						log.Info(fmt.Sprintf("StartWatching: Availability Group Names Set: %d", n))
					}
				}

				time.Sleep(1 * time.Second)
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Error("error:", err)
			}
		}
	}(agm)

	ag_name_file := "C:\\dev\\github.com\\isitsql\\config\\ag_names.csv"
	err = watcher.Add(ag_name_file)
	if err != nil {
		log.Error(err)
		return errors.Wrap(err, "watcher.add")
	}
	log.Infof("watching file: %s", ag_name_file)

	return nil
}

// coalesce watches fsnotify events and returns when no new event has happened
// for one second or after five seconds
func coalesce(in <-chan fsnotify.Event, out chan<- int) {
	defer failure.HandlePanic()
	timer := time.NewTicker(1 * time.Second)
	var events int // count of events

	active := false
	first := time.Time{}
	last := time.Time{}

	for {
		select {
		case <-in:
			events++
			//log.Debugf("filewatcher-in: %s:%s (%d)", e.Name, e.Op.String(), events)
			last = time.Now()
			if !active {
				first = time.Now()
			}
			active = true

		case <-timer.C:
			if active {
				if time.Since(first) > time.Duration(5*time.Second) || time.Since(last) > time.Duration(2*time.Second) {
					//log.Debugf("filwatcher-out: active: %v first:%v last:%v", active, first, last)
					out <- events
					active = false
					events = 0
				}
			}
		}
	}
}
