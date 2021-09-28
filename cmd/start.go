package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/pastel-utility/constants"
	"github.com/pastelnetwork/pastel-utility/structure"
	"github.com/pastelnetwork/pastel-utility/utils"
	"github.com/pkg/errors"
)

/*var (
	wg sync.WaitGroup
)
*/
var (
	// node flags
	flagNodeExtIP string
	flagReIndex   bool

	// walletnode flag
	flagDevMode bool

	// masternode flags
	flagMasterNodeIsActivate bool

	flagMasterNodeName       string
	flagMasterNodeIsCreate   bool
	flagMasterNodeIsUpdate   bool
	flagMasterNodeTxID       string
	flagMasterNodeInd        string
	flagMasterNodePort       int
	flagMasterNodePrivateKey string
	flagMasterNodePastelID   string
	flagMasterNodePassPhrase string
	flagMasterNodeRPCIP      string
	flagMasterNodeRPCPort    int
	flagMasterNodeP2PIP      string
	flagMasterNodeP2PPort    int

	flagMasterNodeSSHIP   string
	flagMasterNodeSSHPort int
)

type startCommand uint8

const (
	nodeStart startCommand = iota
	walletStart
	superNodeStart
	superNodeColdHotStart
	rqService
	ddService
	wnService
	snService
)

