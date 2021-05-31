// +build windows

package configurer

import (
	"os"
	"path/filepath"
	"syscall"
)

// DefaultWorkingDir returns the default config path for Windows OS.
func DefaultWorkingDir() string {
	homeDir, _ := os.UserHomeDir()
	appDir := beforeVistaAppDir

	v, _ := syscall.GetVersion()
	if v&0xff > 5 {
		appDir = sinceVistaAppDir
	}
	return filepath.Join(homeDir, filepath.FromSlash(appDir), "Pastel")
}

// DefaultZksnarkDir returns the default config path for Windows OS.
func DefaultZksnarkDir() string {
	homeDir, _ := os.UserHomeDir()
	appDir := beforeVistaAppDir

	v, _ := syscall.GetVersion()
	if v&0xff > 5 {
		appDir = sinceVistaAppDir
	}
	return filepath.Join(homeDir, filepath.FromSlash(appDir), "PastelParams")
}
