package c2

import "strings"

// Connection holds a connection that will be sent back to the app
type Connection struct {
	Key               string
	Server            string
	DisplayName       string
	Tags              []string
	CredentialName    string
	IgnoreBackups     bool
	IgnoreBackupsList []string
	Alias             bool
}

// BackupDescript returns a text description of the ignore backup fields for the GUI
func (c Connection) BackupDescription() string {
	if c.IgnoreBackups {
		return "Ignore All"
	}
	if len(c.IgnoreBackupsList) > 0 {
		return "Ignore: " + strings.Join(c.IgnoreBackupsList, ", ")
	}
	return ""
}