func setupStartSubCommand(config *configs.Config,
	startCommand startCommand,
	f func(context.Context, *configs.Config) error,
) *cli.Command {
	commonFlags := []*cli.Flag{
		cli.NewFlag("ip", &flagNodeExtIP).
			SetUsage(green("Optional, WAN address of the host")),
		cli.NewFlag("dir", &config.PastelExecDir).SetAliases("d").
			SetUsage(green("Optional, Location of pastel node directory")).SetValue(config.Configurer.DefaultPastelExecutableDir()),
		cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
			SetUsage(green("Optional, location of working directory")).SetValue(config.Configurer.DefaultWorkingDir()),
		cli.NewFlag("reindex", &flagReIndex).SetAliases("r").
			SetUsage(green("Optional, Start with reindex")),
	}

	walletNodeFlags := []*cli.Flag{
		cli.NewFlag("development-mode", &flagDevMode),
	}

	superNodeFlags := []*cli.Flag{

		cli.NewFlag("activate", &flagMasterNodeIsActivate).
			SetUsage(green("Optional, if specified, will try to enable node as Masternode (start-alias).")),
		cli.NewFlag("name", &flagMasterNodeName).
			SetUsage(yellow("Required, name of the Masternode to start (and create or update in the masternode.conf if --create or --update are specified)")).SetRequired(),
		cli.NewFlag("pkey", &flagMasterNodePrivateKey).
			SetUsage(green("Optional, Masternode private key, if omitted, new masternode private key will be created")),

		cli.NewFlag("create", &flagMasterNodeIsCreate).
			SetUsage(green("Optional, if specified, will create Masternode record in the masternode.conf.")),
		cli.NewFlag("update", &flagMasterNodeIsUpdate).
			SetUsage(green("Optional, if specified, will update Masternode record in the masternode.conf.")),
		cli.NewFlag("txid", &flagMasterNodeTxID).
			SetUsage(yellow("Required (only if --update or --create specified), collateral payment txid , transaction id of 5M collateral MN payment")),
		cli.NewFlag("ind", &flagMasterNodeInd).
			SetUsage(yellow("Required (only if --update or --create specified), collateral payment output index , output index in the transaction of 5M collateral MN payment")),
		cli.NewFlag("pastelid", &flagMasterNodePastelID).
			SetUsage(green("Optional, pastelid of the Masternode. If omitted, new pastelid will be created and registered")),
		cli.NewFlag("passphrase", &flagMasterNodePassPhrase).
			SetUsage(yellow("Required (only if --update or --create specified), passphrase to pastelid private key")),
		cli.NewFlag("port", &flagMasterNodePort).
			SetUsage(green("Optional, Port for WAN IP address of the node , default - 9933 (19933 for Testnet)")),
		cli.NewFlag("rpc-ip", &flagMasterNodeRPCIP).
			SetUsage(green("Optional, supernode IP address. If omitted, value passed to --ip will be used")),
		cli.NewFlag("rpc-port", &flagMasterNodeRPCPort).
			SetUsage(green("Optional, supernode port, default - 4444 (14444 for Testnet")),
		cli.NewFlag("p2p-ip", &flagMasterNodeP2PIP).
			SetUsage(green("Optional, Kademlia IP address, if omitted, value passed to --ip will be used")),
		cli.NewFlag("p2p-port", &flagMasterNodeP2PPort).
			SetUsage(green("Optional, Kademlia port, default - 4445 (14445 for Testnet)")),
	}

	superNodeColdHotFlags := []*cli.Flag{
		cli.NewFlag("ssh-ip", &flagMasterNodeSSHIP).
			SetUsage(red("Required (only if --remote specified), remote supernode specific, SSH address of the remote HOT node")),
		cli.NewFlag("ssh-port", &flagMasterNodeSSHPort).
			SetUsage(green("Optional, remote supernode specific, SSH port of the remote HOT node")).SetValue(22),
		cli.NewFlag("remote-dir", &config.RemotePastelExecDir).
			SetUsage(green("Optional, Location where of pastel node directory on the remote computer (default: $HOME/pastel-utility)")),
		cli.NewFlag("remote-work-dir", &config.RemoteWorkingDir).
			SetUsage(green("Optional, Location of working directory on the remote computer (default: $HOME/.pastel")).SetValue("$HOME/.pastel"),
		cli.NewFlag("ssh-key", &sshKey).
			SetUsage(yellow("Optional, Path to SSH private key")),
		cli.NewFlag("ssh-dir", &config.RemotePastelUtilityDir).SetAliases("rpud").
			SetUsage(yellow("Required, Location where to copy pastel-utility on the remote computer")).SetRequired(),
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
		commandName = string(constants.WalletNode)
		commandMessage = "Start walletnode"
	case superNodeStart:
		commandFlags = append(superNodeFlags, commonFlags[:]...)
		commandName = string(constants.SuperNode)
		commandMessage = "Start supernode"
	case superNodeColdHotStart:
		commandFlags = append(append(superNodeFlags, commonFlags[:]...), superNodeColdHotFlags[:]...)
		commandName = string(constants.SuperNode) + "-coldhot"
		commandMessage = "Start supernode in Cold/Hod mode"
	case rqService:
		commandFlags = commonFlags
		commandName = string(constants.RQService)
		commandMessage = "Start RaptorQ service only"
	case ddService:
		commandFlags = commonFlags
		commandName = string(constants.DDService)
		commandMessage = "Start dupe detection service only"
	case wnService:
		commandFlags = commonFlags
		commandName = string(constants.WalletNode) + "-service"
		commandMessage = "Start WalletNode service only"
	case snService:
		commandFlags = commonFlags
		commandName = string(constants.SuperNode) + "-service"
		commandMessage = "Start SuperNode service only"
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
	config := configs.InitConfig()

	startNodeSubCommand := setupStartSubCommand(config, nodeStart, runStartNodeSubCommand)
	startWalletSubCommand := setupStartSubCommand(config, walletStart, runStartWalletSubCommand)
	startSuperNodeSubCommand := setupStartSubCommand(config, superNodeStart, runLocalSuperNodeSubCommand)
	startSuperNodeCOldHotSubCommand := setupStartSubCommand(config, superNodeColdHotStart, runSuperNodeColdHotSubCommand)

	startRQServiceCommand := setupStartSubCommand(config, rqService, runRQService)
	startDDServiceCommand := setupStartSubCommand(config, ddService, runDDService)
	startWNServiceCommand := setupStartSubCommand(config, wnService, runWalletNodeService)
	startSNServiceCommand := setupStartSubCommand(config, snService, runSuperNodeService)

	startCommand := cli.NewCommand("start")
	startCommand.SetUsage(blue("Performs start of the system for both WalletNode and SuperNodes"))
	startCommand.AddSubcommands(startNodeSubCommand)
	startCommand.AddSubcommands(startWalletSubCommand)
	startCommand.AddSubcommands(startSuperNodeSubCommand)
	startCommand.AddSubcommands(startSuperNodeCOldHotSubCommand)

	startCommand.AddSubcommands(startRQServiceCommand)
	startCommand.AddSubcommands(startDDServiceCommand)
	startCommand.AddSubcommands(startWNServiceCommand)
	startCommand.AddSubcommands(startSNServiceCommand)

	return startCommand

}

///// Top level start commands

// Sub Command
func runStartNodeSubCommand(ctx context.Context, config *configs.Config) error {
	if err := runPastelNode(ctx, config, flagReIndex, flagNodeExtIP, ""); err != nil {
		log.WithContext(ctx).WithError(err).Error("pasteld failed to start")
		return err
	}
	return nil
}

// Sub Command
func runStartWalletSubCommand(ctx context.Context, config *configs.Config) error {
	// *************  1. Start pastel node  *************
	if err := runPastelNode(ctx, config, flagReIndex, flagNodeExtIP, ""); err != nil {
		log.WithContext(ctx).WithError(err).Error("pasteld failed to start")
		return err
	}

	// *************  2. Start rq-servce    *************
	if err := runRQService(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("rqservice failed to start")
		return err
	}

	// *************  3. Start wallet node  *************
	if err := runWalletNodeService(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("walletnode failed to start")
		return err
	}

	return nil
}

// Sub Command
func runLocalSuperNodeSubCommand(ctx context.Context, config *configs.Config) error {

	// *************  1. Parse pastel config parameters  *************
	log.WithContext(ctx).Info("Reading pastel.conf")
	if err := ParsePastelConf(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to parse pastel config")
		return err
	}
	log.WithContext(ctx).Infof("Finished Reading pastel.conf! Starting Supernode in %s mode", config.Network)

	// *************  2. Parse parameters  *************
	log.WithContext(ctx).Info("Checking arguments")
	if err := checkStartMasterNodeParams(ctx, config, false); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to validate input arguments")
		return err
	}
	log.WithContext(ctx).Info("Finished checking arguments!")

	if flagMasterNodeIsCreate || flagMasterNodeIsUpdate {
		log.WithContext(ctx).Info("Prepare masternode parameters")
		if err := prepareMasterNodeParameters(ctx, config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to validate and prepare masternode parameters")
			return err
		}
		if err := createOrUpdateMasternodeConf(ctx, config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to create or update masternode.conf")
			return err
		}
		if err := createOrUpdateSuperNodeConfig(ctx, config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to update supernode.yml")
			return err
		}
	}

	// Get conf data from masternode.conf File
	privKey, extIP, _ /*extPort*/, err := getMasternodeConfData(ctx, config, flagMasterNodeName)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to get masternode details from masternode.conf")
		return err
	}

	if extIP != flagNodeExtIP {
		err := errors.Errorf("External IP address in masternode.conf MUST match WAN address of the node! IP in masternode.conf - %s, WAN IP passed or identified - %s", extIP, flagNodeExtIP)
		log.WithContext(ctx).WithError(err).Error("pasteld failed to start")
		return err
	}

	// *************  3. Start Node as Masternode  *************
	log.WithContext(ctx).Infof("Starting pasteld as masternode: nodeName: %s; mnPrivKey: %s", flagMasterNodeName, privKey)
	if err := runPastelNode(ctx, config, true, flagNodeExtIP, privKey); err != nil { //in masternode mode pasteld MUST be started with reindex flag
		log.WithContext(ctx).WithError(err).Error("pasteld failed to start as masternode")
		return err
	}

	// *************  4. Wait for blockchain and masternodes sync  *************
	if err := CheckMasterNodeSync(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("pasteld failed to synchronize, add some peers and try again")
		return err
	}

	// *************  5. Enable Masternode  ***************
	if flagMasterNodeIsActivate {
		log.WithContext(ctx).Infof("Starting MN alias - %s", flagMasterNodeName)
		if err := runStartAliasMasternode(ctx, config, flagMasterNodeName); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to start alias - %s", flagMasterNodeName)
			return err
		}
	}

	// *************  6. Start rq-servce    *************
	if err := runRQService(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("rqservice failed to start")
		return err
	}

	// *************  6. Start dd-servce    *************
	if err := runDDService(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("ddservice failed to start")
		return err
	}

	// *************  7. Start supernode  **************
	if err := runSuperNodeService(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to start supernode service")
		return err
	}

	return nil
}

