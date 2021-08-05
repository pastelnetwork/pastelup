// +build windows

package configurer

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/pastelnetwork/pastel-utility/constants"
)

const (
	beforeVistaAppDir = "Application Data"
	sinceVistaAppDir  = "AppData/Roaming"
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

// DefaultPastelExecutableDir returns the default pastel executable path for Windows OS.
func DefaultPastelExecutableDir() string {
	homeDir, _ := os.UserHomeDir()
	appDir := beforeVistaAppDir

	v, _ := syscall.GetVersion()
	if v&0xff > 5 {
		appDir = sinceVistaAppDir
	}
	return filepath.Join(homeDir, filepath.FromSlash(appDir), "PastelWallet")
}

// GetDownloadPath returns download path of the pastel executables.
func GetDownloadPath(version string, tool constants.ToolType, architectrue constants.ArchitectureType) string {
	var ret string

	versionSubURL := constants.GetVersionSubURL(version)

	if tool == constants.PastelD {
		ret = fmt.Sprintf("%s/%s/%s-%s-%s%s", constants.DownloadBaseURL, versionSubURL, tool, "win", architectrue, ".zip")
	} else if tool == constants.SuperNode || tool == constants.WalletNode {
		ret = fmt.Sprintf("%s/%s/%s-%s-%s%s", constants.DownloadBaseURL, versionSubURL, tool, "windows", architectrue, ".zip")
	} else if tool == constants.RQService {
		ret = fmt.Sprintf("%s/%s/%s-%s-%s%s", constants.DownloadBaseURL, versionSubURL, tool, "win10", "x64", ".zip")
	}

	return ret
}
