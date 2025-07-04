// /*
// Package ring provides a simple implementation of a ring buffer.
// */
package waitgroupring

// import (
// 	"encoding/json"
// 	"time"

// 	"github.com/pkg/errors"
// )

// // /*
// // The DefaultCapacity of an uninitialized Ring buffer.
// // Changing this value only affects ring buffers created after it is changed.
// // */
// // var DefaultCapacity = 60

// // SQLWaitGroup holds the events we want to enqueue
// type SQLWaitGroup struct {
// 	EventTime time.Time
// 	Duration  time.Duration
// 	Groups    map[string]int64
// }

// /*
// WaitGroupRing implements a Circular Buffer.
// The default value of the Ring struct is a valid (empty) Ring buffer with capacity DefaultCapacify.
// */
// type WaitGroupRing struct {
// 	head int // the most recent value written
// 	tail int // the least recent value written
// 	buff []*SQLWaitGroup
// }

// /*
// SetCapacity sets the maximum size of the ring buffer.
// */
// func (r *WaitGroupRing) SetCapacity(size int) {
// 	r.checkInit()
// 	r.extend(size)
// }

// /*
// Capacity returns the current capacity of the ring buffer.
// */
// func (r WaitGroupRing) Capacity() int {
// 	return len(r.buff)
// }

// /*
// Enqueue a value into the Ring buffer.
// */
// func (r *WaitGroupRing) Enqueue(i *SQLWaitGroup) {
// 	r.checkInit()
// 	r.set(r.head+1, i)
// 	old := r.head
// 	r.head = r.mod(r.head + 1)
// 	if old != -1 && r.head == r.tail {
// 		r.tail = r.mod(r.tail + 1)
// 	}
// }

// /*
// Dequeue a value from the Ring buffer.
// Returns nil if the ring buffer is empty.
// */
// func (r *WaitGroupRing) Dequeue() *SQLWaitGroup {
// 	var g *SQLWaitGroup
// 	r.checkInit()
// 	if r.head == -1 {
// 		return g
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
// Peek reads the value that Dequeue would have dequeued without actually dequeuing it.
// Returns nil if the ring buffer is empty.
// */
// func (r *WaitGroupRing) Peek() *SQLWaitGroup {
// 	r.checkInit()
// 	var g *SQLWaitGroup
// 	if r.head == -1 {
// 		return g
// 	}
// 	return r.get(r.tail)
// }

// /*
// GetNewest returns the most recently enqueued value
// */
// func (r *WaitGroupRing) GetNewest() (*SQLWaitGroup, bool) {
// 	var g *SQLWaitGroup
// 	r.checkInit()
// 	if r.head == -1 {
// 		return g, false
// 	}
// 	return r.get(r.head), true
// }

// /*
// Values returns a slice of all the values in the circular buffer without modifying them at all.
// The returned slice can be modified independently of the circular buffer. However, the values inside the slice
// are shared between the slice and circular buffer.
// */
// func (r *WaitGroupRing) Values() []*SQLWaitGroup {
// 	if r.head == -1 {
// 		return nil
// 	}
// 	arr := make([]*SQLWaitGroup, 0, r.Capacity())
// 	for i := 0; i < r.Capacity(); i++ {
// 		idx := r.mod(i + r.tail)
// 		arr = append(arr, r.get(idx))
// 		if idx == r.head {
// 			break
// 		}
// 	}
// 	return arr
// }

// // TODO write a custom marsheller & unmarshaller
// // MarshalJSON brings back the JSON
// // func (r *WaitGroupRing) MarshalJSON() ([]byte, error) {
// // 	return json.Marshal(&struct {
// // 		ID   int64  `json:"id"`
// // 		Name string `json:"name"`
// // 	}{
// // 		ID:   2,
// // 		Name: "waitGroupRing",
// // 	})
// // }

// // MarshalJSON brings back the JSON
// func (r *WaitGroupRing) MarshalJSON() ([]byte, error) {
// 	var err error
// 	v := r.Values()
// 	j, err := json.Marshal(v)
// 	return j, err
// }

// // UnmarshalJSON unmarshalls the JSON
// func (r *WaitGroupRing) UnmarshalJSON(b []byte) error {
// 	//type h8 []*SQLWaitGroup

