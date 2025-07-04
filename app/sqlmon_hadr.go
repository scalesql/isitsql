package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func HadrCheckNamesFile() error {
	exe, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "os.executable")
	}
	wd := filepath.Dir(exe)
	agfile := filepath.Join(wd, "config", "ag_names.csv")

	// if the file doesn't exist, then write it
	if _, err := os.Stat(agfile); os.IsNotExist(err) {

		FileHeader := `#########################################################################
#
# This file maps Availability Groups to friendly display names
# The first column is the domain
# The second column is the AG Name or Listener name
# The third column is the Display Name
#
# * A duplicate will override a previous entry
# * Blank lines are ignored
# * A line starting with # is a comment
#
# This file is read each time it is updated
#
########################################################################

# Sample Entries
# Domain, Listener, db-txn.static.loc

`
		FileHeader = strings.Replace(FileHeader, "\n", "\r\n", -1)
		err = os.WriteFile(agfile, []byte(FileHeader), 0660)
		if err != nil {
			WinLogln("Error writing waits.txt", err)
			return err
		}
		WinLogln("/config/ag_names.csv created.")
	}

	return nil
}
