package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/pastel-utility/configurer"
	"github.com/pastelnetwork/pastel-utility/structure"
	"github.com/pastelnetwork/pastel-utility/utils"
	"golang.org/x/term"
)

var (
	errSubCommandRequired           = fmt.Errorf("subcommand is required")
	errMasterNodeNameRequired       = fmt.Errorf("required --name, name of the Masternode to start and create in the masternode.conf if `--create` or `--update` are specified")
	errMasterNodeTxIDRequired       = fmt.Errorf("required --txid, transaction id of 5M collateral MN payment")
	errMasterNodeINDRequired        = fmt.Errorf("required --ind, output index in the transaction of 5M collateral MN payment")
	errMasterNodePwdRequired        = fmt.Errorf("required --passphrase <passphrase to pastelid private key>, if --pastelid is omitted")
	errMasterNodeSSHIPRequired      = fmt.Errorf("required if --coldhot is specified, SSH address of the remote HOT node")
	errMasterNodeColdNodeIPRequired = fmt.Errorf("required, WAN address of the host")
	errSetTestnet                   = fmt.Errorf("please initialize pastel.conf as testnet mode")
	errSetMainnet                   = fmt.Errorf("please initialize pastel.conf as mainnet mode")
	errGetExternalIP                = fmt.Errorf("cannot get external ip address")
	errNotFoundPastelCli            = fmt.Errorf("cannot find pastel-cli on SSH server")
	errNotFoundMNOutput             = fmt.Errorf("cannot find output of masternode")
)

var (
	flagInteractiveMode bool
	flagRestart         bool

	flagMasterNodeName       string
	flagMasterNodeIsTestNet  bool
	flagMasterNodeIsCreate   bool
	flagMasterNodeIsUpdate   bool
	flagMasterNodeTxID       string
	flagMasterNodeIND        string
	flagMasterNodeIP         string
	flagMasterNodePort       int
	flagMasterNodePrivateKey string
	flagMasterNodePastelID   string
	flagMasterNodePassPhrase string
	flagMasterNodeRPCIP      string
	flagMasterNodeRPCPort    int
	flagMasterNodeP2PIP      string
	flagMasterNodeP2PPort    int

	flagMasterNodeColdHot    bool
	flagMasterNodeSSHIP      string
	flagMasterNodeSSHPort    int
	flagMasterNodeColdNodeIP string
	flagMasterNodePastelPath string
)

func setupStartCommand() *cli.Command {
	config := configs.New()

	startCommand := cli.NewCommand("start")
	startCommand.SetUsage("usage")
	addLogFlags(startCommand, config)

	superNodeSubcommand := cli.NewCommand("supernode")
	superNodeSubcommand.CustomHelpTemplate = GetColoredSubCommandHeaders()
	superNodeSubcommand.SetUsage(cyan("Starts supernode"))
	superNodeSubcommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "superNodeSubCommand", config)
		if err != nil {
			return err
		}
		return runStartSuperNodeSubCommand(ctx, config)
	})
	superNodeFlags := []*cli.Flag{
		cli.NewFlag("i", &flagInteractiveMode),
		cli.NewFlag("r", &flagRestart),
	}
	superNodeSubcommand.AddFlags(superNodeFlags...)

	masterNodeSubCommand := cli.NewCommand("masternode")
	masterNodeSubCommand.CustomHelpTemplate = GetColoredSubCommandHeaders()
	masterNodeSubCommand.SetUsage(cyan("Starts master node"))
	masterNodeSubCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "masterNodeSubCommand", config)
		if err != nil {
			return err
		}
		return runStartMasterNodeSubCommand(ctx, config)
	})
	masterNodeFlags := []*cli.Flag{
		cli.NewFlag("i", &flagInteractiveMode),
		cli.NewFlag("r", &flagRestart),
		cli.NewFlag("name", &flagMasterNodeName).SetUsage("name of the Master node").SetRequired(),
		cli.NewFlag("testnet", &flagMasterNodeIsTestNet),
		cli.NewFlag("create", &flagMasterNodeIsCreate),
		cli.NewFlag("update", &flagMasterNodeIsUpdate),
		cli.NewFlag("txid", &flagMasterNodeTxID),
		cli.NewFlag("ind", &flagMasterNodeIND),
		cli.NewFlag("ip", &flagMasterNodeIP),
		cli.NewFlag("port", &flagMasterNodePort),
		cli.NewFlag("pkey", &flagMasterNodePrivateKey),
		cli.NewFlag("pastelid", &flagMasterNodePastelID),
		cli.NewFlag("passphrase", &flagMasterNodePassPhrase),
		cli.NewFlag("rpc-ip", &flagMasterNodeRPCIP),
		cli.NewFlag("rpc-port", &flagMasterNodeRPCPort),
		cli.NewFlag("p2p-ip", &flagMasterNodeP2PIP),
		cli.NewFlag("p2p-port", &flagMasterNodeP2PPort),
		cli.NewFlag("coldhot", &flagMasterNodeColdHot),
		cli.NewFlag("ssh-ip", &flagMasterNodeSSHIP),
		cli.NewFlag("ssh-port", &flagMasterNodeSSHPort),
		cli.NewFlag("coldnode-ip", &flagMasterNodeColdNodeIP),
		cli.NewFlag("pastelpath", &flagMasterNodePastelPath),
	}
	masterNodeSubCommand.AddFlags(masterNodeFlags...)

	nodeSubCommand := cli.NewCommand("node")
	nodeSubCommand.CustomHelpTemplate = GetColoredSubCommandHeaders()
	nodeSubCommand.SetUsage(cyan("Starts specified node"))
	nodeSubCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "nodeSubCommand", config)
		if err != nil {
			return err
		}
		return runStartNodeSubCommand(ctx, config)
	})
	nodeFlags := []*cli.Flag{
		cli.NewFlag("i", &flagInteractiveMode),
		cli.NewFlag("r", &flagRestart),
	}
	nodeSubCommand.AddFlags(nodeFlags...)

	walletSubCommand := cli.NewCommand("wallet")
	walletSubCommand.CustomHelpTemplate = GetColoredSubCommandHeaders()
	walletSubCommand.SetUsage(cyan("Starts wallet"))
	walletSubCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "nodeSubCommand", config)
		if err != nil {
			return err
		}
		return runStartWalletSubCommand(ctx, config)
	})
	walletFlags := []*cli.Flag{
		cli.NewFlag("i", &flagInteractiveMode),
		cli.NewFlag("r", &flagRestart),
	}
	walletSubCommand.AddFlags(walletFlags...)

	startCommand.AddSubcommands(
		superNodeSubcommand,
		masterNodeSubCommand,
		nodeSubCommand,
		walletSubCommand,
	)

	startCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "startcommand", config)
		if err != nil {
			return err
		}
		if len(args) == 0 {
			return errSubCommandRequired
		}
		return runStart(ctx, config)
	})
	return startCommand
}

