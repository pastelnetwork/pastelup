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

var (
	// errSubCommandRequired             = fmt.Errorf("subcommand is required")
	errNotFoundPastelCli              = fmt.Errorf("cannot find pastel-cli on SSH server")
	errNotFoundRemotePastelUtilityDir = fmt.Errorf("cannot find remote pastel-utility dir")
	errNotStartPasteld                = fmt.Errorf("pasteld was not started")
	errMasternodeStartAlias           = fmt.Errorf("masternode start alias failed")
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
	flagMasterNodeIND        string
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
		cli.NewFlag("ind", &flagMasterNodeIND).
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
			SetUsage(green("Optional, Location of working directory on the remote computer (default: $HOME/pastel-utility)")),
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
	startSuperNodeSubCommand := setupStartSubCommand(config, superNodeStart, runMasterNodeOnHotHotSubCommand)
	startSuperNodeCOldHotSubCommand := setupStartSubCommand(config, superNodeColdHotStart, runMasterNodeOnColdHotSubCommand)

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

func runPastelNode(ctx context.Context, config *configs.Config, reindex bool, extIP string, mnPrivKey string) (err error) {
	var pastelDPath string

	if pastelDPath, err = checkPastelFilePath(ctx, config.PastelExecDir, constants.PasteldName[utils.GetOS()]); err != nil {
		log.WithContext(ctx).WithError(err).Error("Could not find pasteld")
		return err
	}
	if _, err = checkPastelFilePath(ctx, config.PastelExecDir, constants.PastelConfName); err != nil {
		log.WithContext(ctx).WithError(err).Error("Could not find pastel.conf")
		return err
	}
	if err = checkZksnarkParams(ctx, config); err != nil {
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

	if !checkPastelDRunning(ctx, config) {
		log.WithContext(ctx).WithError(err).Error("pasteld didn't start")
		return errNotStartPasteld
	}

	return nil
}

func checkZksnarkParams(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Checking pastel param files...")

	zksnarkPath := config.Configurer.DefaultZksnarkDir()

	for _, zksnarkParamsName := range configs.ZksnarkParamsNames {
		zksnarkParamsPath := filepath.Join(zksnarkPath, zksnarkParamsName)

		log.WithContext(ctx).Infof("Checking pastel param file : %s", zksnarkParamsPath)
		checkSum, err := utils.GetChecksum(ctx, zksnarkParamsPath)
		if err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to check param file : %s", zksnarkParamsPath)
			return err
		} else if checkSum != constants.PastelParamsCheckSums[zksnarkParamsName] {
			log.WithContext(ctx).Errorf("Wrong checksum of the pastel param file: %s", zksnarkParamsPath)
			return errors.Errorf("Wrong checksum of the pastel param file: %s", zksnarkParamsPath)
		}
	}

	return nil
}

func checkPastelDRunning(ctx context.Context, config *configs.Config) (ret bool) {
	var failCnt = 0
	var err error

	log.WithContext(ctx).Info("Waiting the pasteld to be started...")

	for {
		if _, err = RunPastelCLI(ctx, config, "getinfo"); err != nil {
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

	go RunCMDWithEnvVariable("python3",
		"DUPEDETECTIONCONFIGPATH",
		ddConfigFilePath,
		execPath)
	time.Sleep(10000 * time.Millisecond)

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

func runPastelService(ctx context.Context, config *configs.Config, toolType constants.ToolType, toolFileName string, args ...string) (err error) {

	log.WithContext(ctx).Infof("Starting %s", toolType)

	var execPath string
	if execPath, err = checkPastelFilePath(ctx, config.PastelExecDir, toolFileName); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Could not find %s", toolType)
		return err
	}

	go RunCMD(execPath, args...)
	time.Sleep(10000 * time.Millisecond)

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

// Sub Command
func runMasterNodeOnHotHotSubCommand(ctx context.Context, config *configs.Config) error {

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
	}

	// Get conf data from masternode.conf File
	privKey, extIP, _ /*extPort*/, pastelID, err := getMasternodeConfData(ctx, config, flagMasterNodeName)
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
	log.WithContext(ctx).Infof("Starting pasteld as masternode: nodeName: %s; mnPrivKey: %s; pastelID: %s;", flagMasterNodeName, privKey, pastelID)
	if err := runPastelNode(ctx, config, true, flagNodeExtIP, privKey); err != nil { //in masternode mode pasteld MUST be started with reindex flag
		log.WithContext(ctx).WithError(err).Error("pasteld failed to start")
		return err
	}

	// *************  4. Wait for blockchain and masternodes sync  *************
	if err := checkMasterNodeSync(ctx, config); err != nil {
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
	if err := updateSuperNodeConfig(ctx, config, pastelID); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to update supernode.yml")
		return err
	}

	if err := runSuperNodeService(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to start supernode service")
		return err
	}

	return nil
}

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

	// –create | –update - Optional, if specified, will create or update Masternode record in the masternode.conf. Following are the parameters of --create and/or --update:
	if flagMasterNodeIsCreate || flagMasterNodeIsUpdate {
		if len(flagMasterNodeTxID) == 0 {
			err := fmt.Errorf("required if create or update specified: --txid, transaction id of 5M collateral MN payment")
			log.WithContext(ctx).WithError(err).Error("Missing parameter --txid")
			return err
		}

		if len(flagMasterNodeIND) == 0 {
			err := fmt.Errorf("required if create or update specified: --ind, output index in the transaction of 5M collateral MN payment")
			log.WithContext(ctx).WithError(err).Error("Missing parameter --ind")
			return err
		}

		if len(flagMasterNodePassPhrase) == 0 {
			err := fmt.Errorf("required parameter if --create or --update specified: --passphrase <passphrase to pastelid private key>")
			log.WithContext(ctx).WithError(err).Error("Missing parameter --passphrase")
			return err
		}
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

func prepareMasterNodeParameters(ctx context.Context, config *configs.Config) (err error) {

	// this function must only be called when --create or --update
	if !flagMasterNodeIsCreate && !flagMasterNodeIsUpdate {
		return nil
	}

	bReIndex := true // if masternode.conf exist pasteld MUST be start with reindex flag
	if flagMasterNodeIsCreate {
		if _, _, err = backupConfFile(config); err != nil {
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
	if err = checkMasterNodeSync(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("pasteld failed to synchronize, add some peers and try again")
		return err
	}

	// Search collateral transaction
	var outputs string
	if outputs, err = RunPastelCLI(ctx, config, "masternode", "outputs"); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to get masternode outputs from pasteld")
		return err
	}
	var mnOutputs map[string]interface{}
	if err := json.Unmarshal([]byte(outputs), &mnOutputs); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to parse masternode outputs json")
		return err
	}
	if len(mnOutputs) == 0 ||
		mnOutputs[flagMasterNodeTxID] == nil ||
		mnOutputs[flagMasterNodeTxID] != flagMasterNodeIND {

		err = errors.Errorf("Cannot find masternode outputs = %s:%s", flagMasterNodeTxID, flagMasterNodeIND)
		log.WithContext(ctx).WithError(err).Error("Try again after some time")
		return err
	}

	// if receives PSL go to next step
	log.WithContext(ctx).Infof("masternode outputs = %s, %s", flagMasterNodeTxID, flagMasterNodeIND)

	// Create new MN private key
	if len(flagMasterNodePrivateKey) == 0 {
		var mnPrivKey string
		if mnPrivKey, err = RunPastelCLI(ctx, config, "masternode", "genkey"); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to generate new masternode private key")
			return err
		}
		flagMasterNodePrivateKey = strings.TrimSuffix(mnPrivKey, "\n")
	}
	log.WithContext(ctx).Infof("masternode private key = %s", flagMasterNodePrivateKey)

	// Create new pastelid
	if len(flagMasterNodePastelID) == 0 {

		log.WithContext(ctx).Info("Masternode PastelID is empty - will create new one")

		if len(flagMasterNodePassPhrase) == 0 { //check one more time just because
			err := fmt.Errorf("required parameter if --create or --update specified: --passphrase <passphrase to pastelid private key>")
			log.WithContext(ctx).WithError(err).Error("Missing parameter --passphrase")
			return err
		}

		var pastelid string
		if pastelid, err = RunPastelCLI(ctx, config, "pastelid", "newkey", flagMasterNodePassPhrase); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to generate new pastelid key")
			return err
		}

		var pastelidSt structure.RPCPastelID
		if err = json.Unmarshal([]byte(pastelid), &pastelidSt); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to parse pastelid json")
			return err
		}
		flagMasterNodePastelID = pastelidSt.Pastelid
	}
	log.WithContext(ctx).Infof("Masternode pastelid = %s", flagMasterNodePastelID)

	return stopPastelDAndWait(ctx, config)
}

func createOrUpdateMasternodeConf(ctx context.Context, config *configs.Config) (err error) {

	// this function must only be called when --create or --update
	if !flagMasterNodeIsCreate && !flagMasterNodeIsUpdate {
		return nil
	}

	confData := map[string]interface{}{
		flagMasterNodeName: map[string]string{
			"mnAddress":  flagNodeExtIP + ":" + fmt.Sprintf("%d", flagMasterNodePort),
			"mnPrivKey":  flagMasterNodePrivateKey,
			"txid":       flagMasterNodeTxID,
			"outIndex":   flagMasterNodeIND,
			"extAddress": flagNodeExtIP + ":" + fmt.Sprintf("%d", flagMasterNodeRPCPort),
			"extP2P":     flagMasterNodeP2PIP + ":" + fmt.Sprintf("%d", flagMasterNodeP2PPort),
			"extCfg":     "",
			"extKey":     flagMasterNodePastelID,
		},
	}

	if flagMasterNodeIsCreate {
		data, _ := json.Marshal(confData)
		// Create masternode.conf file
		if err = createConfFile(data, config); err != nil {
			return err
		}
	} else if flagMasterNodeIsUpdate {
		// Create masternode.conf file
		if _, err = updateMasternodeConfFile(confData, config); err != nil {
			return err
		}
	}

	data, _ := json.Marshal(confData)
	log.WithContext(ctx).Infof("masternode.conf = %s", string(data))

	return nil
}

func createConfFile(confData []byte, config *configs.Config) (err error) {

	var masternodeConfPath string
	if masternodeConfPath, _, err = backupConfFile(config); err != nil {
		return err
	}

	confFile, err := os.Create(masternodeConfPath)
	confFile.Write(confData)
	if err != nil {
		return err
	}
	defer confFile.Close()

	return nil
}

func backupConfFile(config *configs.Config) (masternodeConfPath string, masternodeConfPathBackup string, err error) {
	workDirPath := config.WorkingDir

	if config.Network == constants.NetworkTestnet {
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
			return "", "", err
		}
		if _, err := os.Stat(masternodeConfPath); err == nil { // delete after back up if still exist
			if err = os.Remove(masternodeConfPath); err != nil {
				return "", "", err
			}
		}
	}

	return masternodeConfPath, masternodeConfPathBackup, nil
}

