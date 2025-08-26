

# IsItSQL Quick Start

## 1\. Launch the application

The simplest way is to double-click the `isitsql.exe` file in the directory. You can also open a console window and run the application. Any errors will be displayed in the console window. A simple Control-C (or Break) will exit the application.

**After launching, navigate your browser to [http://localhost:8143](http://localhost:8143) to view the monitor.** It polls each server once per minute in the background. Pages refrehes automatically every minute.

## 2\. Adding a server to monitor

Navigate to the [Add Server](http://localhost:8143/settings/servers/add) page via the Gears icon in the upper right corner. I suggest starting with a server you can connect to via a trusted connection. Please remember that this application is running as you. Enter the server name in the FQDN column and save it.

Your server should be polled and available when the page refreshes back to the list of servers to monitor.

_Note:_ If you enter a server using either a SQL Server login or custom connection string and then change to run the application as a service you will need to re-enter this information because it will be encrypted.

# Table of Contents

1.  [What's New](#whatsnew)
2.  [Required Permissions](#permissions)
3.  [Running as a Service](#service)
3.  [Settings](#settings)
3.  [Features](#features)
4.  [Other Notes](#other)
5.  [Connection Strings](#connectionstrings)
6.  [Bulk Adding Servers](#bulkadd)
8.  [Feedback and Known Issues](#feedback)
9.  [Extended Events](#xe)
10. [Server Documentation](#docs)
11. [Store Metrics in a SQL Server Database](#repository)
11. [Previous Releases](#previous)

<a id="whatsnew"></a>

## What's New

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

### 2.2 (October 2024)
* Added Dynamic Waits
* Clarified whether times were from the SQL Server or IsItSQL

### 2.1 (February 2024)
* Display server documentation.  Mardown documentation can be associated with each server and will be displayed in a `Docs` tab.  See the [Server Documentation](#docs) section for more information.
* Capture the IP address of the server.  It does this using `sys.dm_tcp_listener_states` and `sys.dm_exec_connections`.  The IP address of each server is displayed on the `Docs` tab.  All IP addresses are displayed on Global -> IP Addresses page.


### 2.0.4 (13 February 2024)
* Reduce memory usage.  Legacy waits are only read off disk when requested.
* Added `MEMORY_ALLOCATION_EXT` wait
* Fixed display of the time since a snapshot was created
* Standardized date displays on various pages


### 2.0.3 (20 January 2024)
* Servers can be listed in user editable files.  See the documentation in the `optional` folder.  This is hepful if you want IsItSQL running in multiple data centers sharing the same list of servers.
* Support non-GUID server keys.  If you put the servers in a configuration file, you can have URLs like `/server/db-txn`. 
* Added a Server Connection Detail page at `/server/:server_key/conn`
* Improve reporting of `tempdb` size
* Limit support for Availability Groups to SQL Server 2014 and higher
* Added a Connection Test utility
* The Server Detail page now shows the Windows version.  The SQL Server Versions page at `/versions` shows the operating system.  Downloading that page as CSV adds the architecture, install date, cores, and memory.
* SQL Servers in containers ignore the "other" CPU percentage
* Add a `/memory` page that shows OS memory, SQL Server memory and the amount free
* Waits are now polled in real time and only waits on user sessions are displayed by default.  See the [Features](#features) section.
* Switched from the ODBC driver to the native Microsoft GO driver to connect to SQL Server.
* Date and time formatting are improved throughout the application
* Improved the reporting of blocked sessions.  This should make it easier to identify the root cause of blocking.
* More polling queries timeout if they get blocked.  This is mostly AG queries while adding or removing nodes or databases.

### 1.7.14 (8 March 2023)
* On the databases page,
  * Hide the log backups if in SIMPLE recovery
  * (BETA) Show the AG status, send log size, and redo log size for the database
* Change various refresh intervals
* Exclude PWAIT_EXTENSIBILITY_CLEANUP_TASK
* Edit server list includes link to the stats
* Most pages display the server name in gray next to the friendly name
* Most pages have a filter on them
* Added support for ODBC18 and prioritized the ODBC drivers
* Support optional encryption attributeds on newer drivers
* Updated to GO 1.19
* On the AG page, add a link back to the primary node


### 1.7.11 (18 August 2022)
* Sort the Availability Group list consistently
* Add SQL Server 2022 support
* Improve support for SQL Server on Linux
* Ensure log directory is created
* Better error handling in various web pages


### 1.7.9 (2 August 2022)
* Bug fix for SQL Server 2008 Activity page
* Add Availability Group display names.  See [Features](#features) below.

### 1.7.6 (28 July 2022)
* Add support for newer ODBC drivers
* Better line ending on Windows for the log file
* Support `ignoredBackups.csv` or `ignoredBackups.txt`
* AG page displays Listener names instead of AG names
* Improve clean up for the NDJSON files that cache metrics

### 1.7.2 (28 June 2022)
* Update to latest ODBC drivers
* Activity query shows orphaned sessions that are blocking

### 1.7 (8 May 2022)

* Save the last hour of performance data on restart.  These files are stored in the `cache` directory.
* The log is a little more human readable.  It supports both debug and trace log levels that can be set in the settings file.
* Purging old log files actually purges the files
* Changed the `tags` menu so that the submenus are at the top
* Waits now show a a per minute value.  This works much better if the server can't poll for a minute or two.
* Improved error handling throughout
* Add a `usage` page that shows server, edition, cores, and various values to estimate cores used over the last hour.  This also include a CSV download.

### 1.6 (22 February 2022)

* High send or recieve queues in Availability Groups are shown as alerts or warnings.  See the [Features](#features) section for more details.
* Better logic to find the installed date
* Server status and metrics are saved between restarts in JSON files in the `cache` folder
* CPU monitoring is incremental.  Previously it would ask the server for the last 60 minutes of CPU activity and just use that.  Now it only uses values after the last value it already had.  This mainly helps on the active node of an Availability Group during failover.
* Minor improvements to memory usage


### 1.4 (7 September 2021)

* Added an Application menu with links to Monitored Servers, Settings, Credentials, the Application Log, and a few other pages
* Added an About page that links to lots of internal pages
* Fixed a bug where ignored backups with trailing spaces wouldn't ignore
* Added a page to list all database snapshots and their age

### 1.3.9 (8 August 2021)

* Supports domain login to edit settings
* Most server displays now use the max memory if it is configured
* A "Versions" page lists all the versions and editions at http://localhost:8143/versions.  You can also download this as CSV.  The CSV includes tags, cores, edition, and memory so you can do rudimentary licensing validation.
* The [Log Events](http://localhost:8143/log) now includes a filter.  You can reach this by clicking on the "Refreshed" label on the upper right.
* If servers have the proper extended event session, those events can be filtered. If you haven't created the Extended Event session, the code to do so is listed on the page.  Please see the Extended Events section in this document.
* The Availability Group page now has JSON output at http://localhost:8143/ag/json.  This should be suitable for generating alerts.


<a id="permissions"></a>

## Required Permissions

The application requires the following permissions for the login that is running the service or executable or the login that is used to connect to the server:

*   `VIEW SERVER STATE` - View basic server information such as CPU, disk I/O, waits, and version.
*   `VIEW ANY DEFINITION` - View database details such as name, status, and size.
*   `msdb` database: `SQLAgentReaderRole` and `db_datareader` database role - View SQL Server Agent Job information.  (It also works with `sysadmin` or `db_owner` but those aren't recommended.)
* Permissions can be easily granted using [SQLDSC](https://www.sqldsc.com/)

<a id="service"></a>

## Running as a Service

There isn't an installer but it is very easy to configure this as a service. Ideally this service will run as a domain service account. Please stop the application before completing these steps.

1.  Identify the domain account
2.  Launch SECPOL.MSC, Navigate to Local Policies -> User Rights Assignment and add the account to the Log on as a Service policy
3.  Grant the service account MODIFY permission on this directory
4.  Open an _Administrator console window_ in this directory and run `isitsql.exe install`. This installs the executable as a service.
5.  Open the Local Services control panel and (1) change the service to run as the service account and (2) start automatically.
6.  Start the service. It will create a log directory here for error logs.
7.  If the application will be accessed from another machine, verify the Windows Firewall allows inbound access on port 8143.

**Note: The service account is used to encrypt key data in the JSON files in the config directory. If the service account is changed, this information will need to be reentered.**

Upgrading is as simple as stopping the service, copying a new executable, and restarting the service.

<a id="settings"></a>

## Settings

The [Settings Page](http://localhost:8143/settings) has the following options.

* Concurrent Pollers should be left at zero.
* It listens on port 8143 by default.  Changing this requires a restart.
* The logo on the upper-left is a link to the "Home Page" field.  This defaults to `/`.  I typically override it to a tag of my choosing: `/tag/prod`.  You can always get back to all the server by choosing Tags -> All Servers from the menu.
* You can configure the threshold for backup alerts.  They default to 36 hours for full and 90 minutes for logs.

### Settings - Security

There are three options to control who can change settings in the application:
* Save from any client
* Only Save from localhost (must RDP to the server to edit settings)
* Settings Page Admin Group.  You must be a member of this group to save the settings page.

Domain group membership is **experimental**.  The Domain Group must be entered here.  You can log in at `http://localhost:8143/login`.  The user must be entered in the form `user@domain.com`.  

<a id="features"></a>

## Features

### Availability Group Alerts

IsItSQL provides simple monitoring for Availability Group latency.  There are two settings in `./config/settings.json`:

```
"ag_alert_mb": 10000
"ag_warn_mb": 1000
```

If an AG send or recive queue is over the warn threshold, the screen will show a warning.  If they are above the alert threshold, it will show an alert.

AG warnings and alerts are available in JSON form at `http://localhost:8143/ag/json`.  This will list any server whose status isn't healthy or that has latency.

### Availability Group Display Names

IsItSQL displays the Listeners on the Availability Group page.  We can override this using the `config/ag_names.csv` file.  This file is only read on startup.  It looks like this:

```csv
# Domain, AG_Name, Friendly_Name
PROD, AG1, db-txn.static.loc
```

The second field can be an Availability Group name or a Listener name.  This is used if you have static DNS entries that point to Availability Group Listeners.  It will also use the Display Name in any alerts that are displayed.

### Waits
Prior to 2.0, waits were captured every minute from `sys.dm_os_wait_stats` which means we only saw them when the wait ended.  Starting in 2.0, waits are polled every second from running processes and updated on the page every minute.  

This means we only see significant for user sessions and don't include waits for background processes.  This shows fewer waits but they are more timely and actionable.

There is a server wait page at `/server/:server_key/w2` that compares the two waits.  

<a id="other"></a>

## Other Notes

1.  To remove the service, stop the service, and type `isitsql.exe remove` from an Administrator console.
2.  To update the service, stop the service, repace the executable and restart the service.
3.  You can see other command line options by typing `isitsql.exe /?`. There aren't many.

1.  The dashboard displays the three servers tagged with "dashboard". You can display other servers by hacking the URL. For example, http://localhost:8143/dashboard/{GUID #1}/{GUID #2}/{GUID #3} will display those three specific servers.
2. Pressing F11 in the browser will remove all chrome and display a nice dashboard.
6. The service writes JSON files with server details and metrics in the `cache` folder.  These are used for history between restarts.

<a id="connectionstrings"></a>

## Connection Strings

_You shouldn't need to enter connection strings very often. This is here in case you do._

The application will suggest a driver on startup. You can find the drivers installed by opening the ODBC data sources and looking at the Drivers tab. If you need to choose a driver I'd suggest the following priority:

1.  {[SQL Server Native Client 11.0](https://www.microsoft.com/en-us/download/confirmation.aspx?id=29065)} Click on install instructions and look for Microsoft "SQL Server 2012 Native Client". This is also the [version that ships with SQL Server 2016](https://msdn.microsoft.com/en-us/library/ms131321.aspx).
2.  {[ODBC Driver 11 for SQL Server](https://www.microsoft.com/en-us/download/details.aspx?id=36434)}
3.  {SQL Server}. Most servers have the generic SQL Server ODBC driver installed. It works but won't support some of the more advanced connection string settings.

### Sample Connection Strings

*   Driver={SQL Server Native Client 11.0};Server=127.0.0.1;Database=tempdb;Trusted_Connection=Yes;App=IsItSql;
*   Driver={SQL Server Native Client 11.0};Server=MyServer;Database=tempdb;Trusted_Connection=Yes;App=IsItSql;
*   Driver={SQL Server Native Client 11.0};Server=127.0.0.1,1433;Database=tempdb;Trusted_Connection=Yes;App=IsItSql;
*   Driver={ODBC Driver 11 for SQL Server};Server=127.0.0.1;Database=tempdb;uid=test;pwd=test;App=IsItSql;
*   Driver={SQL Server};Server=myserver.mydomain.fqdn;Database=tempdb;uid=test;pwd=test;App=IsItSql;

<a id="bulkadd"></a>

## Bulk Adding servers

The application tries to import any servers found in servers.txt at start up. We can take advantage of this to bulk import servers.

This is a CSV file with up to three values: server, "display name", and tags. The server can be a name (NetBIOS, FQDN, IP, etc.) or a connection string. If the server name specifies a port, (ex. MyServer,1433) it should be enclosed in quotes. If you use just a name it will use a trusted connection. Connection strings should always be in quotes.

If you don't include a display name it will use the server name.

Tags is a comma separated list of tags. Obviously it needs to be enclosed in quotes.

<a id="xe"></a>

## Extended Event Session

Each server page looks for an Extended Event session named "ErrorSession". It expects a session holding SQL Server errors in a ring buffer. That session should be defined like this:

    IF EXISTS (select * from sys.server_event_sessions where [name] = 'ErrorSession')
      ALTER EVENT SESSION ErrorSession ON SERVER STATE = STOP

    IF EXISTS (select * from sys.server_event_sessions where [name] = 'ErrorSession')
      DROP EVENT SESSION ErrorSession ON SERVER 

    CREATE EVENT SESSION ErrorSession ON SERVER 
      ADD EVENT sqlserver.error_reported            
      -- collect failed SQL statement, the SQL stack that led to the error, 
      -- the database id in which the error happened and the username that ran the statement 

      (
          ACTION (sqlserver.sql_text, sqlserver.tsql_stack, sqlserver.database_id, 
        sqlserver.username, sqlserver.client_app_name, sqlserver.client_hostname, package0.collect_system_time)
          WHERE severity >= 14 and error_number <> 2557 and error_number <> 17830
      )  
      ADD TARGET package0.ring_buffer    
          (SET max_memory = 1024)
    WITH (max_dispatch_latency = 1 seconds, EVENT_RETENTION_MODE = ALLOW_SINGLE_EVENT_LOSS, STARTUP_STATE = ON)

    IF EXISTS (select * from sys.server_event_sessions where [name] = 'ErrorSession')
      ALTER EVENT SESSION ErrorSession ON SERVER STATE = START

<a id="repository"></a>

## Store Server Metrics in a SQL Server Database
As of version 2.5, IsItSQL can store key metrics in a SQL Server database.  These are stored in three tables:
* `server_metric` - stores basic metrics such as cpu usage, cores, memory, disk usage, etc.
* `request_wait` - stores the dynamic waits
* `server_wait` - stores the server level waits

This is configured using a TOML file named `isitsql.toml` in the same folder as the executable.  The format is:

```toml
[repository]
host = "D40\\SQL2016"
database = "IsItSQL"
# credential = "isitsql"
```

* Any backslash needs to be escaped
* It uses trusted authentication unless a credential name is specified
* The database must already exist
* The service account (or credential) must have the `db_owner` role in the database to create the objects and write data
* IsItSQL will create the needed tables at startup
* The Repository database server must be SQL Server 2016 or higher

<a id="docs"></a>

## Server Documentation
IsItSQL can read Markdown documentation for each server and display it as HTML.  Visiting the `Docs` tab loads any related documentation.  IsItSQL searches in a `docs` folder in the same folder as the executable.  It will search in the root of `docs` and in a subfolder matching the server's domain (see below).  It will search for the following patterns:

* *Friendly Name*.  If you set the friendly name as `db-txn`, it wil search for `db-txn.md`.
* *FQDN*.  Searches for `fqdn.md`.  This can be a static DNS, server name, IP address, etc.
* *Computer Name*.  This is the first part from `@@SERVERNAME`.  Or maybe the entire `@@SERVERNAME`.
* *Computer and Instance*.  If `@@SERVERNAME` has a computer and an instance, it will look for `computer__instance.md`. If one of the instances is the default instance, that will match `computer_mssqlserver.md`
* *Instances under the computer folder*.  If three instances on the same box are `S01\SALES`, `S01\DW`, and `S01`, you can have a folder structure like this.  The `mssqlserver.md` file is for the default instance.

```
    docs/
    ├─ s01/
    │  ├─ mssqlserver.md
    │  ├─ sales.md
    │  ├─ dw.md
```
* *Availability Group Names* 
* *Listener Names*
* *Domain Folders*.  Servers can also be arranged under domains.  It will search for the same file patterns above, but in a folder named for the domain.

```
    docs/
    ├─ domain1/
    │  ├─ s01/
    │  │  ├─ mssqlserver.md
    │  │  ├─ sales.md
    │  │  ├─ dw.md
    ├─ domain2/
    │  ├─ s02.md
```

### Important Notes
1. The files are parsed each time the page is refreshed.  It is fast and rarely used so it isn't worth caching them.
2. **ALL** files that match the above rules for a server are displayed on the `Docs` tab.  They are displayed in the order listed above.
3. The bottom of the `Docs` tab will display the files used.
4. It seems to work well to start start each file with an `H2` heading (aka "`##`").  The server name at the top of the page will be diplayed with an `H1`.  Here is a simple example:

```
    ## ServerName

    The text you want to display in Markdown.
```


<a id="feedback"></a>

## Feedback and Known Issues

Please email [Bill Graziano](mailto:billg@scalesql.com) with any issues.

<a id="previous"></a>

## Previous Releases

### 1.2 (April 2021)

*   Added filters to the application error log and the extended events page.
*   On server lists, display memory percentage as the percentage of the memory cap for the instance. Hovering over the memory field shows the cap the physical memory on the box.
*   The Extended Events page refreshes every 60 seconds and displays in absolute time instead of relative times. See [Extended Events](#xe) for more details.
*   The settings page allows you to set a URL for the "home page". This is the link the "Is It SQL" text in the upper left links to. I've set this to a tag for key servers such as "/tag/your-tag". The Tags menu has a link for "All Servers".

### 1.0.37 (July 2020)

*   Shared Credentials can be defined. These are SQL Server logins and passwords. These can then be assigned to servers. This is useful when a common SQL Server login for monitoring is used. These are stored encrypted in `connections.json`.
*   On the [Server List](http://localhost:8143/settings/servers) page, you can filter entries.

### 1.0.36 (October 2019)

*   Navigating to [http://localhost:8143/backups/json](http://localhost:8143/backups/json) will return a JSON document of all missing backups.
*   Improve alerting on missing backups.
*   Polling now checks if a server is up and does an AG health check every 10 seconds. A full poll is done every minute.
*   All usage and error reporting has been removed.
*   All functionality that was in BETA has been enabled.

Further, all Enterprise features are now enabled. This includes:

*   IsItSQL will assign tags for domain, edition, and version so you can easily group servers together.
*   Capture database mirroring status
*   Monitor availability group health and backups
*   Single page showing all instance missing backups

### What's New - 1.0.29 (18 April 2018)

*   The app better handles "names" that repoint to new instances. For example, an AG listener or static DNS entry that switches to a new instance doesn't create odd spikes in disk I/O or waits. It also better handles reseting metrics on server restarts.
*   You can choose which servers appear in the dashboard by assigning them a "dashboard" tag. It will show the friendly name you've entered and sort by that name.
*   Backup reporting now reports an AG backup from any node. If you are looking at the database page for a node in an AG, it will show that a backup was completed for that database even if it was done on another node. You can hover over the backup and it will show which node completed the backp, when it was done, and what file it sent the backup to.

### What's New - 1.0.28 (30 August 2017)

*   The Dashboard is now populated by the first three servers assigned the "dashboard" tag.
*   We better handle servers that are the primary for multiple Availability Groups.
*   Fixed a bug with the save settings on localhost

### What's New - 1.0.27 (10 July 2017)

*   The tags have been broken out into user tags and auto generated tags. Full tagging functionality is available when you sign up for the [newsletter](http://www.scalesql.com/isitsql/). I've found this makes it easier to work with the tags I assign but still lets me get to the auto generated tags.
*   The Memory column of the server list now shows the memory that SQL Server is using, the total memory on the server, and the percentage of those two numbers. It highlights the total memory on the server. I found that's the number I'm looking for most often.
*   If you hover over the Cores column, you'll see something like this: `SQL Cores Used: 1.29; Other Cores Used: 0.45`. This is just multiplying the CPU percentage by the number of cores. That math is also performed on the total line. I have tags for my VM hosts and that lets me see the CPU usage looks like in "effetive cores" for each host.
*   Many of the wait types around SQL Server 2016 Availability Groups and Query Store have been cleaned up. You should see more interesting waits now.
*   The list page doesn't show the SQL Server version any more. You can see that on the server detail page. There are auto generated tags for the version so it's still easy to find all your old 2005 servers. Sign up for the [newsletter](http://www.scalesql.com/isitsql/) to enable tagging.

### 1.0.25 (3 May 2017)

*   **The README above has been completely rewritten for this release. Please read it!**
*   Polling now times out after one minute.
*   A number of settings are now configurable via the GUI. These are the number of concurrent pollers, the port to host the web server, how often to expect a backup, and a security setting for changing these settings.
*   Servers are now entered using the GUI.
*   The menu has been simplified to include pages that summarize data across servers under a "Global" menu item
*   Backups in different timezones are now handled appropriately.
*   Servers identified by a GUID instead of a S# notation in the URLs.

### 1.0.24 (April 6, 2017)

*   Add a page to show database servers that don't have appropriate backups.
*   Increased the number of servers that can be polled concurrently
*   Cleaned up various wait group names
*   Added a page to show a summary of all your servers
*   The active sessions will show a percent complete if one is available. Hover over the duration to see it.

### 1.0.23 (December 13, 2016)

*   Include support for availability groups. Is it SQL displays the health of any availability groups it finds. Note: This is an Enteprise feature. Sign up for the mailing list above and I'll send details on enabling this feature.
*   Disk performance now breaks out the MB/sec, IOPS, averge IO size, and average duration over the one minute monitoring period. It does this for reads and writes. This column is also sortable based on the IOPS. This gives an easy way to see which servers are generating the most disk I/O.
*   The pages that show a list of servers now include a total line. It totals the disk I/O, SQL batches per second, SQL Server memory, and the size of data and log files. This lets you see the total load you're placing on your infrastructure across your all servers.

### 1.0.22 (November 10, 2016)

*   Database Mirroring is show in two places:
    *   It appears on the server page will show any databases that are mirrored.
    *   Second, there's a global database mirroring page that will show each mirrored database across all servers. It will show the status, partner, and send and receive queue sizes. It also includes a "priority column". This gives an easy way to prioritize databases that aren't online and synchronzied or have a send or receive queue.
*   The log size of database is split out into its own column which makes it sortable.
*   Information that is polled in real-time is identified with a cool lightning bolt. Every page refresh will update this data.

### 1.0.20 (September 19, 2016)

*   BETA: User-defined tags can be assigned to each server. Please sign up for the newsletter to enable this feature.
*   Assorted small bug fixes

### August 24, 2016 (1.0.19)

*   You can now view basic information about the databases on a server
*   We now prefer the ODBC 13 driver to the ODBC 11 driver.
*   If you run two instances or launch the EXE while the service is running we provide a better error
*   The menu bar now stays on top while scrolling down.
*   Processes waiting on BROKER_RECEIVE_WAITFOR are now excluded from the list of active processes

### August 4, 2016 (1.0.18)

*   Unreachable servers are displayed at the top of every page. Previously some pages may have displayed them twice or not at all
*   Sessions with a wait type of WAITFOR no longer show up as Active Sessions when looking at a server
*   Previously the database size included snapshots. Snapshots are no longer included in when computing the data size. The next update should add them back in but only include the actual data in the snapshot.
*   The default sorting for some columns has been changed to show the higher values first. For example, CPU percentage and database size.

### August 1, 2016 (1.0.17)

*   The table sort now retains the sort between page refreshes.
*   Any unreachable server now shows at the top of each page since a page may be sorted in a way that wouldn't show it.
*   If a server is unreachable, it will only log once. It will then log when it becomes reachable.
*   Polling should be faster due to more concurrent threads.
*   All JavaScript, CSS, HTML, fonts, etc. have been moved inside the executable.
*   SQL Server 2005 support is included. Barely.

### July 28, 2016 (1.0.16)

*   Includes support for limited tags
*   Added a simple gradient for CPU
*   Changed the disk I/O chart to show writes as a line to fix the random gaps

### July 19, 2016 (1.0.15)

*   The homepage now displays the number of databases and their total size
*   Added support for SQL Server Native Client 10.0
*   Improved graphing of CPU for servers in different time zones
*   Improve reporting of active sessions for SPIDs in unusual states

### July 14, 2016

*   The disk graph is improved. There's still an issue in some browsers with gaps however.

### July 13, 2016

*   The app captures the domain name and displays it when you hover over a server name.

</div>

</div>

* * *

<footer>

© 2019 ScaleOut Consulting, LLC

</footer>

</div>