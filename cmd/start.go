package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/pastel-utility/constants"
	"github.com/pastelnetwork/pastel-utility/structure"
	"github.com/pastelnetwork/pastel-utility/utils"

	"golang.org/x/term"
)

var (
	errSubCommandRequired      = fmt.Errorf("subcommand is required")
	errMasterNodeNameRequired  = fmt.Errorf("required --name, name of the Masternode to start and create in the masternode.conf if `--create` or `--update` are specified")
	errMasterNodeTxIDRequired  = fmt.Errorf("required --txid, transaction id of 5M collateral MN payment")
	errMasterNodeINDRequired   = fmt.Errorf("required --ind, output index in the transaction of 5M collateral MN payment")
	errMasterNodePwdRequired   = fmt.Errorf("required --passphrase <passphrase to pastelid private key>, if --pastelid is omitted")
	errMasterNodeSSHIPRequired = fmt.Errorf("required if --coldhot is specified, SSH address of the remote HOT node")
	// errMasterNodeColdNodeIPRequired = fmt.Errorf("required, WAN address of the host")
	errSetTestnet        = fmt.Errorf("please initialize pastel.conf as testnet mode")
	errSetMainnet        = fmt.Errorf("please initialize pastel.conf as mainnet mode")
	errGetExternalIP     = fmt.Errorf("cannot get external ip address")
	errNotFoundPastelCli = fmt.Errorf("cannot find pastel-cli on SSH server")
	// errNotFoundMNOutput             = fmt.Errorf("cannot find output of masternode")
	errNotFoundPastelPath = fmt.Errorf("cannot find pasteld/pastel-cli path. please install node")
	errNetworkModeInvalid = fmt.Errorf("invalid network mode")
)

var (
	flagInteractiveMode bool
	flagRestart         bool

	// node flags
	flagNodeExtIP string
	flagReIndex   bool

	// masternode flags
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

	flagMasterNodeColdHot       bool
	flagMasterNodeSSHIP         string
	flagMasterNodeSSHPort       int
	flagMasterNodeColdNodeIP    string
	flagMasterNodePastelPath    string
	flagMasterNodeSupernodePath string


)

