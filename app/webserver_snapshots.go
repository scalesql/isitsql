package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"time"
)

func snapshotList(w http.ResponseWriter, req *http.Request) {
	ss := servers.CloneUnique()

	type Snap struct {
		URL        string
		ServerName string
		Domain     string
		Name       string
		Source     string
		Size       int64
		Created    time.Time
		CreatedUTC time.Time
	}

	snaps := make([]Snap, 0)

	for _, srv := range ss {
		for _, snap := range srv.Snapshots {
			snaps = append(snaps, Snap{
				URL:        path.Join(srv.URL(), "databases"),
				ServerName: srv.ServerName,
				Domain:     srv.Domain,
				Name:       snap.Name,
				Source:     snap.Source,
				Size:       snap.Size,
				Created:    snap.Created,
				CreatedUTC: snap.CreatedUTC,
			})
		}
	}

	requestURL := req.URL.String()
	if requestURL == "/snapshots/json" {
		// jsonAlerts := make([]serverSummary, 0)
		// for _, v := range instanceSummary {
		// 	jsonAlerts = append(jsonAlerts, v)
		// }
		js, err := json.Marshal(snaps)
		if err != nil {
			WinLogln("Error: jsonsnapshots: ", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(js)
		return
	}

	var pageData struct {
		Context
		Snapshots []Snap
	}

	pageData.Context = Context{
		Title:       "Snapshots - IsItSQL",
		HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
		SortedKeys:  servers.SortedKeys,
		TagList:     globalTagList.getTags(),
		SelectedTag: "",
		ErrorList:   getServerErrorList(),
		AppConfig:   getGlobalConfig(),
	}

	pageData.Snapshots = snaps

	renderFSDynamic(w, "snapshots", pageData)
}