// Sub Command
func runRQService(ctx context.Context, config *configs.Config) error {

	rqExecName := constants.PastelRQServiceExecName[utils.GetOS()]

	var rqServiceArgs []string
	rqServiceArgs = append(rqServiceArgs,
		fmt.Sprintf("--config-file=%s", config.Configurer.GetRQServiceConfFile(config.WorkingDir)))

	if err := runPastelService(ctx, config, constants.RQService, rqExecName, rqServiceArgs...); err != nil {
		log.WithContext(ctx).WithError(err).Error("rqservice failed")
		return err
	}
	return nil
}

// Sub Command
func runDDService(ctx context.Context, config *configs.Config) (err error) {
	log.WithContext(ctx).Infof("Starting dupe detection service")

	var execPath string
	if execPath, err = checkPastelFilePath(ctx, config.PastelExecDir, constants.DupeDetectionExecName); err != nil {
		log.WithContext(ctx).WithError(err).Error("Could not find dupe detection service script")
		return err
	}

	ddConfigFilePath := filepath.Join(config.Configurer.GetHomeDir(),
		"pastel_dupe_detection_service",
		"dupe_detection_support_files",
		"config.ini")

	go RunCMD("python3", execPath, ddConfigFilePath)

	time.Sleep(10 * time.Second)

	if output, err := FindRunningProcess(constants.DupeDetectionExecName); len(output) == 0 {
		err = errors.Errorf("dd-service failed to start")
		log.WithContext(ctx).WithError(err).Error("dd-service failed to start")
		return err
	} else if err != nil {
		log.WithContext(ctx).WithError(err).Error("failed to test if dd-servise is running")
	} else {
		log.WithContext(ctx).Info("dd-service is successfully started")
	}

	return nil
}

// Sub Command
func runWalletNodeService(ctx context.Context, config *configs.Config) error {

	walletnodeExecName := constants.WalletNodeExecName[utils.GetOS()]
	log.WithContext(ctx).Infof("Starting walletnode service - %s", walletnodeExecName)

	var wnServiceArgs []string
	wnServiceArgs = append(wnServiceArgs,
		fmt.Sprintf("--config-file=%s", config.Configurer.GetWalletNodeConfFile(config.WorkingDir)))
	if flagDevMode {
		wnServiceArgs = append(wnServiceArgs, "--swagger")
	}

	log.WithContext(ctx).Infof("Options : %s", wnServiceArgs)
	if err := runPastelService(ctx, config, constants.WalletNode, walletnodeExecName, wnServiceArgs...); err != nil {
		log.WithContext(ctx).WithError(err).Error("walletnode service failed")
		return err
	}
	return nil
}

// Sub Command
func runSuperNodeService(ctx context.Context, config *configs.Config) error {

	supernodeConfigPath := config.Configurer.GetSuperNodeConfFile(config.WorkingDir)
	supernodeExecName := constants.SuperNodeExecName[utils.GetOS()]
	log.WithContext(ctx).Infof("Starting Supernode service - %s", supernodeExecName)

	var snServiceArgs []string
	snServiceArgs = append(snServiceArgs,
		fmt.Sprintf("--config-file=%s", supernodeConfigPath))

	log.WithContext(ctx).Infof("Options : %s", snServiceArgs)
	if err := runPastelService(ctx, config, constants.SuperNode, supernodeExecName, snServiceArgs...); err != nil {
		log.WithContext(ctx).WithError(err).Error("supernode failed")
		return err
	}
	return nil
}

