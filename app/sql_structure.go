package app

import (
	"database/sql"
	"net"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/scalesql/isitsql/internal/cpuring"
	"github.com/scalesql/isitsql/internal/diskio"
	"github.com/scalesql/isitsql/internal/dwaits"
	"github.com/scalesql/isitsql/internal/metricvaluering"
	"github.com/scalesql/isitsql/internal/mssql"
	"github.com/scalesql/isitsql/internal/mssql/agent"
	"github.com/scalesql/isitsql/internal/waitmap"
)

type PollError struct {
	FriendlyName string
	InstanceName string
	Error        string
	ErrorRaw     string
	LastPollTime time.Time
}

// DisplayName returns the name to display for a SqlServer
func (s *SqlServer) DisplayName() string {
	// First non-empty of (FriendlyName, ServerName, FQDN, MapKey)
	if s.FriendlyName != "" {
		return s.FriendlyName
	}

	if s.ServerName != "" {
		return s.ServerName
	}

	if s.FQDN != "" {
		return s.FQDN
	}

	return s.MapKey
}

type SqlServerWrapper struct {
	sync.RWMutex
	SqlServer
	DB   *sql.DB `json:"-"`
	stop chan struct{}
}

func (wr *SqlServerWrapper) CloneSqlServer() SqlServer {
	wr.RLock()
	defer wr.RUnlock()
	return wr.SqlServer
}

