---
applyTo: '**'
---
Coding standards, domain knowledge, and preferences that AI should follow.

## Project Overview

IsItSQL is a **Windows service** written in **Go** that monitors Microsoft SQL Server instances. It provides a web-based dashboard (Bootstrap 5) for viewing server health, availability groups, backups, sessions, waits, and jobs. It can also run interactively for debugging.

## Architecture

- **`app/`** — Main application package: web server, SQL polling, service management, configuration, and global state. Most application logic lives here.
- **`internal/`** — Internal packages for domain-specific logic (HADR, settings, metrics repository, wait tracking, session management, connection strings, tags, etc.).
- **`static/`** — Embedded static assets (CSS, JS, icons, fonts, HTML templates) served by the web server. Uses Go `embed` with an option to serve from the local filesystem during development.
- **`cmd/`** — Standalone command-line utilities (connection tests, linting, crawling, etc.).
- **`config/`** — Runtime configuration directory (`settings.json`).
- **`cache/`** — JSON cache files for server state.

## Language & Frameworks

- **Go 1.23+** with modules (`go.mod`)
- **`sqlx`** for SQL Server database access via `github.com/microsoft/go-mssqldb`
- **`goose`** for database migrations (embedded SQL files in `internal/mrepo/migrations/`)
- **`logrus`** for structured logging
- **`pkg/errors`** for error wrapping
- **`routegroup`** for HTTP routing (from `go-pkgz/routegroup`)
- **`gorilla/sessions`** for session management
- **Bootstrap 5.2.3** for the web UI (no Glyphicons — use Bootstrap Icons or SVG icons)
- **jQuery + tablesorter** for sortable tables
- **TOML** (`pelletier/go-toml/v2`) for `isitsql.toml` configuration
- **HCL** (`hashicorp/hcl/v2`) for server connection files
- **`testify/assert`** for test assertions

## Configuration

- **`settings.json`** (in `config/`) — App settings (port, security policy, credentials, session key). Read/written by `internal/settings` package. Encrypted using Windows DPAPI.
- **`isitsql.toml`** (in exe directory) — Repository database connection settings (`[repository]` section with `host`, `database`, optional `credential`). Parsed by `app/toml_config.go` into `IsItSQLTOML` struct using `pelletier/go-toml/v2`.
- **HCL files** (in `servers_xxx/` or similar) — Server connection definitions when running in `ModeFile`. Watched for changes with `radovskyb/watcher`.
- **Config modes**: `ModeGUI` (settings managed via web UI) or `ModeFile` (settings from HCL files).

## Build Tags

- `dev` build tag enables the `DEV` constant (`app/build_dev.go`), which injects test data and changes polling behavior. Production builds use `app/build_normal.go` with `DEV = false`.

## Coding Conventions

### Go
- Use `context.WithTimeout` for all database calls. The standard timeout is 3 seconds (`internal/mrepo`).
- Wrap errors with `errors.Wrap(err, "context")` from `github.com/pkg/errors`.
- Use `fmt.Errorf("message: %w", err)` for errors in `internal/` packages (newer style is acceptable).
- Use `logrus` for logging. Use `logrus.Error`, `logrus.Debug`, `logrus.Trace` appropriately.
- Use `WinLogln` / `WinLogf` for messages that should appear in the Windows Event Log.
- Tests use `github.com/stretchr/testify/assert`. Test files are in the same package as the code they test.
- Prefer `sqlx.NamedExecContext` with named parameters (`:paramName`) for SQL inserts.
- Use `embed.FS` for embedded files (migrations, static assets).
- Global state uses `sync.RWMutex` for thread safety.
- GUID/UUID values should be uppercased and validated.

### HTML Templates
- Templates use Go `html/template` with `{{ define "content" }}` blocks.
- Use **Bootstrap 5** classes. Do **not** use Bootstrap 3/4 patterns:
  - Dropdowns: use `form-select` (not `form-control` on `<select>`)
  - Offsets: use `offset-sm-5` (not `col-sm-offset-5`)
  - Icons: use Bootstrap Icons (`bi bi-*`) or SVG files from `/static/icons/` (no Glyphicons)
  - Spacing: use Bootstrap utility classes (`mt-3`, `me-2`, `gap-2`, `text-decoration-none`)
  - Navbar: uses `navbar-expand-xl` breakpoint
- The base template is `static/templates/base.html`.

### CSS
- Custom styles go in `static/css/isitsql.css`.
- Base font size is 14px (set on `html`).

### SQL
- Target is Microsoft SQL Server.
- Migration files use `goose` format in `internal/mrepo/migrations/` (e.g., `0001_metric.sql`).
- Schema version table is `isitsql_schema_version`.
- Use `[schema].[table]` bracket notation for SQL Server identifiers.

## Windows Service

- The application runs as a Windows service (`IsItSql`) using `golang.org/x/sys/windows/svc`.
- Supports `install`, `remove`, `start`, `stop`, `pause`, `continue`, and `debug` commands.
- Panic recovery is handled by `internal/failure` package with `defer failure.HandlePanic()`.

## Security Notes

- Settings page access is controlled by `SecurityPolicyType` (`open`, `localhost`).
- Credentials are encrypted with Windows DPAPI before storage.
- LDAP integration exists for authentication (`internal/ad`, `go-ldap/ldap`).
- Prometheus metrics are exposed for monitoring.