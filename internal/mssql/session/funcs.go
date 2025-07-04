package session

import "fmt"

func secondsToShortString(s int) string {
	// > 72 hours is days
	if s > 259200 {
		u := s / (24 * 3600)
		return fmt.Sprintf("%dd", u)
	}

	if s > 10800 {
		u := s / (3600)
		return fmt.Sprintf("%dh", u)
	}

	if s > 300 {
		u := s / (60)
		return fmt.Sprintf("%dm", u)
	}
	// 5m - 180m
	// 0 - 300 seconds -> seconds

	if s >= 0 {
		return fmt.Sprintf("%ds", s)
	}

	return fmt.Sprintf("Invalid(%d)", s)

}