///// Run helpers
func runPastelNode(ctx context.Context, config *configs.Config, reindex bool, extIP string, mnPrivKey string) (err error) {
	var pastelDPath string

	if pastelDPath, err = checkPastelFilePath(ctx, config.PastelExecDir, constants.PasteldName[utils.GetOS()]); err != nil {
		log.WithContext(ctx).WithError(err).Error("Could not find pasteld")
		return err
	}

	if _, err = checkPastelFilePath(ctx, config.WorkingDir, constants.PastelConfName); err != nil {
		log.WithContext(ctx).WithError(err).Error("Could not find pastel.conf")
		return err
	}
	if err = CheckZksnarkParams(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("Wrong ZKSnark files")
		return err
	}

	if len(extIP) == 0 {
		if extIP, err = GetExternalIPAddress(); err != nil {
			log.WithContext(ctx).WithError(err).Error("Could not get external IP address")
			return err
		}
	}

	var pasteldArgs []string
	pasteldArgs = append(pasteldArgs,
		fmt.Sprintf("--datadir=%s", config.WorkingDir),
		fmt.Sprintf("--externalip=%s", extIP))

	if reindex {
		pasteldArgs = append(pasteldArgs, "--reindex", "--txindex=1")
	}

	if len(mnPrivKey) != 0 {
		pasteldArgs = append(pasteldArgs, "--masternode", fmt.Sprintf("--masternodeprivkey=%s", mnPrivKey))
	}

	log.WithContext(ctx).Infof("Starting -> %s %s", pastelDPath, strings.Join(pasteldArgs, " "))

	pasteldArgs = append(pasteldArgs, "--daemon")
	go RunCMD(pastelDPath, pasteldArgs...)

	if !CheckPastelDRunning(ctx, config) {
		err = fmt.Errorf("pasteld was not started")
		log.WithContext(ctx).WithError(err).Error("pasteld didn't start")
		return err
	}

	return nil
}

func runPastelService(ctx context.Context, config *configs.Config, toolType constants.ToolType, toolFileName string, args ...string) (err error) {

	log.WithContext(ctx).Infof("Starting %s", toolType)

	var execPath string
	if execPath, err = checkPastelFilePath(ctx, config.PastelExecDir, toolFileName); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Could not find %s", toolType)
		return err
	}

	go RunCMD(execPath, args...)
	time.Sleep(10 * time.Second)

	log.WithContext(ctx).Infof("Check %s is running...", toolType)
	isServiceRunning := CheckProcessRunning(toolType)
	if isServiceRunning {
		log.WithContext(ctx).Infof("The %s started succesfully!", toolType)
	} else {
		if output, err := RunCMD(execPath, args...); err != nil {
			log.WithContext(ctx).Errorf("%s start failed! : %s", toolType, output)
			return err
		}
	}

	return nil
}

