// Package cpuring provides a simple implementation of a ring buffer.
// This should only be written to within polling which locks at the
// SqlServer level.  That is why it doesn't need locks.  Anything that queries it
// should copy the entire SqlServer -- locking it in the process.

package cpuring

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

// CPU records the usage of CPU at a particular time
type CPU struct {
	At    time.Time `json:"at,omitempty"`
	SQL   int       `json:"sql,omitempty"`
	Other int       `json:"other,omitempty"`
}

// defaultCapacity of an uninitialized Ring buffer.
// Changing this value only affects ring buffers created after it is changed.
var defaultCapacity = 60

// Ring implements a Circular Buffer.
// The default value of the Ring struct is a valid (empty) Ring buffer with capacity DefaultCapacify.
type Ring struct {
	buff []*CPU
	head int
	tail int
}

// New returns a new Ring of size
func New(size int) Ring {
	var r Ring
	r.checkInit()
	r.extend(size)
	return r
}

// SetCapacity sets the maximum size of the ring buffer.
func (r *Ring) SetCapacity(size int) {
	r.checkInit()
	r.extend(size)
}

// Capacity returns the current capacity of the ring buffer.
func (r *Ring) Capacity() int {
	return len(r.buff)
}

// Len returns the length of the ring buffer
func (r *Ring) Len() int {
	if r.head == -1 {
		return 0
	}
	if r.head >= r.tail {
		return r.head - r.tail + 1
	}
	return len(r.buff)
}

// Enqueue a value into the Ring buffer.
func (r *Ring) Enqueue(i *CPU) {
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
func (r *Ring) Dequeue() *CPU {
	var w *CPU
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
func (r *Ring) Peek() *CPU {
	r.checkInit()
	var w *CPU
	if r.head == -1 {
		return w
	}
	return r.get(r.tail)
}

// GetNewest returns the most recently added value.  It returns false if
// there is no value to return.
func (r *Ring) GetNewest() (*CPU, bool) {
	var w *CPU
	r.checkInit()
	if r.head == -1 {
		return w, false
	}
	return r.get(r.head), true
}

// Values returns a slice of all the values in the circular buffer without modifying them at all.
// The returned slice can be modified independently of the circular buffer. However, the values inside the slice
// are shared between the slice and circular buffer.
func (r *Ring) Values() []*CPU {
	if r.head == -1 {
		return nil
	}
	arr := make([]*CPU, 0, r.Capacity())
	for i := 0; i < r.Capacity(); i++ {
		idx := r.mod(i + r.tail)
		arr = append(arr, r.get(idx))
		if idx == r.head {
			break
		}
	}
	return arr
}

// MarshalJSON marshals to a byte array
func (r Ring) MarshalJSON() ([]byte, error) {
	var wr struct {
		Buffer []*CPU `json:"buffer"`
		Head   int    `json:"head"`
		Tail   int    `json:"tail"`
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
func (r *Ring) UnmarshalJSON(b []byte) error {
	var wr struct {
		Buffer []*CPU `json:"buffer"`
		Head   int    `json:"head"`
		Tail   int    `json:"tail"`
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

func (cpu CPU) String() string {
	return fmt.Sprintf("%s: sql: %d other: %d", cpu.At, cpu.SQL, cpu.Other)
}

/*************************************************************************************************
*** Unexported methods beyond this point.
**************************************************************************************************/

// set a value at the given unmodified index and returns the modified index of the value
func (r *Ring) set(p int, v *CPU) {
	r.buff[r.mod(p)] = v
}

// get a value based at a given unmodified index
func (r *Ring) get(p int) *CPU {
	return r.buff[r.mod(p)]
}

// mod returns the modified index of an unmodified index
func (r *Ring) mod(p int) int {
	return p % len(r.buff)
}

// checkInit checks if the WaitGroup is initialized and inits if needed
func (r *Ring) checkInit() {
	var w *CPU
	if r.buff == nil {
		r.buff = make([]*CPU, defaultCapacity)
		for i := range r.buff {
			r.buff[i] = w
		}
		r.head, r.tail = -1, 0
	}
}

// extend the Ring to the specified size.  Will reduce
// the Ring if needed.
func (r *Ring) extend(size int) {
	if size == len(r.buff) {
		return
	} else if size < len(r.buff) {
		r.buff = r.buff[0:size]
	}
	newb := make([]*CPU, size-len(r.buff))

	for i := range newb {
		var w CPU
		newb[i] = &w
	}
	r.buff = append(r.buff, newb...)
}
