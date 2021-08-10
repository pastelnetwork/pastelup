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

// GetHomeDir returns the home path for Windows OS.
func GetHomeDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return homeDir, nil
}

// DefaultWorkingDir returns the default config path for Windows OS.
func DefaultWorkingDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	appDir := beforeVistaAppDir

	v, _ := syscall.GetVersion()
	if v&0xff > 5 {
		appDir = sinceVistaAppDir
	}

	return filepath.Join(homeDir, filepath.FromSlash(appDir), "Pastel"), nil
}

// DefaultZksnarkDir returns the default config path for Windows OS.
func DefaultZksnarkDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	appDir := beforeVistaAppDir

	v, _ := syscall.GetVersion()
	if v&0xff > 5 {
		appDir = sinceVistaAppDir
	}

	return filepath.Join(homeDir, filepath.FromSlash(appDir), "PastelParams"), nil
}

// DefaultPastelExecutableDir returns the default pastel executable path for Windows OS.
func DefaultPastelExecutableDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	appDir := beforeVistaAppDir

	v, _ := syscall.GetVersion()
	if v&0xff > 5 {
		appDir = sinceVistaAppDir
	}
	return filepath.Join(homeDir, filepath.FromSlash(appDir), "PastelWallet"), nil
}

// GetDownloadPath returns download path of the pastel executables.
func GetDownloadPath(version string, tool constants.ToolType, architectrue constants.ArchitectureType) string {

	versionSubURL := constants.GetVersionSubURL(version)

	if tool == constants.PastelD || tool == constants.RQService {
		return fmt.Sprintf("%s/%s/%s-%s-%s%s", constants.DownloadBaseURL, versionSubURL, tool, "win", architectrue, ".zip")
	}

	return fmt.Sprintf("%s/%s/%s-%s-%s%s", constants.DownloadBaseURL, versionSubURL, tool, "windows", architectrue, ".zip")

}
