package pastelcli

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

type Command string

const (
	GetInfo        Command = "getInfo"
	GetBalance     Command = "getbalance"
	SendToAdress   Command = "sendtoaddress"
	MasterNodeSync Command = "mnsync"
	Stop           Command = "stop"
	PastelIDNewKey Command = "pastelid"
	MasterNode     Command = "masternode"
	GetNewAddress  Command = "getnewaddress"
)

type CLICommunicator interface {
	RunCommand(Command) (interface{}, error)
	RunCommandWithArgs(Command, interface{}) (interface{}, error)
}

type Client struct{}

func NewClient() *Client { return &Client{} }

func (client *Client) RunCommand(cmd Command) (interface{}, error) {
	return client.do(string(cmd), nil)
}

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
