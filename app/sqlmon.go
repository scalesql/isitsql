package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/radovskyb/watcher"
	uuid "github.com/satori/go.uuid"
	"github.com/scalesql/isitsql/internal/bucket"
	"github.com/scalesql/isitsql/internal/c2"
	"github.com/scalesql/isitsql/internal/cxnstring"
	"github.com/scalesql/isitsql/internal/dwaits"
	"github.com/scalesql/isitsql/internal/failure"
	"github.com/scalesql/isitsql/internal/fileio"
	"github.com/scalesql/isitsql/internal/hadr"
	"github.com/scalesql/isitsql/internal/mrepo"
	"github.com/scalesql/isitsql/internal/settings"
	"github.com/scalesql/isitsql/internal/waitmap"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

// setup runs synchronously and must succeed before continuing
// We don't currently need anything in here
func setup() error {
	WinLogln("Launching Setup...")

	if DEV {
		WinLogln("DEV build!")
	}
	return nil
}

// SetupWrapper is called by the service. It does setup,
// loads AG names, connections, and cached stats
func SetupWrapper() error {
	err := setupAsync()
	if err != nil {
		WinLogln(errors.Wrap(err, "setupasync"))
		logrus.Error(err.Error())
		elog.Error(1, err.Error())
	}

	if AppConfigMode == ModeGUI {
		logrus.Trace("setupwrapper: reading JSON...")
		err = SetupHadrNames()
		if err != nil {
			WinLogln(errors.Wrap(err, "setuphadrnames"))
			logrus.Error(err.Error())
			elog.Error(1, err.Error())
		}

		err = setupConnectionsAsync()
		if err != nil {
			WinLogln(errors.Wrap(err, "setupconnectionsasync"))
			logrus.Error(err.Error())
			elog.Error(1, err.Error())
		}
	} else {
		processHCLFiles()
		watchHCLFiles()
	}

	return nil
}

func processHCLFiles() {
	path, err := c2.Path()
	if err != nil {
		WinLogln(errors.Wrap(err, "c2.path"))
		logrus.Error(err.Error())
	}
	logrus.Tracef("servers folder: '%s'", path)
	logrus.Trace("setupwrapper: reading HCL...")
	c2map, msgs, err := c2.GetHCLFiles()
	if err != nil {
		WinLogln(errors.Wrap(err, "c2.getconfig"))
		logrus.Error(err.Error())
	}
	for _, msg := range msgs {
		WinLogln(msg)
		logrus.Error(msg)
	}
	if len(msgs) > 0 || err != nil {
		return
	}
	err = PushC2Servers(c2map)
	if err != nil {
		WinLogln(err.Error())
		logrus.Error(err.Error())
	}
}

func watchHCLFiles() {
	path, err := c2.Path()
	if err != nil {
		WinLogln(errors.Wrap(err, "c2.path"))
		logrus.Error(errors.Wrap(err, "c2.path"))
		return
	}
	w := watcher.New()

	// If SetMaxEvents is not set, the default is to send all events.
	w.SetMaxEvents(1)

	r := regexp.MustCompile(`^.*\.hcl$`)
	w.AddFilterHook(watcher.RegexFilterHook(r, false))

	go func() {
		defer failure.HandlePanic()
		ticker := time.NewTicker(1 * time.Hour)
		for {
			select {
			case <-ticker.C:
				processHCLFiles()
			case event := <-w.Event:
				logrus.Trace(event)
				processHCLFiles()
			case err := <-w.Error:
				WinLogln(err)
				logrus.Error(err)
			case <-w.Closed:
				return
			}
		}
	}()

	if err := w.AddRecursive(path); err != nil {
		WinLogln(errors.Wrap(err, "w.addrecursive"))
		logrus.Error(errors.Wrap(err, "w.addrecursive"))
		return
	}
	go func() {
		defer failure.HandlePanic()
		freq := 1 * time.Minute
		if DEV {
			freq = 1 * time.Second
		}
		if err := w.Start(freq); err != nil {
			WinLogln(errors.Wrap(err, "w.start"))
			logrus.Error(errors.Wrap(err, "w.start"))
			return
		}
	}()
	WinLogf("watching HCL files: '%s'", path)
	logrus.Infof("add watch path: '%s'", path)
}

