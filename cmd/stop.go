package cmd

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"os"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pastelnetwork/pastelup/configs"
	"github.com/pastelnetwork/pastelup/constants"
	"github.com/pastelnetwork/pastelup/utils"
)

type stopCommand uint8

const (
	nodeStop stopCommand = iota
	walletStop
	superNodeStop
	allStop
	ddServiceStop
	rqServiceStop
	ddImgServerStop
	wnServiceStop
	snServiceStop
	remoteStop
	bridgeServiceStop
	hermesServiceStop
)

var (
	stopCmdName = map[stopCommand]string{
		nodeStop:          "node",
		walletStop:        "walletnode",
		superNodeStop:     "supernode",
		allStop:           "all",
		ddServiceStop:     "dd-service",
		rqServiceStop:     "rq-service",
		ddImgServerStop:   "imgserver",
		wnServiceStop:     "walletnode-service",
		snServiceStop:     "supernode-service",
		remoteStop:        "remote",
		bridgeServiceStop: "bridge-service",
		hermesServiceStop: "hermes-service",
	}
	stopCmdMessage = map[stopCommand]string{
		nodeStop:          "Stop node",
		walletStop:        "Stop Walletnode",
		superNodeStop:     "Stop Supernode",
		allStop:           "Stop all Pastel services",
		ddServiceStop:     "Stop Dupe Detection service only",
		rqServiceStop:     "Stop RaptorQ service only",
		ddImgServerStop:   "Stop DD Image Server",
		wnServiceStop:     "Stop Walletnode service only",
		snServiceStop:     "Stop Supernode service only",
		remoteStop:        "Stop on Remote Host",
		bridgeServiceStop: "Start bridge-service only",
		hermesServiceStop: "Start hermes-service only",
	}
)

var serviceToProcessOverrides = map[string]string{
	string(constants.DDService): constants.DupeDetectionExecFileName,
}

func setupStopSubCommand(config *configs.Config,
	stopCommand stopCommand, remote bool,
	f func(context.Context, *configs.Config),
) *cli.Command {

	var commandFlags []*cli.Flag

	if !remote {
		commandFlags = []*cli.Flag{
			cli.NewFlag("dir", &config.PastelExecDir).SetAliases("d").
				SetUsage(green("Optional, Location of pastel node directory")).SetValue(config.Configurer.DefaultPastelExecutableDir()),
			cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
				SetUsage(green("Optional, location of working directory")).SetValue(config.Configurer.DefaultWorkingDir()),
		}
	} else {
		commandFlags = []*cli.Flag{
			cli.NewFlag("dir", &config.PastelExecDir).SetAliases("d").
				SetUsage(green("Optional, Location where to create pastel node directory on the remote computer (default: $HOME/pastel)")),
			cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
				SetUsage(green("Optional, Location where to create working directory on the remote computer (default: $HOME/.pastel)")),
		}
	}

	remoteStopFlags := []*cli.Flag{
		cli.NewFlag("ssh-ip", &config.RemoteIP).
			SetUsage(red("Required, SSH address of the remote host")).SetRequired(),
		cli.NewFlag("ssh-port", &config.RemotePort).
			SetUsage(yellow("Optional, SSH port of the remote host, default is 22")).SetValue(22),
		cli.NewFlag("ssh-user", &config.RemoteUser).
			SetUsage(yellow("Optional, Username of user at remote host")),
		cli.NewFlag("ssh-key", &config.RemoteSSHKey).
			SetUsage(yellow("Optional, Path to SSH private key for SSH Key Authentication")),
		cli.NewFlag("inventory", &config.InventoryFile).
			SetUsage(red("Optional, Path to the file with configuration of the remote hosts")),
	}

	var commandName, commandMessage string
	if !remote {
		commandName = stopCmdName[stopCommand]
		commandMessage = stopCmdMessage[stopCommand]
	} else {
		commandName = stopCmdName[remoteStop]
		commandMessage = stopCmdMessage[remoteStop]
	}
	if remote {
		commandFlags = append(commandFlags, remoteStopFlags[:]...)
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
			if err = ParsePastelConf(ctx, config); err != nil {
				return err
			}
			log.WithContext(ctx).Info("Stopping...")
			f(ctx, config)
			log.WithContext(ctx).Info("Finished successfully!")
			return nil
		})
	}
	return subCommand
}