// SqlServer holds the basic structure of a server we're polling
type SqlServer struct {

	// These fields are user entered
	MapKey           string   `json:"map_key,omitempty"`
	FriendlyName     string   `json:"friendly_name,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	FQDN             string   `json:"fqdn,omitempty"`
	ConnectionType   string   `json:"connection_type,omitempty"`
	ConnectionString string   `json:"-,omitempty"`
	CredentialKey    string   `json:"credential_key,omitempty"`
	SlugOverride     string   `json:"slug_override,omitempty"`
	Description      string   `json:"description,omitempty"`

	Stats sql.DBStats `json:"-"`
	//Connection SQLConnection `json:"-"`
	Connection struct {
		SessionID    int16
		NetTransport string
		AuthScheme   string
		LoginName    string
		Latency      time.Duration
	} `json:"-"`

	// ResetOnThisPoll will be set to true if the server name or start time changes.
	// This tells us to reset all the accumulated values
	ResetOnThisPoll bool `json:"reset_on_this_poll,omitempty"`

	// sync.RWMutex

	IsPolling     bool          `json:"is_polling,omitempty"`
	PollActivity  string        `json:"poll_activity,omitempty"`
	PollStart     time.Time     `json:"poll_start,omitempty"`
	PollDuration  time.Duration `json:"poll_duration,omitempty"`
	LastPollTime  time.Time     `json:"last_poll_time,omitempty"`
	LastBigPoll   time.Time     `json:"last_big_poll,omitempty"`
	LastPollError string        `json:"last_poll_error,omitempty"`
	LastPollFail  time.Time     `json:"last_poll_fail,omitempty"`
	PollCount     int           `json:"-"` // should be zero at startup

	// All the fields are populated by the system
	ServerName   string `json:"server_name,omitempty"` // ServerName holds @@SERVERNAME
	PhysicalName string `json:"physical_name,omitempty"`
	Domain       string `json:"domain,omitempty"`

	// RecentCpu         [METRIC_ARRAY_SIZE]Cpu
	StartTime         time.Time         `json:"start_time,omitempty"`
	CurrentTime       time.Time         `json:"current_time,omitempty"`
	CpuCount          int               `json:"cpu_count,omitempty"`
	PhysicalMemoryKB  int64             `json:"physical_memory_kb,omitempty"`
	AvailableMemoryKB int64             `json:"available_memory_kb,omitempty"`
	SqlServerMemoryKB int64             `json:"sql_server_memory_kb,omitempty"`
	MaxMemoryKB       int64             `json:"max_memory_kb,omitempty"`
	MemoryStateDesc   string            `json:"memory_state_desc,omitempty"`
	CPUUsage          cpuring.Ring      `json:"cpu_usage,omitempty"`
	LastCpu           int               `json:"last_cpu,omitempty"`
	LastSQLCPU        int               `json:"last_sqlcpu,omitempty"`
	CoresUsedSQL      float32           `json:"cores_used_sql,omitempty"`
	CoresUsedOther    float32           `json:"cores_used_other,omitempty"`
	Databases         map[int]*Database `json:"databases,omitempty"`
	Metrics           map[string]Metric `json:"metrics,omitempty"`
	Snapshots         []Snapshot        `json:"snapshots,omitempty"`

	ProductLevel       string    `json:"product_level,omitempty"`
	ProductUpdateLevel string    `json:"product_update_level,omitempty"` // CUn, starting with 2012...
	ProductVersion     string    `json:"product_version,omitempty"`      // ProductVersion holds 11.0.2345.1
	VersionString      string    `json:"version_string,omitempty"`       // Version string holds SQL Server 2016
	ProductEdition     string    `json:"product_edition,omitempty"`
	MajorVersion       int       `json:"major_version,omitempty"`
	EditionID          int64     `json:"edition_id,omitempty"`
	Installed          time.Time `json:"installed,omitempty"`
	PLE                int64     `json:"ple,omitempty"`
	SqlPerSecond       int64     `json:"sql_per_second,omitempty"`
	LastWaits          *waitmap.Waits

	SortPriority int `json:"sort_priority,omitempty"` // higher values end up higher in the list

	DatabaseCount        int    `json:"database_count,omitempty"`
	DataSizeKB           int64  `json:"data_size_kb,omitempty"`
	LogSizeKB            int64  `json:"log_size_kb,omitempty"`
	DatabaseStateSummary string `json:"database_state_summary,omitempty"`

	DiskIO      diskio.VirtualFileStats `json:"disk_io,omitempty"`
	DiskIODelta diskio.VirtualFileStats `json:"disk_io_delta,omitempty"`

	BackupRowCount    int                         `json:"backup_row_count,omitempty"`
	BackupMessage     string                      `json:"backup_message,omitempty"`
	Backups           map[string]*databaseBackups `json:"backups,omitempty"`
	LastBackupPoll    time.Time                   `json:"last_backup_poll,omitempty"`
	IgnoreBackups     bool                        `json:"ignore_backups,omitempty"`
	IgnoreBackupsList []string                    `json:"ignore_backups_list,omitempty"`

	OSName      string      `json:"os_name"`
	OSArch      string      `json:"os_arch"`
	InContainer bool        `json:"in_container"`
	IPAdresses  []string    `json:"ip_addresses"`
	WaitBox     *dwaits.Box // `json:"wait_box"`

	RunningJobs agent.JobList
	FailedJobs  []agent.JobHistoryRow
}

// TotalLine is used for totals on the various pages
type TotalLine struct {
	Count             int64
	MachineCount      int64
	DataSizeKB        int64
	LogSizeKB         int64
	DiskIO            diskio.VirtualFileStats
	SQLPerSecond      int64
	SQLServerMemoryKB int64
	PhysicalMemoryKB  int64
	MemoryCapKB       int64
	CPUCount          int
	Databases         int
	CoreUsageFactor   int
	CoresUsedSQL      float32
	CoresUsedOther    float32
}

type Cpu struct {
	EventTime time.Time
	SqlCpu    int
	OtherCpu  int
}

type Metric struct {
	//Pointer int // this points to the last written
	//Values       [METRIC_ARRAY_SIZE]MetricValue
	Accumulating bool                            `json:"accumulating,omitempty"`
	V2           metricvaluering.MetricValueRing `json:"v_2,omitempty"`
}

type ActiveSession struct {
	SessionSessionID  int
	RequestSessionID  int
	StartTime         time.Time
	RunTimeSeconds    int
	Status            string
	StatementText     string
	Database          string
	BlockingSessionID int
	WaitType          string
	WaitTime          int
	WaitResource      string
	HostName          string
	AppName           string
	LoginName         string
	RunTimeText       string
	PercentComplete   int
	Command           string
	OpenTxnCount      int
	HeadBlockerID     int
	TotalBlocked      int
	Depth             int
	BlockerID         int
}

// type ActiveJob struct {
// 	JobName       string
// 	StartTime     time.Time
// 	StepName      string
// 	StepStartTime time.Time
// 	StepNumber    int
// 	SystemTime    time.Time
// }

// SQLConnection holds information about the session
// that IsItSQL is using to connect to SQL Server
type SQLConnection struct {
	SessionID           int
	ConnectTime         time.Time
	NetTransport        string
	ProtocolType        string
	ProtocolVersion     int
	AuthScheme          string
	LoginTime           time.Time
	HostName            string
	ProgramName         string
	ClientVersion       int
	ClientInterfaceName string
	LoginName           string
	Now                 time.Time
}

// Semver returns the first three numbers of the SQL Server version
func (s *SqlServer) Semver() string {
	nums := strings.Split(s.ProductVersion, ".")
	if len(nums) > 3 {
		nums = nums[0:3]
	}
	return strings.Join(nums, ".")
}

func (s *SqlServer) TagString() string {
	if s.Tags == nil {
		return ""
	}
	return strings.Join(s.Tags, ", ")
}

// URL returns the absolute URL for the server in the form
// /server/map-key-guid
func (s SqlServer) URL() string {
	return path.Join("/", "server", strings.ToLower(s.MapKey))
}

func (s SqlServer) SlugURL() string {
	return path.Join("/", "s", s.Slug())
}

// Slug computes a URL based on the FQDN.
// It only uses the FQDN at this point
func (s SqlServer) Slug() string {
	if s.SlugOverride != "" {
		return s.SlugOverride
	}
	if s.FriendlyName != "" {
		return strings.ToLower(s.FriendlyName)
	}
	addr := net.ParseIP(s.FQDN)
	if addr != nil {
		// TODO use computer/instance
		return strings.ToLower(s.MapKey)
	}
	if s.FQDN == "" {
		return strings.ToLower(s.MapKey)
	}
	if strings.Contains(s.FQDN, ":") {
		return strings.ToLower(s.MapKey)
	}
	// TODO first dot separated string of FQDN and instance
	host, instance := mssql.SplitServerName(s.FQDN)
	host = strings.ToLower(host)
	instance = strings.ToLower(instance)
	// if host has dots
	if strings.Contains(host, ".") {
		parts := strings.Split(host, ".")
		host = parts[0]
	}
	return path.Join(host, instance)
}

func (srv SqlServer) IP2CSVString() string {
	//arr := make([]string, 0, 12)
	result := strings.Join(srv.IPAdresses, ", ")
	return result
}
