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

// DefaultPastelExecutableDir returns the default pastel executable path for Linux OS.
func DefaultPastelExecutableDir() string {
	homeDir, _ := os.UserConfigDir()
	return filepath.Join(homeDir, "Pastel")
}
