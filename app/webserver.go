package app

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"math"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/leekchan/gtf"
	"github.com/pkg/errors"
	"github.com/scalesql/isitsql/internal/appringlog"
	"github.com/scalesql/isitsql/internal/build"
	"github.com/scalesql/isitsql/internal/diskio"
	"github.com/scalesql/isitsql/internal/gui"
	"github.com/scalesql/isitsql/internal/hadr"
	"github.com/scalesql/isitsql/internal/logring"
	"github.com/scalesql/isitsql/internal/mssql/session"
	"github.com/scalesql/isitsql/internal/settings"
	"github.com/scalesql/isitsql/internal/waitmap"
	"github.com/scalesql/isitsql/static"
	"github.com/sirupsen/logrus"
)

type PageAlerts struct {
	Errors   map[string]PollError
	Warnings map[string]PollError
}

// Context is my custom context for web pages
type Context struct {
	Title               string
	Static              string
	Servers             []SqlServer
	OneServer           *SqlServer
	UnixNow             int64
	HeaderRight         string
	ErrorList           PageAlerts
	SortedKeys          []string
	TagList             map[string]tag
	SelectedTag         string
	AppConfig           appConfig
	TotalLine           TotalLine
	MenuTwoSelected     string
	Message             string
	MessageClass        string
	EnableSave          bool
	BackupIssues        int
	ServerPageActiveTab string // used to indicate which (if any) tab should be active
}

// getContext populates the context with basic values
func getContext(title string) Context {
	return Context{
		Title:       title,
		HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
		ErrorList:   getServerErrorList(),
		TagList:     globalTagList.getTags(),
		AppConfig:   getGlobalConfig(),
	}
}

func Chart2(w http.ResponseWriter, req *http.Request) {
	//log.Println("Got a chart2 request...")
	context := Context{Title: "Chart2!"}
	render(w, "chart2", context)
}

func CpuChart(w http.ResponseWriter, req *http.Request) {
	//log.Println("Got a CpuChart request...")
	context := Context{Title: "CpuChart Page"}
	render(w, "cpuchart", context)
}

func (l SqlServerArray) getTotal() TotalLine {
	var t TotalLine
	var d diskio.VirtualFileStats

	instances := make(map[string]bool)
	machines := make(map[string]bool)

	for _, v := range l {

		// if we dont' have this information just continue through the loop
		//v.RLock()
		if v.Domain == "" || v.PhysicalName == "" || v.ServerName == "" {
			//v.RUnlock()
			continue
		}

		machineKey := v.Domain + "\\" + v.PhysicalName
		instanceKey := v.Domain + "\\" + v.ServerName
		//v.RUnlock()

		// Set instance level counters
		_, exists := instances[instanceKey]

		if !exists {
			instances[instanceKey] = true
			t.Count++
			//v.RLock()
			t.DataSizeKB += v.DataSizeKB
			t.LogSizeKB += v.LogSizeKB
			d.Add(v.DiskIODelta)
			t.SQLServerMemoryKB += v.SqlServerMemoryKB

			//t.CPUCount += v.CpuCount
			t.Databases += v.DatabaseCount
			t.CoreUsageFactor += v.LastSQLCPU
			t.CoresUsedSQL += v.CoresUsedSQL
			//v.RUnlock()

			t.SQLPerSecond += v.GetLastSqlPerSecond()
		}

		// set machine level counters
		_, exists = machines[machineKey]
		if !exists {
			machines[machineKey] = true
			//v.RLock()
			t.CPUCount += v.CpuCount
			t.MachineCount++
			t.PhysicalMemoryKB += v.PhysicalMemoryKB
			t.CoresUsedOther += v.CoresUsedOther
			t.MemoryCapKB += v.MemoryCap()
			//v.RUnlock()
		}
	}

	t.DiskIO = d

	return t
}

func infoGraphicPage(w http.ResponseWriter, req *http.Request) {

	//t0 := time.Now()
	// servers.RLock()
	// s := make(SqlServerArray, len(servers.Servers))
	// for i, v := range servers.SortedKeys {
	// 	s[i] = servers.Servers[v]
	// }
	// servers.RUnlock()

	s := servers.CloneAll()

	t := s.getTotal()

	context := Context{
		Title:       "Summary - IsItSQL",
		HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
		SortedKeys:  servers.SortedKeys,
		TagList:     globalTagList.getTags(),
		SelectedTag: "",
		ErrorList:   getServerErrorList(),
		AppConfig:   getGlobalConfig(),
		TotalLine:   t,
	}

	//t0 = time.Now()
	renderFS(w, "infographic", context)
	//d2 := time.Now().Sub(t0)
	// msg := fmt.Sprint("Get Servers: ", d0, "  Get Total: ", d1, "  Render: ", d2)
	//msg := fmt.Sprintf("Index Page: Get Servers: %v  Get Total: %v  Render: %v", d0, d1, d2)
	//GLOBAL_RINGLOG.Enqueue(msg)
}

func pollingPage(w http.ResponseWriter, req *http.Request) {
	type poll struct {
		MapKey             string
		URL                string
		DisplayName        string
		Domain             string
		ServerName         string
		CSSClass           string
		IsPolling          bool
		PollStart          time.Time
		PollDuration       time.Duration
		LastPollTime       time.Time
		LastPollError      string
		LastPollErrorClean string
	}

	// Get the list of keys
	// keys := servers.Keys()
	ss := servers.CloneAll()
	polls := make([]poll, 0, len(ss))
	for _, s := range ss {
		var p poll
		p.MapKey = s.MapKey
		p.URL = s.URL()
		p.DisplayName = s.DisplayName()
		p.Domain = s.Domain
		p.ServerName = s.ServerName
		p.CSSClass = s.GetTableCssClass()
		p.IsPolling = s.IsPolling
		p.PollStart = s.PollStart
		p.PollDuration = s.PollDuration
		p.LastPollError = s.LastPollError
		p.LastPollErrorClean = s.LastPollErrorClean(45)
		p.LastPollTime = s.LastPollTime

		if p.IsPolling {
			p.PollDuration = time.Since(p.PollStart)
		}
		polls = append(polls, p)
	}

	context := struct {
		Context
		Polls []poll
	}{
		Context: Context{
			Title:       "Is It SQL - Polling",
			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			SortedKeys:  servers.SortedKeys,
			TagList:     globalTagList.getTags(),
			ErrorList:   getServerErrorList(),
			AppConfig:   getGlobalConfig(),
		},
		Polls: polls,
	}
	renderFSDynamic(w, "polling", context)
}

