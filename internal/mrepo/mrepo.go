// Package mrepo provides a repository for metrics and waits in SQL Server.
package mrepo

import (
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var fs embed.FS

var pool *sqlx.DB

const timeout = 3 * time.Second

/* TODO: If setup fails, launch a GO routine that attempts to connect every minute.
   This will need a package level mutex.
   TODO: Only write metrics for certain tags
*/

func Setup(fqdn, database string, log goose.Logger) error {
	if fqdn == "" || database == "" {
		return fmt.Errorf("fqdn and database must be provided")
	}

	// connect to the database and set the package level connection
	connstr, err := makeconnstr(fqdn, database, "", "")
	if err != nil {
		return fmt.Errorf("connstr: %w", err)
	}
	db, err := sql.Open("sqlserver", connstr.String())
	if err != nil {
		return err
	}

	goose.SetLogger(log)
	goose.SetBaseFS(fs)
	err = goose.SetDialect(string(goose.DialectMSSQL))
	if err != nil {
		return fmt.Errorf("goose.setdialect: %w", err)
	}
	goose.SetTableName("isitsql_schema_version")
	err = goose.Up(db, "migrations")
	if err != nil {
		return fmt.Errorf("goose.up: %w", err)
	}
	// save the connection pool as a sqlx pool
	pool = sqlx.NewDb(db, "sqlserver")
	return nil
}

func makeconnstr(server, database, user, password string) (url.URL, error) {
	host, instance, port := parseFQDN(server)
	if host == "" {
		return url.URL{}, fmt.Errorf("invalid server: %s", server)
	}

	query := url.Values{}
	query.Add("app name", "isitsql.exe")
	query.Add("database", database)
	// query.Add("encrypt", "optional")

	u := url.URL{
		Scheme:   "sqlserver",
		Host:     host,
		RawQuery: query.Encode(),
	}
	if instance != "" {
		u.Path = instance
	}
	if port != 0 {
		u.Host = fmt.Sprintf("%s:%d", host, port)
	}
	if user != "" && password != "" {
		u.User = url.UserPassword(user, password)
	}
	return u, nil
}

// parseFQDN splits a host\instance with an optional port
func parseFQDN(s string) (host, instance string, port int) {
	var err error
	parts := strings.FieldsFunc(s, hostSplitter)
	host = parts[0]
	if len(parts) == 1 {
		return host, "", 0
	}
	if len(parts) == 2 {
		port, err = strconv.Atoi(parts[1])
		if err == nil {
			return host, "", port
		}
		instance = parts[1]
		return host, instance, 0
	}
	if len(parts) == 3 {
		instance = parts[1]
		port, _ = strconv.Atoi(parts[2])
		return host, instance, port
	}

	return host, instance, port
}

// hostSplitter splits a string on :,\ and is used to split FQDN names
func hostSplitter(r rune) bool {
	return r == ':' || r == ',' || r == '\\'
}
