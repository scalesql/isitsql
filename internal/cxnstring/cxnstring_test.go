package cxnstring

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// func TestInvalidDriver(t *testing.T) {
//     var c CxnString
//     err := c.SetDriver("test")
//     if err == nil {
//         t.Error("test driver should fail ")
//     }

//     c.Server = `D30\SQL2012`

//     _, err = c.String()
//     if err == nil {
//         t.Error("test driver should fail ")
//     }

// }

func TestValidDriverStrings(t *testing.T) {
	assert := assert.New(t)
	isValid := isValidDriver("junk")
	assert.False(isValid)
	isValid = isValidDriver("ODBC Driver 11 for SQL Server")
	assert.True(isValid)
}

func TestForServerOnlyl(t *testing.T) {

	c, err := ForServerOnly("TEST")
	if err != nil {
		t.Error("For Server Only: ", err)
	}
	cx, err := c.String()
	if err != nil {
		t.Error("For Server Only.String(): ", err)
	}
	fmt.Println("For Server Only: ", cx)
}

func TestValidDrivers(t *testing.T) {
	assert := assert.New(t)
	var c CxnString
	//err := c.SetDriver("SQL Server Native Client 11.0")
	c.Server = `D30\SQL2012`
	c.TrustedConnection = true
	// if err != nil {
	//     t.Error("Error setting driver: ", err)
	// }

	s, err := c.String()
	assert.NoError(err)
	assert.Equal("Driver={ODBC Driver 18 for SQL Server}; Server=D30\\SQL2012; Trusted_Connection=Yes; App=IsItSql; ", s)

	c.SetDriver("ODBC Driver 11 for SQL Server")
	s, err = c.String()
	if s != "Driver={ODBC Driver 11 for SQL Server}; Server=D30\\SQL2012; Trusted_Connection=Yes; App=IsItSql; " {
		t.Errorf("Invalid connection string: [%s]", s)
	}
	assert.NoError(err)
}
