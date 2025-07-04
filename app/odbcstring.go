package app

import (
	"errors"
	"strings"
)

// GetServer Get the server name from a connection string
func getCXServerName(cx string) (string, error) {

	// var s string
	// var err error

	// Get each ; separated part
	parts := strings.Split(cx, ";")

	// Look in each part for the server
	for _, p := range parts {
		// fmt.Println(p)

		// split each on the =
		pair := strings.Split(p, "=")
		if len(pair) != 2 {
			break
		}

		attr := strings.ToLower(strings.TrimSpace(pair[0]))
		value := strings.TrimSpace(pair[1])

		if attr == "server" {
			return value, nil
		}
	}

	return "", errors.New("server name not found")
}
