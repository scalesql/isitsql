package agent

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// agentTime converts date and time integers to a GO `time.Time`.
func agentTime(dt, tm int32) (time.Time, error) {
	if dt == 0 {
		return time.Time{}, nil
	}
	dtstr := fmt.Sprintf("00000000%d", dt)
	dtstr = dtstr[len(dtstr)-8:]
	tmstr := fmt.Sprintf("000000%d", tm)
	tmstr = tmstr[len(tmstr)-6:]
	fullTime, err := time.Parse("20060102150405", dtstr+tmstr)
	if err != nil {
		return time.Time{}, err
	}
	return fullTime, nil
}

// agentDuration returns an integer duration in hhmmss in a native time.Duration
func agentDuration(dur int32) (time.Duration, error) {
	if dur == 0 {
		return time.Duration(0), nil
	}
	// add zeros
	str := strconv.Itoa(int(dur))
	if len(str) < 6 {
		str = strings.Repeat("0", 6-len(str)) + str
	}

	// should have a string of len  6 or greater
	if len(str) < 6 {
		return time.Duration(0), fmt.Errorf("invalid duration: %d", dur)
	}
	// convert to xxhxxmxxs
	if len(str) == 6 {
		base := fmt.Sprintf("%sh%sm%ss", str[0:2], str[2:4], str[4:6])
		d2, err := time.ParseDuration(base)
		if err != nil {
			return time.Duration(0), fmt.Errorf("invalid duration: %d (%s)", dur, base)
		}
		return d2, nil
	}
	// len > 6
	hoursLen := len(str) - 4
	hh := str[0:hoursLen]
	mm := str[hoursLen : hoursLen+2]
	ss := str[hoursLen+2 : hoursLen+4]
	base := fmt.Sprintf("%sh%sm%ss", hh, mm, ss)
	d2, err := time.ParseDuration(base)
	if err != nil {
		return time.Duration(0), fmt.Errorf("invalid duration: %d (%s)", dur, base)
	}
	return d2, nil
}
