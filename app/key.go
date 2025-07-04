package app

import (
	"os"
	"path/filepath"

	"github.com/scalesql/isitsql/internal/gui"
	"github.com/kardianos/osext"
)

func setUseLocalStatic() {
	dir, err := osext.ExecutableFolder()
	if err != nil {
		WinLogln("setUseLocalStatic: Unable to determine EXEC folder: ", err)
		return
	}

	globalConfig.Lock()
	defer globalConfig.Unlock()
	current := globalConfig.AppConfig.UseLocalStatic
	var new bool

	fullfile := filepath.Join(dir, "uselocal.txt")
	if _, err := os.Stat(fullfile); err == nil {
		new = true
	} else {
		new = false
	}

	if current != new {
		WinLogln("Use Local Static Files: ", new)
	}

	globalConfig.AppConfig.UseLocalStatic = new
	gui.SetUseLocal(new)
}
