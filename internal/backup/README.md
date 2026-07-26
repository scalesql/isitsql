Backup Package
==============
This describes how we will handle backups of databases in Contained Availability Groups.  This applies to SQL Server 2019 or higher.  That's when the physical_database_name column appears even though CAGs didn't show up until SQL Server 2022.

Database struct
---------------
* __Name__ - `sys.databases.name`
* __PhysicalName__ - `COALESCE(sys.databases.physical_database_name, sys.databases.name)` or `sys.databases.name` on pre-SQL 2019 servers

DBKey struct
------------
```go
// DBKey identifies a unique backup row
type DBKey struct {
	Domain   string // DEAFULT_DOMAIN(), fake AG name, (NONE)
	Host     string // AG name for AG databases, @@SERVERNAME
	Database string // Physical Name for AG databases in SQL 2022+
}
```

| Field    | Value                                                     |
|----------|-----------------------------------------------------------|
| Domain   | `COALESCE(DEFAULT_DOMAIN(), AG$$' + ag.[name], '(NONE)')` |
| Host     | `COALESCE(ag.[name], @@SERVERNAME)`                       |
| Database | `COALESCE(db.[physical_database_name], db.[name])`        |

### Notes 
* DEFAULT_DOMAIN() returns WORKGROUP in stand-alone instances.  It returns NULL for stand-alone Linux servers.
* Host should really be named container

How Backups are Stored
----------------------

| Connection | Backup Stmt | Location   | backupset.database_name | Container (Key)     |
|------------|-------------|------------|-------------------------|---------------------|
| Node1      | master      | local msdb | master                  | domain/node1/master |
| Node1      | AG_master   | local msdb | AG_master               | domain/AG/AG_master |
| Node2      | AG_master   | local msdb | AG_master               | domain/AG/AG_master |
| Listener   | AG_master   | AG_msdb    | AG_master               | domain/AG/AG_master |
| Listener   | master      | AG_msdb    | master                  | domain/AG/AG_master |

Query `backupset` to populate struct
------------------------------------
* Connected to a node
    * AG database
        * In `sys.databases`, `name` should equal `physical_database_name` unless `physical_database_name` is NULL for offline databases
        * JOIN `backupset.database_name` to `COALESCE(d.physcial_database_name, d.name)`
        * Key: domain/AG/COALESCE(d.physcial_database_name, d.name)
* Stand-alone database
    * Exact same JOIN, except we use the instance name for the key
    * domain/node/sys.databases.name
* Connected to the Listener
    * Can be in backupset as either `master` or `AG_master`
    * JOIN `backupset.database_name` to `COALESCE(d.physcial_database_name, d.name)`
    * LEFT JOIN to the AG on the session_id to get the AG name
    * Key: domain/AG/COALESCE(d.physcial_database_name, d.name) 


Query the list of databases
---------------------------
```sql
-- List databases along with what backup keys we will use
SELECT 
    db.[name] AS [db.name],
    db.[physical_database_name],

    ag.group_id AS [ag.group_id],
    ag.name AS [ag.name],
    adc.database_name AS [adc.database_name],

    -- the domain is empty for Linux
    -- priority is 
    --  (1) actual domain, 
    --  (2) ag name with some extra characters so all AG nodes have the same value, 
    --  (3) (NONE) -- or maybe, the OS name?
    COALESCE(DEFAULT_DOMAIN(), 'AG$$' + ag.[name], '(NONE)') AS backup_key_domain,
    COALESCE(ag.[name], @@SERVERNAME) AS                        backup_key_container,
    COALESCE(db.[physical_database_name], db.[name]) AS         backup_key_database
FROM    sys.databases db
LEFT JOIN sys.availability_databases_cluster adc 
    ON adc.database_name = COALESCE(db.[physical_database_name], db.[name])
LEFT JOIN sys.availability_groups ag ON ag.group_id = adc.group_id
where db.[name] NOT IN ('tempdb')
ORDER BY ag.name, adc.database_name;
```

Query the backups
-----------------
```sql
-- SQL Server 2022 and higher
-- Are we connected to the Listener of a CAG?
-- If so, we will use the CAG name here
DECLARE @CAG SYSNAME;
SELECT @CAG = ag.[name]
FROM sys.dm_exec_sessions s
JOIN sys.availability_groups ag ON ag.group_id = s.contained_availability_group_id
WHERE session_id = @@SPID;

;WITH rawbackups AS ( 
    SELECT 
        bus.server_name
        ,bus.database_name
        ,bus.backup_start_date
        ,bus.type
        ,bmf.physical_device_name
    from msdb.dbo.backupset bus
    join msdb.dbo.backupmediafamily bmf on bmf.media_set_id = bus.media_set_id
), fixeddb AS (
    SELECT  
        @@SERVERNAME AS instance,
        -- first get AGName_master if we backed up master inside an AG
        -- this also gets master inside master
        -- second, if we backed up AGName_master, get that
        -- third, just get a regular database name
        COALESCE(logicaldb.physical_database_name, physdb.[physical_database_name], logicaldb.[name]) as [database_name],
        backup_start_date, 
        [type], 
        physical_device_name
    FROM    rawbackups
    -- JOIN on the physical database name (for master and msdb)
    LEFT JOIN   sys.databases physdb ON physdb.[physical_database_name] = rawbackups.database_name
    -- JOIN on the logical database name (for master and msdb)
    LEFT JOIN   sys.databases logicaldb ON logicaldb.[name] = rawbackups.database_name 
    -- In the Contained AG, sys.databases.replica_id will be null, but we have the AG from above
), fixedag AS (
    SELECT fixeddb.*
        ,ag.[name] AS ag_name
    FROM fixeddb
    LEFT JOIN sys.databases db ON COALESCE(db.physical_database_name, db.name) = fixeddb.database_name
    LEFT JOIN sys.dm_hadr_database_replica_states hdrs 
        ON db.database_id = hdrs.database_id 
        AND hdrs.is_local = 1  -- Filters to the local replica's stats
    LEFT JOIN sys.availability_groups ag ON hdrs.group_id = ag.group_id
),
sorted AS (
    SELECT *,
        ROW_NUMBER() OVER(PARTITION BY database_name, type ORDER BY backup_start_date DESC) AS RowNumber
    FROM fixedag
)
SELECT 
    [instance]
    ,[database_name]
    ,backup_start_date
    ,[type]
    ,physical_device_name
    ,COALESCE(DEFAULT_DOMAIN(), 'AG$$' + @CAG, '(NONE)') AS backup_key_domain
    ,COALESCE(@CAG, ag_name, @@SERVERNAME) AS backup_key_container
    ,[database_name] as backup_key_database
FROM sorted
WHERE RowNumber = 1 
ORDER BY backup_start_date DESC, database_name;

```

TODO
----
* For stand-alone servers, just store the backups locally?

