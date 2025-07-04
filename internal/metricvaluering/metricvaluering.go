/*
Package metricvaluering is a ring buffer for metrics.
*/
package metricvaluering

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
)

/*
The DefaultCapacity of an uninitialized Ring buffer.

Changing this value only affects ring buffers created after it is changed.
*/
//var DefaultCapacity = 60

/*
MetricValueRing implements a Circular Buffer.
The default value of the Ring struct is a valid (empty) Ring buffer with capacity DefaultCapacify.
*/
type MetricValueRing struct {
	buff []*MetricValue
	head int // the most recent value written
	tail int // the least recent value written

	//sync.RWMutex // all locking is done at the server level
}

type MetricValue struct {
	EventTime      time.Time     `json:"event_time,omitempty"`
	Value          int64         `json:"value,omitempty"`
	AggregateValue int64         `json:"aggregate_value,omitempty"`
	DeltaDuration  time.Duration `json:"delta_duration,omitempty"`
	PolledValue    bool          `json:"polled_value,omitempty"`
	ValuePerSecond int64         `json:"value_per_second,omitempty"`
}

/*
SetCapacity sets the maximum size of the ring buffer.
*/
func (r *MetricValueRing) SetCapacity(size int) {
	// r.Lock()
	// defer r.Unlock()
	r.checkInit()
	r.extend(size)
}

/*
Capacity returns the current capacity of the ring buffer.
*/
func (r *MetricValueRing) Capacity() int {
	// r.RLock()
	// defer r.RUnlock()
	return len(r.buff)
}

/*
Enqueue a value into the Ring buffer.
*/
func (r *MetricValueRing) Enqueue(i *MetricValue) {
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
Dequeue a value from the Ring buffer.

Returns nil if the ring buffer is empty.
*/
func (r *MetricValueRing) Dequeue() *MetricValue {
	// r.Lock()
	// defer r.Unlock()
	r.checkInit()
	if r.head == -1 {
		return nil
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

/*
Read the value that Dequeue would have dequeued without actually dequeuing it.

Returns nil if the ring buffer is empty.
*/
// func (r *MetricValueRing) Peek() *MetricValue {
// 	r.checkInit()
// 	if r.head == -1 {
// 		return nil
// 	}
// 	return r.get(r.tail)
// }

/*
GetLastValue returns the most recently added value

Returns nil if the ring buffer is empty.
*/
func (r *MetricValueRing) GetLastValue() *MetricValue {
	// r.RLock()
	// defer r.RUnlock()

	r.checkInit()
	if r.head == -1 {
		return nil
	}
	return r.get(r.head)
}

/*
Values returns a slice of all the values in the circular buffer without modifying them at all.
The returned slice can be modified independently of the circular buffer. However, the values inside the slice
are shared between the slice and circular buffer.
*/
func (r *MetricValueRing) Values() []*MetricValue {
	// r.RLock()
	// defer r.RUnlock()

	if r.head == -1 {
		return nil
	}
	arr := make([]*MetricValue, 0, r.Capacity())
	for i := 0; i < r.Capacity(); i++ {
		idx := r.mod(i + r.tail)
		arr = append(arr, r.get(idx))
		if idx == r.head {
			break
		}
	}
	return arr
}

// MarshalJSON brings back the JSON
func (r MetricValueRing) MarshalJSON() ([]byte, error) {
	//fmt.Println("metricvaluering: marshaljson")
	var err error
	v := r.Values()
	j, err := json.Marshal(v)
	if err != nil {
		err = errors.Wrap(err, "metricvaluering")
	}
	return j, err
}

// UnmarshalJSON unmarshalls the JSON
// func (r MetricValueRing) UnmarshalJSON(b []byte) error {
// 	fmt.Println("metricvaluering: UNmarshaljson")
// 	r = MetricValueRing{}
// 	return nil
// }

// UnmarshalJSON unmarshalls the JSON
func (r *MetricValueRing) UnmarshalJSON(b []byte) error {
	//type h8 []*SQLWaitGroup

	//fmt.Println("unmarshalling wait group rings...")
	//x := MetricValueRing{}
	var err error
	var h8 []*MetricValue

	err = json.Unmarshal(b, &h8)
	if err != nil {
		//fmt.Println("err: ", err)
		return errors.Wrap(err, "unmarshall")
	}

	for _, v := range h8 {
		// if v.PolledValue {
		// 	fmt.Println("mvr ", k, v)
		// }
		r.Enqueue(v)
	}

	//r = x

	//r = &wgr
	// z := wgr.Values()
	// for k, v := range z {
	// 	fmt.Println(k, v)
	// }

	return nil
}

/***********************************************************************************************************
*** Unexported methods beyond this point.
************************************************************************************************************/

// sets a value at the given unmodified index and returns the modified index of the value
func (r *MetricValueRing) set(p int, v *MetricValue) {
	r.buff[r.mod(p)] = v
}

// gets a value based at a given unmodified index
func (r *MetricValueRing) get(p int) *MetricValue {
	return r.buff[r.mod(p)]
}

// returns the modified index of an unmodified index
func (r *MetricValueRing) mod(p int) int {
	return p % len(r.buff)
}

func (r *MetricValueRing) checkInit() {
	if r.buff == nil {
		r.buff = make([]*MetricValue, 60)
		for i := range r.buff {
			r.buff[i] = nil
		}
		r.head, r.tail = -1, 0
	}
}

func (r *MetricValueRing) extend(size int) {
	if size == len(r.buff) {
		return
	} else if size < len(r.buff) {
		r.buff = r.buff[0:size]
	}
	newb := make([]*MetricValue, size-len(r.buff))
	for i := range newb {
		newb[i] = nil
	}
	r.buff = append(r.buff, newb...)
}
