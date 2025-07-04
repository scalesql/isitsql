package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/billgraziano/dpapi"
	"github.com/kardianos/osext"
	"github.com/pkg/errors"
)

var mu sync.RWMutex

// Connections is for app configuration
type Connections struct {
	SQLCredentials map[string]*SQLCredential `json:"credentials,omitempty"`
	SQLServers     map[string]*SQLServer     `json:"servers"`
}

// ReadConnections returns the configuration setting from the file
func ReadConnections() (Connections, error) {
	mu.RLock()
	defer mu.RUnlock()

	a := newConnections()

	wd, err := osext.ExecutableFolder()
	if err != nil {
		return a, errors.Wrap(err, "executableFolder")
	}

	fileName := filepath.Join(wd, "config", "connections.json")

	// if the file doesn't exist, then create it with defaults
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		err := a.save()
		if err != nil {
			return a, errors.Wrap(err, "save")
		}
		return a, nil
	}

	/* #nosec G304 */
	fileBody, err := os.ReadFile(fileName)
	if err != nil {
		return a, errors.Wrap(err, "readfile")
	}

	err = json.Unmarshal(fileBody, &a)
	if err != nil {
		return a, errors.Wrap(err, "unmarshal")
	}

	return a, nil
}

// ReadConnectionsDecrypted returns the configuration file with all passwords and custom connection strings decrypted
// This is used to read them all in at setup
func ReadConnectionsDecrypted() (Connections, error) {
	var err error

	a, err := ReadConnections()
	if err != nil {
		return a, errors.Wrap(err, "ReadConnections")
	}

	for _, cred := range a.SQLCredentials {
		if cred.Password != "" {
			cred.Password, err = dpapi.Decrypt(cred.Password)
			if err != nil {
				return a, errors.Wrap(err, "Decrypt")
			}
		}
	}

	for _, sql := range a.SQLServers {
		if sql.Password != "" {
			sql.Password, err = dpapi.Decrypt(sql.Password)
			if err != nil {
				return a, errors.Wrap(err, "Decrypt")
			}
		}

		if sql.CustomConnectionString != "" {
			sql.CustomConnectionString, err = dpapi.Decrypt(sql.CustomConnectionString)
			if err != nil {
				return a, errors.Wrap(err, "Decrypt")
			}
		}
	}

	return a, nil
}

// Save writes the configuration settings
func (a *Connections) Save() error {
	mu.Lock()
	defer mu.Unlock()

	err := a.save()
	if err != nil {
		return errors.Wrap(err, "save")
	}
	return nil
}

func (a *Connections) save() error {
	wd, err := osext.ExecutableFolder()
	if err != nil {
		return errors.Wrap(err, "executableFolder")
	}

	file := filepath.Join(wd, "config", "connections.json")

	b, err := json.MarshalIndent(a, "", "\t")
	if err != nil {
		return errors.Wrap(err, "marshal")
	}

	err = os.WriteFile(file, b, 0600)
	if err != nil {
		return errors.Wrap(err, "writefile")
	}

	return nil
}

// newConnections returns a blank TargetConfig
func newConnections() Connections {
	var a Connections
	a.SQLCredentials = make(map[string]*SQLCredential)
	a.SQLServers = make(map[string]*SQLServer)
	return a
}
