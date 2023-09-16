package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/pastelup/configs"
	"github.com/pastelnetwork/pastelup/constants"
	"github.com/pastelnetwork/pastelup/services/pastelcore"
	"github.com/pastelnetwork/pastelup/utils"
)

/*
var (

	wg sync.WaitGroup

)
*/

type startCommand uint8

const (
	nodeStart startCommand = iota
	walletStart
	superNodeStart
	ddService
	rqService
	ddImgServer
	wnService
	snService
	masterNode
	remoteStart
	bridgeService
	hermesService
)

var (
	startCmdName = map[startCommand]string{
		nodeStart:      "node",
		walletStart:    "walletnode",
		superNodeStart: "supernode",
		ddService:      "dd-service",
		rqService:      "rq-service",
		ddImgServer:    "imgserver",
		wnService:      "walletnode-service",
		snService:      "supernode-service",
		masterNode:     "masternode",
		remoteStart:    "remote",
		bridgeService:  "bridge-service",
		hermesService:  "hermes-service",
	}
	startCmdMessage = map[startCommand]string{
		nodeStart:      "Start node",
		walletStart:    "Start Walletnode",
		superNodeStart: "Start Supernode",
		ddService:      "Start Dupe Detection service only",
		rqService:      "Start RaptorQ service only",
		ddImgServer:    "Start dd image server",
		wnService:      "Start Walletnode service only",
		snService:      "Start Supernode service only",
		masterNode:     "Start only pasteld node as Masternode",
		remoteStart:    "Start on Remote host",
		bridgeService:  "Start bridge-service only",
		hermesService:  "Start hermes-service only",
	}
)

