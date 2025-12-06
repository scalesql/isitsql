package waitmap

import (
	"encoding/csv"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kardianos/osext"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var Mapping WaitMapping

// **************************************************
// Structures
// **************************************************

// Wait is one row of sys.dm_os_wait_stats
type Wait struct {
	Wait                string `json:"wait"`
	WaitTime            int64  `json:"wait_time,omitempty"`
	SignalWaitTime      int64  `json:"signal_wait_time,omitempty"`
	WaitTimeDelta       int64  `json:"wait_time_delta,omitempty"`
	SignalWaitTimeDelta int64  `json:"signal_wait_time_delta,omitempty"`
}

// Waits is the grouping off all the waits that
// were queried at particular time and for a
// specified duration
type Waits struct {
	EventTime   time.Time        `json:"event_time"`
	Waits       map[string]Wait  `json:"waits"`
	WaitSummary map[string]int64 `json:"wait_summary"`
	Duration    time.Duration    `json:"duration"`
}

// WaitMapping is all the mappings from wait types to wait groups
// The key of the map is the wait
type WaitMapping struct {
	Mappings     map[string]WaitMap
	sync.RWMutex // This protects the map from wait to wait group
}

func (wm *WaitMapping) Lookup(wait string) WaitMap {
	wm.RLock()
	defer wm.RUnlock()
	// if we don't find it in the map, it will be an empty WaitMap
	v := wm.Mappings[wait]
	return v
}

// WaitMap is what group a wait is mapped to and
// whether it is excluded
type WaitMap struct {
	//	Wait     string
	MappedTo string
	Excluded bool
}

// SetWaitGroups maps wait to wait groups
func (w *Waits) SetWaitGroups() error {
	w.WaitSummary = make(map[string]int64)
	//ok bool
	var mapTo string
	Mapping.RLock()
	defer Mapping.RUnlock()
	for key, value := range w.Waits {
		if w.Waits[key].WaitTimeDelta > 0 {
			// log.Printf("Mapping a wait: %s", key)
			mapTo = ""
			wm, ok := Mapping.Mappings[key]
			if !ok {
				mapTo = key // we didn't find a mapping
			} else {
				if wm.Excluded {
					mapTo = "" // we found a mapping but this wait is excluded
				} else {
					mapTo = wm.MappedTo
				}
			}

			if mapTo != "" {
				_, ok = w.WaitSummary[mapTo]
				if ok {
					w.WaitSummary[mapTo] = w.WaitSummary[mapTo] + value.WaitTimeDelta

				} else {
					w.WaitSummary[mapTo] = value.WaitTimeDelta
				}
				//log.Printf("%s: %d", mapTo, value.WaitTimeDelta)
			}
		}
	}

	//log.Printf("Wait Summary Count: %d", len(w.WaitSummary)

	return nil
}

func checkForUserWaitsFile() error {

	wd, err := osext.ExecutableFolder()
	if err != nil {
		return err
	}
	// WinLogln("Current Directory: " + dir)
	//dir = dir + "/config/"
	waitfile := filepath.Join(wd, "config", "waits.txt")

	// if the file doesn't exist, then write it
	if _, err := os.Stat(waitfile); os.IsNotExist(err) {

		waitFileHeader := `#########################################################################
#
# A base mapping of wait types is provided in waits_base.txt
# This file is used to override those mappings
#
# This file maps waits to common grouping for display
# The first column is the wait type returned by SQL Server
# The second column is the text of the wait grouping that is displayed
#
# * If the second column is blank or doesn't exist, the wait 
#   will be excluded
# * A duplicate will override a previous entry
# * Blank lines are ignored
# * A line starting with # is a comment
#
# This file is read at each polling
#
########################################################################

# Sample Entries
# LCK_M_IU,Locking     # is mapped to the Locking group
# WAITFOR,		       # is excluded


`
		waitFileHeader = strings.Replace(waitFileHeader, "\n", "\r\n", -1)
		err = os.WriteFile(waitfile, []byte(waitFileHeader), 0600)
		if err != nil {
			return errors.Wrap(err, "os.writefile")
		}
		// WinLogln("/config/waits.txt created.")
	}

	return nil

}

func (wm *WaitMapping) ReadWaitMapping(fileName string) error {

	// GLOBAL_RINGLOG.Enqueue("Reading " + fileName + " file...")

	if fileName == "waits.txt" {
		err := checkForUserWaitsFile()
		if err != nil {
			return err
		}
	}

	dir, err := osext.ExecutableFolder()
	if err != nil {
		log.Fatal(err)
	}
	// WinLogln("Current Directory: " + dir)
	dir = dir + "/config/"
	fullfile := filepath.Join(dir, fileName)
	/* #nosec G304 */
	csvfile, err := os.Open(fullfile)
	if err != nil {
		return errors.Wrap(err, "os.open")
	}

	defer func() {
		if err := csvfile.Close(); err != nil {
			logrus.Errorf("waitmap: csvfile.close: %s", err.Error())
		}
	}()

	reader := csv.NewReader(csvfile)
	reader.Comma = ','
	reader.Comment = '#'
	reader.TrimLeadingSpace = true

	// Set to -1 to allow a variable number of fields
	reader.FieldsPerRecord = -1

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			//WinLogln("*********************************************************")
			logrus.Errorf("Bad row in wwaits: %s: %s", fileName, err.Error())
			//WinLogln("*********************************************************")
		} else {
			var waitGroup string
			var excluded bool
			key := strings.ToUpper(record[0])

			if len(record) >= 2 {
				waitGroup = record[1]
			}

			if waitGroup == "" {
				excluded = true
			}

			mapping := WaitMap{MappedTo: waitGroup, Excluded: excluded}
			wm.Lock()
			//_, ok := wm.Mappings[key]
			// if ok {
			// 	WinLogln("Duplicate wait type in config/waits.txt.  Overwriting. ", key, waitGroup)
			// }
			wm.Mappings[key] = mapping
			wm.Unlock()

		}

	}

	return err
}

