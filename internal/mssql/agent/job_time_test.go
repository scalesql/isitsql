package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentTime(t *testing.T) {
	assert := assert.New(t)
	type test struct {
		dt   int32
		tm   int32
		want time.Time
		ok   bool
	}

	tests := []test{
		{0, 0, time.Time{}, true},
		{20211003, 230000, time.Date(2021, 10, 3, 23, 0, 0, 0, time.UTC), true},
		{20241019, 0, time.Date(2024, 10, 19, 0, 0, 0, 0, time.UTC), true},
	}

	for _, tc := range tests {
		got, err := agentTime(tc.dt, tc.tm)
		if tc.ok {
			assert.NoError(err)
		} else {
			assert.Error(err)
		}
		assert.Equal(tc.want, got)
	}
}

func TestAgentDuration(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	type test struct {
		have int32
		want string
		ok   bool
	}

	tests := []test{
		{0, "0s", true},
		{1, "1s", true},
		{12300, "1h23m", true},
		{6100, "61m", true},
		{62, "62s", true},
		{6163, "1h2m3s", true},
		{250000, "25h", true},
		{99_00_00, "99h", true},
		{100_00_00, "100h", true}, // three digit hours
		{101_65_62, "102h6m2s", true},
	}

	for _, tc := range tests {
		want, err := time.ParseDuration(tc.want)
		require.NoError(err)
		got, err := agentDuration(tc.have)
		if tc.ok {
			assert.NoError(err)
		} else {
			assert.Error(err)
		}
		assert.Equal(want, got, "have: %d  want: %s got: %v", tc.have, tc.want, got)
	}
}
