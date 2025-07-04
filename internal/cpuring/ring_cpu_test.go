package cpuring

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCpuRing(t *testing.T) {
	assert := assert.New(t)
	r := New(3)
	assert.Equal(-1, r.head)
	assert.Equal(0, r.tail)
	assert.Equal(3, r.Capacity())
	assert.Equal(0, r.Len())
	assert.Equal(3, len(r.buff))

	cpu, found := r.GetNewest()
	assert.False(found)
	assert.Nil(cpu)

	r.Enqueue(&CPU{time.Now(), 1, 11})
	assert.Equal(1, r.Len())
	assert.Equal(0, r.head)
	assert.Equal(0, r.tail)

	r.Enqueue(&CPU{time.Now(), 2, 22})
	assert.Equal(2, r.Len())
	assert.Equal(1, r.head)
	assert.Equal(0, r.tail)

	r.Enqueue(&CPU{time.Now(), 3, 33})
	assert.Equal(3, r.Len())
	assert.Equal(2, r.head)
	assert.Equal(0, r.tail)

	r.Enqueue(&CPU{time.Now(), 4, 44})
	assert.Equal(3, r.Len())
	assert.Equal(0, r.head)
	assert.Equal(1, r.tail)

	r.Enqueue(&CPU{time.Now(), 5, 55})
	r.Enqueue(&CPU{time.Now(), 6, 66})
	r.Enqueue(&CPU{time.Now(), 7, 77})
	assert.Equal(3, r.Len())
	assert.Equal(0, r.head)
	assert.Equal(1, r.tail)

	oldest := r.Peek()
	assert.Equal(5, oldest.SQL)
	newest, exists := r.GetNewest()
	assert.Equal(7, newest.SQL)
	assert.True(exists)

}

func TestCpuRingMarshal(t *testing.T) {
	is := assert.New(t)
	r := New(3)
	r.Enqueue(&CPU{time.Now(), 5, 55})
	r.Enqueue(&CPU{time.Now(), 6, 66})
	bb, err := r.MarshalJSON()
	is.NoError(err)
	is.NotZero(len(bb))

	r2 := New(5)
	err = r2.UnmarshalJSON(bb)
	is.NoError(err)
	is.Equal(3, r2.Capacity())
	is.Equal(2, r2.Len())
}

func TestCPURingEmpty(t *testing.T) {
	is := assert.New(t)
	var r Ring
	v := r.Values()
	is.Equal(0, len(v))
	bb, err := r.MarshalJSON()
	is.NoError(err)
	is.NotZero(len(bb))

	var r2 Ring
	err = r2.UnmarshalJSON(bb)
	is.NoError(err)

	r2.Enqueue(&CPU{time.Now(), 5, 55})
}