func setupStartSubCommand(config *configs.Config,
	startCommand startCommand, remote bool,
	f func(context.Context, *configs.Config) error,
) *cli.Command {
	commonFlags := []*cli.Flag{
		cli.NewFlag("ip", &config.NodeExtIP).
			SetUsage(green("Optional, WAN address of the host")),
		cli.NewFlag("reindex", &config.ReIndex).
			SetUsage(green("Optional, Start with reindex")),
	}

	var dirsFlags []*cli.Flag

	if !remote {
		dirsFlags = []*cli.Flag{
			cli.NewFlag("dir", &config.PastelExecDir).SetAliases("d").
				SetUsage(green("Optional, Location of pastel node directory")).SetValue(config.Configurer.DefaultPastelExecutableDir()),
			cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
				SetUsage(green("Optional, location of working directory")).SetValue(config.Configurer.DefaultWorkingDir()),
		}
	} else {
		dirsFlags = []*cli.Flag{
			cli.NewFlag("dir", &config.PastelExecDir).SetAliases("d").
				SetUsage(green("Optional, Location where to create pastel node directory on the remote computer (default: $HOME/pastel)")),
			cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
				SetUsage(green("Optional, Location where to create working directory on the remote computer (default: $HOME/.pastel)")),
		}
	}

	walletNodeFlags := []*cli.Flag{
		cli.NewFlag("development-mode", &config.DevMode),
	}

	superNodeStartFlags := []*cli.Flag{
		cli.NewFlag("name", &config.MasterNodeName).
			SetUsage(red("name of the Masternode to start")),
		cli.NewFlag("activate", &config.ActivateMasterNode).
			SetUsage(green("Optional, if specified, will try to enable node as Masternode (start-alias).")),
	}

	masternodeFlags := []*cli.Flag{
		cli.NewFlag("name", &config.MasterNodeName).
			SetUsage(red("name of the Masternode to start")),
	}

	remoteStartFlags := []*cli.Flag{
		cli.NewFlag("ssh-ip", &config.RemoteIP).
			SetUsage(red("Required, SSH address of the remote node")),
		cli.NewFlag("ssh-port", &config.RemotePort).
			SetUsage(green("Optional, SSH port of the remote node")).SetValue(22),
		cli.NewFlag("ssh-user", &config.RemoteUser).
			SetUsage(yellow("Optional, SSH user")),
		cli.NewFlag("ssh-key", &config.RemoteSSHKey).
			SetUsage(yellow("Optional, Path to SSH private key")),
		cli.NewFlag("inventory", &config.InventoryFile).
			SetUsage(red("Optional, Path to the file with configuration of the remote hosts")),
	}

	var commandName, commandMessage string
	if !remote {
		commandName = startCmdName[startCommand]
		commandMessage = startCmdMessage[startCommand]
	} else {
		commandName = startCmdName[remoteStart]
		commandMessage = startCmdMessage[remoteStart]
	}

	commandFlags := append(dirsFlags, commonFlags[:]...)
	if startCommand == walletStart ||
		startCommand == wnService {
		commandFlags = append(commandFlags, walletNodeFlags[:]...)
	} else if startCommand == superNodeStart {
		commandFlags = append(commandFlags, superNodeStartFlags[:]...)
	} else if startCommand == masterNode {
		commandFlags = append(commandFlags, masternodeFlags[:]...)
	}
	if remote {
		commandFlags = append(commandFlags, remoteStartFlags[:]...)
	}

	subCommand := cli.NewCommand(commandName)
	subCommand.SetUsage(cyan(commandMessage))
	subCommand.AddFlags(commandFlags...)
	addLogFlags(subCommand, config)

	if f != nil {
		subCommand.SetActionFunc(func(ctx context.Context, args []string) error {
			ctx, err := configureLogging(ctx, commandMessage, config)
			if err != nil {
				return err
			}

			// Register interrupt handler
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt)
			go func() {
				for {
					<-sigCh

					yes, _ := AskUserToContinue(ctx, "Interrupt signal received, do you want to cancel this process? Y/N")
					if yes {
						log.WithContext(ctx).Info("Gracefully shutting down...")
						cancel()
						os.Exit(0)
					}
				}
			}()
			if !remote {
				if err = ParsePastelConf(ctx, config); err != nil {
					return err
				}
			}
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

func setupStartCommand(config *configs.Config) *cli.Command {

	startNodeSubCommand := setupStartSubCommand(config, nodeStart, false, runStartNodeSubCommand)
	startWalletNodeSubCommand := setupStartSubCommand(config, walletStart, false, runStartWalletNodeSubCommand)
	startSuperNodeSubCommand := setupStartSubCommand(config, superNodeStart, false, runStartSuperNodeSubCommand)

	startRQServiceCommand := setupStartSubCommand(config, rqService, false, runRQService)
	startDDServiceCommand := setupStartSubCommand(config, ddService, false, runDDService)
	startWNServiceCommand := setupStartSubCommand(config, wnService, false, runWalletNodeService)
	startSNServiceCommand := setupStartSubCommand(config, snService, false, runSuperNodeService)
	startMasternodeCommand := setupStartSubCommand(config, masterNode, false, runStartMasternodeService)
	startHermesServiceCommand := setupStartSubCommand(config, hermesService, false, runHermesService)
	startBridgeServiceCommand := setupStartSubCommand(config, bridgeService, false, runBridgeService)
	startDDImgServerCommand := setupStartSubCommand(config, ddImgServer, false, runDDImgServer)

	startSuperNodeRemoteSubCommand := setupStartSubCommand(config, superNodeStart, true, runRemoteSuperNodeStartSubCommand)
	startSuperNodeSubCommand.AddSubcommands(startSuperNodeRemoteSubCommand)

	startWalletNodeRemoteSubCommand := setupStartSubCommand(config, superNodeStart, true, runRemoteWalletNodeStartSubCommand)
	startWalletNodeSubCommand.AddSubcommands(startWalletNodeRemoteSubCommand)

	startNodeRemoteSubCommand := setupStartSubCommand(config, nodeStart, true, runRemoteNodeStartSubCommand)
	startNodeSubCommand.AddSubcommands(startNodeRemoteSubCommand)

	startRQServiceRemoteCommand := setupStartSubCommand(config, rqService, true, runRemoteRQServiceStartSubCommand)
	startRQServiceCommand.AddSubcommands(startRQServiceRemoteCommand)

	startHermesServiceRemoteCommand := setupStartSubCommand(config, hermesService, true, runRemoteHermesServiceStartSubCommand)
	startHermesServiceCommand.AddSubcommands(startHermesServiceRemoteCommand)

	startBridgeServiceRemoteCommand := setupStartSubCommand(config, bridgeService, true, runRemoteBridgeServiceStartSubCommand)
	startBridgeServiceCommand.AddSubcommands(startBridgeServiceRemoteCommand)

	startDDServiceRemoteCommand := setupStartSubCommand(config, ddService, true, runRemoteDDServiceStartSubCommand)
	startDDServiceCommand.AddSubcommands(startDDServiceRemoteCommand)

	startWNServiceRemoteCommand := setupStartSubCommand(config, wnService, true, runRemoteWNServiceStartSubCommand)
	startWNServiceCommand.AddSubcommands(startWNServiceRemoteCommand)

	startSNServiceRemoteCommand := setupStartSubCommand(config, snService, true, runRemoteSNServiceStartSubCommand)
	startSNServiceCommand.AddSubcommands(startSNServiceRemoteCommand)

	startMasternodeRemoteCommand := setupStartSubCommand(config, masterNode, true, runRemoteSNServiceStartSubCommand)
	startMasternodeCommand.AddSubcommands(startMasternodeRemoteCommand)

	startDDImgServerRemoteCommand := setupStartSubCommand(config, ddImgServer, true, runRemoteDDImgServerSubCommand)
	startDDImgServerCommand.AddSubcommands(startDDImgServerRemoteCommand)

	startCommand := cli.NewCommand("start")
	startCommand.SetUsage(blue("Performs start of the system for both WalletNode and SuperNodes"))
	startCommand.AddSubcommands(startNodeSubCommand)
	startCommand.AddSubcommands(startWalletNodeSubCommand)
	startCommand.AddSubcommands(startSuperNodeSubCommand)

	startCommand.AddSubcommands(startRQServiceCommand)
	startCommand.AddSubcommands(startDDServiceCommand)
	startCommand.AddSubcommands(startDDImgServerCommand)
	startCommand.AddSubcommands(startWNServiceCommand)
	startCommand.AddSubcommands(startSNServiceCommand)
	startCommand.AddSubcommands(startMasternodeCommand)
	startCommand.AddSubcommands(startHermesServiceCommand)
	startCommand.AddSubcommands(startBridgeServiceCommand)

	return startCommand

}

///// Top level start commands

// Sub Command
func runStartNodeSubCommand(ctx context.Context, config *configs.Config) error {
	if err := runPastelNode(ctx, config, false, config.ReIndex, config.NodeExtIP, ""); err != nil {
		log.WithContext(ctx).WithError(err).Error("pasteld failed to start")
		return err
	}
	return nil
}

// Sub Command
func runStartWalletNodeSubCommand(ctx context.Context, config *configs.Config) error {
	// *************  1. Start pastel node  *************
	if err := runPastelNode(ctx, config, config.TxIndex == 1, config.ReIndex, config.NodeExtIP, ""); err != nil {
		log.WithContext(ctx).WithError(err).Error("pasteld failed to start")
		return err
	}

	// *************  2. Start rq-servce    *************
	if err := runRQService(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("rqservice failed to start")
		return err
	}

	// *************  3. Check & Start bridge node  *************
	walletConf := config.Configurer.GetWalletNodeConfFile(config.WorkingDir)
	enable, err := checkBridgeEnabled(ctx, walletConf)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("unable to check bridge enabled, skipping start bridge service")
	} else if enable {
		if err := runBridgeService(ctx, config); err != nil {
			log.WithContext(ctx).WithError(err).Error("bridge failed to start")
			return err
		}
	}

	// *************  4. Start wallet node  *************
	if err := runWalletNodeService(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("walletnode failed to start")
		return err
	}

	return nil
}

// Sub Command
func runStartSuperNodeSubCommand(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Starting supernode")
	if err := runStartSuperNode(ctx, config, false); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to start supernode")
		return err
	}
	log.WithContext(ctx).Info("Supernode started successfully")

	return nil
}

func runStartSuperNode(ctx context.Context, config *configs.Config, justInit bool) error {
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

	pastelDIsRunning := false
	if CheckProcessRunning(constants.PastelD) {
		log.WithContext(ctx).Infof("pasteld is already running")
		if yes, _ := AskUserToContinue(ctx,
			"Do you want to stop it and continue? Y/N"); !yes {
			log.WithContext(ctx).Warn("Exiting...")
			return fmt.Errorf("user terminated installation")
		}
		pastelDIsRunning = true
	}

	if config.CreateNewMasterNodeConf || config.AddToMasterNodeConf {
		log.WithContext(ctx).Info("Prepare masternode parameters")
		if err := prepareMasterNodeParameters(ctx, config, !pastelDIsRunning); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to validate and prepare masternode parameters")
			return err
		}
		log.WithContext(ctx).Infof("CONFIG: %+v", config)
		if err := createOrUpdateMasternodeConf(ctx, config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to create or update masternode.conf")
			return err
		}
		if err := createOrUpdateSuperNodeConfig(ctx, config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to update supernode.yml")
			return err
		}
		config.ReIndex = false // prepareMasterNodeParameters already run paslted with txindex and reindex
		// so no need to use reindex again
	}

	if pastelDIsRunning {
		if err := StopPastelDAndWait(ctx, config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Cannot stop pasteld")
			return err
		}
	}

	// *************  3. Start Node as Masternode  *************
	if err := runStartMasternode(ctx, config); err != nil { //in masternode mode pasteld MUST be started with reindex flag
		return err
	}

	// *************  4. Wait for blockchain and masternodes sync  *************
	if _, err := CheckMasterNodeSync(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("pasteld failed to synchronize, add some peers and try again")
		return err
	}

	// *************  5. Enable Masternode  ***************
	if config.ActivateMasterNode {
		log.WithContext(ctx).Infof("Starting MN alias - %s", config.MasterNodeName)
		if err := runStartAliasMasternode(ctx, config, config.MasterNodeName); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to start alias - %s", config.MasterNodeName)
			return err
		}
	}

	if justInit {
		log.WithContext(ctx).Info("Init exits")
		return nil
	}

	// *************  6. Start rq-service    *************
	if err := runRQService(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("rqservice failed to start")
		return err
	}

	// *************  7. Start dd-service    *************
	if err := runDDService(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("ddservice failed to start")
		return err
	}

	// *************  8. Start supernode (& hermes service)  **************
	if err := runSuperNodeService(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to start supernode service")
		return err
	}

	return nil
}

