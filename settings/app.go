package settings

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/billgraziano/dpapi"
	"github.com/kardianos/osext"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

// SecurityPolicyType holds setting page security
type SecurityPolicyType string

// Used to define the security types
const (
	OpenPolicy      SecurityPolicyType = "open"
	LocalHostPolicy SecurityPolicyType = "localhost"
	DomainPolicy    SecurityPolicyType = "domain"
)

// Mutex is a global mutex for settings
var Mutex sync.RWMutex

// AppConfig is for app configuration
type AppConfig struct {

	// From settings.json
	ClientGUID            string             `json:"clientguid"`
	PollWorkers           int                `json:"pollWorkers"`
	Port                  int                `json:"port"`
	SecurityPolicy        SecurityPolicyType `json:"securityPolicy"`
	BackupAlertHours      int                `json:"backupAlertHours"`
	LogBackupAlertMinutes int                `json:"logBackupAlertMinutes"`
	EnableProfiler        bool               `json:"enableProfiler"`
	EnableStatsviz        bool               `json:"enableStatsviz"`
	ErrorReporting        bool               `json:"errorReporting"`
	UsageReporting        bool               `json:"usageReporting"`
	MetricHost            string             `json:"metricHost"`
	AdminDomainGroup      string             `json:"adminDomainGroup"`
	HomePageURL           string             `json:"homePageURL"`
	SessionKey            string             `json:"sessionKey"`
	AGAlertMB             int64              `json:"ag_alert_mb"`
	AGWarnMB              int64              `json:"ag_warn_mb"`
	Debug                 bool               `json:"log_debug"`
	Trace                 bool               `json:"log_trace"`
	PProfLogMB            int                `json:"pprof_log_mb"`

	// Dynamic settings
	// IsEnterprise   bool
	// UseLocalStatic bool
	// IsBeta         bool ``
}

// Save writes the configuration settings
func (a *AppConfig) Save() error {
	Mutex.Lock()
	defer Mutex.Unlock()

	var err error
	err = a.Validate()
	if err != nil {
		return errors.Wrap(err, "validate")
	}

	wd, err := osext.ExecutableFolder()
	if err != nil {
		return errors.Wrap(err, "executableFolder")
	}

	file := filepath.Join(wd, "config", "settings.json")

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

// ReadConfig returns the configuration setting from the file
func ReadConfig() (AppConfig, error) {

	var a AppConfig

	// set the defaults I want.  Unmarshall will override these
	a.PollWorkers = 0
	a.Port = 8143
	a.SecurityPolicy = LocalHostPolicy
	a.BackupAlertHours = 36
	a.LogBackupAlertMinutes = 90
	a.ClientGUID = uuid.NewV4().String()
	a.EnableProfiler = false
	a.ErrorReporting = false
	a.UsageReporting = false
	a.MetricHost = "metrics.isitsql.com"
	a.HomePageURL = "/"

	wd, err := osext.ExecutableFolder()
	if err != nil {
		return a, errors.Wrap(err, "executableFolder")
	}

	fileName := filepath.Join(wd, "config", "settings.json")

	// if the file doesn't exist, then create it with defaults
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		// TODO set a default session key and encrypt it
		key, err := newEncryptedSessionKey()
		if err != nil {
			return a, errors.Wrap(err, "newencryptedsessionkey")
		}
		a.SessionKey = key
		a.Save()
		return a, nil
	}

	// Read the file
	/* #nosec G304 */
	fileBody, err := os.ReadFile(fileName)
	if err != nil {
		return a, errors.Wrap(err, "readfile")
	}
	//fmt.Println("fileBody len: ", len(fileBody))

	err = json.Unmarshal(fileBody, &a)
	if err != nil {
		//fmt.Println(fileBody)
		return a, errors.Wrap(err, "unmarshal")
	}

	// Check for invalid values
	err = a.Validate()
	if err != nil {
		return a, errors.Wrap(err, "validateconfig")
	}

	// if sessionKey is empty -- generate it, encrypt it and save it
	if a.SessionKey == "" {
		logrus.Debug("generating session key...")
		key, err := newEncryptedSessionKey()
		if err != nil {
			return a, errors.Wrap(err, "newsessionkey")
		}
		a.SessionKey = key
		err = a.Save()
		if err != nil {
			return a, errors.Wrap(err, "a.save")
		}
	}
	os.Setenv("ISITSQL_SESSION_KEY", a.SessionKey)

	return a, nil
}

func newEncryptedSessionKey() (string, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", errors.Wrap(err, "rand.read")
	}
	//fmt.Println("key:", key)
	secret, err := dpapi.EncryptBytes(key)
	if err != nil {
		return "", errors.Wrap(err, "dpapi.encryptbytes")
	}
	//fmt.Println("secret:", secret)
	b64 := base64.StdEncoding.EncodeToString(secret)

	//fmt.Println("str:", b64)
	//fmt.Println("saving...")
	return b64, nil
}

// DecryptSessionKey
func (a *AppConfig) DecryptSessionKey() ([]byte, error) {
	// bb, err := base64.StdEncoding.DecodeString(a.SessionKey)
	// if err != nil {
	// 	return []byte{}, errors.Wrap(err, "base64.stdencoding.decodestring")
	// }
	// key, err := dpapi.DecryptBytes(bb)
	// if err != nil {
	// 	return []byte{}, errors.Wrap(err, "dpapi.decryptbytes")
	// }
	// return key, nil
	return DecryptString(a.SessionKey)
}

// DecryptString decrypts a base64 encoded string using DPAPI
func DecryptString(s string) ([]byte, error) {
	bb, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return []byte{}, errors.Wrap(err, "base64.stdencoding.decodestring")
	}
	key, err := dpapi.DecryptBytes(bb)
	if err != nil {
		return []byte{}, errors.Wrap(err, "dpapi.decryptbytes")
	}
	return key, nil
}

// Validate checks for valid values
func (a *AppConfig) Validate() error {

	if a.PollWorkers < 0 {
		return errors.New("pollworks can't be negative")
	}

	if a.SecurityPolicy != OpenPolicy && a.SecurityPolicy != LocalHostPolicy /* && a.SecurityPolicy != DomainPolicy */ {
		return fmt.Errorf("security policy must be: open or localhost.  Found: %s", a.SecurityPolicy)
	}

	if a.Port < 1 || a.Port > 65535 {
		return fmt.Errorf("invalid port: %d", a.Port)
	}

	return nil
}
