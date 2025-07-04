//Package metric provides a simple implementation of a ring buffer.
package metric

import (
	"sync"
	//"encoding/json"
	//"github.com/pkg/errors"
	//"time"
)

/*
The DefaultCapacity of an uninitialized Ring buffer.

Changing this value only affects ring buffers created after it is changed.
*/
//var DefaultCapacity = 60

var mux sync.RWMutex

// Type is the type of value stored
type Type int

// Constants for the type of metric we can store
const (
	CPU     Type = iota
	Batches Type = iota
)

// Key is the key to a value ring
type Key struct {
	ServerKey string
	Type      Type
}

var metrics map[Key]*ValueRing

