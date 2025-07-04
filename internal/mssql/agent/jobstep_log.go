package agent

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	mssql "github.com/microsoft/go-mssqldb"
	_ "github.com/microsoft/go-mssqldb/namedpipe"
	_ "github.com/microsoft/go-mssqldb/sharedmemory"
)

type JobStepLog struct {
	JobName   string `db:"job_name"`
	StepName  string `db:"step_name"`
	StepID    int    `db:"step_id"`
	Subsystem string `db:"subsystem"`
	LogID     int    `db:"log_id"`
	Log       string `db:"log"`
	LogBytes  int    `db:"log_size"`
	StepUID   string `db:"step_uid"`
}

// FetchJobStepLog gets the step output for a job
func FetchJobStepLog(key string, jobid string, pool *sql.DB) ([]JobStepLog, error) {
	var jobUUID mssql.UniqueIdentifier
	err := jobUUID.Scan(jobid)
	if err != nil {
		return []JobStepLog{}, err
	}
	e, err := FetchEnvironment(context.TODO(), pool)
	if err != nil {
		return []JobStepLog{}, err
	}
	if !e.HasPermission() {
		return []JobStepLog{}, &ErrNoPermission{e}
	}
	sqlxdb := sqlx.NewDb(pool, "sqlserver")
	sqlxdb = sqlxdb.Unsafe()
	steps := make([]JobStepLog, 0)
	err = sqlxdb.Select(&steps, jobStepLogQuery, jobUUID)
	if err != nil {
		return []JobStepLog{}, err
	}
	return steps, nil
}

var jobStepLogQuery = `
SELECT  j.[name] as job_name, s.[step_name], s.step_id, s.subsystem, l.* 
FROM	msdb.dbo.sysjobs_view j
JOIN	msdb.dbo.sysjobsteps s ON s.job_id = j.job_id
JOIN	msdb.dbo.sysjobstepslogs l ON l.step_uid = s.step_uid
WHERE 	j.job_id = @p1
ORDER BY s.step_id; 
`
