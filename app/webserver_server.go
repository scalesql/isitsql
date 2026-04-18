package app

import (
	"fmt"
	"html"
	"net/http"
	"slices"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/scalesql/isitsql/internal/docs"
	"github.com/scalesql/isitsql/internal/hadr"
	"github.com/sirupsen/logrus"
)

func ServerAboutPage(w http.ResponseWriter, req *http.Request) {
	var Page struct {
		Context
		Docs         []docs.Document
		NoDocsFolder bool
		Problems     []error
		Values       map[string]any
	}
	Page.Context.ServerPageActiveTab = "about"
	Page.Title = "About"
	Page.TagList = globalTagList.getTags()
	Page.ErrorList = getServerErrorList()
	globalConfig.RLock()
	Page.AppConfig = globalConfig.AppConfig
	globalConfig.RUnlock()
	m := make(map[string]any)
	key := req.PathValue("server")
	s, ok := servers.CloneOne(key)
	if !ok {
		renderErrorPage("Invalid Server", fmt.Sprintf("Server Not Found: %s", key), w)
		return
	}

	Page.OneServer = &s
	Page.Title = s.ServerName + " About"

	m["Version: Build"] = s.ProductVersion
	m["Version: Edition"] = s.ProductEdition
	m["Installed"] = s.Installed.Format("2006-01-02")
	m["OS: Name"] = fmt.Sprintf("%s (%s)", s.OSName, s.OSArch)
	m["OS: IPs"] = s.IP2HTMLList()
	m["Version"] = fmt.Sprintf("%s %s %s", s.VersionString, s.ProductLevel, s.ProductUpdateLevel)
	m["IsItSQL: FQDN"] = s.FQDN
	m["IsItSQL: Tags"] = s.TagString()
	m["Cores"] = fmt.Sprintf("%d", s.CpuCount)
	m["Memory: Used"] = KBToString(s.SqlServerMemoryKB)
	if s.MaxMemorySet() {
		m["Memory: Max"] = KBToString(s.MaxMemoryKB)
	}
	m["Memory: Physical"] = KBToString(s.PhysicalMemoryKB)
	// if s.CredentialKey != "" {
	// 	m["IsItSQL: Credential"] = s.CredentialKey
	// }

	var agnames []string
	var err error
	db, ok := servers.GetDB(key)
	if ok {
		agnames, err = hadr.GetNames(db)
		if err != nil {
			logrus.Error(err, "hadr.getnames")
		}
	}

	start := time.Now()
	names := []string{s.DisplayName(), s.FQDN, s.ServerName}
	names = append(names, agnames...)
	dd, problems, err := docs.Get(s.Domain, names)
	if errors.Is(err, docs.ErrNoDocsFolder) {
		Page.NoDocsFolder = true
	} else {
		if err != nil {
			logrus.Error(errors.Wrap(err, "serverdocspage"))
		}
	}
	dur := time.Since(start)
	dur = dur.Round(time.Millisecond)
	if dur > 1*time.Second {
		logrus.Infof("docs: generate: %s", dur)
	}
	Page.Docs = dd
	Page.Problems = problems
	Page.Values = m
	renderFSDynamic(w, "server-about", Page)
}

func ErrorLogPage(w http.ResponseWriter, req *http.Request) {

	type ErrorLogRow struct {
		RowNumber int
		LogDate   time.Time `db:"LogDate"`
		Process   string    `db:"ProcessInfo"`
		Text      string    `db:"Text"`
	}

	var pageData struct {
		Context
		Rows []ErrorLogRow
	}

	pageData.Context = getContext("Server Not Found")
	pageData.Context.ServerPageActiveTab = "errorlog"

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

	// read the error log
	dbx := sqlx.NewDb(wr.DB, "mssql")
	stmt := "EXECUTE sp_readerrorlog;"
	rows := make([]ErrorLogRow, 100)
	err := dbx.Select(&rows, stmt)
	if err != nil {
		//sessions = make([]*ActiveSession, 0)
		WinLogln(fmt.Sprintf("Server Error Page: %v", err))
		pageData.Context.Message = fmt.Sprintf("Server Error Page: %v", err)
		pageData.Context.MessageClass = "alert-danger"
	}
	for i := 0; i < len(rows); i++ {
		rows[i].RowNumber = i + 1
	}
	slices.Reverse(rows)

	pageData.Rows = rows

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
		htmlTitle += "-Error Log-Is It SQL"
	} else {
		htmlTitle = "Is It Sql"
	}

	pageData.Title = htmlTitle

	renderFSDynamic(w, "server-errorlog", pageData)
}
