package agent

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	mssql "github.com/microsoft/go-mssqldb"
	_ "github.com/microsoft/go-mssqldb/namedpipe"
	_ "github.com/microsoft/go-mssqldb/sharedmemory"
)

// Job holds a SQL Server Agent Job
type Job struct {
	JobID                      mssql.UniqueIdentifier
	OriginatingServer          string
	Name                       string
	Enabled                    bool
	Description                string
	Category                   string
	Owner                      string
	LastRun                    time.Time
	LastRunOutcome             string
	NextRun                    time.Time
	ExecutionStatus            int
	ExecutionStatusDescription string
	ExecutionStep              string
	HistoryURL                 string

	// Fields to control display
	CSSClass string

	// These are not part of the job
	Key        string // key of the server that populated this
	DomainName string
	ServerName string
}

type JobList []Job

type rawJob struct {
	JobID                  mssql.UniqueIdentifier `db:"job_id"`
	OriginatingServer      string                 `db:"originating_server"`
	Name                   string                 `db:"name"`
	Enabled                bool                   `db:"enabled"`
	Description            sql.NullString         `db:"description"`
	Category               sql.NullString         `db:"category"`
	Owner                  sql.NullString         `db:"owner"`
	LastRunDate            int32                  `db:"last_run_date"`
	LastRunTime            int32                  `db:"last_run_time"`
	LastRunOutcome         int32                  `db:"last_run_outcome"`
	NextRunDate            int32                  `db:"next_run_date"`
	NextRunTime            int32                  `db:"next_run_time"`
	CurrentExecutionStatus int32                  `db:"current_execution_status"`
	CurrentExecutionStep   sql.NullString         `db:"current_execution_step"`
	CurrentRetryAttempt    int32                  `db:"current_retry_attempt"`
	HasSchedule            int                    `db:"has_schedule"`
	Type                   int                    `db:"type"`
}

// Maybe pass in domain and server name?
// or query it here
func FetchJobs(ctx context.Context, key string, pool *sql.DB) (JobList, error) {
	var err error

	e, err := FetchEnvironment(ctx, pool)
	if err != nil {
		return JobList{}, err
	}
	if !e.HasPermission() {
		return JobList{}, &ErrNoPermission{e}
	}

	sqlxdb := sqlx.NewDb(pool, "sqlserver")
	sqlxdb = sqlxdb.Unsafe()
	rawjobs := make([]rawJob, 0)
	err = sqlxdb.SelectContext(ctx, &rawjobs, "SET TRANSACTION ISOLATION LEVEL READ UNCOMMITTED; EXEC msdb.dbo.sp_help_job;")
	if err != nil {
		return JobList{}, err
	}
	jobs := make([]Job, 0, len(rawjobs))
	for _, rj := range rawjobs {
		j, err := jobFromRaw(rj, key)
		if err != nil {
			return JobList{}, err
		}
		j.DomainName = e.DefaultDomain
		j.ServerName = e.ServerName
		//println(dt.String(), j.LastRun.String())
		jobs = append(jobs, j)
	}
	// TODO sort by name
	return jobs, nil
}

// FetchRunningJobs queries the jobs and only returns those that aren't Idle or Obsolete
func FetchRunningJobs(ctx context.Context, key string, pool *sql.DB) (JobList, error) {
	jobs, err := FetchJobs(ctx, key, pool)
	if err != nil {
		return JobList{}, err
	}
	running := JobList{}
	for _, j := range jobs {
		if j.ExecutionStatus == 4 || j.ExecutionStatus == 6 {
			continue
		}
		running = append(running, j)
	}
	return running, nil
}

