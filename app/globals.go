package app

import (
	"sync"
	"time"

	"github.com/scalesql/isitsql/internal/appringlog"
	"github.com/scalesql/isitsql/internal/dwaits"
	"github.com/scalesql/isitsql/internal/mrepo"
	//"github.com/scalesql/isitsql/internal/settings"
)

type appConfig struct {
	UseLocalStatic        bool
	EnableProfiler        bool
	EnableStatsviz        bool
	FullBackupAlertHours  int
	LogBackupAlertMinutes int
	HomePageURL           string
	AGAlertMB             int64
	AGWarnMB              int64
	Debug                 bool
	Trace                 bool
}

var globalConfig struct {
	AppConfig appConfig
	sync.RWMutex
}

func getGlobalConfig() appConfig {
	globalConfig.RLock()
	defer globalConfig.RUnlock()
	return globalConfig.AppConfig
}

type globalStatsType struct {
	sync.RWMutex
	StartTime   time.Time
	SessionGUID string
	ClientGUID  string
}

var globalStats globalStatsType

var globalTagList tagList

// var globalSlugMap SlugMap

func init() {
	globalTagList.Tags = make(map[string]tag)
	servers.Servers = make(map[string]*SqlServerWrapper)
	//globalSlugMap.m = make(map[string]string)
}

var servers ServerList

var GLOBAL_RINGLOG appringlog.RingLog

var IsInteractive = true

var buildGit = "undefined"
var buildDate = "undefined"

// var globalWaitsBucket bucket.BucketWriter

var DynamicWaitRepository *dwaits.Repository

// var buildTime = "undefined"

// Yet another global.  This is painful.
var GlobalRepository *mrepo.Repository
