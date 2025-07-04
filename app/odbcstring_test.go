package app

import (
	"testing"
)

func TestGetServerName(t *testing.T) {

	// s, err := GetServer("Driver={SQL Server};Server=127.0.0.1,59625;Database=tempdb;uid=test;pwd=test;App=IsItSql;")
	// fmt.Println(s, err)

	var cxStrings = []struct {
		s        string
		expected string
	}{
		{"Driver={SQL Server};Server=127.0.0.1,59625;Database=tempdb;uid=test;pwd=test;App=IsItSql;", "127.0.0.1,59625"},
		{"Driver={SQL Server};server=127.0.0.1,59625;Database=tempdb;uid=test;pwd=test;App=IsItSql;", "127.0.0.1,59625"},
		{"Driver={SQL Server} ; Server = 127.0.0.1,59625  ;  Database=tempdb;uid=test;pwd=test;App=IsItSql;", "127.0.0.1,59625"},
		{"Driver={SQL Server} ; Server = D30\\Bonk  ;  Database=tempdb;uid=test;pwd=test;App=IsItSql;", "D30\\Bonk"},
	}

	for _, cx := range cxStrings {
		actual, _ := getCXServerName(cx.s)
		if actual != cx.expected {
			t.Errorf("Server Name(%s): expected %s, actual %s", cx.s, cx.expected, actual)
		}
	}

	// Check if it can't find the server attribute
	s, err := getCXServerName("Driver={SQL Server};myserver=127.0.0.1,59625;Database=tempdb;uid=test;pwd=test;App=IsItSql;")
	if err == nil {
		t.Error("Expected an error")
	}
	if s != "" {
		t.Error("Expected s to be empty")
	}
}