func runStart(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Start")
	defer log.WithContext(ctx).Info("End")

	configJSON, err := config.String()
	if err != nil {
		return err
	}
	log.WithContext(ctx).Infof("Config: %s", configJSON)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sys.RegisterInterruptHandler(cancel, func() {
		log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
		os.Exit(0)
	})

	// actions to run goes here

	return nil
}

func runStartNodeSubCommand(_ context.Context, _ *configs.Config) error {
	// TODO: Implement start node command
	panic("")
}

func runStartSuperNodeSubCommand(_ context.Context, _ *configs.Config) error {
	// TODO: Implement start supper node command
	panic("")
}

func runStartMasterNodeSubCommand(ctx context.Context, config *configs.Config) error {
	if flagMasterNodeColdHot {
		return runMasterNodOnColdHot(ctx, config)
	}
	return runMasterNodOnHotHot(ctx, config)
}

func runStartWalletSubCommand(_ context.Context, _ *configs.Config) error {
	// TODO: Implement wallet command
	panic("")
}

func runMasterNodOnHotHot(ctx context.Context, config *configs.Config) error {
	var masternodePrivKey, pastelid, output string
	var err error

	// *************  1. Parse parameters  *************
	fmt.Printf("Checking parameters...\n")
	if err := checkStartMasterNodeParams(ctx, config); err != nil {
		return err
	}
	fmt.Printf("Finished checking parameters!\n")

	fmt.Printf("Checking pastel config...\n")
	if err := CheckPastelConf(config); err != nil {
		return err
	}
	fmt.Printf("Finished checking pastel config!\n")

	// If create master node using HOT/HOT wallet
	if flagMasterNodeIsCreate || flagMasterNodeIsUpdate {

		if flagMasterNodeIsCreate {
			if err = backupConfFile(); err != nil { // delete conf file
				return err
			}
			fmt.Printf("Backup masternode.conf was finished successfully.\n")
			// *************  2.1 Start the Pastel Network Node  *************
			fmt.Printf("Starting pasteld...\npasteld --externalip=%s --reindex --daemon\n", flagMasterNodeIP)
			go RunPasteld(fmt.Sprintf("--externalip=%s", flagMasterNodeIP), "--reindex", "--daemon")

			var failCnt = 0
			for {
				if output, err = runPastelCLI("getaccountaddress", ""); err != nil {
					fmt.Printf("Waiting the pasteld to be started ...\n")
					time.Sleep(10000 * time.Millisecond)
					failCnt++
					if failCnt == 10 {
						fmt.Printf("Can not start with pasteld\n")
						return err
					}
				} else {
					fmt.Printf("Started pasteld successfully!\nHot wallet address = %s\n", output)
					break
				}
			}

			// *************  2.2 Search collateral transaction *************
			if output, err = runPastelCLI("masternode", "outputs"); err != nil {
				fmt.Printf("Cannot find masternode output.\n")
				return err
			}
			var recMasterNode map[string]interface{}
			json.Unmarshal([]byte(output), &recMasterNode)

			if len(recMasterNode) != 0 {
				if recMasterNode[flagMasterNodeTxID] != nil && recMasterNode[flagMasterNodeTxID] == flagMasterNodeIND {
					// if receives PSL go to next step
					fmt.Printf("masternode outputs = %s, %s\n", flagMasterNodeTxID, flagMasterNodeIND)
				} else {
					fmt.Printf("Cannot find masternode output.\n")
					return errNotFoundMNOutput
				}
			}

			// *************  2.3 create new MN private key  *************
			if len(flagMasterNodePrivateKey) == 0 {
				if masternodePrivKey, err = runPastelCLI("masternode", "genkey"); err != nil {
					return err
				}
			} else {
				masternodePrivKey = flagMasterNodePrivateKey
			}

			fmt.Printf("masternode private key = %s\n", masternodePrivKey)

			if len(flagMasterNodePastelID) == 0 && len(flagMasterNodePassPhrase) != 0 {
				//  *************  2.4 create new pastelid  *************
				if output, err = runPastelCLI("pastelid", "newkey", flagMasterNodePassPhrase); err != nil {
					return err
				} // generate a PastelID
				var pastelidSt structure.RPCPastelID
				if err = json.Unmarshal([]byte(output), &pastelidSt); err != nil {
					return err
				}
				pastelid = pastelidSt.Pastelid
			} else {
				pastelid = flagMasterNodePastelID
			}

			fmt.Printf("pastelid = %s\n", pastelid)

			failCnt = 0

			for {
				if output, err = runPastelCLI("getaccountaddress", ""); err != nil {
					fmt.Printf("Waiting the pasteld to be started ...\n")
					time.Sleep(10000 * time.Millisecond)
					failCnt++
					if failCnt == 10 {
						fmt.Printf("Can not start with pasteld\n")
						return err
					}
				} else {
					fmt.Printf("master node address = %s\n", output)
					break
				}
			}

			// *************  2.5 Create or edit masternode.conf  *************
			confData := map[string]interface{}{
				flagMasterNodeName: map[string]string{
					"mnAddress":  flagMasterNodeIP + ":" + fmt.Sprintf("%d", flagMasterNodePort),
					"mnPrivKey":  masternodePrivKey,
					"txid":       flagMasterNodeTxID,
					"outIndex":   flagMasterNodeIND,
					"extAddress": flagMasterNodeIP + ":" + fmt.Sprintf("%d", flagMasterNodeRPCPort),
					"p2pAddress": flagMasterNodeP2PIP + ":" + fmt.Sprintf("%d", flagMasterNodeP2PPort),
					"extCfg":     "",
					"extKey":     pastelid,
				},
			}
			data, _ := json.Marshal(confData)

			// Create masternode.conf file
			if err = createConfFile(data); err != nil {
				return err
			}
			fmt.Printf("masternode.conf = %s\n", string(data))

			fmt.Printf("Stopping pasteld...\n")
			if _, err = runPastelCLI("stop"); err != nil {
				return err
			}
		}

		if flagMasterNodeIsUpdate {
			// Make masternode conf data
			confData := map[string]interface{}{
				flagMasterNodeName: map[string]string{
					"mnAddress":  flagMasterNodeIP + ":" + fmt.Sprintf("%d", flagMasterNodePort),
					"mnPrivKey":  masternodePrivKey,
					"txid":       flagMasterNodeTxID,
					"outIndex":   flagMasterNodeIND,
					"extAddress": flagMasterNodeIP + ":" + fmt.Sprintf("%d", flagMasterNodeRPCPort),
					"p2pAddress": flagMasterNodeP2PIP + ":" + fmt.Sprintf("%d", flagMasterNodeP2PPort),
					"extCfg":     "",
					"extKey":     pastelid,
				},
			}

			// Create masternode.conf file
			if _, err = updateMasternodeConfFile(confData); err != nil {
				return err
			}

			data, _ := json.Marshal(confData)
			fmt.Printf("masternode.conf = %s\n", string(data))
		}
	}

	// Get conf data from masternode.conf File
	var nodeName, privKey, extIP = getStartInfo()

	// *************  3. Start Node as Masternode  *************
	go RunPasteld("-masternode", "-txindex=1", "-reindex", fmt.Sprintf("-masternodeprivkey=%s", privKey), fmt.Sprintf("--externalip=%s", extIP))

	// *************  4. Wait for blockchain and masternodes sync  *************
	var mnstatus structure.RPCPastelMSStatus
	var failCnt = 0

	for {
		if output, err = runPastelCLI("mnsync", "status"); err != nil {
			fmt.Printf("Waiting the pasteld to be started ...\n")
			time.Sleep(10000 * time.Millisecond)
			failCnt++
			if failCnt >= 10 {
				fmt.Printf("Can not start with pasteld\n")
				return err
			}
		} else {
			// Master Node Output
			if err = json.Unmarshal([]byte(output), &mnstatus); err != nil {
				return err
			}

			if mnstatus.AssetName == "Initial" {
				if _, err = runPastelCLI("mnsync", "reset"); err != nil {
					fmt.Printf("master node reset was failed\n")
					return err
				}
				time.Sleep(10000 * time.Millisecond)
			}
			if mnstatus.IsSynced {
				fmt.Printf("master node was synced!\n")
				break
			}
			fmt.Printf("master node was not synced!!!\nWaiting for sync...")
			time.Sleep(10000 * time.Millisecond)
		}
	}

	// *************  5. Enable Masternode  ***************
	failCnt = 0
	for {
		if output, err = runPastelCLI("masternode", "start-alias", nodeName); err != nil {
			fmt.Printf("Waiting the pasteld to be started ...\n")
			time.Sleep(10000 * time.Millisecond)
			failCnt++
			if failCnt == 10 {
				return err
			}
		} else {
			fmt.Printf("The pasteld was started successfully...\n")
			fmt.Printf("masternode alias status = %s\n", output)
			break
		}

	}

	return nil
}