func systemTagPage(w http.ResponseWriter, req *http.Request) {

	var grandTotal TotalLine

	tagMap := getTagLines(true)

	context := struct {
		Context
		Tags map[string]TotalLine
	}{
		Context: Context{
			Title:       "Generated Tags - IsItSQL",
			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			SortedKeys:  servers.SortedKeys,
			TagList:     globalTagList.getTags(),
			ErrorList:   getServerErrorList(),
			AppConfig:   getGlobalConfig(),
			TotalLine:   grandTotal,
		},
		Tags: tagMap,
	}
	renderFSDynamic(w, "tagsummary", context)
}

func getTagLines(isgenerated bool) map[string]TotalLine {

	tagMap := make(map[string]TotalLine)

	// Just get a local copy of the tag map
	globalTagList.RLock()
	//fmt.Println(globalTagList)
	localTagMap := globalTagList.Tags
	globalTagList.RUnlock()

	//fmt.Println(tags)
	for k, t := range localTagMap {
		// skip system tags
		if t.IsGenerated != isgenerated {
			continue
		}

		var tl TotalLine
		ss := make(SqlServerArray, 0, len(t.Servers))
		for key := range t.Servers {
			srv, _ := servers.CloneOne(key)
			ss = append(ss, srv)
		}
		tl = ss.getTotal()

		tagMap[k] = tl
	}

	return tagMap
}

func userTagPage(w http.ResponseWriter, req *http.Request) {

	var grandTotal TotalLine

	tagMap := getTagLines(false)

	context := struct {
		Context
		Tags map[string]TotalLine
	}{
		Context: Context{
			Title:       "User Tags - IsItSQL",
			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			SortedKeys:  servers.SortedKeys,
			TagList:     globalTagList.getTags(),
			ErrorList:   getServerErrorList(),
			AppConfig:   getGlobalConfig(),
			TotalLine:   grandTotal,
		},
		Tags: tagMap,
	}
	renderFSDynamic(w, "tagsummary", context)
}

func tagPage(w http.ResponseWriter, req *http.Request) {

	var t TotalLine
	t.Count = 1

	tagParm := strings.ToLower(req.PathValue("tag"))

	globalTagList.RLock()
	tag, ok := globalTagList.Tags[tagParm]
	globalTagList.RUnlock()

	// This isn't a valid tag
	if !ok {
		renderErrorPage("Invalid Tag", fmt.Sprintf("Tag Not Found: %s", tagParm), w)
		return
	}

	servers.RLock()
	s := make(SqlServerArray, len(tag.Servers))
	i := 0
	for key := range tag.Servers {
		s[i] = servers.Servers[key].SqlServer
		i++
	}
	servers.RUnlock()
	//s := servers.CloneAll()
	// Get the totals
	t = s.getTotal()

	context := Context{
		Title:       fmt.Sprintf("tag: %s - IsItSQL", tagParm),
		Servers:     s,
		HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
		SortedKeys:  servers.SortedKeys,
		TagList:     globalTagList.getTags(),
		SelectedTag: tagParm,
		ErrorList:   getServerErrorList(),
		AppConfig:   getGlobalConfig(),
		TotalLine:   t,
	}
	renderFS(w, "index", context)
}

func logPage(w http.ResponseWriter, req *http.Request) {

	//fmt.Println(GLOBAL_RINGLOG.capacity)

	context := struct {
		Context
		// Title       string
		// UnixNow     int64
		// Servers     map[string]SqlServer
		// ErrorList   map[string]PollError
		// HeaderRight string
		LogEvents []appringlog.RingLogEvent
		//TagList     map[string]tag
	}{
		Context: Context{
			Title:       "Log Events",
			UnixNow:     time.Now().Unix() * 1000,
			ErrorList:   getServerErrorList(),
			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			TagList:     globalTagList.getTags(),
			AppConfig:   getGlobalConfig(),
		},
		LogEvents: GLOBAL_RINGLOG.NewestValues(),
	}

	//fmt.Println(len(context.LogEvents))
	renderFSDynamic(w, "log", context)
}

func dynamicWaitsDiagnostic(w http.ResponseWriter, req *http.Request) {

	context := struct {
		Context
		LogEvents []logring.Event
	}{
		Context: Context{
			Title:       "Log Events",
			UnixNow:     time.Now().Unix() * 1000,
			ErrorList:   getServerErrorList(),
			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			TagList:     globalTagList.getTags(),
			AppConfig:   getGlobalConfig(),
		},
		//LogEvents: GLOBAL_RINGLOG.NewestValues(),
	}

	server := req.PathValue("server")
	servers.RLock()
	wr, ok := servers.Servers[server]
	servers.RUnlock()

	if !ok {
		renderErrorPage("Invalid Server", fmt.Sprintf("Server Not Found: %s", server), w)
		return
	}
	s := wr.CloneSqlServer()
	context.OneServer = &s
	context.LogEvents = []logring.Event{}
	context.LogEvents = s.WaitBox.Messages()

	//fmt.Println(len(context.LogEvents))
	renderFSDynamic(w, "waits-diag", context)
}

