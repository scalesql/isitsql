package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kardianos/osext"
	"github.com/pkg/errors"
)

// GetCachedServer hydrates a server from cache.  It logs most errors
// and returns an empty SqlServer
func GetCachedServer(key string) SqlServer {
	wd, err := osext.ExecutableFolder()
	if err != nil {
		WinLogln(errors.Wrap(err, "executableFolder"))
		return SqlServer{}
	}

	dir := filepath.Join(wd, "cache")
	fileName := filepath.Join(dir, fmt.Sprintf("server.%s.json", key))

	// if the file doesn't exist, then return emty SqlServer with no error
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return SqlServer{}
	}

	/* #nosec G304 */
	fileBody, err := os.ReadFile(fileName)
	if err != nil {
		WinLogln(errors.Wrapf(err, "os.readfile: %s", key))
		return SqlServer{}
	}

	var s SqlServer
	err = json.Unmarshal(fileBody, &s)
	if err != nil {
		WinLogln(errors.Wrapf(err, "json.unmarshal: %s", key))
		return s
	}

	// TODO if more than 1 hour old, return nil
	if time.Since(s.LastBigPoll) > time.Duration(time.Hour) {
		return SqlServer{}
	}
	s.ResetOnThisPoll = true
	s.IsPolling = false
	s.PollActivity = ""
	s.LastPollError = ""
	//WinLogln(fmt.Sprintf("read cached server: %s", key))
	return s
}
