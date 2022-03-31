package pastelclient

import (
	"io"
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

type Communicator interface {
	RunCommand(Command) (interface{}, error)
	RunCommandWithArgs(Command, interface{}) (interface{}, error)
	CloseConnection() error
}

type Client struct {
	rpcClient *rpc.Client
}

func New(conn io.ReadWriteCloser) (*Client, error) {
	client, err := rpc.Dial(network, addr)
	return &Client{
		rpcClient: client,
	}, err
}

func (client *Client) RunCommand(cmd Command) (interface{}, error) {
	var response interface{}
	err := client.rpcClient.Call(string(cmd), nil, &response)
	return response, err
}

func (client *Client) RunCommandWithArgs(cmd Command, args interface{}) (interface{}, error) {
	var response interface{}
	err := client.rpcClient.Call(string(cmd), args, &response)
	return response, err
}

func (client *Client) CloseConnection() error {
	return client.rpcClient.Close()
}
