package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBackupMap(t *testing.T) {
	assert := assert.New(t)
	var src [][]string
	src = append(src, []string{"domain", "a"})
	src = append(src, []string{"domain", "b", "c"})
	src = append(src, []string{"domain", "b", "d"})
	m := ignored2Map(src)
	assert.Equal(2, len(m))
	assert.Equal(m["a"], []string{})
	assert.Equal(m["b"], []string{"c", "d"})
}
