package pastelcore

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pastelnetwork/pastelup/configs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	port = 9932
	addr = ":" + strconv.Itoa(port)
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

// RPCCommunicator represents a struct that can interact with pastelcore RPC server
type RPCCommunicator interface {
	RunCommand(string) (interface{}, error)
	RunCommandWithArgs(string, interface{}) (interface{}, error)
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
func (client *Client) RunCommand(cmd string) (interface{}, error) {
	var empty interface{}
	var resp interface{}
	err := client.do(cmd, &empty, resp)
	return resp, err
}

// RunCommandWithArgs runs an RPC command with args against pastelcore
func (client *Client) RunCommandWithArgs(cmd string, args interface{}) (interface{}, error) {
	var resp interface{}
	err := client.do(cmd, &args, resp)
	return resp, err
}

/*
 * TODO: figure out when the server is available versus when it isnt and create
 * 		 a persistent client instead of initalizing one per call and closing it each time.
 */
func (client *Client) do(cmd string, args, response interface{}) error {
	// we cant keep an open connection because server starts and stops often
	grpclient, err := grpc.Dial(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(credential{
			username: client.username,
			password: client.password,
		}))
	if err != nil {
		return err
	}
	fmt.Printf("making RPC call!\n")
	err = grpclient.Invoke(context.Background(), cmd, args, &response)
	if err != nil {
		return err
	}
	err = grpclient.Close()
	if err != nil {
		return err
	}
	fmt.Printf("successfully closed client!\n")
	return nil
}

type credential struct {
	username string
	password string
}

func (cred credential) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"username": cred.username,
		"password": cred.password,
	}, nil
}

func (cred credential) RequireTransportSecurity() bool {
	return false
}
