package pastelcore

import (
	"net/rpc"
	"strconv"
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
	network = "http" // "http" or "tcp"
	port    = 9932
	addr    = "127.0.0.1:" + strconv.Itoa(port)
)

// Command represents a pastel-cli command to run
type Command string

const (
	// GetInfo is an RPC command
	GetInfo Command = "getInfo"
	// GetBalance is an RPC command
	GetBalance Command = "getbalance"
	// SendToAdress is an RPC command
	SendToAdress Command = "sendtoaddress"
	// MasterNodeSync is an RPC command
	MasterNodeSync Command = "mnsync"
	// Stop is an RPC command
	Stop Command = "stop"
	// PastelIDNewKey is an RPC command
	PastelIDNewKey Command = "pastelid"
	// MasterNode is an RPC command
	MasterNode Command = "masternode"
	// GetNewAddress is an RPC command
	GetNewAddress Command = "getnewaddress"
)

// RPCCommunicator represents a struct that can interact with pastelcore RPC server
type RPCCommunicator interface {
	RunCommand(Command) (interface{}, error)
	RunCommandWithArgs(Command, interface{}) (interface{}, error)
}

// Client represents an rpc client that satisifies the RPCCommunicator interface
type Client struct{}

// NewClient returns a new client
func NewClient() *Client { return &Client{} }

// RunCommand runs an RPC command with no args against pastelcore
func (client *Client) RunCommand(cmd Command) (interface{}, error) {
	return client.do(string(cmd), nil)
}

// RunCommandWithArgs runs an RPC command with args against pastelcore
func (client *Client) RunCommandWithArgs(cmd Command, args interface{}) (interface{}, error) {
	return client.do(string(cmd), args)
}

/*
 * TODO: figure out when the server is available versus when it isnt and create
 * 		 a persistent client instead of initalizing one per call and closing it each time.
 */
func (client *Client) do(cmd string, args interface{}) (interface{}, error) {
	var response interface{}
	rpcClient, err := rpc.Dial(network, addr) // we cant keep an open connection because server starts and stops often
	if err != nil {
		return response, err
	}
	err = rpcClient.Call(cmd, args, &response)
	if err != nil {
		return response, err
	}
	return response, rpcClient.Close()
}