func updateMasternodeConfFile(confData map[string]interface{}, config *configs.Config) (result bool, err error) {
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

func stopPastelDAndWait(ctx context.Context, config *configs.Config) (err error) {
	log.WithContext(ctx).Info("Stopping pasteld...")
	if _, err = RunPastelCLI(ctx, config, "stop"); err != nil {
		return err
	}

	time.Sleep(10000 * time.Millisecond)
	log.WithContext(ctx).Info("Stopped pasteld")
	return nil
}

func checkMasterNodeSync(ctx context.Context, config *configs.Config) (err error) {
	for {

		mnstatus, err := GetMNSyncInfo(ctx, config)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("master node mnsync status call has failed")
			return err
		}

		if mnstatus.AssetName == "Initial" {
			if _, err = RunPastelCLI(ctx, config, "mnsync", "reset"); err != nil {
				log.WithContext(ctx).WithError(err).Error("master node reset has failed")
				return err
			}
			time.Sleep(10000 * time.Millisecond)
		}
		if mnstatus.IsSynced {
			log.WithContext(ctx).Info("master node was synced!")
			break
		}
		log.WithContext(ctx).Info("Waiting for sync...")

		getinfo, err := GetPastelInfo(ctx, config)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("master node getinfo call has failed")
			return err
		}
		log.WithContext(ctx).Infof("Loading blocks - block #%d; Node has %d connection", getinfo.Blocks, getinfo.Connections)

		time.Sleep(10000 * time.Millisecond)
	}

	return nil
}

