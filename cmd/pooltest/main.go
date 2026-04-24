package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/signal"
	"reflect"
	"slices"
	"strings"
	"time"

	mssql "github.com/microsoft/go-mssqldb"
	_ "github.com/microsoft/go-mssqldb/namedpipe"
	_ "github.com/microsoft/go-mssqldb/sharedmemory"
)

type pollResult struct {
	ServerName   string
	AuthMethod   string
	Protocol     string
	ConnStart    time.Time
	SPID         int
	ConnectionID string
}

type poolTestConfig struct {
	Server    string
	Port      int
	Instance  string
	PingFirst bool
	ConnLife  time.Duration
	Interval  time.Duration
	MSF       bool
	Protocol  string
}

func (r pollResult) String() string {
	return fmt.Sprintf("server=%s  auth=%s (%s)  conn_start=%s  spid=%d (%s)",
		r.ServerName, r.AuthMethod, r.Protocol,
		r.ConnStart.Format("15:04:05"),
		r.SPID,
		r.ConnectionID,
	)
}

func main() {
	cfg := poolTestConfig{}

	flag.StringVar(&cfg.Server, "server", "", "server/host to connect")
	flag.IntVar(&cfg.Port, "port", 1433, "port")
	flag.StringVar(&cfg.Instance, "instance", "", "SQL Server instance name (used as URL path)")
	flag.BoolVar(&cfg.PingFirst, "ping", false, "ping the host first to detect IP changes")
	flag.DurationVar(&cfg.ConnLife, "life", 1*time.Minute, "connection max lifetime")
	flag.DurationVar(&cfg.Interval, "interval", 1*time.Second, "polling interval")
	flag.BoolVar(&cfg.MSF, "msf", false, "enable MultiSubnetFailover in connection string")
	flag.StringVar(&cfg.Protocol, "protocol", "", "connection protocol (tcp|np|lpc)")
	flag.Parse()

	if cfg.Server == "" {
		fmt.Println("Error: -s (server) is required")
		flag.Usage()
		os.Exit(1)
	}

	logf("Starting pool test: server=%s port=%d instance=%s ping-first=%v conn-life=%v interval=%v msf=%v",
		cfg.Server, cfg.Port, cfg.Instance, cfg.PingFirst, cfg.ConnLife, cfg.Interval, cfg.MSF)

	db, err := openPool(cfg)
	if err != nil {
		logf("FATAL: failed to open connection pool: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	// Verify the initial connection works before entering the poll loop
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer pingCancel()
	if err := db.PingContext(pingCtx); err != nil {
		logf("FATAL: initial connection failed")
		printError(err)
		os.Exit(1)
	}
	logf("Initial connection successful")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var lastResult pollResult
	var lastIPs []string

	// Resolve and log the initial IPs
	if cfg.PingFirst {
		ips, err := net.DefaultResolver.LookupHost(ctx, cfg.Server)
		if err != nil {
			logf("WARNING: initial DNS lookup failed: %v", err)
		} else {
			slices.Sort(ips)
			lastIPs = ips
			logf("Initial IPs: %v", lastIPs)
		}
	}

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	// Run immediately on start, then on each tick
	poll(ctx, db, &lastResult)

	for {
		select {
		case <-ctx.Done():
			logf("Shutting down...")
			return
		case <-ticker.C:
			if cfg.PingFirst {
				db, lastIPs = checkDNS(ctx, cfg, db, lastIPs)
			}
			poll(ctx, db, &lastResult)
		}
	}
}

// openPool creates a new *sql.DB connection pool to the given server.
func openPool(cfg poolTestConfig) (*sql.DB, error) {

	query := url.Values{}
	query.Add("app name", "pooltest.exe")
	if cfg.MSF {
		query.Add("multisubnetfailover", "true")
	}
	if cfg.Protocol != "" {
		query.Add("protocol", cfg.Protocol)
	}

	u := &url.URL{
		Scheme: "sqlserver",
		// User:   url.UserPassword(username, password),
		Host:     cfg.Server,
		RawQuery: query.Encode(),
	}
	if cfg.Instance != "" {
		u.Path = cfg.Instance
	}
	if cfg.Port != 0 && cfg.Port != 1433 {
		u.Host += fmt.Sprintf(":%d", cfg.Port)
	}

	connStr := u.String()
	logf("Connection string: %s", connStr)

	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	db.SetConnMaxLifetime(cfg.ConnLife)
	return db, nil
}

// checkDNS resolves the host IPs and rebuilds the pool if they changed.
func checkDNS(ctx context.Context, cfg poolTestConfig, db *sql.DB, lastIPs []string) (*sql.DB, []string) {
	ips, err := net.DefaultResolver.LookupHost(ctx, cfg.Server)
	if err != nil {
		logf("ERROR: DNS lookup failed: %v", err)
		return db, lastIPs
	}
	slices.Sort(ips)

	if !slices.Equal(ips, lastIPs) {
		logf("DNS change detected: %v -> %v", lastIPs, ips)
		logf("Rebuilding connection pool")
		db.Close()
		newDB, err := openPool(cfg)
		if err != nil {
			logf("ERROR: failed to rebuild pool: %v", err)
			// Return old (closed) db; next poll will surface the error
			return db, ips
		}
		return newDB, ips
	}
	return db, lastIPs
}

// poll queries the SQL Server and prints results if they changed.
func poll(ctx context.Context, db *sql.DB, last *pollResult) {
	queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var result pollResult
	err := db.QueryRowContext(queryCtx, `
		SELECT 
			@@SERVERNAME AS ServerName,
			CONVERT(VARCHAR(100), CONNECTIONPROPERTY('auth_scheme')) AS AuthMethod,
			CONVERT(VARCHAR(20), c.net_transport) AS Protocol,
			c.connect_time AS ConnStart,
			@@SPID AS SPID,
			CONVERT(VARCHAR(36), c.connection_id) AS ConnectionID
		FROM sys.dm_exec_sessions s
		JOIN sys.dm_exec_connections c ON c.session_id = s.session_id
		WHERE s.session_id = @@SPID;
	`).Scan(
		&result.ServerName,
		&result.AuthMethod,
		&result.Protocol,
		&result.ConnStart,
		&result.SPID,
		&result.ConnectionID,
	)
	if err != nil {
		logf("ERROR: poll query failed")
		printError(err)
		return
	}

	if !reflect.DeepEqual(result, *last) {
		logf("CHANGED: %s", result)
		*last = result
	}
}

// printError prints detailed error information including type, unwrapped type,
// mssql.Error details, RetryableError status, and driver.ErrBadConn status.
func printError(err error) {
	logf("  err: %s", err.Error())
	logf("  error type: %T", err)

	unwrapped := errors.Unwrap(err)
	if unwrapped != nil {
		logf("  unwrapped type: %T", unwrapped)
	}

	var mssqlErr mssql.Error
	if errors.As(err, &mssqlErr) {
		logf("  mssql.Error:")
		logf("    Number:     %d", mssqlErr.Number)
		logf("    State:      %d", mssqlErr.State)
		logf("    Class:      %d", mssqlErr.Class)
		logf("    Message:    %s", mssqlErr.Message)
		logf("    ServerName: %s", mssqlErr.ServerName)
		if len(mssqlErr.All) > 0 {
			logf("    All errors (%d):", len(mssqlErr.All))
			for i, e := range mssqlErr.All {
				logf("      [%d] Number=%d State=%d Class=%d Message=%s",
					i, e.Number, e.State, e.Class, e.Message)
			}
		}
	}

	var retryable mssql.RetryableError
	if errors.As(err, &retryable) {
		logf("  ** RetryableError **")
	}

	if errors.Is(err, driver.ErrBadConn) {
		logf("  ** driver.ErrBadConn **")
	}
}

// logf prints a timestamped message.
func logf(format string, args ...any) {
	ts := time.Now().Format("15:04:05")
	msg := fmt.Sprintf(format, args...)
	// Indent continuation lines under the timestamp
	msg = strings.ReplaceAll(msg, "\n", "\n         ")
	fmt.Printf("%s %s\n", ts, msg)
}
