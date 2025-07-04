package app

import (
	"strconv"
	"strings"
	"time"

	"github.com/scalesql/isitsql/internal/backup"
	"github.com/scalesql/isitsql/internal/hadr"
	"github.com/pkg/errors"
	"golang.org/x/exp/slices"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func (s *SqlServerWrapper) getDatabases() error {
	var err error
	dbs := make(map[int]*Database)
	status := make(map[string]int)

	cfg := getGlobalConfig()

	s.RLock()
	ServerMapKey := s.MapKey
	ServerName := s.ServerName
	ServerVersion := s.MajorVersion
	currentTime := s.CurrentTime
	s.RUnlock()

	tempdbdata, tempdblog, err := s.getTempDBSize()
	if err != nil {
		return errors.Wrap(err, "gettempdbsize")
	}
	var dbQuery string

	// If we should query for AG, otherwise, this instance is the host
	if ServerVersion >= 12 {
		dbQuery = `

		SELECT	d.database_id, d.[name]
			,sizes.DataSizeKB
			,sizes.LogSizeKB
			,state_desc
			,recovery_model_desc
			,COALESCE(compatibility_level, 0)
			,COALESCE(collation_name, 'Unknown')
			,COALESCE(CAST(m.mirroring_guid AS NVARCHAR(128)), '') As MirrorGuid
			,CAST (
				CASE WHEN m.mirroring_guid IS NOT NULL THEN 1 ELSE 0 END
						AS bit) AS IsMirrored
			,COALESCE(CAST(CASE WHEN m.mirroring_role = 1 THEN 1 ELSE 0 END AS BIT), 0) AS IsPrincipal
			,COALESCE(mirroring_state, -1) AS MirrorState 
			,COALESCE(mirroring_state_desc, '') AS MirrorStateDesc
			,COALESCE(mirroring_role, -1) AS MirrorRole
			,COALESCE(mirroring_role_desc, '') AS MirrorRoleDesc
			,COALESCE(mirroring_safety_level, -1) AS MirrorSafety
			,COALESCE(mirroring_safety_level_desc, '') AS MirrorSafetyDesc
			,COALESCE(mirroring_partner_name, '') AS MirrorPartner
			,COALESCE(mirroring_witness_name, '') AS MirrorWitness
			,COALESCE(mirroring_witness_state, -1) AS MirrorWitnessState
			,COALESCE(mirroring_witness_state_desc, '') AS MirrorWitnessStateDesc
			,COALESCE((SELECT	cntr_value
							FROM	sys.dm_os_performance_counters
		--					FROM	tempdb.dbo.dm_os_performance_counters
							WHERE	counter_name = 'Log Send Queue KB'
							and		instance_name = d.[name]),0) AS MirrorSendQueue 
			,COALESCE((SELECT	cntr_value
					FROM	sys.dm_os_performance_counters
		--			FROM	tempdb.dbo.dm_os_performance_counters
					WHERE	counter_name = 'Redo Queue KB'
					and		instance_name = d.[name]),0) AS MirrorRedoQueue
			,d.create_date
			,COALESCE(ag.[name], @@SERVERNAME) AS [HostName]
			,d.is_read_only
		FROM	master.sys.databases d
		JOIN (
				SELECT 
					database_id,
					DataSizeKB = CAST(SUM(CASE WHEN type_desc <> 'LOG' THEN CAST(size AS BIGINT) ELSE 0 END ) * 8 AS BIGINT),
					LogSizeKB = CAST(SUM(CASE WHEN type_desc = 'LOG' THEN CAST(size AS BIGINT) ELSE 0 END ) * 8 AS BIGINT)
				FROM master.sys.master_files
				GROUP BY database_id) AS sizes
			ON sizes.database_id = d.database_id
		LEFT JOIN master.sys.database_mirroring m ON m.database_id = d.database_id
		--	LEFT JOIN tempdb.dbo.database_mirroring m ON m.database_id = d.database_id
		LEFT JOIN sys.availability_replicas r ON r.replica_id = d.replica_id
		LEFT JOIN sys.availability_groups ag on ag.group_id = r.group_id
		WHERE d.source_database_id IS NULL 
		

		`

	} else {
		dbQuery = `
		
		SELECT	d.database_id, d.[name]
			,sizes.DataSizeKB
			,sizes.LogSizeKB
			,state_desc
			,recovery_model_desc
			,COALESCE(compatibility_level, 0)
			,COALESCE(collation_name, 'Unknown')
			,COALESCE(CAST(m.mirroring_guid AS NVARCHAR(128)), '') As MirrorGuid
			,CAST (
				CASE WHEN m.mirroring_guid IS NOT NULL THEN 1 ELSE 0 END
						AS bit) AS IsMirrored
			,COALESCE(CAST(CASE WHEN m.mirroring_role = 1 THEN 1 ELSE 0 END AS BIT), 0) AS IsPrincipal
			,COALESCE(mirroring_state, -1) AS MirrorState 
			,COALESCE(mirroring_state_desc, '') AS MirrorStateDesc
			,COALESCE(mirroring_role, -1) AS MirrorRole
			,COALESCE(mirroring_role_desc, '') AS MirrorRoleDesc
			,COALESCE(mirroring_safety_level, -1) AS MirrorSafety
			,COALESCE(mirroring_safety_level_desc, '') AS MirrorSafetyDesc
			,COALESCE(mirroring_partner_name, '') AS MirrorPartner
			,COALESCE(mirroring_witness_name, '') AS MirrorWitness
			,COALESCE(mirroring_witness_state, -1) AS MirrorWitnessState
			,COALESCE(mirroring_witness_state_desc, '') AS MirrorWitnessStateDesc
			,COALESCE((SELECT	cntr_value
							FROM	sys.dm_os_performance_counters
		--					FROM	tempdb.dbo.dm_os_performance_counters
							WHERE	counter_name = 'Log Send Queue KB'
							and		instance_name = d.[name]),0) AS MirrorSendQueue 
			,COALESCE((SELECT	cntr_value
					FROM	sys.dm_os_performance_counters
		--			FROM	tempdb.dbo.dm_os_performance_counters
					WHERE	counter_name = 'Redo Queue KB'
					and		instance_name = d.[name]),0) AS MirrorRedoQueue
			,d.create_date
			,@@SERVERNAME AS [HostName]
			,d.is_read_only
		FROM	master.sys.databases d
		JOIN (
				SELECT 
					database_id,
					DataSizeKB = CAST(SUM(CASE WHEN type_desc <> 'LOG' THEN CAST(size AS BIGINT) ELSE 0 END ) * 8 AS BIGINT),
					LogSizeKB = CAST(SUM(CASE WHEN type_desc = 'LOG' THEN CAST(size AS BIGINT) ELSE 0 END ) * 8 AS BIGINT)
				FROM master.sys.master_files
				GROUP BY database_id) AS sizes
			ON sizes.database_id = d.database_id
		LEFT JOIN master.sys.database_mirroring m ON m.database_id = d.database_id
		--	LEFT JOIN tempdb.dbo.database_mirroring m ON m.database_id = d.database_id
		WHERE d.source_database_id IS NULL 
		
		`
	}

	// TODO this query can generate values greater than INT
	// TODO need to trap this error
	rows, err := s.DB.Query(dbQuery)
	if err != nil {
		return errors.Wrap(err, "query")
	}

	defer rows.Close()

	var d, l int64
	var c int

	//var mirrorGUID string
	titleCaser := cases.Title(language.AmericanEnglish, cases.NoLower)
	for rows.Next() {
		var db Database
		var dbm mirroredDatabase
		var priority int

		dbm.MapKey = ServerMapKey
		dbm.ServerName = ServerName

		err := rows.Scan(&db.DatabaseID, &db.Name, &db.DataSizeKB, &db.LogSizeKB, &db.StateDesc, &db.RecoveryModelDesc,
			&db.CompatibilityLevel, &db.Collation,
			&dbm.MirrorGUID, &dbm.IsMirrored, &dbm.IsPrincipal,
			&dbm.MirrorState, &dbm.MirrorStateDesc,
			&dbm.MirrorRole, &dbm.MirrorRoleDesc,
			&dbm.MirrorSafety, &dbm.MirrorSafetyDesc,
			&dbm.MirrorPartner,
			&dbm.MirrorWitness,
			&dbm.MirrorWitnessState, &dbm.MirrorWitnessStateDesc,
			&dbm.MirrorSendQueue, &dbm.MirrorRedoQueue,
			&db.CreateDate, &db.Host, &db.IsReadOnly)
		if err != nil {
			return errors.Wrap(err, "scan")
		}

		// if tempdb, use the values from above
		if db.DatabaseID == 2 {
			db.DataSizeKB = int64(tempdbdata)
			db.LogSizeKB = int64(tempdblog)
		}

		dbm.MirrorStateDesc = titleCaser.String(dbm.MirrorStateDesc)
		dbm.MirrorRoleDesc = titleCaser.String(dbm.MirrorRoleDesc)
		dbm.MirrorSafetyDesc = titleCaser.String(dbm.MirrorSafetyDesc)
		dbm.MirrorWitnessStateDesc = titleCaser.String(dbm.MirrorWitnessStateDesc)

		// Set the priority
		if dbm.MirrorSendQueue > 0 {
			priority++
		}
		if dbm.MirrorRedoQueue > 0 {
			priority++
		}

		// synced is ok
		// syncing gets + 1
		// anythign else gets + 10 or some such -- use a switch maybe
		if dbm.MirrorState != 4 && dbm.MirrorState != -1 {
			priority++
		}
		if dbm.MirrorSafety == -1 {
			priority++
		}
		if dbm.MirrorWitnessState != -1 && dbm.MirrorWitnessState != 1 && dbm.MirrorWitness != "" {
			priority++
		}

		/*

			Sort by disconnected (or other), synchronizing, connected,
			Send + Redo queue,
			database + log size

			then assign a priority based on this ranking

		*/

		dbm.DatabaseName = db.Name
		dbm.Priority = priority

		db.Mirroring = dbm
		db.IsMirrored = dbm.IsMirrored

		d += db.DataSizeKB
		l += db.LogSizeKB
		c++

		if cd, ok := status[db.StateDesc]; ok {
			status[db.StateDesc] = cd + 1
		} else {
			status[db.StateDesc] = 1
		}

		// Get any backups for this database
		b, found := backup.Get(db.Host, db.Name)
		if found {
			db.LastBackup = b.FullStarted
			db.LastBackupDevice = b.FullDevice
			db.LastBackupInstance = b.FullInstance
			db.LastLogBackup = b.LogStarted
			db.LastLogBackupDevice = b.LogDevice
			db.LastLogBackupInstance = b.LogInstance
		}

		// Set backup alerts
		db.BackupAlert = true
		if db.Name == "tempdb" || db.StateDesc == "RESTORING" || db.StateDesc == "OFFLINE" {
			db.BackupAlert = false
		} else {
			// Need log backups
			if db.RecoveryModelDesc == "FULL" || db.RecoveryModelDesc == "BULK_LOGGED" {
				if currentTime.Sub(db.LastBackup).Hours() < float64(cfg.FullBackupAlertHours) &&
					currentTime.Sub(db.LastLogBackup).Minutes() < float64(cfg.LogBackupAlertMinutes) {
					db.BackupAlert = false
				}
			} else { // only need full backups
				if currentTime.Sub(db.LastBackup).Hours() < float64(cfg.FullBackupAlertHours) {
					db.BackupAlert = false
				}
			}
		}
		if s.IgnoreBackups {
			db.BackupAlert = false
		}
		if slices.Contains(s.IgnoreBackupsList, strings.ToLower(db.Name)) {
			db.BackupAlert = false
		}
		//println(s.FQDN, s.IgnoreBackups, s.IgnoreBackupsList, db.Name, db.BackupAlert)

		dbs[db.DatabaseID] = &db
	}

	if ServerVersion >= 12 {
		// Get the AG stuff
		agdatabases, err := hadr.GetReplicaDatabases(s.DB)
		if err != nil {
			return errors.Wrap(err, "hadr.getreplicadatabases")
		}

		// push the agdatabases into the master database list
		for i, agdb := range agdatabases {
			// Get the latency from the map
			var send, redo int
			if agdb.IsPrimary {
				send, redo = hadr.PublicAGMap.GetPrimaryDBLatency(agdb.GroupID, agdb.GroupDatabaseID)
			} else {
				send, redo = hadr.PublicAGMap.GetSecondaryDBLatency(agdb.GroupID, agdb.ReplicaID, agdb.GroupDatabaseID)
			}
			agdb.SendQueueKB = send
			agdb.RedoQueueKB = redo

			// Assign to the map we are building
			_, ok := dbs[i]
			if ok {
				dbs[i].IsAG = true
				dbs[i].AGState = agdb.State()
				dbs[i].AGDB = agdb
			}
		}
	}

	var summary []string
	for k, v := range status {
		s := strconv.Itoa(v) + " " + titleCaser.String(k)
		summary = append(summary, s)
	}

	s.Lock()
	s.Databases = dbs
	s.DatabaseCount = c
	s.DataSizeKB = d
	s.LogSizeKB = l
	s.LastPollTime = time.Now()
	s.DatabaseStateSummary = strings.Join(summary, ",")
	s.Unlock()

	return nil
}

// getTempDBSize returns the size of tempdb using the data and log files in the database
// for a more accurate value
func (s *SqlServerWrapper) getTempDBSize() (dataKB int, logKB int, err error) {
	query := `
		SELECT 
			DataSizeKB = CAST(SUM(CASE WHEN type_desc <> 'LOG' THEN CAST(size AS BIGINT) ELSE 0 END ) * 8 AS BIGINT),
			LogSizeKB = CAST(SUM(CASE WHEN type_desc = 'LOG' THEN CAST(size AS BIGINT) ELSE 0 END ) * 8 AS BIGINT)
		FROM tempdb.sys.database_files;
	`
	rows, err := s.DB.Query(query)
	if err != nil {
		return 0, 0, errors.Wrap(err, "query")
	}
	defer rows.Close()
	rows.Next()
	err = rows.Scan(&dataKB, &logKB)
	if err != nil {
		return 0, 0, errors.Wrap(err, "scan")
	}
	err = rows.Err()
	if err != nil {
		return 0, 0, errors.Wrap(err, "rows.err")
	}
	return
}
