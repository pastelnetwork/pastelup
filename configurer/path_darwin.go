// +build darwin

package configurer

import (
	"os"
	"path/filepath"
)

// DefaultWorkingDir returns the default config path for darwin OS.
func DefaultWorkingDir() string {
	homeDir, _ := os.UserConfigDir()
	return filepath.Join(homeDir, "Pastel")
}

// DefaultZksnarkDir returns the default config path for darwin OS.
func DefaultZksnarkDir() string {
	homeDir, _ := os.UserConfigDir()
	return filepath.Join(homeDir, "PastelParams")
}

// DefaultPastelExecutableDir returns the default pastel executable path for darwin OS.
func DefaultPastelExecutableDir() string {
	homeDir, _ := os.UserConfigDir()
	return filepath.Join(homeDir, "Pastel")
}

// GetDownloadPath returns download path of the pastel executables.
func GetDownloadPath(version string, tool constants.ToolType, architectrue constants.ArchitectureType) string {
	var ret string

	versionSubURL := constants.GetVersionSubURL(version)

	if tool == constants.RQService {
		ret = fmt.Sprintf("%s/%s/%s-%s%s", constants.DownloadBaseURL, versionSubURL, tool, "macosx", ".zip")
	} else {
		ret = fmt.Sprintf("%s/%s/%s-%s-%s%s", constants.DownloadBaseURL, versionSubURL, tool, "darwin", architectrue, ".zip")
	}

	return ret
}
