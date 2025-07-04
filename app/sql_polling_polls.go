package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/scalesql/isitsql/internal/diskio"
	"github.com/scalesql/isitsql/internal/gui"
	"github.com/scalesql/isitsql/internal/hadr"
	"github.com/pkg/errors"
)

func (s *SqlServerWrapper) GetActiveSessions() ([]*ActiveSession, error) {
	s.RLock()
	majorVersion := s.MajorVersion
	s.RUnlock()

	stmt := `
			
				
		; WITH	cteSessions AS (
			SELECT DISTINCT	s.session_id, 
					COALESCE(r.blocking_session_id, 0) AS blocking_session_id,
					COALESCE(r.wait_type, '') AS wait_type
			FROM	sys.dm_exec_sessions s
			LEFT JOIN sys.dm_exec_requests r ON r.session_id = s.session_id
			LEFT JOIN sys.dm_tran_session_transactions t ON t.session_id = s.session_id
			WHERE	1=1
			AND		s.session_id <> @@SPID
			AND		s.is_user_process = 1 
			AND (	r.session_id IS NOT NULL -- it has a request
				OR		r.blocking_session_id IS NOT NULL -- OR a request is blocked
				OR		t.session_id IS NOT NULL -- OR an open transaction
			)
		) 
		, cteHierarchy AS (
			SELECT	s.session_id	AS head_blocker_id
					,s.session_id
					,s.blocking_session_id
					,depth = 0
					,wait_type 
			FROM cteSessions s
			WHERE	COALESCE(s.blocking_session_id, 0) = 0 
			UNION ALL
			SELECT	h.head_blocker_id
					,s.session_id 
					,s.blocking_session_id
					,depth = h.depth+1
					,s.wait_type
			FROM cteSessions s 
			JOIN cteHierarchy h ON h.session_id = s.blocking_session_id
		) 
		, cteBlockDetail AS (
			SELECT 
				CASE WHEN head_blocker_id = session_id THEN 0 ELSE head_blocker_id END AS head_blocker_id 
				,session_id
				,wait_type
				,blocking_session_id
				,depth
				,total_blocked = (SELECT COUNT(*) FROM cteHierarchy WHERE head_blocker_id = h.session_id AND session_id <> h.session_id)
			FROM cteHierarchy h
		) 
		, cteBlockFiltered AS (
			-- include sessions that (1) aren't these wait types or (2) are blocked or blocking
			SELECT * 
			FROM	cteBlockDetail bd
			WHERE	wait_type NOT IN ('WAITFOR', 'SP_SERVER_DIAGNOSTICS_SLEEP')
			OR		EXISTS(SELECT * FROM cteBlockDetail WHERE head_blocker_id = bd.session_id)
			OR		EXISTS(SELECT * FROM cteBlockDetail WHERE blocking_session_id = bd.session_id)
		) 
		SELECT	
			bf.session_id AS session_session_id
			,COALESCE(r.session_id, 0) AS request_session_id
			,COALESCE(r.start_time, s.last_request_start_time) as start_time
			,COALESCE(DATEDIFF(ss, r.start_time, GETDATE()), 0) AS RunTimeSeconds
			,COALESCE(r.[status], '') AS [status]
			,COALESCE(SUBSTRING(st.text, (r.statement_start_offset/2)+1, 
				((CASE r.statement_end_offset
				WHEN -1 THEN DATALENGTH(st.text)
				WHEN 0 THEN DATALENGTH(st.text)
				ELSE r.statement_end_offset
				END - r.statement_start_offset)/2) + 1),
				
				ib.event_info,
				'') AS statement_text
				`
	if majorVersion >= 11 {
		stmt += ",COALESCE(DB_NAME(COALESCE(r.database_id, s.database_id, 0)), '') AS [Database] "
	} else {
		stmt += ",COALESCE(DB_NAME(COALESCE(r.database_id, 0)), '') AS [Database] "
	}
	stmt += `
			
			,COALESCE(r.blocking_session_id, 0) AS blocking_session_id
			,COALESCE(r.wait_type, '') AS wait_type
			,COALESCE(r.wait_time, 0) AS wait_time
			,COALESCE(r.wait_resource, '') AS wait_resource
			,COALESCE(s.host_name, '') AS host_name
			,COALESCE(s.program_name, '') as AppName 
			,COALESCE(s.original_login_name, '') AS original_login_name
			,COALESCE(CAST(r.percent_complete AS INT),0) AS percent_complete
			,COALESCE(r.command, '') as command
			
			,COALESCE(s.open_transaction_count, 0) as open_transaction_count
			,COALESCE(head_blocker_id, 0) AS head_blocker_id
			,COALESCE(total_blocked, 0) AS total_blocked
			,COALESCE(depth, 0) AS depth 
			,COALESCe(bf.blocking_session_id, 0) AS blocker_id
		FROM	cteBlockFiltered bf
		LEFT JOIN sys.dm_exec_sessions s ON s.session_id = bf.session_id
		LEFT JOIN sys.dm_exec_requests r ON r.session_id = s.session_id
		OUTER APPLY sys.dm_exec_sql_text(r.sql_handle) AS st
		CROSS APPLY sys.dm_exec_input_buffer(bf.session_id, null) AS ib
		WHERE	1=1
		AND		s.is_user_process = 1 
		AND		s.session_id <> @@SPID
		ORDER BY COALESCE(total_blocked, 0) DESC, COALESCE(depth, 10000) ASC , r.start_time ASC

	`

	rows, err := s.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make([]*ActiveSession, 0)
	//var sessions []*
	for rows.Next() {
		spid := new(ActiveSession)
		err := rows.Scan(&spid.SessionSessionID, &spid.RequestSessionID,
			&spid.StartTime, &spid.RunTimeSeconds, &spid.Status, &spid.StatementText, &spid.Database,
			&spid.BlockingSessionID, &spid.WaitType, &spid.WaitTime, &spid.WaitResource, &spid.HostName,
			&spid.AppName, &spid.LoginName, &spid.PercentComplete, &spid.Command,
			&spid.OpenTxnCount, &spid.HeadBlockerID, &spid.TotalBlocked, &spid.Depth, &spid.BlockerID)
		if err != nil {
			return nil, err
		}
		spid.RunTimeText = gui.SecondsToShortString(spid.RunTimeSeconds)
		sessions = append(sessions, spid)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return sessions, nil
}

func (s *SqlServerWrapper) getDiskIO() error {
	s.RLock()
	p := s.DiskIO
	db := s.DB
	reset := s.ResetOnThisPoll
	s.RUnlock()

	var io diskio.VirtualFileStats
	var err error
	if io, err = diskio.GetFileStats(db); err != nil {
		return err
	}
	s.Lock()
	defer s.Unlock()

	s.DiskIO = io
	s.DiskIODelta = io.Sub(p, reset)

	return nil
}

func (s *SqlServerWrapper) pollAG() error {
	var err error
	//aglist := make(map[string]*hadr.AG)

	s.RLock()
	db := s.DB
	sn := s.ServerName
	s.RUnlock()

	aglist, err := hadr.GetAGList(db, sn)
	if err != nil {
		WinLogf("%s: %s", sn, errors.Wrap(err, "getaglist"))
		//WinLogln("GetAGList", err)
		return errors.Wrap(err, "hadr.getaglist")
	}

	for k, ag := range aglist {
		ag.PrimaryGUID = s.MapKey // which target sent us this AG
		hadr.PublicAGMap.Set(k, ag)
	}

	err = hadr.SetLatency(db)
	if err != nil {
		WinLogln("SetLantencies", err)
		return errors.Wrap(err, "hadr.setlatencies")
	}

	// poll the AG databases
	// dbmap, err := hadr.GetAGDatabases(db)
	// if err != nil {
	// 	WinLogln("GetAGDatabases", err)
	// 	return errors.Wrap(err, "getagdatabases")
	// }

	// // TODO Assign the latency from the map
	// for i, hadrdb := range dbmap {
	// 	var send, redo int
	// 	if hadrdb.IsPrimary {
	// 		send, redo = hadr.PublicAGMap.GetPrimaryDBLatency(hadrdb.GroupID, hadrdb.GroupDatabaseID)
	// 	} else {
	// 		send, redo = hadr.PublicAGMap.GetSecondaryDBLatency(hadrdb.GroupID, hadrdb.ReplicaID, hadrdb.GroupDatabaseID)
	// 	}

	// 	hadrdb.SendQueueKB = send
	// 	hadrdb.RedoQueueKB = redo
	// 	// if send > 0 || redo > 0 {
	// 	// 	fmt.Printf("%v\n", hadrdb)
	// 	// 	fmt.Println(send, redo)
	// 	// }
	// 	dbmap[i] = hadrdb
	// }

	// s.Lock()
	// for k, v := range s.SqlServer.Databases {
	// 	agdb, ok := dbmap[v.DatabaseID]
	// 	if ok {
	// 		s.SqlServer.Databases[k].IsAG = true
	// 		s.SqlServer.Databases[k].AGState = agdb.State()
	// 		s.SqlServer.Databases[k].GroupDatabaseID = agdb.GroupDatabaseID
	// 		s.SqlServer.Databases[k].SendQueueKB = agdb.SendQueueKB
	// 		s.SqlServer.Databases[k].RedoQueueKB = agdb.RedoQueueKB
	// 		// if agdb.SendQueueKB > 0 || agdb.RedoQueueKB > 0 {
	// 		// 	fmt.Printf("DatabaseID: %s (%s): Assigning Queues\n", k, s.SqlServer.Databases[k].Name)
	// 		// }
	// 	} else {
	// 		s.SqlServer.Databases[k].IsAG = false
	// 		s.SqlServer.Databases[k].AGState = ""
	// 		s.SqlServer.Databases[k].GroupDatabaseID = ""
	// 		s.SqlServer.Databases[k].SendQueueKB = 0
	// 		s.SqlServer.Databases[k].RedoQueueKB = 0
	// 	}
	// }
	// s.Unlock()

	return nil
}

func (s *SqlServerWrapper) getName() error {
	s.RLock()
	db := s.DB
	s.RUnlock()

	// TODO query if this is Azure
	// ServerProperty('EngineEdition')
	// If Azure, set the StartTime to IsItSQL start time
	// Not sure about managed instances
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	row := db.QueryRowContext(ctx, `
		USE [master];
		SELECT 
			@@SERVERNAME AS [ServerName], 
			COALESCE(CAST(DEFAULT_DOMAIN() AS NVARCHAR(256)), '') AS [DomainName],
			StartTime  = (SELECT create_date FROM sys.databases WHERE name = 'tempdb')
			`)

	var sn, dd string
	var st time.Time
	err := row.Scan(&sn, &dd, &st)
	if err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	// if we have a start time and it changed on the same server, we are resetting on this poll
	if !s.StartTime.IsZero() && s.StartTime != st && s.ServerName == sn {
		s.ResetOnThisPoll = true
		m := fmt.Sprintf("%s restarted at %s (server time zone) [key: %s]", sn, st.Format("2006-01-02 3:04:05 PM"), s.MapKey)
		WinLogln(m)
	}

	// if we have an instance name and it changed, we are resetting on this poll
	if s.ServerName != "" && s.ServerName != sn {
		s.ResetOnThisPoll = true
		m := fmt.Sprintf("Server %s changed from %s to %s (%s)", s.DisplayName(), s.ServerName, sn, s.MapKey)
		WinLogln(m)
	}

	s.ServerName = sn
	s.Domain = dd
	s.StartTime = st
	s.LastPollTime = time.Now()

	return nil
}

func (s *SqlServerWrapper) getConnectionInfo() error {
	s.RLock()
	db := s.DB
	s.RUnlock()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	start := time.Now()
	row := db.QueryRowContext(ctx, `
			SELECT	TOP 1 
				c.session_id				AS SessionID, 
				c.net_transport				AS NetTransport, 
				c.auth_scheme				AS AuthScheme,  
				s.login_name				AS LoginName 
		FROM	sys.dm_exec_connections c
		JOIN	sys.dm_exec_sessions s ON s.session_id = c.session_id
		WHERE	c.session_id = @@SPID
	`)
	var id int16
	var transport, auth, login string
	err := row.Scan(&id, &transport, &auth, &login)
	if err != nil {
		return err
	}
	latency := time.Since(start)

	s.Lock()
	defer s.Unlock()
	s.Connection.Latency = latency
	s.Connection.SessionID = id
	s.Connection.NetTransport = transport
	s.Connection.AuthScheme = auth
	s.Connection.LoginName = login

	return nil
}

func (s *SqlServerWrapper) getServerInfo() error {

	s.RLock()
	db := s.DB
	s.RUnlock()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	row := db.QueryRowContext(ctx, `
    select 
		cpu_count
		,CAST(SERVERPROPERTY('ProductLevel') AS NVARCHAR(128)) AS ProductLevel
		,CAST(SERVERPROPERTY('ProductVersion') AS NVARCHAR(128)) AS ProductVersion
		,CAST(SERVERPROPERTY('Edition') AS NVARCHAR(128)) AS ProductEdition 
		,CurrentTime = GETDATE() 
		,CAST(SERVERPROPERTY('ComputerNamePhysicalNetBIOS') AS NVARCHAR(128)) AS PhysicalName
		,CAST(COALESCE(SERVERPROPERTY('EditionID'), 0) AS BIGINT) AS EditionID 
		,CAST(COALESCE(SERVERPROPERTY('ProductUpdateLevel'), '') AS NVARCHAR(128)) AS ProductUpdateLevel  
    from sys.dm_os_sys_info `)

	var cc int
	var pl, pv, pe, pn, pul string
	var ct time.Time
	var editionID int64

	err := row.Scan(&cc,
		&pl,
		&pv,
		&pe,
		&ct,
		&pn,
		&editionID,
		&pul)
	if err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	s.CpuCount = cc
	s.PhysicalName = pn
	s.ProductLevel = pl
	s.ProductUpdateLevel = pul
	s.ProductVersion = pv
	s.VersionString = s.ProductVersionString(pv)
	s.ProductEdition = pe
	s.EditionID = editionID
	s.CurrentTime = ct
	s.LastPollTime = time.Now()

	// Set the major version
	s.MajorVersion, err = strconv.Atoi(strings.Split(pv, ".")[0])
	if err != nil {
		s.MajorVersion = 0
	}

	return nil
}

// GetServerMemory gets the RAM available and used
func (s *SqlServerWrapper) GetServerMemory() error {

	s.RLock()
	db := s.DB
	majorVersion := s.MajorVersion
	s.RUnlock()

	var am, pm, sm, max int64
	var memstate string

	// Check for SQL Server 2005
	if majorVersion == 9 {
		row := db.QueryRow("SELECT physical_memory_in_bytes / 1024  FROM sys.dm_os_sys_info; ")

		err := row.Scan(&pm)
		if err != nil {
			return errors.Wrap(err, "sql9: usedmemory")
		}

		row = db.QueryRow("SELECT cntr_value FROM sys.dm_os_performance_counters WHERE counter_name IN ('Total Server Memory (KB)'); ")

		err = row.Scan(&sm)
		if err != nil {
			return errors.Wrap(err, "sql9: totalmemory")
		}

	} else {
		row := db.QueryRow("select available_physical_memory_kb, total_physical_memory_kb, system_memory_state_desc  from sys.dm_os_sys_memory; ")

		err := row.Scan(&am, &pm, &memstate)
		if err != nil {
			return errors.Wrap(err, "sql10: totalmemory")
		}

		row = db.QueryRow("select physical_memory_in_use_kb from sys.dm_os_process_memory; ")

		err = row.Scan(&sm)
		if err != nil {
			return errors.Wrap(err, "sql10: usedmemory")
		}

		row = db.QueryRow("SELECT CAST(value_in_use AS BIGINT) AS max_memory FROM sys.configurations WHERE [name] = 'max server memory (MB)'")
		err = row.Scan(&max)
		if err != nil {
			return errors.Wrap(err, "sql10: maxmemory")
		}
	}

	s.Lock()
	defer s.Unlock()

	s.AvailableMemoryKB = am
	s.PhysicalMemoryKB = pm
	s.SqlServerMemoryKB = sm
	s.MemoryStateDesc = memstate
	// max memory is in MB so convert to KB
	s.MaxMemoryKB = max * 1024
	s.LastPollTime = time.Now()

	return nil
}

// func (s *SqlServer) setPreferredBackup() error {
// 	var err error
// 	var excluded []string

// 	rows, err := s.DB.Query(`
// 		SELECT	[name]
// 		from	sys.databases
// 		WHERE	sys.fn_hadr_backup_is_preferred_replica([name]) = 0
// 	`)
// 	if err != nil {
// 		return errors.Wrap(err, "open")
// 	}
// 	defer rows.Close()

// 	// load the
// 	for rows.Next() {
// 		var db string
// 		err = rows.Scan(&db)
// 		if err != nil {
// 			return errors.Wrap(err, "scan")
// 		}
// 		excluded = append(excluded, db)
// 	}

// 	// Flag the appropriate databases
// 	s.Lock()
// 	defer s.Unlock()
// 	for _, v := range excluded {
// 		_, ok := s.Databases[v]
// 		if ok {
// 			s.Databases[v].IsPreferredBackup = false
// 		}
// 	}

// 	return nil
// }

// // GetCPUUsage populates the CPU usage from the server
// func (s *SqlServerWrapper) GetCPUUsage() error {

// 	s.RLock()
// 	connType := s.ConnectionType

// 	db := s.DB
// 	s.RUnlock()
// 	var stmt string

// 	if connType == "odbc" {
// 		stmt = `
// 			declare @ts_now bigint
// 			select @ts_now = ms_ticks from sys.dm_os_sys_info

// 			select	dateadd (ms, (y.[timestamp] -@ts_now), GETDATE()) as EventTime,
// 					SQLProcessUtilization,
// 					100 - SystemIdle - SQLProcessUtilization as OtherProcessUtilization
// 					,[timestamp]  - @ts_now AS MillisecondsAgo
// 			from (
// 				select
// 				record.value('(./Record/@id)[1]', 'int') as record_id,
// 				record.value('(./Record/SchedulerMonitorEvent/SystemHealth/SystemIdle)[1]', 'int')
// 				as SystemIdle,
// 				record.value('(./Record/SchedulerMonitorEvent/SystemHealth/ProcessUtilization)[1]',
// 				'int') as SQLProcessUtilization,
// 				[timestamp]
// 				from (
// 					select TOP 60 timestamp, convert(xml, record) as record
// 					from sys.dm_os_ring_buffers
// 					where ring_buffer_type = N'RING_BUFFER_SCHEDULER_MONITOR'
// 					and record like '%<SystemHealth>%'
// 					ORDER BY [timestamp] DESC
// 					) as x
// 			) as y
// 			where dateadd (ms, (y.[timestamp] -@ts_now), GETDATE()) > DATEADD(MINUTE, -65, GETDATE())
// 			order by [timestamp] asc`
// 	} else {
// 		stmt = `
// 				declare @ts_now bigint
// 				select @ts_now = ms_ticks from sys.dm_os_sys_info

// 				select	dateadd (ms, (y.[timestamp] -@ts_now), SYSDATETIMEOFFSET()) as EventTime,
// 						SQLProcessUtilization,
// 						100 - SystemIdle - SQLProcessUtilization as OtherProcessUtilization
// 						,[timestamp]  - @ts_now AS MillisecondsAgo
// 				from (
// 					select
// 					record.value('(./Record/@id)[1]', 'int') as record_id,
// 					record.value('(./Record/SchedulerMonitorEvent/SystemHealth/SystemIdle)[1]', 'int')
// 					as SystemIdle,
// 					record.value('(./Record/SchedulerMonitorEvent/SystemHealth/ProcessUtilization)[1]',
// 					'int') as SQLProcessUtilization,
// 					timestamp
// 					from (
// 						select TOP 60 timestamp, convert(xml, record) as record
// 						from sys.dm_os_ring_buffers
// 						where ring_buffer_type = N'RING_BUFFER_SCHEDULER_MONITOR'
// 						and record like '%<SystemHealth>%'
// 						ORDER BY timestamp DESC
// 						) as x
// 				) as y
// 				where dateadd (ms, (y.[timestamp] -@ts_now), SYSDATETIMEOFFSET()) > DATEADD(MINUTE, -65, SYSDATETIMEOFFSET())
// 				order by timestamp asc`
// 	}

// 	// db, err := sql.Open(s.ConnectionType, s.ConnectionString)
// 	// db.SetConnMaxLifetime(30 * time.Second)
// 	// if err != nil {
// 	// 	return err
// 	// }
// 	// defer db.Close()

// 	rows, err := db.Query(stmt)
// 	if err != nil {
// 		return errors.Wrap(err, "open")
// 	}

// 	defer rows.Close()

// 	var a [METRIC_ARRAY_SIZE]Cpu
// 	var lastCPU, lastSQLCPU int
// 	i := 0
// 	now := time.Now()
// 	for rows.Next() {
// 		var c Cpu
// 		var millisecondsAgo int64
// 		err = rows.Scan(&c.EventTime, &c.SqlCpu, &c.OtherCpu, &millisecondsAgo)
// 		if err != nil {
// 			return errors.Wrap(err, "scan")
// 		}

// 		ts := now.Add(time.Millisecond * time.Duration(millisecondsAgo))
// 		c.EventTime = ts

// 		//fmt.Println(i)
// 		a[i] = c
// 		i = i + 1

// 		lastCPU = c.SqlCpu + c.OtherCpu
// 		lastSQLCPU = c.SqlCpu
// 	}

// 	s.Lock()
// 	defer s.Unlock()

// 	s.RecentCpu = a
// 	s.LastPollTime = time.Now()
// 	s.LastCpu = lastCPU
// 	s.LastSQLCPU = lastSQLCPU
// 	s.CoresUsedSQL = float32(s.LastSQLCPU) * float32(s.CpuCount) / 100
// 	s.CoresUsedOther = float32(s.LastCpu-s.LastSQLCPU) * float32(s.CpuCount) / 100

// 	return nil
// }