// 	//fmt.Println("unmarshalling wait group rings...")
// 	//r = &WaitGroupRing{}
// 	var err error
// 	var h8 []*SQLWaitGroup

// 	err = json.Unmarshal(b, &h8)
// 	if err != nil {
// 		//fmt.Println("err: ", err)
// 		return errors.Wrap(err, "unmarshallx")
// 	}

// 	//fmt.Println("wgr: ", len(h8), h8)
// 	for _, v := range h8 {
// 		//fmt.Println(k, v)
// 		r.Enqueue(v)
// 	}

// 	//r = &wgr
// 	// z := wgr.Values()
// 	// for k, v := range z {
// 	// 	fmt.Println(k, v)
// 	// }

// 	return nil
// }

// // func (r *WaitGroupRing) GetTopGroups() (topWaits SortedMapInt64, err error) {

// // 	wgl := make(map[string]int64)
// // 	var ok bool
// // 	//var wurg WaitDisplay
// // 	//t := servers.Servers[server].Waits
// // 	//wv := t.Values()

// // 	// Group all wait groups together

// // 	v := r.Values()
// // 	for _, wv := range v {

// // 		for wg, duration := range wv.WaitSummary {
// // 			_, ok = wgl[wg]
// // 			if duration > 0 {
// // 				//log.println("X")
// // 				if ok {
// // 					wgl[wg] += duration
// // 				} else {
// // 					wgl[wg] = duration
// // 				}
// // 			}
// // 		}
// // 	}

// // 	//sm := sortedMap { m: wgl, s: make([]string, len(wgl))}

// // 	//func sortedKeys(m map[string]int64) []string {
// // 	sm := new(SortedMapInt64)
// // 	sm.BaseMap = wgl
// // 	sm.SortedKeys = make([]string, len(wgl))
// // 	i := 0
// // 	for key := range sm.BaseMap {
// // 		sm.SortedKeys[i] = key
// // 		i++
// // 	}
// // 	sort.Sort(sm)
// // 	//sortedMap := sm
// // 	//return sm.s
// // 	//}

// // 	return *sm, nil
// // }

// // func (r *WaitGroupRing) NonZeroValues() []*SqlWaitGroup {
// // 	if r.head == -1 {
// // 		return nil
// // 	}
// // 	arr := make([]*SqlWaitGroup, 0, r.Capacity())
// // 	for i := 0; i < r.Capacity(); i++ {
// // 		idx := r.mod(i + r.tail)

// // 		w := r.get(idx)
// // 		for key, value := range w.Waits {
// // 			if value.WaitTimeDelta == 0 {
// // 				delete(w.Waits, key)
// // 			}
// // 		}

// // 		arr = append(arr, w)
// // 		if idx == r.head {
// // 			break
// // 		}
// // 	}
// // 	return arr
// // }

// /*************************************************************************************************
// *** Unexported methods beyond this point.
// **************************************************************************************************/

// // sets a value at the given unmodified index and returns the modified index of the value
// func (r *WaitGroupRing) set(p int, v *SQLWaitGroup) {
// 	r.buff[r.mod(p)] = v
// }

// // gets a value based at a given unmodified index
// func (r *WaitGroupRing) get(p int) *SQLWaitGroup {
// 	return r.buff[r.mod(p)]
// }

// // returns the modified index of an unmodified index
// func (r *WaitGroupRing) mod(p int) int {
// 	return p % len(r.buff)
// }

// func (r *WaitGroupRing) checkInit() {
// 	var w *SQLWaitGroup
// 	if r.buff == nil {
// 		r.buff = make([]*SQLWaitGroup, 60)
// 		for i := range r.buff {
// 			r.buff[i] = w
// 		}
// 		r.head, r.tail = -1, 0
// 	}
// }

// func (r *WaitGroupRing) extend(size int) {
// 	if size == len(r.buff) {
// 		return
// 	} else if size < len(r.buff) {
// 		r.buff = r.buff[0:size]
// 	}
// 	newb := make([]*SQLWaitGroup, size-len(r.buff))

// 	for i := range newb {
// 		var w SQLWaitGroup
// 		newb[i] = &w
// 	}
// 	r.buff = append(r.buff, newb...)
// }
