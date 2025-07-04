package cxnstring

import (
	"fmt"

	"github.com/billgraziano/mssqlodbc"
	"github.com/pkg/errors"
)

// sortedDrivers is the list of drivers in the order we want to use them
var sortedDrivers = []string{
	"ODBC Driver 18 for SQL Server",
	"ODBC Driver 17 for SQL Server",
	"SQL Server Native Client 11.0",
	"SQL Server Native Client 10.0",
	"ODBC Driver 13 for SQL Server",
	"ODBC Driver 11 for SQL Server",
	"SQL Server",
}

// CxnString holds a SQL Server connection string
type CxnString struct {
	driver            string
	Server            string
	UID               string
	PWD               string
	TrustedConnection bool
	App               string
}

// ForServerOnly populates the connection string using driver and server name
func ForServerOnly(s string) (CxnString, error) {
	var c CxnString
	// err := c.SetDriver(d)
	// if err != nil {
	//     return err
	// }
	d, err := GetBestDriver()
	if err != nil {
		return c, err
	}

	c.Server = s
	c.driver = d
	c.TrustedConnection = true
	c.App = "IsItSql"
	return c, nil
}

// GetServerCxnString returns a connection string for a server name
func GetServerCxnString(s string) (string, error) {
	var c CxnString

	d, err := GetBestDriver()
	if err != nil {
		return "", err
	}

	c.Server = s
	c.driver = d
	c.TrustedConnection = true
	c.App = "IsItSql"
	cs, err := c.String()
	if err != nil {
		return "", err
	}
	//fmt.Println(cs)
	return cs, nil
}

// Parse returns a fully formed connection string to pass to the GO odbc driver
func (c *CxnString) Parse(string) error {
	c.driver = "x"

	// add the driver
	// add the app name
	// figure out the trusted connection bit
	return nil
}

func (c *CxnString) String() (string, error) {
	// build the string
	if c.driver == "" {
		d, err := GetBestDriver()
		if err != nil {
			return "", err
		}
		c.driver = d
	}

	if c.Server == "" {
		return "", errors.New("invalid server")
	}

	var s string

	s += fmt.Sprintf("Driver={%s}; ", c.driver)
	s += fmt.Sprintf("Server=%s; ", c.Server)

	// set credentials
	if c.TrustedConnection {
		s += "Trusted_Connection=Yes; "
	} else {
		if c.UID == "" || c.PWD == "" {
			return "", errors.New("no user or password specified")
		}
		s += fmt.Sprintf("UID=%s; PWD=%s; ", c.UID, c.PWD)

	}

	if c.App == "" {
		c.App = "IsItSql"
	}

	s += fmt.Sprintf("App=%s; ", c.App)

	return s, nil
}

// SetDriver accepts an ODBC SQL Server driver string which is validated
func (c *CxnString) SetDriver(driver string) error {
	if !isValidDriver(driver) {
		return errors.New("invalid ODBC driver")
	}

	c.driver = driver
	return nil
}

// GetBestDriver returns the best SQL Server driver
func GetBestDriver() (string, error) {
	driver, err := mssqlodbc.BestDriver()
	if err != nil {
		return "", errors.Wrap(err, "mssqlodbc.bestdriver")
	}
	return driver, nil
}

// private methods below --------------------------------------------------

func isValidDriver(d string) bool {
	for _, str := range sortedDrivers {
		if d == str {
			return true
		}
	}
	return false
}
