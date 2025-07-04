package hadr

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type ReplicaDatabase struct {
	SuspendReasonDesc   string `db:"suspend_reason_desc"`
	DatabaseName        string `db:"database_name"`
	AGName              string `db:"ag_name"`
	SyncStateDesc       string `db:"synchronization_state_desc"`
	GroupDatabaseID     string `db:"group_database_id"`
	ReplicaID           string `db:"replica_id"`
	SyncHealthDesc      string `db:"synchronization_health_desc"`
	GroupID             string `db:"group_id"`
	DatabaseStateDesc   string `db:"database_state_desc"`
	DatabaseID          int    `db:"database_id"`
	RedoQueueKB         int
	SendQueueKB         int
	SyncState           int8 `db:"synchronization_state"`
	SuspendReason       int8 `db:"suspend_reason"`
	IsSuspended         bool `db:"is_suspended"`
	DatabaseState       int8 `db:"database_state"`
	SyncHealth          int8 `db:"synchronization_health"`
	IsCommitParticipant bool `db:"is_commit_participant"`
	IsPrimary           bool `db:"is_primary_replica"`
	IsLocal             bool `db:"is_local"`
}

// State returns a text description of the Availability Group state
func (d *ReplicaDatabase) State() string {
	// AGName: Synchronized, Healthy, Online (SUSPEND_REASON)
	// fmt.Printf("PublicAGMap: %+v\n", PublicAGMap)
	// fmt.Println("==================================================")
	// fmt.Printf("ReplicaDatabase: %+v\n", d)
	// fmt.Println("==================================================")
	tc := cases.Title(language.AmericanEnglish)
	agname := PublicAGMap.GetDisplayName(d.GroupID)
	if agname == "" {
		agname = d.AGName
	}
	state := fmt.Sprintf("%s: ", agname)

	// role
	if d.IsPrimary {
		state += "Primary"
	} else {
		state += "Secondary"
	}
	if d.SyncState == 1 || d.SyncState == 2 {
		state += ", " + tc.String(d.SyncStateDesc)
	} else {
		state += ", " + d.SyncStateDesc
	}
	if d.SyncHealth == 2 {
		state += fmt.Sprintf(", %s", tc.String(d.SyncHealthDesc))
	} else {
		state += fmt.Sprintf(", %s", d.SyncHealthDesc)
	}
	if d.DatabaseState == 0 {
		state += fmt.Sprintf(", %s", tc.String(d.DatabaseStateDesc))
	} else {
		state += fmt.Sprintf(", %s", d.DatabaseStateDesc)
	}
	//state += fmt.Sprintf(" %s, %s, %s", tc.String(d.SyncStateDesc), tc.String(d.SyncHealthDesc), tc.String(d.DatabaseStateDesc))
	if d.IsSuspended {
		state += fmt.Sprintf(" (%s)", d.SuspendReasonDesc)
	}
	return state
}

// GetReplicaDatabases returns a list of databases that are on the current node
func GetReplicaDatabases(db *sql.DB) (map[int]ReplicaDatabase, error) {
	m := make(map[int]ReplicaDatabase)
	//var sql string
	var err error

	sqlxdb := sqlx.NewDb(db, "mssql")
	sqlxdb = sqlxdb.Unsafe()

	dbs := []ReplicaDatabase{}
	stmt := getDatabasesStatement
	if DEV {
		stmt += DEVgetDatabasesStatement
	}
	err = sqlxdb.Select(&dbs, stmt)
	if err != nil {
		return m, errors.Wrap(err, "sqlx.select")
	}
	for _, db0 := range dbs {
		newdb := db0
		newdb.GroupID = strings.ToUpper(newdb.GroupID)
		newdb.ReplicaID = strings.ToUpper(newdb.ReplicaID)
		newdb.GroupDatabaseID = strings.ToUpper(newdb.GroupDatabaseID)
		m[newdb.DatabaseID] = newdb
	}
	return m, nil
}

var getDatabasesStatement = `
SELECT	ag.[name] AS ag_name, 
		agdb.database_id, 
		COALESCE(DB_NAME(agdb.database_id), 'NULL') AS database_name,
		agdb.is_local, is_primary_replica,
		agdb.synchronization_state, agdb.synchronization_state_desc,
		agdb.is_commit_participant, 
		agdb.synchronization_health,
		agdb.synchronization_health_desc,
		agdb.database_state, agdb.database_state_desc,
		agdb.is_suspended, 
		COALESCE(agdb.suspend_reason_desc, '') AS suspend_reason_desc
		,CAST(agdb.group_id AS NVARCHAR(128)) AS group_id
		,CAST(agdb.replica_id AS NVARCHAR(128)) AS replica_id
		,CAST(agdb.group_database_id AS NVARCHAR(128)) AS group_database_id
FROM	sys.dm_hadr_database_replica_states agdb
JOIN	sys.availability_groups ag ON ag.group_id = agdb.group_id
JOIN	sys.availability_replicas ar ON ar.group_id = agdb.group_id AND ar.replica_id = agdb.replica_id
JOIN	sys.dm_hadr_availability_replica_states ars ON ars.group_id = agdb.group_id AND ars.replica_id = agdb.replica_id
WHERE	agdb.is_local = 1
`

var DEVgetDatabasesStatement = `
UNION ALL 
	SELECT	
		ag_name = 'MyAG', 
		database_id = 6,
		database_name = COALESCE(DB_NAME(6), 'NULL'),
		is_local = 1, 
		is_primary_replica = 1,
		synchronization_state = 4 , 
		synchronization_state_desc = 'INITIALIZING',
		is_commit_participant = 1, 
		synchronization_health = 0,
		synchronization_health_desc = 'NOT_HEALTHY',
		database_state = 3, 
		database_state_desc = 'OFFLINE',
		is_suspended = 1, 
		suspend_reason_desc = 'SUSPEND_X'
		,group_id = '98597b24-da1d-4f05-80bc-ff050866d986'
		,replica_id = CAST((select service_broker_guid FROM sys.databases WHERE [name] = 'tempdb') AS NVARCHAR(128))
		,group_database_id = CAST((select service_broker_guid FROM sys.databases WHERE database_id = 2) AS NVARCHAR(128))
UNION ALL 
	SELECT	
		ag_name = 'AG-SQL2016', 
		database_id = 8, 
		database_name = COALESCE(DB_NAME(8), 'NULL'),
		is_local = 1, 
		is_primary_replica = 1,
		synchronization_state = 1 , 
		synchronization_state_desc = 'SYNCHRONIZING',
		is_commit_participant = 1, 
		synchronization_health = 2,
		synchronization_health_desc = 'HEALTHY',
		database_state = 0, 
		database_state_desc = 'ONLINE',
		is_suspended = 0, 
		suspend_reason_desc = 'SUSPEND_X'	
		,group_id = '185D6848-C278-47E8-8A8F-049456161D46'
		,replica_id = CAST((select service_broker_guid FROM sys.databases WHERE [name] = 'tempdb')AS NVARCHAR(128))
		,group_database_id = CAST((select service_broker_guid FROM sys.databases WHERE database_id = 4)	 AS NVARCHAR(128))
		`
