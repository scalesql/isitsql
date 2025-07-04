package main

import (
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/scalesql/isitsql/internal/backup"
	"github.com/scalesql/isitsql/internal/c2"
	"github.com/scalesql/isitsql/internal/fileio"
	"github.com/scalesql/isitsql/settings"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/kardianos/osext"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

func main() {
	log.Println("cfg2file.exe...")
	configPath, err := configPath()
	if err != nil {
		log.Fatal(errors.Wrap(err, "configpath"))
	}
	log.Println("'config' path: ", configPath)
	srvpath, err := c2.Path()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("'servers' path:", srvpath)
	err = os.MkdirAll(srvpath, 0600)
	if err != nil {
		log.Fatal(err)
	}

	err = doAGNames(srvpath)
	if err != nil {
		log.Fatal(errors.Wrap(err, "doagnames"))
	}

	err = doConnections(srvpath)
	if err != nil {
		log.Fatal(errors.Wrap(err, "doconnections"))
	}

	// wm := app.WaitMapping{}
	// wm.Mappings = make(map[string]app.WaitMap)
	// err = wm.ReadWaitMapping("waits.txt")
	// if err != nil {
	// 	log.Fatal(errors.Wrap(err, "wm.readwaitmappings"))
	// }
	// log.Printf("wait mappings: %d\n", len(wm.Mappings))

}

// ignored2Map returns a map[instance][]databases
// an empty array means all databases
func ignored2Map(ignored [][]string) map[string][]string {
	m := make(map[string][]string)
	for _, line := range ignored {
		if len(line) < 2 || len(line) > 3 {
			log.Printf("ERROR: skipped: %v\n", line)
			continue
		}
		srv := string(c2.FixSlashes([]byte(line[1])))
		arr, exists := m[srv]
		// if no map entry, add an empty array
		if !exists {
			arr = make([]string, 0)
			m[srv] = arr
		}
		// if len(array) == 3, append to the map's array
		if len(line) == 3 {
			arr = append(arr, line[2])
			m[srv] = arr
		}
	}
	return m
}

func doConnections(srvpath string) error {
	conns, err := settings.ReadConnections()
	if err != nil {
		return errors.Wrap(err, "app.readconnections")
	}
	log.Printf("connections: %d\n", len(conns.SQLServers))

	ignored, err := backup.GetIgnoredBackups()
	if err != nil {
		return errors.Wrap(err, "backup.getignoredbackups")
	}
	log.Printf("ignored backups: %d\n", len(ignored))
	bumap := ignored2Map(ignored)
	log.Printf("%+v\n", bumap)

	sorted := make([]settings.SQLServer, 0)
	for _, conn := range conns.SQLServers {
		sorted = append(sorted, *conn)
	}
	// Sort on FQDN
	sort.Slice(sorted[:], func(i, j int) bool {
		return sorted[i].FQDN < sorted[j].FQDN
	})

	f := hclwrite.NewEmptyFile()
	rootBody := f.Body()
	rootBody.AppendNewBlock("defaults", nil)
	rootBody.AppendNewline()
	//defaultBody := agBlock.Body()

	for _, conn := range sorted {
		if conn.FQDN == "" {
			return errors.New("empty FQDN")
		}
		connBlock := rootBody.AppendNewBlock("server", []string{conn.FQDN})
		connBody := connBlock.Body()
		// connBody.SetAttributeValue("server", cty.StringVal(string(conn.FQDN)))

		if len(conn.FriendlyName) > 0 {
			connBody.SetAttributeValue("display_name", cty.StringVal(string(conn.FriendlyName)))
		}

		if len(conn.Tags) > 0 {
			vals := make([]cty.Value, 0)
			for _, t := range conn.Tags {
				vals = append(vals, cty.StringVal(t))
			}
			connBody.SetAttributeValue("tags", cty.ListVal(vals))
		}

		if len(conn.CredentialKey) > 0 {
			for _, cred := range conns.SQLCredentials {
				if cred.CredentialKey.String() == conn.CredentialKey {
					connBody.SetAttributeValue("credential", cty.StringVal(string(cred.Name)))
				}
			}
		}

		if conn.ServerKey == "" {
			return errors.New("empty server key")
		} else {
			connBody.SetAttributeValue("key", cty.StringVal(string(conn.ServerKey)))
			// connBody.SetAttributeValue("slug", cty.StringVal(string(conn.ServerKey)))
		}

		// check for ignored backups
		fqdn := string(c2.FixSlashes([]byte(conn.FQDN)))
		log.Printf("fqdn: %v\n", fqdn)
		dblist, ok := bumap[fqdn]
		if ok {
			if len(dblist) == 0 {
				connBody.SetAttributeValue("ignore_backups", cty.BoolVal(true))
			} else {
				vals := make([]cty.Value, 0)
				for _, n := range dblist {
					vals = append(vals, cty.StringVal(n))
				}
				connBody.SetAttributeValue("ignore_backups_list", cty.ListVal(vals))
			}
		}

		rootBody.AppendNewline()
	}
	// fmt.Println("-- connections.isitsql.hcl ----------------------------")
	// fmt.Printf("%s\n", f.Bytes())
	// fmt.Println("----------------------------------------------------")
	file := filepath.Join(srvpath, "servers.isitsql.hcl")
	err = os.WriteFile(file, f.Bytes(), 0600)
	if err != nil {
		return errors.Wrap(err, "os.writefile")
	}
	return nil
}

func doAGNames(srvpath string) error {
	agNames, err := fileio.ReadConfigCSV("ag_names.csv")
	if err != nil {
		return errors.Wrap(err, "fileio.readconfigcsv")
	}
	log.Printf("AG names: %d\n", len(agNames))
	f := hclwrite.NewEmptyFile()
	rootBody := f.Body()

	for _, array := range agNames {
		agBlock := rootBody.AppendNewBlock("ag_name", nil)
		agBody := agBlock.Body()
		agBody.SetAttributeValue("domain", cty.StringVal(array[0]))
		agBody.SetAttributeValue("name", cty.StringVal(array[1]))
		agBody.SetAttributeValue("display_name", cty.StringVal(array[2]))
		rootBody.AppendNewline()
	}
	// fmt.Println("-- ag_names.isitsql.hcl ----------------------------")
	// fmt.Printf("%s\n", f.Bytes())
	// fmt.Println("----------------------------------------------------")
	file := filepath.Join(srvpath, "ag_names.hcl")
	err = os.WriteFile(file, f.Bytes(), 0600)
	if err != nil {
		return errors.Wrap(err, "os.writefile")
	}
	return nil
}

// func doSettings() error {
// 	cfg, err := settings.ReadConfig()
// 	if err != nil {
// 		return errors.Wrap(err, "settings.readconfig")
// 	}
// 	// log.Printf("settings: %v\n", cfg)
// 	f := hclwrite.NewEmptyFile()
// 	rootBody := f.Body()
// 	rootBody.SetAttributeValue("port", cty.NumberIntVal(int64(cfg.Port)))
// 	// rootBody.SetAttributeValue("security_policy", cty.StringVal(string(cfg.SecurityPolicy)))
// 	rootBody.SetAttributeValue("backup_alert_hours", cty.NumberIntVal(int64(cfg.BackupAlertHours)))
// 	rootBody.SetAttributeValue("log_alert_minutes", cty.NumberIntVal(int64(cfg.LogBackupAlertMinutes)))
// 	rootBody.SetAttributeValue("enable_profiler", cty.BoolVal(cfg.EnableProfiler))
// 	rootBody.SetAttributeValue("admin_group", cty.StringVal(string(cfg.AdminDomainGroup)))
// 	rootBody.SetAttributeValue("homepage_url", cty.StringVal(string(cfg.HomePageURL)))
// 	rootBody.SetAttributeValue("ag_alert_mb", cty.NumberIntVal(int64(cfg.AGAlertMB)))
// 	rootBody.SetAttributeValue("ag_warn_mb", cty.NumberIntVal(int64(cfg.AGWarnMB)))
// 	rootBody.SetAttributeValue("log_debug", cty.BoolVal(cfg.Debug))
// 	rootBody.SetAttributeValue("log_trace", cty.BoolVal(cfg.Trace))
// 	fmt.Println("-- settings.isitsql.hcl ----------------------------")
// 	fmt.Printf("%s\n", f.Bytes())
// 	fmt.Println("----------------------------------------------------")
// 	err = os.WriteFile("config/settings.isitsql.hcl", f.Bytes(), 0600)
// 	if err != nil {
// 		return errors.Wrap(err, "os.writefile")
// 	}
// 	return nil
// }

func configPath() (string, error) {
	wd, err := osext.ExecutableFolder()
	if err != nil {
		return "", errors.Wrap(err, "executableFolder")
	}

	p := filepath.Join(wd, "config")
	return p, nil
}
