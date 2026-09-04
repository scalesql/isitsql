package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/scalesql/isitsql/internal/backup"
	"github.com/scalesql/isitsql/internal/mssql"
	"github.com/sirupsen/logrus"
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
		WinLogf("Too many backup rows: %s (%s) rows=%d", s.ServerName, s.FQDN, rowCount)
		s.Lock()
		s.BackupMessage = fmt.Sprintf("Too many backup history rows found: %d.  Backup polling disabled for this server.  Please reduce below 1 million rows and restart IsItSQL", rowCount)
		s.Unlock()
		return nil
	}

	err = s.setBackups()
	return err
}

func (s *SqlServerWrapper) setBackupRowCount() (int, error) {

	row := s.DB.QueryRow("SELECT COUNT(*) FROM msdb.dbo.backupset;")
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
	serverName := s.ServerName
	fqdn := s.FQDN
	// domain := s.Domain
	s.RUnlock()
	if majorVersion >= mssql.SQL2022 {
		backupQuery = `
			SET NOCOUNT ON;
			SET XACT_ABORT ON; 

			DECLARE @agdb TABLE (
				database_id	INT PRIMARY KEY,
				logical_database_name SYSNAME NOT NULL UNIQUE,
				physical_database_name SYSNAME NOT NULL UNIQUE,
				ag_name SYSNAME NULL
			)

			-- poll the AGs once
			INSERT @agdb
			SELECT
				db.database_id, 
				db.[name],
				[physical_database_name] = COALESCE(db.[physical_database_name], db.[name])
				,ag.[name] AS ag_name
			FROM sys.databases db  
			LEFT JOIN sys.availability_databases_cluster adc ON adc.database_name = COALESCE(db.[physical_database_name], db.[name])
			LEFT JOIN sys.availability_groups ag ON ag.group_id = adc.group_id
			WHERE db.source_database_id IS NULL;

			-- Prep for incremental polls
			-- DECLARE @backup_set_id INT = 0;
			-- SELECT @backup_set_id = MAX(backup_set_id) - 100 FROM msdb.dbo.backupset

			DECLARE @recent TABLE (
				backup_set_id INT PRIMARY KEY
			)

			-- The most effecient scan for recent backups is without any joins
			;WITH cte_base AS (
				SELECT 
					backup_set_id,
					ROW_NUMBER() OVER(PARTITION BY database_name, type ORDER BY backup_finish_date DESC) AS  RowNumber
				FROM msdb.dbo.backupset bus 
				-- WHERE bus.backup_set_id > @backup_set_id 
			)
			INSERT @recent
			SELECT backup_set_id
			FROM cte_base 
			WHERE RowNumber = 1;


			;WITH cte_all AS (
				SELECT 
					bus.backup_set_id,
					bus.server_name , 
					bus.database_name, 
					bus.backup_start_date , 
					bus.type, 
					bus.media_set_id,
					-- Only get the first backup file for now
					physical_device_name = COALESCE((SELECT TOP 1 physical_device_name FROM msdb.dbo.backupmediafamily where media_set_id = bus.media_set_id ORDER BY physical_device_name), '(Unknown)'),
					ag.ag_name
				FROM @recent r
				JOIN msdb.dbo.backupset bus ON bus.backup_set_id = r.backup_set_id
				LEFT JOIN @agdb ag ON COALESCE(ag.physical_database_name, ag.logical_database_name) = bus.database_name	
			), 
			cte_keys AS (
				SELECT 
					server_name,
					database_name, 
					backup_start_date,
					[type],
					physical_device_name,
					-- COALESCE(DEFAULT_DOMAIN(), 'AG$$' + @CAG, '(NONE)') AS backup_key_domain , 
					-- COALESCE(@CAG, ag_name, @@SERVERNAME) AS backup_key_container ,
					COALESCE(DEFAULT_DOMAIN(), ag_name, '(NONE)') AS backup_key_domain , 
					COALESCE(ag_name, server_name, @@SERVERNAME) AS backup_key_container ,
					[database_name] as backup_key_database  
				FROM cte_all
			)
			SELECT * 
			FROM cte_keys
			-- ORDER BY backup_key_domain, backup_key_container, backup_key_database, [type]


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
				from msdb.dbo.backupset bus
				join msdb.dbo.backupmediafamily bmf on bmf.media_set_id = bus.media_set_id
			)
			SELECT
				@@SERVERNAME AS instance,
				database_name, 
				backup_start_date, 
				type, 
				physical_device_name,
				COALESCE(DEFAULT_DOMAIN(), '(NONE)') AS backup_key_domain,
				COALESCE(ag.name, server_name) as backup_key_container,
				database_name AS backup_key_database
			FROM	CTE
			JOIN	sys.databases d ON d.[name] = CTE.database_name
			LEFT JOIN sys.availability_replicas r ON r.replica_id = d.replica_id
			LEFT JOIN sys.availability_groups ag on ag.group_id = r.group_id
			WHERE RowNumber = 1
			order by database_name, backup_start_date DESC;
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
				from msdb.dbo.backupset bus
				join msdb.dbo.backupmediafamily bmf on bmf.media_set_id = bus.media_set_id
			)
			SELECT 
				@@SERVERNAME AS instance,
				database_name, 
				backup_start_date, 
				type, 
				physical_device_name,
				COALESCE(DEFAULT_DOMAIN(), '(NONE)') AS backup_key_domain,
				server_name AS backup_key_container,
				database_name AS backup_key_database
			FROM CTE    
			WHERE RowNumber = 1
			order by database_name, backup_start_date DESC; 
	`
	}

	start := time.Now()
	rows, err := s.DB.Query(backupQuery)
	if err != nil {
		return errors.Wrap(err, "backup-query")
	}

	defer rows.Close()
	for rows.Next() {
		var keyDomain, keyContainer, keyDatabase string
		var instance, db, device, backupType string
		var start time.Time

		err := rows.Scan(&instance, &db, &start, &backupType, &device, &keyDomain, &keyContainer, &keyDatabase)
		if err != nil {
			return err
		}

		// set the map values
		switch strings.ToUpper(backupType) {
		case "L":
			backup.SetLog(keyDomain, keyContainer, keyDatabase, start, instance, device)
		case "I", "D": // differential or full
			backup.SetFull(keyDomain, keyContainer, keyDatabase, start, instance, device)
		}
	}
	if time.Since(start) > time.Duration(5*time.Second) {
		logrus.Debugf("long backup poll: %s (%s) duration=%s", serverName, fqdn, time.Since(start))
	}

	s.Lock()
	s.BackupMessage = ""
	s.Unlock()

	//	_ = s.setBackupAlert()

	return nil
}