func runRemoteNodeStartSubCommand(ctx context.Context, config *configs.Config) error {
	return runRemoteStart(ctx, config, "node")
}
func runRemoteSuperNodeStartSubCommand(ctx context.Context, config *configs.Config) error {
	return runRemoteStart(ctx, config, "supernode")
}
func runRemoteWalletNodeStartSubCommand(ctx context.Context, config *configs.Config) error {
	return runRemoteStart(ctx, config, "walletnode")
}
func runRemoteRQServiceStartSubCommand(ctx context.Context, config *configs.Config) error {
	return runRemoteStart(ctx, config, "rq-service")
}
func runRemoteDDServiceStartSubCommand(ctx context.Context, config *configs.Config) error {
	return runRemoteStart(ctx, config, "dd-service")
}
func runRemoteWNServiceStartSubCommand(ctx context.Context, config *configs.Config) error {
	return runRemoteStart(ctx, config, "walletnode-service")
}
func runRemoteSNServiceStartSubCommand(ctx context.Context, config *configs.Config) error {
	return runRemoteStart(ctx, config, "supernode-service")
}
func runRemoteHermesServiceStartSubCommand(ctx context.Context, config *configs.Config) error {
	return runRemoteStart(ctx, config, "hermes-service")
}
func runRemoteBridgeServiceStartSubCommand(ctx context.Context, config *configs.Config) error {
	return runRemoteStart(ctx, config, "bridge-service")
}
func runRemoteDDImgServerSubCommand(ctx context.Context, config *configs.Config) error {
	return runRemoteStart(ctx, config, "imgserver")
}

func runRemoteStart(ctx context.Context, config *configs.Config, tool string) error {
	log.WithContext(ctx).Infof("Starting remote %s", tool)

	// Start remote node
	startOptions := tool

	if len(config.MasterNodeName) > 0 {
		startOptions = fmt.Sprintf("%s --name=%s", startOptions, config.MasterNodeName)
	}

	if config.ActivateMasterNode {
		startOptions = fmt.Sprintf("%s --activate", startOptions)
	}

	if len(config.NodeExtIP) > 0 {
		startOptions = fmt.Sprintf("%s --ip=%s", startOptions, config.NodeExtIP)
	}
	if config.ReIndex {
		startOptions = fmt.Sprintf("%s --reindex", startOptions)
	}
	if config.DevMode {
		startOptions = fmt.Sprintf("%s --development-mode", startOptions)
	}
	if len(config.PastelExecDir) > 0 {
		startOptions = fmt.Sprintf("%s --dir=%s", startOptions, config.PastelExecDir)
	}
	if len(config.WorkingDir) > 0 {
		startOptions = fmt.Sprintf("%s --work-dir=%s", startOptions, config.WorkingDir)
	}

	startSuperNodeCmd := fmt.Sprintf("%s start %s", constants.RemotePastelupPath, startOptions)
	if _, err := executeRemoteCommandsWithInventory(ctx, config, []string{startSuperNodeCmd}, false, false); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to start %s on remote host", tool)
	}

	log.WithContext(ctx).Infof("Remote %s started successfully", tool)
	return nil
}

func runStartMasternodeService(ctx context.Context, config *configs.Config) error {
	// *************  1. Parse pastel config parameters  *************
	log.WithContext(ctx).Info("Reading pastel.conf")
	if err := ParsePastelConf(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to parse pastel config")
		return err
	}
	log.WithContext(ctx).Infof("Finished Reading pastel.conf! Starting Masternode service in %s mode", config.Network)

	// *************  2. Parse parameters  *************
	log.WithContext(ctx).Info("Checking arguments")
	if err := checkStartMasterNodeParams(ctx, config, false); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to validate input arguments")
		return err
	}
	log.WithContext(ctx).Info("Finished checking arguments!")

	return runStartMasternode(ctx, config)
}

