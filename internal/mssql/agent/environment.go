package agent

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Environment holds information about the server that Agent is attached to and
// certain permissions
type Environment struct {
	SystemUser    string `db:"system_user"`
	DefaultDomain string `db:"default_domain"`
	ServerName    string `db:"server_name"`
	SysAdmin      bool   `db:"is_sysadmin"`
	AgentReader   bool   `db:"is_agentreader"`
	DataReader    bool   `db:"is_datareader"`
	DBOwner       bool   `db:"is_dbowner"`
}

type ErrNoPermission struct {
	Environment
}

func (nop *ErrNoPermission) Error() string {
	return fmt.Sprintf("%s\\%s: '%s' needs SQLAgentReaderRole and db_datareader in msdb",
		nop.DefaultDomain, nop.ServerName, nop.SystemUser)
}

// FetchEnvironment from the connection
func FetchEnvironment(ctx context.Context, pool *sql.DB) (Environment, error) {
	sqldb := sqlx.NewDb(pool, "sqlserver")
	var e Environment
	err := sqldb.GetContext(ctx, &e, connInfoQuery)
	return e, err
}

// HasPermission checks if the current connection has permission to query
// job history.
func (e Environment) HasPermission() bool {
	if e.SysAdmin || e.DBOwner {
		return true
	}
	if e.AgentReader && e.DataReader {
		return true
	}
	return false
}

var connInfoQuery = `
USE [msdb];

SELECT 
	COALESCE(SYSTEM_USER, '') AS [system_user],
	COALESCE(DEFAULT_DOMAIN(), '') AS [default_domain],
	COALESCE(@@SERVERNAME, '') AS [server_name],
	CAST(ISNULL(IS_SRVROLEMEMBER(N'sysadmin'), 0) AS BIT) AS is_sysadmin,
	CAST(ISNULL(IS_MEMBER(N'SQLAgentReaderRole'), 0) AS BIT) AS is_agentreader,
	CAST(ISNULL(IS_MEMBER(N'db_datareader'), 0) AS BIT) AS is_datareader,
	CAST(ISNULL(IS_MEMBER(N'db_owner'), 0) AS BIT) AS is_dbowner ; 
`
