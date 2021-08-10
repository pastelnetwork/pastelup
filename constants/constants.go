package constants

import "fmt"

// OSType - Windows, Linux, MAC, Unknown
type OSType string

// ToolType - gonode, pasteld, dd-service, rq-service
type ToolType string

// ArchitectureType - amd64
type ArchitectureType string

const (
	// DownloadBaseURL - The base URL of the pastel release files
	DownloadBaseURL string = "https://download.pastel.network"

	// PastelConfName - pastel config file name
	PastelConfName string = "pastel.conf"

	// PastelUtilityConfigFilePath - The path of the config of pastel-utility
	PastelUtilityConfigFilePath string = "./pastel-utility.conf"

	// PastelUtilityLogFilePath - The path of the log of pastel-utility
	PastelUtilityLogFilePath string = "./pastel-utility-remote-log.txt"

	// PipRequirmentsFileName - pip install requirements file name
	PipRequirmentsFileName string = "requirements.txt"

	// DupeDetectionImageFingerPrintDataBase - dupe_detection_image_fingerprint_database file
	DupeDetectionImageFingerPrintDataBase string = "dupe_detection_image_fingerprint_database.sqlite"

	// PastelUtilityDownloadURL - The path of pastel-utility for install supernode remote
	PastelUtilityDownloadURL string = "https://github.com/pastelnetwork/pastel-utility/releases/download/v0.5.8/pastel-utility-linux-amd64"

	// RequirementDownloadURL - The path of requirement.txt for install pip
	RequirementDownloadURL string = "https://download.pastel.network/machine-learning/requirements.txt"
)

// Windows type
const (
	Windows OSType = "Windows"
	Linux   OSType = "Linux"
	Mac     OSType = "MAC"
	Unknown OSType = "Unknown"
)

// Tool type
const (
	WalletNode ToolType = "gonode/walletnode"
	SuperNode  ToolType = "gonode/supernode"
	PastelD    ToolType = "pasteld/pastel"
	DDService  ToolType = "dd-service/ddservice"
	RQService  ToolType = "rq-service/rqservice"
)

// Architecture type
const (
	AMD64 ArchitectureType = "amd64"
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

// WalletNodeExecName - The name of the wallet node
var WalletNodeExecName = map[OSType]string{
	Windows: "walletnode-windows-amd64.exe",
	Linux:   "walletnode-linux-amd64",
	Mac:     "walletnode-darwin-amd64",
	Unknown: "",
}

// SuperNodeExecName - The name of the pastel wallet node
var SuperNodeExecName = map[OSType]string{
	Windows: "supernode-windows-amd64.exe",
	Linux:   "supernode-linux-amd64",
	Mac:     "supernode-darwin-amd64",
	Unknown: "",
}

// PastelExecArchiveName - The name of the pastel executable files
var PastelExecArchiveName = map[OSType]string{
	Windows: "pastel-win-amd64.zip",
	Linux:   "pastel-ubuntu20.04-amd64.zip",
	Mac:     "pastel-darwin-amd64.zip",
	Unknown: "",
}

// WalletNodeExecArchiveName - The download url of the  wallet node file
var WalletNodeExecArchiveName = map[OSType]string{
	Windows: "walletnode-windows-amd64.zip",
	Linux:   "walletnode-linux-amd64.zip",
	Mac:     "walletnode-darwin-amd64.zip",
	Unknown: "",
}

// SupperNodeExecArchiveName - The download url of the  super node file
var SupperNodeExecArchiveName = map[OSType]string{
	Windows: "supernode-windows-amd64.zip",
	Linux:   "supernode-linux-amd64.zip",
	Mac:     "supernode-darwin-amd64.zip",
	Unknown: "",
}

// PastelParamsCheckSums - CheckSum of pastel param files , to avoid download if exists
var PastelParamsCheckSums = map[string]string{
	"sapling-output.params": "2f0ebbcbb9bb0bcffe95a397e7eba89c29eb4dde6191c339db88570e3f3fb0e4",
	"sapling-spend.params":  "8e48ffd23abb3a5fd9c5589204f32d9c31285a04b78096ba40a79b75677efc13",
	"sprout-groth16.params": "b685d700c60328498fbde589c8c7c484c722b788b265b72af448a5bf0ee55b50",
	"sprout-proving.key":    "8bc20a7f013b2b58970cddd2e7ea028975c88ae7ceb9259a5344a16bc2c0eef7",
	"sprout-verifying.key":  "4bd498dae0aacfd8e98dc306338d017d9c08dd0918ead18172bd0aec2fc5df82",
}

// ChromeDownloadURL - The download url of chrome
var ChromeDownloadURL = map[OSType]string{
	Windows: "",
	Linux:   "https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb",
	Mac:     "",
	Unknown: "",
}

// ChromeExecFileName - The download filename of chrome executable
var ChromeExecFileName = map[OSType]string{
	Windows: "",
	Linux:   "google-chrome.deb",
	Mac:     "",
	Unknown: "",
}

// PortList - PortList to open in install supernode
var PortList = []string{"9933", "19933", "4444", "14444"}

// PastelRQServiceArchiveName - The name of the pastel rqservice files
var PastelRQServiceArchiveName = map[OSType]string{
	Windows: "rqservice-win-amd64.zip",
	Linux:   "rqservice-ubuntu20.04-amd64.zip",
	Mac:     "rqservice-darwin-amd64.zip",
	Unknown: "",
}

// PastelRQServiceExecName - The name of the rqservice executable files
var PastelRQServiceExecName = map[OSType]string{
	Windows: "rq-service-win10-x64.exe",
	Linux:   "rq-service-ubuntu20.04-x64",
	Mac:     "rq-service-macosx",
	Unknown: "",
}

// DupeDetectionConfigs - dupe detection path of the supernode config
var DupeDetectionConfigs = []string{
	"dupe_detection_input_files",
	"dupe_detection_support_files",
	"dupe_detection_output_files",
	"dupe_detection_processed_files",
	"dupe_detection_rare_on_internet",
	"mobilenet_v2_140_224",
}

// DupeDetectionSupportDownloadURL - The URL of dupe detection support files
var DupeDetectionSupportDownloadURL = []string{
	"https://download.pastel.network/machine-learning/dupe_detection_image_fingerprint_database.zip",
	"https://download.pastel.network/machine-learning/keras_dupe_classifier.model.zip",
	"https://download.pastel.network/machine-learning/xgboost_dupe_classifier.zip",
	"https://download.pastel.network/machine-learning/nsfw_mobilenet_v2_140_224.zip",
}

// DupeDetectionMobileNetDownloadURL - The URL of dupe detection mobile net files
var DupeDetectionMobileNetDownloadURL = "https://download.pastel.network/machine-learning/nsfw_mobilenet_v2_140_224.zip"

// DupeDetectionSupportFilePath - The target path for downloading dupe detection support files
var DupeDetectionSupportFilePath = "dupe_detection_support_files"

// GetVersionSubURL returns the sub url concerned with version info
func GetVersionSubURL(version string) string {
	switch version {
	case "latest":
		return version
	case "beta":
		return version
	default:
		return fmt.Sprintf("history/%s", version)
	}
}
