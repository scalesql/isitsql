package app

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Home is the home page
func Home(w http.ResponseWriter, req *http.Request) {
	cfg := getGlobalConfig()

	var ss SqlServerArray
	var selectedTag string

	if homeTag := strings.TrimSpace(cfg.HomePageTag); homeTag != "" {
		globalTagList.RLock()
		t, ok := globalTagList.Tags[strings.ToLower(homeTag)]
		globalTagList.RUnlock()
		if ok {
			ss = cloneServersByTag(t)
			selectedTag = homeTag
		} else {
			ss = servers.CloneAll()
		}
	} else {
		ss = servers.CloneAll()
	}

	t := ss.getTotal()

	context := Context{
		Title:       "Is It SQL",
		Servers:     ss,
		HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
		SortedKeys:  servers.SortedKeys,
		TagList:     globalTagList.getTags(),
		SelectedTag: selectedTag,
		ErrorList:   getServerErrorList(),
		AppConfig:   cfg,
		TotalLine:   t,
	}

	renderFS(w, "index", context)
}

// All is the All Servers page
func All(w http.ResponseWriter, req *http.Request) {
	ss := servers.CloneAll()
	t := ss.getTotal()

	context := Context{
		Title:       "Is It SQL",
		Servers:     ss,
		HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
		SortedKeys:  servers.SortedKeys,
		TagList:     globalTagList.getTags(),
		SelectedTag: "",
		ErrorList:   getServerErrorList(),
		AppConfig:   getGlobalConfig(),
		TotalLine:   t,
	}

	renderFS(w, "index", context)
}

// ConnectionsPage lists all the servers and their connection information
func ConnectionsPage(w http.ResponseWriter, req *http.Request) {
	ss := servers.CloneAll()

	context := Context{
		Title:       "Connections-Is It SQL",
		Servers:     ss,
		HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
		SortedKeys:  servers.SortedKeys,
		TagList:     globalTagList.getTags(),
		SelectedTag: "",
		ErrorList:   getServerErrorList(),
		AppConfig:   getGlobalConfig(),
	}

	renderFS(w, "connections", context)
}
