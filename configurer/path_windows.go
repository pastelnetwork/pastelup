// +build windows
// TODO: remove this file
package configurer

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/pastelnetwork/pastel-utility/constants"
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

	return filepath.Join(homeDir, filepath.FromSlash(getAppDir()), "Pastel"), nil
}

// DefaultZksnarkDir returns the default config path for Windows OS.
func DefaultZksnarkDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, filepath.FromSlash(getAppDir()), "PastelParams"), nil
}

// DefaultPastelExecutableDir returns the default pastel executable path for Windows OS.
func DefaultPastelExecutableDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, filepath.FromSlash(getAppDir()), "PastelWallet"), nil
}

// GetDownloadPath returns download path of the pastel executables.
func GetDownloadPath(version string, tool constants.ToolType, architectrue constants.ArchitectureType) string {
	t := "windows"
	if tool == constants.PastelD || tool == constants.RQService {
		t = "win"
	}

	return fmt.Sprintf("%s/%s/%s-%s-%s%s",
		constants.DownloadBaseURL,
		constants.GetVersionSubURL(version),
		tool,
		t,
		architectrue,
		".zip")
}