func getMasternodeConfData(ctx context.Context, config *configs.Config, mnName string) (privKey string,
	extAddr string, extPort string, extkey string, err error) {

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
		return "", "", "", "", err
	}

	var conf map[string]interface{}
	if err := json.Unmarshal([]byte(confFile), &conf); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to parse masternode.conf json %s", confFile)
		return "", "", "", "", err
	}

	mnNode, ok := conf[mnName]
	if !ok {
		err := errors.Errorf("masternode.conf doesn't have node with name - %s", mnName)
		log.WithContext(ctx).WithError(err).Errorf("Failed to parse masternode.conf json %s", confFile)
		return "", "", "", "", err
	}

	confData := mnNode.(map[string]interface{})
	privKey = confData["mnPrivKey"].(string)
	extAddrPort := strings.Split(confData["mnAddress"].(string), ":")
	extAddr = extAddrPort[0] // get Ext IP and Port
	extPort = extAddrPort[1] // get Ext IP and Port
	extKey := confData["extKey"].(string)

	return privKey, extAddr, extPort, extKey, nil
}

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
		log.WithContext(ctx).Error(aliasStatus["errorMessage"])
		return errMasternodeStartAlias
	}

	log.WithContext(ctx).Infof("masternode alias status = %s\n", output)
	return nil
}

