package configs

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/pastel-utility/constants"
	"github.com/pastelnetwork/pastel-utility/utils"
)

const (
	// WalletMainNetConfig - mainnet config for walletnode
	WalletMainNetConfig = `node:
  api:
    hostname: "localhost"
    port: 8080
`
	// WalletTestNetConfig - testnet config for walletnode
	WalletTestNetConfig = `pastel-api:
  port: 19932
  
node:
  api:
    hostname: "localhost"
    port: 8080
`
	// WalletLocalNetConfig - localnet config for walletnode
	WalletLocalNetConfig = `pastel-api:
  hostname: "127.0.0.1"
  port: 29932
  username: ""
  password: ""
  
node:
  api:
    hostname: "localhost"
    port: 8080
`

	// SupernodeDefaultConfig - default config for supernode
	SupernodeDefaultConfig = `node:
  # ` + `pastel_id` + ` must match to active ` + `PastelID` + ` from masternode.
  # To check it out first get the active outpoint from ` + `masteronde status` + `, then filter the result of ` + `tickets list id mine` + ` by this outpoint.
  pastel_id: %s
  server:
    # ` + `listen_address` + ` and ` + `port` + ` must match to ` + `extAddress` + ` from masternode.conf
    listen_addresses: %s
    port: %s
`

	// SupernodeYmlLine1 - default supernode.yml content line 1
	SupernodeYmlLine1 = "node:"
	// SupernodeYmlLine2 - default supernode.yml content line 2
	SupernodeYmlLine2 = "  # ` + `pastel_id` + ` must match to active ` + `PastelID` + ` from masternode."
	// SupernodeYmlLine3 - default supernode.yml content line 3
	SupernodeYmlLine3 = "  # To check it out first get the active outpoint from ` + `masteronde status` + `, then filter the result of ` + `tickets list id mine` + ` by this outpoint."
	// SupernodeYmlLine4 - default supernode.yml content line 4
	SupernodeYmlLine4 = "  pastel_id: %s"
	// SupernodeYmlLine5 - default supernode.yml content line 5
	SupernodeYmlLine5 = "  server:"
	// SupernodeYmlLine6 - default supernode.yml content line 6
	SupernodeYmlLine6 = `    # ` + `listen_address` + ` and ` + `port` + ` must match to ` + `extAddress` + ` from masternode.conf`
	// SupernodeYmlLine7 - default supernode.yml content line 7
	SupernodeYmlLine7 = "    listen_addresses: %s"
	// SupernodeYmlLine8 - default supernode.yml content line 8
	SupernodeYmlLine8 = "    port: %s"

	// RQServiceConfig - default rqserivce config
	RQServiceConfig = `grpc-service = "%s:%s"`

	// ZksnarkParamsURL - url for zksnark params
	ZksnarkParamsURL = "https://download.pastel.network/pastel-params/"

	//DupeDetectionConfig - default config for dupedecteion
	DupeDetectionConfig = `[DUPEDETECTIONCONFIG]
	input_files_path = %s
	support_files_path = %s
	output_files_path = %s
	processed_files_path = %s
	internet_rareness_downloaded_images_path = %s
	nsfw_model_path = %s
	`
)

// ZksnarkParamsNames - slice of zksnark parameters
var ZksnarkParamsNames = []string{
	"sapling-spend.params",
	"sapling-output.params",
	"sprout-proving.key",
	"sprout-verifying.key",
	"sprout-groth16.params",
}

// Config contains configuration of all components of the WalletNode.
type Config struct {
	Main `json:","`
	Init `json:","`
}

// String : returns string from Config fields
func (config *Config) String() (string, error) {
	// The main purpose of using a custom converting is to avoid unveiling credentials.
	// All credentials fields must be tagged `json:"-"`.
	data, err := json.Marshal(config)

	if err != nil {
		return "", err
	}

	return string(data), nil
}

// SaveConfig : save pastel-utility config
func (config *Config) SaveConfig() error {
	data, err := config.String()

	if err != nil {
		return err
	}

	if ioutil.WriteFile(constants.PastelUtilityConfigFilePath, []byte(data), 0644) != nil {
		return err
	}
	return nil
}

// LoadConfig : load the config from config file
func LoadConfig() (cofig *Config, err error) {
	data, err := ioutil.ReadFile(constants.PastelUtilityConfigFilePath)

	if err != nil {
		return nil, err
	}

	var dataConf Config
	err = json.Unmarshal(data, &dataConf)

	return &dataConf, err
}

// New returns a new Config instance
func New() *Config {
	return &Config{
		Main: *NewMain(),
	}
}

// GetConfig : Get the config from config file. If there is no config file then create a new config.
func GetConfig() *Config {
	var config *Config
	var err error
	if utils.CheckFileExist(constants.PastelUtilityConfigFilePath) {
		config, err = LoadConfig()
		if err != nil {
			log.Error("The pastel-utility.conf file is not correct\n")
			os.Exit(-1)
		}
	} else {
		config = New()
	}
	return config
}
