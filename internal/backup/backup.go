package backup

import (
	"strings"
	"sync"
	"time"
)

var (
	mux     sync.RWMutex
	backups map[DBKey]Backup
)

func init() {
	backups = make(map[DBKey]Backup)
}

// DBKey identifies a unique backup row
type DBKey struct {
	Host       string
	Database string
}

// Backup is the most recent backup of a particular type for a database
type Backup struct {
	Host           string
	Database     string
	FullStarted  time.Time
	FullInstance string
	FullDevice   string
	LogStarted   time.Time
	LogInstance  string
	LogDevice    string
}

// Debug just dumps the map
// func Debug() map[DBKey]Backup {
// 	mux.RLock()
// 	defer mux.RUnlock()

// 	new := make(map[DBKey]Backup)
// 	for k, v := range backups {
// 		new[k] = v
// 	}
// 	return new
// }

// SetDatabases adds any new databases and removes any that don't exist any more
// func SetDatabases(host string, dbs []string) {
// 	mux.Lock()
// 	defer mux.Unlock()

// 	// Add any databases that don't exist
// 	for _, db := range dbs {
// 		set(host, db)
// 	}

// 	// Remove any databases that aren't in the array
// 	for k := range backups {
// 		if k.Host == strings.ToUpper(host) {
// 			if !stringInSlice(k.Database, dbs) {
// 				delete(backups, k)
// 			}
// 		}
// 	}
// }

// Set makes sure an entry exists
func Set(host string, db string) {
	mux.Lock()
	defer mux.Unlock()

	set(host, db)
}

func set(host string, db string) {
	key := DBKey{Host: strings.ToUpper(host), Database: strings.ToUpper(db)}
	_, found := backups[key]
	if !found {
		b := Backup{Host: host, Database: db}
		backups[key] = b
		//log.Print("Setting: ", key)
		//log.Print("Map Length: ", len(backups))
	}
}

// Get returns most recent backup of a type of an AG database
func Get(host string, db string) (backup Backup, found bool) {

	mux.RLock()
	defer mux.RUnlock()

	key := DBKey{Host: strings.ToUpper(host), Database: strings.ToUpper(db)}
	backup, found = backups[key]

	return backup, found
}

// Delete removes all backups for a database in an AG
func Delete(host string, db string) {
	mux.Lock()
	mux.Unlock()

	for k := range backups {
		if k.Host == strings.ToUpper(host) && k.Database == strings.ToUpper(db) {
			delete(backups, k)
		}
	}
}

// SetFull records that a full backup took place against an AG database
func SetFull(host string, db string, started time.Time, instance string, device string) {

	mux.Lock()
	defer mux.Unlock()

	key := DBKey{Host: strings.ToUpper(host), Database: strings.ToUpper(db)}
	backup, found := backups[key]
	if found {
		if backup.FullStarted.Before(started) {
			backup.FullStarted = started
			backup.FullInstance = instance
			backup.FullDevice = device
			backups[key] = backup
			return
		}
	} else {
		backup.Host = host
		backup.Database = db
		backup.FullInstance = instance
		backup.FullStarted = started
		backup.FullDevice = device
		backups[key] = backup
		return
	}
}

// SetLog records that a log backup took place against an AG database
func SetLog(host string, db string, started time.Time, instance string, device string) {

	mux.Lock()
	defer mux.Unlock()

	key := DBKey{Host: strings.ToUpper(host), Database: strings.ToUpper(db)}
	backup, found := backups[key]
	if found {
		if backup.LogStarted.Before(started) {
			backup.LogStarted = started
			backup.LogInstance = instance
			backup.LogDevice = device
			backups[key] = backup
			return
		}
	} else {
		backup.Host = host
		backup.Database = db
		backup.LogInstance = instance
		backup.LogStarted = started
		backup.LogDevice = device
		backups[key] = backup
		return
	}
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if strings.ToUpper(b) == strings.ToUpper(a) {
			return true
		}
	}
	return false
}
