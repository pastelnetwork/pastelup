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
	"github.com/pastelnetwork/pastel-utility/configurer"
	"github.com/pastelnetwork/pastel-utility/constants"
	"github.com/pastelnetwork/pastel-utility/structure"
	"github.com/pastelnetwork/pastel-utility/utils"
	"github.com/pkg/errors"

	"golang.org/x/term"
)

var (
	// errSubCommandRequired             = fmt.Errorf("subcommand is required")
	errMasterNodeNameRequired         = fmt.Errorf("required --name, name of the Masternode to start and create in the masternode.conf if `--create` or `--update` are specified")
	errMasterNodeTxIDRequired         = fmt.Errorf("required --txid, transaction id of 5M collateral MN payment")
	errMasterNodeINDRequired          = fmt.Errorf("required --ind, output index in the transaction of 5M collateral MN payment")
	errMasterNodePwdRequired          = fmt.Errorf("required --passphrase <passphrase to pastelid private key>, if --pastelid is omitted")
	errMasterNodeSSHIPRequired        = fmt.Errorf("required if --coldhot is specified, SSH address of the remote HOT node")
	errSetTestnet                     = fmt.Errorf("please initialize pastel.conf as testnet mode")
	errSetMainnet                     = fmt.Errorf("please initialize pastel.conf as mainnet mode")
	errGetExternalIP                  = fmt.Errorf("cannot get external ip address")
	errNotFoundPastelCli              = fmt.Errorf("cannot find pastel-cli on SSH server")
	errNotFoundPastelPath             = fmt.Errorf("cannot find pasteld/pastel-cli path. please install node")
	errNetworkModeInvalid             = fmt.Errorf("invalid network mode")
	errNotFoundRemotePastelUtilityDir = fmt.Errorf("cannot find remote pastel-utility dir")
	errNotFoundPastelParamPath        = fmt.Errorf("pastel param files are not correct")
	errNotStartPasteld                = fmt.Errorf("pasteld was not started")
	errMasternodeStartAlias           = fmt.Errorf("masternode start alias failed")
)

var (
	flagInteractiveMode bool

	// node flags
	flagNodeExtIP string
	flagReIndex   bool

	// walletnode flag
	flagCheckForce bool
	flagSwagger    bool

	// masternode flags
	flagMasterNodeName          string
	flagMasterNodeIsCreate      bool
	flagMasterNodeIsUpdate      bool
	flagMasterNodeTxID          string
	flagMasterNodeIND           string
	flagMasterNodePort          int
	flagMasterNodePrivateKey    string
	flagMasterNodePastelID      string
	flagMasterNodePassPhrase    string
	flagMasterNodeRPCIP         string
	flagMasterNodeRPCPort       int
	flagMasterNodeP2PIP         string
	flagMasterNodeP2PPort       int
	flagMasterNodeColdHot       bool
	flagMasterNodeSSHIP         string
	flagMasterNodeSSHPort       int
	flagMasterNodePastelPath    string
	flagMasterNodeSupernodePath string
)

type startCommand uint8

const (
	nodeStart startCommand = iota
	walletStart
	superNodeStart
	remoteStart
	//highLevel
)

func setupStartSubCommand(config *configs.Config,
	startCommand startCommand,
	f func(context.Context, *configs.Config) error,
) *cli.Command {
	commonFlags := []*cli.Flag{
		cli.NewFlag("ip", &flagNodeExtIP).
			SetUsage(green("Required, WAN address of the host")).SetRequired(),
		cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
			SetUsage(green("Optional, location where to create working directory")).SetValue(config.WorkingDir),
		cli.NewFlag("network", &config.Network).SetAliases("n").
			SetUsage(green("Optional, network type, can be - \"mainnet\" or \"testnet\"")).SetValue("mainnet"),
		cli.NewFlag("interactive", &flagInteractiveMode).SetAliases("i").
			SetUsage(green("Optinoal, Start with interactive mode")),
		cli.NewFlag("reindex", &flagReIndex).SetAliases("r").
			SetUsage(green("Optional, Start with reindex")),
	}

	walletNodeFlags := []*cli.Flag{
		cli.NewFlag("check-and-force-install", &flagCheckForce).SetAliases("cf").
			SetUsage(green("walletnode specific: Optional, check pastel params before starting walletnode")),
		cli.NewFlag("swagger", &flagSwagger),
	}

	superNodeFlags := []*cli.Flag{

		cli.NewFlag("name", &flagMasterNodeName).
			SetUsage(green("Required, name of the Masternode to start and create in the masternode.conf if --create or --update are specified")).SetRequired(),
		cli.NewFlag("port", &flagMasterNodePort).
			SetUsage(green("Optional, Port for WAN IP address of the node , default - 9933 (19933 for Testnet)")),
		cli.NewFlag("pkey", &flagMasterNodePrivateKey).
			SetUsage(green("Optinoal, Pmasternode priv key, if omitted, new masternode private key will be created")),
		cli.NewFlag("create", &flagMasterNodeIsCreate).
			SetUsage(green("Optional, if specified, will create Masternode record in the masternode.conf.")),
		cli.NewFlag("update", &flagMasterNodeIsUpdate).
			SetUsage(green("Optional, if specified, will update Masternode record in the masternode.conf.")),
		cli.NewFlag("txid", &flagMasterNodeTxID).
			SetUsage(green("collateral payment txid , transaction id of 5M collateral MN payment")),
		cli.NewFlag("ind", &flagMasterNodeIND).
			SetUsage(green("collateral payment output index , output index in the transaction of 5M collateral MN payment")),
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
		cli.NewFlag("ssh-ip", &flagMasterNodeSSHIP).
			SetUsage(green("remote supernode specific: Required, SSH address of the remote HOT node")),
		cli.NewFlag("ssh-port", &flagMasterNodeSSHPort).
			SetUsage(green("remote supernode specific: Optional, SSH port of the remote HOT node")).SetValue(22),
		cli.NewFlag("remote-work-dir", &config.RemoteWorkingDir).
			SetUsage(green("remote supernode specific: Required, location of the working directory")),
		cli.NewFlag("pastelpath", &flagMasterNodePastelPath).
			SetUsage(green("remote supernode specific: Optional, The path of the pasteld")),
	}

	var commandName, commandMessage string
	var commandFlags []*cli.Flag

	switch startCommand {
	case nodeStart:
		commandFlags = commonFlags
		commandName = "node"
		commandMessage = "Start node"
	case walletStart:
		commandFlags = append(walletNodeFlags, commonFlags[:]...)
		commandName = "walletnode"
		commandMessage = "Start walletnode"
	case superNodeStart:
		commandFlags = append(superNodeFlags, commonFlags[:]...)
		commandName = "supernode"
		commandMessage = "Start supernode"
	default:
		commandFlags = append(append(walletNodeFlags, commonFlags[:]...), superNodeFlags[:]...)
	}

	subCommand := cli.NewCommand(commandName)
	subCommand.SetUsage(cyan(commandMessage))
	subCommand.AddFlags(commandFlags...)
	if f != nil {
		subCommand.SetActionFunc(func(ctx context.Context, args []string) error {
			ctx, err := configureLogging(ctx, commandMessage, config)
			if err != nil {
				return err
			}

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			sys.RegisterInterruptHandler(cancel, func() {
				log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
				os.Exit(0)
			})

			log.WithContext(ctx).Info("Starting")
			err = f(ctx, config)
			if err != nil {
				return err
			}
			log.WithContext(ctx).Info("Finished successfully!")

			return nil
		})
	}
	return subCommand
}

