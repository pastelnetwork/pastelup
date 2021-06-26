package constants

// OSType - Windows, Linux, MAC, Unknown
type OSType string

const (
	OS_WINDOWS OSType = "Windows"
	OS_LINUX   OSType = "Linux"
	OS_MAC     OSType = "MAC"
	OS_UNKNOWN OSType = "Unknown"

	PASTEL_CONF_NAME string = "pastel.conf"

	PASTEL_UTILITY_CONFIG_FILE_PATH string = "./pastel-utility.conf"
)

var PASTELD_NAME = map[OSType]string{
	OS_WINDOWS: "pasteld.exe",
	OS_LINUX:   "pasteld",
	OS_MAC:     "pasteld",
	OS_UNKNOWN: "",
}

var PASTEL_CLI_NAME = map[OSType]string{
	OS_WINDOWS: "pastel-cli.exe",
	OS_LINUX:   "pastel-cli",
	OS_MAC:     "pastel-cli",
	OS_UNKNOWN: "",
}

var PASTEL_WALLET_EXEC_NAME = map[OSType]string{
	OS_WINDOWS: "walletnode-windows-amd64.exe",
	OS_LINUX:   "walletnode-linux-amd64",
	OS_MAC:     "walletnode-darwin-amd64",
	OS_UNKNOWN: "",
}

var PASTEL_EXEC_ARCHIVE_NAME = map[OSType]string{
	OS_WINDOWS: "pastel-win64-rc5.1.zip",
	OS_LINUX:   "pastel-ubuntu20.04-rc5.1.tar.gz",
	OS_MAC:     "pastel-osx-rc5.1.tar.gz",
	OS_UNKNOWN: "",
}

var PASTEL_DOWNLOAD_URL = map[OSType]string{
	OS_WINDOWS: "https://github.com/pastelnetwork/pastel/releases/download/v1.0-rc5.1/pastel-win64-rc5.1.zip",
	OS_LINUX:   "https://github.com/pastelnetwork/pastel/releases/download/v1.0-rc5.1/pastel-ubuntu20.04-rc5.1.tar.gz",
	OS_MAC:     "https://github.com/pastelnetwork/pastel/releases/download/v1.0-rc5.1/pastel-osx-rc5.1.tar.gz",
	OS_UNKNOWN: "",
}

var PASTEL_WALLET_DOWNLOAD_URL = map[OSType]string{
	OS_WINDOWS: "https://github.com/pastelnetwork/gonode/releases/download/v1.0-rc5.1/walletnode-windows-amd64",
	OS_LINUX:   "https://github.com/pastelnetwork/gonode/releases/download/v1.0-rc5.1/walletnode-linux-amd64",
	OS_MAC:     "https://github.com/pastelnetwork/gonode/releases/download/v1.0-rc5.1/walletnode-darwin-amd64",
	OS_UNKNOWN: "",
}
