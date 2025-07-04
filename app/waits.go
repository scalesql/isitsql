package app

import (
	"time"

	"github.com/scalesql/isitsql/internal/waitmap"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// **************************************************
// Methods
// **************************************************

// Note: this code isn't called as of now
// And it needs to handle the ResetOnThisPoll flag
// func (s *SqlServerWrapper) pollWaits2() error {
// 	var err error

// 	s.RLock()
// 	cxnType := s.ConnectionType
// 	cxnSTring := s.ConnectionString
// 	s.RUnlock()
// 	db, err := sql.Open(cxnType, cxnSTring)
// 	if err != nil {
// 		return err
// 	}
// 	defer db.Close()

// 	var rows *sql.Rows
// 	rows, err = db.Query("select wait_type, wait_time_ms from sys.dm_os_wait_stats where wait_time_ms > 0;")
// 	if err != nil {
// 		return err
// 	}
// 	defer rows.Close()

// 	var waits Waits
// 	waits.EventTime = time.Now()

// 	waits.Waits = make(map[string]Wait, 100)
// 	for rows.Next() {
// 		var w Wait
// 		w.SignalWaitTime = 0
// 		w.SignalWaitTimeDelta = 0
// 		w.WaitTimeDelta = 0

// 		err := rows.Scan(&w.Wait, &w.WaitTime)
// 		if err != nil {
// 			//log.Fatal(err)
// 			return err
// 		}

// 		waits.Waits[w.Wait] = w
// 	}

// 	// Get the previous waits
// 	s.RLock()
// 	previousWaits := s.CurrentWaits
// 	s.RUnlock()

// 	//var wg waitgroupring.SQLWaitGroup

// 	if previousWaits == nil {
// 		s.Lock()
// 		s.CurrentWaits = &waits
// 		s.Unlock()
// 		return nil
// 	}

// 	// Build the wait groups
// 	//wg, err = buildWaitGroups(previousWaits, &waits)

// 	// Enqueue them
// 	if err == nil {
// 		servers.Lock()
// 		s.CurrentWaits = &waits
// 		//		s.WaitGroups.Enqueue(&wg)
// 		servers.Unlock()
// 	} else {
// 		WinLogln("buildWaitGroups Error: ", err)
// 	}

// 	return nil
// }

// func buildWaitGroups(previous, current *Waits) (waitgroupring.SQLWaitGroup, error) {
// 	var wg waitgroupring.SQLWaitGroup
// 	var err error
// 	var waitTime int64

// 	wg.Groups = make(map[string]int64, 10)
// 	wg.EventTime = current.EventTime
// 	wg.Duration = current.EventTime.Sub(previous.EventTime)

// 	// Lock the wait map with defer
// 	WAIT_MAPPINGS.RLock()
// 	defer WAIT_MAPPINGS.RUnlock()

// 	// Loop through the waits
// 	mapTo := ""

// 	for key, value := range current.Waits {

// 		// get the wait time
// 		p, previousExists := previous.Waits[key]
// 		if previousExists {
// 			waitTime = value.WaitTime - p.WaitTime
// 		} else {
// 			waitTime = value.WaitTime
// 		}

// 		if waitTime < 0 {
// 			waitTime = 0
// 		}

// 		// Get the wait group
// 		mapTo = ""
// 		wm, ok := WAIT_MAPPINGS.Mappings[key]
// 		if !ok {
// 			mapTo = key // we didn't find a mapping
// 		} else {
// 			if wm.Excluded {
// 				mapTo = "" // we found a mapping but it's excluded
// 			} else {
// 				mapTo = wm.MappedTo
// 			}
// 		}

// 		if mapTo != "" {
// 			_, ok = wg.Groups[mapTo]
// 			if ok {
// 				wg.Groups[mapTo] = wg.Groups[mapTo] + waitTime

// 			} else {
// 				wg.Groups[mapTo] = waitTime
// 			}
// 			//log.Printf("%s: %d", mapTo, value.WaitTimeDelta)
// 		}

// 	}

// 	//WinLogln(mapTo)

// 	// if it exists -> increase by the different
// 	// if it doesn't exist, -> write with the difference

// 	return wg, err
// }

// PollWaits polls the database for waits
func (s *SqlServerWrapper) PollWaits() error {
	var err error
	s.RLock()
	db := s.DB
	reset := s.ResetOnThisPoll
	previousWaits := s.LastWaits // we need the previous waits so we can DIFF
	key := s.MapKey
	pollCount := s.PollCount
	s.RUnlock()

	rows, err := db.Query("select wait_type, wait_time_ms from sys.dm_os_wait_stats where wait_time_ms > 0;")
	if err != nil {
		return errors.Wrap(err, "query")
	}
	defer rows.Close()

	var waits waitmap.Waits

	waits.EventTime = time.Now()
	if previousWaits != nil {
		waits.Duration = waits.EventTime.Sub(previousWaits.EventTime)
	}

	waits.Waits = make(map[string]waitmap.Wait, 200)
	for rows.Next() {
		var w waitmap.Wait
		err := rows.Scan(&w.Wait, &w.WaitTime)
		if err != nil {
			return errors.Wrap(err, "scan")
		}

		// we need previous waits, duration > 0, and no reset
		// in order to calculate the delta
		if previousWaits != nil && waits.Duration != 0 && !reset {
			pw, ok := previousWaits.Waits[w.Wait]
			if ok {
				totalDelta := float64(w.WaitTime - pw.WaitTime)
				if totalDelta > 0.0 {
					// get the per second value
					delta := totalDelta / waits.Duration.Seconds()
					w.WaitTimeDelta = int64(delta * 60.0) // convert to per minute
				}
			}
		}
		waits.Waits[w.Wait] = w
	}

	// logrus.Debugf("[%s] pollwaits: waits: %d", key, len(waits.Waits))

	// Figure out the wait Summary
	waits.SetWaitGroups()
	s.Lock()
	s.LastWaits = &waits
	s.Unlock()

	// write the waits to the bucket
	// err = globalWaitsBucket.Write(s.MapKey, waits)
	// if err != nil {
	// 	return errors.Wrap(err, "bucketwriter.write")
	// }

	// write locally
	err = waitmap.WriteWaitFile(key, waits)
	if err != nil {
		logrus.Error(errors.Wrap(err, "s.writewaits"))
	}

	// purge every 10th poll
	if pollCount%10 == 9 {
		err = waitmap.PurgeWaitFiles(key)
		if err != nil {
			logrus.Error(errors.Wrap(err, "purgewaitfiles"))
		}
	}

	return nil
}
