package c2

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFixBackslashes(t *testing.T) {
	assert := assert.New(t)
	type test struct {
		in  []byte
		bb  []byte
		str string
	}
	tests := []test{
		{[]byte("a"), []byte("a"), "a"},
		{[]byte(`a\b`), []byte(`a\\b`), "a\\\\b"},
		{[]byte(`a\\b`), []byte(`a\\b`), "a\\\\b"},
	}
	for _, tc := range tests {
		got := FixSlashes(tc.in)
		println(tc.in)
		fmt.Printf("got: %v\n", got)
		println(string(got))
		assert.Equal(tc.bb, got, "bytes don't match")
		assert.Equal(tc.str, string(got), "string doesn't match")
	}
}