func runMasterNodOnColdHot(ctx context.Context, config *configs.Config) error {
	var masternodePrivKey, pastelid, output string
	var err error

	fmt.Printf("Checking parameters...\n")
	if err := checkStartMasterNodeParams(ctx, config); err != nil {
		fmt.Printf("Checking parameters occurs error -> %s\n", err)
		return err
	}
	fmt.Printf("Finished checking parameters!\n")
	fmt.Printf("Checking pastel config...\n")
	if err := CheckPastelConf(config); err != nil {
		return err
	}
	fmt.Printf("Finished checking pastel config!\n")
	// If create master node using HOT/HOT wallet
	if flagMasterNodeIsCreate || flagMasterNodeIsUpdate {

		if flagMasterNodeIsCreate {
			if err = backupConfFile(); err != nil { // delete conf file
				return err
			}

			fmt.Printf("Start pasteld\n./pasteld --externalip=%s --reindex --daemon\n", flagMasterNodeColdNodeIP)
			go RunPasteld(fmt.Sprintf("--externalip=%s", flagMasterNodeColdNodeIP), "--reindex", "--daemon")

			var failCnt = 0
			for {
				if output, err = runPastelCLI("getaccountaddress", ""); err != nil {
					fmt.Printf("Waiting the pasteld to be started ...\n")
					time.Sleep(10000 * time.Millisecond)
					failCnt++
					if failCnt == 10 {
						fmt.Printf("Can not start with pasteld\n")
						return err
					}
				} else {
					fmt.Printf("Hot wallet address = %s\n", output)
					break
				}
			}

			// ***************  3.1 Search collateral transaction  ***************
			for {
				if output, err = runPastelCLI("masternode", "outputs"); err != nil {
					fmt.Printf("masternode outputs\n")
					return err
				}
				var recMasterNode map[string]interface{}
				json.Unmarshal([]byte(output), &recMasterNode)

				if len(recMasterNode) != 0 {
					if recMasterNode[flagMasterNodeTxID] != nil && recMasterNode[flagMasterNodeTxID] == flagMasterNodeIND {
						// if receives PSL go to next step
						fmt.Printf("masternode outputs = %s, %s\n", flagMasterNodeTxID, flagMasterNodeIND)
						break
					}
				}

				time.Sleep(10000 * time.Millisecond)
			}

			// ***************  3.2 Create new MN private key  ***************
			if len(flagMasterNodePrivateKey) == 0 {
				if masternodePrivKey, err = runPastelCLI("masternode", "genkey"); err != nil {
					return err
				}
			} else {
				masternodePrivKey = flagMasterNodePrivateKey
			}
			fmt.Printf("masternode private key = %s\n", masternodePrivKey)

			// ***************  3.3 create new pastelid  ***************
			if len(flagMasterNodePastelID) == 0 && len(flagMasterNodePassPhrase) != 0 {
				// Check masternode status
				var mnstatus structure.RPCPastelMSStatus

				for {
					if output, err = runPastelCLI("mnsync", "status"); err != nil {
						fmt.Printf("Waiting the pasteld to be started ...\n")
						time.Sleep(10000 * time.Millisecond)
						failCnt++
						if failCnt >= 10 {
							fmt.Printf("Can not start with pasteld\n")
							return err
						}
					} else {
						// Master Node Output
						if err = json.Unmarshal([]byte(output), &mnstatus); err != nil {
							return err
						}
						if mnstatus.AssetName == "Initial" {
							fmt.Printf("master node resets status\n")
							if _, err = runPastelCLI("mnsync", "reset"); err != nil {
								fmt.Printf("master node reset was failed\n")
								return err
							}
							time.Sleep(10000 * time.Millisecond)
						} else {
							if mnstatus.IsSynced {
								fmt.Printf("master node was synced!\n")
								break
							}
							fmt.Printf("Waiting for sync...")
							time.Sleep(10000 * time.Millisecond)
						}
					}
				}

				if output, err = runPastelCLI("pastelid", "newkey", flagMasterNodePassPhrase); err != nil {
					return err
				}

				var pastelidSt structure.RPCPastelID
				if err = json.Unmarshal([]byte(output), &pastelidSt); err != nil {
					return err
				}
				pastelid = pastelidSt.Pastelid
			} else {
				pastelid = flagMasterNodePastelID
			}

			fmt.Printf("pastelid = %s\n", pastelid)

			fmt.Printf("Stopping pasteld...\n")
			if _, err = runPastelCLI("stop"); err != nil {
				return err
			}
			time.Sleep(5000 * time.Millisecond)
			fmt.Printf("Stopped pasteld\n")
			// ***************  3.4 Create or edit masternode.conf - this should be NEW masternode.conf, any existing should be backed up  ***************
			// Make masternode conf data
			confData := map[string]interface{}{
				flagMasterNodeName: map[string]string{
					"mnAddress":  flagMasterNodeIP + ":" + fmt.Sprintf("%d", flagMasterNodePort),
					"mnPrivKey":  masternodePrivKey,
					"txid":       flagMasterNodeTxID,
					"outIndex":   flagMasterNodeIND,
					"extAddress": flagMasterNodeIP + ":" + fmt.Sprintf("%d", flagMasterNodeRPCPort),
					"p2pAddress": flagMasterNodeP2PIP + ":" + fmt.Sprintf("%d", flagMasterNodeP2PPort),
					"extCfg":     "",
					"extKey":     pastelid,
				},
			}
			data, _ := json.Marshal(confData)

			// Create masternode.conf file
			if err = createConfFile(data); err != nil {
				return err
			}
			fmt.Println(string(data))
		}

		if flagMasterNodeIsUpdate {
			// Make masternode conf data
			confData := map[string]interface{}{
				flagMasterNodeName: map[string]string{
					"mnAddress":  flagMasterNodeIP + ":" + fmt.Sprintf("%d", flagMasterNodePort),
					"mnPrivKey":  masternodePrivKey,
					"txid":       flagMasterNodeTxID,
					"outIndex":   flagMasterNodeIND,
					"extAddress": flagMasterNodeIP + ":" + fmt.Sprintf("%d", flagMasterNodeRPCPort),
					"p2pAddress": flagMasterNodeP2PIP + ":" + fmt.Sprintf("%d", flagMasterNodeP2PPort),
					"extCfg":     "",
					"extKey":     pastelid,
				},
			}

			// Create masternode.conf file
			if _, err = updateMasternodeConfFile(confData); err != nil {
				return err
			}
		}
	}

	// ***************  4. Execute following commands over SSH on the remote node (using ssh-ip and ssh-port)  ***************
	username, password, _ := credentials()

	if err = remoteHotNodeCtrl(username, password); err != nil {
		fmt.Printf("%s\n", err)
		return err
	}
	fmt.Printf("The hot wallet node has been successfully launched!\n")
	// ***************  5. Enable Masternode  ***************
	// Get conf data from masternode.conf File
	var _, _, extIP = getStartInfo()

	// Start Node as Masternode
	go RunPasteld("-txindex=1", "-reindex", fmt.Sprintf("--externalip=%s", extIP))

	var mnstatus structure.RPCPastelMSStatus
	var failCnt = 0

	for {
		if output, err = runPastelCLI("mnsync", "status"); err != nil {
			fmt.Printf("Waiting the pasteld to be started ...\n")
			time.Sleep(10000 * time.Millisecond)
			failCnt++
			if failCnt >= 10 {
				fmt.Printf("Can not start with pasteld\n")
				return err
			}
		} else {
			// Master Node Output
			if err = json.Unmarshal([]byte(output), &mnstatus); err != nil {
				return err
			}

			if mnstatus.AssetName == "Initial" {
				if _, err = runPastelCLI("mnsync", "reset"); err != nil {
					fmt.Printf("master node reset was failed\n")
					return err
				}
				time.Sleep(10000 * time.Millisecond)
			}
			if mnstatus.IsSynced {
				fmt.Printf("master node was synced!\n")
				break
			}
			fmt.Printf("Waiting for sync...")
			time.Sleep(10000 * time.Millisecond)
		}
	}

	if output, err = runPastelCLI("masternode", "start-alias", flagMasterNodeName); err != nil {
		return err
	}

	fmt.Printf("masternode alias status = %s\n", output)

	// ***************  6. Stop Cold Node  ***************
	if _, err = runPastelCLI("stop"); err != nil {
		return err
	}
	return nil
}

