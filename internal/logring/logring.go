package logring

import (
	"cmp"
	"fmt"
	"slices"
	"sync"
	"time"
)

// Event that will be inserted into the Logring
type Event struct {
	Ptr     int
	TS      time.Time
	Message string
}

// Logring is a circular buffer of log messages
type Logring struct {
	mu       sync.RWMutex
	capacity int
	buff     []Event
	ptr      int // where the next value will be inserted/written
}

// New Logring
func New(n int) *Logring {
	mr := Logring{
		capacity: n,
		buff:     make([]Event, 0, n),
	}
	return &mr
}

// Enqueue a message into the Logring
func (r *Logring) Enqueue(msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enqueue(msg)
}

// Enqueue a formatted message into the Logring
func (r *Logring) Enqueuef(str string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	msg := fmt.Sprintf(str, args...)
	r.enqueue(msg)
}

// enqueue the message
func (r *Logring) enqueue(msg string) {
	e := Event{Ptr: r.ptr, TS: time.Now(), Message: msg}
	// if the array isn't full, we are just appending
	if r.ptr < r.capacity {
		r.buff = append(r.buff, e)
	} else {
		// write the value where the pointer points
		r.buff[r.ptr%r.capacity] = e
	}
	r.ptr++
}

// Size of the Logring
func (r *Logring) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.buff)
}

// Values of the Logring in the order they were added
func (r *Logring) Values() []Event {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.values()
}

// return the values in the order they were added
func (r *Logring) values() []Event {
	// if we are less than the size of the array
	// or if we just wrote to the last slot in the array
	// we can just return the buffer as is
	if r.ptr < r.capacity || r.ptr == 0 {
		// make a new array so we don't sort the buffer
		new := make([]Event, len(r.buff))
		copy(new, r.buff)
		return new
	}
	// return from the ptr forward, then up to ptr
	ee := make([]Event, 0, len(r.buff))
	ee = append(ee, r.buff[r.ptr%r.capacity:]...)
	ee = append(ee, r.buff[:r.ptr%r.capacity]...)
	return ee
}

// Newest returns the events in reverse order they were added.
// The newest events are first.
func (r *Logring) Newest() []Event {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.newest()
}

// returns the events in reverse order of inserted
func (r *Logring) newest() []Event {
	ee := r.values()
	slices.SortStableFunc(ee, func(a, b Event) int {
		return cmp.Compare[int](b.Ptr, a.Ptr)
	})
	return ee
}

// Display to the console, the raw values of the logring.
// This is useful for debugging.
func (r *Logring) Display() {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := fmt.Sprintf("ptr=%-2d  len=%d  mod=%d", r.ptr, len(r.buff), r.ptr%r.capacity)
	out += fmt.Sprintf("  buff=%v", ee2array(r.buff))
	out += fmt.Sprintf("  vals=%v", ee2array(r.values()))
	out += fmt.Sprintf("  newest=%v", ee2array(r.newest()))
	fmt.Printf("%s\n", out)
}

func ee2array(ee []Event) []string {
	new := make([]string, len(ee))
	for i := 0; i < len(ee); i++ {
		new[i] = ee[i].Message
	}
	return new
}