func infoPage(w http.ResponseWriter, req *http.Request) {

	var err error

	u, err := user.Current()
	if err != nil {
		WinLogln("Error getting user: ", err)
	}

	m := make(map[string]string)

	globalStats.RLock()
	defer globalStats.RUnlock()

	cfg := getGlobalConfig()

	m["Service: Started"] = fmt.Sprintf("%s (%s)", globalStats.StartTime.Format(time.RFC1123), durationToShortString(globalStats.StartTime, time.Now()))
	m["Service: GOMAXPROCS"] = fmt.Sprintf("%d", runtime.GOMAXPROCS(0))
	m["Service: Account"] = u.Username
	m["Service: Account Struct"] = fmt.Sprintf("%+v", *u)
	m["App: Runtime"] = runtime.Version()

	m["App: Use Local Templates"] = strconv.FormatBool(cfg.UseLocalStatic)
	m["App: Config Mode"] = AppConfigMode.String()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	// https://lemire.me/blog/2024/03/17/measuring-your-systems-performance-using-software-go-edition/
	m["Service: Memory Allocated"] = fmt.Sprintf("%v", humanize.Bytes(mem.Alloc))
	m["Service: Memory Allocated Heap"] = fmt.Sprintf("%v", humanize.Bytes(mem.HeapAlloc))
	m["Service: Memory Sys Heap"] = fmt.Sprintf("%v", humanize.Bytes(mem.HeapSys))
	m["Service: Memory Sys"] = fmt.Sprintf("%v", humanize.Bytes(mem.Sys))
	m["Net: Request-Host"] = fmt.Sprintf("%v", req.Host)
	m["Net: Request-RemoteAddr"] = fmt.Sprintf("%v", req.RemoteAddr)

	if h, err := os.Hostname(); err == nil {
		m["Net: HostName"] = h
	} else {
		m["Net: HostName"] = err.Error()
		WinLogln(errors.Wrap(err, "info: os.hostname"))
	}

	ip, err := settings.IPFromRequest(req)
	if err != nil {
		m["Net: ipFromRequest-Error"] = err.Error()
	}
	m["Net: IPFromRequest"] = ip

	m["Net: X-Forwarded-For"] = req.Header.Get("X-Forwarded-For")
	m["Net: X-Real-IP"] = req.Header.Get("X-Real-IP")
	//m["settings.IPAddressFromRequest"], _ = settings.IPFromRequest(req)

	s, err := settings.ReadConfig()
	if err != nil {
		WinLogln("readConfig", err)
	} else {
		// m["App: Usage Reporting"] = fmt.Sprintf("%v", s.UsageReporting)
		// m["App: Error Reporting"] = fmt.Sprintf("%v", s.ErrorReporting)
		m["App: pprof Enabled"] = fmt.Sprintf("%v", s.EnableProfiler)
		m["App: Statsviz Enabled"] = fmt.Sprintf("%v", s.EnableStatsviz)
	}

	m["App: Git"] = buildGit
	m["App: Build Date"] = buildDate

	ips, err := settings.GetLocalIPs()
	if err != nil {
		m["Net: Local IPs"] = err.Error()
	} else {
		m["Net: Local IPs"] = strings.Join(ips, ", ")
	}

	canSave, err := settings.CanSave(req)
	if err != nil {
		m["User: Can Save"] = err.Error()
	} else {
		m["User: Can Save"] = strconv.FormatBool(canSave)
	}

	context := struct {
		Context
		// Title       string
		// UnixNow     int64
		// Servers     map[string]SqlServer
		// ErrorList   map[string]PollError
		// HeaderRight string

		//TagList     map[string]tag
		Values map[string]string
	}{
		Context: Context{
			Title:       "Information",
			UnixNow:     time.Now().Unix() * 1000,
			ErrorList:   getServerErrorList(),
			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			TagList:     globalTagList.getTags(),
			AppConfig:   getGlobalConfig(),
		},
		Values: m,
	}

	//fmt.Println(len(context.LogEvents))
	renderFSDynamic(w, "info", context)
}

