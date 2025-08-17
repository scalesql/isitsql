package mrepo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/scalesql/isitsql/internal/waitring"
)

// WriteMetrics writes the collected metrics to the repository.
func (r *Repository) WriteMetrics(ts time.Time, m map[string]any) {
	if r == nil {
		return
	}
	if r.pool == nil {
		return
	}
	m["ts"] = ts
	m["ts_date"] = truncateDate(ts) // truncate to date
	m["ts_time"] = ts.Truncate(time.Minute)
	query := insertFromMap("dbo", "server_metric", m)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel() // ensure the context is cancelled
	_, err := r.pool.NamedExecContext(ctx, query, m)
	r.handleError(err)
}

// WriteWaits writes the collected waits to the repository.
func (r *Repository) WriteWaits(key, server, table string, start time.Time, w waitring.WaitList) {
	if r == nil {
		return
	}
	if r.pool == nil || len(w.Waits) == 0 {
		return
	}
	rows := []map[string]any{}
	for wait, tm := range w.Waits {
		if tm < 1000 { // skip waits less than 1 second
			continue
		}
		row := map[string]any{
			"ts":            w.TS,
			"ts_date":       truncateDate(w.TS), // truncate to date
			"ts_time":       w.TS.Truncate(time.Minute),
			"server_key":    key,
			"server_name":   server,
			"wait_type":     wait,
			"wait_time_sec": tm / 1000, // convert to seconds
			"server_start":  start,
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return // no waits to write
	}

	query := fmt.Sprintf(`INSERT [dbo].[%s] (ts, ts_date, ts_time, server_key, server_name, server_start, wait_type, wait_time_sec) 
						VALUES (:ts, :ts_date, :ts_time, :server_key, :server_name, :server_start, :wait_type, :wait_time_sec)`, table)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel() // ensure the context is cancelled
	_, err := r.pool.NamedExecContext(ctx, query, rows)
	r.handleError(err)
}

func insertFromMap(schema, table string, data map[string]any) string {
	var columns []string
	var namedParams []string
	for key := range data {
		columns = append(columns, key)
		namedParams = append(namedParams, ":"+key) // Using named parameters for sqlx
	}

	columnsStr := strings.Join(columns, ", ")
	namedParamsStr := strings.Join(namedParams, ", ")

	query := fmt.Sprintf("INSERT INTO [%s].[%s] (%s) VALUES (%s)", schema, table, columnsStr, namedParamsStr)
	return query
}

// truncateDate returns the date at midnight in the original time zone.
func truncateDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