func setupStopCommand(config *configs.Config) *cli.Command {

	stopNodeSubCommand := setupStopSubCommand(config, nodeStop, false, runStopNodeSubCommand)
	stopWalletSubCommand := setupStopSubCommand(config, walletStop, false, runStopWalletSubCommand)
	stopSuperNodeSubCommand := setupStopSubCommand(config, superNodeStop, false, runStopSuperNodeSubCommand)
	stopAllSubCommand := setupStopSubCommand(config, allStop, false, runStopAllSubCommand)

	stopRQSubCommand := setupStopSubCommand(config, rqServiceStop, false, stopRQServiceSubCommand)
	stopDDSubCommand := setupStopSubCommand(config, ddServiceStop, false, stopDDServiceSubCommand)
	stopWNSubCommand := setupStopSubCommand(config, wnServiceStop, false, stopWNServiceSubCommand)
	stopSNSubCommand := setupStopSubCommand(config, snServiceStop, false, stopSNServiceSubCommand)

	stopHermesServiceCommand := setupStopSubCommand(config, hermesServiceStop, false, stopHermesService)
	stopBridgeServiceCommand := setupStopSubCommand(config, bridgeServiceStop, false, stopBridgeService)

	stopDDImgServerCommand := setupStopSubCommand(config, ddImgServerStop, false, stopDDImgServer)

	stopNodeSubCommand.AddSubcommands(setupStopSubCommand(config, nodeStop, true, runStopNodeRemoteSubCommand))
	stopWalletSubCommand.AddSubcommands(setupStopSubCommand(config, walletStop, true, runStopWalletNodeRemoteSubCommand))
	stopSuperNodeSubCommand.AddSubcommands(setupStopSubCommand(config, superNodeStop, true, runStopSuperNodeRemoteSubCommand))
	stopAllSubCommand.AddSubcommands(setupStopSubCommand(config, allStop, true, runStopAllRemoteSubCommand))
	stopRQSubCommand.AddSubcommands(setupStopSubCommand(config, rqServiceStop, true, runStopRQServiceRemoteSubCommand))
	stopDDSubCommand.AddSubcommands(setupStopSubCommand(config, ddServiceStop, true, runStopDDServiceRemoteSubCommand))
	stopWNSubCommand.AddSubcommands(setupStopSubCommand(config, wnServiceStop, true, runStopWNServiceRemoteSubCommand))
	stopSNSubCommand.AddSubcommands(setupStopSubCommand(config, snServiceStop, true, runStopSNServiceRemoteSubCommand))

	stopHermesServiceCommand.AddSubcommands(setupStopSubCommand(config, hermesServiceStop, true, runStopHermesServiceRemoteSubCommand))
	stopBridgeServiceCommand.AddSubcommands(setupStopSubCommand(config, bridgeServiceStop, true, runStopBridgeServiceRemoteSubCommand))
	stopDDImgServerCommand.AddSubcommands(setupStopSubCommand(config, ddImgServerStop, true, runStopDDImgServerRemoteSubCommand))

	stopCommand := cli.NewCommand("stop")
	stopCommand.SetUsage(blue("Performs stop of the system for both WalletNode and SuperNodes"))
	stopCommand.AddSubcommands(stopNodeSubCommand)
	stopCommand.AddSubcommands(stopWalletSubCommand)
	stopCommand.AddSubcommands(stopSuperNodeSubCommand)
	stopCommand.AddSubcommands(stopAllSubCommand)

	stopCommand.AddSubcommands(stopRQSubCommand)
	stopCommand.AddSubcommands(stopDDSubCommand)
	stopCommand.AddSubcommands(stopDDImgServerCommand)
	stopCommand.AddSubcommands(stopWNSubCommand)
	stopCommand.AddSubcommands(stopSNSubCommand)

	return stopCommand
}

func runStopNodeSubCommand(ctx context.Context, config *configs.Config) {
	_ = stopServices(ctx, []constants.ToolType{constants.PastelD}, config)
}

func runStopWalletSubCommand(ctx context.Context, config *configs.Config) {
	servicesToStop := []constants.ToolType{
		constants.WalletNode,
		constants.RQService,
		constants.PastelD,
		constants.Bridge,
	}
	_ = stopServices(ctx, servicesToStop, config)
	log.WithContext(ctx).Info("Walletnode stopped successfully")
}

