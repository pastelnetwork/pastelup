package configs

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

// Inventory defines top level of Inventory file
type Inventory struct {
	ServerGroups []ServerGroup `yaml:"server-groups,omitempty"`
}

// ServerGroup defines group of remote hosts with common parameters or context
type ServerGroup struct {
	Name    string                    `yaml:"name,omitempty"`
	Common  CommonInventoryParameters `yaml:"common,omitempty"`
	Servers []InventoryServer         `yaml:"servers,omitempty"`
}

// CommonInventoryParameters defines common parameters of server group
type CommonInventoryParameters struct {
	User         string `yaml:"user,omitempty"`
	IdentityFile string `yaml:"identity-file,omitempty"`
	Port         int    `yaml:"port,omitempty"`
}

// InventoryServer defines remote host
type InventoryServer struct {
	CommonInventoryParameters
	Name string `yaml:"name,omitempty"`
	Host string `yaml:"host,omitempty"`
}

// ReadInventory read and load inventory file
func ReadInventory(path string) (Inventory, error) {
	var inv Inventory
	invFile, err := ioutil.ReadFile(path)
	if err != nil {
		return inv, errors.Errorf("failed to read Inventory file: %v", err)
	}

	err = yaml.Unmarshal(invFile, &inv)
	if err != nil {
		return inv, errors.Errorf("failed to load Inventory: %v", err)
	}

	return inv, nil
}
