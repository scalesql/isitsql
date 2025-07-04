package app

import (
	"fmt"
	"net/http"
	"time"
)

func ipPage(w http.ResponseWriter, req *http.Request) {
	ss := servers.CloneAll()

	context := Context{
		Title:       "IP Addresses - IsItSQL",
		HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
		SortedKeys:  servers.SortedKeys,
		TagList:     globalTagList.getTags(),
		SelectedTag: "",
		ErrorList:   getServerErrorList(),
		AppConfig:   getGlobalConfig(),
		Servers:     ss,
	}

	renderFS(w, "ips", context)
}

// func versionPageCSV(w http.ResponseWriter, req *http.Request) {
// 	ss := servers.CloneUnique()
// 	rows := make([][]string, 0)
// 	w.Header().Set("Content-Disposition", "attachment; filename=sql_server_versions.csv")
// 	w.Header().Set("Content-Type", req.Header.Get("Content-Type"))
// 	row := []string{"instance", "domain", "version", "level", "update", "build", "edition", "os", "arch", "installed", "cores", "memory_mb"}
// 	rows = append(rows, row)
// 	for _, s := range ss {
// 		row = []string{s.ServerName, s.Domain, s.VersionString, s.ProductLevel,
// 			s.ProductUpdateLevel, s.ProductVersion, s.ProductEdition, s.OSName, s.OSArch,
// 			s.Installed.Format("2006-01-02"),
// 			strconv.Itoa(s.CpuCount), strconv.Itoa(int(s.PhysicalMemoryKB / 1024))}
// 		rows = append(rows, row)
// 	}
// 	writer := csv.NewWriter(w)
// 	writer.UseCRLF = true
// 	err := writer.WriteAll(rows)
// 	if err != nil {
// 		http.Error(w, errors.Wrap(err, "writer.writeall").Error(), http.StatusInternalServerError)
// 	}
// 	err = writer.Error()
// 	if err != nil {
// 		http.Error(w, errors.Wrap(err, "writer.error").Error(), http.StatusInternalServerError)
// 	}
// }
