package mssql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	assert := assert.New(t)
	v := VersionToString("1.2.3")
	assert.Equal("00000001000000020000000300000000", v)
}
