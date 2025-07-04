package app

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/scalesql/isitsql/internal/dwaits"
	"github.com/scalesql/isitsql/internal/failure"
	"github.com/scalesql/isitsql/settings"
	"github.com/billgraziano/mssqlodbc"
	"github.com/kardianos/osext"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"

	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/microsoft/go-mssqldb/namedpipe"
	_ "github.com/microsoft/go-mssqldb/sharedmemory"
)

func (list *ServerList) importCSV() error {
	dir, err := osext.ExecutableFolder()
	if err != nil {
		log.Fatal(err)
	}

	fullfile := filepath.Join(dir, "servers.txt")

	// Check if the file exists
	if _, err := os.Stat(fullfile); os.IsNotExist(err) {
		return nil
	}

	WinLogln("Importing servers.txt...")
	/* #nosec G304 */
	csvfile, err := os.Open(fullfile)
	if err != nil {
		WinLogln("Error reading servers.txt: " + err.Error())
		return err
	}

	defer func() {
		if err := csvfile.Close(); err != nil {
			WinLogln(errors.Wrap(err, "csvfile.close"))
		}
	}()

	reader := csv.NewReader(csvfile)
	reader.Comma = ','
	reader.Comment = '#'
	reader.TrimLeadingSpace = true

	// Set to -1 to allow a variable number of fields
	reader.FieldsPerRecord = -1

	var target, friendlyName string
	var tags []string

	//var userKeys []string

	for {
		tags = make([]string, 0)
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			WinLogln("*********************************************************")
			WinLogln("Bad row in servers.txt:", err)
			WinLogln("*********************************************************")
		} else {

			friendlyName = ""
			target = record[0]
			if len(record) > 1 {
				friendlyName = record[1]
			}

			//userKeys = append(userKeys, target)

			// see if we have tags to process
			if len(record) > 2 {
				tags, _ = parseTags(record[2])
			}

			var c settings.SQLServer

			p := strings.Index(target, "=")

			// if there's no "=" then it's not a connection string
			if p == -1 {
				c = settings.SQLServer{
					FQDN:              target,
					FriendlyName:      friendlyName,
					TrustedConnection: true,
					Tags:              tags,
				}

				WinLogln("Importing a trusted connection server FQDN: ", c.FQDN)
				_, err = settings.AddSQLServer(c)
				if err != nil {
					return errors.Wrap(err, "addsqlserver")
				}

			} else { // Figure out a connection string

				// Try to parse the connection string
				x, err := mssqlodbc.Parse(target)
				if err != nil {
					str := fmt.Sprintf("Error parsing connection string: %s. Imported as custom connection string.", target)
					WinLogln(str)

					// Force empty values so we import a connection string
					x.Server = ""
					x.User = ""
					x.Password = ""
				}

				// If I get  server, login and password, convert it to a SQL Login
				if x.Server != "" && x.User != "" && x.Password != "" {
					c = settings.SQLServer{
						FriendlyName: friendlyName,
						Tags:         tags,
						FQDN:         x.Server,
						Login:        x.User,
						Password:     x.Password,
					}

					WinLogln("Importing a parsed conenction string: ", c.FQDN, c.FriendlyName, c.Login)
					_, err = settings.AddSQLServer(c)
					if err != nil {
						return errors.Wrap(err, "addsqlserver")
					}

				} else {
					// First just add as a custom connection string
					c = settings.SQLServer{
						FriendlyName:           friendlyName,
						Tags:                   tags,
						CustomConnectionString: target,
					}

					WinLogln("Importing a custom connection string: ", c.FriendlyName)
					_, err = settings.AddSQLServer(c)
					if err != nil {
						return errors.Wrap(err, "addsqlserver")
					}

				}

			}
		}
	}

	// Sort the server list
	list.SortKeys()

	// Map the tags
	list.mapTags()

	//TODO Rename the servers.txt to servers_imported_20170430_HHMMSS.txt
	//newFileName = "servers_imported_X.txt"

	// Close the CSV file.  The other close will generate an error but oh well
	err = csvfile.Close()
	if err != nil {
		fmt.Println("close:", err)
	}

	now := time.Now().Format("20060102_150405")
	newName := fmt.Sprintf("servers_imported_%s.txt", now)
	newFile := filepath.Join(dir, newName)
	err = os.Rename(fullfile, newFile)
	if err != nil {
		return errors.Wrap(err, "rename")
	}

	msg := fmt.Sprintf("servers.txt imported and renamed to %s.  PLEASE DELETE THIS IF IT HAS PASSWORDS.", newFile)
	WinLogln(msg)

	return nil
}

