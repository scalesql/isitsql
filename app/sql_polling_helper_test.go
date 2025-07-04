package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerVersion(t *testing.T) {
	assert := assert.New(t)
	s := SqlServer{}
	assert.Equal("SQL Server 2016", s.ProductVersionString("13.0"))
	assert.Equal("SQL Server Unknown", s.ProductVersionString(""))
	assert.Equal("SQL Server 17.0", s.ProductVersionString("17.0"))
	assert.Equal("SQL Server 7.0", s.ProductVersionString("7.0"))
	assert.Equal("SQL Server 2022", s.ProductVersionString("16.0.600.9"))
}