func PushC2Servers(c2map c2.ConfigMaps) error {
	// Set the HADR names
	agNames := make([][]string, 0)
	for k, v := range c2map.AGs {
		agn := []string{k.Domain, k.Name, v}
		agNames = append(agNames, agn)
	}
	n, dirty, err := hadr.PublicAGMap.SetAGNames(agNames)
	if err != nil {
		WinLogln(errors.Wrap(err, "agmap.setagnames"))
	} else {
		if dirty {
			WinLogln(fmt.Sprintf("PushC2Servers: Availability Group Names Set: %d", n))
		}
	}

	// Go through existing and delete
	existingKeys := servers.Keys()
	for _, k := range existingKeys {
		_, ok := c2map.Connections[k]
		if !ok {
			servers.Delete(k)
		}
	}

	// Get a map of credentials to use
	creds, err := settings.ListSQLCredentials()
	if err != nil {
		WinLogln(err)
	}
	credentialMap := make(map[string]string)
	for _, cred := range creds {
		credname := strings.ToLower(cred.Name)
		_, ok := credentialMap[credname]
		if ok {
			WinLogln(fmt.Sprintf("duplicate credential name: %s", credname))
			continue
		}
		credentialMap[credname] = cred.CredentialKey.String()
	}

	// go through C2 files
	for k, v := range c2map.Connections {
		//fmt.Printf("k: '%s'  v: '%+v' server: %s\n", k, v, *v.Server)
		server := settings.SQLServer{
			//ServerKey: uuid.FromStringOrNil(k),
			ServerKey: k,
		}
		server.FQDN = v.Server
		server.FriendlyName = v.DisplayName
		if v.CredentialName != "" {
			// Lookup the credential
			credname := strings.ToLower(v.CredentialName)
			credkey, ok := credentialMap[credname]
			if ok {
				server.CredentialKey = credkey
			} else {
				WinLogf("missing credential: %s", v.CredentialName)
			}
		} else {
			server.TrustedConnection = true
		}
		server.Tags = v.Tags
		server.IgnoreBackups = v.IgnoreBackups
		server.IgnoreBackupsList = v.IgnoreBackupsList
		srv, exists := servers.CloneOne(k)

		if exists {
			if srv.FQDN != server.FQDN ||
				srv.FriendlyName != server.FriendlyName ||
				srv.CredentialKey != server.CredentialKey ||
				!slices.Equal(srv.Tags, server.Tags) ||
				srv.IgnoreBackups != server.IgnoreBackups ||
				!slices.Equal(srv.IgnoreBackupsList, server.IgnoreBackupsList) {
				err = servers.UpdateFromSettings(k, server)
				if err != nil {
					WinLogln(err)
				}
			}
		} else {
			err = servers.AddFromSettings(k, server, false)
			if err != nil {
				WinLogln(err)
			}
		}
	}
	return nil
}

