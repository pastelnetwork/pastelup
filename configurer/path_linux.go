// +build linux

package configurer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pastelnetwork/pastel-utility/constants"
)

// GetHomeDir returns the home path for Linux OS.
func GetHomeDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return homeDir, nil
}

// DefaultWorkingDir returns the default config path for Linux OS.
func DefaultWorkingDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".pastel"), nil
}

// DefaultZksnarkDir returns the default config path for Linux OS.
func DefaultZksnarkDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".pastel-params"), nil
}

// DefaultPastelExecutableDir returns the default pastel executable path for Linux OS.
func DefaultPastelExecutableDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, "pastel"), nil
}

// GetDownloadPath returns download path of the pastel executables.
func GetDownloadPath(version string, tool constants.ToolType, architectrue constants.ArchitectureType) string {
	t := "linux"
	if tool == constants.PastelD || tool == constants.RQService {
		t = "ubuntu20.04"
	}

	return fmt.Sprintf("%s/%s/%s-%s-%s%s",
		constants.DownloadBaseURL,
		constants.GetVersionSubURL(version),
		tool,
		t,
		architectrue,
		".zip")
}