func remoteHotNodeCtrl(username string, password string) error {
	var output []byte
	var pastelCliPath string
	fmt.Printf("Connecting to SSH Hot node wallet -> %s:%d...\n", flagMasterNodeSSHIP, flagMasterNodeSSHPort)
	client, err := utils.DialWithPasswd(fmt.Sprintf("%s:%d", flagMasterNodeSSHIP, flagMasterNodeSSHPort), username, password)
	if err != nil {
		return err
	}
	defer client.Close()

	// Find pasteld
	fmt.Printf("Check pasteld default path...\n")
	out, err := client.Cmd(fmt.Sprintf("test -e %s && echo file exists || echo file not found", flagMasterNodePastelPath)).Output()
	if err != nil {
		return err
	}

	if strings.TrimRight(string(out), "\n") != "file exists" {
		fmt.Printf("Finding pasteld executable on $HOME path\n")
		out, err = client.Cmd("find $HOME -iname pasteld").Output()
		if err != nil {
			return err
		}

		pastelPaths := strings.Split(string(out), "\n")

		if len(pastelPaths) > 0 {
			fmt.Printf("%s\n\n", string(out))
			for {
				indexStr, err := readstrings("Please input index of pasteld path to use")
				if err != nil {
					return err
				}

				index, err := strconv.Atoi(indexStr)
				if err != nil {
					return err
				}

				if len(pastelPaths) < index {
					fmt.Printf("Please input index correctly\n")
				} else {
					flagMasterNodePastelPath = pastelPaths[index]
					break
				}
			}
		}

	} else {
		fmt.Printf("Found pasteld on default path!\n")
	}

	fmt.Printf("Checking pastel-cli path...\n")
	out, err = client.Cmd("find $HOME -iname pastel-cli").Output()
	if err != nil {
		return err
	}

	pastelCliPaths := strings.Split(string(out), "\n")

	if len(pastelCliPaths) == 0 {
		return errNotFoundPastelCli
	}

	pastelCliPath = pastelCliPaths[0]

	fmt.Printf("Found pastel-cli path on %s\n", pastelCliPath)

	go client.Cmd(fmt.Sprintf("%s --reindex --externalip=%s --daemon", flagMasterNodePastelPath, flagMasterNodeIP)).Run()

	time.Sleep(10000 * time.Millisecond)

	var mnstatus structure.RPCPastelMSStatus
	failCnt := 0

	for {
		if output, err = client.Cmd(fmt.Sprintf("%s mnsync status", pastelCliPath)).Output(); err != nil {
			fmt.Printf("Waiting the pasteld to be started ...\n")
			time.Sleep(10000 * time.Millisecond)
			failCnt++
			if failCnt >= 10 {
				fmt.Printf("Can not start with pasteld\n")
				return err
			}
		} else {
			// Master Node Output
			if err = json.Unmarshal([]byte(output), &mnstatus); err != nil {
				return err
			}
			if mnstatus.AssetName == "Initial" {
				if _, err = client.Cmd(fmt.Sprintf("%s mnsync reset", pastelCliPath)).Output(); err != nil {
					fmt.Printf("master node reset was failed\n")
					return err
				}
				time.Sleep(10000 * time.Millisecond)
			} else {
				if mnstatus.IsSynced {
					fmt.Printf("master node was synced!\n")
					break
				}
				fmt.Printf("Waiting for sync...")
				time.Sleep(10000 * time.Millisecond)
			}
		}
	}

	if _, err = client.Cmd(fmt.Sprintf("%s stop", pastelCliPath)).Output(); err != nil {
		fmt.Printf("Error - stop pasteld\n")
		return err
	}

	time.Sleep(5000 * time.Millisecond)

	cmdLine := fmt.Sprintf("%s --masternode --txindex=1 --reindex --masternodeprivkey=%s --externalip=%s --daemon", flagMasterNodePastelPath, flagMasterNodePrivateKey, flagMasterNodeIP)
	fmt.Printf("%s\n", cmdLine)
	go client.Cmd(cmdLine).Run()

	time.Sleep(10000 * time.Millisecond)

	failCnt = 0

	for {
		if output, err = client.Cmd(fmt.Sprintf("%s mnsync status", pastelCliPath)).Output(); err != nil {
			fmt.Printf("Waiting the pasteld to be started ...\n")
			time.Sleep(10000 * time.Millisecond)
			failCnt++
			if failCnt >= 10 {
				fmt.Printf("Can not start with pasteld\n")
				return err
			}
		} else {
			// Master Node Output
			if err = json.Unmarshal([]byte(output), &mnstatus); err != nil {
				return err
			}
			if mnstatus.AssetName == "Initial" {
				if _, err = client.Cmd(fmt.Sprintf("%s mnsync reset", pastelCliPath)).Output(); err != nil {
					fmt.Printf("master node reset was failed\n")
					return err
				}
				time.Sleep(10000 * time.Millisecond)
			} else {
				if mnstatus.IsSynced {
					fmt.Printf("master node was synced!\n")
					break
				}
				fmt.Printf("Waiting for sync...")
				time.Sleep(10000 * time.Millisecond)
			}
		}
	}

	return nil
}

