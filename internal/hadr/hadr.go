package hadr

import (
	"context"
	"database/sql"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// ErrNoAG is returned when no rows are returned by the DMV
var ErrNoAG = errors.New("hadr: no availability groups")

// AG is an Availability Group
type AG struct {
	PollTime       time.Time
	Health         string
	Domain         string
	PrimaryReplica string
	State          string
	Name           string
	GUID           string
	DisplayName    string
	PrimaryGUID    string // which target sent us this AG
	Replicas       []*Replica
	Listeners      []Listener
	Latencies      []Latency
	isHealthy      bool
}

// Replica holds the various AG replicas
type Replica struct {
	Name             string
	AvailabilityMode string
	Failover         string
	Role             string
	State            string
	Health           string
	SendQueue        int64
	SendRate         int64
	RedoQueue        int64
	RedoRate         int64
	ReadyDatabases   int32
	TotalDatabases   int32
	IsPrimary        bool
	IsHealthy        bool
}

// Listener is an Availability Group Listener
type Listener struct {
	GUID       string
	Name       string
	IPConfig   string
	Port       int
	Conformant bool
}

// IsHealthy checks the last poll time and health to flag for errors
func (ag *AG) IsHealthy() bool {
	if ag.isHealthy && time.Since(ag.PollTime) < time.Duration(3*time.Minute) {
		return true
	}
	return false
}

// GetNames returns all AG names and Listener names hosted by a server
func GetNames(db *sql.DB) ([]string, error) {
	var stmt string
	list := make([]string, 0)
	if DEV {
		stmt = `
			SELECT 'AG1' AS [name]
			UNION
			SELECT 'Listener1' AS [name]
		`
	} else {
		stmt = `
			IF CAST(PARSENAME(CAST(SERVERPROPERTY('productversion') AS varchar(20)), 4) AS INT) < 12 
				RETURN; 

			SELECT	[name] 
			FROM	sys.availability_groups
			WHERE 	is_distributed = 0
			UNION
			SELECT	[dns_name] as [name]
			FROM	sys.availability_group_listeners
		`
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	rows, err := db.QueryContext(ctx, stmt)
	if err != nil {
		return []string{}, errors.Wrap(err, "db.query")
	}
	defer rows.Close()

	for rows.Next() {
		var str string
		err = rows.Scan(&str)
		if err != nil {
			return []string{}, errors.Wrap(err, "rows.scan")
		}
		list = append(list, str)
	}

	return list, nil
}

// GetAGList gets a list of all AGs hosted on this node
func GetAGList(db *sql.DB, serverName string) (map[string]*AG, error) {
	m := make(map[string]*AG)
	var sql string
	var err error

	// Get basic AG info
	if DEV {
		// return two fake AGs
		sql = `SELECT 'AG-'+ CAST(COALESCE(SERVERPROPERTY('InstanceName'), @@SERVERNAME) AS nvarchar(128)) AS [name]
			, @@SERVERNAME AS primary_replica
			, CASE WHEN RAND() > .30 THEN 'ONLINE' ELSE 'ONLINE_IN_PROGRESS' END  as health
			, CASE WHEN RAND() > .50 THEN 'HEALTHY' ELSE 'NOT_HEALTHY' END  as syncHealth
			, X = CAST((select service_broker_guid from master.sys.databases where [name] = 'tempdb') AS NVARCHAR(128))
			, COALESCE(CAST(DEFAULT_DOMAIN() AS NVARCHAR(256)), '') AS [DomainName]
			UNION 
			SELECT 'AG2-'+@@SERVERNAME AS [name]
			, @@SERVERNAME AS primary_replica
			, 'ONLINE' as health
			, CASE WHEN RAND() > .50 THEN 'HEALTHY' ELSE 'NOT_HEALTHY' END  as syncHealth
			, X = CAST((select service_broker_guid from master.sys.databases where [name] = 'msdb') AS NVARCHAR(128))
			, COALESCE(CAST(DEFAULT_DOMAIN() AS NVARCHAR(256)), '') AS [DomainName]`
	} else {

		sql = `		
			SELECT 
				COALESCE(ag.[name], '(NULL)') AS [name],
				COALESCE(ags.primary_replica, '(NULL)') AS primary_replica,
				COALESCE(primary_recovery_health_desc, '(NULL)') AS primary_recovery_health_desc, 
				COALESCE(synchronization_health_desc, '(NULL)') AS synchronization_health_desc,
				COALESCE(CAST(ag.group_id AS NVARCHAR(128)), '(NULL)') AS group_id,
				COALESCE(CAST(DEFAULT_DOMAIN() AS NVARCHAR(256)), '') AS domain_name
			FROM	sys.availability_groups ag
			JOIN	sys.dm_hadr_availability_group_states ags ON ags.group_id = ag.group_id
			WHERE 	ag.is_distributed = 0 
			`
	}
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	rows, err := db.QueryContext(ctx, sql)
	if err != nil {
		dur := time.Since(start).String()
		return m, errors.Wrapf(err, "query-ag: %s", dur)
	}
	defer rows.Close()

	for rows.Next() {
		var ag AG
		ag.PollTime = time.Now()
		ag.isHealthy = false

		err = rows.Scan(&ag.Name, &ag.PrimaryReplica, &ag.State, &ag.Health, &ag.GUID, &ag.Domain)
		if err != nil {
			return m, errors.Wrap(err, "query-ag-scan")
		}

		// Ignore any non-primary replicas
		if !strings.EqualFold(ag.PrimaryReplica, serverName) {
			continue
		}

		ag.DisplayName = ag.Name
		if strings.EqualFold(ag.Health, "HEALTHY") {
			ag.isHealthy = true
		}

		err = ag.getNodes(db, ag.GUID)
		if err != nil {
			return m, errors.Wrap(err, "ag-get-nodes")
		}

		// Add to the map
		m[ag.GUID] = &ag
	}

	if DEV {
		sql = `
		SELECT
	   		group_id = CAST((select service_broker_guid from master.sys.databases where [name] = 'tempdb') AS NVARCHAR(128)),
	   		listener_id = '51565DE9-79F3-4879-99FC-265AC1575358',
	   		dns_name = 'Listener1'+'-'+@@SERVERNAME,
	   		port = 1433,
	   		is_conformant = 1,
	   		ip_configuration_string_from_cluster = '(''IP Address: 10.34.130.46'' or ''IP Address: 10.34.2.46'')'
		UNION

		SELECT
	   		group_id = '7129B676-D85F-4F67-B74F-D1070FA16C5E',
	   		listener_id = '0775F42B-964C-4F49-8988-0FF3AE2287E1',
	   		dns_name = 'Listener2'+'-'+@@SERVERNAME,
	   		port = 1433,
	   		is_conformant = 1,
	   		ip_configuration_string_from_cluster = '(''IP Address: 10.34.130.99'' or ''IP Address: 10.34.2.99'')'

			   UNION

		SELECT
				group_id = CAST((select service_broker_guid from master.sys.databases where [name] = 'msdb') AS NVARCHAR(128)),
				listener_id = '6BB79AF5-7CC4-45A5-AB08-87F5CD26D058',
				dns_name = 'Listener3'+'-'+@@SERVERNAME,
				port = 1433,
				is_conformant = 1,
				ip_configuration_string_from_cluster = '(''IP Address: 10.34.130.299'' or ''IP Address: 10.34.2.299'')'
		`
	} else {
		sql = `
			SELECT
				CAST(agl.group_id AS NVARCHAR(128)) AS group_id,
				UPPER(CAST(agl.listener_id AS NVARCHAR(128))) AS listener_id,
				agl.dns_name,
				agl.[port],
				agl.is_conformant,
				agl.ip_configuration_string_from_cluster
			FROM	sys.availability_groups ag
			JOIN	sys.availability_group_listeners  agl ON agl.group_id = ag.group_id
			WHERE	ag.is_distributed = 0 
			ORDER BY agl.dns_name;
`
	}

	r2, err := db.Query(sql)
	if err != nil {
		return m, errors.Wrap(err, "listener")
	}
	defer r2.Close()

	for r2.Next() {
		var l Listener
		var groupID string
		err = r2.Scan(&groupID, &l.GUID, &l.Name, &l.Port, &l.Conformant, &l.IPConfig)
		if err != nil {
			return m, errors.Wrap(err, "query-listener-scan")
		}
		ag, exists := m[groupID]
		if exists {
			ag.Listeners = append(ag.Listeners, l)
		}
	}
	return m, nil
}

func (ag *AG) getNodes(db *sql.DB, aguid string) error {
	var rows *sql.Rows
	var err error
	var sql string

	if DEV {
		sql = `
			SELECT @@SERVERNAME,'SYNCHRONOUS_COMMIT','AUTOMATIC','PRIMARY',	'CONNECTED',CASE WHEN RAND() > .10 THEN 'HEALTHY' ELSE 'NOT_HEALTHY' END,0,0,0,0,1,2
			UNION
			SELECT 'Server2',   'SYNCHRONOUS_COMMIT','AUTOMATIC',	'SECONDARY','CONNECTED',CASE WHEN RAND() > .10 THEN 'HEALTHY' ELSE 'NOT_HEALTHY' END,CAST(RAND() * 1024 * 2 AS BIGINT) ,2, CAST(RAND() * 1024 * 20480  AS BIGINT) ,4,1,2
			UNION
			SELECT 'Server3',   'ASYNCHRONOUS_COMMIT','MANUAL',	'SECONDARY','CONNECTED',CASE WHEN RAND() > .10 THEN 'HEALTHY' ELSE 'NOT_HEALTHY' END,CAST(RAND() * 1024 * 2 AS BIGINT) ,2, CAST(RAND() * 1024 * 20480  AS BIGINT) ,4,1,2
			`
	} else {
		sql = `	
			;WITH Latency AS (
				SELECT	group_id,
						replica_id,
						SUM(log_send_queue_size) as SendQueue,
						SUM(log_send_rate) AS SendRate,
						SUM(redo_queue_size) AS RedoQueue,
						SUM(redo_rate) AS RedoRate

				FROM	sys.dm_hadr_database_replica_states
				WHERE 	group_id = @p1
				GROUP BY group_id, replica_id
			), Failover_Ready AS (
				SELECT	replica_id,
						SUM(CASE WHEN is_failover_ready = 1 THEN 1 ELSE 0 END) AS ready_databases,
						COUNT(*) AS total_databases
				FROM	sys.dm_hadr_database_replica_cluster_states
				GROUP BY replica_id
			)
			SELECT	ar.replica_server_name, 
					ar.availability_mode_desc, 
					ar.failover_mode_desc, 
					ars.role_desc,
					ars.connected_state_desc,
					ars.synchronization_health_desc
					,COALESCE(L.SendQueue, 0) AS send_queue
					,COALESCE(L.SendRate, 0)  AS send_rate
					,COALESCE(L.RedoQueue, 0) AS [redo_queue]
					,COALESCE(L.RedoRate, 0)  AS redo_rate
					,COALESCE(FR.ready_databases, 0) AS ready_databases
					,COALESCE(FR.total_databases, 0) AS total_databases 
			FROM	sys.availability_replicas ar
			JOIN	sys.dm_hadr_availability_replica_states ars ON ars.replica_id = ar.replica_id
			LEFT JOIN	Latency L ON L.group_id = ar.group_id AND L.replica_id = ar.replica_id
			LEFT JOIN	Failover_Ready FR ON FR.replica_id = ar.replica_id
			WHERE 	ar.group_id = @p2
			ORDER BY ar.replica_server_name;


					`
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	rows, err = db.QueryContext(ctx, sql, aguid, aguid)
	if err != nil {
		return errors.Wrap(err, "query ag details")
	}
	defer rows.Close()

	//t.Replicas = make([]Replica, 0)
	for rows.Next() {
		var r Replica
		err := rows.Scan(&r.Name, &r.AvailabilityMode, &r.Failover, &r.Role, &r.State, &r.Health,
			&r.SendQueue, &r.SendRate, &r.RedoQueue, &r.RedoRate, &r.ReadyDatabases, &r.TotalDatabases)
		if err != nil {
			return errors.Wrap(err, "scan ag details")
		}

		if strings.EqualFold(r.Name, ag.PrimaryReplica) {
			r.IsPrimary = true
		}

		if r.State == "CONNECTED" && r.Health == "HEALTHY" {
			r.IsHealthy = true
		}

		// if r.Role == "SECONDARY" {
		// 	r.Role = strings.ToLower(r.Role)
		// }

		if r.AvailabilityMode == "ASYNCHRONOUS_COMMIT" {
			r.AvailabilityMode = strings.ToLower("asynchronous")
		}

		if r.Failover == "MANUAL" {
			r.Failover = strings.ToLower("manual")
		}

		ag.Replicas = append(ag.Replicas, &r)
	}
	return nil
}

// ListenerNames returns the list of Listeners as a CSV
func (ag *AG) ListenerNames() string {
	if ag.Listeners == nil {
		return ""
	}
	if len(ag.Listeners) == 0 {
		return ""
	}
	names := make([]string, 0)
	for _, l := range ag.Listeners {
		names = append(names, l.Name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// SetDisplayName sets the DisplayName field
// Each array entry needs 3 values: domain, AG name, display name
func (ag *AG) SetDisplayName(names [][]string) {

	// check the map
	for _, arr := range names {
		if len(arr) == 3 {
			if strings.EqualFold(ag.Domain, arr[0]) {
				// Compare the AG name
				if strings.EqualFold(ag.Name, arr[1]) {
					ag.DisplayName = arr[2]
					return
				}
				// Compare the listener name
				for _, l := range ag.Listeners {
					if strings.EqualFold(l.Name, arr[1]) {
						ag.DisplayName = arr[2]
						return
					}
				}
			}
		}
	}
	// do the listenrs
	listeners := ag.ListenerNames()
	if listeners != "" {
		ag.DisplayName = listeners
		return
	}
	if ag.Name != "" {
		ag.DisplayName = ag.Name
		return
	}
	if ag.GUID != "" {
		ag.DisplayName = ag.GUID
		return
	}
	ag.DisplayName = "(Unknown)"
}
