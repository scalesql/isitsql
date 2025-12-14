package app

import (
	"fmt"
	"net/http"
	"time"

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