func readstrings(comment string) (string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s-> ", comment)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(line), nil
}

func credentials() (string, string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}

	fmt.Print("Enter Password: ")
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", "", err
	}

	password := string(bytePassword)
	return strings.TrimSpace(username), strings.TrimSpace(password), nil
}

func checkStartMasterNodeParams(_ context.Context, _ *configs.Config) error {
	if len(flagMasterNodeName) == 0 {
		return errMasterNodeNameRequired
	}

	if len(flagMasterNodeIP) == 0 {
		if flagMasterNodeColdHot {
			return errGetExternalIP
		}

		externalIP, err := GetExternalIPAddress()

		if err != nil {
			return errGetExternalIP
		}
		flagMasterNodeIP = externalIP
	}

	if flagMasterNodeIsCreate || flagMasterNodeIsUpdate {
		if len(flagMasterNodeTxID) == 0 {
			return errMasterNodeTxIDRequired
		}

		if len(flagMasterNodeIND) == 0 {
			return errMasterNodeINDRequired
		}

		if len(flagMasterNodePastelID) == 0 {
			if len(flagMasterNodePassPhrase) == 0 {
				return errMasterNodePwdRequired
			}
		}
	}

	if flagMasterNodeColdHot {
		if len(flagMasterNodeSSHIP) == 0 {
			return errMasterNodeSSHIPRequired
		}

		if len(flagMasterNodeColdNodeIP) == 0 {
			return errMasterNodeColdNodeIPRequired
		}

		flagMasterNodeSSHPort = func() int {
			if flagMasterNodeSSHPort == 0 {
				return 22
			}
			return flagMasterNodeSSHPort
		}()

		flagMasterNodePastelPath = func() string {
			if len(flagMasterNodePastelPath) == 0 {
				return "$HOME/pastel/pasteld"
			}
			return flagMasterNodePastelPath
		}()
	}

	flagMasterNodeRPCIP = func() string {
		if len(flagMasterNodeRPCIP) == 0 {
			return flagMasterNodeIP
		}
		return flagMasterNodeRPCIP
	}()
	flagMasterNodeP2PIP = func() string {
		if len(flagMasterNodeP2PIP) == 0 {
			return flagMasterNodeIP
		}
		return flagMasterNodeP2PIP
	}()

	if flagMasterNodeIsTestNet {
		flagMasterNodePort = func() int {
			if flagMasterNodePort == 0 {
				return 19933
			}
			return flagMasterNodePort
		}()
		flagMasterNodeRPCPort = func() int {
			if flagMasterNodeRPCPort == 0 {
				return 14444
			}
			return flagMasterNodeRPCPort
		}()
		flagMasterNodeP2PPort = func() int {
			if flagMasterNodeP2PPort == 0 {
				return 14445
			}
			return flagMasterNodeP2PPort
		}()
	} else {
		flagMasterNodePort = func() int {
			if flagMasterNodePort == 0 {
				return 9933
			}
			return flagMasterNodePort
		}()
		flagMasterNodeRPCPort = func() int {
			if flagMasterNodeRPCPort == 0 {
				return 4444
			}
			return flagMasterNodeRPCPort
		}()
		flagMasterNodeP2PPort = func() int {
			if flagMasterNodeP2PPort == 0 {
				return 4445
			}
			return flagMasterNodeP2PPort
		}()
	}
	return nil
}