///// Validates input parameters
func checkStartMasterNodeParams(ctx context.Context, config *configs.Config, coldHot bool) error {

	// --name supernode name - Required, name of the Masternode to start and create in the masternode.conf if --create or --update are specified
	if len(flagMasterNodeName) == 0 {
		err := fmt.Errorf("required: --name, name of the Masternode to start")
		log.WithContext(ctx).WithError(err).Error("Missing parameter --name")
		return err
	}

	// --ip WAN IP address of the node - Required, WAN address of the host
	if len(flagNodeExtIP) == 0 && !coldHot { //coldHot will try to get WAN address in the step that is executed on remote host

		log.WithContext(ctx).Info("--ip flag is ommited, trying to get our WAN IP address")
		externalIP, err := GetExternalIPAddress()
		if err != nil {
			err := fmt.Errorf("cannot get external ip address")
			log.WithContext(ctx).WithError(err).Error("Missing parameter --ip")
			return err
		}
		flagNodeExtIP = externalIP
		log.WithContext(ctx).Infof("WAN IP address - %s", flagNodeExtIP)
	}

	if !flagMasterNodeIsCreate { // if we don't create new masternode.conf - it must exist!
		var masternodeConfPath string
		if config.Network == constants.NetworkTestnet {
			masternodeConfPath = filepath.Join("testnet3", "masternode.conf")
		} else {
			masternodeConfPath = "masternode.conf"
		}

		if _, err := checkPastelFilePath(ctx, config.WorkingDir, masternodeConfPath); err != nil {
			log.WithContext(ctx).WithError(err).Error("Could not find masternode.conf - use --create flag")
			return err
		}
	}

	if coldHot {
		if len(flagMasterNodeSSHIP) == 0 {
			err := fmt.Errorf("required if --coldhot is specified, –-ssh-ip, SSH address of the remote HOT node")
			log.WithContext(ctx).WithError(err).Error("Missing parameter --ssh-ip")
			return err
		}
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

	portList := GetSNPortList(config)

	flagMasterNodePort = func() int {
		if flagMasterNodePort == 0 {
			return portList[constants.NodePort]
		}
		return flagMasterNodePort
	}()
	flagMasterNodeRPCPort = func() int {
		if flagMasterNodeRPCPort == 0 {
			return portList[constants.SNPort]
		}
		return flagMasterNodeRPCPort
	}()
	flagMasterNodeP2PPort = func() int {
		if flagMasterNodeP2PPort == 0 {
			return portList[constants.P2PPort]
		}
		return flagMasterNodeP2PPort
	}()

	return nil
}

///// Run helpers
func prepareMasterNodeParameters(ctx context.Context, config *configs.Config) (err error) {

	// this function must only be called when --create or --update
	if !flagMasterNodeIsCreate && !flagMasterNodeIsUpdate {
		return nil
	}

	bReIndex := true // if masternode.conf exist pasteld MUST be start with reindex flag
	if flagMasterNodeIsCreate {
		if _, err = backupConfFile(ctx, config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to backup masternode.conf")
			return err
		}
		bReIndex = flagReIndex
	}

	log.WithContext(ctx).Infof("Starting pasteld")
	if err = runPastelNode(ctx, config, bReIndex, flagNodeExtIP, ""); err != nil {
		log.WithContext(ctx).WithError(err).Error("pasteld failed to start")
		return err
	}

	// Check masternode status
	if err = CheckMasterNodeSync(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("pasteld failed to synchronize, add some peers and try again")
		return err
	}

	if err := checkCollateral(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("Missing collateral transaction")
		return err
	}

	if err := checkPassphrase(ctx); err != nil {
		log.WithContext(ctx).WithError(err).Error("Missing passphrase")
		return err
	}

	if err := checkMasternodePrivKey(ctx, config, nil); err != nil {
		log.WithContext(ctx).WithError(err).Error("Missing masternode private key")
		return err
	}

	if err := checkPastelID(ctx, config, nil); err != nil {
		log.WithContext(ctx).WithError(err).Error("Missing masternode PastelID")
		return err
	}

	return StopPastelDAndWait(ctx, config)
}

func checkPastelID(ctx context.Context, config *configs.Config, client *utils.Client) (err error) {
	if len(flagMasterNodePastelID) == 0 {

		log.WithContext(ctx).Info("Masternode PastelID is empty - will create new one")

		if len(flagMasterNodePassPhrase) == 0 { //check one more time just because
			err := fmt.Errorf("required parameter if --create or --update specified: --passphrase <passphrase to pastelid private key>")
			log.WithContext(ctx).WithError(err).Error("Missing parameter --passphrase")
			return err
		}

		var pastelid string
		if client == nil {
			pastelid, err = RunPastelCLI(ctx, config, "pastelid", "newkey", flagMasterNodePassPhrase)
			if err != nil {
				log.WithContext(ctx).WithError(err).Error("Failed to generate new pastelid key")
				return err
			}
		} else {
			pastelcliPath := filepath.Join(config.RemotePastelExecDir, constants.PastelCliName[utils.GetOS()])
			out, err := client.Cmd(fmt.Sprintf("%s %s %s", pastelcliPath, "pastelid newkey",
				flagMasterNodePassPhrase)).Output()
			if err != nil {
				log.WithContext(ctx).WithError(err).Error("Failed to generate new pastelid key on Hot node")
				return err
			}
			pastelid = string(out)
			fmt.Println("generated pastel key on hotnode: ", pastelid)
		}

		var pastelidSt structure.RPCPastelID
		if err = json.Unmarshal([]byte(pastelid), &pastelidSt); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to parse pastelid json")
			return err
		}
		flagMasterNodePastelID = pastelidSt.Pastelid
	}
	log.WithContext(ctx).Infof("Masternode pastelid = %s", flagMasterNodePastelID)
	return nil
}

func runSuperNodeColdHotSubCommand(ctx context.Context, config *configs.Config) (err error) {
	runner := &ColdHotRunner{
		config: config,
		opts: &ColdHotRunnerOpts{
			sshUser: sshUser,
			sshIP:   flagMasterNodeSSHIP,
			sshPort: flagMasterNodeSSHPort,
			sshKey:  sshKey,
		},
	}

	log.WithContext(ctx).Info("run supernode coldhot init")
	if err := runner.Init(ctx); err != nil {
		log.WithContext(ctx).WithError(err).Error("init coldhot runner failed.")
		return err
	}

	log.WithContext(ctx).Info("running supernode coldhot runner")
	if err := runner.Run(ctx); err != nil {
		log.WithContext(ctx).WithError(err).Error("run coldhot runner failed.")
		return err
	}
	log.WithContext(ctx).Info("run supernode coldhot successfull")

	return nil
}

func checkMasternodePrivKey(ctx context.Context, config *configs.Config, client *utils.Client) (err error) {
	if len(flagMasterNodePrivateKey) == 0 {
		log.WithContext(ctx).Info("Masternode private key is empty - will create new one")

		var mnPrivKey string
		if client == nil {
			mnPrivKey, err = RunPastelCLI(ctx, config, "masternode", "genkey")
			if err != nil {
				log.WithContext(ctx).WithError(err).Error("Failed to generate new masternode private key")
				return err
			}
		} else {
			pastelcliPath := filepath.Join(config.RemotePastelExecDir, constants.PastelCliName[utils.GetOS()])
			cmd := fmt.Sprintf("%s %s", pastelcliPath, "masternode genkey")
			out, err := client.Cmd(cmd).Output()
			if err != nil {
				log.WithContext(ctx).WithField("cmd", cmd).WithField("out", string(out)).WithError(err).Error("Failed to generate new masternode private key on Hot node")
				return err
			}

			mnPrivKey = string(out)
			fmt.Println("generated priv key on hotnode: ", mnPrivKey)
		}

		flagMasterNodePrivateKey = strings.TrimSuffix(mnPrivKey, "\n")
	}
	log.WithContext(ctx).Infof("masternode private key = %s", flagMasterNodePrivateKey)
	return nil
}

