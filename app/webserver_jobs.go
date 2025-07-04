package app

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/scalesql/isitsql/internal/docs"
	"github.com/scalesql/isitsql/internal/mssql/agent"
	"github.com/pkg/errors"
)

func ServerJobsPage(w http.ResponseWriter, req *http.Request) {
	template := "server-jobs"
	var Page struct {
		Context
		Docs         []docs.Document
		NoDocsFolder bool
		Problems     []error
		Jobs         agent.JobList
		JobHistory   []agent.JobHistoryRow
	}
	Page.Context.ServerPageActiveTab = "all-jobs"
	Page.JobHistory = []agent.JobHistoryRow{}
	Page.Title = "Jobs"
	Page.TagList = globalTagList.getTags()
	Page.ErrorList = getServerErrorList()
	globalConfig.RLock()
	Page.AppConfig = globalConfig.AppConfig
	globalConfig.RUnlock()

	key := req.PathValue("server")
	s, ok := servers.CloneOne(key)
	if !ok {
		renderErrorPage("Invalid Server", fmt.Sprintf("Server Not Found: %s", key), w)
		return
	}

	Page.OneServer = &s
	Page.Title = s.ServerName + " Jobs-All"

	Page.Jobs = agent.JobList{}
	pool, err := servers.NewPool(key)
	if err != nil {
		WinLogln(errors.Wrap(err, "servers.newpool"))
		Page.Problems = []error{errors.Wrap(err, "servers.newpool")}
		renderFSDynamic(w, template, Page)
		return
	}
	defer pool.Close()

	jobs, err := agent.FetchJobs(context.TODO(), key, pool)
	if err != nil {
		WinLogln(errors.Wrap(err, "agent.jobs"))
		Page.Problems = []error{errors.Wrap(err, "agent.jobs")}
		renderFSDynamic(w, template, Page)
		return
	}
	// TODO assign jobs to server
	Page.Jobs = jobs
	renderFSDynamic(w, template, Page)
}

func ServerJobsActivePage(w http.ResponseWriter, req *http.Request) {
	template := "server-jobs"
	var Page struct {
		Context
		Docs         []docs.Document
		NoDocsFolder bool
		Problems     []error
		Jobs         agent.JobList
		JobHistory   []agent.JobHistoryRow
	}
	Page.JobHistory = []agent.JobHistoryRow{}
	Page.Context.ServerPageActiveTab = "active-jobs"
	Page.Title = "Jobs"
	Page.TagList = globalTagList.getTags()
	Page.ErrorList = getServerErrorList()
	globalConfig.RLock()
	Page.AppConfig = globalConfig.AppConfig
	globalConfig.RUnlock()

	key := req.PathValue("server")
	s, ok := servers.CloneOne(key)
	if !ok {
		renderErrorPage("Invalid Server", fmt.Sprintf("Server Not Found: %s", key), w)
		return
	}

	Page.OneServer = &s
	Page.Title = s.ServerName + " Jobs-Active"

	Page.Jobs = agent.JobList{}
	pool, err := servers.NewPool(key)
	if err != nil {
		WinLogln(errors.Wrap(err, "servers.newpool"))
		Page.Problems = []error{errors.Wrap(err, "servers.newpool")}
		renderFSDynamic(w, template, Page)
		return
	}
	defer pool.Close()

	jobs, err := agent.FetchRunningJobs(context.TODO(), key, pool)
	if err != nil {
		WinLogln(errors.Wrap(err, "agent.fetchrunningjobs"))
		Page.Problems = []error{errors.Wrap(err, "agent.fetchrunningjobs")}
		renderFSDynamic(w, template, Page)
		return
	}
	Page.Jobs = jobs

	history, err := agent.FetchRecentFailures(context.TODO(), key, pool)
	if err != nil {
		WinLogln(errors.Wrap(err, "agent.fetchrecentfailures"))
		Page.Problems = []error{errors.Wrap(err, "agent.fetchrecentfailures")}
		renderFSDynamic(w, template, Page)
		return
	}
	Page.JobHistory = history

	// TODO assign jobs and history to server
	renderFSDynamic(w, template, Page)
}