// GetExternalIPAddress runs shell command and returns external IP address
func GetExternalIPAddress() (externalIP string, err error) {
	return RunCMD("curl", "ipinfo.io/ip")
}

// RunPasteld runs pasteld
func RunPasteld(args ...string) (output string, err error) {
	if flagMasterNodeIsTestNet {
		args = append(args, "--testnet")
		output, err = RunCMD("./pasteld", args...)
	} else {
		output, err = RunCMD("./pasteld", args...)
	}
	return output, err
}

// Run pastel-cli
func runPastelCLI(args ...string) (output string, err error) {
	return RunCMD("./pastel-cli", args...)
}

// Create or Update masternode.conf File
func createConfFile(confData []byte) (err error) {
	workDirPath := configurer.DefaultWorkingDir()
	var masternodeConfPath, masternodeConfPathBackup string

	if flagMasterNodeIsTestNet {
		masternodeConfPath = workDirPath + "/testnet3/masternode.conf"
		masternodeConfPathBackup = workDirPath + "/testnet3/masternode_%s.conf"
	} else {
		masternodeConfPath = workDirPath + "/masternode.conf"
		masternodeConfPathBackup = workDirPath + "/masternode_%s.conf"
	}
	if _, err := os.Stat(masternodeConfPath); err == nil { // if masternode.conf File exists , backup
		oldFileName := masternodeConfPath
		currentTime := time.Now()
		backupFileName := fmt.Sprintf(masternodeConfPathBackup, currentTime.Format("2021-01-01 23:59:59"))
		err := os.Rename(oldFileName, backupFileName)
		if err != nil {
			return err
		}

	}

	confFile, err := os.Create(masternodeConfPath)
	confFile.Write(confData)
	if err != nil {
		return err
	}
	defer confFile.Close()

	return nil
}

