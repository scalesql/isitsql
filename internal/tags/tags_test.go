package tags

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMerge(t *testing.T) {
	assert := assert.New(t)
	m := Merge(&[]string{"a"}, &[]string{"b", "a"})
	assert.Equal([]string{"a", "b"}, m)
}

func TestMergeNil(t *testing.T) {
	assert := assert.New(t)
	m := Merge(nil, &[]string{"c"}, &[]string{"b", "a"}, nil, &[]string{"A"})
	assert.Equal([]string{"a", "b", "c"}, m)
}

func TestMergeNoVals(t *testing.T) {
	assert := assert.New(t)
	m := Merge(nil, nil)
	assert.Equal([]string{}, m)
}
