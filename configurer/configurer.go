package configurer

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-errors/errors"
	"github.com/pastelnetwork/pastel-utility/constants"
	"github.com/pastelnetwork/pastel-utility/utils"
)

const (
	templateDownloadURL = constants.DownloadBaseURL + "/%s/%s/%s"
)

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
func (c *configurer) GetDownloadURL(version string, tool constants.ToolType) (*url.URL, string, error) {
	var name string
	switch tool {
	case constants.WalletNode:
		name = constants.WalletNodeExecName[c.osType]
		tool = constants.GoNode
	case constants.RQService:
		name = constants.PastelRQServiceExecName[c.osType]
	case constants.PastelD:
		name = constants.PastelExecArchiveName[c.osType]
	case constants.SuperNode:
		name = constants.SuperNodeExecName[c.osType]
		tool = constants.GoNode
	case constants.DDService:
		name = constants.DupeDetectionExecName
	default:
		return nil, "", errors.Errorf("unknow tool: %s", tool)
	}

	urlString := fmt.Sprintf(
		templateDownloadURL,
		constants.GetVersionSubURL(version),
		tool,
		name)

	url, err := url.Parse(urlString)
	if err != nil {
		return nil, "", errors.Errorf("failed to parse url, err: %s", err)
	}
	tokens := strings.Split(urlString, "/")

	return url, tokens[len(tokens)-1], nil
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
	osType := utils.GetOS()
	switch osType {
	case constants.Linux:
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, errors.Errorf("failed to get user home dir: %v", err)
		}
		return newLinuxConfigurer(homeDir), nil
	case constants.Mac:
		homeDir, err := os.UserConfigDir()
		if err != nil {
			return nil, errors.Errorf("failed to get user config dir dir: %v", err)
		}
		return newDarwinConfigurer(homeDir), nil
	case constants.Windows:
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, errors.Errorf("failed to get user home dir: %v", err)
		}
		return newWindowsConfigurer(homeDir), nil
	default:
		return nil, errors.New("unknown os")
	}
}
