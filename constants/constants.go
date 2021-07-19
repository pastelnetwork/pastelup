package constants

// OSType - Windows, Linux, MAC, Unknown
type OSType string

const (
	// Windows - Current OS is Windows
	Windows OSType = "Windows"
	// Linux - Current OS is Linux
	Linux OSType = "Linux"
	// Mac - Current OS is MAC
	Mac OSType = "MAC"
	// Unknown - Current OS is unknown
	Unknown OSType = "Unknown"

	// PastelConfName - pastel config file name
	PastelConfName string = "pastel.conf"

	// PastelUtilityConfigFilePath - The path of the config of pastel-utility
	PastelUtilityConfigFilePath string = "./pastel-utility.conf"
)

// PasteldName - The name of the pasteld
var PasteldName = map[OSType]string{
	Windows: "pasteld.exe",
	Linux:   "pasteld",
	Mac:     "pasteld",
	Unknown: "",
}

// PastelCliName - The name of the pastel-cli
var PastelCliName = map[OSType]string{
	Windows: "pastel-cli.exe",
	Linux:   "pastel-cli",
	Mac:     "pastel-cli",
	Unknown: "",
}

// PastelWalletExecName - The name of the pastel wallet node
var PastelWalletExecName = map[OSType]string{
	Windows: "walletnode-windows-amd64.exe",
	Linux:   "walletnode-linux-amd64",
	Mac:     "walletnode-darwin-amd64",
	Unknown: "",
}

// PastelSuperNodeExecName - The name of the pastel wallet node
var PastelSuperNodeExecName = map[OSType]string{
	Windows: "supernode-windows-amd64.exe",
	Linux:   "supernode-linux-amd64",
	Mac:     "supernode-darwin-amd64",
	Unknown: "",
}

// PastelExecArchiveName - The name of the pastel executable files
var PastelExecArchiveName = map[OSType]string{
	Windows: "pastel-win64-rc5.1.zip",
	Linux:   "pastel-ubuntu20.04-rc5.1.tar.gz",
	Mac:     "pastel-osx-rc5.1.tar.gz",
	Unknown: "",
}

// PastelDownloadURL - The download url of pastel executables
var PastelDownloadURL = map[OSType]string{
	Windows: "https://github.com/pastelnetwork/pastel/releases/download/v1.0-rc5.1/pastel-win64-rc5.1.zip",
	Linux:   "https://github.com/pastelnetwork/pastel/releases/download/v1.0-rc5.1/pastel-ubuntu20.04-rc5.1.tar.gz",
	Mac:     "https://github.com/pastelnetwork/pastel/releases/download/v1.0-rc5.1/pastel-osx-rc5.1.tar.gz",
	Unknown: "",
}

// PastelWalletDownloadURL - The download url of the pastel wallet node
var PastelWalletDownloadURL = map[OSType]string{
	Windows: "https://github.com/pastelnetwork/gonode/releases/download/v1.0-rc5.1/walletnode-windows-amd64",
	Linux:   "https://github.com/pastelnetwork/gonode/releases/download/v1.0-rc5.1/walletnode-linux-amd64",
	Mac:     "https://github.com/pastelnetwork/gonode/releases/download/v1.0-rc5.1/walletnode-darwin-amd64",
	Unknown: "",
}

// PastelSuperNodeDownloadURL - The download url of the pastel wallet node
var PastelSuperNodeDownloadURL = map[OSType]string{
	Windows: "https://github.com/pastelnetwork/gonode/releases/download/v1.0-rc5.1/supernode-linux-amd64",
	Linux:   "https://github.com/pastelnetwork/gonode/releases/download/v1.0-rc5.1/supernode-windows-amd64",
	Mac:     "https://github.com/pastelnetwork/gonode/releases/download/v1.0-rc5.1/supernode-darwin-amd64",
	Unknown: "",
}