func runStartMasternode(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Reading pastel.conf")
	if err := ParsePastelConf(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to parse pastel config")
		return err
	}
	log.WithContext(ctx).Infof("Finished Reading pastel.conf! Starting Supernode in %s mode", config.Network)

	// Get conf data from masternode.conf File
	privKey, extIP, _ /*extPort*/, err := getMasternodeConfData(ctx, config, config.MasterNodeName, config.NodeExtIP)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to get masternode details from masternode.conf")
		return err
	}

	if len(config.NodeExtIP) == 0 {
		log.WithContext(ctx).Info("--ip flag is ommited, trying to get our WAN IP address")
		externalIP, err := utils.GetExternalIPAddress()
		if err != nil {
			err := fmt.Errorf("cannot get external ip address")
			log.WithContext(ctx).WithError(err).Error("Missing parameter --ip")
			return err
		}
		config.NodeExtIP = externalIP
		log.WithContext(ctx).Infof("WAN IP address - %s", config.NodeExtIP)
	}
	if extIP != config.NodeExtIP {
		err := errors.Errorf("External IP address in masternode.conf MUST match WAN address of the node! IP in masternode.conf - %s, WAN IP passed or identified - %s", extIP, config.NodeExtIP)
		log.WithContext(ctx).WithError(err).Error("pasteld failed to start")
		return err
	}

	// *************  Start Node as Masternode  *************
	log.WithContext(ctx).Infof("Starting pasteld as masternode: nodeName: %s; mnPrivKey: %s", config.MasterNodeName, privKey)
	if err := runPastelNode(ctx, config, true, config.ReIndex, config.NodeExtIP, privKey); err != nil { //in masternode mode pasteld MUST be started with reindex flag
		log.WithContext(ctx).WithError(err).Error("pasteld failed to start as masternode")
		return err
	}
	return nil
}

// Sub Command
func runRQService(ctx context.Context, config *configs.Config) error {
	serviceEnabled := false
	sm, err := NewServiceManager(utils.GetOS(), config.Configurer.DefaultHomeDir())
	if err != nil {
		log.WithContext(ctx).Warn(err.Error())
	} else {
		serviceEnabled = true
	}
	if serviceEnabled {
		// if the service isn't registered, this will be a noop
		srvStarted, err := sm.StartService(ctx, config, constants.RQService)
		if err != nil {
			log.WithContext(ctx).Errorf("Failed to start service for %v: %v", constants.RQService, err)
			return err
		}
		if srvStarted {
			log.WithContext(ctx).Infof("Check %s is running...", constants.RQService)
			time.Sleep(10 * time.Second)
			if CheckProcessRunning(constants.RQService) {
				return nil
			}
		}
	}
	rqExecName := constants.PastelRQServiceExecName[utils.GetOS()]
	var rqServiceArgs []string
	configFile := config.Configurer.GetRQServiceConfFile(config.WorkingDir)
	rqServiceArgs = append(rqServiceArgs, fmt.Sprintf("--config-file=%s", configFile))
	if err := runPastelService(ctx, config, constants.RQService, rqExecName, rqServiceArgs...); err != nil {
		log.WithContext(ctx).WithError(err).Error("rqservice failed")
		return err
	}
	return nil
}

// Sub Command
func runDDService(ctx context.Context, config *configs.Config) (err error) {
	serviceEnabled := false
	sm, err := NewServiceManager(utils.GetOS(), config.Configurer.DefaultHomeDir())
	if err != nil {
		log.WithContext(ctx).Warn(err.Error())
	} else {
		serviceEnabled = true
	}
	srvStarted := false
	if serviceEnabled {
		// if the service isn't registered, this will be a noop
		srvStarted, err = sm.StartService(ctx, config, constants.DDService)
		if err != nil {
			log.WithContext(ctx).Errorf("Failed to start service for %v: %v", constants.DDService, err)
			return err
		}
	}
	if !srvStarted {
		var execPath string
		if execPath, err = checkPastelFilePath(ctx, config.PastelExecDir, utils.GetDupeDetectionExecName()); err != nil {
			log.WithContext(ctx).WithError(err).Error("Could not find dupe detection service script")
			return err
		}

		ddConfigFilePath := filepath.Join(config.Configurer.DefaultHomeDir(),
			constants.DupeDetectionServiceDir,
			constants.DupeDetectionSupportFilePath,
			constants.DupeDetectionConfigFilename)

		python := "python3"
		if utils.GetOS() == constants.Windows {
			python = "python"
		}
		venv := filepath.Join(config.PastelExecDir, constants.DupeDetectionSubFolder, "venv")
		cmd := fmt.Sprintf("source %v/bin/activate && %v %v %v", venv, python, execPath, ddConfigFilePath)
		go RunCMD("bash", "-c", cmd)
	}

	time.Sleep(10 * time.Second)

	if output, err := FindRunningProcess(constants.DupeDetectionExecFileName); len(output) == 0 {
		err = errors.Errorf("dd-service failed to start")
		log.WithContext(ctx).WithError(err).Error("dd-service failed to start")
		return err
	} else if err != nil {
		log.WithContext(ctx).WithError(err).Error("failed to test if dd-service is running")
	} else {
		log.WithContext(ctx).Info("dd-service is successfully started")
	}
	return nil
}

func runDDImgServer(ctx context.Context, config *configs.Config) (err error) {
	sm, err := NewServiceManager(utils.GetOS(), config.Configurer.DefaultHomeDir())
	if err != nil {
		log.WithContext(ctx).Error(err.Error())
		return err
	}

	// if the service isn't registered, this will be a noop
	_, err = sm.StartService(ctx, config, constants.DDImgService)
	if err != nil {
		log.WithContext(ctx).Errorf("Failed to start service for %v: %v", constants.DDImgService, err)
		return err
	}

	return nil
}

