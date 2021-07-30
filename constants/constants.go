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

	// PipRequirmentsFileName - pip install requirements file name
	PipRequirmentsFileName string = "requirements.txt"

	// DupeDetectionImageFingerPrintDataBase - dupe_detection_image_fingerprint_database file
	DupeDetectionImageFingerPrintDataBase string = "dupe_detection_image_fingerprint_database.sqlite"
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
	Windows: "https://download.pastel.network/latest/pasteld/pastel-win64-rc5.1.zip",
	Linux:   "https://download.pastel.network/latest/pasteld/pastel-ubuntu20.04-rc5.1.tar.gz",
	Mac:     "https://download.pastel.network/latest/pasteld/pastel-osx-rc5.1.tar.gz",
	Unknown: "",
}

// PastelDownloadReleaseURL - The download url of pastel executables for release
var PastelDownloadReleaseURL = "https://download.pastel.network/pasteld/"

// PastelDownloadReleaseFileName - The download filename of pastel executables for release
var PastelDownloadReleaseFileName = map[OSType]string{
	Windows: "pastel-win64-",
	Linux:   "pastel-ubuntu20.04-",
	Mac:     "pastel-osx-",
	Unknown: "",
}

// PastelDownloadReleaseFileExtension - The download file extension of pastel executables for release
var PastelDownloadReleaseFileExtension = map[OSType]string{
	Windows: ".zip",
	Linux:   ".tar.gz",
	Mac:     ".tar.gz",
	Unknown: "",
}

// PastelWalletDownloadURL - The download url of the pastel wallet node
var PastelWalletDownloadURL = map[OSType]string{
	Windows: "https://download.pastel.network/latest/gonode/walletnode-windows-amd64.zip",
	Linux:   "https://download.pastel.network/latest/gonode/walletnode-linux-amd64.zip",
	Mac:     "https://download.pastel.network/latest/gonode/walletnode-darwin-amd64.zip",
	Unknown: "",
}

// PastelSuperNodeDownloadURL - The download url of the pastel super node
var PastelSuperNodeDownloadURL = map[OSType]string{
	Windows: "https://download.pastel.network/latest/gonode/supernode-windows-amd64.zip",
	Linux:   "https://download.pastel.network/latest/gonode/supernode-linux-amd64.zip",
	Mac:     "https://download.pastel.network/latest/gonode/supernode-darwin-amd64.zip",
	Unknown: "",
}

// PastelWalletSuperReleaseDownloadURL - The download url of the pastel wallet, super node for release
var PastelWalletSuperReleaseDownloadURL = "https://download.pastel.network/gonode/"

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
	Windows: "https://download.pastel.network/latest/pasteld/pastel-win64-rc5.1.zip",
	Linux:   "https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb",
	Mac:     "https://download.pastel.network/latest/pasteld/pastel-osx-rc5.1.tar.gz",
	Unknown: "",
}

// ChromeExecFileName - The download filename of chrome executable
var ChromeExecFileName = map[OSType]string{
	Windows: "pastel-win64-",
	Linux:   "google-chrome.deb",
	Mac:     "pastel-osx-",
	Unknown: "",
}

// PortList - PortList to open in install supernode
var PortList = []string{"9933", "19933", "4444", "14444"}
