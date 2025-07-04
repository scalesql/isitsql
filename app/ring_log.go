package app

import (
	"sync"
	"time"
)

/*
The DefaultCapacity of an uninitialized Ring buffer.
Changing this value only affects ring buffers created after it is changed.
*/
//var DefaultCapacity int = 10

/*
Type Ring implements a Circular Buffer.
The default value of the Ring struct is a valid (empty) Ring buffer with capacity DefaultCapacify.
*/

type RingLogEvent struct {
	LogTime time.Time
	Message string
}

type RingLog struct {
	sync.RWMutex
	buff []RingLogEvent
	head int // the most recent value written
	tail int // the least recent value written
}

/*
Set the maximum size of the ring buffer.
*/
// func (r *RingLog) SetCapacity(size int) {
// 	r.checkInit()
// 	r.extend(size)
// }

/*
Capacity returns the current capacity of the ring buffer.
*/
func (r *RingLog) capacity() int {
	return len(r.buff)
}

/*
Enqueue a value into the Ring buffer.
*/
func (r *RingLog) Enqueue(m string) {

	e := RingLogEvent{LogTime: time.Now(), Message: m}
	r.Lock()
	defer r.Unlock()

	r.checkInit()
	r.set(r.head+1, e)
	old := r.head
	r.head = r.mod(r.head + 1)
	if old != -1 && r.head == r.tail {
		r.tail = r.mod(r.tail + 1)
	}
}

/*
Dequeue a value from the Ring buffer.
Returns nil if the ring buffer is empty.
*/
// func (r *Ring) Dequeue() RingLogEvent {
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

// /*
// Read the value that Dequeue would have dequeued without actually dequeuing it.
// Returns nil if the ring buffer is empty.
// */
// func (r *Ring) Peek() RingLogEvent {
// 	r.checkInit()
// 	if r.head == -1 {
// 		return nil
// 	}
// 	return r.get(r.tail)
// }

/*
Values returns a slice of all the values in the circular buffer without modifying them at all.
The returned slice can be modified independently of the circular buffer. However, the values inside the slice
are shared between the slice and circular buffer.
*/
func (r *RingLog) Values() []RingLogEvent {
	r.RLock()
	defer r.RUnlock()

	if r.head == -1 {
		return []RingLogEvent{}
	}
	arr := make([]RingLogEvent, 0, r.capacity())
	for i := 0; i < r.capacity(); i++ {
		idx := r.mod(i + r.tail)
		arr = append(arr, r.get(idx))
		if idx == r.head {
			break
		}
	}
	return arr
}

func (r *RingLog) NewestValues() []RingLogEvent {
	r.RLock()
	defer r.RUnlock()

	if r.head == -1 {
		return []RingLogEvent{}
	}
	arr := make([]RingLogEvent, 0, r.capacity())
	for t := 0; t < r.capacity(); t++ {
		idx := r.mod(t + r.tail)
		arr = append(arr, r.get(idx))
		if idx == r.head {
			break
		}
	}

	// reverse the array
	for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
		arr[i], arr[j] = arr[j], arr[i]
	}

	return arr
}

/**
*** Unexported methods beyond this point.
**/

// sets a value at the given unmodified index and returns the modified index of the value
func (r *RingLog) set(p int, v RingLogEvent) {

	r.buff[r.mod(p)] = v
}

// gets a value based at a given unmodified index
func (r *RingLog) get(p int) RingLogEvent {
	return r.buff[r.mod(p)]
}

// returns the modified index of an unmodified index
func (r *RingLog) mod(p int) int {
	return p % len(r.buff)
}

func (r *RingLog) checkInit() {

	if r.buff == nil {
		r.buff = make([]RingLogEvent, 1000)
		// for i := range r.buff {
		// 	r.buff[i] = nil
		// }
		r.head, r.tail = -1, 0
	}
}

// func (r *RingLog) extend(size int) {

// 	if size == len(r.buff) {
// 		return
// 	} else if size < len(r.buff) {
// 		r.buff = r.buff[0:size]
// 	}
// 	newb := make([]RingLogEvent, size-len(r.buff))
// 	// for i := range newb {
// 	// 	newb[i] = nil
// 	// }
// 	r.buff = append(r.buff, newb...)
// }
