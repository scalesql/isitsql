package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIdentifier(t *testing.T) {
	assert := assert.New(t)
	type test struct {
		in    string
		error bool
		out   string
	}
	tests := []test{
		{"abc", false, "abc"},
		{"a", true, ""},
		{"ab", false, "ab"},
		{"$$$", true, ""},
		{"abc ", false, "abc"},
		{"kceus-dbtxn01p", false, "kceus-dbtxn01p"},
		{"kceus_dbtxn01p", false, "kceus_dbtxn01p"},
		{"host.name", false, "host.name"},
		{"s2", false, "s2"},
		{"T1", false, "t1"},
		{"ca7b0c9d-b77a-4a05-a393-7fc505ce659d", false, "ca7b0c9d-b77a-4a05-a393-7fc505ce659d"},
	}
	for _, tc := range tests {
		id, err := NewIdentifier(tc.in)
		if tc.error {
			assert.Error(err)
		} else {
			assert.NoError(err)
		}
		assert.Equal(Identifier(tc.out), id)

	}
}
