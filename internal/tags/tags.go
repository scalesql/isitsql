package tags

import (
	"strings"

	"golang.org/x/exp/slices"
)

// Merge together arrays of strings.  Order is not guaranteed.
// All tags are converted to lower-case.
// All nils returns zero length arrays.
func Merge(src ...*[]string) []string {
	all := make(map[string]string)
	for _, array := range src {
		if array == nil {
			continue
		}
		for _, str := range *array {
			str = strings.ToLower(str)
			all[str] = ""
		}
	}

	merged := make([]string, 0)
	for key := range all {
		merged = append(merged, key)
	}
	slices.Sort(merged)
	return merged
}