func FetchJob(ctx context.Context, key string, jobID string, pool *sql.DB) (Job, error) {
	var err error

	var jobUUID mssql.UniqueIdentifier
	err = jobUUID.Scan(jobID)
	if err != nil {
		return Job{}, err
	}

	e, err := FetchEnvironment(ctx, pool)
	if err != nil {
		return Job{}, err
	}
	if !e.HasPermission() {
		return Job{}, &ErrNoPermission{e}
	}

	sqlxdb := sqlx.NewDb(pool, "sqlserver")
	sqlxdb = sqlxdb.Unsafe()
	rawjob := rawJob{}
	stmt := "SET TRANSACTION ISOLATION LEVEL READ UNCOMMITTED; EXEC msdb.dbo.sp_help_job @job_aspect='job', @job_id=@p1"
	err = sqlxdb.GetContext(ctx, &rawjob, stmt, jobUUID.String())
	if err != nil {
		return Job{}, err
	}

	j, err := jobFromRaw(rawjob, key)
	if err != nil {
		return Job{}, err
	}
	j.DomainName = e.DefaultDomain
	j.ServerName = e.ServerName

	return j, nil
}

func jobFromRaw(rj rawJob, key string) (Job, error) {
	//fmt.Printf("rj.JobID: %+v\n", rj.JobID)

	// var jobUUID mssql.UniqueIdentifier

	u, err := url.Parse("/")
	if err != nil {
		return Job{}, err
	}
	u.Path = path.Join("/", "server", key, "jobs", strings.ToLower(rj.JobID.String()), "history")
	//println(u.String())
	j := Job{
		Key:               key,
		JobID:             rj.JobID,
		OriginatingServer: rj.OriginatingServer,
		Name:              rj.Name,
		Enabled:           rj.Enabled,
		Category:          rj.Category.String,
		Description:       rj.Description.String,
		Owner:             rj.Owner.String,
		ExecutionStep:     rj.CurrentExecutionStep.String,
		ExecutionStatus:   int(rj.CurrentExecutionStatus),
		HistoryURL:        u.String(),
	}
	//println(rj.HasSchedule, rj.Enabled, rj.LastRunDate, rj.LastRunTime, rj.LastRunOutcome)
	dt, err := agentTime(rj.LastRunDate, rj.LastRunTime)
	if err == nil {
		j.LastRun = dt
	}

	dt, err = agentTime(rj.NextRunDate, rj.NextRunTime)
	if err == nil {
		j.NextRun = dt
	}

	switch rj.LastRunOutcome {
	case 0:
		j.LastRunOutcome = "Failed"
	case 1:
		j.LastRunOutcome = "Succeeded"
	case 3:
		j.LastRunOutcome = "Cancelled"
	case 5:
		j.LastRunOutcome = ""
	default:
		j.LastRunOutcome = fmt.Sprintf("undefined: %d", rj.LastRunOutcome)
	}

	switch rj.CurrentExecutionStatus {
	case 1:
		j.ExecutionStatusDescription = "Executing"
	case 2:
		j.ExecutionStatusDescription = "Waiting for Thread"
	case 3:
		j.ExecutionStatusDescription = "Between Retries"
	case 4:
		j.ExecutionStatusDescription = ""
	case 5:
		j.ExecutionStatusDescription = "Suspended"
	case 6:
		j.ExecutionStatusDescription = "Obsolete"
	case 7:
		j.ExecutionStatusDescription = "Performing Completion"
	default:
		j.ExecutionStatusDescription = fmt.Sprintf("undefined: %d", rj.CurrentExecutionStatus)

	}

	/*
		failed and idle is danger
		any other bad case is warning
	*/

	switch {
	case rj.LastRunOutcome == 0 && rj.CurrentExecutionStatus == 4: // failed and idle
		j.CSSClass = "danger"
	case rj.LastRunOutcome == 3 || rj.CurrentExecutionStatus == 3: // cancelled or between retries
		j.CSSClass = "warning"
	case rj.LastRunOutcome == 0 && rj.CurrentExecutionStatus == 1: // failed and executing
		j.CSSClass = "warning"

	default:
	}

	if strings.HasPrefix(j.Category, "[Uncategorized") {
		j.Category = ""
	}
	if strings.HasPrefix(j.ExecutionStep, "0") {
		j.ExecutionStep = ""
	}

	if j.Description == "No description available." {
		j.Description = ""
	}

	return j, nil
}

func (j Job) StepLogURL() string {
	u := url.URL{}
	u.Path = path.Join("/", "server", j.Key, "jobs", strings.ToLower(j.JobID.String()), "steplog")
	return u.String()
}
