package configurer

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/pastelnetwork/pastel-utility/constants"
	"github.com/pastelnetwork/pastel-utility/utils"
)

const templateDownloadURL = constants.DownloadBaseURL + "/%s/%s-%s-%s.zip"

type configurer struct {
	workingDir          string
	zksnarkDir          string
	pastelExecutableDir string
	homeDir             string
	architecture        constants.ArchitectureType
	osType              constants.OSType
}

// GetHomeDir returns the home path.
func (c *configurer) GetHomeDir() string {
	return c.homeDir
}

// DefaultWorkingDir returns the default config path.
func (c *configurer) DefaultWorkingDir() string {
	if c.osType == constants.Windows {
		return filepath.Join(c.homeDir, filepath.FromSlash(getAppDir()), c.workingDir)
	}
	return filepath.Join(c.homeDir, c.workingDir)
}

// DefaultZksnarkDir returns the default config path.
func (c *configurer) DefaultZksnarkDir() string {
	if c.osType == constants.Windows {
		return filepath.Join(c.homeDir, filepath.FromSlash(getAppDir()), c.zksnarkDir)
	}
	return filepath.Join(c.homeDir, c.zksnarkDir)
}

// DefaultPastelExecutableDir returns the default pastel executable path.
func (c *configurer) DefaultPastelExecutableDir() string {
	if c.osType == constants.Windows {
		return filepath.Join(c.homeDir, filepath.FromSlash(getAppDir()), c.pastelExecutableDir)
	}
	return filepath.Join(c.homeDir, c.pastelExecutableDir)
}

// GetDownloadURL returns download url of the pastel executables.
func (c *configurer) GetDownloadURL(version string, tool constants.ToolType) (*url.URL, error) {
	var toolType string
	switch c.osType {
	case constants.Mac:
		toolType = "darwin"
	case constants.Linux:
		toolType = "linux"
		if tool == constants.PastelD || tool == constants.RQService {
			toolType = "ubuntu20.04"
		}
	case constants.Windows:
		toolType = "windows"
		if tool == constants.PastelD || tool == constants.RQService {
			toolType = "win"
		}
	}

	url, err := url.Parse(fmt.Sprintf(
		templateDownloadURL,
		constants.GetVersionSubURL(version),
		tool,
		toolType,
		c.architecture))
	if err != nil {
		return nil, errors.Errorf("failed to parse url, err: %s", err)
	}
	return url, nil
}

func newLinuxConfigurer(homeDir string) IConfigurer {
	return &configurer{
		workingDir:          ".pastel",
		zksnarkDir:          ".pastel-params",
		pastelExecutableDir: "pastel",
		homeDir:             homeDir,
		architecture:        constants.AMD64,
		osType:              constants.Linux,
	}
}

func newDarwinConfigurer(homeDir string) IConfigurer {
	return &configurer{
		workingDir:          "Pastel",
		zksnarkDir:          "PastelParams",
		pastelExecutableDir: "Pastel",
		homeDir:             homeDir,
		architecture:        constants.AMD64,
		osType:              constants.Mac,
	}
}

func newWindowsConfigurer(homeDir string) IConfigurer {
	return &configurer{
		workingDir:          "Pastel",
		zksnarkDir:          "PastelParams",
		pastelExecutableDir: "PastelWallet",
		homeDir:             homeDir,
		architecture:        constants.AMD64,
		osType:              constants.Windows,
	}
}

// NewConfigurer return a new configurer instance
func NewConfigurer() (IConfigurer, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.Errorf("failed to get user home dir, err: %s", err)
	}

	osType := utils.GetOS()
	switch osType {
	case constants.Linux:
		return newLinuxConfigurer(homeDir), nil
	case constants.Mac:
		return newDarwinConfigurer(homeDir), nil
	case constants.Windows:
		return newWindowsConfigurer(homeDir), nil
	default:
		return nil, errors.New("unknown os")
	}
}
