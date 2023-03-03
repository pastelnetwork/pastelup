package cmd

import (
	"context"
	"fmt"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/pastelup/configs"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
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
	Name         string `yaml:"name,omitempty"`
	Host         string `yaml:"host,omitempty"`
	User         string `yaml:"user,omitempty"`
	IdentityFile string `yaml:"identity-file,omitempty"`
	Port         int    `yaml:"port,omitempty"`
}

// AnsibleInventory defines top level of Ansible Inventory file
type AnsibleInventory struct {
	AnsibleHostGroups map[string]AnsibleInventoryGroup
}

type AnsibleInventoryGroup struct {
	AnsibleHosts    map[string]AnsibleVars `yaml:"hosts"`
	AnsibleHostVars AnsibleVars            `yaml:"vars"`
}

type AnsibleVars map[string]string

// ReadLegacyInventory read and load pastelup's legacy inventory file
func (i *Inventory) ReadLegacyInventory(path string) error {
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

func (i *Inventory) ReadAnsibleYamlInventory(path string) error {
	// Read YAML file
	file, err := os.ReadFile(path)
	if err != nil {
		return errors.Errorf("failed to read Inventory file: %v", err)
	}

	// Parse YAML data into Inventory struct
	aInventory := AnsibleInventory{}
	err = yaml.Unmarshal(file, &aInventory.AnsibleHostGroups)
	if err != nil {
		return errors.Errorf("failed to load Inventory: %v", err)
	}

	for groupName, group := range aInventory.AnsibleHostGroups {
		var servers []InventoryServer
		for serverName, serverVars := range group.AnsibleHosts {
			server := InventoryServer{
				Name: serverName,
				Host: serverVars["ansible_host"],
			}
			if user, ok := serverVars["ansible_user"]; ok {
				server.User = user
			}
			if identityFile, ok := serverVars["ansible_ssh_private_key_file"]; ok {
				server.IdentityFile = identityFile
			}
			if port, ok := serverVars["ansible_port"]; ok {
				portInt, err := strconv.Atoi(port)
				if err != nil {
					log.Printf("error converting port for server %s: %s", serverName, err)
				} else {
					server.Port = portInt
				}
			}
			servers = append(servers, server)
		}
		serverGroup := ServerGroup{
			Name: groupName,
			Common: CommonInventoryParameters{
				User:         group.AnsibleHostVars["ansible_user"],
				IdentityFile: group.AnsibleHostVars["ansible_private_key_file"],
			},
			Servers: servers,
		}
		if port, ok := group.AnsibleHostVars["ansible_port"]; ok {
			portInt, err := strconv.Atoi(port)
			if err != nil {
				log.Printf("error converting port for group %s: %s", groupName, err)
			} else {
				serverGroup.Common.Port = portInt
			}
		}
		i.ServerGroups = append(i.ServerGroups, serverGroup)
	}
	return nil
}

// ExecuteCommands executes commands on all hosts from inventory
func (i *Inventory) ExecuteCommands(ctx context.Context, config *configs.Config, commands []string) error {
	var filters []string
	if config.InventoryFilter != "" {
		filters = strings.Split(config.InventoryFilter, ",")
	}

	for _, sg := range i.ServerGroups {
		if len(filters) > 0 {
			index := sort.SearchStrings(filters, sg.Name)
			if index < len(filters) && filters[index] == sg.Name {
			} else {
				continue
			}
		}
		fmt.Printf(green("\n********** Accessing host group %s **********\n"), sg.Name)
		config.RemoteUser = ""
		config.RemoteIP = ""
		config.RemotePort = 0
		config.RemoteSSHKey = ""

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
				log.WithContext(ctx).WithError(err).Errorf("Failed to execute command on remote host %s"+
					" [IP:%s; Port:%d; User:%s; KeyFile:%s; ]",
					srv.Name, config.RemoteIP, config.RemotePort, config.RemoteUser, config.RemoteSSHKey)
			}
		}
	}
	return nil
}