func setupStartCommand() *cli.Command {
	config := configs.GetConfig()

	startNodeSubCommand := setupStartSubCommand(config, nodeStart, runStartNodeSubCommand)
	startWalletSubCommand := setupStartSubCommand(config, walletStart, runStartWalletSubCommand)
	startSuperNodeSubCommand := setupStartSubCommand(config, superNodeStart, runStartSuperNodeSubCommand)

	startCommand := cli.NewCommand("start")
	startCommand.SetUsage(blue("Performs start of the system for both WalletNode and SuperNodes"))
	startCommand.AddSubcommands(startNodeSubCommand)
	startCommand.AddSubcommands(startWalletSubCommand)
	startCommand.AddSubcommands(startSuperNodeSubCommand)

	return startCommand

}

func runStartNodeSubCommand(ctx context.Context, config *configs.Config) error {
	return runComponents(ctx, config, constants.PastelD)
}

func runStartSuperNodeSubCommand(ctx context.Context, config *configs.Config) error {
	return runComponents(ctx, config, constants.SuperNode)
}

func runStartWalletSubCommand(ctx context.Context, config *configs.Config) error {
	return runComponents(ctx, config, constants.WalletNode)
}

func runMasterNodOnHotHot(ctx context.Context, config *configs.Config) error {
	var err error

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
	if _, err = getMasternodeConf(ctx, config); err != nil {
		return err
	}

	// Get conf data from masternode.conf File
	var nodeName, privKey, extIP, pastelID, extPort string
	if nodeName, privKey, extIP, pastelID, extPort, err = getStartInfo(config); err != nil {
		return err
	}

	// *************  3. Start Node as Masternode  *************
	go RunPasteld(ctx, config, "--masternode", "--txindex=1", "--reindex", fmt.Sprintf("--masternodeprivkey=%s", privKey), fmt.Sprintf("--externalip=%s", extIP))

	// *************  4. Wait for blockchain and masternodes sync  *************
	if !checkPastelDRunning(ctx, config) {
		return errNotStartPasteld
	}

	if err = checkMasterNodeSync(ctx, config); err != nil {
		return err
	}

	// *************  5. Enable Masternode  ***************
	if err = runStartAliasMasternode(ctx, config, nodeName); err != nil {
		return err
	}

	// *************  6. Start rq-servce    *************
	if err = runPastelService(ctx, config, constants.RQService, false); err != nil {
		return err
	}

	// *************  7. Start supernode  **************
	log.WithContext(ctx).Info("Starting supernode")
	log.WithContext(ctx).Debug("Configure supernode setting")

	workDirPath := filepath.Join(config.WorkingDir, "supernode")

	fileName, err := utils.CreateFile(ctx, filepath.Join(workDirPath, "supernode.yml"), true)
	if err != nil {
		return err
	}

	// write to file
	if err = utils.WriteFile(fileName, fmt.Sprintf(configs.SupernodeDefaultConfig, pastelID, extIP, extPort)); err != nil {
		return err
	}

	log.WithContext(ctx).Info("Configuring supernode was finished")

	if flagInteractiveMode {
		RunCMDWithInteractive(filepath.Join(config.PastelExecDir, constants.SuperNodeExecName[utils.GetOS()]),
			fmt.Sprintf("--config-file=%s", filepath.Join(config.WorkingDir, "supernode", "supernode.yml")))
	} else {
		go RunCMD(filepath.Join(config.PastelExecDir, constants.SuperNodeExecName[utils.GetOS()]),
			fmt.Sprintf("--config-file=%s", filepath.Join(config.WorkingDir, "supernode", "supernode.yml")))
		log.WithContext(ctx).Info("Waiting for supernode started...")
		time.Sleep(10000 * time.Millisecond)
	}

	return nil
}

