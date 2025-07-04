package app

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
)

type usageRow struct {
	ServerName         string
	Domain             string
	URL                string
	VersionString      string
	ProductLevel       string
	ProductUpdateLevel string
	ProductVersion     string
	Installed          time.Time
	ProductEdition     string
	CpuCount           int
	PhysicalMemoryKB   int64
	CoresUsedSQL       float32
	CoresUsedOther     float32
	AvgCoresSQL        float32
	AvgCoresOther      float32
	PeakCoresSQL       float32
	PeakCoresOther     float32
	GetTableCssClass   string
}

func usagePage(w http.ResponseWriter, req *http.Request) {
	//ss := servers.CloneUnique()
	rows, err := getUsageRows()
	if err != nil {
		WinLogln(fmt.Sprintf("Usage Page: %s", err.Error()))
	}

	context := struct {
		Context
		Usage []usageRow
	}{
		Context: Context{
			Title:       "Usage - IsItSQL",
			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			SortedKeys:  servers.SortedKeys,
			TagList:     globalTagList.getTags(),
			SelectedTag: "",
			ErrorList:   getServerErrorList(),
			AppConfig:   getGlobalConfig(),
		},
		Usage: rows,
	}

	renderFSDynamic(w, "usage", context)
}

func usagePageCSV(w http.ResponseWriter, req *http.Request) {
	ss := servers.CloneUnique()
	rows := make([][]string, 0)
	w.Header().Set("Content-Disposition", "attachment; filename=sql_server_usage.csv")
	w.Header().Set("Content-Type", req.Header.Get("Content-Type"))
	row := []string{"instance", "domain", "version", "level", "update", "build", "edition", "installed", "cores", "memory_mb",
		"cores_sql_last", "cores_sql_avg", "cores_sql_peak",
		"cores_other_last", "cores_other_avg", "cores_other_peak"}
	rows = append(rows, row)
	for _, s := range ss {
		peaksql, peakother := s.PeakCoresUsed()
		avgsql, avgother := s.AverageCoresUsed()
		row = []string{s.ServerName, s.Domain, s.VersionString, s.ProductLevel,
			s.ProductUpdateLevel, s.ProductVersion, s.ProductEdition, s.Installed.Format("2006-01-02"),
			strconv.Itoa(s.CpuCount), strconv.Itoa(int(s.PhysicalMemoryKB / 1024)),
			fmt.Sprintf("%.1f", s.CoresUsedSQL), fmt.Sprintf("%.1f", avgsql), fmt.Sprintf("%.1f", peaksql),
			fmt.Sprintf("%.1f", s.CoresUsedOther), fmt.Sprintf("%.1f", avgother), fmt.Sprintf("%.1f", peakother),
		}
		rows = append(rows, row)
	}
	writer := csv.NewWriter(w)
	writer.UseCRLF = true
	err := writer.WriteAll(rows)
	if err != nil {
		http.Error(w, errors.Wrap(err, "writer.writeall").Error(), http.StatusInternalServerError)
	}
	err = writer.Error()
	if err != nil {
		http.Error(w, errors.Wrap(err, "writer.error").Error(), http.StatusInternalServerError)
	}
}

func getUsageRows() ([]usageRow, error) {
	ss := servers.CloneUnique()
	rows := make([]usageRow, 0)
	for _, s := range ss {
		var ur usageRow
		err := copier.Copy(&ur, &s)
		if err != nil {
			return rows, err
		}
		ur.GetTableCssClass = s.GetTableCssClass()
		ur.PeakCoresSQL, ur.PeakCoresOther = s.PeakCoresUsed()
		ur.AvgCoresSQL, ur.AvgCoresOther = s.AverageCoresUsed()
		rows = append(rows, ur)
	}
	return rows, nil
}
