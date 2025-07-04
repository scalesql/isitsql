package agent

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	mssql "github.com/microsoft/go-mssqldb"
	_ "github.com/microsoft/go-mssqldb/namedpipe"
	_ "github.com/microsoft/go-mssqldb/sharedmemory"
)

// JobHistoryRow holds rows from msdb.dbo.sysjobhistory
type JobHistoryRow struct {
	InstanceID       int32                  `db:"instance_id"`
	JobID            mssql.UniqueIdentifier `db:"job_id"`
	JobName          sql.NullString         `db:"job_name"`
	StepID           int32                  `db:"step_id"`
	StepName         sql.NullString         `db:"step_name"`
	SQLMessageID     int32                  `db:"sql_message_id"`
	SQLSeverity      int32                  `db:"sql_severity"`
	Message          sql.NullString         `db:"message"`
	RunStatus        int32                  `db:"run_status"`
	RunDate          int32                  `db:"run_date"`     // Consider using a date parser for a more readable format
	RunTime          int32                  `db:"run_time"`     // Same here, a time parser might help
	RunDuration      int32                  `db:"run_duration"` // Could be split into hours, minutes, seconds if needed
	RetriesAttempted int32                  `db:"retries_attempted"`
	Server           sql.NullString         `db:"server"`

	RunStatusDescription string
	RunTimeNative        time.Time
	RunDurationNative    time.Duration
	MessageHTML          template.HTML
	// Fields to control display
	CSSClass string

	// These are not part of the job
	DomainName string
	ServerName string
	Key        string // key of the server that populated this
}

func (h JobHistoryRow) MessagesURL() string {
	u := url.URL{}
	u.Path = path.Join("/", "server", h.Key, "jobs", strings.ToLower(h.JobID.String()), "history", fmt.Sprintf("%d", h.InstanceID))
	return u.String()
}

func (h JobHistoryRow) JobURL() string {
	u := url.URL{}
	u.Path = path.Join("/", "server", h.Key, "jobs", strings.ToLower(h.JobID.String()), "history")
	return u.String()
}

// this will need status
func fetchhistory(ctx context.Context, key string, pool *sql.DB, stmt string, jobid string, parms ...any) ([]JobHistoryRow, error) {
	var jobUUID mssql.UniqueIdentifier
	if jobid != "" {
		err := jobUUID.Scan(jobid)
		if err != nil {
			return []JobHistoryRow{}, err
		}
	}

	e, err := FetchEnvironment(ctx, pool)
	if err != nil {
		return []JobHistoryRow{}, err
	}
	if !e.HasPermission() {
		return []JobHistoryRow{}, &ErrNoPermission{e}
	}

	sqlxdb := sqlx.NewDb(pool, "sqlserver")
	sqlxdb = sqlxdb.Unsafe()
	history := make([]JobHistoryRow, 0)

	sqlparms := []any{}
	if jobid != "" {
		sqlparms = append(sqlparms, jobUUID)
	}
	sqlparms = append(sqlparms, parms...)
	err = sqlxdb.SelectContext(ctx, &history, stmt, sqlparms...)
	if err != nil {
		return []JobHistoryRow{}, err
	}
	for i := range history {
		history[i].Key = key
		switch history[i].RunStatus {
		case 0:
			history[i].RunStatusDescription = "Failed"
		case 1:
			history[i].RunStatusDescription = "Succeeded"
		case 2:
			history[i].RunStatusDescription = "Retry"
		case 3:
			history[i].RunStatusDescription = "Canceled"
		case 4:
			history[i].RunStatusDescription = "In Progress"
		default:
			history[i].RunStatusDescription = fmt.Sprintf("undefined: %d", history[i].RunStatus)
		}

		history[i].RunTimeNative, err = agentTime(history[i].RunDate, history[i].RunTime)
		if err != nil {
			return []JobHistoryRow{}, fmt.Errorf("invalid SQL values run_date=%d run_time=%d", history[i].RunDate, history[i].RunTime)
		}

		history[i].RunDurationNative, err = agentDuration(history[i].RunDuration)
		if err != nil {
			return []JobHistoryRow{}, fmt.Errorf("invalid SQL values run_duration=%d", history[i].RunDuration)
		}
		history[i].MessageHTML = template.HTML(splitAndFormatMessage(history[i].Message.String))
		history[i].DomainName = e.DefaultDomain
		history[i].ServerName = e.ServerName
		history[i].Key = key
	}
	return history, nil
}

