Pool Test
=========

This app tests connection pooling and failover for load balancers moving SQL Server.

Parameters
----------
* `-server server` -- only the host part
* `-instance instance` -- only the optional instance part
* `-port port` -- defaults to 1433
* `-ping` --defaults to false. Lookup the IP address of the server.  If it has changed, created a new connection pool.
* `-life duration` -- lifetime of the connection in the pool.  Defaults to 1m.
* `-interval duration` -- how often to poll. Defaults to 1s.
* `-msf` -- enable multisubnet failover
* `-protocol protocol` --force a protocol (tcp|np|lpc)

Logic
-----
* Start a connection pool at the beginning
    * use the connection life time parameter
* Poll the SQL Server listed every interval
    * if -ping-first,
        * ping the host to get the IPs.  The host may return multiple IPs.
        * If they changed, build a new connection pool.
    * query the server name, authentiction method, connection start time, session start time, and SPID
    * if any of those changed from the last poll, print them
* for all logging, include the hh:mm:ss time
* for all errors
    * Print the error type and the unwrapped error type
    * if it is a go-mssqlb.Error, print the Number, State, Class, Message, ServerName,
all the errors in All.
    * If it is a RetryableError, indicate this
    * Also indicate if this is a driver.ErrBadConn error




