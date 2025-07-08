package waitring

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWaitRingTop(t *testing.T) {

}
func TestWaitRing(t *testing.T) {
	assert := assert.New(t)
	r := New(5)

	last := r.Last()
	assert.Equal(0, len(last.Waits)) // no waits yet

	for i := 0; i < 16; i++ {
		ts := time.Unix(0, 0)
		ts = ts.Add(time.Duration(i) * time.Minute)
		wm := WaitList{TS: ts, Waits: make(map[string]int64)}
		r.Enqueue(wm)
		v := r.Values()
		last := r.Last()
		if i <= 4 {
			assert.Equal(i+1, len(v))              // len is the number of inserts
			assert.Equal(time.Unix(0, 0), v[0].TS) // first entry is 0, 0
			assert.Equal(ts, v[i].TS)              // last entry is last inserted
			assert.Equal(ts, last.TS)              // last entry is last inserted
		} else {
			assert.Equal(5, len(v))                       // we always have five
			assert.Equal(ts.Add(-4*time.Minute), v[0].TS) // first entry is four minutes ago (five total values)
			assert.Equal(ts, v[4].TS)                     // last entry is last inserted
			assert.Equal(ts, last.TS)                     // last entry is last inserted
		}
	}
}

func TestWaitRingMarshal(t *testing.T) {
	assert := assert.New(t)
	r := New(5)
	ts := time.Unix(0, 0)
	m := map[string]int64{"v1": 7}
	wm := WaitList{TS: ts, Waits: m}
	r.Enqueue(wm)
	bb, err := json.Marshal(&r)
	assert.NoError(err)
	assert.Equal(`{"data":[{"ts":"1969-12-31T18:00:00-06:00","waits":{"v1":7}},{"ts":"0001-01-01T00:00:00Z","waits":null},{"ts":"0001-01-01T00:00:00Z","waits":null},{"ts":"0001-01-01T00:00:00Z","waits":null},{"ts":"0001-01-01T00:00:00Z","waits":null}],"p":1}`, string(bb))

	r0 := New(5)
	err = json.Unmarshal(bb, &r0)
	assert.NoError(err)
	assert.Equal(1, r0.p)
	assert.Equal(time.Unix(0, 0), r.data[0].TS)
	assert.Equal(1, len(r.data[0].Waits))
}
