package app

import (
	"database/sql"
	"regexp"
	"strings"
)

var atatVersionRegex = regexp.MustCompile(`(?m) on\s(?P<os>.*)\s<(?P<arch>.*)>`)

func (wrap *SqlServerWrapper) PollContainer() error {
	// These fields only exist in SQL Server 2019 and higher
	if wrap.MajorVersion < 15 {
		return nil
	}
	var containerType int
	err := wrap.DB.QueryRow("select container_type from sys.dm_os_sys_info").Scan(&containerType)
	// if err == sql.ErrNoRows, we will parse an empty string and get "unknown"
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
	}
	wrap.Lock()
	if containerType != 0 {
		wrap.InContainer = true
	} else {
		wrap.InContainer = false
	}
	wrap.Unlock()
	return nil
}

// PollOS reads @@VERSION for the operating system information
func (wrap *SqlServerWrapper) PollOS() error {
	var rawVersion string
	err := wrap.DB.QueryRow("SELECT @@VERSION").Scan(&rawVersion)
	// if err == sql.ErrNoRows, we will parse an empty string and get "unknown"
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
	}
	version := parseatatversion(rawVersion)
	wrap.Lock()
	wrap.OSName = version.os
	wrap.OSArch = version.arch
	wrap.Unlock()
	return nil
}

type atatversion struct {
	os   string
	arch string
}

// parseatatversion takes the result of @@VERSION and returns
// the parsed values
func parseatatversion(s string) atatversion {
	v := atatversion{os: "unknown", arch: "unknown"}
	match := atatVersionRegex.FindStringSubmatch(s)
	results := make(map[string]string)
	for i, name := range match {
		results[atatVersionRegex.SubexpNames()[i]] = name
	}
	os, exists := results["os"]
	if exists {
		v.os = os
	}
	arch, exists := results["arch"]
	if exists {
		v.arch = strings.ToLower(arch)
	}
	return v
}
