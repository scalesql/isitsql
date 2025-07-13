package mrepo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDateTruncate(t *testing.T) {
	assert := assert.New(t)
	dt := time.Date(2025, 1, 2, 11, 34, 56, 0, time.UTC)
	truncated := truncateDate(dt)
	assert.Equal(dt.Day(), truncated.Day())

	dt = time.Date(2025, 1, 2, 23, 59, 59, 0, time.UTC)
	truncated = truncateDate(dt)
	assert.Equal(dt.Day(), truncated.Day())

	dt = time.Date(2025, 1, 2, 23, 59, 59, 0, time.Local)
	truncated = truncateDate(dt)
	assert.Equal(dt.Day(), truncated.Day())
}

func TestTruncateTime(t *testing.T) {
	assert := assert.New(t)
	dt := time.Date(2025, 1, 2, 11, 34, 56, 0, time.UTC)
	truncated := dt.Truncate(time.Minute)
	assert.Equal(34, truncated.Minute())
	assert.Equal(0, truncated.Second())
	assert.Equal(0, truncated.Nanosecond())

	dt = time.Date(2025, 1, 2, 23, 59, 59, 999999999, time.Local)
	truncated = dt.Truncate(time.Minute)
	assert.Equal(59, truncated.Minute())
	truncated = dt.Truncate(time.Minute)
	assert.Equal(0, truncated.Second())
	assert.Equal(0, truncated.Nanosecond())
}
