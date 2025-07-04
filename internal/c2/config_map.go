package c2

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/scalesql/isitsql/internal/tags"
	"gobn.github.io/coalesce"
)

type AGIdentifier struct {
	Domain string
	Name   string
}

type ConnectionMap map[string]Connection
type AGMap map[AGIdentifier]string

type ConfigMaps struct {
	Files       []string
	Connections ConnectionMap
	AGs         AGMap
}

func makeMap(names []string, files []ConnectionFile) (ConfigMaps, []string) {
	msgs := make([]string, 0)
	fileConfig := ConfigMaps{Files: names}
	var idregex = regexp.MustCompile(`^([a-zA-Z0-9][a-zA-Z0-9-_\.]*[a-zA-Z0-9-_]{0,1})$`)
	cm := make(ConnectionMap, 0)
	agm := make(AGMap, 0)
	for _, cf := range files {
		for _, i := range cf.Instances {
			conn := Connection{Tags: []string{}, IgnoreBackupsList: []string{}}
			// Assign any defaults
			if cf.Defaults != nil {
				if cf.Defaults.Credential != nil {
					conn.CredentialName = *cf.Defaults.Credential
				}
				if cf.Defaults.Tags != nil {
					conn.Tags = *cf.Defaults.Tags
				}
				if cf.Defaults.IgnoreBackupsList != nil {
					conn.IgnoreBackupsList = *cf.Defaults.IgnoreBackupsList
				}
				if cf.Defaults.IgnoreBackups != nil {
					conn.IgnoreBackups = *cf.Defaults.IgnoreBackups
				}
			}
			key := *coalesce.String(i.Key, &i.ID)
			key = strings.Replace(key, `\`, "-", -1)
			key = strings.TrimSpace(key)
			key = strings.ToLower(key)
			if !idregex.MatchString(string(key)) {
				msgs = append(msgs, fmt.Sprintf("invalid key: '%s'", key))
				continue
			}
			conn.Key = key
			conn.Server = *coalesce.String(i.Server, &i.ID)
			// conn.DisplayName = *coalesce.String(i.DisplayName, &i.ID, ptr(""))
			// Don't use the ID as the display name by default
			conn.DisplayName = *coalesce.String(i.DisplayName, ptr(""))
			if i.Tags != nil {
				// conn.Tags = *i.Tags
				conn.Tags = tags.Merge(&conn.Tags, i.Tags)
			}
			if i.Credential != nil {
				conn.CredentialName = *i.Credential
			}
			if i.IgnoreBackups != nil {
				conn.IgnoreBackups = *i.IgnoreBackups
			}
			if i.IgnoreBackupsList != nil {
				//conn.IgnoreBackupsList = *i.IgnoreBackupsList
				conn.IgnoreBackupsList = tags.Merge(&conn.IgnoreBackupsList, i.IgnoreBackupsList)
			}
			// lower-case the database list
			for j := range conn.IgnoreBackupsList {
				conn.IgnoreBackupsList[j] = strings.ToLower(conn.IgnoreBackupsList[j])
			}
			_, ok := cm[key]
			if ok {
				msgs = append(msgs, fmt.Sprintf("duplicate key: '%s'", key))
				continue
			}
			cm[key] = conn
		}
		fileConfig.Connections = cm

		for _, ag := range cf.AGNames {
			agkey := AGIdentifier{Domain: ag.Domain, Name: ag.Name}
			_, exists := agm[agkey]
			if exists {
				msgs = append(msgs, fmt.Sprintf("duplicate AG: domain='%s', ag='%s'", ag.Domain, ag.Name))
				continue
			}
			agm[agkey] = ag.DisplayName
		}
		fileConfig.AGs = agm
	}
	return fileConfig, msgs
}
