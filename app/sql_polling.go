package app

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/scalesql/isitsql/internal/logonce"
	"github.com/scalesql/isitsql/internal/mrepo"
	"github.com/scalesql/isitsql/internal/mssql/agent"
	"github.com/scalesql/isitsql/internal/waitmap"
	"github.com/scalesql/isitsql/internal/waitring"
	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"
)

var longPollThreshold = 120 * time.Second

// BatchUpdates run varioius batch jobs that need to happen every minute
func (s *ServerList) BatchUpdates() {
	setUseLocalStatic()
	s.SortKeys()
	s.mapTags()
}

// PollServers launches the polling for all the servers
func (s *ServerList) PollServers() {
	err := waitmap.Mapping.ReadWaitMapping("waits.txt")
	if err != nil {
		WinLogln("Error reading wait mappings: ", err)
	}

	s.RLock()
	topoll := s.SortedKeys
	s.RUnlock()

	for _, key := range topoll {
		globalPool.Poll(key)
	}
}

// getAllMetrics polls a SQL Server and updates metrics.  It returns a flag
// indicating if this was a big poll (to write the cache) and an error
func (s *SqlServerWrapper) getAllMetrics(forcequick bool) (bool, error) {
	var err error
	longPollError := errors.New("poll timeout after one minute")
	pollStartTime := time.Now()

	thisSortPriority := 999999

	err = s.resetDB()
	if err != nil {
		s.Lock()
		s.SortPriority = thisSortPriority - 1
		s.Unlock()
		return false, errors.Wrap(err, "resetDB")
	}

	if time.Since(pollStartTime) > longPollThreshold {
		return false, errors.Wrap(longPollError, "resetdb")
	}

	err = s.getName()
	if err != nil {
		s.Lock()
		s.SortPriority = thisSortPriority - 1
		s.Unlock()
		return false, errors.Wrap(err, "getName")
	}

	if time.Since(pollStartTime) > longPollThreshold {
		return false, errors.Wrap(longPollError, "longpoll: getname")
	}

	err = s.getServerInfo()
	if err != nil {
		return false, errors.Wrap(err, "getServerInfo")
	}

	if time.Since(pollStartTime) > longPollThreshold {
		return false, errors.Wrap(longPollError, "getserverinfo")
	}

	// Get availability groups
	s.RLock()
	majorVersion := s.MajorVersion
	s.RUnlock()

	if majorVersion >= 12 {
		if err = s.pollAG(); err != nil {
			return false, errors.Wrap(err, "pollAG")
		}
	}

	if forcequick {
		return false, nil
	}

	err = s.getConnectionInfo()
	if err != nil {
		return false, err
	}

	// Get the database stats
	s.Lock()
	s.Stats = s.DB.Stats()
	s.Unlock()

	// Get the database connection

	// Is it time for a big poll?
	s.RLock()
	//displayName := s.DisplayName()
	lastBigPoll := s.LastBigPoll
	reset := s.ResetOnThisPoll
	s.RUnlock()

	if !reset && time.Since(lastBigPoll) < time.Duration(51*time.Second) {
		return false, nil
	}

	// Start a big poll
	//WinLogln("Big Poll:", displayName)
	s.Lock()
	s.LastBigPoll = time.Now()
	s.Unlock()

	err = s.GetServerMemory()
	if err != nil {
		return true, errors.Wrap(err, "GetServerMemory")
	}

	if time.Since(pollStartTime) > longPollThreshold {
		return true, errors.Wrap(longPollError, "getservermemory")
	}

	err = s.PollOS()
	if err != nil {
		return true, errors.Wrap(err, "pollos")
	}

	// keep going if there is an error
	err = s.PollContainer()
	if err != nil {
		WinLogln(errors.Wrap(err, "pollcontainer"))
	}

	// err = s.GetCPUUsage()
	// if err != nil {
	// 	return true, errors.Wrap(err, "GetCpuUsage")
	// }

	err = s.GetCPU2()
	if err != nil {
		return true, errors.Wrap(err, "getcpu2")
	}

	if time.Since(pollStartTime) > longPollThreshold {
		return true, errors.Wrap(longPollError, "getcpu")
	}

	err = s.GetMetric(
		"sql",
		"SELECT [cntr_value] FROM sys.dm_os_performance_counters WHERE [counter_name] = 'Batch Requests/sec'",
		true)
	if err != nil {
		return true, errors.Wrap(err, "get sql/second")
	}

	if time.Since(pollStartTime) > longPollThreshold {
		return true, errors.Wrap(longPollError, "sqlbatches")
	}

	err = s.GetMetric(
		"bytesread",
		"select SUM(num_of_bytes_read) from sys.dm_io_virtual_file_stats(NULL, NULL)",
		true)
	if err != nil {
		return true, errors.Wrap(err, "bytesread")
	}

	if time.Since(pollStartTime) > longPollThreshold {
		return true, errors.Wrap(longPollError, "bytesread")
	}

	err = s.GetMetric(
		"byteswritten",
		"select SUM(num_of_bytes_written) from sys.dm_io_virtual_file_stats(NULL, NULL)",
		true)
	if err != nil {
		return true, errors.Wrap(err, "byteswritten")
	}

	if time.Since(pollStartTime) > longPollThreshold {
		return true, errors.Wrap(longPollError, "byteswritten")
	}

	err = s.GetMetric(
		"ple",
		"SELECT [cntr_value] FROM sys.dm_os_performance_counters WHERE [counter_name] = 'Page life expectancy' and instance_name = ''",
		false)
	if err != nil {
		return true, errors.Wrap(err, "ple")
	}

	val, err := s.GetLastMetric("ple")
	if err == nil {
		s.Lock()
		s.PLE = val.Value
		s.Unlock()
	}

	if time.Since(pollStartTime) > longPollThreshold {
		return true, errors.Wrap(longPollError, "ple")
	}

	err = s.PollWaits()
	if err != nil {
		return true, errors.Wrap(err, "pollwaits")
	}

	if time.Since(pollStartTime) > longPollThreshold {
		return true, errors.Wrap(longPollError, "pollwaits")
	}

	// // if err = s.pollWaits2(); err != nil {
	// // 	return errors.Wrap(err, "pollWaits2")
	// // }

	// if time.Since(pollStartTime) > longPollThreshold {
	// 	return true, errors.Wrap(longPollErrorlongPollThreshold, "pollwaits2")
	// }

	// Get Disk IO
	if err = s.getDiskIO(); err != nil {
		return true, errors.Wrap(err, "getDiskIO")
	}

	if time.Since(pollStartTime) > longPollThreshold {
		return true, errors.Wrap(longPollError, "getdiskio")
	}

	// Poll on the third time and every fifth time through
	// This gets the AG backups much quicker
	s.RLock()
	lastBackupPoll := s.LastBackupPoll
	s.RUnlock()

	// poll backups every five minutes
	if time.Since(lastBackupPoll) > 5*time.Minute {
		if err = s.pollBackups(); err != nil {
			return true, errors.Wrap(err, "pollBackups")
		}
	}

	// Get AGs and databases
	if s.MajorVersion >= 12 {
		if err = s.pollAG(); err != nil {
			return false, errors.Wrap(err, "pollAG")
		}
	}

	err = s.getDatabases()
	if err != nil {
		return true, errors.Wrap(err, "getDatabases")
	}

	if time.Since(pollStartTime) > longPollThreshold {
		return true, errors.Wrap(longPollError, "getdatabases")
	}

	err = s.getSnapshots()
	if err != nil {
		return true, errors.Wrap(err, "getsnapshots")
	}

	if time.Since(pollStartTime) > longPollThreshold {
		return true, errors.Wrap(longPollError, "getsnapshots")
	}

	// Installed
	err = s.getInstallDate()
	if err != nil {
		return true, errors.Wrap(err, "getinstalldate")
	}

	// IP Addresses
	if majorVersion > 10 {
		err = s.getIP()
		if err != nil {
			logonce.Error(err.Error())
		}
	}

	// Get the running jobs and recent failed jobs
	running, err := agent.FetchRunningJobs(context.TODO(), s.MapKey, s.DB)
	if err != nil {
		return true, err
	}
	s.Lock()
	s.RunningJobs = running
	s.Unlock()

	failed, err := agent.FetchRecentFailures(context.TODO(), s.MapKey, s.DB)
	if err != nil {
		return true, err
	}
	s.Lock()
	s.FailedJobs = failed
	s.Unlock()

	s.Lock()
	s.SortPriority = thisSortPriority
	s.Unlock()

	s.WriteToRepository()
	return true, nil
}

