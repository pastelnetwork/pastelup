package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/pastel-utility/constants"
	"github.com/pastelnetwork/pastel-utility/utils"
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
			SetUsage(green("Location where to create pastel node directory")).SetValue(config.PastelExecDir),
		cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
			SetUsage(green("Location where to create working directory")).SetValue(config.WorkingDir),
	}

	var commandName, commandMessage string
	var commandFlags []*cli.Flag = commonFlags

	switch stopCommand {
	case nodeStop:
		commandName = "node"
		commandMessage = "Stop node"
	case walletStop:
		commandName = "walletnode"
		commandMessage = "Stop walletnode"
	case superNodeStop:
		commandName = "supernode"
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
	config := configs.GetConfig()

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

func runStopAllSubCommand(ctx context.Context, config *configs.Config) error {

	var err error
	// *************  Kill process super node  *************
	log.WithContext(ctx).Info("Start stopping supernode process")
	if _, err = processKill(ctx, config, constants.SuperNode); err != nil {
		return err
	}
	log.WithContext(ctx).Info("The Supernode stopped.")

	// *************  Kill process wallet node  *************
	log.WithContext(ctx).Info("Start stopping walletnode process")
	if _, err = processKill(ctx, config, constants.WalletNode); err != nil {
		return err
	}
	log.WithContext(ctx).Info("The Walletnode stopped.")

	// *************  Kill process wallet node  *************
	log.WithContext(ctx).Info("Start stopping rqservice process")
	if _, err = processKill(ctx, config, constants.RQService); err != nil {
		return err
	}
	log.WithContext(ctx).Info("The rqservice stopped.")

	// TODO: Implement Stop node command
	log.WithContext(ctx).Info("Pasteld process kill starting.")
	if _, err = stopPatelCLI(ctx, config); err != nil {
		return err
	}
	log.WithContext(ctx).Info("Pasteld process ended.")

	time.Sleep(10000 * time.Millisecond)

	log.WithContext(ctx).Info("End successfully")

	return nil
}

func runStopNodeSubCommand(ctx context.Context, config *configs.Config) error {

	log.WithContext(ctx).Info("Start stopping Pasteld process")
	if _, err := stopPatelCLI(ctx, config); err != nil {
		return err
	}
	log.WithContext(ctx).Info("Pasteld process ended.")
	log.WithContext(ctx).Info("End successfully")

	return nil
}

func runStopWalletSubCommand(ctx context.Context, config *configs.Config) error {

	var err error

	// *************  Kill process wallet node  *************
	log.WithContext(ctx).Info("Start stopping Walletnode process")

	if _, err = processKill(ctx, config, constants.WalletNode); err != nil {
		return err
	}
	log.WithContext(ctx).Info("Walletnode process ended.")

	// *************  Kill process rqservice  *************
	log.WithContext(ctx).Info("Start stopping rqservice process")
	if _, err = processKill(ctx, config, constants.RQService); err != nil {
		return err
	}
	log.WithContext(ctx).Info("The rqservice process ended.")

	// *************  Stop pasteld  *************
	log.WithContext(ctx).Info("Start stopping Pasteld process")

	if _, err = stopPatelCLI(ctx, config); err != nil {
		return err
	}
	log.WithContext(ctx).Info("Pasteld process ended.")

	log.WithContext(ctx).Info("End successfully")

	return nil
}

func runStopSuperNodeSubCommand(ctx context.Context, config *configs.Config) error {
	var err error

	// *************  Kill process super node  *************
	log.WithContext(ctx).Info("Start stopping Supernode process")

	if _, err = processKill(ctx, config, constants.SuperNode); err != nil {
		return err
	}
	log.WithContext(ctx).Info("Supernode process ended.")

	// *************  Kill process rqservice  *************
	log.WithContext(ctx).Info("Start stopping rqservice process")

	if _, err = processKill(ctx, config, constants.RQService); err != nil {
		return err
	}
	log.WithContext(ctx).Info("The rqservice process ended.")

	// *************  Stop pastel super node   *************
	log.WithContext(ctx).Info("Start stopping Pasteld process")

	if _, err = stopPatelCLI(ctx, config); err != nil {
		return err
	}
	log.WithContext(ctx).Info("Pasteld process ended.")

	return nil
}

func stopPatelCLI(ctx context.Context, config *configs.Config) (output string, err error) {
	if _, err = runPastelCLI(ctx, config, "stop"); err != nil {
		return "", err
	}

	return "", nil
}

func processKill(ctx context.Context, config *configs.Config, toolType constants.ToolType) (output string, err error) {
	var pID string
	var processID int

	execPath := ""
	execName := ""
	switch toolType {
	case constants.WalletNode:
		execPath = filepath.Join(config.PastelExecDir, constants.WalletNodeExecName[utils.GetOS()])
		execName = constants.WalletNodeExecName[utils.GetOS()]
	case constants.SuperNode:
		execPath = filepath.Join(config.PastelExecDir, constants.SuperNodeExecName[utils.GetOS()])
		execName = constants.SuperNodeExecName[utils.GetOS()]
	case constants.RQService:
		execPath = filepath.Join(config.PastelExecDir, constants.PastelRQServiceExecName[utils.GetOS()])
		execName = constants.PastelRQServiceExecName[utils.GetOS()]
	default:
		execPath = filepath.Join(config.PastelExecDir, constants.PastelRQServiceExecName[utils.GetOS()])
		execName = constants.PastelRQServiceExecName[utils.GetOS()]
	}

	if utils.GetOS() == constants.Windows {
		RunCMDWithInteractive("Taskkill", "/IM", execName, "/F")

	} else {
		matches, _ := filepath.Glob("/proc/*/exe")
		for _, file := range matches {
			target, _ := os.Readlink(file)
			if len(target) > 0 {
				if target == execPath {
					split := strings.Split(file, "/")

					pID = split[len(split)-2]
					processID, err = strconv.Atoi(pID)
					proc, err := os.FindProcess(processID)
					if err != nil {
						log.WithContext(ctx).Errorf("Can not find process %s", execName)
					}
					// Kill the process
					proc.Kill()

					break
				}
			}
		}

	}

	return
}
