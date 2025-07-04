package app

import (
	"fmt"
	"net/http"
	"time"
)

// Home is the home page
func Home(w http.ResponseWriter, req *http.Request) {
	// servers.RLock()
	// s := make(SqlServerArray, len(servers.Servers))
	// for i, v := range servers.SortedKeys {
	// 	s[i] = servers.Servers[v]
	// }
	// servers.RUnlock()
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
