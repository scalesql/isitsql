package session

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

const maxDepth = 100 // max recursion depth to find blocking

// TODO print a message if we hit the max recursion depth

// Session holds a SQL Server session that may also have a Request and/or open transactions.
type Session struct {
	SessionID       int16     `db:"session_id"`
	RequestID       int32     `db:"request_id"`
	HasRequest      bool      `db:"has_request"`
	StartTime       time.Time `db:"start_time"`
	RunTimeSeconds  int       `db:"RunTimeSeconds"`
	RunTimeText     string
	Status          string `db:"status"`
	StatementText   string `db:"statement_text"`
	Database        string `db:"database"`
	WaitType        string `db:"wait_type"`
	WaitTime        int    `db:"wait_time"`
	WaitResource    string `db:"wait_resource"`
	HostName        string `db:"host_name"`
	AppName         string `db:"AppName"`
	LoginName       string `db:"original_login_name"`
	PercentComplete int    `db:"percent_complete"`
	Command         string `db:"command"`
	OpenTxnCount    int    `db:"open_transaction_count"`

	BlockerID     int16 `db:"blocking_session_id"`
	HeadBlockerID int16
	TotalBlocked  int
	Depth         int
	Path          string
}

// func dumpSessions(ss []Session) {
// 	sort.SliceStable(ss, func(i, j int) bool {
// 		return ss[i].SessionID < ss[j].SessionID
// 	})
// 	for _, s := range ss {
// 		fmt.Printf("id=%-2d  blocked_id=%-2d  total_blocked=%-2d path=%s\n", s.SessionID, s.BlockerID, s.TotalBlocked, s.Path)
// 	}
// }

// populateBlocking takes an array of sessions, populates the headblocker and the total blocked sessions
func populateBlocking(ss []Session) error {
	// populate the map tree of blockers -- used for head blockers
	m := make(map[int16]int16) // session -> blocked by
	for _, s := range ss {
		if s.BlockerID != 0 {
			m[s.SessionID] = s.BlockerID
		}
	}
	totalBlockMap := make(map[int16]int)

	// 	// populate the map of head blockers
	headmap := make(map[int16]int) // session_id -> count of blocked sessions
	for i, s := range ss {
		id, parents, path, err := headBlocker(s.SessionID, m, []int16{}, 0, "")
		if err != nil {
			return err
		}
		ss[i].Path = path + "/"
		if id != s.SessionID { // if we came back with a different session_id
			ss[i].HeadBlockerID = id
			headmap[id] += 1
		}

		// go through the parents and increase their blocking count
		for _, v := range parents {
			if v == s.SessionID {
				continue
			}
			_, exists := totalBlockMap[v]
			if exists {
				totalBlockMap[v] += 1
			} else {
				totalBlockMap[v] = 1
			}
		}
	}

	// put the totalBlockMap values back into the array
	for i, s := range ss {
		totalBlocked, found := totalBlockMap[s.SessionID]
		if found {
			ss[i].TotalBlocked = totalBlocked
		}
	}
	return nil
}

// querySessions returns an array of Session objects
func querySessions(ctx context.Context, db *sql.DB, majorVersion int) ([]Session, error) {
	dbx := sqlx.NewDb(db, "mssql")
	var stmt string
	if majorVersion >= 11 {
		stmt = sessionQuery
	} else {
		stmt = SessionQueryLegacy
	}
	sessions := make([]Session, 0)
	err := dbx.SelectContext(ctx, &sessions, stmt)
	if err != nil {
		return []Session{}, err
	}
	return sessions, nil
}

// Get the active sessions on a database server with blocking
func Get(ctx context.Context, db *sql.DB, majorVersion int) ([]Session, error) {
	sessions, err := querySessions(ctx, db, majorVersion)
	if err != nil {
		return []Session{}, errors.Wrap(err, "querySessions")
	}
	for i, s := range sessions {
		sessions[i].RunTimeText = secondsToShortString(s.RunTimeSeconds)
	}

	err = populateBlocking(sessions)
	if err != nil {
		logrus.Error(err)
	}

	// filter...
	final := make([]Session, 0, len(sessions))
	for _, s := range sessions {
		// exclude these two if they have NO blocking and NO open transactions
		if s.WaitType == "WAITFOR" || s.WaitType == "SP_SERVER_DIAGNOSTICS_SLEEP" {
			if s.BlockerID == 0 && s.TotalBlocked == 0 && s.OpenTxnCount == 0 {
				continue
			}
		}
		final = append(final, s)
	}

	// By default, we start with the most blocked sessions and then the oldest sessions
	sort.SliceStable(final, func(i, j int) bool {
		if final[i].TotalBlocked != final[j].TotalBlocked {
			return final[i].TotalBlocked > final[j].TotalBlocked
		}
		return final[i].StartTime.Before(final[j].StartTime)
	})

	// Filter out sketchy characters
	// TODO This doesn't seem to work.  I'm not sure what this character is
	// TODO Shorten strings to a reasonable length or filter them on the GUI
	for i := range final {
		final[i].StatementText = TrimSQL(final[i].StatementText, 2000)
		final[i].StatementText = strings.ReplaceAll(final[i].StatementText, "�", "")
	}

	return final, nil
}

