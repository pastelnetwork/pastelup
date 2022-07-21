package constants

import (
	"fmt"
	"path/filepath"
)

// OSType - Windows, Linux, MAC, Unknown
type OSType string

// ToolType - gonode, pasteld, ddservice, rqservice
type ToolType string

// ArchitectureType - amd64
type ArchitectureType string

const (
	// DownloadBaseURL - The base URL of the pastel release files
	DownloadBaseURL string = "https://download.pastel.network"

	// PastelConfName - pastel config file name
	PastelConfName string = "pastel.conf"

	// PastelUtilityConfigFilePath - The path of the config of pastelup
	PastelUtilityConfigFilePath string = "./pastelup.conf"

	// PastelUtilityLogFilePath - The path of the log of pastelup
	PastelUtilityLogFilePath string = "./pastelup-remote-log.txt"

	// PipRequirmentsFileName - pip install requirements file name
	PipRequirmentsFileName string = "requirements.txt"

	// DupeDetectionImageFingerPrintDataBase - dupe_detection_image_fingerprint_database file
	DupeDetectionImageFingerPrintDataBase string = "dupe_detection_image_fingerprint_database.sqlite"

	// Windows type
	Windows OSType = "Windows"
	// Linux type
	Linux OSType = "Linux"
	// Mac type
	Mac OSType = "MAC"
	// Unknown type
	Unknown OSType = "Unknown"

	// Pastelup type
	Pastelup ToolType = "pastelup"
	// WalletNode type
	WalletNode ToolType = "walletnode"
	// SuperNode type
	SuperNode ToolType = "supernode"
	// GoNode type
	GoNode ToolType = "gonode"
	// PastelD type
	PastelD ToolType = "pasteld"
	// DDService type
	DDService ToolType = "dd-service"
	// RQService type
	RQService ToolType = "rq-service"
	// DDImgService type
	DDImgService ToolType = "dd-img-server"
	// AMD64 is architecture type
	AMD64 ArchitectureType = "amd64"
	// DupeDetectionArchiveName is archive name for dupe detection
	DupeDetectionArchiveName = "dupe-detection-server.zip"
	// DupeDetectionSubFolder is the subfolder where dd scripts live
	DupeDetectionSubFolder = "dd-service"
	// DupeDetectionConfigFilename is the config file name for dd
	DupeDetectionConfigFilename = "config.ini"
	// DupeDetectionExecFileName  is exec name for dupe detection
	DupeDetectionExecFileName = "dupe_detection_server.py"
	// PortCheckURL is URL of port checker service
	PortCheckURL = "http://portchecker.com?q="
	// IPCheckURL is URL of IP checker service
	IPCheckURL = "http://ipinfo.io/ip"
	// NetworkRegTest defines regtest network mode
	NetworkRegTest = "regtest"
	// NetworkTestnet defines testnet network mode
	NetworkTestnet = "testnet"
	// NetworkMainnet defines mainnet network mode
	NetworkMainnet = "mainnet"
	// BurnAddressTestnet defines testnet burn address
	BurnAddressTestnet = "tPpasteLBurnAddressXXXXXXXXXXX3wy7u"
	// BurnAddressMainnet defines mainnet burn address
	BurnAddressMainnet = "PtpasteLBurnAddressXXXXXXXXXXbJ5ndd"
	// DupeDetectionServiceDir defines location for dupe detection service
	DupeDetectionServiceDir = "pastel_dupe_detection_service"

	// SystemdServicePrefix prefix of all pastel services
	SystemdServicePrefix = "pastel-"
	// SystemdSystemDir location of systemd folder in Linux system
	SystemdSystemDir = "/etc/systemd/system"

	// RQServiceDir defines location for rq-service file exchange dir
	RQServiceDir = "rqfiles"
	// P2PDataDir defines location for p2p data dir
	P2PDataDir = "p2pdata"
	// MDLDataDir defines location for MDL data dir
	MDLDataDir = "mdldata"

	// LogConfigDefaultCompress defines supernode log compress
	LogConfigDefaultCompress = true
	// LogConfigDefaultMaxSizeMB defines supernode log max size
	LogConfigDefaultMaxSizeMB = 100
	// LogConfigDefaultMaxAgeDays defines supernode log max age
	LogConfigDefaultMaxAgeDays = 3
	// LogConfigDefaultMaxBackups defines supernode log max backups
	LogConfigDefaultMaxBackups = 10

	// SuperNodeDefaultCommonLogLevel defines supernode common log level
	SuperNodeDefaultCommonLogLevel = "info"
	// SuperNodeDefaultP2PLogLevel defines supernode p2p log level
	SuperNodeDefaultP2PLogLevel = "error"
	// SuperNodeDefaultMetaDBLogLevel defines supernode meta db log level
	SuperNodeDefaultMetaDBLogLevel = "error"
	// SuperNodeDefaultDDLogLevel defines supernode dupe detection log level
	SuperNodeDefaultDDLogLevel = "error"

	// WalletNodeDefaultLogLevel defines walletnode log level
	WalletNodeDefaultLogLevel = "info"

	// RQServiceDefaultPort defines rqservice port
	RQServiceDefaultPort = 50051

	// DDServerDefaultPort defines dd-server port
	DDServerDefaultPort = 50052

	// StorageChallengeExpiredDuration defines expired duration storage challenge process
	StorageChallengeExpiredDuration = "3m"

	// NumberOfChallengeReplicas defines number of storage challenge replicas
	NumberOfChallengeReplicas = 1

	// TempDir defines temporary directory
	TempDir = "tmp"

	// RemotePastelupPath - Remote pastelup path
	RemotePastelupPath = "/tmp/pastelup"
)