func (s *SqlServerWrapper) WriteToRepository() {
	var err error
	mm := make(map[string]any)

	s.RLock()
	mm["server_key"] = s.MapKey
	mm["server_name"] = s.ServerName
	mm["cpu_cores"] = s.CpuCount
	mm["cpu_sql_pct"] = s.LastSQLCPU
	mm["cpu_other_pct"] = s.LastCpu - s.LastSQLCPU
	metric, ok := s.Metrics["sql"]
	if ok {
		mm["batches_per_second"] = metric.V2.GetLastValue().ValuePerSecond
	} else {
		mm["batches_per_second"] = 0
	}
	mm["page_life_expectancy"] = s.PLE
	mm["memory_used_mb"] = s.SqlServerMemoryKB / 1024
	delta := s.DiskIODelta

	// get the requestWaits
	requestWaits := s.WaitBox.Repository().Last(s.MapKey) // waitring.Waitlist
	serverWaits := s.LastWaits                            // waitmap.Waits
	s.RUnlock()

	// set the per second values
	seconds := float64(delta.SampleMS) / 1000.0
	mm["disk_read_iops"] = int64(float64(delta.Reads) / seconds)
	mm["disk_write_iops"] = int64(float64(delta.Writes) / seconds)
	if seconds > 0 {
		mm["disk_read_kb_sec"] = int64((float64(delta.ReadBytes) / 1024.0) / seconds)
		mm["disk_write_kb_sec"] = int64((float64(delta.WriteBytes) / 1024.0) / seconds)
	}
	if delta.Reads > 0 {
		mm["disk_read_latency_ms"] = delta.ReadStall / delta.Reads //delta.ReadBytes / delta.Reads
	} else {
		mm["disk_read_latency_ms"] = 0
	}
	if delta.Writes > 0 {
		mm["disk_write_latency_ms"] = delta.WriteStall / delta.Writes
	} else {
		mm["disk_write_latency_ms"] = 0
	}

	ts := time.Now()
	err = mrepo.WriteMetrics(ts, mm)
	if err != nil {
		// Log the error but don't return it
		logrus.Error(errors.Wrap(err, "mrepo.write"))
	}

	err = mrepo.WriteWaits(s.MapKey, s.ServerName, "request_wait", requestWaits)
	if err != nil {
		// Log the error but don't return it
		logrus.Error(errors.Wrap(err, "mrepo.writewaits.request"))
	}

	// Convert serverWaits to waitring.WaitList
	// so we can write it to the repository
	sw := waitring.WaitList{
		TS:    serverWaits.EventTime,
		Waits: serverWaits.WaitSummary,
	}
	err = mrepo.WriteWaits(s.MapKey, s.ServerName, "server_wait", sw)
	if err != nil {
		// Log the error but don't return it
		logrus.Error(errors.Wrap(err, "mrepo.writewaits.server"))
	}

}

