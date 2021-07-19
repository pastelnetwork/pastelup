// +build linux

package configurer

import (
	"os"
	"path/filepath"
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
	return filepath.Join(homeDir, ".pastel")
}
