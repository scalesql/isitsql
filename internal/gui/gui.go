package gui

import (
	"errors"

	"github.com/scalesql/isitsql/settings"
)

var MessageClassDanger = "alert-danger"
var MessageClassWarning = "alert-warning"
var MessageClsssSuccess = "alert-success"

// ServerPage is ued to pass values back from settings-server-*
type ServerPage struct {
	FQDN             string
	FriendlyName     string
	Connection       string
	CredentialKey    string
	Login            string
	Password         string
	ConnectionString string //`schema:"name"`
	Tags             string
}

// Validate checks for basic errors in what's passed back
func (s *ServerPage) Validate() error {

	if s.FQDN == "" && s.Connection != settings.AuthCustom {
		return errors.New("FQDN is required for Trusted and SQL Logins")
	}

	if s.Connection == settings.AuthCredential && s.CredentialKey == "" {
		return errors.New("please choose a credential")
	}

	if s.Connection == settings.AuthUserPassword && (s.Login == "") {
		return errors.New("please enter a login name and password")
	}

	if s.Connection == settings.AuthCustom && s.ConnectionString == "" {
		return errors.New("please enter a connection string")
	}

	return nil
}
