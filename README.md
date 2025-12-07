# Is It SQL
A simple SQL Server monitoring tool to determine if SQL Server is causing the current problem.

This is designed to allow a moderately technical person to determine if SQL Server is likely the cause of any current issue.  A screenshot can be sent to a Database Administrator to decide if further follow up is needed.  It runs entirely in memory and doesn't require SQL Server itself to run.  It only has a one-hour history.


![IsItSQL screenshot](assets/img/screenshot-20250713.png "Is It SQL screenshot")

## Features
* Show running sessions and any blocking
* Highlight  unreachable machines or unhealthy Availability Groups on every page
* Show CPU usage for SQL Server and other services
* Show batches per second, disk IO, and current Waits
* List databases, their size, and status
* List databases with missing backups.  This is also available as a JSON file.
* List Availability Groups and their state
* Show SQL Server Agent jobs and their status
* List recent SQL Server errors when the Extended Event session is created
* Show basic server information such as version, edition, and IP addresses
* Show database snapshots
* Show Database Mirroring status
* Download all SQL Servers to CSV showing version, edition, etc.

Please see the [Documentation](static/docs/README.md) for more details.

## What's New

### vNext 
* Improve MarkDown formatting in the server information page
* Ignore XE_LIVE_TARGET_TVF wait type
* Lots of HTML and forms cleanup

### 2.5 (August 2025) 
* Option to store key server metrics in a SQL Server Database
* Support protocols for connections such as "tcp:fqdn.com".  It supports "tcp", "np" (named pipes), and "lpc" (shared memory).  The default is "tcp".  The Server Information page has a link to the Server Connection page that will show the connection details.
* Upgraded jQuery and Bootstrap.  This no longer supports Internet Explorer.
* Ignore distributed Availability Groups for now
* Added an Outbound Connections page at `/connections` that shows the FQDN, @@SERVERNAME, protocol, and authorization scheme for all the SQL Server connections.
* Improved Javascript error handling
* Cleaned up nesting in the HTML templates

### 2.3 (January 2025)
* Added SQL Server Agent jobs.  The IsItSQL service account needs `db_datareader` and `SQLAgentReaderRole` in `msdb`.
* Added Prometheus metrics.  See the About page for the link.
* Reduced locking
* Fixed issue with charts always in UTC
* Updated GO version and all dependencies
