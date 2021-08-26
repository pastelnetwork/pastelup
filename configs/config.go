package configs

import (
	"encoding/json"
	"os"

	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/pastel-utility/configurer"
)

const (
	// WalletDefaultConfig - default config for walletnode
	WalletDefaultConfig = `
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
  port: {{.RaptorqPort}}
`

	// SupernodeDefaultConfig - default config for supernode
	SupernodeDefaultConfig = `
node:
  pastel_id: {{.PasteID}} 
  pass_phrase: {{.Passphrase}}
  preburnt_tx_min_confirmation: 3
  preburnt_tx_confirmation_timeout: 8 
  server:
    listen_addresses: "0.0.0.0"
    port: {{.SuperNodePort}}
  userdata_process:
    number_super_nodes: 3
    minimal_node_confirm_success: 2

p2p:
  listen_address: "0.0.0.0"
  port: {{.P2PPort}}
  data_dir: {{.P2PPortDataDir}}

metadb:
  listen_address: "0.0.0.0"
  http_port: {{.MDLPort}}
  raft_port: {{.RAFTPort}}
  data_dir: {{.MDLDataDir}}

userdb:
  schema-path: {{.MDLDataDir}}/schema_v1.sql
  write-template-path: {{.MDLDataDir}}/user_info_write.tmpl
  query-template-path: {{.MDLDataDir}}/user_info_query.tmpl

raptorq:
  hostname: "localhost"
  port: {{.RaptorqPort}}

dupe-detection:
  input_dir: "input"
  output_dir: "output"
  data_file: "dupe_detection_support_files/dupe_detection_image_fingerprint_database.sqlite"
`

	// RQServiceDefaultConfig - default rqserivce config
	RQServiceDefaultConfig = `grpc-service = "{{.HostName}}:{{.Port}}"`

	// ZksnarkParamsURL - url for zksnark params
	ZksnarkParamsURL = "https://download.pastel.network/pastel-params/"

	//DupeDetectionConfig - default config for dupedecteion
	DupeDetectionConfig = `
[DUPEDETECTIONCONFIG]
input_files_path = %s/
support_files_path = %s/
output_files_path = %s/
processed_files_path = %s/
internet_rareness_downloaded_images_path = %s/
nsfw_model_path = %s/
`
)

// WalletNodeConfig defines configurations for walletnode
type WalletNodeConfig struct {
	RaptorqPort int
}

// SuperNodeConfig defines configurations for supernode
type SuperNodeConfig struct {
	PasteID        string
	Passphrase     string
	SuperNodePort  int
	P2PPort        int
	P2PPortDataDir string
	MDLPort        int
	RAFTPort       int
	MDLDataDir     string
	RaptorqPort    int
}

// RQServiceConfig defines configurations for rqservice
type RQServiceConfig struct {
	HostName string
	Port     int
}

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

// New returns a new Config instance
func New() *Config {
	return &Config{
		Main: *NewMain(),
	}
}

// InitConfig : Get the config from config file. If there is no config file then create a new config.
func InitConfig() *Config {
	var config = New()

	c, err := configurer.NewConfigurer()
	if err != nil {
		log.WithError(err).Error("failed to initialize configurer")
		os.Exit(-1)
	}
	config.Configurer = c
	return config
}