// setupAsync is inside the async launcher.  It completes before
// polling or the web service is launched.
func setupAsync() error {
	var err error

	globalStats.Lock()
	globalStats.StartTime = time.Now()
	globalStats.SessionGUID = uuid.NewV4().String()
	globalStats.Unlock()

	// Setup Log Dir
	err = settings.MakeDir("log")
	if err != nil {
		return errors.Wrap(err, "settings.makedir")
	}

	// Create the config directory if it doesn't exist
	err = settings.SetupConfigDir()
	if err != nil {
		return errors.Wrap(err, "settings.setupconfigdir")
	}

	// Read the settings file.  We don't need it now but can't continue without it
	s, err := settings.ReadConfig()
	if err != nil {
		return errors.Wrap(err, "readconfig")
	}

	if s.Debug {
		if logrus.DebugLevel > logrus.GetLevel() {
			logrus.SetLevel(logrus.DebugLevel)
			logrus.Debug("settings: debug enabled")
		}
	}
	if s.Trace {
		if logrus.TraceLevel > logrus.GetLevel() {
			logrus.SetLevel(logrus.TraceLevel)
			logrus.Trace("settings: trace enabled")
		}
	}

	// set some global settings
	globalConfig.Lock()
	globalConfig.AppConfig.EnableProfiler = s.EnableProfiler
	globalConfig.AppConfig.EnableStatsviz = s.EnableStatsviz
	globalConfig.AppConfig.FullBackupAlertHours = s.BackupAlertHours
	globalConfig.AppConfig.LogBackupAlertMinutes = s.LogBackupAlertMinutes
	globalConfig.AppConfig.HomePageURL = s.HomePageURL
	globalConfig.AppConfig.AGAlertMB = s.AGAlertMB
	globalConfig.AppConfig.AGWarnMB = s.AGWarnMB
	globalConfig.AppConfig.Debug = s.Debug
	globalConfig.AppConfig.Trace = s.Trace

	err = settings.MakeDir("cache")
	if err != nil {
		return errors.Wrap(err, "settings.makedir")
	}

	globalConfig.Unlock()

	globalStats.Lock()
	globalStats.ClientGUID = s.ClientGUID
	globalStats.Unlock()

	// We only save so that we'll get any defaults if the file doesn't exist
	err = s.Save()
	if err != nil {
		return errors.Wrap(err, "saveconfig")
	}

	// setup the metric repository database
	if s.Repository.Host != "" && s.Repository.Database != "" {
		repo, err := mrepo.NewRepository(s.Repository.Host, s.Repository.Database, logrus.WithContext(context.Background()), &GLOBAL_RINGLOG)
		GlobalRepository = repo
		if err != nil {
			// WinLogln(errors.Wrap(err, "mrepo.setup"))
			WinLogErr(errors.Wrap(err, "REPOSITORY"))
		} else {

			WinLogf("REPOSITORY: host='%s' database='%s'", s.Repository.Host, s.Repository.Database)
		}
	}

	if getGlobalConfig().EnableProfiler {
		WinLogln("pprof enabled on http://localhost:6060/debug/pprof/ ")
		go func() {
			defer failure.HandlePanic()
			log.Println("pprof viewer: ", http.ListenAndServe("localhost:6060", nil))
		}()
	}

	// wd, err := osext.ExecutableFolder()
	// if err != nil {
	// 	return errors.Wrap(err, "osext.executablefolder")
	// }

	// clean up old log entries
	err = bucket.PurgeFiles("log", "isitsql", "log", 24*90*time.Hour)
	if err != nil {
		return errors.Wrap(err, "bucket.purgefiles")
	}

	DynamicWaitRepository, err = dwaits.NewRepository(context.Background())
	if err != nil {
		logrus.Error(errors.Wrap(err, "dwaits.newrepository"))
	}
	err = DynamicWaitRepository.ReadHistory()
	if err != nil {
		logrus.Error(errors.Wrap(err, "w2.readhistory"))
	}

	//dir := filepath.Join(wd, "cache")
	// err = globalWaitsBucket.Start(dir, "waits")
	// if err != nil {
	// 	return errors.Wrap(err, "globalwaitsbucket.start")
	// }

	// Configure the polling pool (I'm not sure this is really used)
	// I think everything polls in it's own GO routine
	globalPool = NewPool(s.PollWorkers)

	setUseLocalStatic()

	servers.Servers = make(map[string]*SqlServerWrapper)
	hadr.PublicAGMap, err = hadr.NewAGMap()
	if err != nil {
		WinLogln(errors.Wrap(err, "hadr.newagmap"))
	}

	waitmap.Mapping.Mappings = make(map[string]waitmap.WaitMap)
	waitmap.Mapping.SetBaseWaitMapping()
	WinLogln("Wait Types after loading base types: ", len(waitmap.Mapping.Mappings))

	err = waitmap.Mapping.ReadWaitMapping("waits.txt")
	if err != nil {
		WinLogln("Invalid Wait Mappings File: config/waits.txt: ", err)
	}
	WinLogln("Wait Types after waits.txt: ", len(waitmap.Mapping.Mappings))

	// Report the best driver
	d, err := cxnstring.GetBestDriver()
	if err != nil {
		return err
	}
	logrus.Debugf("Suggested Driver: %s", d)

	return nil
}

func SetupHadrNames() error {
	// Check for ag_names.csv
	err := HadrCheckNamesFile()
	if err != nil {
		WinLogln(errors.Wrap(err, "hadrchecknamesfiles"))
	}

	// Read ag_names.csv
	agNames, err := fileio.ReadConfigCSV("ag_names.csv")
	if err != nil {
		WinLogln(errors.Wrap(err, "fileio.readconfigcsv"))
	}
	n, dirty, err := hadr.PublicAGMap.SetAGNames(agNames)
	if err != nil {
		WinLogln(errors.Wrap(err, "agmap.setagnames"))
	} else {
		if dirty {
			WinLogln(fmt.Sprintf("SetupHADRName: Availability Group Names Set: %d", n))
		}
	}
	return nil
}

func setupConnectionsAsync() error {
	var err error
	stgSQL, err := settings.ReadConnections()
	if err != nil {
		return errors.Wrap(err, "readconnections")
	}

	// We only save so that we'll get any defaults if the file doesn't exist
	err = stgSQL.Save()
	if err != nil {
		return errors.Wrap(err, "stgsql.save")
	}

	err = servers.importCSV()
	if err != nil {
		return errors.Wrap(err, "importCSV")
	}

	// Read in the JSON settings
	err = readConnections()
	if err != nil {
		return errors.Wrap(err, "readConnections")
	}
	return nil
}