func runStopSuperNodeSubCommand(ctx context.Context, config *configs.Config) {
	servicesToStop := []constants.ToolType{
		constants.SuperNode,
		constants.RQService,
		constants.DDImgService,
		constants.DDService,
		constants.PastelD,
		constants.Hermes,
	}
	_ = stopServices(ctx, servicesToStop, config)
	log.WithContext(ctx).Info("Suppernode stopped successfully")
}

func runStopNodeRemoteSubCommand(ctx context.Context, config *configs.Config) {
	runRemoteStop(ctx, config, "node")
}
func runStopWalletNodeRemoteSubCommand(ctx context.Context, config *configs.Config) {
	runRemoteStop(ctx, config, "walletnode")
}
func runStopSuperNodeRemoteSubCommand(ctx context.Context, config *configs.Config) {
	runRemoteStop(ctx, config, "supernode")
}
func runStopAllRemoteSubCommand(ctx context.Context, config *configs.Config) {
	runRemoteStop(ctx, config, "all")
}
func runStopRQServiceRemoteSubCommand(ctx context.Context, config *configs.Config) {
	runRemoteStop(ctx, config, "rq-service")
}
func runStopDDServiceRemoteSubCommand(ctx context.Context, config *configs.Config) {
	runRemoteStop(ctx, config, "dd-service")
}
func runStopDDImgServerRemoteSubCommand(ctx context.Context, config *configs.Config) {
	runRemoteStop(ctx, config, "imgserver")
}
func runStopWNServiceRemoteSubCommand(ctx context.Context, config *configs.Config) {
	runRemoteStop(ctx, config, "walletnode-service")
}
func runStopSNServiceRemoteSubCommand(ctx context.Context, config *configs.Config) {
	runRemoteStop(ctx, config, "supernode-service")
}
func runStopHermesServiceRemoteSubCommand(ctx context.Context, config *configs.Config) {
	runRemoteStop(ctx, config, "hermes-service")
}
func runStopBridgeServiceRemoteSubCommand(ctx context.Context, config *configs.Config) {
	runRemoteStop(ctx, config, "bridge-service")
}

// special handling for remote command
func runRemoteStop(ctx context.Context, config *configs.Config, tool string) {
	log.WithContext(ctx).Infof("Stopping remote %s", tool)

	stopOptions := tool
	if len(config.PastelExecDir) > 0 {
		stopOptions = fmt.Sprintf("%s --dir %s", stopOptions, config.PastelExecDir)
	}
	if len(config.WorkingDir) > 0 {
		stopOptions = fmt.Sprintf("%s --work-dir %s", stopOptions, config.WorkingDir)
	}

	stopSuperNodeCmd := fmt.Sprintf("%s stop %s", constants.RemotePastelupPath, stopOptions)
	if err := executeRemoteCommandsWithInventory(ctx, config, []string{stopSuperNodeCmd}, false); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to stop %s on remote host", tool)
	}

	log.WithContext(ctx).Infof("Remote %s stopped successfully", tool)
}

func runStopAllSubCommand(ctx context.Context, config *configs.Config) {
	servicesToStop := []constants.ToolType{
		constants.SuperNode,
		constants.RQService,
		constants.WalletNode,
		constants.DDImgService,
		constants.DDService,
		constants.PastelD,
		constants.Hermes,
		constants.Bridge,
	}
	_ = stopServices(ctx, servicesToStop, config)
	log.WithContext(ctx).Info("All stopped successfully")
}

func stopRQServiceSubCommand(ctx context.Context, config *configs.Config) {
	_ = stopServices(ctx, []constants.ToolType{constants.RQService}, config)
}

func stopDDServiceSubCommand(ctx context.Context, config *configs.Config) {
	_ = stopServices(ctx, []constants.ToolType{constants.DDService}, config)
}

func stopDDImgServer(ctx context.Context, config *configs.Config) {
	_ = stopServices(ctx, []constants.ToolType{constants.DDImgService}, config)
}

func stopWNServiceSubCommand(ctx context.Context, config *configs.Config) {
	_ = stopServices(ctx, []constants.ToolType{constants.WalletNode}, config)
}

func stopSNServiceSubCommand(ctx context.Context, config *configs.Config) {
	_ = stopServices(ctx, []constants.ToolType{constants.SuperNode}, config)
}

