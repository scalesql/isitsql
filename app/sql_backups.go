package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/scalesql/isitsql/internal/backup"
	"github.com/scalesql/isitsql/internal/mssql"
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

func (s *SqlServerWrapper) setBackups() error {
	var backupQuery string

	s.RLock()
	majorVersion := s.MajorVersion
	domain := s.Domain
	s.RUnlock()
	if majorVersion >= mssql.SQL2022 {
		backupQuery = `
			-- SQL Server 2022 and higher
			-- Are we connected to the Listener of a CAG?
			-- If so, we will use the CAG name here
			DECLARE @CAG SYSNAME;
			SELECT @CAG = ag.[name]
			FROM sys.dm_exec_sessions s
			JOIN sys.availability_groups ag ON ag.group_id = s.contained_availability_group_id
			WHERE session_id = @@SPID;

			;WITH rawbackups AS ( 
				select 
					bus.server_name
					,bus.database_name
					,bus.backup_start_date
					,bus.type
					,bmf.physical_device_name
				--,* 
				from msdb.dbo.backupset bus
				join msdb.dbo.backupmediafamily bmf on bmf.media_set_id = bus.media_set_id
			), fixeddb AS (
				SELECT	COALESCE(@CAG, ag.name, server_name) as host,
						@@SERVERNAME AS instance,
						-- CTE.database_name AS raw_name,
						COALESCE(logdb.physical_database_name, physdb.[physical_database_name]) as [database_name],
						backup_start_date, 
						[type], 
						physical_device_name
				FROM	rawbackups
				-- JOIN on the physical database name (for master and msdb)
				LEFT JOIN	sys.databases physdb ON physdb.[physical_database_name] = rawbackups.database_name
				-- JOIN on the physical database name (for master and msdb)
				LEFT JOIN   sys.databases logdb ON logdb.[name] = rawbackups.database_name 
				-- In the AG, sys.databases.replica_id will be null, but we have the AG from above
				LEFT JOIN sys.availability_replicas r ON r.replica_id = logdb.replica_id
				LEFT JOIN sys.availability_groups ag on ag.group_id = r.group_id
			), sorted AS (
				SELECT *,
					ROW_NUMBER() OVER(PARTITION BY database_name, type ORDER BY backup_start_date DESC) AS RowNumber
				FROM fixeddb
			)
			SELECT host, [instance], database_name, backup_start_date, [type], physical_device_name
			FROM sorted
			WHERE RowNumber = 1 
			ORDER BY database_name, backup_start_date DESC;

		`

	} else if majorVersion >= mssql.SQL2014 {
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

		// set the map values
		switch strings.ToUpper(backupType) {
		case "L":
			backup.SetLog(domain, host, db, start, instance, device)
		case "I", "D": // differential or full
			backup.SetFull(domain, host, db, start, instance, device)
		}
	}

	s.Lock()
	s.BackupMessage = ""
	s.Unlock()

	//	_ = s.setBackupAlert()

	return nil
}
