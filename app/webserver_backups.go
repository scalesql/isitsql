package app

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/scalesql/isitsql/internal/backup"
)

func allBackupsPage(w http.ResponseWriter, req *http.Request) {
	var err error
	var ignored [][]string
	htmlTitle := html.EscapeString("Backup Issues - Is It SQL")

	//t0 := time.Now()

	type instanceKey struct {
		Domain     string
		ServerName string
	}

	type dbKey struct {
		Domain       string
		ServerName   string
		DatabaseName string
	}

	type backupDetail struct {
		Domain                string
		ServerName            string
		DatabaseName          string
		LastBackup            time.Time
		LastBackupDevice      string
		LastBackupInstance    string
		LastLogBackup         time.Time
		LastLogBackupDevice   string
		LastLogBackupInstance string
		RecoveryModelDesc     string
		DataSizeKB            int64
		LogSizeKB             int64
		CreateDate            time.Time
		ServerMapKey          string
		CurrentTime           time.Time
	}

	instanceAlerts := make(map[instanceKey]string)
	dbAlerts := make(map[dbKey]*backupDetail)

	instances := servers.Pointers()
	for _, s := range instances {

		s.RLock()
		msg := s.BackupMessage
		ik := instanceKey{
			Domain:     s.Domain,
			ServerName: s.ServerName,
		}
		s.RUnlock()
		// get any instance alerts
		if msg != "" {
			s.RLock()
			if _, ok := instanceAlerts[ik]; !ok {
				instanceAlerts[ik] = fmt.Sprintf("%s (%s): %s", s.ServerName, s.Domain, s.BackupMessage)
			}
			s.RUnlock()

			// if we found an instance alert, don't put the database alerts in
			continue
		}

		// get any database alerts
		s.RLock()
		for _, d := range s.Databases {
			var dk dbKey
			//fmt.Println("      ", d.Name, d.BackupAlert)

			// Since we are pulling backups from any node
			// We no longer care about the preferred backups
			// If it has an alert, include it
			if d.BackupAlert /* && d.IsPreferredBackup */ {
				dk = dbKey{
					Domain:       strings.ToUpper(s.Domain),
					ServerName:   strings.ToUpper(s.ServerName),
					DatabaseName: strings.ToUpper(d.Name),
				}
				//fmt.Println("      ", dk)

				if _, ok := dbAlerts[dk]; !ok {
					//fmt.Println("* Adding: ", dk)
					detail := backupDetail{
						Domain:                s.Domain,
						ServerName:            s.ServerName,
						DatabaseName:          d.Name,
						LastBackup:            d.LastBackup,
						LastBackupDevice:      d.LastBackupDevice,
						LastBackupInstance:    d.LastBackupInstance,
						LastLogBackup:         d.LastLogBackup,
						LastLogBackupDevice:   d.LastLogBackupDevice,
						LastLogBackupInstance: d.LastLogBackupInstance,
						RecoveryModelDesc:     d.RecoveryModelDesc,
						DataSizeKB:            d.DataSizeKB,
						LogSizeKB:             d.LogSizeKB,
						CreateDate:            d.CreateDate,
						ServerMapKey:          s.MapKey,
						CurrentTime:           s.CurrentTime,
					}
					dbAlerts[dk] = &detail

				}
			}
		}
		s.RUnlock()
	}

	// Get the databases to ignore
	ignored, err = backup.GetIgnoredBackups()
	if err != nil {
		WinLogln("Error: getIgnoredBackups: ", err)
	}
	for _, v := range ignored {

		// check for a blank line
		if len(v) == 1 && strings.TrimSpace(v[0]) == "" {
			continue
		}

		// if we don't have 2 or 3 columns, skip it
		if len(v) < 2 || len(v) > 3 {
			//msg := fmt.Sprintln
			WinLogln(fmt.Sprintln("Invalid 'Ignored Database Entry' in file: ", v))
			continue
		}

		var database string
		domain := strings.ToUpper(v[0])
		server := strings.ToUpper(v[1])
		if len(v) == 3 {
			database = v[2]
		}

		// delete one database
		if database != "" {
			dbk := dbKey{
				Domain:       strings.ToUpper(domain),
				ServerName:   strings.ToUpper(server),
				DatabaseName: strings.ToUpper(database),
			}
			delete(dbAlerts, dbk)
		} else { // we are ignoring an entire server
			for k2 := range dbAlerts {
				if k2.Domain == domain && k2.ServerName == server {
					delete(dbAlerts, k2)
				}
			}
		}
	}

	////////////////////////////////////////////////////
	// Set up the summary lines
	////////////////////////////////////////////////////
	type serverSummary struct {
		Domain          string
		ServerName      string
		Count           int
		DataSizeKB      int64
		LogSizeKB       int64
		OldestBackup    time.Time
		OldestLogBackup time.Time
		ServerMapKey    string
		CurrentTime     time.Time
	}

	instanceSummary := make(map[instanceKey]serverSummary)
	for _, v := range dbAlerts {
		ik := instanceKey{
			Domain:     v.Domain,
			ServerName: v.ServerName,
		}

		// handle the instance instanceSummary
		summary, ok := instanceSummary[ik]

		// if it doesn't exist, add it
		if !ok {
			j := serverSummary{
				Domain:          v.Domain,
				ServerName:      v.ServerName,
				Count:           1,
				DataSizeKB:      v.DataSizeKB,
				LogSizeKB:       v.LogSizeKB,
				OldestBackup:    v.LastBackup,
				OldestLogBackup: v.LastLogBackup,
				ServerMapKey:    v.ServerMapKey,
				CurrentTime:     v.CurrentTime,
			}
			instanceSummary[ik] = j
		} else {
			summary.Count++
			summary.DataSizeKB += v.DataSizeKB
			summary.LogSizeKB += v.LogSizeKB
			if v.LastBackup.Before(summary.OldestBackup) {
				summary.OldestBackup = v.LastBackup
			}
			if v.LastLogBackup.Before(summary.OldestLogBackup) {
				summary.OldestLogBackup = v.LastLogBackup
			}
			instanceSummary[ik] = summary
		}

	}

	var ignoredBackupLines []string
	for i := range ignored {
		ignoredBackupLines = append(ignoredBackupLines, strings.Join(ignored[i], ", "))
	}

	context := struct {
		Context
		InstanceAlerts  map[instanceKey]string
		DatabaseAlerts  map[dbKey]*backupDetail
		InstanceSummary map[instanceKey]serverSummary
		IgnoredBackups  []string
	}{
		Context: Context{
			Title:       htmlTitle,
			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			ErrorList:   getServerErrorList(),
			TagList:     globalTagList.getTags(),
			AppConfig:   getGlobalConfig(),
		},
		InstanceAlerts:  instanceAlerts,
		DatabaseAlerts:  dbAlerts,
		InstanceSummary: instanceSummary,
		IgnoredBackups:  ignoredBackupLines,
	}

	requestURL := req.URL.String()
	if requestURL == "/backups/json" {
		jsonAlerts := make([]serverSummary, 0)
		for _, v := range instanceSummary {
			jsonAlerts = append(jsonAlerts, v)
		}
		js, err := json.Marshal(jsonAlerts)
		if err != nil {
			WinLogln("Error: jsonbackups: ", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
		return
	}

	renderFSDynamic(w, "backups", context)
}
