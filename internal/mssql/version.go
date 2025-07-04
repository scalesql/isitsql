package mssql

import (
	"fmt"
	"strconv"
	"strings"
)

// VersionToString converts a SQL Server version to a sortable string.
// 1.3.4 becomes "000000010000000300000004000000000".
// It expands or contracts to a length of four and zeroes out missing values.
// Each componenet is configured with leading zeroes to a length of eight.
func VersionToString(version string) string {
	raw := strings.Split(version, ".")
	for i := 0; i < 4; i++ {
		raw = append(raw, "0")
	}
	first4 := raw[:4]
	var new string
	for _, s := range first4 {
		v, err := strconv.Atoi(s)
		if err == nil {
			new += fmt.Sprintf("%08d", v)
		} else {
			new += "00000000"
		}
	}
	return new
}