func aboutPage(w http.ResponseWriter, req *http.Request) {
	var err error
	// TODO: make a map where the value has display text and hover text
	m := make(map[string]string)

	u, err := user.Current()
	if err != nil {
		WinLogln("about: user.current", err)
	} else {
		m["Service: Account"] = u.Username
	}

	//logrus.Debug("about: locking stats...")
	globalStats.RLock()
	defer globalStats.RUnlock()
	//logrus.Debug("about: lock acquired")
	m["Service: Started"] = fmt.Sprintf("%s (%s)", globalStats.StartTime.Format(time.RFC1123), durationToShortString(globalStats.StartTime, time.Now()))
	m["CPU: GOMAXPROCS"] = fmt.Sprintf("%d", runtime.GOMAXPROCS(0))

	exe, err := os.Executable()
	if err != nil {
		WinLogln(errors.Wrap(err, "about: os.executable"))
	} else {
		m["Service: Executable"] = fmt.Sprintf("%s (PID: %d)", exe, os.Getpid())
	}

	host, err := os.Hostname()
	if err != nil {
		WinLogln(errors.Wrap(err, "about: os.hostname:"))
	} else {
		m["Service: Host"] = host
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	//TODO https://lemire.me/blog/2024/03/17/measuring-your-systems-performance-using-software-go-edition/
	//TODO focus on HeapSys and HeapInUse
	m["Memory: Alloc"] = fmt.Sprintf("%v", humanize.Bytes(mem.Alloc))
	m["Memory: HeapAlloc"] = fmt.Sprintf("%v", humanize.Bytes(mem.HeapAlloc))
	m["Memory: HeapSys"] = fmt.Sprintf("%v", humanize.Bytes(mem.HeapSys))
	m["Memory: HeapInUse"] = fmt.Sprintf("%v", humanize.Bytes(mem.HeapInuse))
	m["Memory: Sys"] = fmt.Sprintf("%v", humanize.Bytes(mem.Sys))

	m["Version"] = build.Version()
	m["Version: Commit"] = build.Commit()
	m["Version: Built"] = build.Built().String()

	context := struct {
		Context
		Values   map[string]string
		Profiler bool
	}{
		Context: Context{
			Title:       "About",
			UnixNow:     time.Now().Unix() * 1000,
			ErrorList:   getServerErrorList(),
			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			TagList:     globalTagList.getTags(),
			AppConfig:   getGlobalConfig(),
		},
		Values: m,
	}

	// renderFSDynamic(w, "about", context)
	err = gui.ExecuteTemplates(w, context, "templates/base.html", "templates/about.html")
	if err != nil {
		WinLogln(err)
		logrus.Error(errors.Wrap(err, "about.html"))
	}
}

// func aboutPage(w http.ResponseWriter, req *http.Request) {

// 	context := struct {
// 		Context
// 	}{
// 		Context: Context{
// 			Title:       "About",
// 			UnixNow:     time.Now().Unix() * 1000,
// 			ErrorList:   getServerErrorList(),
// 			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
// 			AppConfig:   getGlobalConfig(),
// 		},
// 	}
// 	renderFSDynamic(w, "about", context)
// }

// func panicApp(w http.ResponseWriter, req *http.Request) {
// 	panic("OMG Panic!")
// }

// func racePage(w http.ResponseWriter, req *http.Request) {
// 	var wg sync.WaitGroup
// 	wg.Add(5)
// 	for i := 0; i < 5; i++ {
// 		go func() {
// 			//lint:ignore
// 			fmt.Println(i) // Not the 'i' you are looking for.
// 			wg.Done()
// 		}()
// 	}
// 	wg.Wait()
// }

func getServerErrorList() PageAlerts {
	pa := PageAlerts{
		Errors:   make(map[string]PollError),
		Warnings: make(map[string]PollError),
	}

	cfg := getGlobalConfig()
	if cfg.AGAlertMB == 0 {
		cfg.AGAlertMB = math.MaxInt64
	}
	if cfg.AGWarnMB == 0 {
		cfg.AGWarnMB = math.MaxInt64
	}

	keys := servers.Keys()
	for _, k := range keys {
		servers.RLock()
		ptr, ok := servers.Servers[k]
		servers.RUnlock()
		if !ok {
			continue
		}
		ptr.RLock()
		if ptr.LastPollError != "" {
			pa.Errors[ptr.MapKey] = PollError{
				FriendlyName: ptr.DisplayName(),
				InstanceName: ptr.ServerName,
				Error:        ptr.LastPollErrorClean(120),
				ErrorRaw:     ptr.LastPollError,
				LastPollTime: ptr.LastPollTime}
		}
		ptr.RUnlock()
	}

	// Get the repository error
	tm, err := GlobalRepository.RepositoryError()
	if err != nil {
		pa.Errors["IsItSQL: Repository:"] = PollError{
			FriendlyName: "IsItSQL: Repository:",
			Error:        err.Error(),
			LastPollTime: tm,
		}
	}

	// Get any AG errors
	aglist := hadr.PublicAGMap.Groups()
	for _, ag := range aglist {
		// compute the total send queue and redo queue
		var send, redo int64
		for _, r := range ag.Replicas {
			send += r.SendQueue
			redo += r.RedoQueue
		}
		sendMB := send / 1024
		redoMB := redo / 1024
		if ag.IsHealthy() && sendMB <= cfg.AGAlertMB && sendMB <= cfg.AGWarnMB && redoMB <= cfg.AGAlertMB && redoMB <= cfg.AGWarnMB {
			continue
		}
		// something needs to be displayed to the user
		mapKey := fmt.Sprintf("%s:%s", "AG", ag.DisplayName)
		pe := PollError{
			InstanceName: fmt.Sprintf("AG: %s (%s)", ag.DisplayName, ag.PrimaryReplica),
			Error:        fmt.Sprintf("%s: %s  (send: %s;  redo: %s)", ag.State, ag.Health, KBToString(send), KBToString(redo)),
			LastPollTime: ag.PollTime}

		// if the AG isn't online, or we are over the Alert levels
		if ag.State != "ONLINE" || send/1024 > cfg.AGAlertMB || redo/1024 > cfg.AGAlertMB {
			pa.Errors[mapKey] = pe
		} else {
			pa.Warnings[mapKey] = pe
		}
	}

	return pa
}

func dashboardPage(w http.ResponseWriter, req *http.Request) {

	// Put the parameters in an array
	var serverList [3]string
	parm := req.PathValue("servers")
	parm = strings.TrimSpace(parm)
	// If there aren't paramters, get the first three from tag or list
	if parm == "/" || parm == "" {

		// if we have a "dashboard" tag, get the first three of those
		globalTagList.RLock()
		dashTag, ok := globalTagList.Tags["dashboard"]
		globalTagList.RUnlock()

		// we have a dashboard tag
		if ok {
			s := dashTag.Servers

			// get the first three from that tag
			var i int
			for k := range s {
				if i > 2 {
					break
				}
				serverList[i] = k
				i++
			}
			// else just get the first three keys
		} else {
			servers.RLock()
			sk := servers.SortedKeys
			servers.RUnlock()

			if len(sk) > 0 {
				serverList[0] = sk[0]
			}

			if len(sk) > 1 {
				serverList[1] = sk[1]
			}

			if len(sk) > 2 {
				serverList[2] = sk[2]
			}

		}
	} else {
		i := 0
		for _, v := range strings.Split(parm, "/") {
			if len(v) > 0 {
				if i < 3 {
					serverList[i] = v
				}
				i++
			}
		}
	}

	context := struct {
		Context
		S2         []SqlServer
		ServerList [3]string
	}{
		Context: Context{
			Title:       "Dashboard",
			UnixNow:     time.Now().Unix() * 1000,
			ErrorList:   getServerErrorList(),
			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			TagList:     globalTagList.getTags(),
			AppConfig:   getGlobalConfig(),
		},
	}

	a := make([]SqlServer, 0)

	// Go through our list of servers for the dashboard and copy them to a new object
	for _, slv := range serverList {
		// servers.RLock()
		// s, ok := servers.Servers[slv]
		// servers.RUnlock()
		// if ok {
		// 	s.RLock()
		// 	tmp := *s
		// 	s.RUnlock()
		// 	a = append(a, &tmp)
		// }
		srv, ok := servers.CloneOne(slv)
		if ok {
			a = append(a, srv)
		}
	}

	// Sort based on the Display Name
	sort.Slice(a, func(i, j int) bool { return a[i].DisplayName() < a[j].DisplayName() })
	context.S2 = a

	renderFSDynamic(w, "dashboard", context)
}

func serverXEPage(w http.ResponseWriter, req *http.Request) {

	id := req.PathValue("server")
	wr, ok := servers.GetWrapper(id)
	if !ok {
		renderErrorPage("Invalid Server", fmt.Sprintf("Server Not Found: %s", id), w)
		return
	}
	s := wr.CloneSqlServer()

	var htmlTitle string
	if len(s.ServerName) > 0 {
		htmlTitle = html.EscapeString(s.ServerName) + " - Is It SQL"
	} else {
		htmlTitle = "Is It Sql"
	}

	var events []*xEvent
	var err error
	events, err = wr.getXESessions()
	if err != nil {
		logrus.Error(errors.Wrap(err, "getxesessions"))
	}

	// sort the slice properly
	sort.Slice(events, func(i, j int) bool {
		return events[i].TimeStamp.Before(events[j].TimeStamp)
	})

	// Reverse it to put the newest first
	// There's probably a better way to do this
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}

	context := struct {
		Context
		// Title                 string
		// OneServer             *SqlServer
		// UnixNow               int64
		Events []*xEvent
		// HeaderRight           string
		// ErrorList             map[string]PollError
		//TagList               map[string]tag
		// Jobs      []*ActiveJob
	}{
		Context: Context{
			Title:               htmlTitle,
			OneServer:           &s,
			HeaderRight:         fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			ErrorList:           getServerErrorList(),
			TagList:             globalTagList.getTags(),
			AppConfig:           getGlobalConfig(),
			ServerPageActiveTab: "xe",
		},
		Events: events,
	}

	renderFSDynamic(w, "xe", context)

}

