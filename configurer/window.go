// +build windows

package configurer

import "syscall"

const (
	beforeVistaAppDirWindows = "Application Data"
	sinceVistaAppDirWindows  = "AppData/Roaming"
)

func getAppDir() string {
	appDir := beforeVistaAppDirWindows

	v, _ := syscall.GetVersion()
	if v&0xff > 5 {
		appDir = sinceVistaAppDirWindows
	}
	return appDir
}