// FetchRecentFailures gets jobs in the last four days that have not succeeded
func FetchRecentFailures(ctx context.Context, key string, pool *sql.DB) ([]JobHistoryRow, error) {
	// only failures
	// last 7 days - now to yyyyyymmdd int for a parameter
	dateStr := time.Now().AddDate(0, 0, -7).Format("20060102")
	dateInt, err := strconv.Atoi(dateStr)
	if err != nil {
		return []JobHistoryRow{}, err
	}
	stmt := fmt.Sprintf(`%s
	AND step_id = 0 
	AND sjh.run_date >= @p1
	AND sjh.run_status NOT IN (1) -- not success
	ORDER BY instance_id DESC `, queryHistorySelect)
	return fetchhistory(ctx, key, pool, stmt, "", dateInt)
}

// FetchJobCompletions returns all the completed runs of a job
func FetchJobCompletions(ctx context.Context, key string, pool *sql.DB, jobid string) ([]JobHistoryRow, error) {
	stmt := fmt.Sprintf(`%s
AND step_id = 0 
AND sj.job_id = @p1
ORDER BY instance_id DESC `, queryHistorySelect)
	return fetchhistory(ctx, key, pool, stmt, jobid)
}

// FetchJobMessages returns all the messages for one run of a job
func FetchJobMessages(ctx context.Context, key string, pool *sql.DB, jobid string, instanceid int) ([]JobHistoryRow, error) {
	stmt := fmt.Sprintf(`%s
AND sj.job_id = @p1
AND sjh.instance_id <= @p2
AND sjh.instance_id > (SELECT COALESCE(MAX(instance_id), 0) 
						 FROM msdb.dbo.sysjobhistory
						 WHERE instance_id < @p2
						 AND job_id = @p1
						 AND step_id = 0)
ORDER BY instance_id DESC `, queryHistorySelect)
	return fetchhistory(ctx, key, pool, stmt, jobid, instanceid)
}

// FetchJobMessages returns all the messages for one run of a job
func FetchJobMessagesCurrent(ctx context.Context, key string, pool *sql.DB, jobid string) ([]JobHistoryRow, error) {
	stmt := fmt.Sprintf(`%s
AND sj.job_id = @p1
AND sjh.instance_id > (SELECT MAX(instance_id) 
						 FROM msdb.dbo.sysjobhistory
						 WHERE job_id = @p1
						 AND step_id = 0)
ORDER BY instance_id DESC `, queryHistorySelect)
	return fetchhistory(ctx, key, pool, stmt, jobid)
}

var queryHistorySelect = `
  SET TRANSACTION ISOLATION LEVEL READ UNCOMMITTED; 
  SELECT
  	 sjh.instance_id,
     sj.job_id,
     job_name = sj.[name],
     sjh.step_id,
     sjh.step_name,
     sjh.sql_message_id,
     sjh.sql_severity,
     sjh.[message],
     sjh.run_status,
     sjh.run_date,
     sjh.run_time,
     sjh.run_duration,
     sjh.retries_attempted,
     sjh.[server]
  FROM msdb.dbo.sysjobhistory                sjh
  JOIN msdb.dbo.sysjobs_view sj on sj.job_id = sjh.job_id
  WHERE 1=1
`

func splitAndFormatMessage(message string) string {
	// Define a named regular expression pattern
	pattern := `(?P<msg>\[SQLSTATE\s+\d+\]\s*(\(\w+\s*\d+\)\.?\s*)?)` // Matches individual words; you can customize this pattern as needed
	re := regexp.MustCompile(pattern)

	var result strings.Builder
	lastIndex := 0

	// the first ". " will be the executed by unless the user name has that pattern.
	// Executed as user: NT SERVICE\SQLAgent$SQL2016. The first message...
	message = strings.Replace(message, ". ", ". <br>", 1)
	message = "<span style='font-family: monospace;'>" + message + "</span>"

	// Find all matches and iterate through them
	for _, match := range re.FindAllStringSubmatchIndex(message, -1) {
		// Append the substring before the match
		result.WriteString(message[lastIndex:match[0]])

		// Append the match itself with an HTML break tag
		result.WriteString("<span style='color: silver;'>")
		result.WriteString(message[match[0]:match[1]])
		result.WriteString("</span><br>")

		// Update the last index to the end of the match
		lastIndex = match[1]
	}

	// Append any remaining part of the message
	result.WriteString(message[lastIndex:])

	return result.String()
}
