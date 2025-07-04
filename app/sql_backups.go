package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/scalesql/isitsql/internal/backup"
	"github.com/pkg/errors"
)

type databaseBackups struct {
	DatabaseName       string    `json:"database_name,omitempty"`
	FullStart          time.Time `json:"full_start,omitempty"`
	FullPhysicalDevice string    `json:"full_physical_device,omitempty"`
	LogStart           time.Time `json:"log_start,omitempty"`
	LogPhysicalDevice  string    `json:"log_physical_device,omitempty"`
}

func (s *SqlServerWrapper) pollBackups() error {
	var rowCount int
	var err error

	s.Lock()
	s.LastBackupPoll = time.Now()
	s.Unlock()

	// if we got an error last time or we have more than 1 million rows, just don't poll anymore
	// if rowCount == -1 || rowCount > 1000000 {
	// 	return nil
	// }

	if rowCount, err = s.setBackupRowCount(); err != nil {
		return err
	}

	if rowCount == -1 || rowCount > 1000000 {
		s.Lock()
		s.BackupMessage = fmt.Sprintf("Too many backup history rows found: %d.  Backup polling disabled for this server.  Please reduce below 1 million rows and restart IsItSQL", rowCount)
		s.Unlock()
		return nil
	}

	err = s.setBackups()
	return err
}

func (s *SqlServerWrapper) setBackupRowCount() (int, error) {

	row := s.DB.QueryRow(`
	
		;WITH CTE AS ( 
		select 
			database_name
			,backup_start_date
			,type
			,ROW_NUMBER() OVER(PARTITION BY database_name, type ORDER BY backup_start_date DESC) AS RowNumber
			,bmf.physical_device_name
		--,* 
		from msdb.dbo.backupset bus
		join msdb.dbo.backupmediafamily bmf on bmf.media_set_id = bus.media_set_id
		)
		SELECT COUNT(*) FROM CTE;
		`)

	var rowCount int
	err := row.Scan(&rowCount)

	s.Lock()
	defer s.Unlock()

	if err != nil {
		s.BackupRowCount = -1
		s.BackupMessage = err.Error()
		return -1, err
	}

	s.BackupRowCount = rowCount
	s.BackupMessage = ""

	return rowCount, nil
}

// func (s *SqlServer) setBackupAlert() error {

// 	setg, err := settings.ReadConfig()
// 	if err != nil {
// 		return errors.Wrap(err, "readconfig")
// 	}

// 	// Update the backup Values
// 	s.Lock()
// 	defer s.Unlock()

// 	instanceName := s.ServerName

// 	// check all the databases against our list of backups
// 	for k, d := range s.Databases {

// 		agname := d.AGName

// 		// Alert until I know better
// 		d.BackupAlert = true

// 		// We don't care about tempdb
// 		if d.Name == "tempdb" {
// 			d.BackupAlert = false
// 			continue
// 		}

// 		// We don't care about restoring or offline databases
// 		if d.StateDesc == "RESTORING" || d.StateDesc == "OFFLINE" {
// 			d.BackupAlert = false
// 			continue
// 		}

// 		// if this isn't the preferred backup for this database we don't care
// 		// I don't care about this any more
// 		// if d.IsPreferredBackup == false {
// 		// 	d.BackupAlert = false
// 		// 	continue
// 		// }

// 		// get the backups for this database
// 		bu, ok := s.Backups[k]
// 		if ok {
// 			d.LastBackup = bu.FullStart
// 			d.LastBackupDevice = bu.FullPhysicalDevice
// 			d.LastBackupInstance = instanceName
// 			d.LastLogBackup = bu.LogStart
// 			d.LastLogBackupDevice = bu.LogPhysicalDevice
// 			d.LastLogBackupInstance = instanceName

// 			// Assume this is a good backup
// 			d.BackupAlert = false
// 		}

