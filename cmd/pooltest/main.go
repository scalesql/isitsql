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
	ConnStart    time.Time
	SPID         int
	ConnectionID string
}

func (r pollResult) String() string {
	return fmt.Sprintf("server=%s  auth=%s  conn_start=%s  spid=%d  conn_id=%s",
		r.ServerName, r.AuthMethod,
		r.ConnStart.Format("15:04:05"),
		r.SPID,
		r.ConnectionID,
	)
}

func main() {
	var (
		server    string
		port      int
		pingFirst bool
		connLife  time.Duration
		interval  time.Duration
		msf       bool
	)

	flag.StringVar(&server, "s", "", "server/host to connect")
	flag.IntVar(&port, "p", 1433, "port")
	flag.BoolVar(&pingFirst, "ping-first", false, "ping the host first to detect IP changes")
	flag.DurationVar(&connLife, "conn-life", 1*time.Minute, "connection max lifetime")
	flag.DurationVar(&interval, "interval", 1*time.Second, "polling interval")
	flag.BoolVar(&msf, "msf", false, "enable MultiSubnetFailover in connection string")
	flag.Parse()

	if server == "" {
		fmt.Println("Error: -s (server) is required")
		flag.Usage()
		os.Exit(1)
	}

	logf("Starting pool test: server=%s port=%d ping-first=%v conn-life=%v interval=%v msf=%v",
		server, port, pingFirst, connLife, interval, msf)

	db, err := openPool(server, port, connLife, msf)
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
	if pingFirst {
		ips, err := net.DefaultResolver.LookupHost(ctx, server)
		if err != nil {
			logf("WARNING: initial DNS lookup failed: %v", err)
		} else {
			slices.Sort(ips)
			lastIPs = ips
			logf("Initial IPs: %v", lastIPs)
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run immediately on start, then on each tick
	poll(ctx, db, &lastResult)

	for {
		select {
		case <-ctx.Done():
			logf("Shutting down...")
			return
		case <-ticker.C:
			if pingFirst {
				db, lastIPs = checkDNS(ctx, server, port, connLife, msf, db, lastIPs)
			}
			poll(ctx, db, &lastResult)
		}
	}
}

// openPool creates a new *sql.DB connection pool to the given server.
func openPool(server string, port int, connLife time.Duration, msf bool) (*sql.DB, error) {

	query := url.Values{}
	query.Add("app name", "pooltest.exe")
	if msf {
		query.Add("multisubnetfailover", "true")
	}

	u := &url.URL{
		Scheme: "sqlserver",
		// User:   url.UserPassword(username, password),
		Host: server,
		// Path:  instance, // if connecting to an instance instead of a port
		RawQuery: query.Encode(),
	}
	if port != 0 && port != 1433 {
		u.Host += fmt.Sprintf(":%d", port)
	}
	connStr := u.String()
	logf("Connection string: %s", connStr)
	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	db.SetConnMaxLifetime(connLife)
	return db, nil
}

// checkDNS resolves the host IPs and rebuilds the pool if they changed.
func checkDNS(ctx context.Context, server string, port int, connLife time.Duration, msf bool, db *sql.DB, lastIPs []string) (*sql.DB, []string) {
	ips, err := net.DefaultResolver.LookupHost(ctx, server)
	if err != nil {
		logf("ERROR: DNS lookup failed: %v", err)
		return db, lastIPs
	}
	slices.Sort(ips)

	if !slices.Equal(ips, lastIPs) {
		logf("DNS change detected: %v -> %v", lastIPs, ips)
		logf("Rebuilding connection pool")
		db.Close()
		newDB, err := openPool(server, port, connLife, msf)
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
			c.connect_time AS ConnStart,
			@@SPID AS SPID,
			CONVERT(VARCHAR(36), c.connection_id) AS ConnectionID
		FROM sys.dm_exec_sessions s
		JOIN sys.dm_exec_connections c ON c.session_id = s.session_id
		WHERE s.session_id = @@SPID;
	`).Scan(
		&result.ServerName,
		&result.AuthMethod,
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
