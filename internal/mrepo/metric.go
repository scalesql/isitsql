package mrepo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/scalesql/isitsql/internal/waitring"
)

const timeout = 3 * time.Second

// WriteMetrics writes the collected metrics to the repository.
func WriteMetrics(ts time.Time, m map[string]any) error {
	if pool == nil {
		return nil
	}
	m["metric_time"] = ts
	query := insertFromMap("dbo", "server_metric", m)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel() // ensure the context is cancelled
	_, err := pool.NamedExecContext(ctx, query, m)
	return err
}

// WriteWaits writes the collected waits to the repository.
func WriteWaits(key, name string, w waitring.WaitList) error {
	if pool == nil || len(w.Waits) == 0 {
		return nil
	}
	rows := []map[string]any{}
	for wait, tm := range w.Waits {
		row := map[string]any{
			"metric_time":  w.TS,
			"server_key":   key,
			"server_name":  name,
			"wait_type":    wait,
			"wait_time_ms": tm,
		}
		rows = append(rows, row)
	}

	query := "INSERT dbo.server_wait (metric_time, server_key, server_name, wait_type, wait_time_ms) VALUES (:metric_time, :server_key, :server_name, :wait_type, :wait_time_ms)"
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel() // ensure the context is cancelled
	_, err := pool.NamedExecContext(ctx, query, rows)
	return err
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