func (list *ServerList) Keys() []string {
	list.RLock()
	defer list.RUnlock()
	a := make([]string, len(list.Servers))
	var i int
	for k := range list.Servers {
		a[i] = k
		i++
	}
	return a
}

// Pointers gets an array of pointes to servers
// func (list *ServerList) Pointers() []*SqlServer {
// 	list.RLock()
// 	defer list.RUnlock()
// 	a := make(SqlServerArray, len(list.Servers))
// 	var i int
// 	for _, v := range list.Servers {
// 		a[i] = v
// 		i++
// 	}
// 	return a
// }

// SortKeys returns a list of sorted keys
func (list *ServerList) SortKeys() {

	sm := new(sortedMapString)
	bm := make(map[string]string)

	var skey string

	localPointers := list.Pointers()

	// list.Lock()
	// defer list.Unlock()
	for _, v := range localPointers {

		v.RLock()
		sortPriority := strconv.Itoa(v.SortPriority)
		displayName := v.DisplayName()
		k := v.MapKey
		v.RUnlock()

		skey = "000000" + sortPriority
		skey = string(skey[len(skey)-6:])

		bm[k] = skey + strings.ToLower(displayName)
	}

	sm.BaseMap = bm
	list.RLock()
	sm.SortedKeys = make([]string, len(list.Servers))
	list.RUnlock()

	i := 0
	for key := range sm.BaseMap {
		sm.SortedKeys[i] = key
		i++
	}

	sort.Sort(sm)

	list.Lock()
	list.SortedKeys = sm.SortedKeys
	list.Unlock()

}

func (s *SqlServerWrapper) resetDB() error {
	var err error

	s.RLock()
	connType := s.ConnectionType
	connString := s.ConnectionString
	s.RUnlock()

	db, err := sql.Open(connType, connString)
	if err != nil {
		WinLogln("Error opening database connection: ", err)
		return errors.Wrap(err, "Open")
	}

	// Firewalls are blocking connections older than 60 minutes
	// This limits us to 3 logins per hour which seems reasonable
	db.SetConnMaxLifetime(20 * time.Minute)

	s.Lock()
	oldDB := s.DB
	s.DB = db
	s.Unlock()

	err = oldDB.Close()
	if err != nil {
		WinLogln("Error closing database connection: ", err)
		return errors.Wrap(err, "Close")
	}

	return nil
}

// Delete removes a server from polling
func (list *ServerList) Delete(key string) error {

	list.RLock()
	s, ok := list.Servers[key]
	list.RUnlock()
	if !ok {
		return fmt.Errorf("key not found: %s", key)
	}

	list.Lock()
	delete(list.Servers, key)
	list.Unlock()

	list.SortKeys()
	list.mapTags()
	s.stop <- struct{}{}
	s.WaitBox.Stop()

	WinLogln(fmt.Sprintf("Deleting: %s (%s)", s.DisplayName(), key))

	return nil
}