// Sub Command
func runWalletNodeService(ctx context.Context, config *configs.Config) error {
	serviceEnabled := false
	sm, err := NewServiceManager(utils.GetOS(), config.Configurer.DefaultHomeDir())
	if err != nil {
		log.WithContext(ctx).Warn(err.Error())
	} else {
		serviceEnabled = true
	}
	if serviceEnabled {
		// if the service isn't registered, this will be a noop
		srvStarted, err := sm.StartService(ctx, config, constants.WalletNode)
		if err != nil {
			log.WithContext(ctx).Errorf("Failed to start service for %v: %v", constants.WalletNode, err)
			return err
		}
		if srvStarted {
			log.WithContext(ctx).Infof("Check %s is running...", constants.WalletNode)
			time.Sleep(10 * time.Second)
			if CheckProcessRunning(constants.WalletNode) {
				return nil
			}
		}
	}
	walletnodeExecName := constants.WalletNodeExecName[utils.GetOS()]
	log.WithContext(ctx).Infof("Starting walletnode service - %s", walletnodeExecName)
	var wnServiceArgs []string
	wnServiceArgs = append(wnServiceArgs,
		fmt.Sprintf("--config-file=%s", config.Configurer.GetWalletNodeConfFile(config.WorkingDir)))
	wnServiceArgs = append(wnServiceArgs,
		fmt.Sprintf("--pastel-config-file=%s/pastel.conf", config.WorkingDir))
	if config.DevMode {
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
func runBridgeService(ctx context.Context, config *configs.Config) error {
	bridgeConf := config.Configurer.GetBridgeConfFile(config.WorkingDir)
	if err := checkBridgeConfigPastelID(ctx, config, bridgeConf); err != nil {
		log.WithContext(ctx).Errorf("Failed to verify bridge Pastelid %v: %v", bridgeConf, err)
		return err
	}

	serviceEnabled := false
	sm, err := NewServiceManager(utils.GetOS(), config.Configurer.DefaultHomeDir())
	if err != nil {
		log.WithContext(ctx).Warn(err.Error())
	} else {
		serviceEnabled = true
	}
	if serviceEnabled {
		// if the service isn't registered, this will be a noop
		srvStarted, err := sm.StartService(ctx, config, constants.Bridge)
		if err != nil {
			log.WithContext(ctx).Errorf("Failed to start service for %v: %v", constants.Bridge, err)
			return err
		}
		if srvStarted {
			log.WithContext(ctx).Infof("Check %s is running...", constants.Bridge)
			time.Sleep(10 * time.Second)
			if CheckProcessRunning(constants.Bridge) {
				return nil
			}
		}
	}
	bridgeExecName := constants.BridgeExecName[utils.GetOS()]
	log.WithContext(ctx).Infof("Starting bridge service - %s", bridgeExecName)
	var bridgeServiceArgs []string

	bridgeServiceArgs = append(bridgeServiceArgs,
		fmt.Sprintf("--config-file=%s", bridgeConf))
	bridgeServiceArgs = append(bridgeServiceArgs,
		fmt.Sprintf("--pastel-config-file=%s/pastel.conf", config.WorkingDir))

	log.WithContext(ctx).Infof("Options : %s", bridgeServiceArgs)
	if err := runPastelService(ctx, config, constants.Bridge, bridgeExecName, bridgeServiceArgs...); err != nil {
		log.WithContext(ctx).WithError(err).Error("bridge service failed")
		return err
	}

	return nil
}

// Sub Command
func runSuperNodeService(ctx context.Context, config *configs.Config) error {
	serviceEnabled := false
	sm, err := NewServiceManager(utils.GetOS(), config.Configurer.DefaultHomeDir())
	if err != nil {
		log.WithContext(ctx).Warn(err.Error())
	} else {
		serviceEnabled = true
	}
	if serviceEnabled {
		// if the service isn't registered, this will be a noop
		srvStarted, err := sm.StartService(ctx, config, constants.SuperNode)
		if err != nil {
			log.WithContext(ctx).Errorf("Failed to start service for %v: %v", constants.SuperNode, err)
		}
		if srvStarted {
			log.WithContext(ctx).Infof("Check %s is running...", constants.SuperNode)
			time.Sleep(10 * time.Second)
			if CheckProcessRunning(constants.SuperNode) {
				if err := runHermesService(ctx, config); err != nil {
					log.WithContext(ctx).WithError(err).Error("sn-service started bu start hermes service failed")
					return err
				}
				return nil
			}
		}
	}
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

	if err := runHermesService(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("start hermes service failed")
		return err
	}

	return nil
}

// Sub Command
func runHermesService(ctx context.Context, config *configs.Config) error {
	if err := setPastelIDAndPassphraseInHermesConf(ctx, config); err != nil {
		log.WithContext(ctx).Errorf("Failed to set pastelID & passphrase in config file : %v", err)
		return nil
	}

	serviceEnabled := false
	sm, err := NewServiceManager(utils.GetOS(), config.Configurer.DefaultHomeDir())
	if err != nil {
		log.WithContext(ctx).Warn(err.Error())
	} else {
		serviceEnabled = true
	}
	if serviceEnabled {
		// if the service isn't registered, this will be a noop
		srvStarted, err := sm.StartService(ctx, config, constants.Hermes)
		if err != nil {
			log.WithContext(ctx).Errorf("Failed to start service for %v: %v", constants.Hermes, err)
		}
		if srvStarted {
			log.WithContext(ctx).Infof("Check %s is running...", constants.Hermes)
			time.Sleep(10 * time.Second)
			if CheckProcessRunning(constants.Hermes) {
				return nil
			}
		}
	}
	hermesConfigPath := config.Configurer.GetHermesConfFile(config.WorkingDir)
	hermesExecName := constants.HermesExecName[utils.GetOS()]
	log.WithContext(ctx).Infof("Starting hermes service - %s", hermesExecName)

	var hermesServiceArgs []string
	hermesServiceArgs = append(hermesServiceArgs,
		fmt.Sprintf("--config-file=%s", hermesConfigPath))

	log.WithContext(ctx).Infof("Options : %s", hermesServiceArgs)
	if err := runPastelService(ctx, config, constants.Hermes, hermesExecName, hermesServiceArgs...); err != nil {
		log.WithContext(ctx).WithError(err).Error("hermes start failed")
		return err
	}

	return nil
}

func setPastelIDAndPassphraseInHermesConf(ctx context.Context, config *configs.Config) error {
	hermesConfigPath := config.Configurer.GetHermesConfFile(config.WorkingDir)
	supernodeConfigPath := config.Configurer.GetSuperNodeConfFile(config.WorkingDir)

	// Read existing supernode confif file
	var snConfFile []byte
	snConfFile, err := os.ReadFile(supernodeConfigPath)
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to open existing supernode.yml file at - %s", supernodeConfigPath)
		return err
	}
	snConf := make(map[string]interface{})
	if err = yaml.Unmarshal(snConfFile, &snConf); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to parse existing supernode.yml file at - %s", supernodeConfigPath)
		return err
	}

	// extract pastelID & Passphrase
	node := snConf["node"].(map[interface{}]interface{})
	pastelID := fmt.Sprintf("%s", node["pastel_id"])
	passPhrase := fmt.Sprintf("%s", node["pass_phrase"])

	// Read hermes config file
	var hermesConfFile []byte
	hermesConfFile, err = os.ReadFile(hermesConfigPath)
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to open existing hermes.yml file at - %s", hermesConfigPath)
		return err
	}

	hermesConf := make(map[string]interface{})
	if err = yaml.Unmarshal(hermesConfFile, &hermesConf); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to parse existing hermes.yml file at - %s", hermesConfigPath)
		return err
	}

	// Update hermes config file
	hermesConf["pastel_id"] = pastelID
	hermesConf["pass_phrase"] = passPhrase

	var hermesConfFileUpdated []byte
	if hermesConfFileUpdated, err = yaml.Marshal(&hermesConf); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to unparse yml for hermes.yml file at - %s", hermesConfigPath)
		return err
	}

	if os.WriteFile(hermesConfigPath, hermesConfFileUpdated, 0644) != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to update hermes.yml file at - %s", hermesConfigPath)
		return err
	}

	return nil
}