func stopHermesService(ctx context.Context, config *configs.Config) {
	_ = stopServices(ctx, []constants.ToolType{constants.Hermes}, config)
}

func stopBridgeService(ctx context.Context, config *configs.Config) {
	_ = stopServices(ctx, []constants.ToolType{constants.Bridge}, config)
}

func stopPatelCLI(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Stopping Pasteld")
	_, err := GetPastelInfo(ctx, config)
	if err != nil {
		log.WithContext(ctx).Info("Pasteld is not running!")
		return nil
	}
	err = StopPastelDAndWait(ctx, config)
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to stop Pasteld")
	}
	if CheckProcessRunning(constants.PastelD) {
		log.WithContext(ctx).Warn("Failed to stop Pasteld")
		return errors.Errorf("Failed to stop Pasteld")
	}
	log.WithContext(ctx).Info("Pasteld stopped")
	return nil
}

func stopServicesWithConfirmation(ctx context.Context, config *configs.Config, services []constants.ToolType) error {
	var servicesToStop []constants.ToolType
	for _, service := range services {
		log.WithContext(ctx).Infof("Stopping %s...", string(service))
		if service == constants.PastelD {
			_, err := GetPastelInfo(ctx, config)
			if err == nil { // this means the pastel-cli is running
				servicesToStop = append(servicesToStop, service)
			}
			continue
		}
		pid, err := GetRunningProcessPid(service)
		if err != nil {
			log.WithContext(ctx).Error(fmt.Sprintf("Failed validating if '%v' service is running: %v", service, err))
			return err
		}
		if pid != 0 {
			servicesToStop = append(servicesToStop, service)
		}
		log.WithContext(ctx).Infof("%s stopped", string(service))
	}
	if len(servicesToStop) == 0 {
		return nil
	}
	if !config.Force {
		question := fmt.Sprintf("To perform this update, we need to kill these services: %v. Is this ok? Y/N", servicesToStop)
		ok, _ := AskUserToContinue(ctx, question)
		if !ok {
			return fmt.Errorf("user did not accept confirmation to stop services")
		}
	}
	return stopServices(ctx, servicesToStop, config)
}

func stopServices(ctx context.Context, services []constants.ToolType, config *configs.Config) error {
	servicesEnabled := false
	sm, err := NewServiceManager(utils.GetOS(), config.Configurer.DefaultHomeDir())
	if err != nil {
		log.WithContext(ctx).Warnf("services not enabled for your OS %v", utils.GetOS())
	} else {
		servicesEnabled = true
	}
	for _, service := range services {
		log.WithContext(ctx).Infof("Stopping %s service...", string(service))

		if servicesEnabled {
			isRegistered := sm.IsRegistered(ctx, config, service)
			if !isRegistered {
				log.WithContext(ctx).Infof("skipping stop service because %v is not a registered service", service)
			} else {
				log.WithContext(ctx).Infof("Try to stop %s as system service...", string(service))
				err := sm.StopService(ctx, config, service)
				if err != nil {
					log.WithContext(ctx).Errorf("unable to stop %s as system service: %v, will try to kill it", string(service), err)
				} else {
					continue
				}
			}
		}

		switch service {
		case constants.PastelD:
			err = stopPatelCLI(ctx, config)
			if err != nil {
				log.WithContext(ctx).Errorf("unable to stop pasteld: %v", err)
				return err
			}
		case constants.DDService:
			searchTerm := constants.DupeDetectionExecFileName
			pid, err := FindRunningProcessPid(ctx, searchTerm)
			if err != nil {
				log.WithContext(ctx).Errorf("unable to find service %v to stop it: %v", service, err)
				return err
			}
			log.WithContext(ctx).Infof("Killing process: %v\n", pid)
			err = KillProcessByPid(ctx, pid)
			if err != nil {
				log.WithContext(ctx).Errorf("unable to stop service %v: %v", service, err)
				return err
			}
		default:
			process := service
			override, ok := serviceToProcessOverrides[string(service)]
			if ok {
				process = constants.ToolType(override)
			}
			err := KillProcess(ctx, process) // kill process in case the service wasn't registered
			if err != nil {
				log.WithContext(ctx).Error(fmt.Sprintf("unable to kill process %v: %v", service, err))
				return err
			}
		}
		log.WithContext(ctx).Infof("%s service stopped", string(service))
	}
	return nil
}
