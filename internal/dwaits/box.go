package dwaits

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/pkg/errors"
	"github.com/scalesql/isitsql/internal/failure"
	"github.com/scalesql/isitsql/internal/logring"
	"github.com/scalesql/isitsql/internal/waitmap"
	"github.com/scalesql/isitsql/internal/waitring"
	"github.com/sirupsen/logrus"
)

// https://guzalexander.com/2017/05/31/gracefully-exit-server-in-go.html
// https://stackoverflow.com/questions/66862063/go-context-store-cancel-function-returned-by-withcancelctx
// Store a context and a cancel function?  Maybe just the cancel function?

// ErrLowVersion identifies if a server doesn't support per second polling
var ErrLowVersion = fmt.Errorf("version too low")

// Error returns the last error in polling
func (b *Box) Error() error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.err
}

// setup gets the basic server information and does a version check
func (b *Box) setup(ctx context.Context) error {
	b.messages.Enqueue("setup: entering")
	defer b.messages.Enqueue("setup: exiting")
	// Get basic server information
	var domain, server string
	var started time.Time
	var major int
	err := b.db.QueryRowContext(ctx, `
				SELECT 	COALESCE(DEFAULT_DOMAIN(), '') AS [domain], 
						@@SERVERNAME AS [server_name], 
						create_date AS [started_time],
						CAST(serverproperty('ProductMajorVersion') AS INT) AS major
				FROM 	sys.databases 
				WHERE 	name = 'tempdb'`).Scan(&domain, &server, &started, &major)
	if err != nil {
		return err
	}

	// Only run for SQL Server 2008 and higher
	if major < 10 {
		return ErrLowVersion
	}
	b.domain = domain
	b.server = server
	b.booted = started
	return nil
}

// Start a Box polling a server every second
func (b *Box) Start(ctx context.Context, repo *Repository, key string, connType string, connString string) error {
	if b.messages == nil {
		b.messages = logring.New(1000)
	}
	logrus.Tracef("box.start: %s", key)
	b.messages.Enqueue("start:entering")
	defer b.messages.Enqueue("start:exiting")

	// start a GO routine to start this
	go func(b *Box) {
		defer failure.HandlePanic()
		b.key = key
		b.first = true
		b.repo = repo
		b.ctx, b.ctxCanel = context.WithCancel(ctx)

		var err error
		b.db, err = sql.Open(connType, connString)
		if err != nil {
			logrus.Error(errors.Wrap(err, "box: start: sql.open"))
			return
		}

		// try, if ok, goto connected
		err = b.setup(ctx)
		if err == ErrLowVersion { // if we are too low a version, log it and don't start polling
			_ = b.db.Close()
			logrus.Warnf("box: low version: %s (%s): exiting", key, b.server)
			b.messages.Enqueue("start: low version")
			return
		}
		var connectTicker *time.Ticker

		if err == nil { // if there isn't an error, skip to connected
			goto connected
		}
		logrus.Warnf("box: start: retrying: %s: %s", b.key, err)
		b.messages.Enqueue("start: retrying")
		// if there was an error, loop to try every 10s
		connectTicker = time.NewTicker(10 * time.Second)
		defer connectTicker.Stop()

	out:
		for {
			select {
			case <-b.ctx.Done(): // if we get a cancel
				b.messages.Enqueue("start: <-b.ctx.Done")
				_ = b.db.Close()
				connectTicker.Stop()
				return
			case <-connectTicker.C:
				b.messages.Enqueue("start: <-connectTicker.C")
				err = b.setup(ctx)
				if err == ErrLowVersion {
					_ = b.db.Close()
					logrus.Warnf("box: low version: %s (%s): exiting", key, b.server)
					connectTicker.Stop()
					return
				}
				if err == nil {
					connectTicker.Stop()
					break out
				}
			}
		}

	connected:
		// check for version too low error
		if err == ErrLowVersion {
			logrus.Warnf("box: low version: %s (%s): exiting", key, b.server)
			return
		}
		b.messages.Enqueue("start: connected")
		b.requests = make(map[int16]request)
		b.Waits = make(map[string]int64)

		// sleep some random amount between 0 and 1s so these are staggered
		h := fnv.New32a()
		_, _ = h.Write([]byte(b.key)) // if errors, just use zero
		msdelay := h.Sum32() % 1000

		pollTicker := watch.Ticker(1 * time.Second)
		go func(b *Box, delay int) { // this is the polling GO routine
			b.messages.Enqueue("box: pollticker: entering")
			defer failure.HandlePanic()
			time.Sleep(time.Duration(delay) * time.Millisecond)
			logrus.Debugf("box: polling waits: %s (%s)", b.key, b.server)
			b.messages.Enqueuef("box: polling waits: %s (%s)", b.key, b.server)
			for {
				select {
				case <-b.ctx.Done():
					b.messages.Enqueue("box: pollticker: <-b.ctx.Done()")
					_ = b.db.Close()
					logrus.Tracef("box: %s: polling exiting", b.key)
					return
				case <-pollTicker.C:
					b.messages.Enqueue("box: pollticker: <-pollTicker.C")
					b.pollWaitsWrapper()
				}
			}
		}(b, int(msdelay))

		emitTicker := watch.Ticker(emitFrequency * time.Second)
		go func(b *Box, delay int) { // this is the emit GO routine
			b.messages.Enqueue("box: emitticker: entering")
			defer failure.HandlePanic()
			time.Sleep(time.Duration(delay) * time.Millisecond)
			time.Sleep(500 * time.Millisecond) // offset the emitted from the polling
			for {
				select {
				case <-b.ctx.Done():
					logrus.Tracef("box: %s: emitting exiting", b.key)
					return
				case <-emitTicker.C:
					b.messages.Enqueue("box: emitticker: <-emitTicker.C")
					b.emitWaits()
				}
			}
		}(b, int(msdelay))
	}(b)
	return nil
}

