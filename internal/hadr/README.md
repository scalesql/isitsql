```
SELECT	ag.[name], agdb.database_id, agdb.is_local, is_primary_replica,
		agdb.synchronization_state, agdb.synchronization_state_desc,
		agdb.is_commit_participant, 
		agdb.synchronization_health,
		agdb.synchronization_health_desc,
		agdb.database_state, agdb.database_state_desc,
		agdb.is_suspended, agdb.suspend_reason, agdb.suspend_reason_desc
FROM	sys.dm_hadr_database_replica_states agdb
JOIN	sys.availability_groups ag ON ag.group_id = agdb.group_id
JOIN	sys.availability_replicas ar ON ar.group_id = agdb.group_id AND ar.replica_id = agdb.replica_id
JOIN	sys.dm_hadr_availability_replica_states ars ON ars.group_id = agdb.group_id AND ars.replica_id = agdb.replica_id
WHERE	agdb.is_local = 1


select db_name(database_id), * from sys.dm_hadr_database_replica_states
select * from sys.availability_databases_cluster
select * from sys.dm_hadr_cached_database_replica_states

```


