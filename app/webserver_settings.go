package app

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/schema"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/scalesql/isitsql/internal/c2"
	"github.com/scalesql/isitsql/internal/gui"
	"github.com/scalesql/isitsql/internal/settings"
)

func settingsPage(w http.ResponseWriter, r *http.Request) {

	context := struct {
		Context
		Pollers          int
		Port             int
		SecurityPolicy   settings.SecurityPolicyType
		BackupHours      int
		LogBackupMinutes int
		HomePageURL      string
		AdminDomainGroup string
		// Profiling bool
	}{
		Context: Context{
			Title:           "Settings",
			UnixNow:         time.Now().Unix() * 1000,
			ErrorList:       getServerErrorList(),
			HeaderRight:     fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			TagList:         globalTagList.getTags(),
			AppConfig:       getGlobalConfig(),
			MenuTwoSelected: "settings",
			MessageClass:    gui.MessageClsssSuccess,
		},
	}

	var err error

	var port int
	var policy string
	//var enableSave bool
	var backupHours int
	var logMinutes int

	var s settings.AppConfig

	context.EnableSave, err = settings.CanSave(r)
	if err != nil {
		context.Message = fmt.Sprintf("can save error: %s", err.Error())
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}

	s, err = settings.ReadConfig()
	if err != nil {
		context.Message = fmt.Sprintf("readconfig error: %s", err.Error())
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}

	// if r.Method == "POST" && !enableSave {
	// 	context.Message = "Save not allowed"
	// 	context.MessageClass = gui.MessageClassDanger
	// 	goto RenderForm
	// }

	// Process a POST
	if r.Method == "POST" {
		if !context.EnableSave {
			msg := errors.New("settings: post but can't save")
			GLOBAL_RINGLOG.Enqueue(msg.Error())
			context.Message = msg.Error()
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		err = r.ParseForm()
		if err != nil {
			context.Message = fmt.Sprintf("parse form error: %s", err.Error())
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		// Port
		port, err = strconv.Atoi(r.PostFormValue("port"))
		if err != nil {
			context.Message = fmt.Sprintf("non-numeric port: %s", r.PostFormValue("port"))
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		if port < 1 || port > 65535 {
			context.Message = "port should be 1 to 65535"
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}
		s.Port = port

		homePageURL := strings.TrimSpace(r.PostFormValue("homePage"))
		if homePageURL == "" {
			homePageURL = "/"
		}
		s.HomePageURL = homePageURL

		// Backup alert
		backupHours, err = strconv.Atoi(r.PostFormValue("backupHours"))
		if err != nil {
			context.Message = fmt.Sprintf("non-numeric backupHours: %s", r.PostFormValue("backupHours"))
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		if backupHours < 0 {
			context.Message = "Please enter a positive number of hours"
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}
		s.BackupAlertHours = backupHours

		// Log alert
		logMinutes, err = strconv.Atoi(r.PostFormValue("logMinutes"))
		if err != nil {
			context.Message = fmt.Sprintf("non-numeric logMinutes: %s", r.PostFormValue("logMinutes"))
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		if logMinutes < 0 {
			context.Message = "Please enter a positive number of minutes"
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		s.LogBackupAlertMinutes = logMinutes

		// Security Policy
		policy = r.FormValue("securityPolicy")

		switch policy {
		case "open":
			s.SecurityPolicy = settings.OpenPolicy
		case "localhost":
			s.SecurityPolicy = settings.LocalHostPolicy
		default:
			context.Message = "Invalid security policy"
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		s.AdminDomainGroup = strings.TrimSpace(r.PostFormValue("adminGroup"))
		err = s.Save()
		if err == nil {
			context.Message = "Settings saved"
		} else {
			context.Message = err.Error()
		}
		globalConfig.Lock()
		globalConfig.AppConfig.HomePageURL = s.HomePageURL
		globalConfig.Unlock()
	}
RenderForm:

	//context.Pollers = s.PollWorkers
	context.Port = s.Port
	context.SecurityPolicy = s.SecurityPolicy
	context.BackupHours = s.BackupAlertHours
	context.LogBackupMinutes = s.LogBackupAlertMinutes
	context.HomePageURL = s.HomePageURL
	context.AdminDomainGroup = s.AdminDomainGroup

	renderFSDynamic(w, "settings", context)
}

func slugsPage(w http.ResponseWriter, r *http.Request) {

	type pageRow struct {
		Key                   string
		FQDN                  string
		FriendlyName          string
		Tags                  []string
		ConnectionDescription string
		CredentialKey         string
		Domain                string
		ServerName            string
		Version               string
		ProductEdition        string
		ProductVersion        string
		ProductVersionString  string
		ProductLevel          string
		LastPollTime          time.Time
		LastPollError         string
		LastPollErrorClean    string
		CSSTableClass         string
		LinkName              string
		URL                   string
		SlugURL               string
		SlugOverride          string
	}

	context := struct {
		Context
		//Servers     map[string]*settings.SQLServer
		Servers     []pageRow
		Credentials map[string]*settings.SQLCredential
	}{
		Context: Context{
			Title:       "Slugs",
			UnixNow:     time.Now().Unix() * 1000,
			ErrorList:   getServerErrorList(),
			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			TagList:     globalTagList.getTags(),
			AppConfig:   getGlobalConfig(),
		},
	}

	var err error
	a, err := settings.ReadConnections()
	if err != nil {
		context.Message = err.Error()
		context.MessageClass = gui.MessageClassDanger
		WinLogln(err.Error())
	}
	//context.Servers = a.SQLServers
	context.Credentials = a.SQLCredentials

	var rows []pageRow
	// Populate the pageRow details
	for k, c := range a.SQLServers {
		pr := pageRow{
			Key:          k,
			FQDN:         c.FQDN,
			FriendlyName: c.FriendlyName,
			Tags:         c.Tags,
			LinkName:     c.LinkName(),
		}

		switch {
		case c.TrustedConnection:
			pr.ConnectionDescription = "Trusted"
		case c.CredentialKey != "":
			cred, ok := a.SQLCredentials[c.CredentialKey]
			if ok {
				pr.ConnectionDescription = fmt.Sprintf("Cred: %s", cred.Name)
			} else {
				pr.ConnectionDescription = fmt.Sprintf("Cred: %s", "Unknown")
			}
		case c.Login != "":
			pr.ConnectionDescription = fmt.Sprintf("SQL Login: %s", c.Login)
		case c.CustomConnectionString != "":
			pr.ConnectionDescription = "Conn String"
		default:
			pr.ConnectionDescription = "Unknown"
		}

		s, ok := servers.CloneOne(k)
		if ok {
			pr.Domain = s.Domain
			pr.ServerName = s.ServerName
			//pr.ProductVersionString = s.ProductVersionString()
			pr.ProductVersion = s.ProductVersion
			pr.ProductVersionString = s.VersionString
			pr.ProductLevel = s.ProductLevel
			pr.ProductEdition = s.ProductEdition
			pr.LastPollTime = s.LastPollTime
			pr.LastPollError = s.LastPollError
			pr.LastPollErrorClean = s.LastPollErrorClean(45)
			pr.CSSTableClass = s.GetTableCssClass()
			pr.URL = s.URL()
			pr.SlugURL = s.SlugURL()
			pr.SlugOverride = s.SlugOverride
		}
		rows = append(rows, pr)
	}

	context.Servers = rows
	renderFSDynamic(w, "slugs", context)
}

func serverListPage(w http.ResponseWriter, r *http.Request) {

	type pageRow struct {
		Key                   string
		FQDN                  string
		FriendlyName          string
		Tags                  []string
		ConnectionDescription string
		CredentialKey         string
		Domain                string
		ServerName            string
		Version               string
		ProductEdition        string
		ProductVersion        string
		ProductVersionString  string
		ProductLevel          string
		LastPollTime          time.Time
		LastPollError         string
		LastPollErrorClean    string
		CSSTableClass         string
		LinkName              string
		URL                   string
		SlugURL               string
		SlugOverride          string
	}

	context := struct {
		Context
		//Servers     map[string]*settings.SQLServer
		Servers     []pageRow
		Credentials map[string]*settings.SQLCredential
		FileConfig  bool
	}{
		Context: Context{
			Title:       "SQL Servers",
			UnixNow:     time.Now().Unix() * 1000,
			ErrorList:   getServerErrorList(),
			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			TagList:     globalTagList.getTags(),
			AppConfig:   getGlobalConfig(),
		},
	}

	if AppConfigMode == ModeFile {
		context.FileConfig = true
		context.Message = "File Config Mode: Monitored servers are configured in the HCL files"
		context.MessageClass = gui.MessageClassDanger
		renderFSDynamic(w, "settings-server-list", context)
		return
	}

	var err error
	a, err := settings.ReadConnections()
	if err != nil {
		context.Message = err.Error()
		context.MessageClass = gui.MessageClassDanger
		WinLogln(err.Error())
	}
	//context.Servers = a.SQLServers
	context.Credentials = a.SQLCredentials

	var rows []pageRow
	// Populate the pageRow details
	for k, c := range a.SQLServers {
		pr := pageRow{
			Key:          k,
			FQDN:         c.FQDN,
			FriendlyName: c.FriendlyName,
			Tags:         c.Tags,
			LinkName:     c.LinkName(),
		}

		switch {
		case c.TrustedConnection:
			pr.ConnectionDescription = "Trusted"
		case c.CredentialKey != "":
			cred, ok := a.SQLCredentials[c.CredentialKey]
			if ok {
				pr.ConnectionDescription = fmt.Sprintf("Cred: %s", cred.Name)
			} else {
				pr.ConnectionDescription = fmt.Sprintf("Cred: %s", "Unknown")
			}
		case c.Login != "":
			pr.ConnectionDescription = fmt.Sprintf("SQL Login: %s", c.Login)
		case c.CustomConnectionString != "":
			pr.ConnectionDescription = "Conn String"
		default:
			pr.ConnectionDescription = "Unknown"
		}

		s, ok := servers.CloneOne(k)
		if ok {
			pr.Domain = s.Domain
			pr.ServerName = s.ServerName
			//pr.ProductVersionString = s.ProductVersionString()
			pr.ProductVersion = s.ProductVersion
			pr.ProductVersionString = s.VersionString
			pr.ProductLevel = s.ProductLevel
			pr.ProductEdition = s.ProductEdition
			pr.LastPollTime = s.LastPollTime
			pr.LastPollError = s.LastPollError
			pr.LastPollErrorClean = s.LastPollErrorClean(45)
			pr.CSSTableClass = s.GetTableCssClass()
			pr.URL = s.URL()
			pr.SlugURL = s.SlugURL()
			pr.SlugOverride = s.SlugOverride
		}
		rows = append(rows, pr)
	}

	context.Servers = rows
	renderFSDynamic(w, "settings-server-list", context)
}

func connListPage(w http.ResponseWriter, r *http.Request) {
	context := struct {
		Context
		Credentials map[string]*settings.SQLCredential
		FileConfig  bool
		Messages    []string
		Maps        c2.ConfigMaps
	}{
		Context: Context{
			Title:       "Connections",
			UnixNow:     time.Now().Unix() * 1000,
			ErrorList:   getServerErrorList(),
			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			TagList:     globalTagList.getTags(),
			AppConfig:   getGlobalConfig(),
		},
	}

	if AppConfigMode == ModeGUI {
		context.FileConfig = true
		context.Message = "GUI Config Mode: Monitored servers are configured in the web site"
		context.MessageClass = gui.MessageClassDanger
		renderFSDynamic(w, "settings-conn-list", context)
		return
	}

	fc, msgs, err := c2.GetHCLFiles()
	if err != nil {
		context.Message = err.Error()
		context.MessageClass = gui.MessageClassDanger
	}
	context.Maps = fc
	context.Messages = msgs
	// context.Messages = []string{"Message 1", "Message 2"}

	renderFSDynamic(w, "settings-conn-list", context)
}

func credentialListPage(w http.ResponseWriter, r *http.Request) {

	context := struct {
		Context
		Credentials map[string]*settings.SQLCredential
	}{
		Context: Context{
			Title:           "Credentials",
			UnixNow:         time.Now().Unix() * 1000,
			ErrorList:       getServerErrorList(),
			HeaderRight:     fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			TagList:         globalTagList.getTags(),
			AppConfig:       getGlobalConfig(),
			MenuTwoSelected: "credentials",
		},
	}

	var err error
	a, err := settings.ReadConnections()
	if err != nil {
		context.Message = err.Error()
		context.MessageClass = gui.MessageClassDanger
		WinLogln(err.Error())
	} else {
		context.Credentials = a.SQLCredentials
	}

	renderFSDynamic(w, "settings-credentials-list", context)
}

func credentialAddPage(w http.ResponseWriter, r *http.Request) {

	var err error

	context := struct {
		Context
	}{
		Context: Context{
			Title:           "Credentials - Add",
			UnixNow:         time.Now().Unix() * 1000,
			ErrorList:       getServerErrorList(),
			HeaderRight:     fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			TagList:         globalTagList.getTags(),
			AppConfig:       getGlobalConfig(),
			MenuTwoSelected: "credentials",
		},
	}

	canSave, err := settings.CanSave(r)
	if err != nil {
		context.Message = fmt.Sprintf("can save error: %s", err.Error())
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}

	context.EnableSave = canSave

	if r.Method == "POST" {

		if !context.EnableSave {
			msg := errors.New("credential: post but can't save")
			GLOBAL_RINGLOG.Enqueue(msg.Error())
			context.Message = msg.Error()
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		err = r.ParseForm()
		if err != nil {
			msg := errors.Wrap(err, "error parsing form")
			GLOBAL_RINGLOG.Enqueue(msg.Error())
			context.Message = msg.Error()
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		a, err := settings.ReadConnections()
		if err != nil {
			msg := errors.Wrap(err, "readConnections")
			GLOBAL_RINGLOG.Enqueue(msg.Error())
			context.Message = msg.Error()
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		credentialName := r.PostFormValue("credentialName")
		login := r.PostFormValue("login")
		password := r.PostFormValue("password")

		// check lengths
		if len(credentialName) == 0 || len(login) == 0 || len(password) == 0 {
			context.Message = "All fields must have a value"
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		// save it
		// I'm not sure why I was reading it twice
		//a, err := settings.ReadConnections()

		// check if this name exists
		for _, v := range a.SQLCredentials {
			if strings.EqualFold(v.Name, credentialName) {
				context.Message = fmt.Sprintf("A credential named '%s' already exists", credentialName)
				context.MessageClass = gui.MessageClassDanger
				goto RenderForm

			}
		}

		// c := settings.SQLCredential{
		// 	Name:     credentialName,
		// 	Login:    login,
		// 	Password: password,
		// }

		_, err = settings.AddSQLCredential(credentialName, login, password)
		if err != nil {
			context.Message = fmt.Sprintf("error saving settings: %s", err.Error())
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		// redirect to list
		http.Redirect(w, r, "/settings/credentials", http.StatusSeeOther)

	}

	// var err error
	// a, err := settings.ReadConnections()
	// if err != nil {
	// 	context.Message = err.Error()
	// 	context.MessageClass = gui.MessageClassDanger
	// 	WinLogln(err.Error())
	// }

RenderForm:

	renderFSDynamic(w, "settings-credentials-add", context)
}

func serverEditPage(w http.ResponseWriter, r *http.Request) {

	var err error

	var s *settings.SQLServer
	var a settings.Connections
	//var auth string
	var canSaveFlag bool
	//var ok bool

	fv := new(gui.ServerPage)

	context := struct {
		Context
		//SQLServer   *settings.SQLServer
		Credentials map[string]*settings.SQLCredential
		FormValues  *gui.ServerPage
		ServerKey   string
	}{
		Context: Context{
			Title:       "SQL Server - Edit",
			UnixNow:     time.Now().Unix() * 1000,
			ErrorList:   getServerErrorList(),
			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			TagList:     globalTagList.getTags(),
			AppConfig:   getGlobalConfig(),
		},
		FormValues: &gui.ServerPage{}, // empty so we don't end up with nil later
	}

	serverKey := r.PathValue("server")
	context.ServerKey = serverKey
	// TODO remove to handle non-UUID keys
	serverKeyUUID, err := uuid.FromString(serverKey)
	if err != nil {
		context.Message = "server key isn't uuid"
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}

	context.EnableSave, err = settings.CanSave(r)
	if err != nil {
		context.Message = fmt.Sprintf("can save error: %s", err.Error())
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}

	// Populate the credentials
	a, err = settings.ReadConnections()
	if err != nil {
		context.Message = fmt.Sprintln("error reading connection: ", err.Error())
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}
	context.Credentials = a.SQLCredentials

	s, err = settings.GetSQLServer(serverKey)
	if err == settings.ErrNotFound {
		context.Message = fmt.Sprintf("Server '%s' doesn't exist", serverKey)
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}
	if err != nil {
		context.Message = fmt.Sprintf("error getting server: %s", err.Error())
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}

	canSaveFlag, err = settings.CanSave(r)
	if err != nil {
		context.Message = fmt.Sprintf("can save error: %s", err.Error())
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}
	context.EnableSave = canSaveFlag

	if AppConfigMode == ModeFile {
		context.Message = "File Config Mode: Please edit servers in the '.hcl' files."
		context.MessageClass = gui.MessageClassWarning
		context.EnableSave = false
	}

	// Set the form values
	fv.FQDN = s.FQDN
	fv.FriendlyName = s.FriendlyName
	//fmt.Println("Reading Tags: ", len(s.Tags), s.Tags)
	fv.Tags = strings.Join(s.Tags, ", ")
	fv.CredentialKey = s.CredentialKey
	fv.Login = s.Login
	fv.Password = s.Password
	fv.ConnectionString = s.CustomConnectionString
	fv.Connection, err = getAuth(s)
	if err != nil {
		context.Message = "Invalid connection configuration"
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}
	context.FormValues = fv

	// Hide the connection string if we can't save
	if r.Method == "GET" && !canSaveFlag && len(s.CustomConnectionString) > 0 {
		fv.ConnectionString = ""
		context.Message = "The connection string is hidden"
		context.MessageClass = gui.MessageClassWarning
	}

	// Hide the password if we can't save
	if r.Method == "GET" && !canSaveFlag && len(s.Password) > 0 {
		fv.ConnectionString = ""
		context.Message = "The password is hidden"
		context.MessageClass = gui.MessageClassWarning
	}

	if r.Method == "POST" && !canSaveFlag {
		context.Message = "Save not allowed"
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}

	if r.Method == "POST" {

		if !context.EnableSave {
			msg := errors.New("server: post but can't save")
			GLOBAL_RINGLOG.Enqueue(msg.Error())
			context.Message = msg.Error()
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		//s = new(settings.SQLServer)
		//s.ServerKey = serverKeyUUID

		err = r.ParseForm()
		if err != nil {
			context.Message = fmt.Sprintf("parse form error: %s", err.Error())
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		p2 := new(gui.ServerPage)
		decoder := schema.NewDecoder()
		err = decoder.Decode(p2, r.PostForm)
		if err != nil {
			context.Message = fmt.Sprintf("parse decode error: %s", err.Error())
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}
		context.FormValues = p2

		//fmt.Println("crednetial key: ", p2.CredentialKey)

		err = p2.Validate()
		//fmt.Println("validate: ", err)
		if err != nil {
			context.Message = err.Error()
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		// Check for duplicates Friendly Names
		if p2.FriendlyName != "" {
			for _, v := range a.SQLServers {
				if strings.EqualFold(v.FriendlyName, p2.FriendlyName) && v.ServerKey != serverKeyUUID.String() {
					// fmt.Println("v: ", v.ServerKey, "s: ", s.ServerKey)
					context.Message = fmt.Sprintf("A SQL Server named '%s' already exists", p2.FriendlyName)
					context.MessageClass = gui.MessageClassDanger
					goto RenderForm
				}
			}
		}

		// Setup for the save
		s.FQDN = p2.FQDN
		s.FriendlyName = p2.FriendlyName

		// Populate the tags
		tags := strings.Split(p2.Tags, ",")
		//fmt.Println("->", p2.Tags, "<-", tags)
		s.Tags = make([]string, 0)
		for _, v := range tags {
			v = strings.TrimSpace(v)
			v = strings.Replace(v, " ", "-", -1)
			if v != "" {
				s.Tags = append(s.Tags, v)
			}
		}
		//fmt.Println(s.Tags)

		switch p2.Connection {
		case settings.AuthTrusted:
			s.TrustedConnection = true
			s.CredentialKey = ""
			s.Login = ""
			s.Password = ""
			s.CustomConnectionString = ""

		case settings.AuthCredential:
			s.TrustedConnection = false
			s.CredentialKey = p2.CredentialKey
			s.Login = ""
			s.Password = ""
			s.CustomConnectionString = ""

		case settings.AuthUserPassword:
			s.TrustedConnection = false
			s.CredentialKey = ""
			s.Login = p2.Login
			if p2.Password != "" {
				s.Password = p2.Password
			}
			s.CustomConnectionString = ""

		case settings.AuthCustom:
			s.TrustedConnection = false
			s.CredentialKey = ""
			s.Login = ""
			s.Password = ""
			s.CustomConnectionString = p2.ConnectionString

		default:
			context.Message = "Invalid authentication (server edit page)"
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		// Keep a copy before I save it.
		// Save encrypts and I don't want that
		x := *s

		err = settings.SaveSQLServer(serverKey, s)
		if err != nil {
			context.Message = fmt.Sprintf("save error: %s", err.Error())
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		err = servers.UpdateFromSettings(serverKey, x)
		if err != nil {
			context.Message = fmt.Sprintf("error updating to polling: %s", err.Error())
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		// Pause to it can try to poll
		time.Sleep(400 * time.Millisecond)

		//redirect to list
		http.Redirect(w, r, "/settings/servers", http.StatusSeeOther)
	}

RenderForm:

	renderFSDynamic(w, "settings-server-edit", context)
}

func serverDeletePage(w http.ResponseWriter, r *http.Request) {

	var err error

	var s *settings.SQLServer
	var a settings.Connections

	fv := new(gui.ServerPage)

	context := struct {
		Context
		//SQLServer   *settings.SQLServer
		Credentials map[string]*settings.SQLCredential
		FormValues  *gui.ServerPage
		ServerKey   string
	}{
		Context: Context{
			Title:       "SQL Server - Delete",
			UnixNow:     time.Now().Unix() * 1000,
			ErrorList:   getServerErrorList(),
			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			TagList:     globalTagList.getTags(),
			AppConfig:   getGlobalConfig(),
		},
	}

	serverKey := r.PathValue("server")
	context.ServerKey = serverKey
	// TODO remove to handle non-UUID keys
	_, err = uuid.FromString(serverKey)
	if err != nil {
		context.Message = "server key isn't uuid"
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}

	context.EnableSave, err = settings.CanSave(r)
	if err != nil {
		context.Message = fmt.Sprintf("can save error: %s", err.Error())
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}

	if AppConfigMode == ModeFile {
		context.Message = "File Config Mode: Please delete servers in the '.hcl' files."
		context.MessageClass = gui.MessageClassWarning
		context.EnableSave = false
	}

	// Populate the credentials
	a, err = settings.ReadConnections()
	if err != nil {
		context.Message = fmt.Sprintln("error reading connection: ", err.Error())
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}
	context.Credentials = a.SQLCredentials

	s, err = settings.GetSQLServer(serverKey)
	if err == settings.ErrNotFound {
		context.Message = fmt.Sprintf("Server '%s' doesn't exist", serverKey)
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}

	if err != nil {
		context.Message = fmt.Sprintf("error getting server: %s", err.Error())
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}

	// Set the form values
	fv.FQDN = s.FQDN
	fv.FriendlyName = s.FriendlyName
	//fmt.Println("Reading Tags: ", len(s.Tags), s.Tags)
	fv.Tags = strings.Join(s.Tags, ", ")
	fv.CredentialKey = s.CredentialKey
	fv.Login = s.Login
	fv.Password = s.Password
	fv.ConnectionString = s.CustomConnectionString
	fv.Connection, err = getAuth(s)
	if err != nil {
		context.Message = "Invalid connection configuration"
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}
	context.FormValues = fv

	// Hide the connection string if we can't save
	if r.Method == "GET" && !context.EnableSave && len(s.CustomConnectionString) > 0 {
		fv.ConnectionString = ""
		context.Message = "The connection string is hidden"
		context.MessageClass = gui.MessageClassWarning
	}

	// Hide the password if we can't save
	if r.Method == "GET" && !context.EnableSave && len(s.Password) > 0 {
		fv.ConnectionString = ""
		context.Message = "The password is hidden"
		context.MessageClass = gui.MessageClassWarning
	}

	if r.Method == "POST" && !context.EnableSave {
		context.Message = "Save not allowed"
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}

	if r.Method == "POST" {

		if !context.EnableSave {
			msg := errors.New("server (delete): post but can't save")
			GLOBAL_RINGLOG.Enqueue(msg.Error())
			context.Message = msg.Error()
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		// delete the credential
		err = settings.DeleteSQLServer(serverKey)
		if err != nil {
			context.Message = fmt.Sprintf("Delete error: %s", err.Error())
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		err = servers.Delete(serverKey)
		if err != nil {
			context.Message = fmt.Sprintf("error deleting from polling: %s", err.Error())
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		// redirect to list
		http.Redirect(w, r, "/settings/servers", http.StatusSeeOther)

	}

RenderForm:

	renderFSDynamic(w, "settings-server-delete", context)
}

func serverAddPage(w http.ResponseWriter, r *http.Request) {

	var err error

	var a settings.Connections
	fv := new(gui.ServerPage)

	if r.Method == "GET" {
		fv.Connection = "trusted"
	}

	context := struct {
		Context
		//SQLServer *settings.SQLServer
		Credentials map[string]*settings.SQLCredential
		FormValues  *gui.ServerPage
		//Auth string
	}{
		Context: Context{
			Title:       "SQL Server - Add",
			UnixNow:     time.Now().Unix() * 1000,
			ErrorList:   getServerErrorList(),
			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			TagList:     globalTagList.getTags(),
			AppConfig:   getGlobalConfig(),
		},
	}

	context.EnableSave, err = settings.CanSave(r)
	if err != nil {
		context.Message = fmt.Sprintf("can save error: %s", err.Error())
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}

	if AppConfigMode == ModeFile {
		context.Message = "File Config Mode: Please add servers in the './config/*.hcl' files."
		context.MessageClass = gui.MessageClassWarning
		context.EnableSave = false
	}

	// Populate the credentials
	a, err = settings.ReadConnections()
	if err != nil {
		context.Message = fmt.Sprintln("error reading connection: ", err.Error())
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}

	context.FormValues = fv
	context.Credentials = a.SQLCredentials

	if r.Method == "POST" {

		if !context.EnableSave {
			msg := errors.New("server (add): post but can't save")
			GLOBAL_RINGLOG.Enqueue(msg.Error())
			context.Message = msg.Error()
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		var s settings.SQLServer

		err = r.ParseForm()
		if err != nil {
			context.Message = fmt.Sprintf("parse form error: %s", err.Error())
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		decoder := schema.NewDecoder()
		err = decoder.Decode(fv, r.PostForm)
		if err != nil {
			context.Message = fmt.Sprintf("parse decode error: %s", err.Error())
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}
		context.FormValues = fv

		// Check for duplicates Friendly Name
		if fv.FriendlyName != "" {
			for _, v := range a.SQLServers {
				if strings.EqualFold(v.FriendlyName, fv.FriendlyName) {
					// fmt.Println("v: ", v.ServerKey, "s: ", s.ServerKey)
					context.Message = fmt.Sprintf("A SQL Server named '%s' already exists", fv.FriendlyName)
					context.MessageClass = gui.MessageClassDanger
					goto RenderForm
				}
			}
		}

		err = fv.Validate()
		if err != nil {
			context.Message = err.Error()
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		// Populate the tags
		tags := strings.Split(fv.Tags, ",")
		for _, v := range tags {
			v = strings.TrimSpace(v)
			v = strings.Replace(v, " ", "-", -1)
			if v != "" {
				s.Tags = append(s.Tags, v)
			}
		}

		// set the values
		s.FQDN = fv.FQDN
		s.FriendlyName = fv.FriendlyName

		switch fv.Connection {
		case settings.AuthTrusted:
			s.TrustedConnection = true
		case settings.AuthCredential:
			s.CredentialKey = fv.CredentialKey
		case settings.AuthUserPassword:
			s.Login = fv.Login
			s.Password = fv.Password
		case settings.AuthCustom:
			s.CustomConnectionString = fv.ConnectionString
		default:
			context.Message = fmt.Sprintf("Invalid conenction type: %s", fv.Connection)
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		// Save it
		key, err := settings.AddSQLServer(s)
		if err != nil {
			context.Message = fmt.Sprintf("error saving server: %s", err.Error())
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		err = servers.AddFromSettings(key, s, true)
		if err != nil {
			context.Message = fmt.Sprintf("error adding to polling: %s", err.Error())
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		// Pause to it can try to poll
		time.Sleep(400 * time.Millisecond)

		// Redirect
		http.Redirect(w, r, "/settings/servers", http.StatusSeeOther)
	}

RenderForm:

	renderFSDynamic(w, "settings-server-add", context)
}

// func validateSQLServer(r *http.Request, auth string, s *settings.SQLServer) (settings.SQLServer, string, error) {

// 	// fmt.Println(s)
// 	// fmt.Println("validating: ", s.ServerKey, s.FriendlyName)
// 	var err error

// 	err = r.ParseForm()
// 	if err != nil {
// 		msg := errors.Wrap(err, "error parsing form")
// 		GLOBAL_RINGLOG.Enqueue(msg.Error())
// 		return *s, auth, msg
// 	}

// 	// Set all the values I can that won't cause errors
// 	s.FQDN = r.PostFormValue("fqdn")
// 	s.FriendlyName = r.PostFormValue("friendlyName")

// 	auth = r.PostFormValue("connection")
// 	//fmt.Println("posted auth: ", auth)

// 	if auth == "trusted" {
// 		s.TrustedConnection = true
// 	}

// 	s.CredentialKey = r.PostFormValue("credentialKey")
// 	//fmt.Println("credentialKey: ", s.CredentialKey)
// 	s.Login = r.PostFormValue("user")
// 	s.Password = r.PostFormValue("pwd")
// 	s.CustomConnectionString = r.PostFormValue("cxnstring")

// 	// Fixup the tags
// 	tags := strings.Split(r.PostFormValue("tags"), ",")
// 	for _, v := range tags {
// 		v = strings.TrimSpace(v)
// 		v = strings.Replace(v, " ", "-", -1)
// 		s.Tags = append(s.Tags, v)
// 	}

// 	// Start validating fields
// 	if s.FQDN == "" {
// 		return *s, auth, errors.New("A fully-qualified domain name is required")
// 	}

// 	a, err := settings.ReadConnections()
// 	if err != nil {
// 		msg := errors.Wrap(err, "readConnections")
// 		GLOBAL_RINGLOG.Enqueue(msg.Error())
// 		return *s, auth, msg
// 	}

// 	// check if this name exists
// 	if s.FriendlyName != "" {
// 		for _, v := range a.SQLServers {
// 			if strings.EqualFold(v.FriendlyName, s.FriendlyName) && v.ServerKey != s.ServerKey {
// 				// fmt.Println("v: ", v.ServerKey, "s: ", s.ServerKey)
// 				err = fmt.Errorf("A SQL Server named '%s' already exists", s.FriendlyName)
// 				return *s, auth, err
// 			}
// 		}
// 	}

// 	switch auth {
// 	case "trusted":
// 		s.TrustedConnection = true
// 	case "credential":
// 		if s.CredentialKey == "" {
// 			return *s, auth, errors.New("Please select a credential")
// 		}
// 	case "userpass":
// 		if s.Login == "" || s.Password == "" {
// 			return *s, auth, errors.New("Please enter a login and password")
// 		}
// 	case "custom":
// 		if s.CustomConnectionString == "" {
// 			return *s, auth, errors.New("Pleae enter a custom connection string")
// 		}
// 	default:
// 		return *s, auth, fmt.Errorf("invalid auth: %s", auth)
// 	}

// 	return *s, auth, nil
// }

func getAuth(s *settings.SQLServer) (string, error) {
	if s.TrustedConnection {
		return "trusted", nil
	}

	if s.CredentialKey != "" {
		return "credential", nil
	}

	if s.Login != "" {
		return "userpass", nil
	}

	if s.CustomConnectionString != "" {
		return "custom", nil
	}

	return "", errors.New("Invalid authorization configuration")
}

func credentialEditPage(w http.ResponseWriter, r *http.Request) {

	var err error
	var ok bool
	var c *settings.SQLCredential
	var a settings.Connections

	credentialKey := r.PathValue("credential")

	context := struct {
		Context
		Credential    *settings.SQLCredential
		CredentialKey string
	}{
		Context: Context{
			Title:           "Credentials - Edit",
			UnixNow:         time.Now().Unix() * 1000,
			ErrorList:       getServerErrorList(),
			HeaderRight:     fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			TagList:         globalTagList.getTags(),
			AppConfig:       getGlobalConfig(),
			MenuTwoSelected: "credentials",
		},
		CredentialKey: credentialKey,
	}

	context.EnableSave, err = settings.CanSave(r)
	if err != nil {
		msg := errors.Wrap(err, "cansave")
		context.Message = msg.Error()
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}

	a, err = settings.ReadConnections()
	if err != nil {
		msg := errors.Wrap(err, "readConnections")
		GLOBAL_RINGLOG.Enqueue(msg.Error())
		context.Message = msg.Error()
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}

	// Get the existing with default values
	c, ok = a.SQLCredentials[credentialKey]
	if !ok {
		context.Message = fmt.Sprintf("Credential '%s' doesn't exist", credentialKey)
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}

	context.Credential = c

	if r.Method == "POST" {

		if !context.EnableSave {
			msg := errors.New("credential: post but can't save")
			GLOBAL_RINGLOG.Enqueue(msg.Error())
			context.Message = msg.Error()
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		err = r.ParseForm()
		if err != nil {
			msg := errors.Wrap(err, "error parsing form")
			GLOBAL_RINGLOG.Enqueue(msg.Error())
			context.Message = msg.Error()
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		credentialName := r.PostFormValue("credentialName")
		login := r.PostFormValue("login")
		password := r.PostFormValue("password")

		// check lengths
		if len(credentialName) == 0 || len(login) == 0 {
			context.Message = "Name and Login must have values"
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		// check if this name exists
		if !strings.EqualFold(c.Name, credentialName) {
			for _, v := range a.SQLCredentials {
				if strings.EqualFold(v.Name, credentialName) {
					context.Message = fmt.Sprintf("A credential named '%s' already exists", credentialName)
					context.MessageClass = gui.MessageClassDanger
					goto RenderForm

				}
			}
		}

		// Handle the password
		if len(password) != 0 {
			c.Password = password
		}

		// Start the save process
		c.Name = credentialName
		c.Login = login

		//err = a.Save()
		err = settings.SaveSQLCredential(*c)
		if err != nil {
			context.Message = fmt.Sprintf("Error saving: %s", err.Error())
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		context.Message = fmt.Sprintf("Credential '%s' saved", credentialKey)
		context.MessageClass = gui.MessageClsssSuccess

	}

	context.Credential = c

	// 	// save it
	// 	a, err := settings.ReadConnections()

	// 	c := settings.SQLCredential {
	// 		Name: credentialName,
	// 		Login: login,
	// 		Password: password,
	// 	}

	// 	_, err = settings.AddSQLCredential(c)
	// 	if err != nil {
	// 		context.Message = fmt.Sprintf("error saving settings: %s", err.Error())
	// 		context.MessageClass = gui.MessageClassDanger
	// 		goto RenderForm
	// 	}

	// 	// redirect to list
	// 	http.Redirect(w, r, "/settings/credentials", http.StatusSeeOther)

	// }

	// var err error
	// a, err := settings.ReadConnections()
	// if err != nil {
	// 	context.Message = err.Error()
	// 	context.MessageClass = gui.MessageClassDanger
	// 	WinLogln(err.Error())
	// }

RenderForm:

	renderFSDynamic(w, "settings-credentials-edit", context)
}

func credentialDeletePage(w http.ResponseWriter, r *http.Request) {

	var err error
	var ok bool
	var c *settings.SQLCredential
	var i int
	var a settings.Connections

	credentialKey := r.PathValue("credential")

	context := struct {
		Context
		Credential    *settings.SQLCredential
		CredentialKey string
	}{
		Context: Context{
			Title:           "Credentials - Edit",
			UnixNow:         time.Now().Unix() * 1000,
			ErrorList:       getServerErrorList(),
			HeaderRight:     fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			TagList:         globalTagList.getTags(),
			AppConfig:       getGlobalConfig(),
			MenuTwoSelected: "credentials",
		},
		CredentialKey: credentialKey,
	}

	enableSave, err := settings.CanSave(r)
	if err != nil {
		context.Message = fmt.Sprintf("can save error: %s", err.Error())
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}

	context.EnableSave = enableSave

	a, err = settings.ReadConnections()
	if err != nil {
		msg := errors.Wrap(err, "readConnections")
		GLOBAL_RINGLOG.Enqueue(msg.Error())
		context.Message = msg.Error()
		context.MessageClass = gui.MessageClassDanger
		context.EnableSave = false
		goto RenderForm
	}

	// Get the existing with default values
	c, ok = a.SQLCredentials[credentialKey]
	if !ok {
		context.Message = fmt.Sprintf("Credential '%s' doesn't exist", credentialKey)
		context.MessageClass = gui.MessageClassDanger
		context.EnableSave = false
		goto RenderForm
	}

	context.Credential = c

	// Check if anything uses the credential
	for _, v := range a.SQLServers {
		if v.CredentialKey == c.CredentialKey.String() {
			i++
		}
	}

	if i > 0 {
		context.EnableSave = false
		context.Message = fmt.Sprintf("This credential is used by %d server(s)", i)
		context.MessageClass = gui.MessageClassDanger
		goto RenderForm
	}

	if r.Method == "POST" {

		if !context.EnableSave {
			msg := errors.New("credential (delete): post but can't save")
			GLOBAL_RINGLOG.Enqueue(msg.Error())
			context.Message = msg.Error()
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		// delete the credential
		err = settings.DeleteSQLCredential(credentialKey)
		if err != nil {
			context.Message = fmt.Sprintf("Delete error: %s", err.Error())
			context.MessageClass = gui.MessageClassDanger
			goto RenderForm
		}

		// redirect to list
		http.Redirect(w, r, "/settings/credentials", http.StatusSeeOther)

	}

RenderForm:

	renderFSDynamic(w, "settings-credentials-delete", context)
}
