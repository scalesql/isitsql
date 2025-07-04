package logring

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLogRingThree(t *testing.T) {
	require := require.New(t)
	var v []Event
	r := New(3)
	require.Equal(0, r.Size())
	v = r.Values()
	require.Equal([]Event{}, v)
	r.enqueue("a")
	require.Equal(1, r.ptr)
	require.Equal(1, r.Size())
	v = r.Values()
	require.Equal("a", v[0].Message)
	r.enqueue("b")
	_ = r.newest() // newest was sorting the base buffer
	require.Equal("a", v[0].Message)
	r.enqueue("c")
	_ = r.newest() // newest was sorting the base buff
	require.Equal("a", v[0].Message)
	fmt.Printf("[a,b,c]: %+v\n", r.buff)
	r.enqueue("d")
	require.Equal(1, r.ptr%r.capacity)
	fmt.Printf("[d,b,c]: %+v\n", r.buff)
	r.Enqueuef("%s", "e")
	require.Equal(5, r.ptr)
	require.Equal(3, r.Size())
	v = r.Values()
	//fmt.Printf("[d,e,c]: %+v\n", r.buff)
	//fmt.Printf("[c,d,e]: %+v\n", v)
	require.Equal(3, len(v))
	require.Equal("c", v[0].Message)
	require.Equal("d", v[1].Message)
	require.Equal("e", v[2].Message)
	r.enqueue("f")
	v = r.Values()
	require.Equal("d", v[0].Message)
	require.Equal("e", v[1].Message)
	require.Equal("f", v[2].Message)
	r.Enqueue("g")
	v = r.Values()
	require.Equal("e", v[0].Message)
	require.Equal("f", v[1].Message)
	require.Equal("g", v[2].Message)

	v = r.Newest()
	require.Equal("g", v[0].Message)
	require.Equal("f", v[1].Message)
	require.Equal("e", v[2].Message)
	require.Equal(3, len(r.buff))

}

func TestLogRingFive(t *testing.T) {
	require := require.New(t)
	var v []Event
	r := New(5)
	require.Equal(0, r.Size())
	v = r.Values()
	require.Equal([]Event{}, v)
	r.enqueue("a")
	require.Equal(1, r.ptr)
	require.Equal(1, r.Size())
	v = r.Values()
	require.Equal("a", v[0].Message)
	r.enqueue("b")
	r.enqueue("c")
	r.enqueue("d")
	r.Enqueuef("%s", "e")
	r.enqueue("f")
	r.Enqueue("g")
	r.Enqueue("h")
	ee := r.Values()
	require.Equal("d", ee[0].Message)
}
