// Package metric provides a simple implementation of a ring buffer.
package metric

import (
	//"encoding/json"
	//"github.com/pkg/errors"
	//"sync"
	"time"
)

// ValueRing implements a Circular Buffer.
// The default value of the Ring struct is a valid (empty) Ring buffer with capacity DefaultCapacify.
type ValueRing struct {
	buff []*Value
	head int // the most recent value written
	tail int // the least recent value written
}

// Value holds the actual value
type Value struct {
	TimeStamp      time.Time
	Value          int64
	Delta          int64
	Duration       time.Duration
	PolledValue    bool
	DeltaPerSecond int64
}

// capacity returns the current capacity of the ring buffer.
func (r *ValueRing) capacity() int {
	return len(r.buff)
}

/*
Enqueue a value into the Ring buffer.
*/
func (r *ValueRing) enqueue(i *Value) {
	// r.Lock()
	// defer r.Unlock()
	r.checkInit()
	r.set(r.head+1, i)
	old := r.head
	r.head = r.mod(r.head + 1)
	if old != -1 && r.head == r.tail {
		r.tail = r.mod(r.tail + 1)
	}
}

/*
last returns the most recently added value

Returns nil if the ring buffer is empty.
*/
func (r *ValueRing) last() *Value {

	r.checkInit()
	if r.head == -1 {
		return nil
	}
	return r.get(r.head)
}

/*
Dequeue a value from the Ring buffer.

Returns nil if the ring buffer is empty.
*/
// func (r *ValueRing) dequeue() *Value {
// 	// r.Lock()
// 	// defer r.Unlock()
// 	r.checkInit()
// 	if r.head == -1 {
// 		return nil
// 	}
// 	v := r.get(r.tail)
// 	if r.tail == r.head {
// 		r.head = -1
// 		r.tail = 0
// 	} else {
// 		r.tail = r.mod(r.tail + 1)
// 	}
// 	return v
// }

/***********************************************************************************************************
*** Unexported methods beyond this point.
************************************************************************************************************/

// sets a value at the given unmodified index and returns the modified index of the value
func (r *ValueRing) set(p int, v *Value) {
	r.buff[r.mod(p)] = v
}

// gets a value based at a given unmodified index
func (r *ValueRing) get(p int) *Value {
	return r.buff[r.mod(p)]
}

// returns the modified index of an unmodified index
func (r *ValueRing) mod(p int) int {
	return p % len(r.buff)
}

func (r *ValueRing) checkInit() {
	if r.buff == nil {
		r.buff = make([]*Value, 60)
		for i := range r.buff {
			r.buff[i] = nil
		}
		r.head, r.tail = -1, 0
	}
}

func (r *ValueRing) extend(size int) {
	if size == len(r.buff) {
		return
	} else if size < len(r.buff) {
		r.buff = r.buff[0:size]
	}
	newb := make([]*Value, size-len(r.buff))
	for i := range newb {
		newb[i] = nil
	}
	r.buff = append(r.buff, newb...)
}