func checkPassphrase(ctx context.Context) error {
	if len(flagMasterNodePassPhrase) == 0 {

		_, flagMasterNodePassPhrase = AskUserToContinue(ctx, "No --passphrase provided."+
			" Please type new passphrase and press Enter. Or N to exit")
		if strings.EqualFold(flagMasterNodePassPhrase, "n") ||
			len(flagMasterNodePassPhrase) == 0 {

			flagMasterNodePassPhrase = ""
			err := fmt.Errorf("required parameter if --create or --update specified: --passphrase <passphrase to pastelid private key>")
			log.WithContext(ctx).WithError(err).Error("User terminated - exiting")
			return err
		}
	}
	log.WithContext(ctx).Infof(red(fmt.Sprintf("passphrase - %s", flagMasterNodePassPhrase)))
	return nil
}

func getMasternodeOutputs(ctx context.Context, config *configs.Config) (map[string]string, error) {

	var mnOutputs map[string]string
	outputs, err := RunPastelCLI(ctx, config, "masternode", "outputs")
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to get masternode outputs from pasteld")
		return nil, err
	}
	if len(outputs) != 0 {
		if err := json.Unmarshal([]byte(outputs), &mnOutputs); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to parse masternode outputs json")
			return nil, err
		}
	}
	return mnOutputs, nil
}

func checkCollateral(ctx context.Context, config *configs.Config) (err error) {

	var address string

	if len(flagMasterNodeTxID) == 0 || len(flagMasterNodeInd) == 0 {

		log.WithContext(ctx).Warn(red("No collateral --txid and/or --ind provided"))
		yes, _ := AskUserToContinue(ctx, "Search existing masternode collateral ready transaction in the wallet? Y/N")

		if yes {

			var mnOutputs map[string]string
			mnOutputs, err = getMasternodeOutputs(ctx, config)
			if err != nil {
				log.WithContext(ctx).WithError(err).Error("Failed")
				return err
			}

			if len(mnOutputs) > 0 {

				n := 0
				arr := []string{}
				for txid, txind := range mnOutputs {
					log.WithContext(ctx).Warn(red(fmt.Sprintf("%d - %s:%s", n, txid, txind)))
					arr = append(arr, txid)
					n++
				}
				_, strNum := AskUserToContinue(ctx, "Enter number to use, or N to exit")
				dNum, err := strconv.Atoi(strNum)
				if err != nil || dNum < 0 || dNum >= n {
					err = fmt.Errorf("user terminated - no collateral funds")
					log.WithContext(ctx).WithError(err).Error("No collateral funds - exiting")
					return err
				}

				flagMasterNodeTxID = arr[dNum]
				flagMasterNodeInd = mnOutputs[flagMasterNodeTxID]
			} else {
				log.WithContext(ctx).Warn(red("No existing collateral ready transactions"))
			}
		}
	}

	if len(flagMasterNodeTxID) == 0 || len(flagMasterNodeInd) == 0 {

		collateralAmount := "5"
		collateralCoins := "PSL"
		if config.Network == constants.NetworkTestnet {
			collateralAmount = "1"
			collateralCoins = "LSP"
		}

		yes, _ := AskUserToContinue(ctx, fmt.Sprintf("Do you want to generate new local address and send %sM %s to it from another wallet? Y/N",
			collateralAmount, collateralCoins))

		if !yes {
			err = fmt.Errorf("no collateral funds")
			log.WithContext(ctx).WithError(err).Error("No collateral funds - exiting")
			return err
		}
		address, err = RunPastelCLI(ctx, config, "getnewaddress")
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to get new address")
			return err
		}
		address = strings.Trim(address, "\n")
		log.WithContext(ctx).Warnf(red(fmt.Sprintf("Your new address for collateral payment is %s", address)))
		log.WithContext(ctx).Warnf(red(fmt.Sprintf("Use another wallet to send exactly %sM %s to that address.", collateralAmount, collateralCoins)))
		_, newTxid := AskUserToContinue(ctx, "Enter txid of the send and press Enter to continue when ready")
		flagMasterNodeTxID = strings.Trim(newTxid, "\n")
	}

	for i := 1; i <= 10; i++ {

		var mnOutputs map[string]string
		mnOutputs, err = getMasternodeOutputs(ctx, config)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed")
			return err
		}

		txind, ok := mnOutputs[flagMasterNodeTxID]
		if ok {
			flagMasterNodeInd = txind
			break
		}

		log.WithContext(ctx).Info("Waiting for transaction...")
		time.Sleep(10 * time.Second)
		if i == 10 {
			yes, _ := AskUserToContinue(ctx, "Still no collateral transaction. Continue? - Y/N")
			if !yes {
				err := fmt.Errorf("user terminated")
				log.WithContext(ctx).WithError(err).Error("Exiting")
				return err
			}
			i = 1
		}
	}

	if len(flagMasterNodeTxID) == 0 || len(flagMasterNodeInd) == 0 {

		err := errors.Errorf("Cannot find masternode outputs = %s:%s", flagMasterNodeTxID, flagMasterNodeInd)
		log.WithContext(ctx).WithError(err).Error("Try again after some time")
		return err
	}

	// if receives PSL go to next step
	log.WithContext(ctx).Infof(red(fmt.Sprintf("masternode outputs = %s, %s", flagMasterNodeTxID, flagMasterNodeInd)))
	return nil
}