func setupStartCommand() *cli.Command {
	config := configs.GetConfig()

	startCommand := cli.NewCommand("start")
	startCommand.SetUsage("usage")
	addLogFlags(startCommand, config)

	superNodeSubCommand := cli.NewCommand("supernode")
	superNodeSubCommand.CustomHelpTemplate = GetColoredSubCommandHeaders()
	superNodeSubCommand.SetUsage(cyan("Starts supernode"))
	superNodeSubCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "superNodeSubCommand", config)
		if err != nil {
			return err
		}
		return runStartSuperNodeSubCommand(ctx, config)
	})
	masterNodeFlags := []*cli.Flag{
		cli.NewFlag("i", &flagInteractiveMode),
		cli.NewFlag("r", &flagRestart),
		cli.NewFlag("name", &flagMasterNodeName).
			SetUsage("name of the Master node").SetRequired(),
		cli.NewFlag("testnet", &flagMasterNodeIsTestNet),
		cli.NewFlag("create", &flagMasterNodeIsCreate).
			SetUsage(green("Optional, if specified, will create Masternode record in the masternode.conf.")),
		cli.NewFlag("update", &flagMasterNodeIsUpdate).
			SetUsage(green("Optional, if specified, will create Masternode record in the masternode.conf.")),
		cli.NewFlag("txid", &flagMasterNodeTxID).
			SetUsage(green("collateral payment txid , transaction id of 5M collateral MN payment")),
		cli.NewFlag("ind", &flagMasterNodeIND).
			SetUsage(green("collateral payment output index , output index in the transaction of 5M collateral MN payment")),
		cli.NewFlag("ip", &flagMasterNodeIP).
			SetUsage(green("WAN address of the host")).SetRequired(),
		cli.NewFlag("port", &flagMasterNodePort).
			SetUsage(green("Port for WAN IP address of the node - Optional, default - 9933 (19933 for Testnet)")),
		cli.NewFlag("pkey", &flagMasterNodePrivateKey).
			SetUsage(green("Pmasternode priv key- Optional, if omitted, new masternode private key will be created")),
		cli.NewFlag("pastelid", &flagMasterNodePastelID).
			SetUsage(green("pastelid of the Masternode. Optional, if omitted, new pastelid will be created and registered")),
		cli.NewFlag("passphrase", &flagMasterNodePassPhrase).
			SetUsage(green("passphrase to pastelid private key, Required, if pastelid is omitted")),
		cli.NewFlag("rpc-ip", &flagMasterNodeRPCIP).
			SetUsage(green("supernode IP address - Optional, if omitted, value passed to --ip will be used")),
		cli.NewFlag("rpc-port", &flagMasterNodeRPCPort).
			SetUsage(green("supernode port - Optional, default - 4444 (14444 for Testnet")),
		cli.NewFlag("p2p-ip", &flagMasterNodeP2PIP).
			SetUsage(green("Kademlia IP address - Optional, if omitted, value passed to --ip will be used")),
		cli.NewFlag("p2p-port", &flagMasterNodeP2PPort).
			SetUsage(green("Kademlia port - Optional, default - 4445 (14445 for Testnet)")),
		cli.NewFlag("coldhot", &flagMasterNodeColdHot),
		cli.NewFlag("ssh-ip", &flagMasterNodeSSHIP),
		cli.NewFlag("ssh-port", &flagMasterNodeSSHPort).SetValue(22),
		cli.NewFlag("coldnode-ip", &flagMasterNodeColdNodeIP),
		cli.NewFlag("pastelpath", &flagMasterNodePastelPath),
		cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
			SetUsage(green("Location where to create working directory")).SetValue(config.WorkingDir),
		cli.NewFlag("network", &config.Network).SetAliases("n").
			SetUsage(green("Network type, can be - \"mainnet\" or \"testnet\"")).SetValue("mainnet"),
	}
	superNodeSubCommand.AddFlags(masterNodeFlags...)

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
		cli.NewFlag("reindex", &flagReIndex),
		cli.NewFlag("ip", &flagNodeExtIP).
			SetUsage(green("WAN address of the host")).SetRequired(),
		cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
			SetUsage(green("Location where to create working directory")).SetValue(config.WorkingDir),
		cli.NewFlag("network", &config.Network).SetAliases("n").
			SetUsage(green("Network type, can be - \"mainnet\" or \"testnet\"")).SetValue("mainnet"),
	}
	nodeSubCommand.AddFlags(nodeFlags...)

	walletSubCommand := cli.NewCommand("walletnode")
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
		cli.NewFlag("release", &flagRestart).SetAliases("r").
			SetUsage(green("Optional, release version to install, default is the latest")),
		cli.NewFlag("reindex", &flagReIndex),
		cli.NewFlag("ip", &flagNodeExtIP).
			SetUsage(green("WAN address of the host")).SetRequired(),
		cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
			SetUsage(green("Location where to create working directory")).SetValue(config.WorkingDir),
		cli.NewFlag("network", &config.Network).SetAliases("n").
			SetUsage(green("Network type, can be - \"mainnet\" or \"testnet\"")).SetValue("mainnet"),
	}
	walletSubCommand.AddFlags(walletFlags...)

	startCommand.AddSubcommands(
		superNodeSubCommand,
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

func runStartNodeSubCommand(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info(fmt.Sprintf("Start node on %s", utils.GetOS()))
	defer log.WithContext(ctx).Info("End successfully")

	configJSON, err := config.String()
	if err != nil {
		return err
	}

	if len(config.WorkingDir) != 0 {
		InitializeFunc(ctx, config)
	}

	err = updatePastelConfigFile(ctx, filepath.Join(config.WorkingDir, "pastel.conf"), config)

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

	// TODO: Implement start node command
	var pastelDPath string

	if _, pastelDPath, _, _, err = checkPastelInstallPath(ctx, config, ""); err != nil {
		return errNotFoundPastelPath
	}

	if err = checkStartNodeParams(ctx, config); err != nil {
		return err
	}

	var pasteldArgs = fmt.Sprintf("--%s  --datadir=%s", config.Network, config.WorkingDir)

	if flagInteractiveMode {
		if flagReIndex {
			log.WithContext(ctx).Info(fmt.Sprintf("Starting pasteld...\n%s --externalip=%s --txindex=1 --reindex %s", pastelDPath, flagNodeExtIP, pasteldArgs))
			if err = RunPasteldWithInteractive(ctx, config, fmt.Sprintf("--externalip=%s", flagNodeExtIP), "--reindex", "--txindex==1"); err != nil {
				return err
			}
		} else {
			log.WithContext(ctx).Info(fmt.Sprintf("Starting pasteld...\n%s --externalip=%s %s", pastelDPath, flagNodeExtIP, pasteldArgs))
			if err = RunPasteldWithInteractive(ctx, config, fmt.Sprintf("--externalip=%s", flagNodeExtIP)); err != nil {
				return err
			}
		}

	} else {
		if flagReIndex {
			log.WithContext(ctx).Info(fmt.Sprintf("Starting pasteld...\n%s --externalip=%s --txindex=1 --reindex --daemon %s", pastelDPath, flagNodeExtIP, pasteldArgs))
			go RunPasteld(ctx, config, fmt.Sprintf("--externalip=%s", flagNodeExtIP), "--reindex", "--txindex=1", "--daemon")
		} else {
			log.WithContext(ctx).Info(fmt.Sprintf("Starting pasteld...\n%s --externalip=%s --daemon %s", pastelDPath, flagNodeExtIP, pasteldArgs))
			go RunPasteld(ctx, config, fmt.Sprintf("--externalip=%s", flagNodeExtIP), "--daemon")
		}

		var failCnt = 0
		for {
			if _, err = runPastelCLI(ctx, config, "getaccountaddress", ""); err != nil {
				log.WithContext(ctx).Info("Waiting the pasteld to be started ...")
				time.Sleep(10000 * time.Millisecond)
				failCnt++
				if failCnt == 10 {
					log.WithContext(ctx).Errorf("pasteld was not started!")
					return err
				}
			} else {

				log.WithContext(ctx).Info("Started pasteld successfully!")
				break
			}
		}
	}
	return nil
}

func runStartSuperNodeSubCommand(ctx context.Context, config *configs.Config) error {

	if len(config.WorkingDir) != 0 {
		InitializeFunc(ctx, config)
	}

	var err = updatePastelConfigFile(ctx, filepath.Join(config.WorkingDir, "pastel.conf"), config)

	if err != nil {
		return err
	}

	if flagMasterNodeColdHot {
		return runMasterNodOnColdHot(ctx, config)
	}
	return runMasterNodOnHotHot(ctx, config)
}

func runStartWalletSubCommand(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info(fmt.Sprintf("Start wallet node on %s", utils.GetOS()))
	defer log.WithContext(ctx).Info("End successfully")

	configJSON, err := config.String()
	if err != nil {
		return err
	}

	log.WithContext(ctx).Infof("Config: %s", configJSON)

	if len(config.WorkingDir) != 0 {
		InitializeFunc(ctx, config)
	}

	err = updatePastelConfigFile(ctx, filepath.Join(config.WorkingDir, "pastel.conf"), config)

	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sys.RegisterInterruptHandler(cancel, func() {
		log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
		os.Exit(0)
	})

	// TODO: Implement wallet command
	var pastelDPath, _ string

	// *************  1. Start pastel node  *************
	if _, pastelDPath, _, _, err = checkPastelInstallPath(ctx, config, ""); err != nil {
		return errNotFoundPastelPath
	}

	if err = checkStartNodeParams(ctx, config); err != nil {
		return err
	}

	var pasteldArgs = fmt.Sprintf("--%s  --datadir=%s", config.Network, config.WorkingDir)

	if flagReIndex {
		log.WithContext(ctx).Info(fmt.Sprintf("Starting pasteld...\n%s --externalip=%s --txindex=1 --reindex --daemon %s", pastelDPath, flagNodeExtIP, pasteldArgs))
		go RunPasteld(ctx, config, fmt.Sprintf("--externalip=%s", flagNodeExtIP), "--reindex", "--txindex=1", "--daemon")
	} else {
		log.WithContext(ctx).Info(fmt.Sprintf("Starting pasteld...\n%s --externalip=%s --daemon %s", pastelDPath, flagNodeExtIP, pasteldArgs))
		go RunPasteld(ctx, config, fmt.Sprintf("--externalip=%s", flagNodeExtIP), "--daemon")
	}

	var failCnt = 0
	for {
		if _, err = runPastelCLI(ctx, config, "getaccountaddress", ""); err != nil {
			log.WithContext(ctx).Info("Waiting the pasteld to be started ...")
			time.Sleep(10000 * time.Millisecond)
			failCnt++
			if failCnt == 10 {
				log.WithContext(ctx).Error("pasteld was not started!")
				return err
			}
		} else {
			log.WithContext(ctx).Info("pasteld was started successfully!")
			break
		}
	}

	// *************  2. Start wallet node  *************
	var workDirPath = filepath.Join(config.WorkingDir, "walletnode", "wallet.yml")

	if flagInteractiveMode {
		if err = runPastelWalletNodeWithInteractive(ctx, config, fmt.Sprintf("--config-file=%s", workDirPath)); err != nil {
			log.WithContext(ctx).Error("wallet node start was failed!")
			return err
		}
	} else {
		go runPastelWalletNode(ctx, config, fmt.Sprintf("--config-file=%s", workDirPath))

		time.Sleep(10000 * time.Millisecond)

		log.WithContext(ctx).Info("Wallet node was started successfully!")
	}
	return nil
}

func runMasterNodOnHotHot(ctx context.Context, config *configs.Config) error {
	var masternodePrivKey, pastelid, output string
	var err error

	if len(config.WorkingDir) != 0 {
		InitializeFunc(ctx, config)
	}

	if config.Network == "testnet" {
		flagMasterNodeIsTestNet = true
	}

	// *************  1. Parse parameters  *************
	log.WithContext(ctx).Info("Checking parameters")
	if err := checkStartMasterNodeParams(ctx, config); err != nil {
		return err
	}
	log.WithContext(ctx).Info("Finished checking parameters!")

	log.WithContext(ctx).Info("Checking pastel config...")
	if err := CheckPastelConf(config); err != nil {
		return err
	}
	log.WithContext(ctx).Info("Finished checking pastel config!")

	// If create master node using HOT/HOT wallet
	if flagMasterNodeIsCreate || flagMasterNodeIsUpdate {

		if flagMasterNodeIsCreate {
			if err = backupConfFile(config); err != nil { // delete conf file
				return err
			}
			log.WithContext(ctx).Info("Backup masternode.conf was finished successfully.")
			// *************  2.1 Start the Pastel Network Node  *************
			log.WithContext(ctx).Info(fmt.Sprintf("Starting pasteld...\npasteld --externalip=%s --reindex --daemon\n", flagMasterNodeIP))
			go RunPasteld(ctx, config, fmt.Sprintf("--externalip=%s", flagMasterNodeIP), "--reindex", "--daemon")

			var failCnt = 0
			for {
				if output, err = runPastelCLI(ctx, config, "getaccountaddress", ""); err != nil {
					log.WithContext(ctx).Info("Waiting the pasteld to be started ...")
					time.Sleep(10000 * time.Millisecond)
					failCnt++
					if failCnt == 10 {
						log.WithContext(ctx).Error("pasteld was not started!")
						return err
					}
				} else {
					log.WithContext(ctx).Info(fmt.Sprintf("Started pasteld successfully!\nHot wallet address = %s", output))
					break
				}
			}

			// *************  2.2 Search collateral transaction *************
			if output, err = runPastelCLI(ctx, config, "masternode", "outputs"); err != nil {
				log.WithContext(ctx).Info("Cannot find masternode output.")
				return err
			}
			var recMasterNode map[string]interface{}
			json.Unmarshal([]byte(output), &recMasterNode)

			if len(recMasterNode) != 0 {

				if recMasterNode[flagMasterNodeTxID] != nil && recMasterNode[flagMasterNodeTxID] == flagMasterNodeIND {
					// if receives PSL go to next step
					log.WithContext(ctx).Info(fmt.Sprintf("masternode outputs = %s, %s", flagMasterNodeTxID, flagMasterNodeIND))
				}
			}

			// *************  2.3 create new MN private key  *************
			if len(flagMasterNodePrivateKey) == 0 {
				if masternodePrivKey, err = runPastelCLI(ctx, config, "masternode", "genkey"); err != nil {
					return err
				}
			} else {
				masternodePrivKey = flagMasterNodePrivateKey
			}

			log.WithContext(ctx).Info(fmt.Sprintf("masternode private key = %s", masternodePrivKey))

			if len(flagMasterNodePastelID) == 0 && len(flagMasterNodePassPhrase) != 0 {
				//  *************  2.4 create new pastelid  *************
				if output, err = runPastelCLI(ctx, config, "pastelid", "newkey", flagMasterNodePassPhrase); err != nil {
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

			log.WithContext(ctx).Info(fmt.Sprintf("pastelid = %s", pastelid))

			failCnt = 0

			for {
				if output, err = runPastelCLI(ctx, config, "getaccountaddress", ""); err != nil {
					log.WithContext(ctx).Info("Waiting the pasteld to be started ...")
					time.Sleep(10000 * time.Millisecond)
					failCnt++
					if failCnt == 10 {
						log.WithContext(ctx).Info("Can not start with pasteld")
						return err
					}
				} else {
					log.WithContext(ctx).Info(fmt.Sprintf("master node address = %s", output))
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
			if err = createConfFile(data, config); err != nil {
				return err
			}
			log.WithContext(ctx).Info(fmt.Sprintf("masternode.conf = %s", string(data)))

			log.WithContext(ctx).Info("Stopping pasteld...")
			if _, err = runPastelCLI(ctx, config, "stop"); err != nil {
				return err
			}

			time.Sleep(10000 * time.Millisecond)
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
			if _, err = updateMasternodeConfFile(confData, config); err != nil {
				return err
			}

			data, _ := json.Marshal(confData)
			log.WithContext(ctx).Info(fmt.Sprintf("masternode.conf = %s", string(data)))
		}
	}

	// Get conf data from masternode.conf File
	var nodeName, privKey, extIP, pastelID, extPort = getStartInfo(config)

	// *************  3. Start Node as Masternode  *************
	go RunPasteld(ctx, config, "--masternode", "--txindex=1", "--reindex", fmt.Sprintf("--masternodeprivkey=%s", privKey), fmt.Sprintf("--externalip=%s", extIP))

	// *************  4. Wait for blockchain and masternodes sync  *************
	var mnstatus structure.RPCPastelMSStatus
	var failCnt = 0

	for {
		if output, err = runPastelCLI(ctx, config, "mnsync", "status"); err != nil {
			log.WithContext(ctx).Info("Waiting the pasteld to be started ...")
			time.Sleep(10000 * time.Millisecond)
			failCnt++
			if failCnt >= 10 {
				log.WithContext(ctx).Error("pasteld was not started!")
				return err
			}
		} else {
			// Master Node Output
			if err = json.Unmarshal([]byte(output), &mnstatus); err != nil {
				return err
			}

			if mnstatus.AssetName == "Initial" {
				if _, err = runPastelCLI(ctx, config, "mnsync", "reset"); err != nil {
					log.WithContext(ctx).Error("master node reset was failed")
					return err
				}
				time.Sleep(10000 * time.Millisecond)
			}
			if mnstatus.IsSynced {
				log.WithContext(ctx).Info("master node was synced!")
				break
			}
			log.WithContext(ctx).Info("Waiting for sync...")
			time.Sleep(10000 * time.Millisecond)
		}
	}

	// *************  5. Enable Masternode  ***************
	failCnt = 0
	for {
		if output, err = runPastelCLI(ctx, config, "masternode", "start-alias", nodeName); err != nil {
			log.WithContext(ctx).Info("Waiting the pasteld to be started ...")
			time.Sleep(10000 * time.Millisecond)
			failCnt++
			if failCnt == 10 {
				log.WithContext(ctx).Error("pasteld was not started!")
				return err
			}
		} else {
			log.WithContext(ctx).Info("The pasteld was started successfully!")
			log.WithContext(ctx).Info(fmt.Sprintf("masternode alias status = %s\n", output))
			break
		}

	}

	// *************  6. Start supernode  **************
	log.WithContext(ctx).Info("Start supernode")
	log.WithContext(ctx).Debug("Configure supernode setting")

	workDirPath := filepath.Join(config.WorkingDir, "supernode")

	// create file

	fileName, err := utils.CreateFile(ctx, filepath.Join(workDirPath, "supernode.yml"), true)
	if err != nil {
		return err
	}

	// write to file
	file, err := os.OpenFile(fileName, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Populate pastel.conf line-by-line to file.
	_, err = file.WriteString(fmt.Sprintf(configs.SupernodeDefaultConfig, pastelID, extIP, extPort)) // creates server line
	if err != nil {
		return err
	}

	log.WithContext(ctx).Info("Configuring supernode was finished")
	log.WithContext(ctx).Info("Start supernode")

	if flagInteractiveMode {
		RunCMDWithInteractive(filepath.Join(config.PastelExecDir, constants.PastelSuperNodeExecName[utils.GetOS()]),
			fmt.Sprintf("--config-file=%s", filepath.Join(config.WorkingDir, "supernode", "supernode.yml")))
	} else {
		go RunCMD(filepath.Join(config.PastelExecDir, constants.PastelSuperNodeExecName[utils.GetOS()]),
			fmt.Sprintf("--config-file=%s", filepath.Join(config.WorkingDir, "supernode", "supernode.yml")))
		log.WithContext(ctx).Info("Waiting for supernode started...")
		time.Sleep(10000 * time.Millisecond)
	}

	log.WithContext(ctx).Info("Supernode was started successfully")
	return nil
}

func runMasterNodOnColdHot(ctx context.Context, config *configs.Config) error {
	var masternodePrivKey, pastelid, output string
	var err error

	if len(config.WorkingDir) != 0 {
		InitializeFunc(ctx, config)
	}

	if config.Network == "testnet" {
		flagMasterNodeIsTestNet = true
	}

	log.WithContext(ctx).Info("Checking parameters...")
	if err := checkStartMasterNodeParams(ctx, config); err != nil {
		log.WithContext(ctx).Error(fmt.Sprintf("Checking parameters occurs error -> %s", err))
		return err
	}
	log.WithContext(ctx).Info("Finished checking parameters!")
	log.WithContext(ctx).Info("Checking pastel config...")
	if err := CheckPastelConf(config); err != nil {
		log.WithContext(ctx).Error("pastel.conf was not correct!")
		return err
	}
	log.WithContext(ctx).Info("Finished checking pastel config!")
	// If create master node using HOT/HOT wallet
	if flagMasterNodeIsCreate || flagMasterNodeIsUpdate {

		if flagMasterNodeIsCreate {
			if err = backupConfFile(config); err != nil { // delete conf file
				return err
			}

			log.WithContext(ctx).Info(fmt.Sprintf("Start pasteld\n./pasteld --externalip=%s --reindex --daemon", flagMasterNodeSSHIP))
			go RunPasteld(ctx, config, fmt.Sprintf("--externalip=%s", flagMasterNodeSSHIP), "--reindex", "--daemon")

			var failCnt = 0
			for {
				if output, err = runPastelCLI(ctx, config, "getaccountaddress", ""); err != nil {
					log.WithContext(ctx).Info("Waiting the pasteld to be started ...")
					time.Sleep(10000 * time.Millisecond)
					failCnt++
					if failCnt == 10 {
						log.WithContext(ctx).Info("Can not start with pasteld")
						return err
					}
				} else {
					log.WithContext(ctx).Info(fmt.Sprintf("Hot wallet address = %s", output))
					break
				}
			}

			// ***************  3.1 Search collateral transaction  ***************
			if output, err = runPastelCLI(ctx, config, "masternode", "outputs"); err != nil {
				log.WithContext(ctx).Error("This above command doesn't run!")
				return err
			}
			var recMasterNode map[string]interface{}
			json.Unmarshal([]byte(output), &recMasterNode)

			if len(recMasterNode) != 0 {

				if recMasterNode[flagMasterNodeTxID] != nil && recMasterNode[flagMasterNodeTxID] == flagMasterNodeIND {
					// if receives PSL go to next step
					log.WithContext(ctx).Info(fmt.Sprintf("masternode outputs = %s, %s", flagMasterNodeTxID, flagMasterNodeIND))
				}
			}

			// ***************  3.2 Create new MN private key  ***************
			if len(flagMasterNodePrivateKey) == 0 {
				if masternodePrivKey, err = runPastelCLI(ctx, config, "masternode", "genkey"); err != nil {
					return err
				}
				flagMasterNodePrivateKey = masternodePrivKey
			} else {
				masternodePrivKey = flagMasterNodePrivateKey
			}
			log.WithContext(ctx).Info(fmt.Sprintf("masternode private key = %s", masternodePrivKey))

			// ***************  3.3 create new pastelid  ***************
			if len(flagMasterNodePastelID) == 0 && len(flagMasterNodePassPhrase) != 0 {
				// Check masternode status
				var mnstatus structure.RPCPastelMSStatus

				for {
					if output, err = runPastelCLI(ctx, config, "mnsync", "status"); err != nil {
						log.WithContext(ctx).Info("Waiting the pasteld to be started ...")
						time.Sleep(10000 * time.Millisecond)
						failCnt++
						if failCnt >= 10 {
							log.WithContext(ctx).Error("pasteld was not started!")
							return err
						}
					} else {
						// Master Node Output
						if err = json.Unmarshal([]byte(output), &mnstatus); err != nil {
							return err
						}
						if mnstatus.AssetName == "Initial" {
							log.WithContext(ctx).Info("master node resets status")
							if _, err = runPastelCLI(ctx, config, "mnsync", "reset"); err != nil {
								log.WithContext(ctx).Info("master node reset was failed")
								return err
							}
							time.Sleep(10000 * time.Millisecond)
						} else {
							if mnstatus.IsSynced {
								log.WithContext(ctx).Info("master node was synced!")
								break
							}
							log.WithContext(ctx).Info("Waiting for sync...")
							time.Sleep(10000 * time.Millisecond)
						}
					}
				}

				if output, err = runPastelCLI(ctx, config, "pastelid", "newkey", flagMasterNodePassPhrase); err != nil {
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

			log.WithContext(ctx).Info(fmt.Sprintf("pastelid = %s", pastelid))

			log.WithContext(ctx).Info("Stopping pasteld...")
			if _, err = runPastelCLI(ctx, config, "stop"); err != nil {
				return err
			}
			time.Sleep(5000 * time.Millisecond)
			log.WithContext(ctx).Info("Stopped pasteld")
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
			if err = createConfFile(data, config); err != nil {
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
			if _, err = updateMasternodeConfFile(confData, config); err != nil {
				return err
			}
		}
	}

	// ***************  4. Execute following commands over SSH on the remote node (using ssh-ip and ssh-port)  ***************
	username, password, _ := credentials()

	if err = remoteHotNodeCtrl(ctx, config, username, password); err != nil {
		log.WithContext(ctx).Error(fmt.Sprintf("%s\n", err))
		return err
	}
	log.WithContext(ctx).Info("The hot wallet node has been successfully launched!")
	// ***************  5. Enable Masternode  ***************
	// Get conf data from masternode.conf File
	var _, _, extIP, _, _ = getStartInfo(config)

	// Start Node as Masternode
	go RunPasteld(ctx, config, "-txindex=1", "-reindex", fmt.Sprintf("--externalip=%s", extIP))

	var mnstatus structure.RPCPastelMSStatus
	var failCnt = 0

	for {
		if output, err = runPastelCLI(ctx, config, "mnsync", "status"); err != nil {
			log.WithContext(ctx).Info("Waiting the pasteld to be started ...")
			time.Sleep(10000 * time.Millisecond)
			failCnt++
			if failCnt >= 10 {
				log.WithContext(ctx).Error("pasteld was not started!")
				return err
			}
		} else {
			// Master Node Output
			if err = json.Unmarshal([]byte(output), &mnstatus); err != nil {
				return err
			}

			if mnstatus.AssetName == "Initial" {
				if _, err = runPastelCLI(ctx, config, "mnsync", "reset"); err != nil {
					log.WithContext(ctx).Error("master node reset was failed")
					return err
				}
				time.Sleep(10000 * time.Millisecond)
			}
			if mnstatus.IsSynced {
				log.WithContext(ctx).Info("master node was synced!")
				break
			}
			log.WithContext(ctx).Info("Waiting for sync...")
			time.Sleep(10000 * time.Millisecond)
		}
	}

	if output, err = runPastelCLI(ctx, config, "masternode", "start-alias", flagMasterNodeName); err != nil {
		return err
	}

	// RPCPastelMSStatus RPC result structure from masternode status

	var aliasStatus map[string]interface{}

	if err = json.Unmarshal([]byte(output), &aliasStatus); err != nil {
		return err
	}

	if aliasStatus["result"] == "failed" {
		log.WithContext(ctx).Error(aliasStatus["errorMessage"])
		// return err
	} else {
		log.WithContext(ctx).Info(fmt.Sprintf("masternode alias status = %s\n", output))
	}

	// ***************  6. Stop Cold Node  ***************
	if _, err = runPastelCLI(ctx, config, "stop"); err != nil {
		return err
	}

	// *************  7. Start supernode  **************
	log.WithContext(ctx).Info("Start supernode")
	log.WithContext(ctx).Debug("Configure supernode setting")

	

	var supernodeYmlString = fmt.Sprintf(configs.SupernodeDefaultConfig, pastelid, extIP, fmt.Sprintf("%d", flagMasterNodeRPCPort))

	
	log.WithContext(ctx).Info("Configuring supernode was finished")
	log.WithContext(ctx).Info("Start supernode")

	client, err := utils.DialWithPasswd(fmt.Sprintf("%s:%d", flagMasterNodeSSHIP, flagMasterNodeSSHPort), username, password)
	if err != nil {
		return err
	}

	var remoteSuperNodePath = string(flagMasterNodePastelPath[1 : len(flagMasterNodePastelPath)-8])

	remoteSuperNodePath = filepath.Join(remoteSuperNodePath, "supernode", "supernode.yml")


	log.WithContext(ctx).Info(fmt.Sprintf("RemoteSuperNodePath:%s", remoteSuperNodePath))

	log.WithContext(ctx).Info(fmt.Sprintf("RemoteSuperNodeYmlString:%s", supernodeYmlString))

	log.WithContext(ctx).Info(fmt.Sprintf("RemoteSuperNodeCmd1:%s", fmt.Sprintf("%s %s", filepath.Join(config.PastelExecDir, constants.PastelSuperNodeExecName[utils.GetOS()]), fmt.Sprintf("--config-file=%s", remoteSuperNodePath))))

	_, err = client.Cmd(fmt.Sprintf("echo %s > %s", supernodeYmlString, remoteSuperNodePath)).Output()
	if err != nil {
		return err
	}

	log.WithContext(ctx).Info(fmt.Sprintf("echo %s > %s", supernodeYmlString, remoteSuperNodePath))

	go client.Cmd(fmt.Sprintf("%s %s", filepath.Join(config.PastelExecDir, constants.PastelSuperNodeExecName[utils.GetOS()]), fmt.Sprintf("--config-file=%s", remoteSuperNodePath))).Run()

	defer client.Close()

	log.WithContext(ctx).Info("Remote:::Waiting for supernode started...")
	time.Sleep(10000 * time.Millisecond)

	log.WithContext(ctx).Info("Remote:::Supernode was started successfully")
	return nil
}

func remoteHotNodeCtrl(ctx context.Context, _ *configs.Config, username string, password string) error {
	var output []byte
	var pastelCliPath string
	log.WithContext(ctx).Info(fmt.Sprintf("Connecting to SSH Hot node wallet -> %s:%d...", flagMasterNodeSSHIP, flagMasterNodeSSHPort))
	client, err := utils.DialWithPasswd(fmt.Sprintf("%s:%d", flagMasterNodeSSHIP, flagMasterNodeSSHPort), username, password)
	if err != nil {
		return err
	}
	defer client.Close()

	// remoteClient = client

	// Find pasteld
	log.WithContext(ctx).Info("Check pasteld default path...")
	out, err := client.Cmd(fmt.Sprintf("test -e %s && echo file exists || echo file not found", flagMasterNodePastelPath)).Output()
	if err != nil {
		return err
	}

	if strings.TrimRight(string(out), "\n") != "file exists" {
		log.WithContext(ctx).Info("Finding pasteld executable on $HOME path")
		out, err = client.Cmd("find $HOME -iname pasteld").Output()
		if err != nil {
			return err
		}

		pastelPaths := strings.Split(string(out), "\n")

		if len(pastelPaths) > 0 {
			log.WithContext(ctx).Info(fmt.Sprintf("%s\n", string(out)))
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
					log.WithContext(ctx).Warn("Please input index correctly")
				} else {
					flagMasterNodePastelPath = pastelPaths[index]
					break
				}
			}
		}

	} else {
		log.WithContext(ctx).Info("Found pasteld on default path!")
	}

	log.WithContext(ctx).Info("Checking pastel-cli path...")
	out, err = client.Cmd("find $HOME -iname pastel-cli").Output()
	if err != nil {
		return err
	}

	pastelCliPaths := strings.Split(string(out), "\n")

	if len(pastelCliPaths) == 0 {
		return errNotFoundPastelCli
	}

	pastelCliPath = pastelCliPaths[0]

	log.WithContext(ctx).Info(fmt.Sprintf("Found pastel-cli path on %s", pastelCliPath))

	if flagMasterNodeIsTestNet {
		go client.Cmd(fmt.Sprintf("%s --reindex --testnet --externalip=%s --daemon", flagMasterNodePastelPath, flagMasterNodeIP)).Run()
	} else {
		go client.Cmd(fmt.Sprintf("%s --reindex --externalip=%s --daemon", flagMasterNodePastelPath, flagMasterNodeIP)).Run()
	}

	time.Sleep(10000 * time.Millisecond)

	var mnstatus structure.RPCPastelMSStatus
	failCnt := 0

	for {
		if output, err = client.Cmd(fmt.Sprintf("%s mnsync status", pastelCliPath)).Output(); err != nil {
			log.WithContext(ctx).Info("Remote:::Waiting the pasteld to be started ...")
			time.Sleep(10000 * time.Millisecond)
			failCnt++
			if failCnt >= 10 {
				log.WithContext(ctx).Error("Remote:::Pasteld was not started!")
				return err
			}
		} else {
			// Master Node Output
			if err = json.Unmarshal([]byte(output), &mnstatus); err != nil {
				return err
			}
			if mnstatus.AssetName == "Initial" {
				if _, err = client.Cmd(fmt.Sprintf("%s mnsync reset", pastelCliPath)).Output(); err != nil {
					log.WithContext(ctx).Error("Remote:::master node reset was failed")
					return err
				}
				time.Sleep(10000 * time.Millisecond)
			} else {
				if mnstatus.IsSynced {
					log.WithContext(ctx).Info("Remote:::master node was synced!")
					break
				}
				log.WithContext(ctx).Info("Remote:::Waiting for sync...")
				time.Sleep(10000 * time.Millisecond)
			}
		}
	}

	if _, err = client.Cmd(fmt.Sprintf("%s stop", pastelCliPath)).Output(); err != nil {
		log.WithContext(ctx).Error("Error - stopping on pasteld")
		return err
	}

	time.Sleep(5000 * time.Millisecond)

	if flagMasterNodeIsTestNet {
		cmdLine := fmt.Sprintf("%s --masternode --testnet --txindex=1 --reindex --masternodeprivkey=%s --externalip=%s --daemon", flagMasterNodePastelPath, flagMasterNodePrivateKey, flagMasterNodeIP)
		log.WithContext(ctx).Info(fmt.Sprintf("%s\n", cmdLine))
		go client.Cmd(cmdLine).Run()
	} else {
		cmdLine := fmt.Sprintf("%s --masternode --txindex=1 --reindex --masternodeprivkey=%s --externalip=%s --daemon", flagMasterNodePastelPath, flagMasterNodePrivateKey, flagMasterNodeIP)
		log.WithContext(ctx).Info(fmt.Sprintf("%s\n", cmdLine))
		go client.Cmd(cmdLine).Run()
	}

	time.Sleep(10000 * time.Millisecond)

	failCnt = 0

	for {
		if output, err = client.Cmd(fmt.Sprintf("%s mnsync status", pastelCliPath)).Output(); err != nil {
			log.WithContext(ctx).Info("Remote:::Waiting the pasteld to be started ...")
			time.Sleep(10000 * time.Millisecond)
			failCnt++
			if failCnt >= 10 {
				log.WithContext(ctx).Error("Remote:::pasteld was not started!")
				return err
			}
		} else {
			// Master Node Output
			if err = json.Unmarshal([]byte(output), &mnstatus); err != nil {
				return err
			}
			if mnstatus.AssetName == "Initial" {
				if _, err = client.Cmd(fmt.Sprintf("%s mnsync reset", pastelCliPath)).Output(); err != nil {
					log.WithContext(ctx).Error("master node reset was failed")
					return err
				}
				time.Sleep(10000 * time.Millisecond)
			} else {
				if mnstatus.IsSynced {
					log.WithContext(ctx).Info("master node was synced!")
					break
				}
				log.WithContext(ctx).Info("Remote:::Waiting for sync...")
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
	fmt.Print("\n")
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

		flagMasterNodePastelPath = func() string {
			if len(flagMasterNodePastelPath) == 0 {
				return "$HOME/pastel/pasteld"
			}
			return flagMasterNodePastelPath
		}()

		flagMasterNodeSupernodePath = func() string {
			if len(flagMasterNodeSupernodePath) == 0 {
				return "$HOME/pastel/supernode-linux-amd64"
			}
			return flagMasterNodeSupernodePath
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

func checkStartNodeParams(_ context.Context, _ *configs.Config) error {
	var err error

	if len(flagNodeExtIP) == 0 {
		if flagNodeExtIP, err = GetExternalIPAddress(); err != nil {
			return err
		}
	}

	return nil
}

// GetExternalIPAddress runs shell command and returns external IP address
func GetExternalIPAddress() (externalIP string, err error) {
	return RunCMD("curl", "ipinfo.io/ip")
}

// RunPasteld runs pasteld
func RunPasteld(ctx context.Context, config *configs.Config, args ...string) (output string, err error) {
	var pastelDPath string

	if _, pastelDPath, _, _, err = checkPastelInstallPath(ctx, config, ""); err != nil {
		return pastelDPath, errNotFoundPastelPath
	}

	if !(config.Network == "mainnet" || config.Network == "testnet") {
		return pastelDPath, errNetworkModeInvalid
	}

	args = append(args, fmt.Sprintf("--datadir=%s", config.WorkingDir))

	if config.Network == "testnet" {
		flagMasterNodeIsTestNet = true
	}

	if flagMasterNodeIsTestNet {
		args = append(args, "--testnet")
		output, err = RunCMD(pastelDPath, args...)
	} else {
		output, err = RunCMD(pastelDPath, args...)
	}

	return output, err
}

// RunPasteldWithInteractive runs pasteld with interactive
func RunPasteldWithInteractive(ctx context.Context, config *configs.Config, args ...string) (err error) {
	var pastelDPath string

	if _, pastelDPath, _, _, err = checkPastelInstallPath(ctx, config, ""); err != nil {
		return errNotFoundPastelPath
	}

	if !(config.Network == "mainnet" || config.Network == "testnet") {
		return errNetworkModeInvalid
	}

	args = append(args, fmt.Sprintf("--datadir=%s", config.WorkingDir))

	if config.Network == "testnet" {
		flagMasterNodeIsTestNet = true
	}

	if flagMasterNodeIsTestNet {
		args = append(args, "--testnet")
		return RunCMDWithInteractive(pastelDPath, args...)
	}

	return RunCMDWithInteractive(pastelDPath, args...)
}

// Run pastel-cli
func runPastelCLI(ctx context.Context, config *configs.Config, args ...string) (output string, err error) {
	var pastelCliPath string

	if _, _, pastelCliPath, _, err = checkPastelInstallPath(ctx, config, ""); err != nil {
		return "", errNotFoundPastelPath
	}

	args = append([]string{fmt.Sprintf("--datadir=%s", config.WorkingDir)}, args...)

	return RunCMD(pastelCliPath, args...)
}

func runPastelWalletNode(ctx context.Context, config *configs.Config, args ...string) (output string, err error) {
	var pastelWalletNodePath string

	if _, _, _, pastelWalletNodePath, err = checkPastelInstallPath(ctx, config, "wallet"); err != nil {
		return pastelWalletNodePath, errNotFoundPastelPath
	}

	return RunCMD(pastelWalletNodePath, args...)
}

func runPastelWalletNodeWithInteractive(ctx context.Context, config *configs.Config, args ...string) (err error) {
	var pastelWalletNodePath string

	if _, _, _, pastelWalletNodePath, err = checkPastelInstallPath(ctx, config, "wallet"); err != nil {
		return errNotFoundPastelPath
	}

	return RunCMDWithInteractive(pastelWalletNodePath, args...)
}

// Create or Update masternode.conf File
func createConfFile(confData []byte, config *configs.Config) (err error) {
	workDirPath := config.WorkingDir
	var masternodeConfPath, masternodeConfPathBackup string

	if flagMasterNodeIsTestNet {
		masternodeConfPath = filepath.Join(workDirPath, "testnet3", "masternode.conf")
		masternodeConfPathBackup = filepath.Join(workDirPath, "testnet3", "masternode_%s.conf")
	} else {
		masternodeConfPath = filepath.Join(workDirPath, "masternode.conf")
		masternodeConfPathBackup = filepath.Join(workDirPath, "masternode_%s.conf")
	}
	if _, err := os.Stat(masternodeConfPath); err == nil { // if masternode.conf File exists , backup
		oldFileName := masternodeConfPath
		currentTime := time.Now()
		backupFileName := fmt.Sprintf(masternodeConfPathBackup, currentTime.Format("2021-01-01-23-59-59"))

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

func updateMasternodeConfFile(confData map[string]interface{}, config *configs.Config) (result bool, err error) {
	workDirPath := config.WorkingDir
	var masternodeConfPath string

	if flagMasterNodeIsTestNet {
		masternodeConfPath = filepath.Join(workDirPath, "testnet3", "masternode.conf")
	} else {
		masternodeConfPath = filepath.Join(workDirPath, "masternode.conf")
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

func backupConfFile(config *configs.Config) (err error) {
	workDirPath := config.WorkingDir
	var masternodeConfPath, masternodeConfPathBackup string

	if flagMasterNodeIsTestNet {
		masternodeConfPath = filepath.Join(workDirPath, "testnet3", "masternode.conf")
		masternodeConfPathBackup = filepath.Join(workDirPath, "testnet3", "masternode_%s.conf")
	} else {
		masternodeConfPath = filepath.Join(workDirPath, "masternode.conf")
		masternodeConfPathBackup = filepath.Join(workDirPath, "masternode_%s.conf")
	}
	if _, err := os.Stat(masternodeConfPath); err == nil { // if masternode.conf File exists , backup
		oldFileName := masternodeConfPath
		currentTime := time.Now()
		backupFileName := fmt.Sprintf(masternodeConfPathBackup, currentTime.Format("2021-01-01-23-59-59"))
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

func getStartInfo(config *configs.Config) (nodeName string, privKey string, extIP string, pastelID string, extPort string) {
	var masternodeConfPath string

	if flagMasterNodeIsTestNet {
		masternodeConfPath = filepath.Join(config.WorkingDir, "testnet3", "masternode.conf")
	} else {
		masternodeConfPath = filepath.Join(config.WorkingDir, "masternode.conf")
	}

	// Read ConfData from masternode.conf
	confFile, err := ioutil.ReadFile(masternodeConfPath)
	if err != nil {
		return "", "", "", "", ""
	}

	var conf map[string]interface{}
	json.Unmarshal([]byte(confFile), &conf)

	for key := range conf {
		nodeName = key // get Node Name
	}
	confData := conf[nodeName].(map[string]interface{})
	extAddr := strings.Split(confData["mnAddress"].(string), ":") // get Ext IP
	extKey := confData["extKey"].(string)

	return nodeName, confData["mnPrivKey"].(string), extAddr[0], extKey, extAddr[1]
}

// CheckPastelConf check configuration of pastel settings.
func CheckPastelConf(config *configs.Config) (err error) {
	workDirPath := config.WorkingDir

	if _, err := os.Stat(workDirPath); os.IsNotExist(err) {
		return err
	}

	if _, err := os.Stat(filepath.Join(workDirPath, "pastel.conf")); os.IsNotExist(err) {
		return err
	}

	if config.Network == "testnet" {
		flagMasterNodeIsTestNet = true
	}

	if flagMasterNodeIsTestNet {
		var file, err = os.OpenFile(filepath.Join(workDirPath, "pastel.conf"), os.O_RDWR, 0644)
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
		var file, err = os.OpenFile(filepath.Join(workDirPath, "pastel.conf"), os.O_RDWR, 0644)
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

func checkPastelInstallPath(ctx context.Context, config *configs.Config, flagMode string) (pastelDirPath string, pasteldPath string, pastelCliPath string, pastelWalletNodePath string, err error) {
	if _, err = os.Stat(filepath.Join(config.WorkingDir, constants.PastelConfName)); os.IsNotExist(err) {
		log.WithContext(ctx).Error("could not find pastel.conf")
		return "", "", "", "", fmt.Errorf("could not find pastel.conf")
	}

	if _, err = os.Stat(config.PastelExecDir); os.IsNotExist(err) {
		log.WithContext(ctx).Error("could not find pastel node path")
		return "", "", "", "", fmt.Errorf("could not find pastel node path")
	}
	pastelDirPath = config.PastelExecDir

	if _, err = os.Stat(filepath.Join(config.PastelExecDir, constants.PasteldName[utils.GetOS()])); os.IsNotExist(err) {
		log.WithContext(ctx).Error("could not find pasteld path")
		return "", "", "", "", fmt.Errorf("could not find pasteld path")
	}
	pasteldPath = filepath.Join(config.PastelExecDir, constants.PasteldName[utils.GetOS()])

	if _, err = os.Stat(filepath.Join(config.PastelExecDir, constants.PastelCliName[utils.GetOS()])); os.IsNotExist(err) {
		log.WithContext(ctx).Error("could not find pastel-cli path")
		return "", "", "", "", fmt.Errorf("could not find pastel-cli path")
	}
	pastelCliPath = filepath.Join(config.PastelExecDir, constants.PastelCliName[utils.GetOS()])
	if flagMode == "wallet" {
		if _, err = os.Stat(filepath.Join(config.PastelExecDir, constants.PastelWalletExecName[utils.GetOS()])); os.IsNotExist(err) {
			log.WithContext(ctx).Error("could not find wallet node path")
			return "", "", "", "", fmt.Errorf("could not find wallet node path")
		}
		pastelWalletNodePath = filepath.Join(config.PastelExecDir, constants.PastelWalletExecName[utils.GetOS()])
	}

	return pastelDirPath, pasteldPath, pastelCliPath, pastelWalletNodePath, err
}
