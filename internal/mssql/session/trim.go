package session

import "golang.org/x/text/unicode/norm"

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
	return string(runes[:n]) + "..."
}
