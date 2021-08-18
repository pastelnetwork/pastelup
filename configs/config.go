package configs

import (
	"encoding/json"
	"github.com/pastelnetwork/pastel-utility/constants"
	"github.com/pastelnetwork/pastel-utility/utils"
	"io/ioutil"
	"os"

	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/pastel-utility/configurer"
)

const (
	// WalletDefaultConfig - default config for walletnode
	WalletDefaultConfig = `
pastel-api:
  hostname: "localhost"
  port: %s
  username: %s 
  password: %s
node:
  api:
    hostname: "localhost"
    port: 8080
  reg_art_tx_min_confirmations: 10
  # Timeout waiting for 
  reg_art_tx_timeout: 26
  reg_act_tx_min_confirmations: 5 
  # Timeout waiting for 
  reg_act_tx_timeout: 13
raptorq:
  hostname: "localhost"
  port: %s
`

	// SupernodeDefaultConfig - default config for supernode
	SupernodeDefaultConfig = `
pastel-api:
  hostname: "localhost"
  port: %s 
  username: %s
  password: %s
node:
  pastel_id: jXXzhJJnfEk4cfJWeyX3t6o6EGBaCKsp6Qn38BUX62gHbBB325t1JqrkisKJGLopZ6QjzfeAmE623oK7GqS3sw 
  pass_phrase: "passphrase"
  preburnt_tx_min_confirmation: 3
  # Timeout in minutes
  preburnt_tx_confirmation_timeout: 8 
  server:
    listen_addresses: "0.0.0.0"
    port: 4444
raptorq:
  hostname: "localhost"
  port: %s
dupe-detection:
  input_dir: "input"
  output_dir: "output"
  data_file: "dupe_detection_image_fingerprint_database.sqlite"
p2p:
  listen_address: "0.0.0.0"
  port: 6000
  data_dir: "p2p-localnet-6000"
metadb:
  listen_address: "0.0.0.0"
  http_port: 4041
  raft_port: 4042
  data_dir: "metadb-4444"
`

/*	// SupernodeYmlLine1 - default supernode.yml content line 1
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
	SupernodeYmlLine8 = "    port: %s"*/

	// RQServiceConfig - default rqserivce config
	RQServiceConfig = `grpc-service = "%s:%s"`

	// ZksnarkParamsURL - url for zksnark params
	ZksnarkParamsURL = "https://download.pastel.network/pastel-params/"

	//DupeDetectionConfig - default config for dupedecteion
	DupeDetectionConfig = `
	[DUPEDETECTIONCONFIG]
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
	Main       `json:","`
	Init       `json:","`
	Configurer configurer.IConfigurer `json:"-"`
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
			log.Errorf("the pastel-utility.conf file is not correct, err: %s", err)
			os.Exit(-1)
		}
	} else {
		config = New()
	}

	c, err := configurer.NewConfigurer()
	if err != nil {
		log.WithError(err).Error("failed to initialize configurer")
		os.Exit(-1)
	}
	config.Configurer = c
	return config
}
