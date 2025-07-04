package mssql

import "strings"

// SplitServerName in COMPUTER\INSTANCE format
func SplitServerName(name string) (host, instance string) {
	if !strings.Contains(name, "\\") {
		return name, ""
	}
	parts := strings.Split(name, "\\")
	return parts[0], parts[1]
}

// // ParseFQDN in COMPUTER[\INSTANCE][,Port] format
// func ParseFQDN(fqdn string) (host, instance string, port uint16, err error) {
// 	type FQDN struct {

// 	}
// }