func ServerJobHistoryPage(w http.ResponseWriter, req *http.Request) {
	template := "server-job-history"
	var Page struct {
		Context
		Docs         []docs.Document
		NoDocsFolder bool
		Problems     []error
		Job          agent.Job
		JobHistory   []agent.JobHistoryRow
		JobCurrent   []agent.JobHistoryRow
		JobStepLogs  []agent.JobStepLog
	}
	Page.Context.ServerPageActiveTab = "jobs"
	Page.Title = "Jobs"
	Page.TagList = globalTagList.getTags()
	Page.ErrorList = getServerErrorList()
	globalConfig.RLock()
	Page.AppConfig = globalConfig.AppConfig
	globalConfig.RUnlock()

	key := req.PathValue("server")
	jobid := req.PathValue("jobid")
	s, ok := servers.CloneOne(key)
	if !ok {
		renderErrorPage("Invalid Server", fmt.Sprintf("Server Not Found: %s", key), w)
		return
	}

	Page.OneServer = &s
	//Page.Title = s.ServerName + "-"

	Page.Job = agent.Job{}
	pool, err := servers.NewPool(key)
	if err != nil {
		WinLogln(errors.Wrap(err, "servers.newpool"))
		Page.Problems = []error{errors.Wrap(err, "servers.newpool")}
		renderFSDynamic(w, template, Page)
		return
	}
	defer pool.Close()

	job, err := agent.FetchJob(context.TODO(), key, jobid, pool)
	if err != nil {
		WinLogln(errors.Wrap(err, "agent.fetchjob"))
		Page.Problems = []error{errors.Wrap(err, "agent.fetchjob")}
		renderFSDynamic(w, template, Page)
		return
	}
	// TODO assign jobs to server
	Page.Title = s.ServerName + "-" + job.Name
	Page.Job = job

	current, err := agent.FetchJobMessagesCurrent(context.TODO(), key, pool, jobid)
	if err != nil {
		WinLogln(errors.Wrap(err, "agent.fetchjobmessagescurrent"))
		Page.Problems = []error{errors.Wrap(err, "agent.fetchjobmessagescurrent")}
		renderFSDynamic(w, template, Page)
		return
	}
	Page.JobCurrent = current

	history, err := agent.FetchJobCompletions(context.TODO(), key, pool, jobid)
	if err != nil {
		WinLogln(errors.Wrap(err, "agent.fetchjobhistory"))
		Page.Problems = []error{errors.Wrap(err, "agent.fetchjobhistory")}
		renderFSDynamic(w, template, Page)
		return
	}
	Page.JobHistory = history

	steps, err := agent.FetchJobStepLog(key, jobid, pool)
	if err != nil {
		WinLogln(errors.Wrap(err, "agent.fetchjobsteplog"))
		Page.Problems = []error{errors.Wrap(err, "agent.fetchjobsteplog")}
		renderFSDynamic(w, template, Page)
		return
	}
	Page.JobStepLogs = steps
	renderFSDynamic(w, template, Page)
}

func ServerJobMessagesPage(w http.ResponseWriter, req *http.Request) {
	template := "server-job-messages"
	var Page struct {
		Context
		Docs         []docs.Document
		NoDocsFolder bool
		Problems     []error
		Job          agent.Job
		JobHistory   []agent.JobHistoryRow
	}
	Page.Context.ServerPageActiveTab = "jobs"
	Page.Title = "Jobs"
	Page.TagList = globalTagList.getTags()
	Page.ErrorList = getServerErrorList()
	globalConfig.RLock()
	Page.AppConfig = globalConfig.AppConfig
	globalConfig.RUnlock()

	key := req.PathValue("server")
	jobid := req.PathValue("jobid")
	strinstanceid := req.PathValue("instanceid")
	instanceid, err := strconv.Atoi(strinstanceid)
	if err != nil {
		renderErrorPage("Invalid instance_id", fmt.Sprintf("Invalid instance_id: %v", err), w)
		return
	}
	s, ok := servers.CloneOne(key)
	if !ok {
		renderErrorPage("Invalid Server", fmt.Sprintf("Server Not Found: %s", key), w)
		return
	}

	Page.OneServer = &s
	//Page.Title = s.ServerName + "-"

	Page.Job = agent.Job{}
	pool, err := servers.NewPool(key)
	if err != nil {
		WinLogln(errors.Wrap(err, "servers.newpool"))
		Page.Problems = []error{errors.Wrap(err, "servers.newpool")}
		renderFSDynamic(w, template, Page)
		return
	}
	defer pool.Close()

	job, err := agent.FetchJob(context.TODO(), key, jobid, pool)
	if err != nil {
		WinLogln(errors.Wrap(err, "agent.fetchjob"))
		Page.Problems = []error{errors.Wrap(err, "agent.fetchjob")}
		renderFSDynamic(w, template, Page)
		return
	}
	// TODO assign jobs to server
	Page.Title = s.ServerName + "-" + job.Name
	Page.Job = job

	history, err := agent.FetchJobMessages(context.TODO(), key, pool, jobid, instanceid)
	if err != nil {
		WinLogln(errors.Wrap(err, "agent.fetchjobmessages"))
		Page.Problems = []error{errors.Wrap(err, "agent.fetchjobmessages")}
		renderFSDynamic(w, template, Page)
		return
	}
	Page.JobHistory = history
	renderFSDynamic(w, template, Page)
}