// ToolTypeServices represents the list of tool types that can be enabled as system services
// i.e. systemd services if on linux
var ToolTypeServices = []ToolType{
	WalletNode,
	SuperNode,
	GoNode,
	DDService,
	RQService,
	DDImgService,
}

// ServiceName defines services name
var ServiceName = map[ToolType]map[OSType]string{
	PastelD:    PasteldName,
	WalletNode: WalletNodeExecName,
	SuperNode:  SuperNodeExecName,
	RQService:  PastelRQServiceExecName,
	Pastelup:   PastelupName,
}

// PastelupName - The name of the pastelup
var PastelupName = map[OSType]string{
	Windows: "pastelup.exe",
	Linux:   "pastelup",
	Mac:     "pastelup",
	Unknown: "",
}

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
	Windows: "walletnode-win-amd64.exe",
	Linux:   "walletnode-linux-amd64",
	Mac:     "walletnode-darwin-amd64",
	Unknown: "",
}

// SuperNodeExecName - The name of the pastel wallet node
var SuperNodeExecName = map[OSType]string{
	Windows: "",
	Linux:   "supernode-linux-amd64",
	Mac:     "",
	Unknown: "",
}

// PastelUpExecName - The name of the pastelup
var PastelUpExecName = map[OSType]string{
	Windows: "pastelup-windows-amd64.exe",
	Linux:   "pastelup-linux-amd64",
	Mac:     "pastelup-darwin-amd64",
}

