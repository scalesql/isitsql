package app

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"time"

	"github.com/scalesql/isitsql/internal/failure"
	"github.com/kardianos/osext"
	"github.com/pkg/errors"
)

// PollRoutine is the new long lived poll
func PollRoutine(s *SqlServerWrapper) {
	defer failure.HandlePanic()

	newpoll(s, true)

	// delay up to 10 seconds to avoid thundering heard
	// and spread the load out
	h := fnv.New32a()
	_, _ = h.Write([]byte(s.MapKey)) // if errors, just use zero
	msdelay := h.Sum32() % 10000
	time.Sleep(time.Duration(msdelay) * time.Millisecond)
	newpoll(s, false)

	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-s.stop:
			WinLogln("Polling stop:", s.MapKey)
			return
		case <-ticker.C:
			newpoll(s, false)
		}
	}
}

func newpoll(m *SqlServerWrapper, forcequick bool) {

	// If we're already polling, don't start again
	m.RLock()
	ispolling := m.IsPolling
	m.RUnlock()
	if ispolling {
		return
	}

	var bigpoll = false

	// duplicate the clean up in case we panic
	defer func() {
		m.Lock()
		m.IsPolling = false
		m.PollActivity = ""
		m.PollDuration = time.Since(m.PollStart)
		m.ResetOnThisPoll = false
		m.Unlock()
	}()

	m.Lock()
	m.IsPolling = true
	m.PollStart = time.Now()
	m.Unlock()

	bigpoll, err := m.getAllMetrics(forcequick)
	if err != nil {
		serverName := m.MapKey
		// TODO shouldn't this be protected?
		if len(m.ServerName) > 0 {
			m.RLock()
			serverName = m.ServerName
			m.RUnlock()
		}

		// Only log if the error changes
		if m.LastPollError != err.Error() {
			WinLogln(serverName, err.Error())
		}
		m.Lock()
		m.LastPollError = err.Error()
		m.LastPollFail = time.Now()
		m.Unlock()
	} else {
		m.Lock()

		// Log if there was previously an error
		if m.LastPollError != "" {
			serverName := m.MapKey
			if len(m.ServerName) > 0 {
				serverName = m.ServerName
			}
			WinLogln(serverName, "*** Error Cleared ***")
		}
		m.LastPollError = ""
		m.LastPollFail = time.Time{}
		m.Unlock()
	}

	// clean up -- also done in the defer in case we panic
	m.Lock()
	m.IsPolling = false
	m.PollActivity = ""
	m.PollDuration = time.Since(m.PollStart)
	m.ResetOnThisPoll = false
	if !forcequick && bigpoll {
		m.PollCount++
	}
	m.Unlock()

	// was this a big poll
	if !forcequick && bigpoll {
		err := m.writeCache()
		if err != nil {
			WinLogln(err)
		}
	}
}

func (sw *SqlServerWrapper) writeCache() error {
	wd, err := osext.ExecutableFolder()
	if err != nil {
		return errors.Wrap(err, "osext.executablefolder")
	}
	dir := filepath.Join(wd, "cache")
	fileName := filepath.Join(dir, fmt.Sprintf("server.%s.json", sw.MapKey))
	//bb, err := json.MarshalIndent(sw.SqlServer, "", "\t")
	sw.RLock()
	bb, err := json.Marshal(sw.SqlServer)
	sw.RUnlock()
	if err != nil {
		return errors.Wrap(err, "json.marshalindent")
	}
	err = os.WriteFile(fileName, bb, 0600)
	if err != nil {
		return errors.Wrap(err, "os.writefile")
	}
	return nil
}
