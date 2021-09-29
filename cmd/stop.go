package cmd

import (
	"context"
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
	allStop
	rqServiceStop
	ddServiceStop
	wnServiceStop
	snServiceStop
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

	var commandName, commandMessage string

	switch stopCommand {
	case nodeStop:
		commandName = "node"
		commandMessage = "Stop node"
	case walletStop:
		commandName = string(constants.WalletNode)
		commandMessage = "Stop walletnode"
	case superNodeStop:
		commandName = string(constants.SuperNode)
		commandMessage = "Stop supernode"
	case allStop:
		commandName = "all"
		commandMessage = "Stop all"

	case rqServiceStop:
		commandName = "rq-service"
		commandMessage = "Stop rq-service"
	case ddServiceStop:
		commandName = "dd-service"
		commandMessage = "Stop dd-service"
	case wnServiceStop:
		commandName = "walletnode-service"
		commandMessage = "Stop walletnode service"
	case snServiceStop:
		commandName = "supernode-service"
		commandMessage = "Stop supernode service"

	default:
		commandName = "all"
		commandMessage = "Stop all"
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
	stopSuperNodeSubCommand := setupStopSubCommand(config, superNodeStop, runStopSuperNodeSubCommand)
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
	stopService(ctx, constants.WalletNode)

	// *************  Kill process rqservice  *************
	stopService(ctx, constants.RQService)

	// *************  Stop pasteld node  *************
	stopPatelCLI(ctx, config)

	log.WithContext(ctx).Info("Walletnode stopped successfully")
}

func runStopSuperNodeSubCommand(ctx context.Context, config *configs.Config) {

	// *************  Kill process super node  *************
	stopService(ctx, constants.SuperNode)

	// *************  Kill process rqservice  *************
	stopService(ctx, constants.RQService)

	// *************  Kill process dd-service  *************
	stopDDService(ctx)

	// *************  Stop pasteld node  *************
	stopPatelCLI(ctx, config)

	log.WithContext(ctx).Info("Suppernode stopped successfully")
}

func runStopAllSubCommand(ctx context.Context, config *configs.Config) {

	// *************  Kill process super node  *************
	stopService(ctx, constants.SuperNode)

	// *************  Kill process wallet node  *************
	stopService(ctx, constants.WalletNode)

	// *************  Kill process rqservice  *************
	stopService(ctx, constants.RQService)

	// *************  Kill process dd-service  *************
	stopDDService(ctx)

	// *************  Stop pasteld node  *************
	stopPatelCLI(ctx, config)

	log.WithContext(ctx).Info("All stopped successfully")
}

func stopRQServiceSubCommand(ctx context.Context, _ *configs.Config) {
	stopService(ctx, constants.RQService)
}

func stopDDServiceSubCommand(ctx context.Context, _ *configs.Config) {
	stopDDService(ctx)
}

func stopWNServiceSubCommand(ctx context.Context, _ *configs.Config) {
	stopService(ctx, constants.WalletNode)
}

func stopSNServiceSubCommand(ctx context.Context, _ *configs.Config) {
	stopService(ctx, constants.SuperNode)
}

func stopPatelCLI(ctx context.Context, config *configs.Config) {
	log.WithContext(ctx).Info("Stopping Pasteld")
	if _, err := RunPastelCLI(ctx, config, "stop"); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to run '%s/pastel-cli stop'", config.WorkingDir)
	}
	time.Sleep(1 * time.Second)
	if CheckProcessRunning(constants.PastelD) {
		log.WithContext(ctx).Warn("Failed to stop pasted using 'pastel-cli stop'")
	} else {
		log.WithContext(ctx).Info("Pasteld stopped")
	}
}

func stopService(ctx context.Context, tool constants.ToolType) {

	log.WithContext(ctx).Infof("Stopping %s process", tool)
	if err := KillProcess(ctx, tool); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to kill %s", tool)
	}
	if CheckProcessRunning(tool) {
		log.WithContext(ctx).Warnf("Failed to kill %s, it is still running", tool)
	} else {
		log.WithContext(ctx).Infof("%s stopped", tool)
	}

	log.WithContext(ctx).Infof("The %s process ended", tool)
}

func stopDDService(ctx context.Context) {
	log.WithContext(ctx).Info("Stopping dd-service process")
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