// 		// Check for an AG backup
// 		agbkup, found := agbackup.Get(agname, k)
// 		if found {
// 			d.LastBackup = agbkup.FullStarted
// 			d.LastBackupDevice = agbkup.FullDevice
// 			d.LastBackupInstance = agbkup.FullInstance
// 			d.LastLogBackup = agbkup.LogStarted
// 			d.LastLogBackupDevice = agbkup.LogDevice
// 			d.LastLogBackupInstance = agbkup.LogInstance
// 			d.BackupAlert = false
// 		}

// 		if s.CurrentTime.Sub(d.LastBackup).Hours() > float64(setg.BackupAlertHours) {
// 			d.BackupAlert = true
// 		}
// 		if d.RecoveryModelDesc == "FULL" && s.CurrentTime.Sub(d.LastLogBackup).Minutes() > float64(setg.LogBackupAlertMinutes) {
// 			d.BackupAlert = true
// 		}
// 	}
// 	return nil
// }

func (s *SqlServerWrapper) setBackups() error {

	var backupQuery string

	s.RLock()
	majorVersion := s.MajorVersion
	s.RUnlock()

	if majorVersion >= 12 {
		backupQuery = `

		;WITH CTE AS ( 
			select 
				bus.server_name
				,bus.database_name
				,bus.backup_start_date
				,bus.type
				,ROW_NUMBER() OVER(PARTITION BY database_name, type ORDER BY backup_start_date DESC) AS RowNumber
				,bmf.physical_device_name
			--,* 
			from msdb.dbo.backupset bus
			join msdb.dbo.backupmediafamily bmf on bmf.media_set_id = bus.media_set_id
		)
		SELECT	COALESCE(ag.name, server_name) as host,
				@@SERVERNAME AS instance 
				,database_name, backup_start_date, type, physical_device_name
		FROM	CTE
		JOIN	sys.databases d ON d.[name] = CTE.database_name
		LEFT JOIN sys.availability_replicas r ON r.replica_id = d.replica_id
		LEFT JOIN sys.availability_groups ag on ag.group_id = r.group_id
		WHERE RowNumber = 1
		order by database_name, backup_start_date desc 
		
		`

	} else {
		backupQuery = `
	
		;WITH CTE AS ( 
			select 
				bus.server_name 
				,bus.database_name
				,bus.backup_start_date
				,bus.type
				,ROW_NUMBER() OVER(PARTITION BY database_name, type ORDER BY backup_start_date DESC) AS RowNumber
				,bmf.physical_device_name
			--,* 
			from msdb.dbo.backupset bus
			join msdb.dbo.backupmediafamily bmf on bmf.media_set_id = bus.media_set_id
		)
		SELECT server_name as host, 
			@@serverName AS instance
			,database_name, backup_start_date, type, physical_device_name
		FROM CTE    
		WHERE RowNumber = 1
		order by database_name, backup_start_date desc 
		

	`
	}

	rows, err := s.DB.Query(backupQuery)
	if err != nil {
		return errors.Wrap(err, "backup-query")
	}

	defer rows.Close()

	for rows.Next() {
		var host, instance, db, device, backupType string
		var start time.Time

		err := rows.Scan(&host, &instance, &db, &start, &backupType, &device)
		if err != nil {
			return err
		}

		// // add to the map
		// _, ok := b[dbName]
		// if !ok {
		// 	//db =
		// 	b[dbName] = &databaseBackups{
		// 		DatabaseName: dbName,
		// 	}
		// }

		// Check if DB is in AG and save the backup
		// var agname string = ""
		// s.RLock()
		// thisdb, found := s.Databases[dbName]
		// if found {
		// 	agname = thisdb.AGName
		// }
		// instance := s.ServerName
		// s.RUnlock()

		// set the map values
		switch strings.ToUpper(backupType) {
		case "L":
			backup.SetLog(host, db, start, instance, device)
			// b[dbName].LogStart = start
			// b[dbName].LogPhysicalDevice = device
			// if agname != "" {
			// 	agbackup.SetLog(agname, dbName, start, instance, device)
			// }
		case "I", "D": // full or differential
			backup.SetFull(host, db, start, instance, device)
		}
	}

	s.Lock()
	s.BackupMessage = ""
	s.Unlock()

	//	_ = s.setBackupAlert()

	return nil
}
