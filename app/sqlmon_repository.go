package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/pkg/errors"
	"github.com/scalesql/isitsql/internal/mrepo"
	"github.com/scalesql/isitsql/internal/settings"
	"github.com/sirupsen/logrus"
)

// setupRepository sets up the SQL Server database repository
func setupRepository() error {
	exe, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "os.executable")
	}
	wd := filepath.Dir(exe)
	fileName := filepath.Join(wd, "isitsql.toml")

	// if the file doesn't exist, then we are done
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return nil // no config file, so nothing to do
	}

	// Read the file
	fileBody, err := os.ReadFile(fileName)
	if err != nil {
		return errors.Wrap(err, "os.readfile")
	}
	var config IsItSQLTOML
	err = toml.Unmarshal(fileBody, &config)
	if err != nil {
		return errors.Wrap(err, "toml.unmarshal")
	}

	// set the connection lifetime
	if config.Polling.ConnectionLifetimeMinutes != 0 {
		dur := time.Duration(config.Polling.ConnectionLifetimeMinutes) * time.Minute
		WinLogf("isitsql.toml: polling.connection_lifetime: %s", dur)
		globalConfig.Lock()
		defer globalConfig.Unlock()
		globalConfig.AppConfig.connectionLifetime = dur
	}

	// if there are no repository settings, then we are done
	if config.Repository.Host == "" && config.Repository.Database == "" && config.Repository.Credential == "" {
		return nil
	}
	if config.Repository.Host == "" {
		return errors.New("toml: host missing")
	}
	if config.Repository.Database == "" {
		return errors.New("toml: database missing")
	}

	var user, pwd string
	if config.Repository.Credential != "" { // we have a credential
		credName := config.Repository.Credential
		conns, err := settings.ReadConnectionsDecrypted()
		if err != nil {
			return errors.Wrap(err, "settings.readconnections")
		}
		found := false
		for _, cred := range conns.SQLCredentials {
			if strings.EqualFold(credName, cred.Name) {
				found = true
				user = cred.Login
				pwd = cred.Password
			}
		}
		if !found {
			return fmt.Errorf("credential not found: %s", credName)
		}
	}

	repository, err := mrepo.NewRepository(config.Repository.Host, config.Repository.Database, user, pwd, logrus.WithContext(context.Background()), &GLOBAL_RINGLOG)
	GlobalRepository = repository // set it with whatever we have
	if err != nil {
		return errors.Wrap(err, "mrepo.newrepository") // but log the error
	}
	msg := fmt.Sprintf("REPOSITORY: host='%s' database='%s'", config.Repository.Host, config.Repository.Database)
	if user != "" {
		msg += fmt.Sprintf(" user=%s", user)
	}
	WinLogf("%s", msg)
	return nil
}
