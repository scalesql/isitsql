package settings

import (
	"encoding/json"
	"fmt"

	"github.com/billgraziano/dpapi"
	"github.com/billgraziano/mssqlh/v2"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// SQLServer holds the servers to poll
type SQLServer struct {
	ServerKey              string   `json:"serverKey"`
	FQDN                   string   `json:"FQDN"`
	FriendlyName           string   `json:"friendlyName,omitempty"`
	TrustedConnection      bool     `json:"trustedConnection"`
	CredentialKey          string   `json:"credentialKey,omitempty"`
	Login                  string   `json:"login,omitempty"`
	Password               string   `json:"password,omitempty"`
	Tags                   []string `json:"tags,omitempty"`
	CustomConnectionString string   `json:"connectionString,omitempty"`
	IgnoreBackups          bool
	IgnoreBackupsList      []string
}

// Types of authorizations
const (
	AuthTrusted      = "trusted"
	AuthCredential   = "credential"
	AuthUserPassword = "userpass"
	AuthCustom       = "custom"
)

// JSON returns a JSON representation of a SQL Server object
func (s *SQLServer) JSON() string {
	b, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		return fmt.Sprintf("error: %s", err.Error())
	}
	return string(b)
}

// AddSQLServer adds a server to the configuration map and returns the serverKey
func AddSQLServer(s SQLServer) (string, error) {
	var err error
	a, err := ReadConnections()
	if err != nil {
		return "", errors.Wrap(err, "readconnections")
	}

	s.ServerKey = uuid.NewV4().String()

	// Blank out all unused values in the connection
	err = s.configureConnection()
	if err != nil {
		return "", errors.Wrap(err, "configureConnection")
	}

	// Encrypt the password and connection string
	if s.Password != "" {
		s.Password, err = dpapi.Encrypt(s.Password)
		if err != nil {
			return "", errors.Wrap(err, "encrypt")
		}
	}

	if s.CustomConnectionString != "" {
		s.CustomConnectionString, err = dpapi.Encrypt(s.CustomConnectionString)
		if err != nil {
			return "", errors.Wrap(err, "encrypt")
		}
	}

	// test for valid connection key
	if s.CredentialKey != "" {
		// fmt.Println("adding server with credential: ", s.CredentialKey)
		_, ok := a.SQLCredentials[s.CredentialKey]
		if !ok {
			return "", errors.New("invalid credential")
		}
	}

	a.SQLServers[s.Key()] = &s

	err = a.Save()
	if err != nil {
		return "", errors.Wrap(err, "save")
	}

	return s.Key(), nil
}

// GetSQLServer gets one SQL Server entry from the configuration file
func GetSQLServer(key string) (*SQLServer, error) {
	var err error

	a, err := ReadConnections()
	if err != nil {
		return nil, errors.Wrap(err, "readconnections")
	}
	s, ok := a.SQLServers[key]
	if !ok {
		return s, ErrNotFound
	}

	if s.Password != "" {
		s.Password, err = dpapi.Decrypt(s.Password)
		if err != nil {
			return nil, errors.Wrap(err, "decrypt")
		}
		// fmt.Println("reading clear text password: ", s.Password)
	}

	if s.CustomConnectionString != "" {
		s.CustomConnectionString, err = dpapi.Decrypt(s.CustomConnectionString)
		if err != nil {
			return nil, errors.Wrap(err, "decrypt")
		}
		// fmt.Println("reading clear text cxn string", s.CustomConnectionString)
	}

	return s, nil
}

// Decrypt decrypts a single string using DPAPI
// func Decrypt(s string) (string, error) {
// 	d, err := dpapi.Decrypt(s)
// 	if err != nil {
// 		return "", errors.Wrap(err, "decrypt")
// 	}
// 	return d, nil
// }

// SaveSQLServer saves an existing SQL Server connection to the config file
func SaveSQLServer(key string, s *SQLServer) error {
	var err error

	a, err := ReadConnections()
	if err != nil {
		return errors.Wrap(err, "readconnections")
	}

	// get the existing one
	_, ok := a.SQLServers[key]
	if !ok {
		return ErrNotFound
	}

	err = s.configureConnection()
	if err != nil {
		return errors.Wrap(err, "configureConnection")
	}

	if s.Password != "" {
		// fmt.Println("writing clear text password: ", s.Password)
		s.Password, err = dpapi.Encrypt(s.Password)
		if err != nil {
			return errors.Wrap(err, "encrypt")
		}
	}

	if s.CustomConnectionString != "" {
		// fmt.Println("writing cxn string: ", s.CustomConnectionString)
		s.CustomConnectionString, err = dpapi.Encrypt(s.CustomConnectionString)
		if err != nil {
			return errors.Wrap(err, "encrypt")
		}
	}

	a.SQLServers[s.Key()] = s

	err = a.Save()
	if err != nil {
		return errors.Wrap(err, "save")
	}

	return nil
}

