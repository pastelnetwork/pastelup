package pastelcore

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/pastelnetwork/pastelup/configs"
	"github.com/pastelnetwork/pastelup/constants"
)

var (
	baseAddr = "http://127.0.0.1:"
	// DefaultClient is the default, overridable, http client to interface with pasteld
	// rpc server
	DefaultClient = http.Client{Timeout: 30 * time.Second}
)

const (
	// GetInfoCmd is an RPC command
	GetInfoCmd = "getinfo"
	// GetBalanceCmd is an RPC command
	GetBalanceCmd = "getbalance"
	// SendToAddressCmd is an R`PC command
	SendToAddressCmd = "sendtoaddress"
	// MasterNodeSyncCmd is an RPC command
	MasterNodeSyncCmd = "mnsync"
	// StopCmd is an RPC command
	StopCmd = "stop"
	// PastelIDCmd is an RPC command
	PastelIDCmd = "pastelid"
	// MasterNodeCmd is an RPC command
	MasterNodeCmd = "masternode"
	// GetNewAddressCmd is an RPC command
	GetNewAddressCmd = "getnewaddress"
)

// RPCRequest represents a jsonrpc request object.
//
// See: http://www.jsonrpc.org/specification#request_object
type RPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      string      `json:"id"`
}

// RPCCommunicator represents a struct that can interact with pastelcore RPC server
type RPCCommunicator interface {
	RunCommand(string, interface{}) error
	RunCommandWithArgs(string, interface{}, interface{}) error
}

// Client represents an rpc client that satisifies the RPCCommunicator interface
type Client struct {
	username, password string
	port               int
	network            string
}

// NewClient returns a new client
func NewClient(config *configs.Config) *Client {
	return &Client{
		username: config.RPCUser,
		password: config.RPCPwd,
		port:     config.RPCPort,
		network:  config.Network,
	}
}

// Addr returns the address of the pasteld rpc server
func (client Client) Addr() string {
	p := client.port
	if p == 0 {
		// need to reimplemtn the logic in common.go GetSNPortList to avoid import cycle
		if client.network == constants.NetworkTestnet {
			p = constants.TestnetPortList[constants.RPCPort]
		} else {
			p = constants.MainnetPortList[constants.RPCPort]
		}
	}
	return baseAddr + strconv.Itoa(p)
}

// RunCommand runs an RPC command with no args against pastelcore. Pass in a pointer to the
// response object so the client can populate it with the servers responses
func (client *Client) RunCommand(cmd string, response interface{}) error {
	var empty interface{}
	return client.do(cmd, &empty, response)
}

// RunCommandWithArgs runs an RPC command with args against pastelcore. Pass in a pointer to the
// response object so the client can populate it with the servers responses
func (client *Client) RunCommandWithArgs(cmd string, args, response interface{}) error {
	return client.do(cmd, &args, response)
}

func (client *Client) do(cmd string, args, response interface{}) error {
	body, err := json.Marshal(RPCRequest{
		JSONRPC: "1.0",
		ID:      "pastelapi",
		Method:  cmd,
		Params:  args,
	})
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", client.Addr(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.SetBasicAuth(client.username, client.password)
	request.Header.Set("Content-Type", "text/plain;")
	result, err := DefaultClient.Do(request)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(result.Body)
	err = decoder.Decode(&response)
	if err != nil {
		return err
	}
	return nil
}