// /// Run helpers
func runPastelNode(ctx context.Context, config *configs.Config, txIndexOne bool, reindex bool, extIP string, mnPrivKey string) (err error) {
	serviceEnabled := false
	sm, err := NewServiceManager(utils.GetOS(), config.Configurer.DefaultHomeDir())
	if err != nil {
		log.WithContext(ctx).Warn(err.Error())
	} else {
		serviceEnabled = true
	}
	srvStarted := false
	if serviceEnabled {
		// if the service isn't registered, this will be a noop
		srvStarted, err = sm.StartService(ctx, config, constants.PastelD)
		if err != nil {
			log.WithContext(ctx).Errorf("Failed to start service for %v: %v", constants.PastelD, err)
			return err
		}
	}
	if !srvStarted {

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
			if extIP, err = utils.GetExternalIPAddress(); err != nil {
				log.WithContext(ctx).WithError(err).Error("Could not get external IP address")
				return err
			}
		}

		var pasteldArgs []string
		pasteldArgs = append(pasteldArgs,
			fmt.Sprintf("--datadir=%s", config.WorkingDir),
			fmt.Sprintf("--externalip=%s", extIP))

		if txIndexOne {
			pasteldArgs = append(pasteldArgs, "--txindex=1")
		}

		if reindex {
			pasteldArgs = append(pasteldArgs, "--reindex")
		}

		if len(mnPrivKey) != 0 {
			pasteldArgs = append(pasteldArgs, "--masternode", fmt.Sprintf("--masternodeprivkey=%s", mnPrivKey))
		}

		log.WithContext(ctx).Infof("Starting -> %s %s", pastelDPath, strings.Join(pasteldArgs, " "))
		pasteldArgs = append(pasteldArgs, "--daemon")
		go RunCMD(pastelDPath, pasteldArgs...)
	}

	if !WaitingForPastelDToStart(ctx, config) {
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
		log.WithContext(ctx).Infof("The %s started successfully!", toolType)
	} else {
		if output, err := RunCMD(execPath, args...); err != nil {
			log.WithContext(ctx).Errorf("%s start failed! : %s", toolType, output)
			return err
		}
	}

	return nil
}

// /// Validates input parameters
func checkStartMasterNodeParams(ctx context.Context, config *configs.Config, coldHot bool) error {

	// --ip WAN IP address of the node - Required, WAN address of the host
	if len(config.NodeExtIP) == 0 && !coldHot { //coldHot will try to get WAN address in the step that is executed on remote host

		log.WithContext(ctx).Info("--ip flag is ommited, trying to get our WAN IP address")
		externalIP, err := utils.GetExternalIPAddress()
		if err != nil {
			err := fmt.Errorf("cannot get external ip address")
			log.WithContext(ctx).WithError(err).Error("Missing parameter --ip")
			return err
		}
		config.NodeExtIP = externalIP
		log.WithContext(ctx).Infof("WAN IP address - %s", config.NodeExtIP)
	}

	if !config.CreateNewMasterNodeConf { // if we don't create new masternode.conf - it must exist!
		masternodeConfPath := getMasternodeConfPath(config, "", "masternode.conf")
		if _, err := checkPastelFilePath(ctx, config.WorkingDir, masternodeConfPath); err != nil {
			log.WithContext(ctx).WithError(err).Error("Could not find masternode.conf - use --create flag")
			return err
		}
	}

	if coldHot {
		if len(config.RemoteIP) == 0 {
			err := fmt.Errorf("required if --coldhot is specified, –-ssh-ip, SSH address of the remote HOT node")
			log.WithContext(ctx).WithError(err).Error("Missing parameter --ssh-ip")
			return err
		}
	}

	if len(config.MasterNodeRPCIP) == 0 {
		config.MasterNodeRPCIP = config.NodeExtIP
	}

	if len(config.MasterNodeP2PIP) == 0 {
		config.MasterNodeP2PIP = config.NodeExtIP
	}

	portList := GetSNPortList(config)
	log.WithContext(ctx).Infof("CONFIG: %+v", config)

	if config.MasterNodePort == 0 {
		config.MasterNodePort = portList[constants.NodePort]
	}

	if config.MasterNodeRPCPort == 0 {
		config.MasterNodeRPCPort = portList[constants.SNPort]
	}

	if config.MasterNodeP2PPort == 0 {
		config.MasterNodeP2PPort = portList[constants.P2PPort]
	}
	log.WithContext(ctx).Infof("CONFIG: %+v", config)

	return nil
}

// /// Run helpers
func prepareMasterNodeParameters(ctx context.Context, config *configs.Config, startPasteld bool) (err error) {

	// this function must only be called when --create or --update
	if !config.CreateNewMasterNodeConf && !config.AddToMasterNodeConf {
		return nil
	}

	if startPasteld {
		log.WithContext(ctx).Infof("Starting pasteld")
		// in masternode mode pasteld MUST be start with txIndex=1 flag
		if err = runPastelNode(ctx, config, true, config.ReIndex, config.NodeExtIP, ""); err != nil {
			log.WithContext(ctx).WithError(err).Error("pasteld failed to start")
			return err
		}
	}

	// Check masternode status
	if _, err = CheckMasterNodeSync(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("pasteld failed to synchronize, add some peers and try again")
		return err
	}

	if err := checkCollateral(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("Missing collateral transaction")
		return err
	}

	// sets Passphrase to flagMasterNodePassphrase
	if err := checkPassphrase(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("Missing passphrase")
		return err
	}

	if err := checkMasternodePrivKey(ctx, config, nil); err != nil {
		log.WithContext(ctx).WithError(err).Error("Missing masternode private key")
		return err
	}

	// sets PastelID to MasterNodePastelID
	if err := checkPastelID(ctx, config, nil); err != nil {
		log.WithContext(ctx).WithError(err).Error("Missing masternode PastelID")
		return err
	}

	if startPasteld {
		if err = StopPastelDAndWait(ctx, config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Cannot stop pasteld")
			return err
		}
	}
	return nil
}