func updateSuperNodeConfig(ctx context.Context, config *configs.Config, pastelID string) error {

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

		node["pastel_id"] = pastelID
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

//FIXME: ColdNot
func runMasterNodeOnColdHotSubCommand(ctx context.Context, config *configs.Config) error {
	//var pastelid string
	var err error

	if len(config.RemotePastelUtilityDir) == 0 {
		return errNotFoundRemotePastelUtilityDir
	}

	log.WithContext(ctx).Info("Checking parameters...")
	if err := checkStartMasterNodeParams(ctx, config, true); err != nil {
		log.WithContext(ctx).WithError(err).Error("Checking parameters failed")
		return err
	}
	log.WithContext(ctx).Info("Finished checking parameters!")

	log.WithContext(ctx).Info("Checking pastel config...")
	if err := ParsePastelConf(ctx, config); err != nil {
		log.WithContext(ctx).Error("pastel.conf was not correct!")
		return err
	}
	log.WithContext(ctx).Info("Finished checking pastel config!")

	if flagMasterNodeIsCreate || flagMasterNodeIsUpdate {
		log.WithContext(ctx).Info("Prepare mastenode parameters")
		if err := prepareMasterNodeParameters(ctx, config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to validate and prepare masternode parameters")
			return err
		}
		if err := createOrUpdateMasternodeConf(ctx, config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to create or update masternode.conf")
			return err
		}
	}

	// ***************  4. Execute following commands over SSH on the remote node (using ssh-ip and ssh-port)  ***************
	username, password, _ := utils.Credentials("", true)

	if err = remoteHotNodeCtrl(ctx, config, username, password); err != nil {
		log.WithContext(ctx).Error(fmt.Sprintf("%s\n", err))
		return err
	}
	log.WithContext(ctx).Info("The hot wallet node has been successfully launched!")
	// ***************  5. Enable Masternode  ***************
	// Get conf data from masternode.conf File
	var extIP string
	if _, extIP, _, _, err = getMasternodeConfData(ctx, config, flagMasterNodeName); err != nil {
		return err
	}

	// Start Node as Masternode
	if err := runPastelNode(ctx, config, flagReIndex, extIP, ""); err != nil {
		log.WithContext(ctx).WithError(err).Error("pasteld failed to start")
		return err
	}

	if err = checkMasterNodeSync(ctx, config); err != nil {
		return err
	}

	if flagMasterNodeIsActivate {
		if err = runStartAliasMasternode(ctx, config, flagMasterNodeName); err != nil {
			return err
		}
	}

	// ***************  6. Stop Cold Node  ***************
	if _, err = RunPastelCLI(ctx, config, "stop"); err != nil {
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

	err = runSuperNodeRemote(ctx, config, client /*, extIP, pastelid*/)
	if err != nil {
		return err
	}

	return nil
}

func remoteHotNodeCtrl(ctx context.Context, config *configs.Config, username string, password string) error {
	var pastelCliPath, testnetOption string
	log.WithContext(ctx).Infof("Connecting to SSH Hot node wallet -> %s:%d...", flagMasterNodeSSHIP, flagMasterNodeSSHPort)
	client, err := utils.DialWithPasswd(fmt.Sprintf("%s:%d", flagMasterNodeSSHIP, flagMasterNodeSSHPort), username, password)
	if err != nil {
		return err
	}
	defer client.Close()

	if config.Network == constants.NetworkTestnet {
		testnetOption = " --testnet"
	}

	// Find pasteld
	pasteldPath := filepath.Join(config.RemotePastelExecDir, constants.PasteldName[utils.GetOS()])
	log.WithContext(ctx).Info("Check pasteld default path...")
	out, err := client.Cmd(fmt.Sprintf("test -e %s && echo file exists || echo file not found", pasteldPath)).Output()
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
				indexStr, err := utils.ReadStrings("Please input index of pasteld path to use")
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
					pasteldPath = pastelPaths[index]
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

	go client.Cmd(fmt.Sprintf("%s --reindex --externalip=%s --daemon%s", pasteldPath, flagNodeExtIP, testnetOption)).Run()

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

	cmdLine := fmt.Sprintf("%s --masternode --txindex=1 --reindex --masternodeprivkey=%s --externalip=%s%s --daemon", pasteldPath, flagMasterNodePrivateKey, flagNodeExtIP, testnetOption)
	log.WithContext(ctx).Infof("%s\n", cmdLine)
	go client.Cmd(cmdLine).Run()

	time.Sleep(10000 * time.Millisecond)

	if err = checkMasterNodeSyncRemote(ctx, config, client, pastelCliPath); err != nil {
		log.WithContext(ctx).Error("Remote::Master node sync failed")
		return err
	}

	return nil
}

func runPastelServiceRemote(ctx context.Context, config *configs.Config, tool constants.ToolType, client *utils.Client) (err error) {
	commandName := filepath.Base(string(tool))
	log.WithContext(ctx).Infof("Remote:::Starting %s", commandName)

	remoteWorkDirPath, remotePastelExecPath, remoteOsType, err := getRemoteInfo(config, client)
	if err != nil {
		return err
	}

	switch tool {
	case constants.RQService:
		remoteRQServiceConfigFilePath := config.Configurer.GetRQServiceConfFile(string(remoteWorkDirPath))

		remoteRQServiceConfigFilePath = strings.ReplaceAll(remoteRQServiceConfigFilePath, "\\", "/")

		pastelRqServicePath := filepath.Join(string(remotePastelExecPath), constants.PastelRQServiceExecName[constants.OSType(string(remoteOsType))])
		pastelRqServicePath = strings.ReplaceAll(pastelRqServicePath, "\\", "/")

		go client.Cmd(fmt.Sprintf("%s %s", pastelRqServicePath, fmt.Sprintf("--config-file=%s", remoteRQServiceConfigFilePath))).Run()

		time.Sleep(10000 * time.Millisecond)

	}

	log.WithContext(ctx).Infof("Remote:::The %s started succesfully!", commandName)
	return nil
}

func runSuperNodeRemote(ctx context.Context, config *configs.Config, client *utils.Client /*, extIP string, pastelid string*/) (err error) {
	log.WithContext(ctx).Info("Remote:::Starting supernode")
	log.WithContext(ctx).Debug("Remote:::Configure supernode setting")

	log.WithContext(ctx).Info("Remote:::Configuring supernode was finished")
	log.WithContext(ctx).Info("Remote:::Start supernode")

	remoteWorkDirPath, remotePastelExecPath, remoteOsType, err := getRemoteInfo(config, client)
	if err != nil {
		return err
	}

	remoteSuperNodeConfigFilePath := config.Configurer.GetSuperNodeConfFile(string(remoteWorkDirPath))

	var remoteSupernodeExecFile string

	remoteSuperNodeConfigFilePath = strings.ReplaceAll(remoteSuperNodeConfigFilePath, "\\", "/")
	remoteSupernodeExecFile = filepath.Join(string(remotePastelExecPath), constants.SuperNodeExecName[constants.OSType(string(remoteOsType))])
	remoteSupernodeExecFile = strings.ReplaceAll(remoteSupernodeExecFile, "\\", "/")

	time.Sleep(5000 * time.Millisecond)

	log.WithContext(ctx).Infof("Remote:::Start supernode command : %s", fmt.Sprintf("%s %s", remoteSupernodeExecFile, fmt.Sprintf("--config-file=%s", remoteSuperNodeConfigFilePath)))

	go client.Cmd(fmt.Sprintf("%s %s", remoteSupernodeExecFile,
		fmt.Sprintf("--config-file=%s", remoteSuperNodeConfigFilePath))).Run()

	defer client.Close()

	log.WithContext(ctx).Info("Remote:::Waiting for supernode started...")
	time.Sleep(5000 * time.Millisecond)

	log.WithContext(ctx).Info("Remote:::Supernode was started successfully")
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
