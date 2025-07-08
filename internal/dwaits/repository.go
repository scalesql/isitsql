package dwaits

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"github.com/scalesql/isitsql/internal/bucket"
	"github.com/scalesql/isitsql/internal/waitring"
	"github.com/sirupsen/logrus"
)

// MAYBE We just close this when the app exits?

// Repository holds the repository of real-time waits
type Repository struct {
	servers map[string]*waitring.Ring
	bw      bucket.BucketWriter
	mu      sync.RWMutex
	open    bool
}

// NewRepository returns a new Repository
func NewRepository(ctx context.Context) (*Repository, error) {
	r := Repository{}
	r.open = true
	r.servers = make(map[string]*waitring.Ring)

	// start a writer
	exe, err := os.Executable()
	if err != nil {
		return &r, errors.Wrap(err, "os.executable")
	}
	dir := filepath.Dir(exe)
	dir = filepath.Join(dir, "cache")
	r.bw.Start(dir, "w2")
	return &r, nil
}

// ReadHistory reads the cached history
func (r *Repository) ReadHistory() error {
	logrus.Debug("w2.repository: readhistory...")
	r.mu.Lock()
	defer r.mu.Unlock()

	exe, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "os.executable")
	}
	dir := filepath.Dir(exe)
	dir = filepath.Join(dir, "cache")
	br, err := bucket.NewReader("w2", dir)
	if err != nil {
		return errors.Wrap(err, "bucket.newreader")
	}
	go br.StartReader()
	var nr, nw int64
	loadStart := time.Now()
	limit := time.Now().Add(-60 * time.Minute)
	timec := time.After(30 * time.Second)

Loop:
	for {
		select {
		case <-timec:
			logrus.Error("cache: w2: reader: timeout")
			break Loop
		case str, ok := <-br.Results:
			if !ok {
				logrus.Trace("cache: w2: reader: empty channel")
				break Loop
			}
			nr++
			// unmarshal and enqueue
			var se bucket.ServerEvent
			logrus.Tracef("cache: waits: reading: unmarshal server event: %d", nr)
			err = json.Unmarshal([]byte(str), &se)
			if err != nil {
				logrus.Error(errors.Wrap(err, "se.unmarshal"))
				continue
			}

			var waits waitring.WaitList
			err = json.Unmarshal([]byte(se.Payload), &waits)
			if err != nil {
				logrus.Error(errors.Wrap(err, "w2: payload.unmarshal"))
				continue
			}

			if waits.TS.Before(limit) {
				continue
			}
			r.writehistory(se.MapKey, waits)
			nw++
		}
	}

	if br.Err != nil {
		logrus.Errorf("br.Err: w2: %s", br.Err)
	}
	loadDuration := time.Since(loadStart)
	logrus.Info(fmt.Sprintf("W2: Read: %s  Used: %s (%s)", humanize.Comma(nr), humanize.Comma(nw), loadDuration.String()))

	return nil
}

// Close the Repository
func (r *Repository) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.open = false
	return nil
}

// Write the waits to the ring buffer and cache
func (r *Repository) Write(key string, waits waitring.WaitList) error {
	// we need the zero waits
	// if len(waits.Waits) == 0 {
	// 	return nil
	// }
	// read the status of the repository
	r.mu.RLock()
	open := r.open
	ring, ok := r.servers[key]
	r.mu.RUnlock()

	if !open {
		return nil
	}

	// if this is a new server, let's add the wait ring
	if !ok {
		r.mu.Lock()
		// double-check in case it was added
		ring, ok = r.servers[key]
		if !ok || ring == nil {
			newring := waitring.New(60)
			ring = &newring
			r.servers[key] = ring
		}
		r.mu.Unlock()
	}

	ring.Enqueue(waits)
	r.bw.Write(key, waits)
	return nil
}

func (r *Repository) writehistory(key string, waits waitring.WaitList) {
	// this is only called from within ReadHistory and that locks
	ring, ok := r.servers[key]
	if !ok {
		nr := waitring.New(60)
		ring = &nr
		r.servers[key] = ring
	}
	ring.Enqueue(waits)
}

func (r *Repository) Values(key string) []waitring.WaitList {
	r.mu.RLock()
	ring, ok := r.servers[key]
	open := r.open
	r.mu.RUnlock()
	if !open || !ok || ring == nil {
		return []waitring.WaitList{}
	}
	return ring.Values()
}

// Last returns the most recent waits for a server key
func (r *Repository) Last(key string) waitring.WaitList {
	r.mu.RLock()
	ring, ok := r.servers[key]
	open := r.open
	r.mu.RUnlock()
	if !open || !ok || ring == nil {
		return waitring.WaitList{
			TS:    time.Time{},
			Waits: make(map[string]int64),
		}
	}
	return ring.Last()
}

// Top returns the top n waits for a particular key
func (r *Repository) Top(key string, n int) waitring.SortedMapInt64 {
	r.mu.RLock()
	ring, ok := r.servers[key]
	open := r.open
	r.mu.RUnlock()
	if !open || !ok || ring == nil {
		return waitring.SortedMapInt64{
			BaseMap:    map[string]int64{},
			SortedKeys: []string{},
		}
	}
	return ring.Top(5)
}

func (r *Repository) Delete(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.open {
		return
	}
	delete(r.servers, key)
}
