package app

// sort a map's keys in descending order of its values.

//import "sort"

type SortedMapInt64 struct {
	BaseMap    map[string]int64
	SortedKeys []string
}

func (sm *SortedMapInt64) Len() int {
	return len(sm.BaseMap)
}

// func (sm *sortedMap) Less(i, j int) bool {
// 	return sm.m[sm.s[i]] > sm.m[sm.s[j]]
// }

func (sm *SortedMapInt64) Less(i, j int) bool {
	a, b := sm.BaseMap[sm.SortedKeys[i]], sm.BaseMap[sm.SortedKeys[j]]
	if a != b {
		// Order by decreasing value.
		return a > b
	}
	// Otherwise, alphabetical order.
	return sm.SortedKeys[j] > sm.SortedKeys[i]

}

func (sm *SortedMapInt64) Swap(i, j int) {
	sm.SortedKeys[i], sm.SortedKeys[j] = sm.SortedKeys[j], sm.SortedKeys[i]
}

// func sortedKeys(m map[string]int64) []string {
// 	sm := new(sortedMap)
// 	sm.m = m
// 	sm.s = make([]string, len(m))
// 	i := 0
// 	for key, _ := range m {
// 		sm.s[i] = key
// 		i++
// 	}
// 	sort.Sort(sm)
// 	return sm.s
// }

type sortedMapString struct {
	BaseMap    map[string]string
	SortedKeys []string
}

func (sm *sortedMapString) Len() int {
	return len(sm.BaseMap)
}

// func (sm *sortedMap) Less(i, j int) bool {
// 	return sm.m[sm.s[i]] > sm.m[sm.s[j]]
// }

func (sm *sortedMapString) Less(i, j int) bool {
	a, b := sm.BaseMap[sm.SortedKeys[i]], sm.BaseMap[sm.SortedKeys[j]]
	if a != b {
		// Order by decreasing value.
		return a < b
	}
	// Otherwise, alphabetical order.
	return sm.SortedKeys[j] > sm.SortedKeys[i]

}

func (sm *sortedMapString) Swap(i, j int) {
	sm.SortedKeys[i], sm.SortedKeys[j] = sm.SortedKeys[j], sm.SortedKeys[i]
}
