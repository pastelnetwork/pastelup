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
	superNodeLogFile    string
	walletNodeLogFile   string
	superNodeConfFile   string
	walletNodeConfFile  string
	rqServiceConfFile   string
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

// DefaultSuperNodeLogFile returns the default supernode log file
func (c *configurer) GetSuperNodeLogFile(workingDir string) string {
	return filepath.Join(workingDir, c.superNodeLogFile)
}

// DefaultWalletNodeLogFile returns the default supernode log file
func (c *configurer) GetWalletNodeLogFile(workingDir string) string {
	return filepath.Join(workingDir, c.walletNodeLogFile)
}

// DefaultSuperNodeConfFile returns the default supernode log file
func (c *configurer) GetSuperNodeConfFile(workingDir string) string {
	return filepath.Join(workingDir, c.superNodeConfFile)
}

// DefaultWalletNodeConfFile returns the default supernode log file
func (c *configurer) GetWalletNodeConfFile(workingDir string) string {
	return filepath.Join(workingDir, c.walletNodeConfFile)
}

// GetRQServiceConfFile returns the default supernode log file
func (c *configurer) GetRQServiceConfFile(workingDir string) string {
	return filepath.Join(workingDir, c.rqServiceConfFile)
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
		return nil, "", errors.Errorf("failed to parse url: %v", err)
	}
	tokens := strings.Split(urlString, "/")

	return url, tokens[len(tokens)-1], nil
}

func (c *configurer) getDownloadGitPath(version string, tool constants.ToolType) (string, string, error) {
	var name string
	var baseLink string
	switch tool {
	case constants.WalletNode:
		name = constants.WalletNodeExecName[c.osType]
		tool = constants.GoNode
		baseLink = constants.GitReposURLBase[constants.WalletNode]
	case constants.RQService:
		return "", "", errors.New("not yet supported")
	case constants.PastelD:
		return "", "", errors.New("not yet supported")
	case constants.SuperNode:
		name = constants.SuperNodeExecName[c.osType]
		tool = constants.GoNode
		baseLink = constants.GitReposURLBase[constants.SuperNode]
	case constants.DDService:
		return "", "", errors.New("not yet supported")
	default:
		return "", "", errors.Errorf("unknow tool: %s", tool)
	}

	urlStringBase := fmt.Sprintf(
		"%s/%s",
		baseLink,
		version)
	return urlStringBase, name, nil
}

// GetDownloadGitURL returns download url of the pastel executables in git
func (c *configurer) GetDownloadGitURL(version string, tool constants.ToolType) (*url.URL, string, error) {
	urlBase, name, err := c.getDownloadGitPath(version, tool)
	if err != nil {
		return nil, "", errors.Errorf("get path failed : %v", err)
	}

	urlString := fmt.Sprintf(
		"%s/%s",
		urlBase,
		name)
	url, err := url.Parse(urlString)
	if err != nil {
		return nil, "", errors.Errorf("failed to parse url: %v", err)
	}

	tokens := strings.Split(urlString, "/")

	return url, tokens[len(tokens)-1], nil
}

// GetDownloadGitcheckSumURL returns checksum url of the pastel executable in git
func (c *configurer) GetDownloadGitcheckSumURL(version string, tool constants.ToolType) (*url.URL, string, error) {
	urlBase, name, err := c.getDownloadGitPath(version, tool)
	if err != nil {
		return nil, "", errors.Errorf("get path failed : %v", err)
	}

	var urlCheckSumString string
	if !strings.Contains(name, ".exe") {
		urlCheckSumString = fmt.Sprintf(
			"%s/%s",
			urlBase,
			name+".sha256")
	} else {
		name = strings.Replace(name, ".exe", ".sha256", -1)
		urlCheckSumString = fmt.Sprintf(
			"%s/%s",
			version,
			name)
	}

	checkSumURL, err := url.Parse(urlCheckSumString)
	if err != nil {
		return nil, "", errors.Errorf("failed to parse checksum url: %v", err)
	}

	tokens := strings.Split(urlCheckSumString, "/")

	return checkSumURL, tokens[len(tokens)-1], nil
}

func newLinuxConfigurer(homeDir string) IConfigurer {
	return &configurer{
		workingDir:          ".pastel",
		superNodeLogFile:    "supernode.log",
		walletNodeLogFile:   "walletnode.log",
		superNodeConfFile:   "supernode.yml",
		walletNodeConfFile:  "walletnode.yml",
		rqServiceConfFile:   "rqservice.toml",
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
		superNodeLogFile:    "supernode.log",
		walletNodeLogFile:   "walletnode.log",
		superNodeConfFile:   "supernode.yml",
		walletNodeConfFile:  "walletnode.yml",
		rqServiceConfFile:   "rqservice.toml",
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
		superNodeLogFile:    "supernode.log",
		walletNodeLogFile:   "walletnode.log",
		superNodeConfFile:   "supernode.yml",
		walletNodeConfFile:  "walletnode.yml",
		rqServiceConfFile:   "rqservice.toml",
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