func updateMasternodeConfFile(confData map[string]interface{}) (result bool, err error) {
	workDirPath := configurer.DefaultWorkingDir()
	var masternodeConfPath string

	if flagMasterNodeIsTestNet {
		masternodeConfPath = workDirPath + "/testnet3/masternode.conf"
	} else {
		masternodeConfPath = workDirPath + "/masternode.conf"
	}

	// Read ConfData from masternode.conf
	confFile, err := ioutil.ReadFile(masternodeConfPath)
	if err != nil {
		return false, err
	}

	var conf map[string]interface{}

	json.Unmarshal([]byte(confFile), &conf)

	for k := range confData {
		if conf[k] != nil {
			confDataValue := confData[k].(map[string]string)
			confValue := conf[k].(map[string]interface{})
			for itemKey := range confDataValue {
				if len(confDataValue[itemKey]) != 0 {
					confValue[itemKey] = confDataValue[itemKey]

				}
			}
		}
	}

	var updatedConf []byte
	if updatedConf, err = json.Marshal(conf); err != nil {
		fmt.Printf("updated conf = %s", updatedConf)
		return false, err
	}

	if ioutil.WriteFile(masternodeConfPath, updatedConf, 0644) != nil {
		return false, err
	}

	return true, nil
}

func backupConfFile() (err error) {
	workDirPath := configurer.DefaultWorkingDir()
	var masternodeConfPath, masternodeConfPathBackup string

	if flagMasterNodeIsTestNet {
		masternodeConfPath = workDirPath + "/testnet3/masternode.conf"
		masternodeConfPathBackup = workDirPath + "/testnet3/masternode_%s.conf"
	} else {
		masternodeConfPath = workDirPath + "/masternode.conf"
		masternodeConfPathBackup = workDirPath + "/masternode_%s.conf"
	}
	if _, err := os.Stat(masternodeConfPath); err == nil { // if masternode.conf File exists , backup
		oldFileName := masternodeConfPath
		currentTime := time.Now()
		backupFileName := fmt.Sprintf(masternodeConfPathBackup, currentTime.Format("2021-01-01 23:59:59"))
		if err := os.Rename(oldFileName, backupFileName); err != nil {
			return err
		}

	}
	if _, err := os.Stat(masternodeConfPath); err == nil { // if masternode.conf File exists , backup
		if err = os.Remove(masternodeConfPath); err != nil {
			return err
		}

	}

	return nil
}

