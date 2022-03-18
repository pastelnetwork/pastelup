package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

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
	superNodeRemoteStop
	allStop
	ddServiceStop
	rqServiceStop
	wnServiceStop
	snServiceStop
)

var (
	stopCmdName = map[stopCommand]string{
		nodeStop:            "node",
		walletStop:          "walletnode",
		superNodeStop:       "supernode",
		superNodeRemoteStop: "remote",
		allStop:             "all",
		ddServiceStop:       "dd-service",
		rqServiceStop:       "rq-service",
		wnServiceStop:       "walletnode-service",
		snServiceStop:       "supernode-service",
	}
	stopCmdMessage = map[stopCommand]string{
		nodeStop:            "Stop node",
		walletStop:          "Stop Walletnode",
		superNodeStop:       "Stop Supernode",
		superNodeRemoteStop: "Stop Supernode on Remote Host",
		allStop:             "Stop all Pastel services",
		ddServiceStop:       "Stop Dupe Detection service only",
		rqServiceStop:       "Stop RaptorQ service only",
		wnServiceStop:       "Stop Walletnode service only",
		snServiceStop:       "Stop Supernode service only",
	}
)

var serviceToProcessOverrides = map[string]string{
	string(constants.DDService): constants.DupeDetectionExecFileName,
}

func setupStopSubCommand(config *configs.Config,
	stopCommand stopCommand,
	f func(context.Context, *configs.Config),
) *cli.Command {

	commonFlags := []*cli.Flag{
		cli.NewFlag("dir", &config.PastelExecDir).SetAliases("d").
			SetUsage(green("Optional, Location of pastel node directory")).SetValue(config.Configurer.DefaultPastelExecutableDir()),
		cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
			SetUsage(green("Optional, location of working directory")).SetValue(config.Configurer.DefaultWorkingDir()),
	}

	if stopCommand == superNodeRemoteStop {
		remoteFlags := []*cli.Flag{
			cli.NewFlag("ssh-ip", &config.RemoteIP).
				SetUsage(red("Required, SSH address of the remote host")).SetRequired(),
			cli.NewFlag("ssh-port", &config.RemotePort).
				SetUsage(yellow("Optional, SSH port of the remote host, default is 22")).SetValue(22),
			cli.NewFlag("ssh-user", &config.RemoteUser).
				SetUsage(yellow("Optional, Username of user at remote host")),
			cli.NewFlag("ssh-key", &config.RemoteSSHKey).
				SetUsage(yellow("Optional, Path to SSH private key for SSH Key Authentication")),
		}

		commonFlags = append(commonFlags, remoteFlags...)
	}

	commandName := stopCmdName[stopCommand]
	commandMessage := stopCmdMessage[stopCommand]

	subCommand := cli.NewCommand(commandName)
	subCommand.SetUsage(cyan(commandMessage))
	subCommand.AddFlags(commonFlags...)

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

			log.WithContext(ctx).Info("Stopping...")
			f(ctx, config)
			log.WithContext(ctx).Info("Finished successfully!")
			return nil
		})
	}
	return subCommand
}

func setupStopCommand() *cli.Command {
	config := configs.InitConfig()

	stopNodeSubCommand := setupStopSubCommand(config, nodeStop, runStopNodeSubCommand)
	stopWalletSubCommand := setupStopSubCommand(config, walletStop, runStopWalletSubCommand)
	stopSuperNodeRemoteSubCommand := setupStopSubCommand(config, superNodeRemoteStop, runStopSuperNodeRemoteSubCommand)
	stopSuperNodeSubCommand := setupStopSubCommand(config, superNodeStop, runStopSuperNodeSubCommand)
	stopSuperNodeSubCommand.AddSubcommands(stopSuperNodeRemoteSubCommand)
	stopallSubCommand := setupStopSubCommand(config, allStop, runStopAllSubCommand)

	stopRQSubCommand := setupStopSubCommand(config, rqServiceStop, stopRQServiceSubCommand)
	stopDDSubCommand := setupStopSubCommand(config, ddServiceStop, stopDDServiceSubCommand)
	stopWNSubCommand := setupStopSubCommand(config, wnServiceStop, stopWNServiceSubCommand)
	stopSNSubCommand := setupStopSubCommand(config, snServiceStop, stopSNServiceSubCommand)

	stopCommand := cli.NewCommand("stop")
	stopCommand.SetUsage(blue("Performs stop of the system for both WalletNode and SuperNodes"))
	stopCommand.AddSubcommands(stopNodeSubCommand)
	stopCommand.AddSubcommands(stopWalletSubCommand)
	stopCommand.AddSubcommands(stopSuperNodeSubCommand)
	stopCommand.AddSubcommands(stopallSubCommand)

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
		constants.PastelD}
	stopServices(ctx, servicesToStop, config)
	log.WithContext(ctx).Info("Walletnode stopped successfully")
}

