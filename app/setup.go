package app

import "github.com/scalesql/isitsql/internal/c2"

//
type ConfigModeType int

const (
	ModeGUI ConfigModeType = iota
	ModeFile
)

var AppConfigMode ConfigModeType = ModeGUI

func (cm ConfigModeType) String() string {
	switch cm {
	case ModeGUI:
		return "GUI"
	case ModeFile:
		return "FILE"
	}
	return "unknown"
}

// SetConfigMode sets the configuration mode of the application.
// If it finds any HCL files, it sets it to file
// Otherwise it leaves it as GUI.
func SetConfigMode() {
	files, err := c2.FindHCLFiles()
	if err != nil {
		WinLogf("setconfigmode: %s", err.Error())
	}
	if len(files) > 0 {
		AppConfigMode = ModeFile
	}
	WinLogf("config mode: %s", AppConfigMode)
}
