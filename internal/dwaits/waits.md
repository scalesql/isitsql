

Request
-------
* SessionID
* Start 
* Wait
* WaitDuration
* map[string]int -- map[mapped_wait]ms

From SQL Server
---------------
* SessionID
* Start
* Wait
* WaitDuration

The Repo
-------
* map[wait]ms

Process
-------
* If first, just save the sessions, repo stays empty
* For each session from SQL Server
    * if id match, start matches, server matches
        * if wait matches and duration is higher
            * repo: save the incremental mapped wait
            * Session: save the wait and new duration
        * new wait,
            * repo: save the mapped wait duration
            * Session: save the wait and the duration
    * else new request
        * repo: save the mapped wait duration
        * Session: save the wait and the duration
* For each session we have
    * If doesn't exist from SQL Server, remove it