func (b *Box) Stop() {
	b.messages.Enqueue("box: stop")
	b.ctxCanel()
}

func (b *Box) emitWaits() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.messages.Enqueuef("box: emitting: waits: %d", len(b.Waits))
	b.repo.Write(b.key, waitring.WaitList{
		TS:    time.Now(),
		Waits: b.Waits,
	})
	// clear the waits
	b.Waits = make(map[string]int64)
}

// pollWaitsWrapper handles state changes in polling waits.
// only log errors on error type changes
func (b *Box) pollWaitsWrapper() {
	b.messages.Enqueue("box: pollwaitswrapper: entering")
	defer b.messages.Enqueue("box: pollwaitswrapper: exiting")
	b.mu.Lock()
	defer b.mu.Unlock()
	err := b.pollwaits()
	var existing, new string
	if b.err != nil {
		existing = b.err.Error()
	}
	if err != nil {
		new = err.Error()
	}
	if existing != new { // we are in a new state
		b.messages.Enqueuef("poll: new state: [%s] -> [%s]", existing, new)
		if b.err == nil { // we are entering an error state
			b.err = err
			if b.statement != nil {
				b.statement.Close()
			}
			b.statement = nil
			b.stateStart = time.Now()
			logrus.Errorf("pollwaits: err: %s (%s): %s", b.key, b.server, err)
			b.messages.Enqueuef("box: pollwaitswrapper: e1: %s", err)
		} else { // we are changing state, maybe to a good one
			if err == nil { // error is ending
				b.err = err
				logrus.Infof("pollwaits: up: %s (%s) down=%s", b.key, b.server, time.Since(b.stateStart).String())
				b.messages.Enqueuef("box: pollwaitswrapper: up: %s (%s) down=%s", b.key, b.server, time.Since(b.stateStart).String())
			} else { // error is changing
				b.err = err
				logrus.Warnf("pollwaits: err: %s (%s): %s", b.key, b.server, err)
				b.messages.Enqueuef("pollwaits: err2: %s", err)
			}
		}
	}
}

func (b *Box) Messages() []logring.Event {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.messages.Newest()
}

