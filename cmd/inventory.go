package cmd

import (
	"context"
	"fmt"
	"github.com/pastelnetwork/pastelup/configs"
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/pastelnetwork/gonode/common/log"
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
func (i *Inventory) Read(path string) error {
	invFile, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.Errorf("failed to read Inventory file: %v", err)
	}

	err = yaml.Unmarshal(invFile, i)
	if err != nil {
		return errors.Errorf("failed to load Inventory: %v", err)
	}

	return nil
}

// ExecuteCommands executes commands on all hosts from inventory
func (i *Inventory) ExecuteCommands(ctx context.Context, config *configs.Config, commands []string) error {
	for _, sg := range i.ServerGroups {
		fmt.Printf(green("\n********** Accessing host group %s **********\n"), sg.Name)

		if len(sg.Common.User) > 0 {
			config.RemoteUser = sg.Common.User
		}
		if sg.Common.Port != 0 {
			config.RemotePort = sg.Common.Port
		}
		if len(sg.Common.IdentityFile) > 0 {
			config.RemoteSSHKey = sg.Common.IdentityFile
		}
		for _, srv := range sg.Servers {
			fmt.Printf(green("\n********** Executing command on %s **********\n"), srv.Name)
			if len(srv.User) > 0 {
				config.RemoteUser = srv.User
			}
			if srv.Port != 0 {
				config.RemotePort = srv.Port
			}
			if len(srv.IdentityFile) > 0 {
				config.RemoteSSHKey = srv.IdentityFile
			}
			config.RemoteIP = srv.Host
			if config.RemotePort == 0 {
				config.RemotePort = 22
			}
			if err := executeRemoteCommands(ctx, config, commands, false); err != nil {
				log.WithContext(ctx).WithError(err).Error("Failed to execute command on remote host %s"+
					" [IP:%s; Port:%d; User:%s; KeyFile:%s; ]",
					srv.Name, config.RemoteIP, config.RemotePort, config.RemoteUser, config.RemoteSSHKey)
			}
		}
	}
	return nil
}