func checkPastelID(ctx context.Context, config *configs.Config, client *utils.Client) (err error) {
	if len(config.MasterNodePastelID) != 0 {
		log.WithContext(ctx).Infof("Masternode pastelid already set = %s", config.MasterNodePastelID)
		return nil
	}

	log.WithContext(ctx).Info("Masternode PastelID is empty - will create new one")

	if len(config.MasterNodePassPhrase) == 0 { //check one more time just because
		err := fmt.Errorf("required parameter if --create or --update specified: --passphrase <passphrase to pastelid private key>")
		log.WithContext(ctx).WithError(err).Error("Missing parameter --passphrase")
		return err
	}

	var pastelid string
	if client == nil {
		var resp map[string]interface{}
		err = pastelcore.NewClient(config).RunCommandWithArgs(
			pastelcore.PastelIDCmd,
			[]string{"newkey", config.MasterNodePassPhrase},
			&resp,
		)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to generate new pastelid key")
			return err
		}
		res := resp["result"].(map[string]interface{})
		pastelid = res["pastelid"].(string)
	} else { //client is not nil when called from ColdHot Init
		pastelcliPath := filepath.Join(config.RemoteHotPastelExecDir, constants.PastelCliName[utils.GetOS()])
		out, err := client.Cmd(fmt.Sprintf("%s %s %s", pastelcliPath, "pastelid newkey",
			config.MasterNodePassPhrase)).Output()
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to generate new pastelid key on Hot node")
			return err
		}
		log.WithContext(ctx).WithField("out", string(out)).Info("Pastelid new key generated")

		var resp map[string]interface{}
		if err := json.Unmarshal(out, &resp); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to unmarshal response of pastelid new key")
			return err
		}
		if val, ok := resp["result"]; ok {
			res := val.(map[string]interface{})
			pastelid = res["pastelid"].(string)
		} else {
			pastelid = resp["pastelid"].(string)
		}

		log.WithContext(ctx).WithField("pastelid", pastelid).Info("generated pastel key on hotnode")
	}
	config.MasterNodePastelID = pastelid

	log.WithContext(ctx).Infof("Masternode pastelid = %s", config.MasterNodePastelID)

	return nil
}

func checkMasternodePrivKey(ctx context.Context, config *configs.Config, client *utils.Client) (err error) {
	if len(config.MasterNodePrivateKey) == 0 {
		log.WithContext(ctx).Info("Masternode private key is empty - will create new one")

		var mnPrivKey string
		if client == nil {
			var resp map[string]interface{}
			err = pastelcore.NewClient(config).RunCommandWithArgs(
				pastelcore.MasterNodeCmd,
				[]string{"genkey"},
				&resp,
			)
			if err != nil {
				log.WithContext(ctx).WithError(err).Error("Failed to generate new masternode private key")
				return err
			}
			mnPrivKey = resp["result"].(string)
		} else { //client is not nil when called from ColdHot Init
			pastelcliPath := filepath.Join(config.RemoteHotPastelExecDir, constants.PastelCliName[utils.GetOS()])
			cmd := fmt.Sprintf("%s %s", pastelcliPath, "masternode genkey")
			out, err := client.Cmd(cmd).Output()
			if err != nil {
				log.WithContext(ctx).WithField("cmd", cmd).WithField("out", string(out)).WithError(err).Error("Failed to generate new masternode private key on Hot node")
				return err
			}

			mnPrivKey = string(out)
			fmt.Println("generated priv key on hotnode: ", mnPrivKey)
		}

		config.MasterNodePrivateKey = strings.TrimSuffix(mnPrivKey, "\n")
	}
	log.WithContext(ctx).Infof("masternode private key = %s", config.MasterNodePrivateKey)
	return nil
}

func checkPassphrase(ctx context.Context, config *configs.Config) error {
	if len(config.MasterNodePassPhrase) == 0 {

		_, config.MasterNodePassPhrase = AskUserToContinue(ctx, "No --passphrase provided."+
			" Please type new passphrase and press Enter. Or N to exit")
		if strings.EqualFold(config.MasterNodePassPhrase, "n") ||
			len(config.MasterNodePassPhrase) == 0 {

			config.MasterNodePassPhrase = ""
			err := fmt.Errorf("required parameter if --create or --update specified: --passphrase <passphrase to pastelid private key>")
			log.WithContext(ctx).WithError(err).Error("User terminated - exiting")
			return err
		}
	}
	log.WithContext(ctx).Infof(red(fmt.Sprintf("passphrase - %s", config.MasterNodePassPhrase)))
	return nil
}

func getMasternodeOutputs(ctx context.Context, config *configs.Config) (map[string]interface{}, error) {
	var mnOutputs map[string]interface{}
	err := pastelcore.NewClient(config).RunCommandWithArgs(
		pastelcore.MasterNodeCmd,
		[]string{"outputs"},
		&mnOutputs,
	)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to get masternode outputs from pasteld")
		return nil, err
	}
	res, ok := mnOutputs["result"].(map[string]interface{})
	if !ok {
		log.WithContext(ctx).Error("Unexpected format for masternode outputs")
		return nil, err
	}
	return res, nil
}