func ServerJobStepLogPage(w http.ResponseWriter, req *http.Request) {
	template := "server-job-steplog"
	var Page struct {
		Context
		Docs         []docs.Document
		NoDocsFolder bool
		Problems     []error
		Job          agent.Job
		JobStepLogs  []agent.JobStepLog
	}
	Page.Context.ServerPageActiveTab = "jobs"
	Page.Title = "Job Step Log"
	Page.TagList = globalTagList.getTags()
	Page.ErrorList = getServerErrorList()
	globalConfig.RLock()
	Page.AppConfig = globalConfig.AppConfig
	globalConfig.RUnlock()

	key := req.PathValue("server")
	jobid := req.PathValue("jobid")

	s, ok := servers.CloneOne(key)
	if !ok {
		renderErrorPage("Invalid Server", fmt.Sprintf("Server Not Found: %s", key), w)
		return
	}

	Page.OneServer = &s
	//Page.Title = s.ServerName + "-"

	Page.Job = agent.Job{}
	pool, err := servers.NewPool(key)
	if err != nil {
		WinLogln(errors.Wrap(err, "servers.newpool"))
		Page.Problems = []error{errors.Wrap(err, "servers.newpool")}
		renderFSDynamic(w, template, Page)
		return
	}
	defer pool.Close()

	job, err := agent.FetchJob(context.TODO(), key, jobid, pool)
	if err != nil {
		WinLogln(errors.Wrap(err, "agent.fetchjob"))
		Page.Problems = []error{errors.Wrap(err, "agent.fetchjob")}
		renderFSDynamic(w, template, Page)
		return
	}
	// TODO assign jobs to server
	Page.Title = s.ServerName + "-" + job.Name
	Page.Job = job

	steps, err := agent.FetchJobStepLog(key, jobid, pool)
	if err != nil {
		WinLogln(errors.Wrap(err, "agent.fetchjobsteplog"))
		Page.Problems = []error{errors.Wrap(err, "agent.fetchjobsteplog")}
		renderFSDynamic(w, template, Page)
		return
	}
	Page.JobStepLogs = steps
	renderFSDynamic(w, template, Page)
}

func AgentJobsPage(w http.ResponseWriter, req *http.Request) {
	template := "jobs"
	var Page struct {
		Context
		Docs         []docs.Document
		NoDocsFolder bool
		Problems     []error
		RunningJobs  agent.JobList
		FailedJobs   []agent.JobHistoryRow
	}
	Page.Title = "Active and Failed Jobs"
	Page.TagList = globalTagList.getTags()
	Page.ErrorList = getServerErrorList()
	globalConfig.RLock()
	Page.AppConfig = globalConfig.AppConfig
	globalConfig.RUnlock()

	running := agent.JobList{}
	failed := make([]agent.JobHistoryRow, 0)
	ss := servers.CloneUnique()
	for _, s := range ss {
		running = append(running, s.RunningJobs...)
		failed = append(failed, s.FailedJobs...)
	}
	Page.RunningJobs = running
	Page.FailedJobs = failed

	renderFSDynamic(w, template, Page)
}
