package configurer

import (
	"net/url"

	"github.com/pastelnetwork/pastel-utility/constants"
)

// IConfigurer returns a interface of Configurer
type IConfigurer interface {
	GetHomeDir() string
	DefaultWorkingDir() string
	DefaultSuperNodeLogFile() string
	DefaultWalletNodeLogFile() string
	DefaultZksnarkDir() string
	DefaultPastelExecutableDir() string
	GetDownloadURL(version string, tool constants.ToolType) (*url.URL, string, error)
}