func renderErrorPage(title, msg string, w http.ResponseWriter) {
	var pageData struct {
		Context
		Message string
	}
	pageData.Context = getContext(title)
	pageData.Message = msg
	renderFSDynamic(w, "error", pageData)
}

func serverW2Page(w http.ResponseWriter, req *http.Request) {
	var pageData struct {
		Context
		Sessions []session.Session
		Blocking bool
	}

	pageData.Context = getContext("Server Not Found")
	pageData.ServerPageActiveTab = "w2"

	server := req.PathValue("server")
	servers.RLock()
	wr, ok := servers.Servers[server]
	servers.RUnlock()

	if !ok {
		renderErrorPage("Invalid Server", fmt.Sprintf("Server Not Found: %s", server), w)
		return
	}
	s := wr.CloneSqlServer()
	pageData.OneServer = &s

	// Get active sessions
	sessions, err := session.Get(context.Background(), wr.DB, wr.MajorVersion)
	if err != nil {
		WinLogln(fmt.Sprintf("Get Active Sessions: %v", err))
		pageData.Context.Message = fmt.Sprintf("Get Active Sessions: %v", err)
		pageData.Context.MessageClass = "alert-danger"
	}
	pageData.Sessions = sessions
	// check for blocking
	for _, s := range sessions {
		if s.BlockerID != 0 || s.TotalBlocked != 0 {
			pageData.Blocking = true
		}
	}

	var htmlTitle string
	if len(s.FriendlyName) > 0 {
		htmlTitle = html.EscapeString(s.FriendlyName)
	}

	if len(s.ServerName) > 0 {
		if len(htmlTitle) > 0 {
			htmlTitle += " (" + html.EscapeString(s.ServerName) + ")"
		} else {
			htmlTitle = html.EscapeString(s.ServerName)
		}
	}

	if len(htmlTitle) > 0 {
		htmlTitle += " - Is It SQL"
	} else {
		htmlTitle = "Is It Sql"
	}

	pageData.Title = htmlTitle

	renderFSDynamic(w, "server-waits", pageData)
}

func newchartPage(w http.ResponseWriter, req *http.Request) {

	var pageData struct {
		Context
		Sessions []session.Session
		Blocking bool
	}

	pageData.Context = getContext("Server Not Found")
	pageData.Context.ServerPageActiveTab = "activity"

	server := req.PathValue("server")
	servers.RLock()
	wr, ok := servers.Servers[server]
	servers.RUnlock()

	if !ok {
		renderErrorPage("Invalid Server", fmt.Sprintf("Server Not Found: %s", server), w)
		return
	}
	s := wr.CloneSqlServer()
	pageData.OneServer = &s

	var htmlTitle string
	if len(s.FriendlyName) > 0 {
		htmlTitle = html.EscapeString(s.FriendlyName)
	}

	if len(s.ServerName) > 0 {
		if len(htmlTitle) > 0 {
			htmlTitle += " (" + html.EscapeString(s.ServerName) + ")"
		} else {
			htmlTitle = html.EscapeString(s.ServerName)
		}
	}

	if len(htmlTitle) > 0 {
		htmlTitle += " - Is It SQL"
	} else {
		htmlTitle = "Is It Sql"
	}

	pageData.Title = htmlTitle

	renderFSDynamic(w, "server-newchart", pageData)
}

func serverPage(w http.ResponseWriter, req *http.Request) {

	var pageData struct {
		Context
		Sessions []session.Session
		Blocking bool
	}

	pageData.Context = getContext("Server Not Found")
	pageData.Context.ServerPageActiveTab = "activity"

	server := req.PathValue("server")
	servers.RLock()
	wr, ok := servers.Servers[server]
	servers.RUnlock()

	if !ok {
		renderErrorPage("Invalid Server", fmt.Sprintf("Server Not Found: %s", server), w)
		return
	}
	s := wr.CloneSqlServer()
	pageData.OneServer = &s

	sessions, err := session.Get(context.Background(), wr.DB, wr.MajorVersion)
	if err != nil {
		//sessions = make([]*ActiveSession, 0)
		WinLogln(fmt.Sprintf("Get Active Sessions: %v", err))
		pageData.Context.Message = fmt.Sprintf("Get Active Sessions: %v", err)
		pageData.Context.MessageClass = "alert-danger"
	}
	pageData.Sessions = sessions

	// check for blocking
	for _, s := range sessions {
		if s.BlockerID != 0 || s.TotalBlocked != 0 {
			pageData.Blocking = true
			break
		}
	}

	var htmlTitle string
	if len(s.FriendlyName) > 0 {
		htmlTitle = html.EscapeString(s.FriendlyName)
	}

	if len(s.ServerName) > 0 {
		if len(htmlTitle) > 0 {
			htmlTitle += " (" + html.EscapeString(s.ServerName) + ")"
		} else {
			htmlTitle = html.EscapeString(s.ServerName)
		}
	}

	if len(htmlTitle) > 0 {
		htmlTitle += " - Is It SQL"
	} else {
		htmlTitle = "Is It Sql"
	}

	pageData.Title = htmlTitle

	renderFSDynamic(w, "server", pageData)
}

