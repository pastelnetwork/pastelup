package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/pastel-utility/constants"
)

type stopCommand uint8

const (
	nodeStop stopCommand = iota
	walletStop
	superNodeStop
	superNodeRemoteStop
	allStop
	rqServiceStop
	ddServiceStop
	wnServiceStop
	snServiceStop
)

var (
	// Stop commands
	stopCmdName = map[stopCommand]string{
		nodeStop:            "node",
		walletStop:          "walletnode",
		superNodeStop:       "supernode",
		superNodeRemoteStop: "remote",
		allStop:             "all",
		rqServiceStop:       "rq-service",
		ddServiceStop:       "dd-service",
		wnServiceStop:       "walletnode-service",
		snServiceStop:       "supernode-service",
	}
)

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
	commandMessage := "Stop " + commandName

	if stopCommand >= allStop {
		commandMessage += " only"
	}

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

	log.WithContext(ctx).Info("End successfully")
}

func runStopWalletSubCommand(ctx context.Context, config *configs.Config) {

	// *************  Kill process wallet node  *************
	stopService(ctx, constants.WalletNode, config)

	// *************  Kill process rqservice  *************
	stopService(ctx, constants.RQService, config)

	// *************  Stop pasteld node  *************
	stopPatelCLI(ctx, config)

	log.WithContext(ctx).Info("Walletnode stopped successfully")
}

func runStopSuperNodeSubCommand(ctx context.Context, config *configs.Config) {

	// *************  Kill process super node  *************
	stopService(ctx, constants.SuperNode, config)

	// *************  Kill process rqservice  *************
	stopService(ctx, constants.RQService, config)

	// *************  Kill process dd-service  *************
	stopDDService(ctx, config)
	//stopService(ctx, constants.DDService, config)

	// *************  Kill process dd-img-server  *************
	stopService(ctx, constants.DDImgService, config)

	// *************  Stop pasteld node  *************
	stopPatelCLI(ctx, config)

	log.WithContext(ctx).Info("Suppernode stopped successfully")
}

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

	// *************  Kill process super node  *************
	stopService(ctx, constants.SuperNode, config)

	// *************  Kill process wallet node  *************
	stopService(ctx, constants.WalletNode, config)

	// *************  Kill process rqservice  *************
	stopService(ctx, constants.RQService, config)

	// *************  Kill process dd-service  *************
	stopDDService(ctx, config)

	// *************  Kill process dd-img-server  *************
	stopService(ctx, constants.DDImgService, config)

	// *************  Stop pasteld node  *************
	stopPatelCLI(ctx, config)

	log.WithContext(ctx).Info("All stopped successfully")
}

func stopRQServiceSubCommand(ctx context.Context, config *configs.Config) {
	stopService(ctx, constants.RQService, config)
}

func stopDDServiceSubCommand(ctx context.Context, config *configs.Config) {
	stopDDService(ctx, config)
}

func stopWNServiceSubCommand(ctx context.Context, config *configs.Config) {
	stopService(ctx, constants.WalletNode, config)
}

func stopSNServiceSubCommand(ctx context.Context, config *configs.Config) {
	stopService(ctx, constants.SuperNode, config)
}

func stopPatelCLI(ctx context.Context, config *configs.Config) {

	log.WithContext(ctx).Info("Stopping Pasteld")
	if err := stopSystemdService(ctx, string(constants.PastelD), config); err != nil {
		// Check if pasteld is already running
		if _, err = RunPastelCLI(ctx, config, "getinfo"); err != nil {
			log.WithContext(ctx).Info("Pasteld is not running!")
			return
		}

		if _, err := RunPastelCLI(ctx, config, "stop"); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to run '%s/pastel-cli stop'", config.WorkingDir)
		}
		time.Sleep(5 * time.Second)
		if CheckProcessRunning(constants.PastelD) {
			log.WithContext(ctx).Warn("Failed to stop pasted using 'pastel-cli stop'")
		} else {
			log.WithContext(ctx).Info("Pasteld stopped")
		}
	}
}

func stopService(ctx context.Context, tool constants.ToolType, config *configs.Config) {

	log.WithContext(ctx).Infof("Stopping %s process", tool)

	// Check if service is installed and running, then check if it is running
	if err := stopSystemdService(ctx, string(tool), config); err != nil {
		if err := KillProcess(ctx, tool); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to kill %s", tool)
		}
		if CheckProcessRunning(tool) {
			log.WithContext(ctx).Warnf("Failed to kill %s, it is still running", tool)
		} else {
			log.WithContext(ctx).Infof("%s stopped", tool)
		}
	}

	log.WithContext(ctx).Infof("The %s process ended", tool)
}

func stopDDService(ctx context.Context, config *configs.Config) {
	log.WithContext(ctx).Info("Stopping dd-service process")

	if err := stopSystemdService(ctx, string(constants.DDService), config); err != nil {
		if pid, err := FindRunningProcessPid(constants.DupeDetectionExecFileName); err != nil {
			log.WithContext(ctx).Infof("dd-service is not running")
		} else if pid != 0 {
			if err := KillProcessByPid(ctx, pid); err != nil {
				log.WithContext(ctx).WithError(err).Error("Failed to kill dd-service'")
			} else {
				log.WithContext(ctx).Info("The dd-service process ended.")
			}
		}
	}
}

func stopServicesWithConfirmation(ctx context.Context, config *configs.Config, services []constants.ToolType) error {
	servicesToStop := []constants.ToolType{}
	for _, service := range services {
		if service == constants.PastelD {
			_, err := RunPastelCLI(ctx, config, "getinfo")
			if err == nil { // this means the pastel-cli is running
				servicesToStop = append(servicesToStop, service)
			}
			continue
		}
		pid, err := FindRunningProcessPid(string(service))
		if err != nil {
			log.WithContext(ctx).Info(fmt.Sprintf("Failed validating if '%v' service is running: %v", service, err))
			return err
		}
		if pid != 0 {
			servicesToStop = append(servicesToStop, service)
		}
	}
	question := fmt.Sprintf("To perform this update, we need to kill these services: %v. Is this ok? (y/n)", servicesToStop)
	ok, _ := AskUserToContinue(ctx, question)
	if !ok {
		return fmt.Errorf("user did not accept confirmation to stop services")
	}
	for _, service := range servicesToStop {
		if service == constants.PastelD {
			stopPatelCLI(ctx, config)
		} else {
			stopService(ctx, service, config)
		}
	}
	return nil
}