///// masternode.conf helpers
func createOrUpdateMasternodeConf(ctx context.Context, config *configs.Config) error {

	// this function must only be called when --create or --update
	if !flagMasterNodeIsCreate && !flagMasterNodeIsUpdate {
		return nil
	}

	confData := map[string]interface{}{
		flagMasterNodeName: map[string]string{
			"mnAddress":  flagNodeExtIP + ":" + fmt.Sprintf("%d", flagMasterNodePort),
			"mnPrivKey":  flagMasterNodePrivateKey,
			"txid":       flagMasterNodeTxID,
			"outIndex":   flagMasterNodeInd,
			"extAddress": flagNodeExtIP + ":" + fmt.Sprintf("%d", flagMasterNodeRPCPort),
			"extP2P":     flagMasterNodeP2PIP + ":" + fmt.Sprintf("%d", flagMasterNodeP2PPort),
			"extCfg":     "",
			"extKey":     flagMasterNodePastelID,
		},
	}
	data, err := json.Marshal(confData)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Invalid new masternode.conf data")
		return err
	}

	if flagMasterNodeIsCreate {
		// Create masternode.conf file
		if err := createConfFile(ctx, config, data); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to create new masternode.conf file")
			return err
		}
	} else if flagMasterNodeIsUpdate {
		// Create masternode.conf file
		if err := updateMasternodeConfFile(ctx, confData, config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to update existing masternode.conf file")
			return err
		}
	}

	log.WithContext(ctx).Infof("masternode.conf = %s", string(data))

	return nil
}

func createConfFile(ctx context.Context, config *configs.Config, confData []byte) error {

	masternodeConfPath, err := backupConfFile(ctx, config)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to backup previous masternode.conf file")
		return err
	}

	err = ioutil.WriteFile(masternodeConfPath, confData, 0644)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to create and write new masternode.conf file")
		return err
	}

	return nil
}

func backupConfFile(ctx context.Context, config *configs.Config) (masternodeConfPath string, err error) {
	workDirPath := config.WorkingDir

	var masternodeConfPathBackup string
	if config.Network == constants.NetworkTestnet {
		masternodeConfPath = filepath.Join(workDirPath, "testnet3", "masternode.conf")
		masternodeConfPathBackup = filepath.Join(workDirPath, "testnet3", "masternode_%s.conf")
	} else {
		masternodeConfPath = filepath.Join(workDirPath, "masternode.conf")
		masternodeConfPathBackup = filepath.Join(workDirPath, "masternode_%s.conf")
	}
	if _, err := os.Stat(masternodeConfPath); err == nil { // if masternode.conf File exists , backup

		if yes, _ := AskUserToContinue(ctx, fmt.Sprintf("Previous masternode.conf found at - %s. "+
			"Do you want to back it up and continue? Y/N", masternodeConfPath)); !yes {

			log.WithContext(ctx).WithError(err).Error("masternode.conf already exists - exiting")
			return "", fmt.Errorf("masternode.conf already exists - %s", masternodeConfPath)
		}

		oldFileName := masternodeConfPath
		currentTime := time.Now()
		backupFileName := fmt.Sprintf(masternodeConfPathBackup, currentTime.Format("2021-01-01-23-59-59"))
		if err := os.Rename(oldFileName, backupFileName); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to rename %s to %s", oldFileName, backupFileName)
			return "", err
		}
		if _, err := os.Stat(oldFileName); err == nil { // delete after back up if still exist
			if err = os.Remove(oldFileName); err != nil {
				log.WithContext(ctx).WithError(err).Errorf("Failed to remove %s", oldFileName)
				return "", err
			}
		}
	}

	return masternodeConfPath, nil
}

func updateMasternodeConfFile(ctx context.Context, confData map[string]interface{}, config *configs.Config) error {
	workDirPath := config.WorkingDir
	var masternodeConfPath string

	if config.Network == constants.NetworkTestnet {
		masternodeConfPath = filepath.Join(workDirPath, "testnet3", "masternode.conf")
	} else {
		masternodeConfPath = filepath.Join(workDirPath, "masternode.conf")
	}

	// Read ConfData from masternode.conf
	confFile, err := ioutil.ReadFile(masternodeConfPath)
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to read existing masternode.conf file - %s", masternodeConfPath)
		return err
	}

	var conf map[string]interface{}

	err = json.Unmarshal(confFile, &conf)
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Invalid updated masternode.conf file - %s", masternodeConfPath)
		return err
	}

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
		log.WithContext(ctx).WithError(err).Error("Invalid masternode.conf data")
		return err
	}

	if ioutil.WriteFile(masternodeConfPath, updatedConf, 0644) != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to write masternode.conf")
		return err
	}

	return nil
}