func (sw *SqlServerWrapper) getIP() error {
	// TODO: parse and lookup the FQDN to get an IP address and port
	// Because containers won't know their IP address
	// TODO: only get unique values for "result" below
	type listener struct {
		ServerName string `db:"server_name"`
		IPAddress  string `db:"ip_address"`
		Port       uint16 `db:"port"`
		Type       string `db:"type_desc"`
	}

	dbQuery := `
		IF CAST(PARSENAME(CAST(SERVERPROPERTY('productversion') AS varchar(20)), 4) AS INT) <= 10
			RETURN;
		SELECT	DISTINCT @@SERVERNAME AS [server_name], ip_address, port, type_desc
		FROM	sys.dm_tcp_listener_states
		WHERE	ip_address NOT IN ('::', '0.0.0.0', '::1'/*, '127.0.0.1'*/)
		AND     type_desc NOT IN ('Database Mirroring')
		UNION 
		SELECT	DISTINCT @@SERVERNAME AS [Server], local_net_address, local_tcp_port, protocol_type
		FROM	sys.dm_exec_connections  
		WHERE	local_net_address IS NOT NULL
		AND     local_net_address NOT IN ('::1', '127.0.0.1')
		AND     protocol_type NOT IN ('Database Mirroring')
	`

	allips := make([]netip.AddrPort, 0)
	rows, err := sw.DB.Query(dbQuery)
	if err != nil {
		return errors.Wrap(err, "query")
	}
	defer rows.Close()
	for rows.Next() {
		l := listener{}
		err = rows.Scan(&l.ServerName, &l.IPAddress, &l.Port, &l.Type)
		if err != nil {
			return errors.Wrap(err, "rows.scan")
		}
		addr, err := netip.ParseAddr(l.IPAddress)
		if err != nil {
			return errors.Wrap(err, "netip.parseaddr")
		}
		addrp := netip.AddrPortFrom(addr, l.Port)
		allips = append(allips, addrp)
	}
	err = rows.Err()
	if err != nil {
		return errors.Wrap(err, "rows.err")
	}

	// Lookup the FQDN
	fqdnIPs, err := sw.lookupIPs()
	if err != nil {
		sw.RLock()
		fqdn, key := sw.FQDN, sw.MapKey
		logrus.Error(errors.Wrapf(err, "lookupip: %s: %s", key, fqdn))
	}
	allips = append(allips, fqdnIPs...)

	// sort based on full ip then port
	sort.SliceStable(allips, func(i, j int) bool {
		if allips[i].Addr().Less(allips[j].Addr()) {
			return true
		}
		if allips[i].Port() < allips[j].Port() {
			return true
		}
		return false
	})

	// then string with some ports removed, then unique based on the string
	uniq := make(map[string]bool)
	result := make([]string, 0, len(allips))
	for _, addr := range allips {
		var str string
		if addr.Port() == 1433 || addr.Port() == 0 {
			str = addr.Addr().String()
		} else {
			if addr.Addr().Is4() {
				str = fmt.Sprintf("%s:%d", addr.Addr().String(), addr.Port())
			} else {
				str = fmt.Sprintf("[%s]:%d", addr.Addr().String(), addr.Port())
			}
		}
		if !uniq[str] {
			uniq[str] = true
			result = append(result, str)
		}
	}
	sw.Lock()
	defer sw.Unlock()
	sw.IPAdresses = result
	return nil
}

