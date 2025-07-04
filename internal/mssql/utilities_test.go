package mssql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitServerName(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		name     string
		host     string
		instance string
	}{
		{"a", "a", ""},
		{"a.b.c", "a.b.c", ""},
		{"a.b.c\\test", "a.b.c", "test"},
		{"a.b.c\\test\\extra", "a.b.c", "test"},
	}
	for _, tc := range tests {
		host, instance := SplitServerName(tc.name)
		assert.Equal(tc.host, host)
		assert.Equal(tc.instance, instance)
	}
}