// headBlocker recursively travels the block path returning
// the head blocking session ID
// it keeps track of where it has been and exits on a duplicate
func headBlocker(id int16, m map[int16]int16, p []int16, depth int, path string) (int16, []int16, string, error) {
	if depth > maxDepth {
		return 0, []int16{}, "", fmt.Errorf("headblocker: depth: %d", depth)
	}
	if p == nil {
		p = []int16{}
	}
	// we have already been to this id on this pass up the tree
	// this is likely a deadlock or circular lock
	if slices.Contains(p, id) {
		return id, p, path, nil
	}
	p = append(p, id)
	bid, exists := m[id] // session -> blocked by; bid=blocked by
	if !exists {
		return id, p, fmt.Sprintf("/%d", id) + path, nil
	}
	return headBlocker(bid, m, p, depth+1, fmt.Sprintf("/%d", id)+path)
}

var sessionQuery = `
	SELECT
		s.session_id
		,COALESCE(r.request_id, 0) AS request_id
		,CAST(CASE WHEN r.request_id IS NULL THEN 0 ELSE 1 END AS BIT) as has_request
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
			'(no text available)') AS statement_text
		,COALESCE(DB_NAME(COALESCE(r.database_id, s.database_id, 0)), '') AS [database] 
		--  ,COALESCE(DB_NAME(COALESCE(r.database_id, 0)), '') AS [database] 
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
		--,0 as open_transaction_count "
	FROM	sys.dm_exec_sessions s
	LEFT JOIN sys.dm_exec_requests r ON r.session_id = s.session_id
	OUTER APPLY sys.dm_exec_sql_text(r.sql_handle) AS st
	CROSS APPLY sys.dm_exec_input_buffer(s.session_id, null) AS ib
	WHERE	1=1
	AND		s.is_user_process = 1 
	AND		s.session_id <> @@SPID
	AND		(r.request_id IS NOT NULL 
			OR s.open_transaction_count > 0 )
	-- ORDER BY NEWID()
`

var SessionQueryLegacy = `
	SELECT	
		s.session_id
		,COALESCE(r.request_id, 0) AS request_id
		,CAST(CASE WHEN r.request_id IS NULL THEN 0 ELSE 1 END AS BIT) as has_request
		,COALESCE(r.start_time, s.last_request_start_time) as start_time
		,COALESCE(DATEDIFF(ss, r.start_time, GETDATE()), 0) AS RunTimeSeconds
		,COALESCE(r.[status], '') AS [status]
		,COALESCE(SUBSTRING(st.text, (r.statement_start_offset/2)+1, 
			((CASE r.statement_end_offset
			WHEN -1 THEN DATALENGTH(st.text)
			WHEN 0 THEN DATALENGTH(st.text)
			ELSE r.statement_end_offset
			END - r.statement_start_offset)/2) + 1),
			--ib.event_info, 
			'(no text available)') AS statement_text
		--,COALESCE(DB_NAME(COALESCE(r.database_id, s.database_id, 0)), '') AS [database] 
		,COALESCE(DB_NAME(COALESCE(r.database_id, 0)), '') AS [database] 
		,COALESCE(r.blocking_session_id, 0) AS blocking_session_id
		,COALESCE(r.wait_type, '') AS wait_type
		,COALESCE(r.wait_time, 0) AS wait_time
		,COALESCE(r.wait_resource, '') AS wait_resource
		,COALESCE(s.host_name, '') AS host_name
		,COALESCE(s.program_name, '') as AppName 
		,COALESCE(s.original_login_name, '') AS original_login_name
		,COALESCE(CAST(r.percent_complete AS INT),0) AS percent_complete
		,COALESCE(r.command, '') as command
		--,COALESCE(s.open_transaction_count, 0) as open_transaction_count 
		,0 as open_transaction_count 
	FROM	sys.dm_exec_sessions s
	LEFT JOIN sys.dm_exec_requests r ON r.session_id = s.session_id
	OUTER APPLY sys.dm_exec_sql_text(r.sql_handle) AS st
	-- CROSS APPLY sys.dm_exec_input_buffer(s.session_id, null) AS ib
	WHERE	1=1
	AND		s.is_user_process = 1 
	AND		s.session_id <> @@SPID
	AND		(r.request_id IS NOT NULL 
			/* OR s.open_transaction_count > 0 */)
	--ORDER BY NEWID()
`