func serverConnPage(w http.ResponseWriter, req *http.Request) {

	var pageData struct {
		Context
		Pool       map[string]interface{}
		Connection map[string]interface{}
	}

	pageData.Context = getContext("Server Not Found")
	pageData.Pool = make(map[string]interface{})
	pageData.Connection = make(map[string]interface{})
	pageData.Context.ServerPageActiveTab = "info"
	server := req.PathValue("server")
	servers.RLock()
	wr, ok := servers.Servers[server]
	servers.RUnlock()
	if !ok {
		renderErrorPage("Invalid Server", fmt.Sprintf("Server Not Found: %s", server), w)
		return
	}

	s := wr.CloneSqlServer()
	pageData.OneServer = &s
	stats := s.Stats

	pageData.Pool["MaxOpenConnections"] = stats.MaxOpenConnections
	pageData.Pool["OpenConnections"] = stats.OpenConnections
	pageData.Pool["InUse"] = stats.InUse
	pageData.Pool["Idle"] = stats.Idle
	pageData.Pool["WaitCount"] = stats.WaitCount
	pageData.Pool["WaitDuration"] = stats.WaitDuration
	pageData.Pool["MaxIdleClosed"] = stats.MaxIdleClosed
	pageData.Pool["MaxIdleTimeClosed"] = stats.MaxIdleTimeClosed
	pageData.Pool["MaxLifetimeClosed"] = stats.MaxLifetimeClosed

	wr.RLock()
	db := wr.DB
	wr.RUnlock()

	rows, err := db.Query(`
		SELECT	c.session_id				AS SessionID, 
				c.connect_time				AS ConnectTime, 
				c.net_transport				AS NetTransport, 
				c.protocol_type				AS ProtocolType, 
				c.protocol_version			AS ProtocolVersion, 
				c.auth_scheme				AS AuthScheme, 
				s.login_time				AS LoginTime, 
				s.host_name					AS HostName, 
				s.program_name				AS ProgramName, 
				s.client_version			AS ClientVersion, 
				s.client_interface_name		AS ClientInterfaceName, 
				s.login_name				AS LoginName, 
				SYSDATETIME()				AS [Now] 
		FROM	sys.dm_exec_connections c
		JOIN	sys.dm_exec_sessions s ON s.session_id = c.session_id
		WHERE	c.session_id = @@SPID
	`)

	if err == nil {
		defer rows.Close()
		cols, _ := rows.Columns()

		pointers := make([]interface{}, len(cols))
		container := make([]interface{}, len(cols))
		for i := range pointers {
			pointers[i] = &container[i]
		}
		rows.Next()
		rows.Scan(pointers...)

		for i, v := range container {
			//lint:ignore S1034 we don't use the type
			switch v.(type) {
			case []byte:
				pageData.Connection[cols[i]] = string(v.([]byte))
			default:
				pageData.Connection[cols[i]] = v
			}
		}
	} else {
		WinLogln(errors.Wrap(err, "query.connection"))
	}

	var htmlTitle string
	if len(s.FriendlyName) > 0 {
		htmlTitle = html.EscapeString(s.FriendlyName)
	}

	if len(s.ServerName) > 0 {
		if len(htmlTitle) > 0 {
			htmlTitle += " (" + html.EscapeString(s.ServerName) + ")"
		} else {
			htmlTitle = html.EscapeString(s.ServerName)
		}
	}

	if len(htmlTitle) > 0 {
		htmlTitle += " - Is It SQL"
	} else {
		htmlTitle = "Is It Sql"
	}

	pageData.Title = htmlTitle

	renderFSDynamic(w, "server-conn", pageData)
}

func JSONError(w http.ResponseWriter, err interface{}, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(err)
}

