package hadr

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type Latency struct {
	GroupID             string `db:"group_id"`
	ReplicaID           string `db:"replica_id"`
	GroupDatabaseID     string `db:"group_database_id"`
	SendQueueKB         int    `db:"log_send_queue_size"`
	RedoQueueKB         int    `db:"redo_queue_size"`
	SecondaryLagSeconds int    `db:"secondary_lag_seconds"`
}

func SetLatency(db *sql.DB) error {
	sqlxdb := sqlx.NewDb(db, "mssql")
	latenciesFromDB := []Latency{}
	stmt := getLatencyStatement
	if DEV {
		stmt += getLatencyStatementDEV
	}
	err := sqlxdb.Select(&latenciesFromDB, stmt)
	if err != nil {
		return errors.Wrap(err, "sqlx.select")
	}
	// build a map for each group_id with an array for the latencies
	var agl = make(map[string][]Latency)
	for _, l := range latenciesFromDB {
		// make sure in map
		latencies, ok := agl[l.GroupID]
		if !ok {
			latencies = make([]Latency, 0)
		}
		latencies = append(latencies, l)
		agl[l.GroupID] = latencies
	}

	// Add the arrays to the AGMap
	for id, latencies := range agl {
		PublicAGMap.SetLatencies(id, latencies)
	}
	return nil
}

var getLatencyStatement = `
SELECT	CAST(group_id AS NVARCHAR(128)) as group_id
		,CAST(replica_id AS NVARCHAR(128)) as replica_id
		,CAST(group_database_id AS NVARCHAR(128)) as group_database_id
		,COALESCE(log_send_queue_size, -1) AS log_send_queue_size
		,COALESCE(redo_queue_size, -1) AS redo_queue_size
		--,secondary_lag_seconds
FROM	sys.dm_hadr_database_replica_states
WHERE	is_local = 0
`

var getLatencyStatementDEV = `
UNION ALL
SELECT	group_id = '98597b24-da1d-4f05-80bc-ff050866d986'
		,CAST((select service_broker_guid FROM sys.databases WHERE [name] = 'tempdb') AS NVARCHAR(128)) as replica_id
		,CAST((select service_broker_guid FROM sys.databases WHERE database_id = 2) AS NVARCHAR(128)) as group_database_id
		,log_send_queue_size = 37
		,redo_queue_size = -1
		--,secondary_lag_seconds = 1
UNION ALL
SELECT	group_id = '98597b24-da1d-4f05-80bc-ff050866d986'
		,CAST((select service_broker_guid FROM sys.databases WHERE [name] = 'tempdb') AS NVARCHAR(128)) as replica_id
		,CAST((select service_broker_guid FROM sys.databases WHERE database_id = 4) AS NVARCHAR(128)) as group_database_id
		,log_send_queue_size = 19
		,redo_queue_size = 12345634
		--,secondary_lag_seconds = 9 
`
