//Package metric provides a simple implementation of a ring buffer.
package metric

import (
	// "sync"
	// "encoding/json"
	"github.com/pkg/errors"
	//"time"
)

func init() {
	// metrics = make(map[Key]*ValueRing)
	reset()
}

// Enqueue adds a value
func Enqueue(k Key, v Value) error {
	mux.Lock()
	defer mux.Unlock()

	var vr *ValueRing
	vr, ok := metrics[k]
	if !ok {
		vr := &ValueRing{}
		vr.enqueue(&v)
		//fmt.Println("enqueued value: ", v)
		return nil
	}
	// Get the previous
	// Figure out the difference
	// Enqueue
	vr.enqueue(&v)
	//fmt.Println("enqueued value: ", v)
	return nil
}

// Values returns an array of values
func Values(k Key) ([]Value, error) {
	mux.RLock()
	defer mux.RUnlock()

	//var err error
	var vr *ValueRing
	//fmt.Println(metrics)
	vr, ok := metrics[k]
	if !ok {
		return nil, errors.New("invalid key")
	}

	if vr.head == -1 {
		return nil, errors.New("empty array")
	}

	arr := make([]Value, 0, vr.capacity())
	for i := 0; i < vr.capacity(); i++ {
		idx := vr.mod(i + vr.tail)
		arr = append(arr, *vr.get(idx))
		if idx == vr.head {
			break
		}
	}
	return arr, nil
}

/*
V returns a slice of all the values in the circular buffer without modifying them at all.
The returned slice can be modified independently of the circular buffer. However, the values inside the slice
are shared between the slice and circular buffer.
*/
func (r *ValueRing) V() []*Value {
	mux.RLock()
	defer mux.RUnlock()

	if r.head == -1 {
		return nil
	}
	arr := make([]*Value, 0, r.capacity())
	for i := 0; i < r.capacity(); i++ {
		idx := r.mod(i + r.tail)
		arr = append(arr, r.get(idx))
		if idx == r.head {
			break
		}
	}
	return arr
}

// Reset deletes all value
func Reset() {
	mux.Lock()
	defer mux.Unlock()
	reset()
}

func reset() {
	metrics = make(map[Key]*ValueRing)
}

//SetCapacity sets the maximum size of the ring buffer.
func (r *ValueRing) SetCapacity(size int) {
	// r.Lock()
	// defer r.Unlock()
	r.checkInit()
	r.extend(size)
}

/*
Peek reads the value that Dequeue would have dequeued
without actually dequeuing it.
Returns nil if the ring buffer is empty.
*/
func (r *ValueRing) Peek() *Value {
	r.checkInit()
	if r.head == -1 {
		return nil
	}
	return r.get(r.tail)
}

// // MarshalJSON brings back the JSON
// func (r ValueRing) MarshalJSON() ([]byte, error) {
// 	//fmt.Println("ValueRing: marshaljson")
// 	var err error
// 	v := r.Values()
// 	j, err := json.Marshal(v)
// 	if err != nil {
// 		err = errors.Wrap(err, "ValueRing")
// 	}
// 	return j, err
// }

// //UnmarshalJSON unmarshalls the JSON
// func (r ValueRing) UnmarshalJSON(b []byte) error {
// 	fmt.Println("ValueRing: UNmarshaljson")
// 	r = ValueRing{}
// 	return nil
// }

// // UnmarshalJSON unmarshalls the JSON
// func (r *ValueRing) UnmarshalJSON(b []byte) error {
// 	//type h8 []*SQLWaitGroup

// 	//fmt.Println("unmarshalling wait group rings...")
// 	//x := ValueRing{}
// 	var err error
// 	var h8 []*Value

// 	err = json.Unmarshal(b, &h8)
// 	if err != nil {
// 		//fmt.Println("err: ", err)
// 		return errors.Wrap(err, "unmarshall")
// 	}

// 	for _, v := range h8 {
// 		// if v.PolledValue {
// 		// 	fmt.Println("mvr ", k, v)
// 		// }
// 		r.Enqueue(v)
// 	}

// 	//r = x

// 	//r = &wgr
// 	// z := wgr.Values()
// 	// for k, v := range z {
// 	// 	fmt.Println(k, v)
// 	// }

// 	return nil
// }
