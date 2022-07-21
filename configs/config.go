package configs

import (
	"encoding/json"
	"os"

	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/pastelup/configurer"
)

const (
	// WalletDefaultConfig - default config for walletnode
	WalletDefaultConfig = `
log-config:
  log-level: {{.LogLevel}}
  log-file: {{.LogFilePath}}
  log-compress: {{.LogCompress}}
  log-max-size-mb: {{.LogMaxSizeMB}}
  log-max-age-days: {{.LogMaxAgeDays}}
  log-max-backups: {{.LogMaxBackups}}
quiet: true
temp-dir: {{.WNTempDir}}
work-dir: {{.WNWorkDir}}
rq-files-dir: {{.RQDir}}

node:
  api:
    hostname: "localhost"
    port: 8080
  burn_address: {{.BurnAddress}} 
raptorq:
  host: "localhost"
  port: {{.RaptorqPort}}
`

	// SupernodeDefaultConfig - default config for supernode
	SupernodeDefaultConfig = `
log-config:
  log-file: {{.LogFilePath}}
  log-compress: {{.LogCompress}}
  log-max-size-mb: {{.LogMaxSizeMB}}
  log-max-age-days: {{.LogMaxAgeDays}}
  log-max-backups: {{.LogMaxBackups}}
  log-levels:
      common: {{.LogLevelCommon}}
      p2p: {{.LogLevelP2P}}
      metadb: {{.LogLevelMetadb}}
      dd: {{.LogLevelDD}}
    
quiet: true
temp-dir: {{.SNTempDir}}
work-dir: {{.SNWorkDir}}
rq-files-dir: {{.RQDir}}
dd-service-dir: {{.DDDir}}

node:
  pastel_id: {{.PasteID}} 
  pass_phrase: {{.Passphrase}}
  server:
    listen_addresses: "0.0.0.0"
    port: {{.SuperNodePort}}
  storage_challenge_expired_duration: {{.StorageChallengeExpiredDuration}}
  number_of_challenge_replicas: {{.NumberOfChallengeReplicas}}

p2p:
  listen_address: "0.0.0.0"
  port: {{.P2PPort}}
  data_dir: {{.P2PDataDir}}

metadb:
  # is_leader: false
  # none_voter: true
  listen_address: "0.0.0.0"
  http_port: {{.MDLPort}}
  raft_port: {{.RAFTPort}}
  data_dir: {{.MDLDataDir}}

raptorq:
  host: "localhost"
  port: {{.RaptorqPort}}

dd-server:
  host: "localhost"
  port: {{.DDServerPort}}
  dd-temp-file-dir: "dd-server"
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
	// DDImgServerStart - actual script to start dd image server - start_dd_img_server.sh
	DDImgServerStart = `#!/bin/bash
cd {{.DDImgServerDir}}
python3 -m  http.server 80`

	// SystemdService - /etc/systemd/sysstem/rq-service.service
	SystemdService = `[Unit]
Description={{.Desc}}
    
[Service]
Type=simple
Restart=always
RestartSec=10
WorkingDirectory={{.WorkDir}}
ExecStart={{.ExecCmd}}
User={{.User}}

[Install]
WantedBy=multi-user.target
`
)

// WalletNodeConfig defines configurations for walletnode
type WalletNodeConfig struct {
	LogLevel      string
	LogFilePath   string
	LogCompress   bool
	LogMaxSizeMB  int
	LogMaxAgeDays int
	LogMaxBackups int
	WNTempDir     string
	WNWorkDir     string
	RQDir         string
	RaptorqPort   int
	BurnAddress   string
}

// SuperNodeConfig defines configurations for supernode
type SuperNodeConfig struct {
	LogFilePath                     string
	LogCompress                     bool
	LogMaxSizeMB                    int
	LogMaxAgeDays                   int
	LogMaxBackups                   int
	LogLevelCommon                  string
	LogLevelP2P                     string
	LogLevelMetadb                  string
	LogLevelDD                      string
	SNTempDir                       string
	SNWorkDir                       string
	RQDir                           string
	DDDir                           string
	PasteID                         string
	Passphrase                      string
	SuperNodePort                   int
	P2PPort                         int
	P2PDataDir                      string
	MDLPort                         int
	RAFTPort                        int
	MDLDataDir                      string
	RaptorqPort                     int
	DDServerPort                    int
	NumberOfChallengeReplicas       int
	StorageChallengeExpiredDuration string
}

// RQServiceConfig defines configurations for rqservice
type RQServiceConfig struct {
	HostName string
	Port     int
}

// DDImgServerServiceScript defines service file for /etc/systemd/system
type DDImgServerServiceScript struct {
	DDImgServerStartScript string
}

// DDImgServerStartScript actual script to start dd image server
type DDImgServerStartScript struct {
	DDImgServerDir string
}

// SystemdServiceScript defines service file for /etc/systemd/system
type SystemdServiceScript struct {
	ExecCmd string
	Desc    string
	WorkDir string
	User    string
}

// ZksnarkParamsNamesV2 - slice of zksnark parameters
var ZksnarkParamsNamesV2 = []string{
	"sapling-spend.params",
	"sapling-output.params",
	"sprout-groth16.params",
}

// ZksnarkParamsNamesV1 - slice of zksnark parameters
var ZksnarkParamsNamesV1 = []string{
	"sprout-proving.key",
	"sprout-verifying.key",
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
