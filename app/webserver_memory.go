package app

import (
	"fmt"
	"net/http"
	"time"
)

func memoryPage(w http.ResponseWriter, req *http.Request) {
	ss := servers.CloneUnique()

	context := Context{
		Title:       "Memory - IsItSQL",
		HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
		SortedKeys:  servers.SortedKeys,
		TagList:     globalTagList.getTags(),
		SelectedTag: "",
		ErrorList:   getServerErrorList(),
		AppConfig:   getGlobalConfig(),
		Servers:     ss,
	}

	renderFS(w, "memory", context)
}
