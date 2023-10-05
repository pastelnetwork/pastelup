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
	// Hermes type
	Hermes ToolType = "hermes"
	// Bridge type
	Bridge ToolType = "bridge"
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

	// BridgeServiceDefaultPort defines bridge service port
	BridgeServiceDefaultPort = 60061

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

// ServiceName defines services name
var ServiceName = map[ToolType]map[OSType]string{
	PastelD:    PasteldName,
	WalletNode: WalletNodeExecName,
	SuperNode:  SuperNodeExecName,
	Bridge:     BridgeExecName,
	Hermes:     HermesExecName,
	RQService:  PastelRQServiceExecName,
	Pastelup:   PastelupName,
}

// PastelupName - The name of the pastelup
var PastelupName = map[OSType]string{
	Windows: "pastelup-win-amd64.exe",
	Linux:   "pastelup-linux-amd64",
	Mac:     "pastelup-darwin-amd64",
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
	Windows: "pastelup-win-amd64.exe",
	Linux:   "pastelup-linux-amd64",
	Mac:     "pastelup-darwin-amd64",
}

// HermesExecName - The name of the hermes
var HermesExecName = map[OSType]string{
	Windows: "",
	Linux:   "hermes-linux-amd64",
	Mac:     "",
}

// BridgeExecName - The name of the bridge
var BridgeExecName = map[OSType]string{
	Windows: "bridge-win-amd64.exe",
	Linux:   "bridge-linux-amd64",
	Mac:     "bridge-darwin-amd64",
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
	"https://download.pastel.network/other/machine-learning/pastel_image_dupe_detection_model_v_1_0.pth.tar",
	"https://download.pastel.network/other/machine-learning/pca_byol_bw.vt",
	"https://download.pastel.network/other/machine-learning/bayesian_ridge_model_3.joblib",
	"https://download.pastel.network/other/machine-learning/nsfw_mobilenet_v2_140_224.zip",
}

// DupeDetectionSupportChecksum - The checksum of dupe detection support files
var DupeDetectionSupportChecksum = map[string]string{
	"pastel_image_dupe_detection_model_v_1_0.pth.tar": "5fb3e01cc73654825d17b8d49c60cc138912fca0fe752d5476ed00e8827ae082",
	"pca_byol_bw.vt":                "2515062da53f39fb08cbabdc392beff9749240021513f0bf6fed1eeac2aae3de",
	"bayesian_ridge_model_3.joblib": "8960914863f6a9859e440d57b4af59e17932b6f4d2ebe9515e89138072a52b06",
	"nsfw_mobilenet_v2_140_224":     "57278dd5a158b7ad8cb4f8d32f26c5f836d0d038044a2119f76fecff7aa241e9",
}

// DupeDetectionSupportFilePath - The target path for downloading dupe detection support files
var DupeDetectionSupportFilePath = "support_files"

// GetVersionSubURL returns the sub url concerned with version info
func GetVersionSubURL(network string, version string) string {
	if version == "" {
		if network == "" {
			return "latest-release"
		}
		return fmt.Sprintf("latest-release/%s", network)
	}
	return fmt.Sprintf("other/archive/%s/%s", network, version)
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

// NetworkModes are valid network modes
var NetworkModes = []string{
	NetworkMainnet,
	NetworkTestnet,
	NetworkRegTest,
}

// NoNetworkModesSetErr is an error returned if install or update command is initiated without
// explicitly providing the requested network mode parameter
type NoNetworkModesSetErr struct{}

func (e NoNetworkModesSetErr) Error() string {
	return `
--network or -n must be provided. Can be either 'mainnet' or 'testnet'
	
	pastelup install <service> -n mainnet
	
More information can be found: https://download.pastel.network/#"`
}