func checkCollateral(ctx context.Context, config *configs.Config) error {
	var err error

	if config.DontCheckCollateral && len(config.MasterNodeTxID) != 0 && len(config.MasterNodeTxInd) != 0 {
		return nil
	}

	if len(config.MasterNodeTxID) == 0 || len(config.MasterNodeTxInd) == 0 {

		log.WithContext(ctx).Warn(red("No collateral --txid and/or --ind provided"))
		yes, _ := AskUserToContinue(ctx, "Search existing masternode collateral ready transaction in the wallet? Y/N")

		if yes {
			var mnOutputs map[string]interface{}
			mnOutputs, err = getMasternodeOutputs(ctx, config)
			if err != nil {
				log.WithContext(ctx).WithError(err).Error("Failed")
				return err
			}

			if len(mnOutputs) > 0 {

				n := 0
				var arr []string
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

				config.MasterNodeTxID = arr[dNum]
				config.MasterNodeTxInd = mnOutputs[config.MasterNodeTxID].(string)
			} else {
				log.WithContext(ctx).Warn(red("No existing collateral ready transactions"))
			}
		}
	}

	if len(config.MasterNodeTxID) == 0 || len(config.MasterNodeTxInd) == 0 {

		collateralAmount := "5"
		collateralCoins := "PSL"
		if config.Network == constants.NetworkTestnet {
			collateralAmount = "1"
			collateralCoins = "LSP"
		} else if config.Network == constants.NetworkRegTest {
			collateralAmount = "0.1"
			collateralCoins = "REG"
		}

		yes, _ := AskUserToContinue(ctx, fmt.Sprintf("Do you want to generate new local address and send %sM %s to it from another wallet? Y/N",
			collateralAmount, collateralCoins))

		if !yes {
			err = fmt.Errorf("no collateral funds")
			log.WithContext(ctx).WithError(err).Error("No collateral funds - exiting")
			return err
		}
		var address string
		out, err := RunPastelCLI(ctx, config, pastelcore.GetNewAddressCmd)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to get new address")
			return err
		}

		address = strings.Trim(out, "\n")
		log.WithContext(ctx).Warnf(red(fmt.Sprintf("Your new address for collateral payment is %s", address)))
		log.WithContext(ctx).Warnf(red(fmt.Sprintf("Use another wallet to send exactly %sM %s to that address.", collateralAmount, collateralCoins)))
		_, newTxid := AskUserToContinue(ctx, "Enter txid of the send and press Enter to continue when ready")
		config.MasterNodeTxID = strings.Trim(newTxid, "\n")
	}

	for i := 1; i <= 10; i++ {

		var mnOutputs map[string]interface{}
		mnOutputs, err = getMasternodeOutputs(ctx, config)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed")
			return err
		}

		txind, ok := mnOutputs[config.MasterNodeTxID].(string)
		if ok {
			config.MasterNodeTxInd = txind
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

	if len(config.MasterNodeTxID) == 0 || len(config.MasterNodeTxInd) == 0 {

		err := errors.Errorf("Cannot find masternode outputs = %s:%s", config.MasterNodeTxID, config.MasterNodeTxInd)
		log.WithContext(ctx).WithError(err).Error("Try again after some time")
		return err
	}

	// if receives PSL go to next step
	log.WithContext(ctx).Infof(red(fmt.Sprintf("masternode outputs = %s, %s", config.MasterNodeTxID, config.MasterNodeTxInd)))
	return nil
}

// /// Masternode specific
func runStartAliasMasternode(ctx context.Context, config *configs.Config, masternodeName string) error {
	var aliasStatus map[string]interface{}
	err := pastelcore.NewClient(config).RunCommandWithArgs(
		pastelcore.MasterNodeCmd,
		[]string{"start-alias", masternodeName},
		&aliasStatus,
	)
	if err != nil {
		return err
	}
	if aliasStatus["result"] == "failed" {
		err = fmt.Errorf("masternode start alias failed")
		log.WithContext(ctx).WithError(err).Error(aliasStatus["errorMessage"])
		return err
	}
	log.WithContext(ctx).Infof("masternode alias status = %s\n", aliasStatus["result"])
	return nil
}

// /// supernode.yml hlpers
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
		ddDirPath := filepath.Join(config.Configurer.DefaultHomeDir(), constants.DupeDetectionServiceDir)

		toolConfig, err := utils.GetServiceConfig(string(constants.SuperNode), configs.SupernodeDefaultConfig, &configs.SuperNodeConfig{
			LogFilePath:                     config.Configurer.GetSuperNodeLogFile(config.WorkingDir),
			LogCompress:                     constants.LogConfigDefaultCompress,
			LogMaxSizeMB:                    constants.LogConfigDefaultMaxSizeMB,
			LogMaxAgeDays:                   constants.LogConfigDefaultMaxAgeDays,
			LogMaxBackups:                   constants.LogConfigDefaultMaxBackups,
			LogLevelCommon:                  constants.SuperNodeDefaultCommonLogLevel,
			LogLevelP2P:                     constants.SuperNodeDefaultP2PLogLevel,
			LogLevelMetadb:                  constants.SuperNodeDefaultMetaDBLogLevel,
			LogLevelDD:                      constants.SuperNodeDefaultDDLogLevel,
			SNTempDir:                       snTempDirPath,
			SNWorkDir:                       config.WorkingDir,
			RQDir:                           rqWorkDirPath,
			DDDir:                           ddDirPath,
			SuperNodePort:                   portList[constants.SNPort],
			P2PPort:                         portList[constants.P2PPort],
			P2PDataDir:                      p2pDataPath,
			MDLPort:                         portList[constants.MDLPort],
			RAFTPort:                        portList[constants.RAFTPort],
			MDLDataDir:                      mdlDataPath,
			RaptorqPort:                     constants.RQServiceDefaultPort,
			DDServerPort:                    constants.DDServerDefaultPort,
			NumberOfChallengeReplicas:       constants.NumberOfChallengeReplicas,
			StorageChallengeExpiredDuration: constants.StorageChallengeExpiredDuration,
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
		snConfFile, err = os.ReadFile(supernodeConfigPath)
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

		node["pastel_id"] = config.MasterNodePastelID
		node["pass_phrase"] = config.MasterNodePassPhrase
		node["storage_challenge_expired_duration"] = constants.StorageChallengeExpiredDuration
		node["number_of_challenge_replicas"] = constants.NumberOfChallengeReplicas

		var snConfFileUpdated []byte
		if snConfFileUpdated, err = yaml.Marshal(&snConf); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to unparse yml for supernode.yml file at - %s", supernodeConfigPath)
			return err
		}
		if os.WriteFile(supernodeConfigPath, snConfFileUpdated, 0644) != nil {
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
