package ad

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseName(t *testing.T) {
	assert := assert.New(t)
	u, d, err := ParseName("bill@mydomain.com")
	assert.NoError(err)
	assert.Equal("bill", u)
	assert.Equal("mydomain.com", d)
}

func TestParseCN(t *testing.T) {
	assert := assert.New(t)
	type test struct {
		got  string
		want string
	}
	tests := []test{
		{"CN=eus-sql-dpa-sysadmin,OU=org-sqlgroups,OU=org,DC=core,DC=org,DC=us,DC=loc", "eus-sql-dpa-sysadmin"},
		{"CN=sql backup shares,OU=Security Groups,OU=Groups,OU=Org,DC=domain,DC=com", "sql backup shares"},
		{"CN=sql-sysadmins,CN=Users,DC=demo,DC=loc", "sql-sysadmins"},
	}
	for _, tc := range tests {

		got := parseCN(tc.got)
		println(got)
		assert.Equal(tc.want, got)
	}
}