// UpdateFromSettings updates an entry
func (list *ServerList) UpdateFromSettings(key string, c settings.SQLServer) error {
	dirty := false
	list.RLock()
	s, ok := list.Servers[key]
	list.RUnlock()
	if !ok {
		return fmt.Errorf("key not found: %s", key)
	}

	s.Lock()
	if s.FriendlyName != c.FriendlyName {
		dirty = true
		s.FriendlyName = c.FriendlyName
	}
	if !slices.Equal(s.Tags, c.Tags) {
		dirty = true
		s.Tags = c.Tags
	}
	if s.IgnoreBackups != c.IgnoreBackups {
		dirty = true
		s.IgnoreBackups = c.IgnoreBackups
	}
	if !slices.Equal(s.IgnoreBackupsList, c.IgnoreBackupsList) {

		dirty = true
		s.IgnoreBackupsList = c.IgnoreBackupsList
	}
	if s.FQDN != c.FQDN {
		dirty = true
		s.FQDN = c.FQDN
	}
	s.Unlock()

	if dirty {
		s.SetConectionString(key, c)
		list.SortKeys()
		list.mapTags()

		// Launch a poll in the background
		go func(keyToPoll string) {
			defer failure.HandlePanic()
			globalPool.Poll(keyToPoll)
		}(key)

		WinLogln(fmt.Sprintf("Updating: %s (%s)", s.DisplayName(), key))
	}
	return nil
}

// SetConectionString sets the connection string from a settings.SQLServer
func (s *SqlServerWrapper) SetConectionString(key string, c settings.SQLServer) error {

	cs, err := c.ConnectionString()
	if err != nil {
		return errors.Wrap(err, "connectionString")
	}

	s.Lock()
	s.ConnectionString = cs
	s.ConnectionType = "sqlserver"
	// s.ConnectionType = "mssql"
	s.CredentialKey = c.CredentialKey
	s.Unlock()

	return nil
}

// AddFromSettings adds based on a settings entry
func (list *ServerList) AddFromSettings(key string, c settings.SQLServer, pollNow bool) error {

	var err error
	var s SqlServerWrapper

	// Let's see if we have anything cached
	// All errors are logged in this function
	s.SqlServer = GetCachedServer(key)

	s.FriendlyName = c.FriendlyName
	s.FQDN = c.FQDN
	s.BackupMessage = "Backups haven't polled yet"
	s.Tags = c.Tags
	s.MapKey = key
	s.IgnoreBackups = c.IgnoreBackups
	s.IgnoreBackupsList = c.IgnoreBackupsList

	// Set the connection string
	err = s.SetConectionString(key, c)
	if err != nil {
		return errors.Wrap(err, "setconnectionstring")
	}

	if s.Metrics == nil {
		s.Metrics = make(map[string]Metric)
	}
	s.LastPollError = ""

	//fmt.Println("s.Metrics: ", s.Metrics)

	// TODO This needs to happen at the first poll
	// Maybe just set s.DB = nil here and fix it on polling

	// Otherwise we can hang here
	db, err := sql.Open(s.ConnectionType, s.ConnectionString)

	if err != nil {
		WinLogln("Error opening database connection: ", err)
	}
	// Firewalls are blocking connections older than 60 minutes
	// This limits us to 3 logins per hour which seems reasonable
	db.SetConnMaxLifetime(20 * time.Minute)
	s.DB = db
	s.stop = make(chan struct{}, 1)

	list.Lock()
	list.Servers[key] = &s
	list.Unlock()
	list.SortKeys()
	list.mapTags()

	// Launch a poll in the background
	// if pollNow {
	// 	go func(keyToPoll string) {
	// 		globalPool.Poll(keyToPoll)
	// 	}(key)
	// }
	go PollRoutine(&s)

	//cfg := getGlobalConfig()
	//if cfg.DynamicWaits {
	if s.WaitBox == nil {
		s.WaitBox = &dwaits.Box{}
	}
	err = s.WaitBox.Start(context.Background(), DynamicWaitRepository, key, s.ConnectionType, s.ConnectionString)
	if err != nil {
		errmsg := fmt.Sprintf("%s: %s", key, errors.Wrap(err, "waitbox.start"))
		logrus.Error(errmsg)
		WinLogln(errmsg)
	}
	//}

	WinLogln(fmt.Sprintf("Adding: %s (%s)", s.DisplayName(), key))
	logrus.Tracef("CXN: %s => '%s' (%s)", s.DisplayName(), s.ConnectionString, s.ConnectionType)

	return nil
}

// ShortDuration returns a duration as a short string
func (d *Database) ShortDuration(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return durationToShortString(t, time.Now())
}
