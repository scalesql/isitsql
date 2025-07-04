package app

import (
	"sync"
)

type SlugMap struct {
	sync.RWMutex
	m map[string]string
}

func (sm *SlugMap) Init() {
	sm.Lock()
	defer sm.Unlock()
	sm.m = make(map[string]string)
}

func (sm *SlugMap) Set(slug, key string) {
	sm.Lock()
	defer sm.Unlock()
	sm.m[slug] = key
}

// Delete
func (sm *SlugMap) Delete(slug string) {
	sm.Lock()
	defer sm.Unlock()
	delete(sm.m, slug)
}

// Get one
func (sm *SlugMap) Get(slug string) (string, bool) {
	sm.RLock()
	defer sm.RUnlock()
	val, found := sm.m[slug]
	return val, found
}

// Get all
// Get dupes
