package app

import (
	"encoding/json"
	"fmt"
	"html"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/scalesql/isitsql/internal/hadr"
	"github.com/pkg/errors"
)

func agPage(w http.ResponseWriter, req *http.Request) {

	htmlTitle := html.EscapeString("Availability Groups - Is It SQL")

	cfg := getGlobalConfig()
	if cfg.AGAlertMB == 0 {
		cfg.AGAlertMB = math.MaxInt64
	}
	if cfg.AGWarnMB == 0 {
		cfg.AGWarnMB = math.MaxInt64
	}

	agGroups := hadr.PublicAGMap.Groups()

	type agSummaryRow struct {
		Name           string    `json:"name"`
		GUID           string    `json:"guid"`
		IsHealthy      bool      `json:"is_healthy"`
		Health         string    `json:"health"`
		State          string    `json:"state"`
		PrimaryReplica string    `json:"primary_replica"`
		SendQueue      int64     `json:"send_queue_kb"`
		RedoQueue      int64     `json:"redo_queue_kb"`
		DisplayName    string    `json:"display_name"`
		PollTime       time.Time `json:"poll_time"`
		HTMLClass      string    `json:"html_class"`
		PrimaryGUID    string
	}

	agList := make([]agSummaryRow, 0)

	for _, v := range agGroups {
		// get the queues
		var send, redo int64
		for _, r := range v.Replicas {
			send += r.SendQueue
			redo += r.RedoQueue
		}

		sr := agSummaryRow{
			Name:           v.Name,
			GUID:           v.GUID,
			IsHealthy:      v.IsHealthy(),
			Health:         v.Health,
			State:          v.State,
			PrimaryReplica: v.PrimaryReplica,
			SendQueue:      send,
			RedoQueue:      redo,
			DisplayName:    v.DisplayName,
			PollTime:       v.PollTime,
			PrimaryGUID:    v.PrimaryGUID,
		}
		// warning if > 1 GB
		if sr.SendQueue/1024 > cfg.AGWarnMB || sr.RedoQueue/1024 > cfg.AGWarnMB {
			sr.HTMLClass = "warning"
		}
		// alert if > 10 GB
		if !sr.IsHealthy || sr.SendQueue/1024 > cfg.AGAlertMB || sr.RedoQueue/1024 > cfg.AGAlertMB {
			sr.HTMLClass = "danger"
		}
		agList = append(agList, sr)
	}

	sort.SliceStable(agList, func(i, j int) bool {
		return strings.ToUpper(agList[i].DisplayName) < strings.ToUpper(agList[j].DisplayName)
	})

	sort.SliceStable(agGroups, func(i, j int) bool {
		return strings.ToUpper(agGroups[i].DisplayName) < strings.ToUpper(agGroups[j].DisplayName)
	})

	context := struct {
		Context
		AGs       []hadr.AG
		AGSummary []agSummaryRow
	}{
		Context: Context{
			Title:       htmlTitle,
			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			ErrorList:   getServerErrorList(),
			TagList:     globalTagList.getTags(),
			AppConfig:   cfg,
		},
		AGs:       agGroups,
		AGSummary: agList,
	}

	requestURL := req.URL.String()
	if requestURL == "/ag/json" {
		js, err := json.Marshal(agList)
		if err != nil {
			WinLogln(errors.Wrap(err, "ag.json.marshal"))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(js)
		if err != nil {
			WinLogln(errors.Wrap(err, "ag.json.write"))
		}
		return
	}

	renderFSDynamic(w, "ag", context)
}
