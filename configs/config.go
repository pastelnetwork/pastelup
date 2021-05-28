package configs

import (
	"encoding/json"
)

const (
	WalletDefaultConfig = `node:
	api:
		hostname: "localhost"
		port: 8080
`
	SupernodeDefaultConfig = `node:
	# ` + `pastel_id` + ` must match to active ` + `PastelID` + ` from masternode.
	# To check it out first get the active outpoint from ` + `masteronde status` + `, then filter the result of ` + `tickets list id mine` + ` by this outpoint.
	pastel_id: some-value
	server:
		# ` + `listen_address` + ` and ` + `port` + ` must match to ` + `extAddress` + ` from masternode.conf
		listen_addresses: "127.0.0.1"
		port: 4444
`
	ZksnarkParamsURL = "https://z.cash/downloads/"
)

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

// New returns a new Config instance
func New() *Config {
	return &Config{
		Main: *NewMain(),
	}
}
