package gui

import (
	"testing"
)

func TestDuration(t *testing.T) {

	var durationTests = []struct {
		s        int
		expected string
	}{
		{260000, "3d"},
		{252000, "70h"},
		{-1, "Invalid(-1)"},
		{301, "5m"},
		{300, "300s"},
		{1, "1s"},
		{0, "0s"},
	}

	for _, dt := range durationTests {
		actual := SecondsToShortString(dt.s)
		if actual != dt.expected {
			t.Errorf("secondsToShortString(%d): expected %s, actual %s", dt.s, dt.expected, actual)
		}
	}

}
