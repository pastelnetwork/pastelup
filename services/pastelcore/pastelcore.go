package pastelcore

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/pastelnetwork/pastelup/configs"
)

/*
All instances to test:
	RunPastelCLI(ctx, config, "getinfo")
	RunPastelCLI(ctx, r.config, "getbalance")
	RunPastelCLI(ctx, r.config, "sendtoaddress", zcashAddr, fmt.Sprintf("%v", amount))
	RunPastelCLI(ctx, config, "mnsync", "status")
	RunPastelCLI(ctx, config, "stop")
	RunPastelCLI(ctx, config, "pastelid", "newkey", flagMasterNodePassPhrase)
	RunPastelCLI(ctx, config, "masternode", "genkey")
	RunPastelCLI(ctx, config, "masternode", "outputs")
	RunPastelCLI(ctx, config, "getnewaddress")
	RunPastelCLI(ctx, config, "masternode", "start-alias", masternodeName)
*/

var (
	port          = 9932
	addr          = "http://127.0.0.1:" + strconv.Itoa(port)
	DefaultClient = http.Client{Timeout: 30 * time.Second}
)

const (
	// GetInfoCmd is an RPC command
	GetInfoCmd = "getInfo"
	// GetBalanceCmd is an RPC command
	GetBalanceCmd = "getbalance"
	// SendToAddressCmd is an RPC command
	SendToAddressCmd = "sendtoaddress"
	// MasterNodeSyncCmd is an RPC command
	MasterNodeSyncCmd = "mnsync"
	// StopCmd is an RPC command
	StopCmd = "stop"
	// PastelIDNewKeyCmd is an RPC command
	PastelIDNewKeyCmd = "pastelid"
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
}

// NewClient returns a new client
func NewClient(config *configs.Config) *Client {
	return &Client{
		username: config.RPCUser,
		password: config.RPCPwd,
	}
}

// RunCommand runs an RPC command with no args against pastelcore
func (client *Client) RunCommand(cmd string, response interface{}) error {
	var empty interface{}
	return client.do(cmd, &empty, response)
}

// RunCommandWithArgs runs an RPC command with args against pastelcore
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
	request, err := http.NewRequest("POST", addr, bytes.NewReader(body))
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
