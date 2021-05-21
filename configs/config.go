package configs

import (
	"encoding/json"
	"fmt"
)

// Config contains configuration of all components of the WalletNode.
type Config struct {
	Main `json:",squash"`
}

func (config *Config) String() string {
	// The main purpose of using a custom converting is to avoid unveiling credentials.
	// All credentials fields must be tagged `json:"-"`.
	data, err := json.Marshal(config)

	if err != nil {
		return fmt.Sprintf("Error to marshal config %v", err)
	}

	return string(data)
}

// New returns a new Config instance
func New() *Config {
	return &Config{
		Main: *NewMain(),
	}
}
