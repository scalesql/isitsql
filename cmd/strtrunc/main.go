package main

import (
	"database/sql"
	"fmt"
	"log"
	"unicode/utf8"

	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/microsoft/go-mssqldb/namedpipe"
	_ "github.com/microsoft/go-mssqldb/sharedmemory"
	"golang.org/x/text/unicode/norm"
)

func main() {
	db, err := sql.Open("sqlserver", "sqlserver://D40/SQL2016")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query(stmt1)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			// id      int64
			// t1      string
			// t2      string
			// b       []byte
			// ascii   int32
			// bytelen int32
			// cast    string
			// nv      string
			str string
		)
		err = rows.Scan(&str)
		if err != nil {
			log.Fatal(err)
		}
		//cast = string(b)
		//fmt.Printf("%d: t1: '%s'  t2: '%s'  len: %d  cast: '%s'  nv: %s\n", id, t1, t2, bytelen, cast, nv)
		println(str)
		if !utf8.ValidString(str) {
			println("  INVALID")
		}
		for i := 0; i < 12; i++ {
			fmt.Printf("  TrimSQL: %s\n", TrimSQL2(str, i))
		}
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
}

var stmt1 = `
SELECT
    N'ABC' +
    NCHAR(0xD83D) + NCHAR(0xDE0A) + -- 😊, valid pair
    N'XYZ' +
    NCHAR(0xD800) +                  -- invalid: unpaired high surrogate
    N'123' AS TestValue
UNION ALL
SELECT
    N'ABC' +
    NCHAR(0xD83D) + NCHAR(0xDE0A) + -- 😊, valid pair
    N'XYZ' +
    NCHAR(0xD800)                   -- invalid: unpaired high surrogate
		AS TestValue
UNION ALL
    SELECT CAST( N'ABC' +
    NCHAR(0xD83D) + NCHAR(0xDE0A) + -- 😊, valid pair
    N'XYZ' +
    NCHAR(0xD800) +                  -- invalid: unpaired high surrogate
    N'123' AS NVARCHAR(MAX)) AS TestValue
UNION ALL
SELECT 
    CAST(NCHAR(0xD83D) + NCHAR(0xDE0A) + NCHAR(0xD83D) + NCHAR(0xDE0A) + NCHAR(0xD83D) + NCHAR(0xDE0A) AS NVARCHAR(MAX))

 `

// TrimSQL returns the left most characters of a string
// it should handle unicode/utf8 without splitting characters
// It is designed for big strings of SQL statements
// it uses a sketchy test to see if the string is already short enough
// https://stackoverflow.com/questions/61353016/why-doesnt-golang-have-substring
// This seems better: https://stackoverflow.com/questions/46415894/golang-truncate-strings-with-special-characters-without-corrupting-data
func TrimSQL(str string, n int) string {
	if n == 0 {
		return ""
	}
	if n < 1 { // -1 is the whole string
		return str
	}
	if len(str) <= n { // fewer bytes than we want characters
		return str
	}
	str = norm.NFC.String(str)
	result := str
	chars := 0
	dots := false
	// https://go.dev/doc/effective_go#for
	// for over a string ranges over the runes.
	// i is bumped to the first byte to each rune,
	// then we just count runes/characters
	for i := range str {
		if chars >= n {
			result = str[:i]
			dots = true
			break
		}
		chars++
	}
	if dots {
		result += "..."
	}
	return result
}

// Consider rewriting TrimSQL
// We are still getting � characters here and there
// https://stackoverflow.com/questions/12311033/extracting-substrings-in-go
// The second answer looks promising
func TrimSQL2(input string, n int) string {
	if n == 0 {
		return ""
	}
	if n < 1 { // -1 is the whole string
		return input
	}
	if len(input) <= n { // fewer bytes than we want characters
		return input
	}
	runes := []rune(input)
	if len(runes) <= n {
		return input
	}
	return string(runes[:n])
}