func (sw *SqlServerWrapper) lookupIPs() ([]netip.AddrPort, error) {
	sw.RLock()
	fqdn := sw.FQDN
	sw.RUnlock()

	// We are doing this to get the IP address for servers in containers
	// If this is a named instance, we aren't going to get the right
	// IP port combination.  So let's skip those.
	if strings.Contains(fqdn, "\\") {
		return []netip.AddrPort{}, nil
	}

	// split name and port
	parts := strings.Split(fqdn, ",")
	host := parts[0]
	var port uint16
	if len(parts) > 1 {
		iPort, err := strconv.Atoi(parts[1])
		if err != nil {
			return []netip.AddrPort{}, errors.Wrap(err, "strconv.atoi")
		}
		if iPort >= 0 && iPort <= 65535 {
			port = uint16(iPort)
		}
	}

	// if not port, assume 1433
	if port == 0 {
		port = 1433
	}

	// if host is an IP address, just return it
	ip, err := netip.ParseAddr(host)
	if err == nil { // we have a valid IP address
		return []netip.AddrPort{netip.AddrPortFrom(ip, port)}, nil
	}

	// lookup the host
	ips, err := net.LookupIP(host)
	if err != nil {
		return []netip.AddrPort{}, errors.Wrap(err, "net.lookupip")
	}

	// add all the IP addresses
	result := make([]netip.AddrPort, 0, len(ips))
	for _, ip := range ips {
		goodip, ok := netip.AddrFromSlice(ip)
		if ok {
			addrp := netip.AddrPortFrom(goodip, port)
			result = append(result, addrp)
		}
	}
	return result, nil
}

func (sw *SqlServerWrapper) getInstallDate() error {
	sw.RLock()
	db := sw.DB
	sw.RUnlock()

	row := db.QueryRow(`
		SELECT TOP 1 
		CAST(COALESCE(create_date, '0001-01-01') AS DATETIME) AS installed
		FROM sys.server_principals WITH (NOLOCK)
		WHERE name = N'NT AUTHORITY\SYSTEM'
		OR name = N'NT AUTHORITY\NETWORK SERVICE'
		ORDER BY installed 
	`)

	var t time.Time
	err := row.Scan(&t)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "row.scan")
	}
	// t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	// println(t.String())
	sw.Lock()
	sw.Installed = t
	sw.Unlock()
	return nil
}
