package main

import (
	"database/sql"
	"log"

	"github.com/billgraziano/mssqlh/v2"
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/microsoft/go-mssqldb/namedpipe"
	_ "github.com/microsoft/go-mssqldb/sharedmemory"
	"github.com/pkg/errors"
)

func main() {
	log.Println("Starting getsessions...")
	conn := mssqlh.NewConnection("D40\\SQL2016", "", "", "", "getsessions.exe")
	log.Printf("connection string: %s", conn.String())

	db, err := sql.Open("sqlserver", conn.String())
	if err != nil {
		log.Fatal(errors.Wrap(err, "sql.open"))
	}
	defer db.Close()

	//var serverName string
	rows, err := db.Query("SELECT name from sys.databases ")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		// err := rows.Scan(&id, &name)
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// log.Println(id, name)
		log.Println("row")
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	// log.Printf("@@SERVERNAME: %s\n", serverName)
	err = db.Close()
	if err != nil {
		log.Print(err)
	}
}

var sessionQuery = `
	SELECT
		s.session_id
		--,COALESCE(r.request_id, 0) AS request_id
		--,CAST(CASE WHEN r.request_id IS NULL THEN 0 ELSE 1 END AS BIT) as has_request
		--,COALESCE(r.start_time, s.last_request_start_time) as start_time
		--,COALESCE(DATEDIFF(ss, r.start_time, GETDATE()), 0) AS RunTimeSeconds
		--,COALESCE(r.[status], '') AS [status]
		--,COALESCE(SUBSTRING(st.text, (r.statement_start_offset/2)+1, 
		--	((CASE r.statement_end_offset
		--	WHEN -1 THEN DATALENGTH(st.text)
		--	WHEN 0 THEN DATALENGTH(st.text)
		--	ELSE r.statement_end_offset
		--	END - r.statement_start_offset)/2) + 1),
		--	ib.event_info, 
		--	'(no text available)') AS statement_text
		--,COALESCE(DB_NAME(COALESCE(r.database_id, s.database_id, 0)), '') AS [database] 
		--,COALESCE(r.blocking_session_id, 0) AS blocking_session_id
		--,COALESCE(r.wait_type, '') AS wait_type
		--,COALESCE(r.wait_time, 0) AS wait_time
		--,COALESCE(r.wait_resource, '') AS wait_resource
		--,COALESCE(s.host_name, '') AS host_name
		--,COALESCE(s.program_name, '') as AppName 
		--,COALESCE(s.original_login_name, '') AS original_login_name
		--,COALESCE(CAST(r.percent_complete AS INT),0) AS percent_complete
		--,COALESCE(r.command, '') as command
		--,COALESCE(s.open_transaction_count, 0) as open_transaction_count 
	FROM	sys.dm_exec_sessions s
	LEFT JOIN sys.dm_exec_requests r ON r.session_id = s.session_id
	OUTER APPLY sys.dm_exec_sql_text(r.sql_handle) AS st
	CROSS APPLY sys.dm_exec_input_buffer(s.session_id, null) AS ib
	WHERE	1=1
	AND		s.is_user_process = 1 
	AND		s.session_id <> @@SPID
	AND		(r.request_id IS NOT NULL 
			OR s.open_transaction_count > 0 )
	ORDER BY NEWID()
`