// Repository returns a pointer to the Repository associated with the Box.
func (b *Box) Repository() *Repository {
	if b == nil {
		return nil
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.repo == nil {
		return nil
	}
	return b.repo
}

func (b *Box) pollwaits() error {
	b.messages.Enqueue("box: pollwaits: entering")
	var err error
	// query the waits and sessions
	if b.statement == nil {
		logrus.Tracef("%s: box: pollwaits: preparing statement: %s", b.server, b.key)
		b.statement, err = b.db.Prepare(requestQuery)
		if err != nil {
			//logrus.Errorf("%s: pollwaits: prepare: %s (%s)", b.server, err, b.key)
			return err
		}
	}
	domain, server, booted, requests, err := queryWaits(b.statement)
	if err != nil {
		return err
	}

	// if first or different server or restart,
	// then we are resetting the sessions,
	// so replace existing sessions, first=false, return
	if b.first || b.domain != domain || b.server != server || b.booted != booted {
		b.messages.Enqueue("pollwaits: resetting sessions")
		b.requests = requests
		b.domain = domain
		b.server = server
		b.booted = booted

		b.requests = requests

		if b.first {
			b.first = false
			return nil
		}
		logrus.Tracef("waits: box: reset: %s: now: %s (%s): %s", b.key, b.server, b.domain, b.booted.Format(time.RFC1123Z))
		return nil
	}

	logrus.Tracef("box: %s: polled requests: %d", b.key, len(requests))
	requestCount := 0
	for id, active := range requests {
		requestCount++
		r, exists := b.requests[id]
		// if it doesn't exist, save the request and save the wait
		// move on to the next active session
		if !exists {
			logrus.Tracef("box: %s: new request: %d", b.key, id)
			b.requests[id] = active
			b.mapandsavewait(active.wait, active.waitTimeMS)
			continue
		}
		// it exists, start matches, wait matches, AND wait is higher
		// save the request, save the incremental wait,
		// and continue
		if r.started.Equal(active.started) { // this is the same request
			if r.wait == active.wait { // it is the same wait
				if active.waitTimeMS > r.waitTimeMS {
					b.mapandsavewait(active.wait, active.waitTimeMS-r.waitTimeMS)
				} else if active.waitTimeMS < r.waitTimeMS {
					b.mapandsavewait(active.wait, active.waitTimeMS)
				} // else they are equal, do nothing
				b.requests[id] = active
				continue
			}
		}
		// something has changed, this is a new request or the wait is different
		b.requests[id] = active
		b.mapandsavewait(active.wait, active.waitTimeMS)
	}
	if requestCount > 0 {
		b.messages.Enqueuef("box: pollwaits: requests: %d", requestCount)
	}

	// go through the existing requests
	// and remove any that aren't active
	// we can do this because we are getting all requests
	for id := range b.requests {
		_, exists := requests[id]
		if !exists {
			delete(b.requests, id)
		}
	}

	b.lastPoll = watch.Now()
	//logrus.Tracef("waits: %s: polling", b.key)
	if b.first {
		b.first = false
	}
	return nil
}

func (b *Box) mapandsavewait(wait string, ms int64) {
	// don't save empty waits
	if wait == "" {
		return
	}

	wm := waitmap.Mapping.Lookup(wait)
	if wm.Excluded {
		return
	}
	if ms > 300*1000 { // 5 minutes
		logrus.Debugf("waits: key: %s  wait: %s  sec: %d", b.key, wait, ms/1000)
		b.messages.Enqueuef("waits: key: %s  wait: %s  sec: %d", b.key, wait, ms/1000)
	}
	if wm.MappedTo != "" {
		wait = wm.MappedTo
	}
	//logrus.Tracef("pollwaits: increase wait: to: %s", waitName)
	w0, exists := b.Waits[wait]
	if !exists {
		b.Waits[wait] = ms
	} else {
		b.Waits[wait] = w0 + ms
	}
}

// queryWaits returns the domain, server, start time, and a map of the sessions and waits
func queryWaits(stmt *sql.Stmt) (string, string, time.Time, map[int16]request, error) {
	rr := make(map[int16]request)

	var domain, server, wait string
	var id int16
	var waitms int64
	var boot, started time.Time
	var err error
	if stmt == nil {
		return "", "", time.Time{}, rr, errors.New("stmt is nil")

	}

	rows, err := stmt.Query()
	if err != nil {
		logrus.Tracef("querywaits: resetting stmt")
		return "", "", time.Time{}, rr, err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&domain, &server, &boot, &id, &started, &wait, &waitms)
		if err != nil {
			return "", "", time.Time{}, rr, err
		}
		// Turn these rows into the session map
		r := request{
			id:         id,
			started:    started,
			wait:       wait,
			waitTimeMS: waitms,
		}
		rr[id] = r
		logrus.Tracef("querywaits: server: %s; id: %d;  wait: %s; ms: %d", server, id, wait, waitms)
	}
	err = rows.Err()
	if err != nil {
		return "", "", time.Time{}, rr, err
	}
	return domain, server, boot, rr, nil
}

var requestQuery = `
USE [master];
SELECT	
		COALESCE(DEFAULT_DOMAIN(), '') AS [domain]
		,@@SERVERNAME AS [server_name]
		,(SELECT create_date FROM sys.databases WHERE name = 'tempdb')  AS [server_started]
		,session_id
		,start_time 
		,COALESCE(wait_type, '') as wait_type
		,wait_time
FROM	sys.dm_exec_requests 
WHERE	session_id IS NOT NULL
AND		session_id <> 0
-- AND		request_id = 0 -- apparently lots of spids have this
AND		COALESCE([status],'') <> 'background'
AND		COALESCE(command, '') <> 'TASK MANAGER'
AND		session_id <> @@SPID 
AND		COALESCE(wait_type, '') NOT IN ( 'SP_SERVER_DIAGNOSTICS_SLEEP', 'WAITFOR', 'BROKER_RECEIVE_WAITFOR' )
`