func runMasterNodOnColdHot(ctx context.Context, config *configs.Config) error {
	var pastelid string
	var err error

	if len(config.RemotePastelUtilityDir) == 0 {
		return errNotFoundRemotePastelUtilityDir
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
	if pastelid, err = getMasternodeConf(ctx, config); err != nil {
		return err
	}

	// ***************  4. Execute following commands over SSH on the remote node (using ssh-ip and ssh-port)  ***************
	username, password, _ := credentials(true)

	if err = remoteHotNodeCtrl(ctx, config, username, password); err != nil {
		log.WithContext(ctx).Error(fmt.Sprintf("%s\n", err))
		return err
	}
	log.WithContext(ctx).Info("The hot wallet node has been successfully launched!")
	// ***************  5. Enable Masternode  ***************
	// Get conf data from masternode.conf File
	var extIP string
	if _, _, extIP, _, _, err = getStartInfo(config); err != nil {
		return err
	}

	// Start Node as Masternode
	go RunPasteld(ctx, config, "-txindex=1", "-reindex", fmt.Sprintf("--externalip=%s", extIP))

	if !checkPastelDRunning(ctx, config) {
		return errNotStartPasteld
	}

	if err = checkMasterNodeSync(ctx, config); err != nil {
		return err
	}

	if err = runStartAliasMasternode(ctx, config, flagMasterNodeName); err != nil {
		return err
	}

	// ***************  6. Stop Cold Node  ***************
	if _, err = runPastelCLI(ctx, config, "stop"); err != nil {
		return err
	}

	client, err := utils.DialWithPasswd(fmt.Sprintf("%s:%d", flagMasterNodeSSHIP, flagMasterNodeSSHPort), username, password)
	if err != nil {
		return err
	}

	// *************  7. Start rq-servce    *************
	if err = runPastelServiceRemote(ctx, config, constants.RQService, client); err != nil {
		return err
	}

	// ***************  8. Start supernode  **************

	err = runSuperNodeRemote(ctx, config, client, extIP, pastelid)
	if err != nil {
		return err
	}

	return nil
}

func remoteHotNodeCtrl(ctx context.Context, config *configs.Config, username string, password string) error {
	var pastelCliPath string
	log.WithContext(ctx).Infof("Connecting to SSH Hot node wallet -> %s:%d...", flagMasterNodeSSHIP, flagMasterNodeSSHPort)
	client, err := utils.DialWithPasswd(fmt.Sprintf("%s:%d", flagMasterNodeSSHIP, flagMasterNodeSSHPort), username, password)
	if err != nil {
		return err
	}
	defer client.Close()

	testnetOption := ""
	if config.Network == "testnet" {
		testnetOption = " --testnet"
	}

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
			log.WithContext(ctx).Infof("%s\n", string(out))
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

	log.WithContext(ctx).Infof("Found pastel-cli path on %s", pastelCliPath)

	go client.Cmd(fmt.Sprintf("%s --reindex --externalip=%s --daemon%s", flagMasterNodePastelPath, flagNodeExtIP, testnetOption)).Run()

	time.Sleep(10000 * time.Millisecond)

	if err = checkMasterNodeSyncRemote(ctx, config, client, pastelCliPath); err != nil {
		log.WithContext(ctx).Error("Remote::Master node sync failed")
		return err
	}

	if _, err = client.Cmd(fmt.Sprintf("%s stop", pastelCliPath)).Output(); err != nil {
		log.WithContext(ctx).Error("Error - stopping on pasteld")
		return err
	}

	time.Sleep(5000 * time.Millisecond)

	cmdLine := fmt.Sprintf("%s --masternode --txindex=1 --reindex --masternodeprivkey=%s --externalip=%s%s --daemon", flagMasterNodePastelPath, flagMasterNodePrivateKey, flagNodeExtIP, testnetOption)
	log.WithContext(ctx).Infof("%s\n", cmdLine)
	go client.Cmd(cmdLine).Run()

	time.Sleep(10000 * time.Millisecond)

	if err = checkMasterNodeSyncRemote(ctx, config, client, pastelCliPath); err != nil {
		log.WithContext(ctx).Error("Remote::Master node sync failed")
		return err
	}

	return nil
}

func runComponents(ctx context.Context, config *configs.Config, startType constants.ToolType) (err error) {
	if len(config.WorkingDir) != 0 {
		InitializeFunc(ctx, config)
	}

	if err = updatePastelConfigFileForNetwork(ctx, filepath.Join(config.WorkingDir, "pastel.conf"), config); err != nil {
		return err
	}

	switch startType {
	case constants.PastelD:
		if err = runPastelNode(ctx, config, flagReIndex, flagInteractiveMode); err != nil {
			return err
		}
	case constants.WalletNode:
		if err = checkAndForceInit(ctx, config); err != nil {
			return err
		}

		// *************  1. Start pastel node  *************
		if err = runPastelNode(ctx, config, flagReIndex, false); err != nil {
			return err
		}

		// *************  2. Start rq-servce    *************
		if err = runPastelService(ctx, config, constants.RQService, false); err != nil {
			return err
		}

		// *************  3. Start wallet node  *************
		log.WithContext(ctx).Info("Starting walletnode")
		workDirPath := filepath.Join(config.WorkingDir, "walletnode", "wallet.yml")

		if flagInteractiveMode {
			if flagSwagger {
				if err = runPastelWalletNodeWithInteractive(ctx, config, "--swagger", fmt.Sprintf("--config-file=%s", workDirPath)); err != nil {
					log.WithContext(ctx).Error("wallet node start was failed!")
					return err
				}
			} else {
				if err = runPastelWalletNodeWithInteractive(ctx, config, fmt.Sprintf("--config-file=%s", workDirPath)); err != nil {
					log.WithContext(ctx).Error("wallet node start was failed!")
					return err
				}
			}
		} else {
			if flagSwagger {
				go runPastelWalletNode(ctx, config, "--swagger", fmt.Sprintf("--config-file=%s", workDirPath))
			} else {
				go runPastelWalletNode(ctx, config, fmt.Sprintf("--config-file=%s", workDirPath))
			}

			time.Sleep(10000 * time.Millisecond)
		}
	case constants.SuperNode:
		if len(flagMasterNodeSSHIP) != 0 {
			flagMasterNodeColdHot = true
		}

		if flagMasterNodeColdHot {
			return runMasterNodOnColdHot(ctx, config)
		}
		return runMasterNodOnHotHot(ctx, config)
	}

	return nil
}

func runStartAliasMasternode(ctx context.Context, config *configs.Config, masternodeName string) (err error) {
	var output string
	if output, err = runPastelCLI(ctx, config, "masternode", "start-alias", masternodeName); err != nil {
		return err
	}
	var aliasStatus map[string]interface{}

	if err = json.Unmarshal([]byte(output), &aliasStatus); err != nil {
		return err
	}

	if aliasStatus["result"] == "failed" {
		log.WithContext(ctx).Error(aliasStatus["errorMessage"])
		return errMasternodeStartAlias
	}

	log.WithContext(ctx).Infof("masternode alias status = %s\n", output)
	return nil
}

func runPastelService(ctx context.Context, config *configs.Config, tool constants.ToolType, interactive bool) (err error) {
	commandName := strings.Split(string(tool), "/")[len(strings.Split(string(tool), "/"))-1]
	log.WithContext(ctx).Infof("Starting %s", commandName)

	var workDirPath, pastelServicePath string
	switch tool {
	case constants.RQService:
		workDirPath = filepath.Join(config.WorkingDir, "rqservice", "rqservice.toml")
		pastelServicePath = filepath.Join(config.PastelExecDir, constants.PastelRQServiceExecName[utils.GetOS()])
		if interactive {
			if err = RunCMDWithInteractive(pastelServicePath, fmt.Sprintf("--config-file=%s", workDirPath)); err != nil {
				log.WithContext(ctx).Error("rqservice start was failed!")
				return err
			}
		} else {
			go RunCMD(pastelServicePath, fmt.Sprintf("--config-file=%s", workDirPath))

			time.Sleep(10000 * time.Millisecond)
		}
	}

	isServiceRunning := checkServiceRunning(ctx, config, tool)
	if isServiceRunning {
		log.WithContext(ctx).Infof("The %s started succesfully!", commandName)
	} else {
		if output, err := RunCMD(pastelServicePath, fmt.Sprintf("--config-file=%s", workDirPath)); err != nil {
			log.WithContext(ctx).Errorf("%s start was failed! : %s", commandName, output)
			return err
		}
	}

	return nil
}
func runPastelServiceRemote(ctx context.Context, config *configs.Config, tool constants.ToolType, client *utils.Client) (err error) {
	commandName := strings.Split(string(tool), "/")[len(strings.Split(string(tool), "/"))-1]
	log.WithContext(ctx).Infof("Remote:::Starting %s", commandName)

	remoteWorkDirPath, remotePastelExecPath, remoteOsType, err := getRemoteInfo(config, client)
	if err != nil {
		return err
	}

	switch tool {
	case constants.RQService:
		workDirPath := filepath.Join(string(remoteWorkDirPath), "rqservice", "rqservice.toml")
		workDirPath = strings.ReplaceAll(workDirPath, "\\", "/")

		pastelRqServicePath := filepath.Join(string(remotePastelExecPath), constants.PastelRQServiceExecName[constants.OSType(string(remoteOsType))])
		pastelRqServicePath = strings.ReplaceAll(pastelRqServicePath, "\\", "/")

		go client.Cmd(fmt.Sprintf("%s %s", pastelRqServicePath, fmt.Sprintf("--config-file=%s", workDirPath))).Run()

		time.Sleep(10000 * time.Millisecond)

	}

	log.WithContext(ctx).Infof("Remote:::The %s started succesfully!", commandName)
	return nil
}

func runSuperNodeRemote(ctx context.Context, config *configs.Config, client *utils.Client, extIP string, pastelid string) (err error) {
	log.WithContext(ctx).Info("Remote:::Starting supernode")
	log.WithContext(ctx).Debug("Remote:::Configure supernode setting")

	log.WithContext(ctx).Info("Remote:::Configuring supernode was finished")
	log.WithContext(ctx).Info("Remote:::Start supernode")

	remoteWorkDirPath, remotePastelExecPath, remoteOsType, err := getRemoteInfo(config, client)
	if err != nil {
		return err
	}

	remoteSuperNodePath := filepath.Join(string(remoteWorkDirPath), "supernode")

	var remoteSuperNodeConfigFilePath = filepath.Join(remoteSuperNodePath, "supernode.yml")

	var remoteSupernodeExecFile string

	remoteSuperNodeConfigFilePath = strings.ReplaceAll(remoteSuperNodeConfigFilePath, "\\", "/")
	remoteSupernodeExecFile = filepath.Join(string(remotePastelExecPath), constants.SuperNodeExecName[constants.OSType(string(remoteOsType))])
	remoteSupernodeExecFile = strings.ReplaceAll(remoteSupernodeExecFile, "\\", "/")

	client.Cmd(fmt.Sprintf("rm %s", remoteSuperNodeConfigFilePath)).Run()

	client.Cmd(fmt.Sprintf("echo -e \"%s\" >> %s", configs.SupernodeYmlLine1, remoteSuperNodeConfigFilePath)).Run()
	client.Cmd(fmt.Sprintf("echo -e \"%s\" >> %s", configs.SupernodeYmlLine2, remoteSuperNodeConfigFilePath)).Run()
	client.Cmd(fmt.Sprintf("echo -e \"%s\" >> %s", configs.SupernodeYmlLine3, remoteSuperNodeConfigFilePath)).Run()
	client.Cmd(fmt.Sprintf("echo -e \"%s\" >> %s", fmt.Sprintf(configs.SupernodeYmlLine4, pastelid), remoteSuperNodeConfigFilePath)).Run()
	client.Cmd(fmt.Sprintf("echo -e \"%s\" >> %s", configs.SupernodeYmlLine5, remoteSuperNodeConfigFilePath)).Run()
	client.Cmd(fmt.Sprintf("echo -e \"%s\" >> %s", configs.SupernodeYmlLine6, remoteSuperNodeConfigFilePath)).Run()
	client.Cmd(fmt.Sprintf("echo -e \"%s\" >> %s", fmt.Sprintf(configs.SupernodeYmlLine7, extIP), remoteSuperNodeConfigFilePath)).Run()
	client.Cmd(fmt.Sprintf("echo -e \"%s\" >> %s", fmt.Sprintf(configs.SupernodeYmlLine8, fmt.Sprintf("%d", flagMasterNodeRPCPort)), remoteSuperNodeConfigFilePath)).Run()

	time.Sleep(5000 * time.Millisecond)

	log.WithContext(ctx).Infof("Remote:::Start supernode command : %s", fmt.Sprintf("%s %s", remoteSupernodeExecFile, fmt.Sprintf("--config-file=%s", remoteSuperNodeConfigFilePath)))

	go client.Cmd(fmt.Sprintf("%s %s", remoteSupernodeExecFile, fmt.Sprintf("--config-file=%s", remoteSuperNodeConfigFilePath))).Run()

	defer client.Close()

	log.WithContext(ctx).Info("Remote:::Waiting for supernode started...")
	time.Sleep(5000 * time.Millisecond)

	log.WithContext(ctx).Info("Remote:::Supernode was started successfully")
	return nil
}

func runPastelNode(ctx context.Context, config *configs.Config, reindex bool, interactive bool) (err error) {
	var pastelDPath string

	if _, pastelDPath, _, _, err = checkPastelInstallPath(ctx, config, ""); err != nil {
		return errNotFoundPastelPath
	}

	if err = checkStartNodeParams(ctx, config); err != nil {
		return err
	}

	var pasteldArgs = fmt.Sprintf("--%s  --datadir=%s", config.Network, config.WorkingDir)

	if interactive {
		if reindex {
			log.WithContext(ctx).Infof("Starting -> %s --externalip=%s --txindex=1 --reindex %s", pastelDPath, flagNodeExtIP, pasteldArgs)
			if err = RunPasteldWithInteractive(ctx, config, fmt.Sprintf("--externalip=%s", flagNodeExtIP), "--reindex", "--txindex==1"); err != nil {
				return err
			}
		} else {
			log.WithContext(ctx).Infof("Starting -> %s --externalip=%s %s", pastelDPath, flagNodeExtIP, pasteldArgs)
			if err = RunPasteldWithInteractive(ctx, config, fmt.Sprintf("--externalip=%s", flagNodeExtIP)); err != nil {
				return err
			}
		}

	} else {
		if reindex {
			log.WithContext(ctx).Infof("Starting -> %s --externalip=%s --txindex=1 --reindex --daemon %s", pastelDPath, flagNodeExtIP, pasteldArgs)
			go RunPasteld(ctx, config, fmt.Sprintf("--externalip=%s", flagNodeExtIP), "--reindex", "--txindex=1", "--daemon")
		} else {
			log.WithContext(ctx).Infof("Starting -> %s --externalip=%s --daemon %s", pastelDPath, flagNodeExtIP, pasteldArgs)
			go RunPasteld(ctx, config, fmt.Sprintf("--externalip=%s", flagNodeExtIP), "--daemon")
		}

		if !checkPastelDRunning(ctx, config) {
			return errNotStartPasteld
		}
	}

	return nil
}

func checkAndForceInit(ctx context.Context, config *configs.Config) (err error) {
	var errPastelExecutable, errPastelParam error

	if flagCheckForce {

		// Check Pastel Executable Files
		log.WithContext(ctx).Info("Checking pastel executable files...")
		_, _, _, _, errPastelExecutable = checkPastelInstallPath(ctx, config, "wallet")

		// Check Pastel Param Flies
		log.WithContext(ctx).Info("Checking pastel param files...")
		errPastelParam = checkPastelParamInstallPath(ctx, config)
		if errPastelExecutable != nil || errPastelParam != nil {
			log.WithContext(ctx).Warn("Walletnode is not installed correctly.")
			config.Force = true
			if len(config.WorkingDir) == 0 {
				if config.WorkingDir, err = configurer.DefaultWorkingDir(); err != nil {
					return err
				}

			}
			if len(config.PastelExecDir) == 0 {
				if config.PastelExecDir, err = configurer.DefaultPastelExecutableDir(); err != nil {
					return err
				}
			}
			if len(config.Version) == 0 {
				config.Version = "latest"
			}
			if err := InitializeFunc(ctx, config); err != nil {
				return err
			}

			log.WithContext(ctx).Info("Installing Walletnode...")
			runInstallWalletSubCommand(ctx, config)

			if len(config.WorkingDir) != 0 {
				InitializeFunc(ctx, config)
			}

			err = updatePastelConfigFileForNetwork(ctx, filepath.Join(config.WorkingDir, "pastel.conf"), config)

			if err != nil {
				return err
			}
		}
	} else {
		if _, _, _, _, errPastelExecutable = checkPastelInstallPath(ctx, config, "wallet"); errPastelExecutable != nil {
			return errNotFoundPastelPath
		}

		if errPastelParam = checkPastelParamInstallPath(ctx, config); errPastelParam != nil {
			return errNotFoundPastelParamPath
		}

	}
	return nil
}

func operateMasterNodeConf(ctx context.Context, config *configs.Config, masternodePrivKey string, pastelid string, isCreate bool) (err error) {
	confData := map[string]interface{}{
		flagMasterNodeName: map[string]string{
			"mnAddress":  flagNodeExtIP + ":" + fmt.Sprintf("%d", flagMasterNodePort),
			"mnPrivKey":  masternodePrivKey,
			"txid":       flagMasterNodeTxID,
			"outIndex":   flagMasterNodeIND,
			"extAddress": flagNodeExtIP + ":" + fmt.Sprintf("%d", flagMasterNodeRPCPort),
			"p2pAddress": flagMasterNodeP2PIP + ":" + fmt.Sprintf("%d", flagMasterNodeP2PPort),
			"extCfg":     "",
			"extKey":     pastelid,
		},
	}

	if isCreate {
		data, _ := json.Marshal(confData)
		// Create masternode.conf file
		if err = createConfFile(data, config); err != nil {
			return err
		}
	} else {
		// Create masternode.conf file
		if _, err = updateMasternodeConfFile(confData, config); err != nil {
			return err
		}
	}

	data, _ := json.Marshal(confData)
	log.WithContext(ctx).Infof("masternode.conf = %s", string(data))

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

func credentials(pwdToo bool) (string, string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}

	password := ""
	if pwdToo {
		fmt.Print("Enter Password: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return "", "", err
		}
		fmt.Print("\n")
		password = string(bytePassword)
	}

	return strings.TrimSpace(username), strings.TrimSpace(password), nil
}

func checkStartMasterNodeParams(_ context.Context, config *configs.Config) error {
	if len(flagMasterNodeName) == 0 {
		return errMasterNodeNameRequired
	}

	if len(flagNodeExtIP) == 0 {
		if flagMasterNodeColdHot {
			return errGetExternalIP
		}

		externalIP, err := GetExternalIPAddress()

		if err != nil {
			return errGetExternalIP
		}
		flagNodeExtIP = externalIP
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
			return flagNodeExtIP
		}
		return flagMasterNodeRPCIP
	}()
	flagMasterNodeP2PIP = func() string {
		if len(flagMasterNodeP2PIP) == 0 {
			return flagNodeExtIP
		}
		return flagMasterNodeP2PIP
	}()

	if config.Network == "testnet" {
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

	if config.Network == "testnet" {
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

	if config.Network == "testnet" {
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

	if config.Network == "testnet" {
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

func getStartInfo(config *configs.Config) (nodeName string, privKey string, extIP string, pastelID string, extPort string, err error) {
	var masternodeConfPath string

	if config.Network == "testnet" {
		masternodeConfPath = filepath.Join(config.WorkingDir, "testnet3", "masternode.conf")
	} else {
		masternodeConfPath = filepath.Join(config.WorkingDir, "masternode.conf")
	}

	// Read ConfData from masternode.conf
	confFile, err := ioutil.ReadFile(masternodeConfPath)
	if err != nil {
		return "", "", "", "", "", err
	}

	var conf map[string]interface{}
	json.Unmarshal([]byte(confFile), &conf)

	for key := range conf {
		nodeName = key // get Node Name
	}
	confData := conf[nodeName].(map[string]interface{})
	extAddr := strings.Split(confData["mnAddress"].(string), ":") // get Ext IP
	extKey := confData["extKey"].(string)

	return nodeName, confData["mnPrivKey"].(string), extAddr[0], extKey, extAddr[1], nil
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
		log.WithContext(ctx).Warn("could not find pastel.conf")
		return "", "", "", "", fmt.Errorf("could not find pastel.conf")
	}

	if _, err = os.Stat(config.PastelExecDir); os.IsNotExist(err) {
		log.WithContext(ctx).Warn("could not find pastel node path")
		return "", "", "", "", fmt.Errorf("could not find pastel node path")
	}
	pastelDirPath = config.PastelExecDir

	if _, err = os.Stat(filepath.Join(config.PastelExecDir, constants.PasteldName[utils.GetOS()])); os.IsNotExist(err) {
		log.WithContext(ctx).Warn("could not find pasteld path")
		return "", "", "", "", fmt.Errorf("could not find pasteld path")
	}
	pasteldPath = filepath.Join(config.PastelExecDir, constants.PasteldName[utils.GetOS()])

	if _, err = os.Stat(filepath.Join(config.PastelExecDir, constants.PastelCliName[utils.GetOS()])); os.IsNotExist(err) {
		log.WithContext(ctx).Warn("could not find pastel-cli path")
		return "", "", "", "", fmt.Errorf("could not find pastel-cli path")
	}
	pastelCliPath = filepath.Join(config.PastelExecDir, constants.PastelCliName[utils.GetOS()])
	if flagMode == "wallet" {
		if _, err = os.Stat(filepath.Join(config.PastelExecDir, constants.WalletNodeExecName[utils.GetOS()])); os.IsNotExist(err) {
			log.WithContext(ctx).Warn("could not find wallet node path")
			return "", "", "", "", fmt.Errorf("could not find wallet node path")
		}
		pastelWalletNodePath = filepath.Join(config.PastelExecDir, constants.WalletNodeExecName[utils.GetOS()])
	}

	return pastelDirPath, pasteldPath, pastelCliPath, pastelWalletNodePath, err
}

func checkPastelParamInstallPath(ctx context.Context, config *configs.Config) (err error) {

	var tmpPath string
	if tmpPath, err = configurer.DefaultWorkingDir(); err != nil {
		return err
	}

	for _, zksnarkParamsName := range configs.ZksnarkParamsNames {
		zksnarkPath := ""
		if config.WorkingDir == tmpPath {
			if zksnarkPath, err = configurer.DefaultZksnarkDir(); err != nil {
				return err
			}
		} else {
			zksnarkPath = filepath.Join(config.WorkingDir, "/.pastel-params/")
		}
		zksnarkParamsPath := filepath.Join(zksnarkPath, zksnarkParamsName)

		log.WithContext(ctx).Infof(fmt.Sprintf("Checking pastel param file : %s", zksnarkParamsPath))
		checkSum, checkSumerr := utils.GetChecksum(ctx, zksnarkParamsPath)
		if checkSumerr != nil {
			return checkSumerr
		} else if checkSum != constants.PastelParamsCheckSums[zksnarkParamsName] {
			log.WithContext(ctx).Errorf(fmt.Sprintf("Checking pastel param file : %s\n", zksnarkParamsPath))
			return errors.Errorf(fmt.Sprintf("Checking pastel param file : %s\n", zksnarkParamsPath))
		}

	}

	return nil
}

func checkPastelDRunning(ctx context.Context, config *configs.Config) (ret bool) {
	var failCnt = 0
	var err error

	log.WithContext(ctx).Info("Waiting the pasteld to be started...")

	for {
		if _, err = runPastelCLI(ctx, config, "getinfo"); err != nil {
			time.Sleep(10000 * time.Millisecond)
			failCnt++
			if failCnt == 10 {
				return false
			}
		} else {
			break
		}
	}

	log.WithContext(ctx).Info("pasteld was started successfully")
	return true
}

func stopPastelDAndWait(ctx context.Context, config *configs.Config) (err error) {
	log.WithContext(ctx).Info("Stopping pasteld...")
	if _, err = runPastelCLI(ctx, config, "stop"); err != nil {
		return err
	}

	time.Sleep(10000 * time.Millisecond)
	log.WithContext(ctx).Info("Stopped pasteld")
	return nil
}

func checkMasterNodeSync(ctx context.Context, config *configs.Config) (err error) {
	var mnstatus structure.RPCPastelMSStatus
	var output string
	for {
		if output, err = runPastelCLI(ctx, config, "mnsync", "status"); err != nil {
			return err
		}

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

	return nil
}

func checkMasterNodeSyncRemote(ctx context.Context, _ *configs.Config, client *utils.Client, pastelCliPath string) (err error) {
	var mnstatus structure.RPCPastelMSStatus
	var output []byte

	for {
		if output, err = client.Cmd(fmt.Sprintf("%s mnsync status", pastelCliPath)).Output(); err != nil {
			return err
		}
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
		}
		if mnstatus.IsSynced {
			log.WithContext(ctx).Info("Remote:::master node was synced!")
			break
		}
		log.WithContext(ctx).Info("Remote:::Waiting for sync...")
		time.Sleep(10000 * time.Millisecond)
	}
	return nil
}

func getRemoteInfo(config *configs.Config, client *utils.Client) (remoteWorkDirPath []byte, remotePastelExecPath []byte, remoteOsType []byte, err error) {

	remotePastelUtilityExec := filepath.Join(config.RemotePastelUtilityDir, "pastel-utility")
	remotePastelUtilityExec = strings.ReplaceAll(remotePastelUtilityExec, "\\", "/")

	remoteWorkDirPath, err = client.Cmd(fmt.Sprintf("%s info --work-dir", remotePastelUtilityExec)).Output()
	if err != nil {
		return nil, nil, nil, err

	}

	remotePastelExecPath, err = client.Cmd(fmt.Sprintf("%s info --exec-dir", remotePastelUtilityExec)).Output()
	if err != nil {
		return nil, nil, nil, err

	}

	remoteOsType, err = client.Cmd(fmt.Sprintf("%s info --os-version", remotePastelUtilityExec)).Output()
	if err != nil {
		return nil, nil, nil, err

	}

	return remoteWorkDirPath, remotePastelExecPath, remoteOsType, nil
}

func getMasternodeConf(ctx context.Context, config *configs.Config) (pastelid string, err error) {
	var masternodePrivKey, output string
	if flagMasterNodeIsCreate || flagMasterNodeIsUpdate {

		if flagMasterNodeIsCreate {
			if err = backupConfFile(config); err != nil { // delete conf file
				return "", err
			}
			log.WithContext(ctx).Info("Backup masternode.conf was finished successfully.")

			log.WithContext(ctx).Infof("Starting -> ./pasteld --externalip=%s --reindex --daemon", flagNodeExtIP)
			go RunPasteld(ctx, config, fmt.Sprintf("--externalip=%s", flagNodeExtIP), "--reindex", "--daemon")

			if !checkPastelDRunning(ctx, config) {
				return "", errNotStartPasteld
			}

			if output, err = runPastelCLI(ctx, config, "getaccountaddress", ""); err != nil {
				return "", err
			}
			log.WithContext(ctx).Infof("Hot wallet address = %s", output)

			// ***************  3.1 Search collateral transaction  ***************
			if output, err = runPastelCLI(ctx, config, "masternode", "outputs"); err != nil {
				log.WithContext(ctx).Error("This above command doesn't run!")
				return "", err
			}
			var recMasterNode map[string]interface{}
			json.Unmarshal([]byte(output), &recMasterNode)

			if len(recMasterNode) != 0 {

				if recMasterNode[flagMasterNodeTxID] != nil && recMasterNode[flagMasterNodeTxID] == flagMasterNodeIND {
					// if receives PSL go to next step
					log.WithContext(ctx).Infof("masternode outputs = %s, %s", flagMasterNodeTxID, flagMasterNodeIND)
				}
			}

			// ***************  3.2 Create new MN private key  ***************
			if len(flagMasterNodePrivateKey) == 0 {
				if masternodePrivKey, err = runPastelCLI(ctx, config, "masternode", "genkey"); err != nil {
					return "", err
				}
				flagMasterNodePrivateKey = strings.TrimSuffix(masternodePrivKey, "\n")
			} else {
				masternodePrivKey = flagMasterNodePrivateKey
			}
			log.WithContext(ctx).Infof("masternode private key = %s", masternodePrivKey)

			// ***************  3.3 create new pastelid  ***************
			if len(flagMasterNodePastelID) == 0 && len(flagMasterNodePassPhrase) != 0 {
				// Check masternode status
				if err = checkMasterNodeSync(ctx, config); err != nil {
					return "", err
				}

				if output, err = runPastelCLI(ctx, config, "pastelid", "newkey", flagMasterNodePassPhrase); err != nil {
					return "", err
				}

				var pastelidSt structure.RPCPastelID
				if err = json.Unmarshal([]byte(output), &pastelidSt); err != nil {
					return "", err
				}
				pastelid = pastelidSt.Pastelid
			} else {
				pastelid = flagMasterNodePastelID
			}

			log.WithContext(ctx).Infof("pastelid = %s", pastelid)

			if !flagMasterNodeColdHot {
				if !checkPastelDRunning(ctx, config) {
					return "", errNotStartPasteld
				}

				if output, err = runPastelCLI(ctx, config, "getaccountaddress", ""); err != nil {
					return "", err
				}
				log.WithContext(ctx).Infof("master node address = %s", output)
			}

			// ***************  3.4 Create or edit masternode.conf - this should be NEW masternode.conf, any existing should be backed up  ***************
			if err = operateMasterNodeConf(ctx, config, masternodePrivKey, pastelid, true); err != nil {
				return "", err
			}

			if err = stopPastelDAndWait(ctx, config); err != nil {
				return "", err
			}
		}

		if flagMasterNodeIsUpdate {
			// Make masternode conf data
			if err = operateMasterNodeConf(ctx, config, masternodePrivKey, pastelid, false); err != nil {
				return "", err
			}
		}
	}
	return pastelid, nil
}

func checkServiceRunning(_ context.Context, config *configs.Config, toolType constants.ToolType) bool {
	var pID string
	var processID int
	execPath := ""
	execName := ""
	switch toolType {
	case constants.RQService:
		execPath = filepath.Join(config.PastelExecDir, constants.PastelRQServiceExecName[utils.GetOS()])
		execName = constants.PastelRQServiceExecName[utils.GetOS()]
	default:
		execPath = filepath.Join(config.PastelExecDir, constants.PastelRQServiceExecName[utils.GetOS()])
		execName = constants.PastelRQServiceExecName[utils.GetOS()]
	}

	if utils.GetOS() == constants.Windows {
		arg := fmt.Sprintf("IMAGENAME eq %s", execName)
		out, err := RunCMD("tasklist", "/FI", arg)
		cnt := strings.Count(out, ",")
		if err != nil {
			return false
		}
		if strings.Contains(out, "No tasks") || cnt == 2 {
			return false
		}

	} else {
		matches, _ := filepath.Glob("/proc/*/exe")
		for _, file := range matches {
			target, _ := os.Readlink(file)
			if len(target) > 0 {
				if target == execPath {
					split := strings.Split(file, "/")

					pID = split[len(split)-2]
					processID, _ = strconv.Atoi(pID)
					_, err := os.FindProcess(processID)
					if err != nil {
						return false
					}
				}
			}
		}

	}

	return true
}
