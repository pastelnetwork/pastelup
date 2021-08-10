// +build darwin

package configurer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pastelnetwork/pastel-utility/constants"
)

// GetHomeDir returns the home path for darwin OS.
func GetHomeDir() (string, error) {
	if homeDir, err := os.UserConfigDir(); err != nil {
		return "", err
	}
	return homeDir, nil
}

// DefaultWorkingDir returns the default config path for darwin OS.
func DefaultWorkingDir() (string, error) {
	if homeDir, err := os.UserConfigDir(); err != nil {
		return "", err
	}
	return filepath.Join(homeDir, "Pastel"), nil
}

// DefaultZksnarkDir returns the default config path for darwin OS.
func DefaultZksnarkDir() (string, error) {
	if homeDir, err := os.UserConfigDir(); err != nil {
		return "", err
	}
	return filepath.Join(homeDir, "PastelParams"), nil
}

// DefaultPastelExecutableDir returns the default pastel executable path for darwin OS.
func DefaultPastelExecutableDir() (string, error) {
	if homeDir, err := os.UserConfigDir(); err != nil {
		return "", err
	}
	return filepath.Join(homeDir, "Pastel"), nil
}

// GetDownloadPath returns download path of the pastel executables.
func GetDownloadPath(version string, tool constants.ToolType, architectrue constants.ArchitectureType) string {

	versionSubURL := constants.GetVersionSubURL(version)

	return fmt.Sprintf("%s/%s/%s-%s-%s%s", constants.DownloadBaseURL, versionSubURL, tool, "darwin", architectrue, ".zip")

}
