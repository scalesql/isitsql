package settings

import (
	//"fmt"

	"github.com/billgraziano/dpapi"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// SQLCredential holds encrypted SQL Server logins
type SQLCredential struct {
	CredentialKey uuid.UUID `json:"credentialKey"`
	Name          string    `json:"name"`
	Login         string    `json:"login"`
	Password      string    `json:"password"`
}

// AddSQLCredential adds a server to the configuration map
func AddSQLCredential(name, login, password string) (string, error) {
	var c SQLCredential
	var err error
	a, err := ReadConnections()
	if err != nil {
		return "", errors.Wrap(err, "readconnections")
	}

	c.CredentialKey = uuid.NewV4()
	c.Name = name
	c.Login = login
	c.Password, err = dpapi.Encrypt(password)
	if err != nil {
		return "", errors.Wrap(err, "encrypt")
	}

	a.SQLCredentials[c.Key()] = &c

	err = a.Save()
	if err != nil {
		return "", errors.Wrap(err, "save")
	}

	return c.Key(), nil
}

// SaveSQLCredential adds a server to the configuration map
func SaveSQLCredential(c SQLCredential) error {

	var err error
	if len(c.Name) == 0 {
		return errors.New("name must have a value")
	}

	if len(c.Login) == 0 {
		return errors.New("Login must have a value")
	}

	a, err := ReadConnections()
	if err != nil {
		return errors.Wrap(err, "readconnections")
	}
	x, ok := a.SQLCredentials[c.Key()]
	if !ok {
		return ErrNotFound
	}
	x.Login = c.Login
	x.Name = c.Name

	// only update the password if one was passed in
	if len(c.Password) != 0 {
		x.Password, err = dpapi.Encrypt(c.Password)
		if err != nil {
			return errors.Wrap(err, "encrypt")
		}
	}

	//a.SQLCredentials[c.Name] = x

	err = a.Save()
	if err != nil {
		return errors.Wrap(err, "save")
	}

	return nil
}

// DeleteSQLCredential removes a credential
func DeleteSQLCredential(key string) error {
	var err error

	a, err := ReadConnections()
	if err != nil {
		return errors.Wrap(err, "readconnections")
	}

	c, ok := a.SQLCredentials[key]
	if !ok {
		return nil
	}

	isused, err := c.IsUsed()
	if err != nil {
		return errors.Wrap(err, "isused")
	}

	if isused {
		return errors.New("Credential is used")
	}

	delete(a.SQLCredentials, key)
	err = a.Save()
	if err != nil {
		return errors.Wrap(err, "save")
	}

	return nil
}

// DeleteSQLServer removes a SQL Server
func DeleteSQLServer(key string) error {
	var err error

	a, err := ReadConnections()
	if err != nil {
		return errors.Wrap(err, "readconnections")
	}

	_, ok := a.SQLServers[key]
	if !ok {
		return nil
	}

	delete(a.SQLServers, key)
	err = a.Save()
	if err != nil {
		return errors.Wrap(err, "save")
	}

	return nil
}

// IsUsed checks if any connections use this credential
func (c *SQLCredential) IsUsed() (bool, error) {

	a, err := ReadConnections()
	if err != nil {
		return false, errors.Wrap(err, "readconnections")
	}

	// Check if anything uses the credential
	for _, v := range a.SQLServers {
		if v.CredentialKey == c.CredentialKey.String() {
			return true, nil
		}
	}

	return false, nil
}

// GetSQLCredential gets a SQLCredential
func GetSQLCredential(key string) (*SQLCredential, error) {
	var err error
	var c *SQLCredential

	a, err := ReadConnections()
	if err != nil {
		return c, errors.Wrap(err, "readconnections")
	}

	c, ok := a.SQLCredentials[key]
	if !ok {
		return c, ErrNotFound
	}
	c.Password, err = dpapi.Decrypt(c.Password)
	if err != nil {
		return c, errors.Wrap(err, "decrypt")
	}

	return c, nil

}

// ListSQLCredentials gets all the SQLCredentials in an array
func ListSQLCredentials() ([]*SQLCredential, error) {

	var err error
	var result []*SQLCredential

	a, err := ReadConnections()
	if err != nil {
		return nil, errors.Wrap(err, "readconnections")
	}
	for _, v := range a.SQLCredentials {
		result = append(result, v)
	}

	return result, nil

}

// Key gets the credential key as a string
func (c *SQLCredential) Key() string {
	return c.CredentialKey.String()
}
