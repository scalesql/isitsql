package app

import (
	"fmt"
	"time"

	"github.com/scalesql/isitsql/internal/hadr"
)

// Database holds a database
type Database struct {
	DatabaseID         int       `json:"database_id,omitempty"`
	Name               string    `json:"name,omitempty"`
	DataSizeKB         int64     `json:"data_size_kb,omitempty"`
	LogSizeKB          int64     `json:"log_size_kb,omitempty"`
	CompatibilityLevel int       `json:"compatibility_level,omitempty"`
	UserAccessDesc     string    `json:"user_access_desc,omitempty"`
	IsReadOnly         bool      `json:"is_read_only,omitempty"`
	StateDesc          string    `json:"state_desc,omitempty"`
	RecoveryModelDesc  string    `json:"recovery_model_desc,omitempty"`
	Collation          string    `json:"collation,omitempty"`
	CreateDate         time.Time `json:"create_date,omitempty"`

	// Host is the instance or availability group for the database
	Host string `json:"host,omitempty"`

	LastBackup            time.Time `json:"last_backup,omitempty"`
	LastBackupDevice      string    `json:"last_backup_device,omitempty"`
	LastBackupInstance    string    `json:"last_backup_instance,omitempty"`
	LastLogBackup         time.Time `json:"last_log_backup,omitempty"`
	LastLogBackupDevice   string    `json:"last_log_backup_device,omitempty"`
	LastLogBackupInstance string    `json:"last_log_backup_instance,omitempty"`
	BackupAlert           bool      `json:"backup_alert,omitempty"`

	// Values for mirroring
	IsMirrored bool             `json:"is_mirrored,omitempty"`
	Mirroring  mirroredDatabase `json:"mirroring,omitempty"`

	IsAG            bool   `json:"is_ag"`
	AGState         string `json:"ag_state"`
	AGDB            hadr.ReplicaDatabase
	GroupDatabaseID string `json:"group_database_id"`

	// HADRField   string
	// HADRHover   string
	//SendQueueKB int `json:"send_queue_kb"`
	//RedoQueueKB int `json:"redo_queue_kb"`
}

func (d Database) IsHADR() bool {
	if d.IsAG || d.IsMirrored {
		return true
	}
	return false
}

func (d Database) SendQueueKB() int {
	if d.IsAG {
		return int(d.AGDB.SendQueueKB)
	}
	if d.IsMirrored {
		return int(d.Mirroring.MirrorSendQueue)
	}
	return 0
}

func (d Database) RedoQueueKB() int {
	if d.IsAG {
		return int(d.AGDB.RedoQueueKB)
	}
	if d.IsMirrored {
		return int(d.Mirroring.MirrorRedoQueue)
	}
	return 0
}

// HADRField returns the string displayed on the Databases page
func (d Database) HADRField() string {
	if d.IsAG {
		return d.AGState
	}
	if d.IsMirrored {
		// {{ .Mirroring.MirrorRoleDesc }}, {{ .Mirroring.MirrorStateDesc }}, {{ .Mirroring.MirrorSafetyDesc }}
		return fmt.Sprintf("DBM: %s, %s, %s", d.Mirroring.MirrorRoleDesc, d.Mirroring.MirrorStateDesc, d.Mirroring.MirrorSafetyDesc)
	}
	return ""
}

// HADRHover displays the field used in the title of the table cell
func (d Database) HADRHover() string {
	if d.IsAG {
		return ""
	}
	if d.IsMirrored {
		// title="Partner: {{ .Mirroring.MirrorPartner }}; Witness: {{ .Mirroring.MirrorWitness }} ({{ .Mirroring.MirrorWitnessStateDesc }})
		return fmt.Sprintf("Partner: %s; Witness: %s %s", d.Mirroring.MirrorPartner, d.Mirroring.MirrorWitness, d.Mirroring.MirrorWitnessStateDesc)
	}
	return ""
}