// PastelExecArchiveName - The name of the pastel executable files
var PastelExecArchiveName = map[OSType]string{
	Windows: "pastel-win-amd64.zip",
	Linux:   "pastel-linux-amd64.zip",
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

// GooglePubKeyURL - The url of the google public key
var GooglePubKeyURL = "https://dl-ssl.google.com/linux/linux_signing_key.pub"

// GooglePPASourceList - The url of the google PPA source list
var GooglePPASourceList = "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main"

// UbuntuSourceListPath - The path of the ubuntu source list
var UbuntuSourceListPath = "/etc/apt/sources.list.d"

// MainnetPortList - PortList of supernode
var MainnetPortList = []int{9933, 9932, 4444, 4445, 4446, 4447}

// TestnetPortList - PortList of supernode
var TestnetPortList = []int{19933, 19932, 14444, 14445, 14446, 14447}

// RegTestPortList - PortList of supernode
var RegTestPortList = []int{18344, 18343, 14444, 14445, 14446, 14447}

// Ports mapping
const (
	NodePort    int = 0
	NodeRPCPort int = 1
	SNPort      int = 2
	P2PPort     int = 3
	MDLPort     int = 4
	RAFTPort    int = 5
)

// PastelRQServiceExecName - The name of the rqservice executable files
var PastelRQServiceExecName = map[OSType]string{
	Windows: "rq-service-win-amd64.exe",
	Linux:   "rq-service-linux-amd64",
	Mac:     "rq-service-darwin-amd64",
	Unknown: "",
}

// DupeDetectionConfigs - dupe detection path of the supernode config
var DupeDetectionConfigs = []string{
	"input_files",
	DupeDetectionSupportFilePath,
	"output_files",
	"processed_files",
	"rare_on_internet",
	filepath.Join(DupeDetectionSupportFilePath, "mobilenet_v2_140_224"),
	"img_server",
}

// DupeDetectionSupportDownloadURL - The URL of dupe detection support files
var DupeDetectionSupportDownloadURL = []string{
	"https://download.pastel.network/machine-learning/DupeDetector_gray.pth.tar",
	"https://download.pastel.network/machine-learning/pca_bw.vt",
	"https://download.pastel.network/machine-learning/registered_image_fingerprints_db.sqlite",
	"https://download.pastel.network/machine-learning/train_0_bw.hdf5",
	"https://download.pastel.network/machine-learning/nsfw_mobilenet_v2_140_224.zip",
}

// DupeDetectionSupportChecksum - The checksum of dupe detection support files
var DupeDetectionSupportChecksum = map[string]string{
	"DupeDetector_gray.pth.tar":               "9eb31f27c5ce362dc558b4b77abedcbe327909f093d97f24acce20fcfb872c36",
	"pca_bw.vt":                               "d1bd688fcfa09f650d42a0160479c2bebf1cf87596645a55220e56696f386c73",
	"train_0_bw.hdf5":                         "659a2d480783709130c56e862a3a6e16d659c6dd063e80271fe51542b8b92590",
	"registered_image_fingerprints_db.sqlite": "5d01d8c944022d8346c25d7f70bc4ff985de1ec40d2465b70c18e7f370ce44f6",
	"mobilenet_v2_140_224":                    "825a4298a25334201ad5fb29e089fce0258c9a13793cc0d0b6a7dbe9c96ad9f3",
}

// DupeDetectionSupportFilePath - The target path for downloading dupe detection support files
var DupeDetectionSupportFilePath = "support_files"

// GetVersionSubURL returns the sub url concerned with version info
func GetVersionSubURL(version string) string {
	switch version {
	case "latest", "beta":
		return version
	default:
		return fmt.Sprintf("history/%s", version)
	}
}

// TODO: Add more dependencies for walletnode/supernode/pasteld in mac/win/linux os

// DependenciesPackages defines some dependencies
var DependenciesPackages = map[ToolType]map[OSType][]string{
	PastelD:    DependenciesPackagesPastelD,
	WalletNode: DependenciesPackagesWalletNode,
	SuperNode:  DependenciesPackagesSuperNode,
	DDService:  DependenciesPackagesDDService,
	RQService:  DependenciesPackagesRQService,
}

// DependenciesPackagesPastelD defines some dependencies for pasteld
var DependenciesPackagesPastelD = map[OSType][]string{
	Linux:   {"libgomp1"},
	Mac:     {},
	Windows: {},
	Unknown: {},
}

// DependenciesPackagesWalletNode defines some dependencies for walletnode
var DependenciesPackagesWalletNode = map[OSType][]string{
	Linux:   {"libgomp1"},
	Mac:     {},
	Windows: {},
	Unknown: {},
}

// DependenciesPackagesSuperNode defines some dependencies for supernode
var DependenciesPackagesSuperNode = map[OSType][]string{
	Linux:   {"libgomp1", "ufw", "curl"},
	Mac:     {},
	Windows: {},
	Unknown: {},
}

// DependenciesPackagesDDService defines some dependencies for walletnode
var DependenciesPackagesDDService = map[OSType][]string{
	Linux:   {"python3-pip", "google-chrome-stable", "libwebp-dev", "python3-venv"},
	Mac:     {},
	Windows: {},
	Unknown: {},
}

// DependenciesPackagesRQService is dependencies for dupe detection service
var DependenciesPackagesRQService = map[OSType][]string{
	Linux:   {},
	Mac:     {},
	Windows: {},
	Unknown: {},
}

// NetworkModes are valid network nmodes
var NetworkModes = []string{
	NetworkMainnet,
	NetworkTestnet,
	NetworkRegTest,
}

// NoVersionSetErr is an error returned if a install or update command is initiated without
// explicitly providing the requested version parameter
type NoVersionSetErr struct{}

func (e NoVersionSetErr) Error() string {
	return `
--release or -r must be provided. Recommened to use 'beta' i.e.
	
	pastelup install <service> -r beta
	
More information can be found: https://download.pastel.network/#"`
}
