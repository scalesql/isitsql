package app

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// ServerList holds the lists of servers we are monitoring
type ServerList struct {
	sync.RWMutex
	Servers    map[string]*SqlServerWrapper
	SortedKeys []string
}

// SqlServerArray is an array of SQL Server pointers
type SqlServerArray []SqlServer

// GetLocalPointers returns an array of pointers to the monitored SQL Server objects
func (l *ServerList) Pointers() []*SqlServerWrapper {

	var list []*SqlServerWrapper

	l.RLock()
	defer l.RUnlock()

	for _, v := range l.Servers {
		list = append(list, v)
	}

	return list
}

// CloneAll returns a copy of the servers
func (sl *ServerList) CloneAll() SqlServerArray {
	keys := sl.Keys()
	ss := make([]SqlServer, 0, len(keys))
	for _, k := range keys {
		srv, ok := sl.CloneOne(k)
		if ok {
			ss = append(ss, srv)
		}
	}
	return ss
}

// CloneUnique returns an array of unique servers based on domain, computer, instance
func (sl *ServerList) CloneUnique() SqlServerArray {
	type primaryKey struct {
		domain string
		server string
	}
	uniques := make(map[primaryKey]bool)
	keys := sl.Keys()
	ss := make([]SqlServer, 0, len(keys))
	for _, k := range keys {
		srv, ok := sl.CloneOne(k)
		if !ok {
			continue
		}
		// if it doesn't exist, add it
		pk := primaryKey{domain: srv.Domain, server: srv.ServerName}
		_, ok = uniques[pk]
		if ok {
			continue
		}
		uniques[pk] = true
		ss = append(ss, srv)
	}
	return ss
}

// CloneOne clones one server
func (sl *ServerList) CloneOne(key string) (SqlServer, bool) {
	var s SqlServer
	sl.RLock()
	wr, ok := sl.Servers[key]
	sl.RUnlock()
	if !ok {
		return s, false
	}
	wr.RLock()
	s = wr.SqlServer
	wr.RUnlock()
	return s, true
}

// GetDB gets the database connection
func (sl *ServerList) GetDB(key string) (*sql.DB, bool) {
	sl.RLock()
	defer sl.RUnlock()
	wr, ok := sl.Servers[key]
	return wr.DB, ok
}

// NewPool returns a *sql.DB.  The main SQL connection is used by polling
// and has short lifetime.  This should last longer. Or it can be closed.
func (list *ServerList) NewPool(key string) (*sql.DB, error) {
	list.RLock()
	wr, ok := list.Servers[key]
	list.RUnlock()
	if !ok {
		return nil, fmt.Errorf("missing key: %s", key)
	}
	wr.RLock()
	connType := wr.ConnectionType
	connString := wr.ConnectionString
	wr.RUnlock()

	pool, err := sql.Open(connType, connString)
	if err != nil {
		return nil, err
	}
	pool.SetConnMaxLifetime(20 * time.Minute)
	return pool, nil
}

// Exists checks if a server exists with the key
func (sl *ServerList) Exists(key string) bool {
	sl.RLock()
	defer sl.RUnlock()
	_, ok := sl.Servers[key]
	return ok
}

// GetWrapper returns the SqlServerWrapper for a key
func (sl *ServerList) GetWrapper(key string) (*SqlServerWrapper, bool) {
	sl.RLock()
	defer sl.RUnlock()
	wr, ok := sl.Servers[key]
	return wr, ok
}
