package app

import (
	"time"
)

type mirroredDatabase struct {
	MapKey                 string    `json:"map_key,omitempty"`
	ServerName             string    `json:"server_name,omitempty"`
	URL                    string    `json:"url,omitempty"`
	DatabaseName           string    `json:"database_name,omitempty"`
	IsMirrored             bool      `json:"is_mirrored,omitempty"`
	IsPrincipal            bool      `json:"is_principal,omitempty"`
	MirrorGUID             string    `json:"mirror_guid,omitempty"`
	MirrorState            int8      `json:"mirror_state,omitempty"`
	MirrorStateDesc        string    `json:"mirror_state_desc,omitempty"`
	MirrorRole             int8      `json:"mirror_role,omitempty"`
	MirrorRoleDesc         string    `json:"mirror_role_desc,omitempty"`
	MirrorSafety           int8      `json:"mirror_safety,omitempty"`
	MirrorSafetyDesc       string    `json:"mirror_safety_desc,omitempty"`
	MirrorPartner          string    `json:"mirror_partner,omitempty"`
	MirrorWitness          string    `json:"mirror_witness,omitempty"`
	MirrorWitnessState     int8      `json:"mirror_witness_state,omitempty"`
	MirrorWitnessStateDesc string    `json:"mirror_witness_state_desc,omitempty"`
	MirrorSendQueue        int64     `json:"mirror_send_queue,omitempty"`
	MirrorRedoQueue        int64     `json:"mirror_redo_queue,omitempty"`
	Priority               int       `json:"priority,omitempty"`
	LastPollTime           time.Time `json:"last_poll_time,omitempty"`
}

func getMirroredDatabases() (map[string]*mirroredDatabase, error) {
	dbs := make(map[string]*mirroredDatabase)

	// We only want one entry per instance
	ssa := servers.CloneUnique()
	for _, s := range ssa {
		for _, d := range s.Databases {
			dbm := d.Mirroring
			dbm.LastPollTime = s.LastPollTime
			if dbm.IsMirrored && dbm.IsPrincipal {
				dbm.URL = s.URL()
				dbs[dbm.MirrorGUID] = &dbm
			}
		}
	}
	return dbs, nil
}