func (wm *WaitMapping) SetBaseWaitMapping() {

	wm.Lock()
	defer wm.Unlock()

	wm.Mappings["ASSEMBLY_LOAD"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["ASYNC_DISKPOOL_LOCK"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["ASYNC_IO_COMPLETION"] = WaitMap{MappedTo: "Disk IO", Excluded: false}
	wm.Mappings["ASYNC_NETWORK_IO"] = WaitMap{MappedTo: "Network", Excluded: false}
	wm.Mappings["AUDIT_GROUPCACHE_LOCK"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["BACKUP"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["BACKUPBUFFER"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["BACKUPIO"] = WaitMap{MappedTo: "Backup", Excluded: false}
	wm.Mappings["BACKUPTHREAD"] = WaitMap{MappedTo: "Backup", Excluded: false}
	wm.Mappings["BROKER_ENDPOINT_STATE_MUTEX"] = WaitMap{MappedTo: "Broker", Excluded: false}
	wm.Mappings["BROKER_EVENTHANDLER"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["BROKER_MASTERSTART"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["BROKER_RECEIVE_WAITFOR"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["BROKER_SERVICE"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["BROKER_SHUTDOWN"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["BROKER_TASK_STOP"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["BROKER_TO_FLUSH"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["BROKER_TRANSMISSION_OBJECT"] = WaitMap{MappedTo: "Broker", Excluded: false}
	wm.Mappings["BROKER_TRANSMISSION_TABLE"] = WaitMap{MappedTo: "Broker", Excluded: false}
	wm.Mappings["BROKER_TRANSMISSION_WORK"] = WaitMap{MappedTo: "Broker", Excluded: false}
	wm.Mappings["BROKER_TRANSMITTER"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["CLR_MANUAL_EVENT"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["BUILTIN_HASHKEY_MUTEX"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["CHECKPOINT_QUEUE"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["CLEAR_DB"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["CLR_AUTO_EVENT"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["CLR_CRST"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["CLR_MONITOR"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["CLR_SEMAPHORE"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["CLR_TASK_START"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["CMEMTHREAD"] = WaitMap{MappedTo: "CMEM Thread", Excluded: false}
	wm.Mappings["COMMIT_TABLE"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["CREATE_DATINISERVICE"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["CXPACKET"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["CXCONSUMER"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["CXROWSET_SYNC"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["DAC_INIT"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["DBMIRROR_DBM_EVENT"] = WaitMap{MappedTo: "Sync Mirror", Excluded: false}
	wm.Mappings["DBMIRRORING_CMD"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["DBMIRROR_EVENTS_QUEUE"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["DBMIRROR_SEND"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["DBMIRROR_DBM_MUTEX"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["DEADLOCK_ENUM_MUTEX"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["DEADLOCK_TASK_SEARCH"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["DEBUG"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["DIRTY_PAGE_POLL"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["DISPATCHER_QUEUE_SEMAPHORE"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["DTC"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["DTC_ABORT_REQUEST"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["DTC_STATE"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["EE_PMOLOCK"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["EE_SPECPROC_MAP_INIT"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["EXCHANGE"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["EXECUTION_PIPE_EVENT_INTERNAL"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["FCB_REPLICA_READ"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["FCB_REPLICA_WRITE"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["FFT_RECOVERY"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["FT_IFTS_RWLOCK"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["FT_IFTS_SCHEDULER_IDLE_WAIT"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["HADR_CLUSAPI_CALL"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["HADR_FILESTREAM_IOMGR"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["HADR_FILESTREAM_IOMGR_IOCOMPLETION"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["HADR_LOGCAPTURE_WAIT"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["HADR_NOTIFICATION_DEQUEUE"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["HADR_SYNC_COMMIT"] = WaitMap{MappedTo: "", Excluded: false}
	wm.Mappings["HADR_TIMER_TASK"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["HADR_WORK_QUEUE"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["IMPPROV_IOWAIT"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["IO_AUDIT_MUTEX"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["IO_COMPLETION"] = WaitMap{MappedTo: "Disk IO", Excluded: false}
	wm.Mappings["LATCH_EX"] = WaitMap{MappedTo: "Latch", Excluded: false}
	wm.Mappings["LATCH_SH"] = WaitMap{MappedTo: "Latch", Excluded: false}
	wm.Mappings["LATCH_UP"] = WaitMap{MappedTo: "Latch", Excluded: false}
	wm.Mappings["LAZYWRITER_SLEEP"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["LCK_M_IS"] = WaitMap{MappedTo: "Lock", Excluded: false}
	wm.Mappings["LCK_M_IU"] = WaitMap{MappedTo: "Lock", Excluded: false}
	wm.Mappings["LCK_M_IX"] = WaitMap{MappedTo: "Lock", Excluded: false}
	wm.Mappings["LCK_M_RI_NL"] = WaitMap{MappedTo: "Lock", Excluded: false}
	wm.Mappings["LCK_M_RS_S"] = WaitMap{MappedTo: "Lock", Excluded: false}
	wm.Mappings["LCK_M_RS_U"] = WaitMap{MappedTo: "Lock", Excluded: false}
	wm.Mappings["LCK_M_S"] = WaitMap{MappedTo: "Lock", Excluded: false}
	wm.Mappings["LCK_M_SCH_M"] = WaitMap{MappedTo: "Lock", Excluded: false}
	wm.Mappings["LCK_M_SCH_S"] = WaitMap{MappedTo: "Lock", Excluded: false}
	wm.Mappings["LCK_M_SIX"] = WaitMap{MappedTo: "Lock", Excluded: false}
	wm.Mappings["LCK_M_U"] = WaitMap{MappedTo: "Lock", Excluded: false}
	wm.Mappings["LCK_M_X"] = WaitMap{MappedTo: "Lock", Excluded: false}
	wm.Mappings["LOGBUFFER"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["LOGMGR_FLUSH"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["LOGMGR_QUEUE"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["LOGMGR_RESERVE_APPEND"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["LOGPOOL_CONSUMER"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["LOGPOOL_CONSUMERSET"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["LOGPOOL_REPLACEMENTSET"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["MEMORY_ALLOCATION_EXT"] = WaitMap{MappedTo: "", Excluded: false}
	wm.Mappings["METADATA_LAZYCACHE_RWLOCK"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["MSQL_DQ"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["MSQL_XACT_MGR_MUTEX"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["MSQL_XP"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["MSSEARCH"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["NET_WAITFOR_PACKET"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["NODE_CACHE_MUTEX"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["OLEDB"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["ONDEMAND_TASK_QUEUE"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["PAGEIOLATCH_EX"] = WaitMap{MappedTo: "Disk IO", Excluded: false}
	wm.Mappings["PAGEIOLATCH_SH"] = WaitMap{MappedTo: "Disk IO", Excluded: false}
	wm.Mappings["PAGEIOLATCH_UP"] = WaitMap{MappedTo: "Disk IO", Excluded: false}
	wm.Mappings["PAGELATCH_EX"] = WaitMap{MappedTo: "PageLatch", Excluded: false}
	wm.Mappings["PAGELATCH_KP"] = WaitMap{MappedTo: "PageLatch", Excluded: false}
	wm.Mappings["PAGELATCH_SH"] = WaitMap{MappedTo: "PageLatch", Excluded: false}
	wm.Mappings["PAGELATCH_UP"] = WaitMap{MappedTo: "PageLatch", Excluded: false}
	wm.Mappings["PARALLEL_BACKUP_QUEUE"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["PARALLEL_REDO_WORKER_WAIT_WORK"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["PERFORMANCE_COUNTERS_RWLOCK"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["PREEMPTIVE_CLOSEBACKUPVDIDEVICE"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_CLUSAPI_CLUSTERRESOURCECONTROL"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_COM_COCREATEINSTANCE"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_COM_CREATEACCESSOR"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_COM_GETCOMMANDTEXT"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_COM_GETDATA"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_COM_GETRESULT"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_COM_GETROWSBYBOOKMARK"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_COM_QUERYINTERFACE"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_COM_RELEASE"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_COM_RELEASEACCESSOR"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_COM_RELEASEROWS"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_COM_RELEASESESSION"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_COM_RESTARTPOSITION"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_COM_SETPARAMETERINFO"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_COM_SETPARAMETERPROPERTIES"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_CREATEPARAM"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_DEBUG"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_DTC_ABORTREQUESTDONE"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_DTC_BEGINTRANSACTION"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_DTC_COMMITREQUESTDONE"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_DTC_ENLIST"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_DTC_PREPAREREQUESTDONE"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_FILESIZEGET"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_LOCKMONITOR"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_ODBCOPS"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OLEDBOPS"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OLEDB_ABORTORCOMMITTRAN"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OLEDB_ABORTTRAN"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OLEDB_GETDATASOURCE"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OLEDB_GETLITERALINFO"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OLEDB_GETPROPERTIES"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OLEDB_GETPROPERTYINFO"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OLEDB_GETSCHEMALOCK"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OLEDB_RELEASE"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OLEDB_SETPROPERTIES"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OLE_UNINIT"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_ACCEPTSECURITYCONTEXT"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_ACQUIRECREDENTIALSHANDLE"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_AUTHENTICATIONOPS"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_AUTHORIZATIONOPS"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_AUTHZGETINFORMATIONFROMCONTEXT"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_AUTHZINITIALIZECONTEXTFROMSID"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_AUTHZINITIALIZERESOURCEMANAGER"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_CLOSEHANDLE"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_CLUSTEROPS"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_COMOPS"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_CREATEFILE"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_CRYPTACQUIRECONTEXT"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_CRYPTIMPORTKEY"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_CRYPTOPS"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_DECRYPTMESSAGE"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_DELETEFILE"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_DELETESECURITYCONTEXT"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_DEVICEIOCONTROL"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_DEVICEOPS"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_DIRSVC_NETWORKOPS"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_DISCONNECTNAMEDPIPE"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_DOMAINSERVICESOPS"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_DTCOPS"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_ENCRYPTMESSAGE"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_FILEOPS"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_FLUSHFILEBUFFERS"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_GENERICOPS"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_GETADDRINFO"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_GETCOMPRESSEDFILESIZE"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_GETDISKFREESPACE"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_GETFILEATTRIBUTES"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_GETPROCADDRESS"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_GETVOLUMENAMEFORVOLUMEMOUNTPOINT"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_GETVOLUMEPATHNAME"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_INITIALIZESECURITYCONTEXT"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_LIBRARYOPS"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_LOADLIBRARY"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_LOOKUPACCOUNTSID"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_MOVEFILE"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_NETVALIDATEPASSWORDPOLICY"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_NETVALIDATEPASSWORDPOLICYFREE"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_PIPEOPS"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_QUERYCONTEXTATTRIBUTES"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_QUERYREGISTRY"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_REPORTEVENT"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_REVERTTOSELF"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_SECURITYOPS"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_SETFILEVALIDDATA"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_SETNAMEDSECURITYINFO"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_SQMLAUNCH"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_WAITFORSINGLEOBJECT"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_WINSOCKOPS"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_WRITEFILE"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_WRITEFILEGATHER"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_OS_WSASETLASTERROR"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_SB_STOPENDPOINT"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_SP_SERVER_DIAGNOSTICS"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["PREEMPTIVE_TRANSIMPORT"] = WaitMap{MappedTo: "Preemptive", Excluded: false}
	wm.Mappings["PREEMPTIVE_XE_CALLBACKEXECUTE"] = WaitMap{MappedTo: "XE", Excluded: false}
	wm.Mappings["PREEMPTIVE_XE_DISPATCHER"] = WaitMap{MappedTo: "XE", Excluded: false}
	wm.Mappings["PREEMPTIVE_XE_GETTARGETSTATE"] = WaitMap{MappedTo: "XE", Excluded: true}
	wm.Mappings["PREEMPTIVE_XE_SESSIONCOMMIT"] = WaitMap{MappedTo: "XE", Excluded: false}
	wm.Mappings["PREEMPTIVE_XE_TARGETFINALIZE"] = WaitMap{MappedTo: "XE", Excluded: false}
	wm.Mappings["PREEMPTIVE_XE_TARGETINIT"] = WaitMap{MappedTo: "XE", Excluded: false}
	wm.Mappings["PREEMPTIVE_XE_TIMERRUN"] = WaitMap{MappedTo: "XE", Excluded: false}
	wm.Mappings["PWAIT_EXTENSIBILITY_CLEANUP_TASK"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["PWAIT_DIRECTLOGCONSUMER_GETNEXT"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["QDS_ASYNC_QUEUE"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["QDS_CLEANUP_STALE_QUERIES_TASK_MAIN_LOOP_SLEEP"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["QDS_PERSIST_TASK_MAIN_LOOP_SLEEP"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["QDS_SHUTDOWN_QUEUE"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["QDS_TASK_SHUTDOWN"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["QDS_TASK_START"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["QUERY_EXECUTION_INDEX_SORT_EVENT_OPEN"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["QUERY_NOTIFICATION_TABLE_MGR_MUTEX"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["QUERY_TASK_ENQUEUE_MUTEX"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["REDO_THREAD_PENDING_WORK"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["REPLICA_WRITES"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["REPL_SCHEMA_ACCESS"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["REQUEST_DISPENSER_PAUSE"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["REQUEST_FOR_DEADLOCK_SEARCH"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["RESOURCE_SEMAPHORE"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["RESOURCE_SEMAPHORE_MUTEX"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["RESOURCE_SEMAPHORE_QUERY_COMPILE"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["SECURITY_CRYPTO_CONTEXT_MUTEX"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["SEQUENTIAL_GUID"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["SLEEP_BPOOL_FLUSH"] = WaitMap{MappedTo: "Disk IO", Excluded: false}
	wm.Mappings["SLEEP_BUFFERPOOL_HELPLW"] = WaitMap{MappedTo: "Disk IO", Excluded: false}
	wm.Mappings["SLEEP_SYSTEMTASK"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["SLEEP_TASK"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["SLEEP_MSDBSTARTUP"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["SNI_CRITICAL_SECTION"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["SNI_TASK_COMPLETION"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["SOSHOST_MUTEX"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["SOS_DISPATCHER_MUTEX"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["SOS_LOCALALLOCATORLIST"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["SOS_PHYS_PAGE_CACHE"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["SOS_PROCESS_AFFINITY_MUTEX"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["SOS_RESERVEDMEMBLOCKLIST"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["SOS_SCHEDULER_YIELD"] = WaitMap{MappedTo: "CPU", Excluded: false}
	wm.Mappings["SOS_SYNC_TASK_ENQUEUE_EVENT"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["SOS_WORK_DISPATCHER"] = WaitMap{MappedTo: "Other", Excluded: true}
	wm.Mappings["SP_SERVER_DIAGNOSTICS_BUFFER_ACCESS"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["SP_SERVER_DIAGNOSTICS_SLEEP"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["SQLCLR_APPDOMAIN"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["SQLCLR_ASSEMBLY"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["SQLCLR_QUANTUM_PUNISHMENT"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["SQLSORT_SORTMUTEX"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["SQLTRACE_BUFFER_FLUSH"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["SQLTRACE_FILE_BUFFER"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["SQLTRACE_FILE_WRITE_IO_COMPLETION"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["SQLTRACE_INCREMENTAL_FLUSH_SLEEP"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["SQLTRACE_LOCK"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["SQLTRACE_WAIT_ENTRIES"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["THREADPOOL"] = WaitMap{MappedTo: "Thread Pool", Excluded: false}
	wm.Mappings["TRACEWRITE"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["TRACE_EVTNOTIF"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["TRANSACTION_MUTEX"] = WaitMap{MappedTo: "TXN Mutex", Excluded: false}
	wm.Mappings["VDI_CLIENT_OTHER"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["VIEW_DEFINITION_MUTEX"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["WAIT_XTP_OFFLINE_CKPT_NEW_LOG"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["WAITFOR"] = WaitMap{MappedTo: "", Excluded: true}
	wm.Mappings["WAITSTAT_MUTEX"] = WaitMap{MappedTo: "Other", Excluded: false}
	wm.Mappings["WRITELOG"] = WaitMap{MappedTo: "WriteLog", Excluded: false}
	wm.Mappings["WRITE_COMPLETION"] = WaitMap{MappedTo: "Disk IO", Excluded: false}
	wm.Mappings["XACTWORKSPACE_MUTEX"] = WaitMap{MappedTo: "Disk IO", Excluded: false}
	wm.Mappings["XE_BUFFERMGR_ALLPROCESSED_EVENT"] = WaitMap{MappedTo: "XE", Excluded: false}
	wm.Mappings["XE_DISPATCHER_WAIT"] = WaitMap{MappedTo: "XE", Excluded: true}
	wm.Mappings["XE_SERVICES_MUTEX"] = WaitMap{MappedTo: "XE", Excluded: false}
	wm.Mappings["XE_TIMER_EVENT"] = WaitMap{MappedTo: "XE", Excluded: true}
	wm.Mappings["XE_TIMER_MUTEX"] = WaitMap{MappedTo: "XE", Excluded: false}
	wm.Mappings["XE_FILE_TARGET_TVF"] = WaitMap{MappedTo: "XE", Excluded: true}
	wm.Mappings["XE_LIVE_TARGET_TVF"] = WaitMap{MappedTo: "XE", Excluded: true}
}
