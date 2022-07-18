package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pastelnetwork/pastelup/configs"
	"github.com/pastelnetwork/pastelup/constants"
	"github.com/pastelnetwork/pastelup/servicemanager"
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
	wnServiceStop
	snServiceStop
	remoteStop
)

var (
	stopCmdName = map[stopCommand]string{
		nodeStop:      "node",
		walletStop:    "walletnode",
		superNodeStop: "supernode",
		allStop:       "all",
		ddServiceStop: "dd-service",
		rqServiceStop: "rq-service",
		wnServiceStop: "walletnode-service",
		snServiceStop: "supernode-service",
		remoteStop:    "remote",
	}
	stopCmdMessage = map[stopCommand]string{
		nodeStop:      "Stop node",
		walletStop:    "Stop Walletnode",
		superNodeStop: "Stop Supernode",
		allStop:       "Stop all Pastel services",
		ddServiceStop: "Stop Dupe Detection service only",
		rqServiceStop: "Stop RaptorQ service only",
		wnServiceStop: "Stop Walletnode service only",
		snServiceStop: "Stop Supernode service only",
		remoteStop:    "Stop on Remote Host",
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
			ParsePastelConf(ctx, config)
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

	stopNodeSubCommand.AddSubcommands(setupStopSubCommand(config, nodeStop, true, runStopNodeRemoteSubCommand))
	stopWalletSubCommand.AddSubcommands(setupStopSubCommand(config, walletStop, true, runStopWalletNodeRemoteSubCommand))
	stopSuperNodeSubCommand.AddSubcommands(setupStopSubCommand(config, superNodeStop, true, runStopSuperNodeRemoteSubCommand))
	stopAllSubCommand.AddSubcommands(setupStopSubCommand(config, allStop, true, runStopAllRemoteSubCommand))
	stopRQSubCommand.AddSubcommands(setupStopSubCommand(config, rqServiceStop, true, runStopRQServiceRemoteSubCommand))
	stopDDSubCommand.AddSubcommands(setupStopSubCommand(config, ddServiceStop, true, runStopDDServiceRemoteSubCommand))
	stopWNSubCommand.AddSubcommands(setupStopSubCommand(config, wnServiceStop, true, runStopWNServiceRemoteSubCommand))
	stopSNSubCommand.AddSubcommands(setupStopSubCommand(config, snServiceStop, true, runStopSNServiceRemoteSubCommand))

	stopCommand := cli.NewCommand("stop")
	stopCommand.SetUsage(blue("Performs stop of the system for both WalletNode and SuperNodes"))
	stopCommand.AddSubcommands(stopNodeSubCommand)
	stopCommand.AddSubcommands(stopWalletSubCommand)
	stopCommand.AddSubcommands(stopSuperNodeSubCommand)
	stopCommand.AddSubcommands(stopAllSubCommand)

	stopCommand.AddSubcommands(stopRQSubCommand)
	stopCommand.AddSubcommands(stopDDSubCommand)
	stopCommand.AddSubcommands(stopWNSubCommand)
	stopCommand.AddSubcommands(stopSNSubCommand)

	return stopCommand
}

func runStopNodeSubCommand(ctx context.Context, config *configs.Config) {
	stopPatelCLI(ctx, config)
	log.WithContext(ctx).Info("Stopped node successfully")
}

func runStopWalletSubCommand(ctx context.Context, config *configs.Config) {
	servicesToStop := []constants.ToolType{
		constants.WalletNode,
		constants.RQService,
		constants.Bridge,
		constants.PastelD}
	stopServices(ctx, servicesToStop, config)
	log.WithContext(ctx).Info("Walletnode stopped successfully")
}

func runStopSuperNodeSubCommand(ctx context.Context, config *configs.Config) {
	servicesToStop := []constants.ToolType{
		constants.Hermes,
		constants.SuperNode,
		constants.RQService,
		constants.DDImgService,
		constants.DDService,
		constants.PastelD}
	stopServices(ctx, servicesToStop, config)
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
func runStopWNServiceRemoteSubCommand(ctx context.Context, config *configs.Config) {
	runRemoteStop(ctx, config, "walletnode-service")
}
func runStopSNServiceRemoteSubCommand(ctx context.Context, config *configs.Config) {
	runRemoteStop(ctx, config, "supernode-service")
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
		constants.PastelD}
	stopServices(ctx, servicesToStop, config)
	log.WithContext(ctx).Info("All stopped successfully")
}

func stopRQServiceSubCommand(ctx context.Context, config *configs.Config) {
	stopServices(ctx, []constants.ToolType{constants.RQService}, config)
}

func stopDDServiceSubCommand(ctx context.Context, config *configs.Config) {
	stopServices(ctx, []constants.ToolType{constants.DDService}, config)
}

func stopWNServiceSubCommand(ctx context.Context, config *configs.Config) {
	stopServices(ctx, []constants.ToolType{constants.WalletNode}, config)
}

func stopSNServiceSubCommand(ctx context.Context, config *configs.Config) {
	stopServices(ctx, []constants.ToolType{constants.Hermes, constants.SuperNode}, config)
}

func stopPatelCLI(ctx context.Context, config *configs.Config) {
	log.WithContext(ctx).Info("Stopping Pasteld")
	_, err := GetPastelInfo(ctx, config)
	if err != nil {
		log.WithContext(ctx).Info("Pasteld is not running!")
		return
	}
	err = StopPastelDAndWait(ctx, config)
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to stop Pasteld")
	}
	if CheckProcessRunning(constants.PastelD) {
		log.WithContext(ctx).Warn("Failed to stop Pasteld")
		return
	}
	log.WithContext(ctx).Info("Pasteld stopped")
}

func stopServicesWithConfirmation(ctx context.Context, config *configs.Config, services []constants.ToolType) error {
	servicesToStop := []constants.ToolType{}
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
	sm, err := servicemanager.New(utils.GetOS(), config.Configurer.DefaultHomeDir())
	if err != nil {
		log.WithContext(ctx).Warnf("services not enabled for your OS %v", utils.GetOS())
	} /*else {
		servicesEnabled = true
	}*/
	for _, service := range services {
		log.WithContext(ctx).Infof("Stopping %s service...", string(service))
		switch service {
		case constants.PastelD:
			stopPatelCLI(ctx, config)
		case constants.DDService:
			searchTerm := constants.DupeDetectionExecFileName
			pid, err := FindRunningProcessPid(string(searchTerm))
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
			if servicesEnabled {
				err := sm.StopService(ctx, service)
				if err != nil {
					log.WithContext(ctx).Errorf("unable to stop service %v: %v", service, err)
					return err
				}
			}
			process := service
			override, ok := serviceToProcessOverrides[string(service)]
			if ok {
				process = constants.ToolType(override)
			}
			err := KillProcess(ctx, process) // kill process incase the service wasnt registered
			if err != nil {
				log.WithContext(ctx).Error(fmt.Sprintf("unable to kill process %v: %v", service, err))
				return err
			}
		}
		log.WithContext(ctx).Infof("%s service stopped", string(service))
	}
	return nil
}
