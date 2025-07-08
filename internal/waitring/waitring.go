package waitring

import (
	"encoding/json"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
)

/*
+-------------------+
| 0 |   |  |   |   |
+-------------------+
|   |   |   |   |  |
+-------------------+

+-------------------+
|   | 1  |  |   |   |
+-------------------+
| a |   |   |   |   |
+-------------------+

+-------------------+
|   |   | 2 |   |   |
+-------------------+
| a | b |   |   |   |
+-------------------+

+-------------------+
|   |   | 7 |   |   |
+-------------------+
| f | g | c | d | e |
+-------------------+
*/

// Ring holds a circular ring buffer that will never grow
// We only keep a pointer to the next insert location
type Ring struct {
	data []WaitList
	p    int // pointer where the next value will be inserted
	mu   sync.RWMutex
}

// WaitList is a map of mapped waits, durations at a particular time
type WaitList struct {
	TS    time.Time        `json:"ts"`
	Waits map[string]int64 `json:"waits"`
}

// GetHistory returns a copy of the history
func (r *Ring) GetHistory() []WaitList {
	r.mu.RLock()
	defer r.mu.RUnlock()
	wl := make([]WaitList, 0, len(r.data))
	_ = copy(wl, r.data)
	return wl
}

// Top returns the top waits over the entire one hour history
// Passing in zero returns all waits
func (r *Ring) Top(n int) SortedMapInt64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	wgl := make(map[string]int64)
	var ok bool
	v := r.values()
	for _, wv := range v {
		for wg, duration := range wv.Waits {
			_, ok = wgl[wg]
			if duration > 0 {
				if ok {
					wgl[wg] += duration
				} else {
					wgl[wg] = duration
				}
			}
		}
	}

	sm := new(SortedMapInt64)
	sm.BaseMap = wgl
	sm.SortedKeys = make([]string, len(wgl))
	i := 0
	for key := range sm.BaseMap {
		sm.SortedKeys[i] = key
		i++
	}
	sort.Sort(sm)
	if n == 0 {
		return *sm
	}
	if n < len(sm.SortedKeys) {
		sm.SortedKeys = sm.SortedKeys[0:n]
	}
	return *sm
}

func (r *Ring) MarshalJSON() ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	println("marshalling")
	out := struct {
		Data []WaitList `json:"data"`
		P    int        `json:"p"`
	}{
		Data: r.data,
		P:    r.p,
	}
	return json.Marshal(out)
}

func (r *Ring) UnmarshalJSON(bb []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	type in struct {
		Data []WaitList `json:"data"`
		P    int        `json:"p"`
	}
	var i in
	err := json.Unmarshal(bb, &i)
	if err != nil {
		return errors.Wrap(err, "json.unmarshal")
	}
	r.p = i.P
	r.data = i.Data
	return nil
}

// New create a fixed size ring buffer
func New(size int) Ring {
	r := Ring{}
	r.data = make([]WaitList, size)
	return r
}

// Enqueue a value
func (r *Ring) Enqueue(wm WaitList) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// just insert the value at the pointer
	// the array is already initialized
	r.data[r.p%len(r.data)] = wm
	r.p += 1
}

// Last returns the most recent value in the ring buffer.
// If the ring is empty, it returns an empty WaitList
func (r *Ring) Last() WaitList {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.p == 0 {
		return WaitList{TS: time.Time{}, Waits: make(map[string]int64)}
	}
	// the last value is at p-1
	return r.data[(r.p-1)%len(r.data)]
}

// Values returns the values in the order they were enqueued
func (r *Ring) Values() []WaitList {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.values()
}

func (r *Ring) values() []WaitList {
	// if we aren't full, just get the first elements
	if r.p < len(r.data)+1 {
		return r.data[0:r.p]
	}
	result := make([]WaitList, 0, len(r.data))

	// get the back half of the array: c,d,e above
	result = append(result, r.data[r.p%len(r.data):len(r.data)]...)

	// get the front of the array: f,g above
	result = append(result, r.data[0:r.p%len(r.data)]...)
	return result

}
