package dwaits

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/scalesql/isitsql/internal/logring"
)

var watch = clock.New()

const emitFrequency = 60

// var verboseWaitBox = false

type Box struct {
	booted     time.Time
	lastPoll   time.Time
	ctx        context.Context
	err        error // Last error (usually polling)
	stateStart time.Time
	ctxCanel   context.CancelFunc
	db         *sql.DB
	requests   map[int16]request
	Waits      map[string]int64 `json:"w2_current"`
	repo       *Repository
	statement  *sql.Stmt
	domain     string
	server     string
	key        string
	mu         sync.RWMutex
	first      bool // First time we are polling this server
	messages   *logring.Logring
}

type request struct {
	started    time.Time
	wait       string
	waitTimeMS int64
	id         int16
}