func serverJSONPage(w http.ResponseWriter, req *http.Request) {
	server := req.PathValue("server")
	wr, ok := servers.GetWrapper(server)
	if !ok {
		JSONError(w, "not found", http.StatusNotFound)
		return
	}
	srv := wr.CloneSqlServer()
	js, err := json.Marshal(srv)
	if err != nil {
		JSONError(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func serverRawPage(w http.ResponseWriter, req *http.Request) {

	var l, l2 int
	var htmlTitle string
	var buf, buf2 bytes.Buffer
	var compressedSize, cs2 int
	var j, j2 []byte
	var err error
	var g, g2 *gzip.Writer
	var jstring, j2string string

	server := req.PathValue("server")

	var s SqlServer
	wr, ok := servers.GetWrapper(server)
	if !ok {
		htmlTitle = "Server Not Found - Is It SQL"
		goto Empty
	}
	s = wr.CloneSqlServer()

	if len(s.ServerName) > 0 {
		htmlTitle = html.EscapeString(s.ServerName) + " - Is It SQL"
	} else {
		htmlTitle = "Is It Sql"
	}

	j, err = json.MarshalIndent(s, "", "    ")
	if err != nil {
		WinLogln("serverRawPage: JsonMarshall: ", err)
	}
	jstring = string(j)

	l = len(jstring)

	// Compress the JSON
	g = gzip.NewWriter(&buf)
	if _, err = g.Write(j); err != nil {
		WinLogln("Compress JSON: ", err)
		return
	}
	if err = g.Close(); err != nil {
		WinLogln("GZIP Writer Close: ", err)
		return
	}

	compressedSize = buf.Len()

	// Mashal without the indents --------------------------------------------------
	j2, err = json.Marshal(s)
	if err != nil {
		WinLogln("serverRawPage: JsonMarshal2: ", err)
	}
	j2string = string(j2)

	l2 = len(j2string)

	// Compress the JSON
	g2 = gzip.NewWriter(&buf2)
	if _, err = g2.Write(j2); err != nil {
		WinLogln("Compress JSON2: ", err)
		return
	}
	if err = g2.Close(); err != nil {
		WinLogln("GZIP Writer Close: ", err)
		return
	}

	cs2 = buf2.Len()

Empty:

	context := struct {
		Context
		RawJSON           string
		JSONLength        int
		CompressedLength  int
		JSONLength2       int
		CompressedLength2 int
	}{
		Context: Context{
			Title:       htmlTitle,
			OneServer:   &s,
			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			ErrorList:   getServerErrorList(),
			TagList:     globalTagList.getTags(),
			AppConfig:   getGlobalConfig(),
		},
		RawJSON:           jstring,
		JSONLength:        l,
		CompressedLength:  compressedSize,
		JSONLength2:       l2,
		CompressedLength2: cs2,
	}

	renderFSDynamic(w, "serverraw", context)
}

func serverQueryStats(w http.ResponseWriter, req *http.Request) {

	id := req.PathValue("server")
	s, ok := servers.CloneOne(id)
	if !ok {
		renderErrorPage("Invalid Server", fmt.Sprintf("Server Not Found: %s", id), w)
		return
	}

	servers.RLock()
	// query stats function is at the wrapper level.  Ugh.
	wr, ok := servers.Servers[id]
	servers.RUnlock()
	if !ok {
		renderErrorPage("Invalid Server", fmt.Sprintf("Server Not Found: %s", id), w)
		return
	}

	var htmlTitle string
	if len(s.ServerName) > 0 {
		htmlTitle = html.EscapeString(s.ServerName) + " - Query Stats - Is It SQL"
	} else {
		htmlTitle = "Is It Sql"
	}

	qs, err := wr.getQueryStats()
	if err != nil {
		//if err.Error() != "Stmt did not create a result set" {
		WinLogln("Error getting query stats: ", err)
		//}
	}

	context := struct {
		Context
		QueryStats []queryStats
	}{
		Context: Context{
			Title:       htmlTitle,
			OneServer:   &s,
			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			ErrorList:   getServerErrorList(),
			TagList:     globalTagList.getTags(),
			AppConfig:   getGlobalConfig(),
		},
		QueryStats: qs,
		//RawJSON: jstring,
	}

	renderFSDynamic(w, "queryStats", context)
}

// func serverBackupPage(w http.ResponseWriter, req *http.Request) {

// 	server := req.PathValue("server")
// 	//log.Println("Got a server page request for", server)
// 	servers.RLock()
// 	s := servers.Servers[server]
// 	servers.RUnlock()

// 	s.RLock()
// 	var htmlTitle string
// 	if len(s.ServerName) > 0 {
// 		htmlTitle = html.EscapeString(s.ServerName) + " - Databases - Is It SQL"
// 	} else {
// 		htmlTitle = "Is It Sql"
// 	}

// 	type fullDB struct {
// 		DB     *Database
// 		Backup *databaseBackups
// 	}

// 	rows := make(map[string]*fullDB)
// 	for k, v := range s.Databases {

// 		// ignore tempdb
// 		if k == "tempdb" {
// 			continue
// 		}

// 		// set the database
// 		db := fullDB{
// 			DB: v,
// 		}
// 		rows[k] = &db

// 		// set the backups
// 		bu := s.Backups[k]

// 		if bu != nil {
// 			rows[k].Backup = bu
// 		} else {
// 			dbb := databaseBackups{
// 				DatabaseName: k,
// 			}
// 			rows[k].Backup = &dbb
// 		}
// 	}

// 	s.RUnlock()

// 	context := struct {
// 		Context
// 		//Databases map[string]*Database
// 		Backups map[string]*fullDB
// 	}{
// 		Context: Context{
// 			Title:       htmlTitle,
// 			OneServer:   s,
// 			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
// 			ErrorList:   getServerErrorList(),
// 			TagList:     globalTagList.getTags(),
// 			AppConfig:   getGlobalConfig(),
// 		},
// 		Backups: rows,
// 	}

// 	renderFSDynamic(w, "backups", context)
// }

func serverDatabasesPage(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("server")
	s, ok := servers.CloneOne(id)
	if !ok {
		renderErrorPage("Invalid Server", fmt.Sprintf("Server Not Found: %s", id), w)
		return
	}

	var htmlTitle string
	if len(s.ServerName) > 0 {
		htmlTitle = html.EscapeString(s.ServerName) + " - Databases - Is It SQL"
	} else {
		htmlTitle = "Is It Sql"
	}

	// Update the backup Values
	// TODO: I don't think I want to do this here
	//_ = s.setBackupAlert()

	context := struct {
		Context
		Databases map[int]*Database
		Snapshots []Snapshot
	}{
		Context: Context{
			Title:               htmlTitle,
			OneServer:           &s,
			HeaderRight:         fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			ErrorList:           getServerErrorList(),
			TagList:             globalTagList.getTags(),
			AppConfig:           getGlobalConfig(),
			ServerPageActiveTab: "databases",
		},
		Databases: s.Databases,
		Snapshots: s.Snapshots,
	}

	renderFSDynamic(w, "databases", context)
}

func mirroredDatabasesPage(w http.ResponseWriter, req *http.Request) {
	htmlTitle := html.EscapeString("Mirrored Databases - Is It SQL")

	dbs, err := getMirroredDatabases()
	if err != nil {
		GLOBAL_RINGLOG.Enqueue(errors.Wrap(err, "getmirroreddatabases").Error())
	}

	context := struct {
		Context
		Databases map[string]*mirroredDatabase
	}{
		Context: Context{
			Title:       htmlTitle,
			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			ErrorList:   getServerErrorList(),
			TagList:     globalTagList.getTags(),
			AppConfig:   getGlobalConfig(),
		},
		Databases: dbs,
	}
	renderFSDynamic(w, "dbm", context)
}

// ServerWaitPage displays the page for the server waits
func ServerWaitPage(w http.ResponseWriter, req *http.Request) {

	type WaitDisplay struct {
		Wait     string
		Duration int64
		MappedTo string
		Excluded bool
	}

	type T1 struct {
		Context
		WaitList      map[string]WaitDisplay
		WaitGroupList SortedMapInt64
		RawJSON       string
	}

	var ok bool
	key := req.PathValue("server")

	var p1 T1
	p1.Title = key
	p1.TagList = globalTagList.getTags()
	p1.ErrorList = getServerErrorList()
	globalConfig.RLock()
	p1.AppConfig = globalConfig.AppConfig
	globalConfig.RUnlock()

	s, ok := servers.CloneOne(key)
	if !ok {
		renderErrorPage("Invalid Server", fmt.Sprintf("Server Not Found: %s", key), w)
		return
	}

	p1.OneServer = &s
	p1.Title = s.ServerName + " Waits"

	//waits := s.Waits
	results, err := waitmap.ReadWaitFiles(key)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		WinLogln(errors.Wrap(err, "serverwaitpage").Error())
		logrus.Error(err)
		return
	}
	// logrus.Infof("results: %d (%s)", len(results), time.Since(start))
	wr := WaitRing{}
	//start = time.Now()
	for _, w := range results {
		waits := w
		wr.Enqueue(&waits)
	}

	j, err := json.MarshalIndent(wr, "", "    ")
	if err != nil {
		WinLogln("serverwaitpage: jsonmarshal: ", err)
	} else {
		p1.RawJSON = string(j)
	}

	wl := make(map[string]WaitDisplay)
	//wgl := make(map[string]int64)

	var wurg WaitDisplay
	t := wr
	wv := t.Values()

	// Group all waits together
	for _, wt := range wv {
		// log.Println(i, wt.EventTime)

		for _, wd := range wt.Waits {
			_, ok = wl[wd.Wait]
			if wd.WaitTimeDelta > 0 {
				if ok {
					wurg = wl[wd.Wait]
					wurg.Duration += wd.WaitTimeDelta
					wl[wd.Wait] = wurg
				} else {
					wurg = WaitDisplay{
						Wait:     wd.Wait,
						Duration: wd.WaitTimeDelta,
						MappedTo: waitmap.Mapping.Mappings[wd.Wait].MappedTo,
						Excluded: waitmap.Mapping.Mappings[wd.Wait].Excluded}
					wl[wd.Wait] = wurg
				}
			}
		}
		p1.WaitList = wl

		// // Group all wait groups together
		// for wg, wgd := range wt.WaitSummary {
		// 	_, ok = wgl[wg]
		// 	if wgd > 0 {
		// 		if ok {
		// 			wgl[wg] += wgd
		// 		} else {
		// 			wgl[wg] = wgd
		// 		}
		// 	}
		// }
		// p1.WaitGroupList = wgl

	}

	twg := t.TopGroups()
	p1.WaitGroupList = twg
	// v := servers.Servers[server].Waits
	// v1 := v.Values()
	// log.Println(len(v1))

	renderFSDynamic(w, "serverwaits", p1)
}

func Chart3(w http.ResponseWriter, req *http.Request) {
	// log.Println("Got a chart3 request...")
	context := Context{Title: "Chart3!"}
	render(w, "chart3", context)
}

func Chart(w http.ResponseWriter, req *http.Request) {
	//log.Println("Got a chart request...")
	context := Context{Title: "Chart!"}
	render(w, "chart", context)
}

func License(w http.ResponseWriter, req *http.Request) {
	context := Context{Title: "License"}
	render(w, "license", context)
}

func GoogleChart(w http.ResponseWriter, req *http.Request) {
	context := Context{Title: "Google Chart"}
	render(w, "googlechart", context)
}

func JsonTest(w http.ResponseWriter, req *http.Request) {
	context := Context{Title: "JSON Test"}
	render(w, "json", context)
}

// This needs error handling!
func render(w http.ResponseWriter, tmpl string, context Context) {
	context.Static = STATIC_URL
	wd, err := os.Executable()
	if err != nil {
		WinLogln(err)
		renderErrorPage("Error", err.Error(), w)
		return
	}

	tmpl_list := []string{filepath.Join(wd, "templates/base.html"),
		filepath.Join(wd, fmt.Sprintf("templates/%s.html", tmpl))}

	t, err := template.New("base.html").Funcs(gtf.GtfFuncMap).ParseFiles(tmpl_list...)
	if err != nil {
		WinLogf("template.new: %v", err)
	}

	// t = t.Funcs(gtf.GtfFuncMap)
	err = t.ExecuteTemplate(w, "base.html", context)
	if err != nil {
		WinLogf("t.executetemplate: %v", err)
	}
}

func renderFS(w http.ResponseWriter, tmpl string, context Context) {

	globalConfig.RLock()
	useLocal := globalConfig.AppConfig.UseLocalStatic
	globalConfig.RUnlock()

	context.Static = STATIC_URL

	baseTemplateString, err := static.ReadFile(useLocal, "templates/base.html")
	if err != nil {
		WinLogln("FSString-base: ", err)
		return
	}

	baseTemplate, err := template.New(tmpl).Funcs(gtf.GtfFuncMap).Funcs(gui.TemplateFuncs).Parse(baseTemplateString)
	if err != nil {
		WinLogln("template-base: ", err)
		return
	}

	templateString, err := static.ReadFile(useLocal, fmt.Sprintf("templates/%s.html", tmpl))
	if err != nil {
		WinLogln("FSString-template: ", tmpl, err)
		return
	}
	finalTemplate, err := template.Must(baseTemplate.Clone()).Parse(templateString)
	if err != nil {
		WinLogln("template-template: ", tmpl, err)
		return
	}

	err = finalTemplate.ExecuteTemplate(w, tmpl, context)
	if err != nil {
		logrus.Errorf("template executing error: %s: %s: ", tmpl, err)
	}

}

func renderFSDynamic(w http.ResponseWriter, tmpl string, c2 interface{}) {
	// TODO: cache all the templates
	// https://stackoverflow.com/questions/50842389/parsing-multiple-templates-in-go

	globalConfig.RLock()
	useLocal := globalConfig.AppConfig.UseLocalStatic
	globalConfig.RUnlock()

	baseTemplateString, err := static.ReadFile(useLocal, "templates/base.html")
	if err != nil {
		WinLogln(errors.Wrap(err, "renderfsdynamic: static.readfile"))
		return
	}

	baseTemplate, err := template.New(tmpl).Funcs(gtf.GtfFuncMap).Funcs(gui.TemplateFuncs).Parse(baseTemplateString)
	if err != nil {
		WinLogln("template-base: ", err)
		return
	}

	templateString, err := static.ReadFile(useLocal, fmt.Sprintf("templates/%s.html", tmpl))
	if err != nil {
		WinLogln("FSString-template: ", tmpl, err)
		return
	}
	finalTemplate, err := template.Must(baseTemplate.Clone()).Parse(templateString)
	if err != nil {
		WinLogln("template-template: ", tmpl, err)
		return
	}

	err = finalTemplate.Execute(w, c2)
	if err != nil {
		logrus.Errorf("template executing error: %s: %s: ", tmpl, err)
	}
	//fmt.Printf("%v: %s: %s\n", useLocal, tmpl, time.Since(start))
}