func getStartInfo() (nodeName string, privKey string, extIP string) {
	workDirPath := configurer.DefaultWorkingDir()
	var masternodeConfPath string

	if flagMasterNodeIsTestNet {
		masternodeConfPath = workDirPath + "/testnet3/masternode.conf"
	} else {
		masternodeConfPath = workDirPath + "/masternode.conf"
	}

	// Read ConfData from masternode.conf
	confFile, err := ioutil.ReadFile(masternodeConfPath)
	if err != nil {
		return "", "", ""
	}

	var conf map[string]interface{}
	json.Unmarshal([]byte(confFile), &conf)

	for key := range conf {
		nodeName = key // get Node Name
	}
	confData := conf[nodeName].(map[string]interface{})
	extAddr := strings.Split(confData["mnAddress"].(string), ":") // get Ext IP
	return nodeName, confData["mnPrivKey"].(string), extAddr[0]
}

// CheckPastelConf check configuration of pastel settings.
func CheckPastelConf(_ *configs.Config) (err error) {
	workDirPath := configurer.DefaultWorkingDir()

	if _, err := os.Stat(workDirPath); os.IsNotExist(err) {
		return err
	}

	if _, err := os.Stat(workDirPath + "/pastel.conf"); os.IsNotExist(err) {
		return err
	}

	if flagMasterNodeIsTestNet {
		var file, err = os.OpenFile(workDirPath+"/pastel.conf", os.O_RDWR, 0644)
		if err != nil {
			return err
		}
		defer file.Close()

		configure, err := ioutil.ReadAll(file)

		if err != nil {
			return err
		}

		if !strings.Contains(string(configure), "testnet=1") {
			return errSetTestnet
		}
	} else {
		var file, err = os.OpenFile(workDirPath+"/pastel.conf", os.O_RDWR, 0644)
		if err != nil {
			return err
		}
		defer file.Close()

		configure, err := ioutil.ReadAll(file)
		if err != nil {
			return err
		}

		if strings.Contains(string(configure), "testnet=1") {
			return errSetMainnet
		}
	}

	return nil
}