func runStopSuperNodeSubCommand(ctx context.Context, config *configs.Config) {
	servicesToStop := []constants.ToolType{
		constants.SuperNode,
		constants.RQService,
		constants.DDImgService,
		constants.DDService,
		constants.PastelD}
	stopServices(ctx, servicesToStop, config)
	log.WithContext(ctx).Info("Suppernode stopped successfully")
}

// special handling for remote command
func runStopSuperNodeRemoteSubCommand(ctx context.Context, config *configs.Config) {
	// Connect to remote
	client, err := prepareRemoteSession(ctx, config)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to prepare remote session")
		return
	}
	defer client.Close()
	// Execute stop remote supernode
	log.WithContext(ctx).Info("Executing stop remote supernode...")
	stopOptions := ""
	if len(config.PastelExecDir) > 0 {
		stopOptions = fmt.Sprintf("--dir %s", config.PastelExecDir)
	}
	if len(config.WorkingDir) > 0 {
		stopOptions = fmt.Sprintf("%s --work-dir %s", stopOptions, config.WorkingDir)
	}
	stopSuperNodeCmd := fmt.Sprintf("%s stop supernode %s", constants.RemotePastelupPath, stopOptions)
	if err := client.ShellCmd(ctx, stopSuperNodeCmd); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to execute stop supernode on remote host")
		return
	}

	log.WithContext(ctx).Info("Suppernode stopped successfully")
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
	stopServices(ctx, []constants.ToolType{constants.SuperNode}, config)
}

func stopPatelCLI(ctx context.Context, config *configs.Config) {
	log.WithContext(ctx).Info("Stopping Pasteld")
	if _, err := RunPastelCLI(ctx, config, "getinfo"); err != nil {
		log.WithContext(ctx).Info("Pasteld is not running!")
		return
	}
	if _, err := RunPastelCLI(ctx, config, "stop"); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to run '%s/pastel-cli stop'", config.WorkingDir)
	}
	time.Sleep(5 * time.Second)
	if CheckProcessRunning(constants.PastelD) {
		log.WithContext(ctx).Warn("Failed to stop pasted using 'pastel-cli stop'")
		return
	}
	log.WithContext(ctx).Info("Pasteld stopped")
}

func stopServicesWithConfirmation(ctx context.Context, config *configs.Config, services []constants.ToolType) error {
	servicesToStop := []constants.ToolType{}
	for _, service := range services {
		log.WithContext(ctx).Infof("Stopping %s...", string(service))
		if service == constants.PastelD {
			_, err := RunPastelCLI(ctx, config, "getinfo")
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
	question := fmt.Sprintf("To perform this update, we need to kill these services: %v. Is this ok? Y/N", servicesToStop)
	ok, _ := AskUserToContinue(ctx, question)
	if !ok {
		return fmt.Errorf("user did not accept confirmation to stop services")
	}
	return stopServices(ctx, servicesToStop, config)
}

func stopServices(ctx context.Context, services []constants.ToolType, config *configs.Config) error {
	servicesEnabled := false
	sm, err := servicemanager.New(utils.GetOS(), config.Configurer.DefaultHomeDir())
	if err != nil {
		log.WithContext(ctx).Warnf("services not enabled for your OS %v", utils.GetOS())
	} else {
		servicesEnabled = true
	}
	for _, service := range services {
		log.WithContext(ctx).Infof("Stopping %s service...", string(service))
		if service == constants.PastelD {
			stopPatelCLI(ctx, config)
		} else {
			if servicesEnabled {
				err = sm.StopService(ctx, service)
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