func getMasternodeConfData(ctx context.Context, config *configs.Config, mnName string) (privKey string,
	extAddr string, extPort string, err error) {

	var masternodeConfPath string

	if config.Network == constants.NetworkTestnet {
		masternodeConfPath = filepath.Join(config.WorkingDir, "testnet3", "masternode.conf")
	} else {
		masternodeConfPath = filepath.Join(config.WorkingDir, "masternode.conf")
	}

	// Read ConfData from masternode.conf
	confFile, err := ioutil.ReadFile(masternodeConfPath)
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to read masternode.conf at %s", masternodeConfPath)
		return "", "", "", err
	}

	var conf map[string]interface{}
	if err := json.Unmarshal([]byte(confFile), &conf); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to parse masternode.conf json %s", confFile)
		return "", "", "", err
	}

	mnNode, ok := conf[mnName]
	if !ok {
		err := errors.Errorf("masternode.conf doesn't have node with name - %s", mnName)
		log.WithContext(ctx).WithError(err).Errorf("Failed to parse masternode.conf json %s", confFile)
		return "", "", "", err
	}

	confData := mnNode.(map[string]interface{})
	privKey = confData["mnPrivKey"].(string)
	extAddrPort := strings.Split(confData["mnAddress"].(string), ":")
	extAddr = extAddrPort[0] // get Ext IP and Port
	extPort = extAddrPort[1] // get Ext IP and Port

	return privKey, extAddr, extPort, nil
}

///// Masternode specific
func runStartAliasMasternode(ctx context.Context, config *configs.Config, masternodeName string) (err error) {
	var output string
	if output, err = RunPastelCLI(ctx, config, "masternode", "start-alias", masternodeName); err != nil {
		return err
	}
	var aliasStatus map[string]interface{}

	if err = json.Unmarshal([]byte(output), &aliasStatus); err != nil {
		return err
	}

	if aliasStatus["result"] == "failed" {
		err = fmt.Errorf("masternode start alias failed")
		log.WithContext(ctx).WithError(err).Error(aliasStatus["errorMessage"])
		return err
	}

	log.WithContext(ctx).Infof("masternode alias status = %s\n", output)
	return nil
}

///// supernode.yml hlpers
func createOrUpdateSuperNodeConfig(ctx context.Context, config *configs.Config) error {

	supernodeConfigPath := config.Configurer.GetSuperNodeConfFile(config.WorkingDir)
	log.WithContext(ctx).Infof("Updating supernode config - %s", supernodeConfigPath)

	if _, err := os.Stat(supernodeConfigPath); os.IsNotExist(err) {
		// create new
		if err = utils.CreateFile(ctx, supernodeConfigPath, true); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to create new supernode.yml file at - %s", supernodeConfigPath)
			return err
		}

		portList := GetSNPortList(config)

		snTempDirPath := filepath.Join(config.WorkingDir, constants.TempDir)
		rqWorkDirPath := filepath.Join(config.WorkingDir, constants.RQServiceDir)
		p2pDataPath := filepath.Join(config.WorkingDir, constants.P2PDataDir)
		mdlDataPath := filepath.Join(config.WorkingDir, constants.MDLDataDir)

		toolConfig, err := utils.GetServiceConfig(constants.SuperNode, configs.SupernodeDefaultConfig, &configs.SuperNodeConfig{
			LogLevel:      constants.SuperNodeDefaultLogLevel,
			LogFilePath:   config.Configurer.GetSuperNodeLogFile(config.WorkingDir),
			SNTempDir:     snTempDirPath,
			SNWorkDir:     config.WorkingDir,
			RQDir:         rqWorkDirPath,
			DDDir:         filepath.Join(config.Configurer.GetHomeDir(), constants.DupeDetectionServiceDir),
			SuperNodePort: portList[constants.SNPort],
			P2PPort:       portList[constants.P2PPort],
			P2PDataDir:    p2pDataPath,
			MDLPort:       portList[constants.MDLPort],
			RAFTPort:      portList[constants.RAFTPort],
			MDLDataDir:    mdlDataPath,
			RaptorqPort:   constants.RRServiceDefaultPort,
		})
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to get supernode config")
			return err
		}
		if err = utils.WriteFile(supernodeConfigPath, toolConfig); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to update new supernode.yml file at - %s", supernodeConfigPath)
			return err
		}

	} else if err == nil {
		//update existing
		var snConfFile []byte
		snConfFile, err = ioutil.ReadFile(supernodeConfigPath)
		if err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to open existing supernode.yml file at - %s", supernodeConfigPath)
			return err
		}
		snConf := make(map[string]interface{})
		if err = yaml.Unmarshal(snConfFile, &snConf); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to parse existing supernode.yml file at - %s", supernodeConfigPath)
			return err
		}

		node := snConf["node"].(map[interface{}]interface{})

		node["pastel_id"] = flagMasterNodePastelID
		node["pass_phrase"] = flagMasterNodePassPhrase

		var snConfFileUpdated []byte
		if snConfFileUpdated, err = yaml.Marshal(&snConf); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to unparse yml for supernode.yml file at - %s", supernodeConfigPath)
			return err
		}
		if ioutil.WriteFile(supernodeConfigPath, snConfFileUpdated, 0644) != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to update supernode.yml file at - %s", supernodeConfigPath)
			return err
		}
	} else {
		log.WithContext(ctx).WithError(err).Errorf("Failed to update or create supernode.yml file at - %s", supernodeConfigPath)
		return err
	}
	log.WithContext(ctx).Info("Supernode config updated")
	return nil
}

/*func getRemoteInfo(config *configs.Config, client *utils.Client) (remoteWorkDirPath []byte, remotePastelExecPath []byte, remoteOsType []byte, err error) {

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
*/
