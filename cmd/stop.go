package cmd

import (
	"context"
	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/pastel-utility/constants"
	"os"
	"time"
)

type stopCommand uint8

const (
	nodeStop stopCommand = iota
	walletStop
	superNodeStop
	allStop
)

func setupStopSubCommand(config *configs.Config,
	stopCommand stopCommand,
	f func(context.Context, *configs.Config) error,
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

func setupStopCommand() *cli.Command {
	config := configs.InitConfig()

	stopNodeSubCommand := setupStopSubCommand(config, nodeStop, runStopNodeSubCommand)
	stopWalletSubCommand := setupStopSubCommand(config, walletStop, runStopWalletSubCommand)
	stopSuperNodeSubCommand := setupStopSubCommand(config, superNodeStop, runStopSuperNodeSubCommand)
	stopallSubCommand := setupStopSubCommand(config, allStop, runStopAllSubCommand)

	stopCommand := cli.NewCommand("stop")
	stopCommand.SetUsage(blue("Performs stop of the system for both WalletNode and SuperNodes"))
	stopCommand.AddSubcommands(stopNodeSubCommand)
	stopCommand.AddSubcommands(stopWalletSubCommand)
	stopCommand.AddSubcommands(stopSuperNodeSubCommand)
	stopCommand.AddSubcommands(stopallSubCommand)

	return stopCommand
}

func runStopNodeSubCommand(ctx context.Context, config *configs.Config) error {

	stopPatelCLI(ctx, config)

	log.WithContext(ctx).Info("End successfully")
	return nil
}

func runStopWalletSubCommand(ctx context.Context, config *configs.Config) error {

	// *************  Kill process wallet node  *************
	stopService(ctx, constants.WalletNode)

	// *************  Kill process rqservice  *************
	stopService(ctx, constants.RQService)

	// *************  Stop pasteld node  *************
	stopPatelCLI(ctx, config)

	log.WithContext(ctx).Info("Walletnode stopped successfully")
	return nil
}

func runStopSuperNodeSubCommand(ctx context.Context, config *configs.Config) error {

	// *************  Kill process super node  *************
	stopService(ctx, constants.SuperNode)

	// *************  Kill process rqservice  *************
	stopService(ctx, constants.RQService)

	// *************  Kill process dd-service  *************
	stopDDService(ctx)

	// *************  Stop pasteld node  *************
	stopPatelCLI(ctx, config)

	log.WithContext(ctx).Info("Suppernode stopped successfully")
	return nil
}

func runStopAllSubCommand(ctx context.Context, config *configs.Config) error {

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

	return nil
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
	if pid, err := FindRunningProcessPid(constants.DupeDetectionExecName); err != nil {
		log.WithContext(ctx).Infof("dd-service is not running")
	} else if pid != 0 {
		if err := KillProcessByPid(ctx, pid); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to kill dd-service'")
		} else {
			log.WithContext(ctx).Info("The dd-service process ended.")
		}
	}
}
