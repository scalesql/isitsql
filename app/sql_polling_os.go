package app

import (
	"database/sql"
	"regexp"
	"strings"
)

var atatVersionRegex = regexp.MustCompile(`(?m) on\s(?P<os>.*)\s<(?P<arch>.*)>`)

func (wrap *SqlServerWrapper) PollContainer() error {
	if wrap.MajorVersion <= 13 { // SQL Server 2016 and earlier
		return nil
	}

	query := `
	SELECT
		 host_platform = (SELECT host_platform FROM sys.dm_os_host_info) 			-- 2017
		,host_distribution = (SELECT host_distribution FROM sys.dm_os_host_info) 	-- 2017
	`
	if wrap.MajorVersion >= 15 { // SQL Server 2019 and later
		query += `
		,container_type = (SELECT container_type FROM sys.dm_os_sys_info) 			-- 2019
		,container_type_desc = (SELECT container_type_desc FROM sys.dm_os_sys_info)	-- 2019
	`
	} else {
		query += `
		,container_type = 0 		
		,container_type_desc = ''
	`
	}

	var platform, distr, cdesc string
	var ctype int32
	err := wrap.DB.QueryRow(query).Scan(&platform, &distr, &ctype, &cdesc)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	wrap.Lock()
	wrap.HostPlatform = platform
	wrap.HostDistribution = distr
	wrap.ContainerType = ctype
	wrap.ContainerDescription = cdesc
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
