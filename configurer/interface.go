package configurer

import (
	"net/url"

	"github.com/pastelnetwork/pastel-utility/constants"
)

type IConfigurer interface {
	GetHomeDir() string
	DefaultWorkingDir() string
	DefaultZksnarkDir() string
	DefaultPastelExecutableDir() string
	GetDownloadURL(version string, tool constants.ToolType) (*url.URL, string, error)
}
