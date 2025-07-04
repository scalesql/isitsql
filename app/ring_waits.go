package app

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/scalesql/isitsql/internal/waitmap"
	"github.com/pkg/errors"
)

// DefaultCapacity of an uninitialized Ring buffer.
// Changing this value only affects ring buffers created after it is changed.
var DefaultCapacity = 60

// DefaultDuration is used to limit the window of events returned from the wait ring
var DefaultDuration = 61 * time.Minute

// WaitRing implements a Circular Buffer.
// The default value of the Ring struct is a valid (empty) Ring buffer with capacity DefaultCapacify.
type WaitRing struct {
	buff []*waitmap.Waits
	head int
	tail int
}

// SetCapacity sets the maximum size of the ring buffer.
func (r *WaitRing) SetCapacity(size int) {
	r.checkInit()
	r.extend(size)
}

// Capacity returns the current capacity of the ring buffer.
func (r WaitRing) Capacity() int {
	return len(r.buff)
}

// Enqueue a value into the Ring buffer.
func (r *WaitRing) Enqueue(i *waitmap.Waits) {
	r.checkInit()
	r.set(r.head+1, i)
	old := r.head
	r.head = r.mod(r.head + 1)
	if old != -1 && r.head == r.tail {
		r.tail = r.mod(r.tail + 1)
	}
}

// Dequeue a value from the Ring buffer.
// Returns nil if the ring buffer is empty.
func (r *WaitRing) Dequeue() *waitmap.Waits {
	var w *waitmap.Waits
	r.checkInit()
	if r.head == -1 {
		return w
	}
	v := r.get(r.tail)
	if r.tail == r.head {
		r.head = -1
		r.tail = 0
	} else {
		r.tail = r.mod(r.tail + 1)
	}
	return v
}

// Peek the value that Dequeue would have dequeued without actually dequeuing it.
// Returns nil if the ring buffer is empty.
func (r *WaitRing) Peek() (waits *waitmap.Waits) {
	r.checkInit()
	var w *waitmap.Waits
	if r.head == -1 {
		return w
	}
	return r.get(r.tail)
}

// GetNewest returns the most recently added value.  It returns false if
// there is no value to return.
func (r *WaitRing) GetNewest() (waits *waitmap.Waits, ok bool) {
	var w *waitmap.Waits
	r.checkInit()
	if r.head == -1 {
		return w, false
	}
	return r.get(r.head), true
}

// Values returns a slice of all the values in the circular buffer without modifying them at all.
// The returned slice can be modified independently of the circular buffer. However, the values inside the slice
// are shared between the slice and circular buffer.
func (r *WaitRing) Values() []*waitmap.Waits {
	return r.values()
}

// TopGroups gets the top wait groups
func (r *WaitRing) TopGroups() (topWaits SortedMapInt64) {
	wgl := make(map[string]int64)
	var ok bool

	// Group all wait groups together
	v := r.values()
	for _, wv := range v {
		for wg, duration := range wv.WaitSummary {
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

	return *sm
}

// NonZeroValues returns the waits that aren't zero
func (r *WaitRing) NonZeroValues() []*waitmap.Waits {
	if r.head == -1 {
		return nil
	}
	//arr := make([]*Waits, 0, r.Capacity())
	// for i := 0; i < r.Capacity(); i++ {
	// 	idx := r.mod(i + r.tail)

	// 	w := r.get(idx)
	// 	for key, value := range w.Waits {
	// 		if value.WaitTimeDelta == 0 {
	// 			delete(w.Waits, key)
	// 		}
	// 	}

	// 	arr = append(arr, w)
	// 	if idx == r.head {
	// 		break
	// 	}
	// }
	waits := r.values()
	for i, wait := range waits {
		for key, value := range wait.Waits {
			if value.WaitTimeDelta == 0 {
				delete(waits[i].Waits, key)
			}
		}
	}
	return waits
}

// MarshalJSON marshals to a byte array
func (r WaitRing) MarshalJSON() ([]byte, error) {
	var wr struct {
		Buffer []*waitmap.Waits `json:"buffer"`
		Head   int              `json:"head"`
		Tail   int              `json:"tail"`
	}
	wr.Head = r.head
	wr.Tail = r.tail
	wr.Buffer = r.buff
	j, err := json.Marshal(wr)
	if err != nil {
		err = errors.Wrap(err, "marshal")
	}
	return j, err
}

// UnmarshalJSON unmarshals to the struct
func (r *WaitRing) UnmarshalJSON(b []byte) error {
	var wr struct {
		Buffer []*waitmap.Waits `json:"buffer"`
		Head   int              `json:"head"`
		Tail   int              `json:"tail"`
	}

	err := json.Unmarshal(b, &wr)
	if err != nil {
		return errors.Wrap(err, "unmarshall")
	}
	r.head = wr.Head
	r.tail = wr.Tail
	r.buff = wr.Buffer

	return nil
}

/*************************************************************************************************
*** Unexported methods beyond this point.
**************************************************************************************************/
// values returns all values within the DefaultDuration
func (r *WaitRing) values() []*waitmap.Waits {
	if r.head == -1 {
		return nil
	}
	limit := time.Now().Add(-1 * DefaultDuration)
	arr := make([]*waitmap.Waits, 0, r.Capacity())
	for i := 0; i < r.Capacity(); i++ {
		idx := r.mod(i + r.tail)
		w := r.get(idx)
		if w.EventTime.After(limit) {
			arr = append(arr, w)
		}
		if idx == r.head {
			break
		}
	}
	return arr
}

// set a value at the given unmodified index and returns the modified index of the value
func (r *WaitRing) set(p int, v *waitmap.Waits) {
	r.buff[r.mod(p)] = v
}

// get a value based at a given unmodified index
func (r *WaitRing) get(p int) *waitmap.Waits {
	return r.buff[r.mod(p)]
}

// mod returns the modified index of an unmodified index
func (r *WaitRing) mod(p int) int {
	return p % len(r.buff)
}

// checkInit checks if the WaitGroup is initialized and inits if needed
func (r *WaitRing) checkInit() {
	var w *waitmap.Waits
	if r.buff == nil {
		r.buff = make([]*waitmap.Waits, DefaultCapacity)
		for i := range r.buff {
			r.buff[i] = w
		}
		r.head, r.tail = -1, 0
	}
}

// extend the WaitRing to the specified size.  Will reduce
// the WaitRing if needed.
func (r *WaitRing) extend(size int) {
	if size == len(r.buff) {
		return
	} else if size < len(r.buff) {
		r.buff = r.buff[0:size]
	}
	newb := make([]*waitmap.Waits, size-len(r.buff))

	for i := range newb {
		var w waitmap.Waits
		newb[i] = &w
	}
	r.buff = append(r.buff, newb...)
}