func readConnections() error {
	var err error
	a, err := settings.ReadConnectionsDecrypted()
	if err != nil {
		return errors.Wrap(err, "readconnections")
	}
	for _, c := range a.SQLServers {
		err = servers.AddFromSettings(c.ServerKey, *c, false)
		if err != nil {
			return errors.Wrap(err, "addfromsettings")
		}
	}
	return nil
}

// func ReadCache() error {

// 	wd, err := osext.ExecutableFolder()
// 	if err != nil {
// 		logrus.Error(errors.Wrap(err, "osext.executablefolder"))
// 		return errors.Wrap(err, "osext.executablefolder")
// 	}
// 	dir := filepath.Join(wd, "cache")
// 	logrus.Debugf("setup: cache: %s", dir)

// 	// read cached wait stats
// 	br, err := bucket.NewReader(bucket.WaitsPrefix, dir)
// 	if err != nil {
// 		logrus.Error(errors.Wrap(err, "bucket.newreader"))
// 		return nil
// 	}

// 	logrus.Debug("cache: reader configured")
// 	go br.StartReader()
// 	var nr, nw int64
// 	loadStart := time.Now()
// 	limit := time.Now().Add(-60 * time.Minute)
// 	timec := time.After(10 * time.Second)
// 	first := true
// 	// drain the channel
// Loop:
// 	for {
// 		select {
// 		case <-timec:
// 			logrus.Error("cache: waits: reader: timeout")
// 			break Loop
// 		case str, ok := <-br.Results:
// 			if first {
// 				logrus.Trace("cache: waits: reading first result")
// 			}
// 			if !ok {
// 				logrus.Debug("cache: waits: reader: empty channel")
// 				break Loop
// 			}
// 			nr++

// 			// unmarshal ServerEvent
// 			var se bucket.ServerEvent
// 			logrus.Tracef("cache: waits: reading: unmarshal server event: %d", nr)
// 			err = json.Unmarshal([]byte(str), &se)
// 			if err != nil {
// 				logrus.Error(errors.Wrap(err, "se.unmarshal"))
// 				continue
// 			}

// 			// unmarshal WaitStats
// 			var waits waitmap.Waits
// 			//WinLogln("cache: reading: unmarshal waits")
// 			err = json.Unmarshal([]byte(se.Payload), &waits)
// 			if err != nil {
// 				logrus.Error(errors.Wrap(err, "waits: payload.unmarshal"))
// 				continue
// 			}

// 			if waits.EventTime.Before(limit) {
// 				// logrus.Trace("cache: reading: more than 60min ago")
// 				continue
// 			}

// 			//get the wrapper
// 			logrus.Trace("cache: waits: reading: get wrapper")
// 			wr, ok := servers.GetWrapper(se.MapKey)
// 			if !ok {
// 				logrus.Tracef("cache: waits: reading: server not found: %s", se.MapKey)
// 				continue
// 			}

// 			// enqueue the waitstats
// 			logrus.Trace("cache: waits: reading: enqueue")
// 			wr.Lock()
// 			wr.Waits.Enqueue(&waits)
// 			if wr.Waits.Capacity() > 60 {
// 				_ = wr.Waits.Dequeue()
// 			}
// 			wr.Unlock()
// 			nw++

// 			if first {
// 				logrus.Trace("cache: waits: processed first result")
// 				first = false
// 			}
// 		}
// 	}

// 	if br.Err != nil {
// 		logrus.Errorf("br.Err: waits: %v", br.Err)
// 	}
// 	loadDuration := time.Since(loadStart)
// 	logrus.Info(fmt.Sprintf("Waits: Read: %s  Used: %s (%s)", humanize.Comma(nr), humanize.Comma(nw), loadDuration.String()))
// 	return nil
// }

// launchBatchUpdates sorts maps, tags, etc.
func launchBatchUpdates() error {
	defer failure.HandlePanic()
	logrus.Debug("Launch Batch Updates...")

	quit := make(chan struct{})
	batchTicker := time.NewTicker(time.Duration(15) * time.Second)
	go func() {
		defer failure.HandlePanic()
		for {
			select {
			case <-batchTicker.C:
				servers.BatchUpdates()
			case <-quit:
				batchTicker.Stop()
				return
			}
		}
	}()

	return nil
}
