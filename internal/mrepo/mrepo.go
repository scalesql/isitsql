// Package mrepo provides a repository for metrics and waits in SQL Server.
package mrepo

import (
	"embed"
	"time"
)

//go:embed migrations/*.sql
var fs embed.FS

const timeout = 3 * time.Second