// Enforces a hierarchy of connection options
func (s *SQLServer) configureConnection() error {
	if s.TrustedConnection {
		s.CustomConnectionString = ""
		s.CredentialKey = ""
		s.Login = ""
		s.Password = ""
		return nil
	}

	if s.CustomConnectionString != "" {
		s.TrustedConnection = false
		s.CredentialKey = ""
		s.Login = ""
		s.Password = ""
		return nil
	}

	if s.CredentialKey != "" {
		s.TrustedConnection = false
		s.CustomConnectionString = ""
		s.Login = ""
		s.Password = ""
		return nil
	}

	if s.Login != "" && s.Password != "" {
		s.TrustedConnection = false
		s.CustomConnectionString = ""
		s.CredentialKey = ""
		return nil
	}

	return errors.New("no way to log in")
}

// Key gets the server key as as string
func (s *SQLServer) Key() string {
	return s.ServerKey
}

// ConnectionDescription returns an English description of the connection
func (s *SQLServer) ConnectionDescription() string {
	if s.TrustedConnection {
		return "Trusted Connection"
	}

	if s.CredentialKey != "" {
		return fmt.Sprintf("Credential Key: %s", s.CredentialKey)
	}

	if s.Login != "" {
		return fmt.Sprintf("SQL Server Login: %s", s.Login)
	}

	if s.CustomConnectionString != "" {
		return "Custom Connection String"
	}

	return "Invalid Connection"
}

// ConnectionString returns a connection string for this SQL Server
func (s *SQLServer) ConnectionString() (string, error) {

	if s.TrustedConnection {
		// cxn := mssqlodbc.Connection{
		// 	Server:  s.FQDN,
		// 	AppName: "IsItSQL",
		// 	Trusted: true,
		// }
		// cs, err := cxn.ConnectionString()
		// if err != nil {
		// 	return "", errors.Wrap(err, "connectionString")
		// }
		// query := url.Values{}
		// query.Add("app name", "IsItSQL")
		// //query.Add("encrypt", "false")
		// //query.Add("TrustServerCertificate", "true")

		// u := &url.URL{
		// 	Scheme: "sqlserver",
		// 	//User:   url.UserPassword(username, password),
		// 	//Host:   fmt.Sprintf("%s:%d", hostname, port),
		// 	Host:     "D40",
		// 	Path:     "SQL2019", // if connecting to an instance instead of a port
		// 	RawQuery: query.Encode(),
		// }
		// println(u.String())
		//return u.String(), nil
		conn := mssqlh.NewConnection(s.FQDN, "", "", "master", "IsItSQL")
		return conn.String(), nil
	}

	if s.CustomConnectionString != "" {
		return s.CustomConnectionString, nil
	}

	if s.Login != "" {
		// cxn := mssqlodbc.Connection{
		// 	Server:   s.FQDN,
		// 	AppName:  "IsItSQL",
		// 	User:     s.Login,
		// 	Password: s.Password,
		// }
		// cs, err := cxn.ConnectionString()
		// if err != nil {
		// 	return "", errors.Wrap(err, "connectionString")
		// }
		// 		return cs, nil
		conn := mssqlh.NewConnection(s.FQDN, s.Login, s.Password, "master", "IsItSQL")
		return conn.String(), nil
	}

	if s.CredentialKey != "" {
		cred, err := GetSQLCredential(s.CredentialKey)
		if err != nil {
			return "", errors.Wrap(err, "getsqlcredential")
		}
		// cxn := mssqlodbc.Connection{
		// 	Server:   s.FQDN,
		// 	AppName:  "IsItSQL",
		// 	User:     cred.Login,
		// 	Password: cred.Password,
		// }
		// cs, err := cxn.ConnectionString()
		// if err != nil {
		// 	return "", errors.Wrap(err, "connectionString")
		// }
		// return cs, nil
		conn := mssqlh.NewConnection(s.FQDN, cred.Login, cred.Password, "master", "IsItSQL")
		return conn.String(), nil
	}

	return "", fmt.Errorf("can't determine connection string (%s)", s.ServerKey)
}

// LinkName returns text that can be used in FQDN is blank
func (s *SQLServer) LinkName() string {
	if s.FQDN != "" {
		return s.FQDN
	}

	if s.FriendlyName != "" {
		return s.FriendlyName
	}

	return s.ServerKey
}
