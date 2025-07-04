package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSemver(t *testing.T) {
	assert := assert.New(t)
	s := SqlServer{ProductVersion: "15.0.2345.2"}
	assert.Equal("15.0.2345", s.Semver())

	s.ProductVersion = "15.0"
	assert.Equal("15.0", s.Semver())
}

func TestURL(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		srv  SqlServer
		want string
	}{
		{SqlServer{MapKey: "abc-def"}, "/server/abc-def"},
	}
	for _, tc := range tests {
		assert.Equal(tc.want, tc.srv.URL())
	}
}

func TestSlug(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		srv  SqlServer
		want string
	}{
		{SqlServer{MapKey: "abc-def"}, "abc-def"},
		{SqlServer{MapKey: "abc-def", SlugOverride: "over"}, "over"},
		{SqlServer{MapKey: "abc-def", FriendlyName: "bill"}, "bill"},
		{SqlServer{MapKey: "abc-def", FQDN: "10.10.2.10"}, "abc-def"},
		{SqlServer{MapKey: "abc-def", FQDN: "server"}, "server"},
		{SqlServer{MapKey: "abc-def", FQDN: "server:1433"}, "abc-def"},
		{SqlServer{MapKey: "abc-def", FQDN: "server\\instance"}, "server/instance"},
		{SqlServer{MapKey: "abc-def", FQDN: "host.domain.com"}, "host"},
		{SqlServer{MapKey: "abc-def", FQDN: "db-txn.domain.com"}, "db-txn"},
		{SqlServer{MapKey: "abc-def", FQDN: "db-txn.domain.com\\NEW"}, "db-txn/new"},
	}
	for _, tc := range tests {
		assert.Equal(tc.want, tc.srv.Slug())
	}
}

