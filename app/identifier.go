package app

import (
	"errors"
	"regexp"
	"strings"
)

// Identifier is primarily used as a map key
type Identifier string

// NewIdentifier returns and Identifier from a string
func NewIdentifier(val string) (Identifier, error) {
	val = strings.TrimSpace(val)
	val = strings.ToLower(val)
	id := Identifier(val)
	if !id.isvalid() {
		return "", errors.New("invalid identifer")
	}

	return id, nil
}

func (id Identifier) isvalid() bool {
	var idregex = regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9-_\.]*[a-zA-Z0-9-_])$`)
	return idregex.MatchString(string(id))
}

// String returns the string value of an identifier
func (id Identifier) String() string {
	return string(id)
}
