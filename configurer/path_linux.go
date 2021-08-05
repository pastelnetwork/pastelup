// +build linux

package configurer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pastelnetwork/pastel-utility/constants"
)

// DefaultWorkingDir returns the default config path for Linux OS.
func DefaultWorkingDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".pastel")
}

// DefaultZksnarkDir returns the default config path for Linux OS.
func DefaultZksnarkDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".pastel-params")
}

// DefaultPastelExecutableDir returns the default pastel executable path for Linux OS.
func DefaultPastelExecutableDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, "pastel")
}

// GetDownloadPath returns download path of the pastel executables.
func GetDownloadPath(version string, tool constants.ToolType, architectrue constants.ArchitectureType) string {
	var ret string

	versionSubURL := constants.GetVersionSubURL(version)
	if tool == constants.PastelD {
		ret = fmt.Sprintf("%s/%s/%s-%s-%s%s", constants.DownloadBaseURL, versionSubURL, tool, "ubuntu20.04", architectrue, ".zip")
	} else if tool == constants.SuperNode || tool == constants.WalletNode {
		ret = fmt.Sprintf("%s/%s/%s-%s-%s%s", constants.DownloadBaseURL, versionSubURL, tool, "linux", architectrue, ".zip")
	} else if tool == constants.RQService {
		ret = fmt.Sprintf("%s/%s/%s-%s%s", constants.DownloadBaseURL, versionSubURL, tool, "ubuntu20", ".zip")
	}

	return ret
}
