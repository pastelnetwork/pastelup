package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"

	"github.com/pastelnetwork/pastelup/common/cli"
	"github.com/pastelnetwork/pastelup/common/log"
	"github.com/pastelnetwork/pastelup/configs"
	"github.com/pastelnetwork/pastelup/constants"
	"github.com/pastelnetwork/pastelup/utils"
)

/*
var (

	wg sync.WaitGroup

)
*/

var (
	flagPurge bool
)

type uninstallCommand uint8

const (
	nodeUninstall uninstallCommand = iota
	walletUninstall
	superNodeUninstall
	ddServiceUninstall
	rqServiceUninstall
	ddImgServerUninstall
	wnServiceUninstall
	snServiceUninstall
	masterNodeUninstall
	remoteUninstall
	bridgeServiceUninstall
	hermesServiceUninstall
)

var (
	uninstallCmdName = map[uninstallCommand]string{
		nodeUninstall:          "node",
		walletUninstall:        "walletnode",
		superNodeUninstall:     "supernode",
		ddServiceUninstall:     "dd-service",
		rqServiceUninstall:     "rq-service",
		ddImgServerUninstall:   "imgserver",
		wnServiceUninstall:     "walletnode-service",
		snServiceUninstall:     "supernode-service",
		masterNodeUninstall:    "masternode",
		remoteUninstall:        "remote",
		bridgeServiceUninstall: "bridge-service",
		hermesServiceUninstall: "hermes-service",
	}
	uninstallCmdMessage = map[uninstallCommand]string{
		nodeUninstall:          "Uninstall node",
		walletUninstall:        "Uninstall Walletnode",
		superNodeUninstall:     "Uninstall Supernode",
		ddServiceUninstall:     "Uninstall Dupe Detection service only",
		rqServiceUninstall:     "Uninstall RaptorQ service only",
		ddImgServerUninstall:   "Uninstall dd image server",
		wnServiceUninstall:     "Uninstall Walletnode service only",
		snServiceUninstall:     "Uninstall Supernode service only",
		masterNodeUninstall:    "Uninstall only pasteld node as Masternode",
		remoteUninstall:        "Uninstall on Remote host",
		bridgeServiceUninstall: "Uninstall bridge-service only",
		hermesServiceUninstall: "Uninstall hermes-service only",
	}
)

func setupUninstallSubCommand(config *configs.Config,
	uninstallCommand uninstallCommand, remote bool,
	f func(context.Context, *configs.Config) error,
) *cli.Command {
	commonFlags := []*cli.Flag{
		cli.NewFlag("purge", &flagPurge).
			SetUsage(green("Optional, Remove all data and configuration files")),
		cli.NewFlag("force", &config.Force).SetAliases("f").
			SetUsage(green("Optional, Force to overwrite config files and re-download ZKSnark parameters")),
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
				SetUsage(green("Optional, Location of pastel node directory on the remote computer (default: $HOME/pastel)")),
			cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
				SetUsage(green("Optional, Location of working directory on the remote computer (default: $HOME/.pastel)")),
		}
	}

	remoteUninstallFlags := []*cli.Flag{
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
		cli.NewFlag("in-parallel", &config.AsyncRemote).
			SetUsage(green("Optional, When using inventory file run remote tasks in parallel")),
	}

	var commandName, commandMessage string
	if !remote {
		commandName = uninstallCmdName[uninstallCommand]
		commandMessage = uninstallCmdMessage[uninstallCommand]
	} else {
		commandName = uninstallCmdName[remoteUninstall]
		commandMessage = uninstallCmdMessage[remoteUninstall]
	}

	commandFlags := append(commonFlags, dirsFlags[:]...)

	if remote {
		commandFlags = append(commandFlags, remoteUninstallFlags[:]...)
	}

	subCommand := cli.NewCommand(commandName)
	subCommand.SetUsage(cyan(commandMessage))
	subCommand.AddFlags(commandFlags...)
	addLogFlags(subCommand, config)

	if f != nil {
		subCommand.SetActionFunc(func(ctx context.Context, _ []string) error {
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

func setupUninstallCommand(config *configs.Config) *cli.Command {

	uninstallNodeSubCommand := setupUninstallSubCommand(config, nodeUninstall, false, runUninstallNodeSubCommand)
	uninstallWalletNodeSubCommand := setupUninstallSubCommand(config, walletUninstall, false, runUninstallWalletNodeSubCommand)
	uninstallSuperNodeSubCommand := setupUninstallSubCommand(config, superNodeUninstall, false, runUninstallSuperNodeSubCommand)

	uninstallRQServiceCommand := setupUninstallSubCommand(config, rqServiceUninstall, false, runUninstallRQService)
	uninstallDDServiceCommand := setupUninstallSubCommand(config, ddServiceUninstall, false, runUninstallDDService)
	uninstallWNServiceCommand := setupUninstallSubCommand(config, wnServiceUninstall, false, runUninstallWalletNodeService)
	uninstallSNServiceCommand := setupUninstallSubCommand(config, snServiceUninstall, false, runUninstallSuperNodeService)
	uninstallHermesServiceCommand := setupUninstallSubCommand(config, hermesServiceUninstall, false, runUninstallHermesService)
	// uninstallBridgeServiceCommand := setupUninstallSubCommand(config, bridgeServiceUninstall, false, runUninstallBridgeService)

	uninstallNodeRemoteSubCommand := setupUninstallSubCommand(config, nodeUninstall, true, runRemoteNodeUninstallSubCommand)
	uninstallNodeSubCommand.AddSubcommands(uninstallNodeRemoteSubCommand)
	uninstallWalletNodeRemoteSubCommand := setupUninstallSubCommand(config, superNodeUninstall, true, runRemoteWalletNodeUninstallSubCommand)
	uninstallWalletNodeSubCommand.AddSubcommands(uninstallWalletNodeRemoteSubCommand)
	uninstallSuperNodeRemoteSubCommand := setupUninstallSubCommand(config, superNodeUninstall, true, runRemoteSuperNodeUninstallSubCommand)
	uninstallSuperNodeSubCommand.AddSubcommands(uninstallSuperNodeRemoteSubCommand)

	uninstallRQServiceRemoteCommand := setupUninstallSubCommand(config, rqServiceUninstall, true, runRemoteRQServiceUninstallSubCommand)
	uninstallRQServiceCommand.AddSubcommands(uninstallRQServiceRemoteCommand)
	uninstallDDServiceRemoteCommand := setupUninstallSubCommand(config, ddServiceUninstall, true, runRemoteDDServiceUninstallSubCommand)
	uninstallDDServiceCommand.AddSubcommands(uninstallDDServiceRemoteCommand)
	uninstallWNServiceRemoteCommand := setupUninstallSubCommand(config, wnServiceUninstall, true, runRemoteWNServiceUninstallSubCommand)
	uninstallWNServiceCommand.AddSubcommands(uninstallWNServiceRemoteCommand)
	uninstallSNServiceRemoteCommand := setupUninstallSubCommand(config, snServiceUninstall, true, runRemoteSNServiceUninstallSubCommand)
	uninstallSNServiceCommand.AddSubcommands(uninstallSNServiceRemoteCommand)
	uninstallHermesServiceRemoteCommand := setupUninstallSubCommand(config, hermesServiceUninstall, true, runRemoteHermesServiceUninstallSubCommand)
	uninstallHermesServiceCommand.AddSubcommands(uninstallHermesServiceRemoteCommand)
	// uninstallBridgeServiceRemoteCommand := setupUninstallSubCommand(config, bridgeServiceUninstall, true, runRemoteBridgeServiceUninstallSubCommand)
	// uninstallBridgeServiceCommand.AddSubcommands(uninstallBridgeServiceRemoteCommand)

	uninstallCommand := cli.NewCommand("uninstall")
	uninstallCommand.SetUsage(blue("Performs uninstall of the system for both WalletNode and SuperNodes"))
	uninstallCommand.AddSubcommands(uninstallNodeSubCommand)
	uninstallCommand.AddSubcommands(uninstallWalletNodeSubCommand)
	uninstallCommand.AddSubcommands(uninstallSuperNodeSubCommand)

	uninstallCommand.AddSubcommands(uninstallRQServiceCommand)
	uninstallCommand.AddSubcommands(uninstallDDServiceCommand)
	uninstallCommand.AddSubcommands(uninstallWNServiceCommand)
	uninstallCommand.AddSubcommands(uninstallSNServiceCommand)
	uninstallCommand.AddSubcommands(uninstallHermesServiceCommand)
	// uninstallCommand.AddSubcommands(uninstallBridgeServiceCommand)

	return uninstallCommand

}

// /// Top level start commands
func removeFile(ctx context.Context, dir string, fileName string) {
	err := os.Remove(path.Join(dir, fileName))
	if err != nil {
		log.WithContext(ctx).Warn(fmt.Sprintf("Unable to delete %s: %v", fileName, err))
	}
}
func removeDir(ctx context.Context, parentDir string, dir string) {
	err := os.RemoveAll(path.Join(parentDir, dir))
	if err != nil {
		log.WithContext(ctx).Warn(fmt.Sprintf("Unable to delete %s: %v", dir, err))
	}
}

func askToContinue(ctx context.Context, config *configs.Config, what string) bool {
	if config.Force {
		return true
	}
	if ok, _ := AskUserToContinue(ctx, "Do you want to remove "+what+"? Y/N"); !ok {
		log.WithContext(ctx).Infof("Skip %s removal", what)
		return false
	}
	return true
}

// Sub Command
func runUninstallNodeSubCommand(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Uninstalling Pastel node")
	if !askToContinue(ctx, config, "Pastel Node") {
		return nil
	}

	runStopNodeSubCommand(ctx, config)
	removeFile(ctx, config.PastelExecDir, constants.PasteldName[utils.GetOS()])
	removeFile(ctx, config.PastelExecDir, constants.PastelCliName[utils.GetOS()])

	//TODO
	/*	if flagPurge {
		if !askToContinue(ctx, config, "all data and configuration of Pastel Node"){
			return nil
		}
		removeFile(ctx, config.WorkingDir, constants.PastelConfName)
		ClearDir(ctx, config.WorkingDir)
	}*/

	log.WithContext(ctx).Info("Pastel node uninstalled successfully")
	return nil
}

// Sub Command
func runUninstallWalletNodeSubCommand(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Uninstalling walletnode")
	if !askToContinue(ctx, config, "Wallet Node") {
		return nil
	}

	runStopWalletSubCommand(ctx, config)
	removeFile(ctx, config.PastelExecDir, constants.PasteldName[utils.GetOS()])
	removeFile(ctx, config.PastelExecDir, constants.PastelCliName[utils.GetOS()])
	removeFile(ctx, config.PastelExecDir, constants.WalletNodeExecName[utils.GetOS()])
	removeFile(ctx, config.PastelExecDir, constants.PastelRQServiceExecName[utils.GetOS()])
	// removeFile(ctx, config.PastelExecDir, constants.BridgeExecName[utils.GetOS()])
	if flagPurge {
		if !askToContinue(ctx, config, "all data and configuration of Walletnode service") {
			return nil
		}
		removeFile(ctx, "", config.Configurer.GetWalletNodeConfFile(config.WorkingDir))
		removeFile(ctx, "", config.Configurer.GetWalletNodeLogFile(config.WorkingDir))
		removeFile(ctx, "", config.Configurer.GetRQServiceConfFile(config.WorkingDir))
		removeFile(ctx, config.WorkingDir, "rqservice.log")
		removeDir(ctx, config.WorkingDir, "rqfiles")
	}

	log.WithContext(ctx).Info("Walletnode uninstalled successfully")
	return nil
}

// Sub Command
func runUninstallSuperNodeSubCommand(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Uninstalling supernode")
	if ok, _ := AskUserToContinue(ctx, "Do you want to remove Super Node? Y/N"); !ok {
		log.WithContext(ctx).Info("Skip Super node removal")
		return nil
	}
	if !askToContinue(ctx, config, "Wallet Node") {
		return nil
	}

	runStopSuperNodeSubCommand(ctx, config)
	removeFile(ctx, config.PastelExecDir, constants.PasteldName[utils.GetOS()])
	removeFile(ctx, config.PastelExecDir, constants.PastelCliName[utils.GetOS()])
	removeFile(ctx, config.PastelExecDir, constants.SuperNodeExecName[utils.GetOS()])
	removeFile(ctx, config.PastelExecDir, constants.PastelRQServiceExecName[utils.GetOS()])
	removeFile(ctx, config.PastelExecDir, constants.HermesExecName[utils.GetOS()])
	removeDir(ctx, config.PastelExecDir, constants.DupeDetectionSubFolder)
	if flagPurge {
		if !askToContinue(ctx, config, "all data and configuration of supernode") {
			return nil
		}
		removeFile(ctx, "", config.Configurer.GetSuperNodeConfFile(config.WorkingDir))
		removeFile(ctx, "", config.Configurer.GetSuperNodeLogFile(config.WorkingDir))
		removeDir(ctx, config.WorkingDir, "p2pdata")
		removeDir(ctx, config.WorkingDir, "tmp")

		removeFile(ctx, "", config.Configurer.GetRQServiceConfFile(config.WorkingDir))
		removeFile(ctx, config.WorkingDir, "rqservice.log")
		removeDir(ctx, config.WorkingDir, "rqfiles")

		removeFile(ctx, "", config.Configurer.GetHermesConfFile(config.WorkingDir))
		removeFile(ctx, "", config.Configurer.GetHermesLogFile(config.WorkingDir))

		removeDir(ctx, config.Configurer.DefaultHomeDir(), constants.DupeDetectionServiceDir)
		removeDir(ctx, config.WorkingDir, "dd-server")
	}

	log.WithContext(ctx).Info("Supernode uninstalled successfully")
	return nil
}

// Sub Command
func runUninstallRQService(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Uninstalling RQ Service")
	if !askToContinue(ctx, config, "RQ service") {
		return nil
	}

	stopRQServiceSubCommand(ctx, config)
	removeFile(ctx, config.PastelExecDir, constants.PastelRQServiceExecName[utils.GetOS()])
	if flagPurge {
		if !askToContinue(ctx, config, "all data and configuration of DD service") {
			return nil
		}
		removeFile(ctx, "", config.Configurer.GetRQServiceConfFile(config.WorkingDir))
		removeFile(ctx, config.WorkingDir, "rqservice.log")
		removeDir(ctx, config.WorkingDir, "rqfiles")
	}

	log.WithContext(ctx).Info("RQ Service uninstalled successfully")
	return nil
}

// Sub Command
func runUninstallDDService(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Uninstalling DD Service")
	if !askToContinue(ctx, config, "DD service") {
		return nil
	}

	stopDDServiceSubCommand(ctx, config)
	removeDir(ctx, config.PastelExecDir, constants.DupeDetectionSubFolder)
	if flagPurge {
		if !askToContinue(ctx, config, "all data and configuration of RQ service") {
			return nil
		}
		removeDir(ctx, config.Configurer.DefaultHomeDir(), constants.DupeDetectionServiceDir)
		removeDir(ctx, config.WorkingDir, "dd-server")
	}

	log.WithContext(ctx).Info("DD Service uninstalled successfully")
	return nil
}

// Sub Command
func runUninstallWalletNodeService(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Uninstalling Walletnode Service")
	if !askToContinue(ctx, config, "Walletnode service") {
		return nil
	}

	stopWNServiceSubCommand(ctx, config)
	removeFile(ctx, config.PastelExecDir, constants.WalletNodeExecName[utils.GetOS()])
	if flagPurge {
		if !askToContinue(ctx, config, "all data and configuration of Walletnode service") {
			return nil
		}
		removeFile(ctx, "", config.Configurer.GetWalletNodeConfFile(config.WorkingDir))
		removeFile(ctx, "", config.Configurer.GetWalletNodeLogFile(config.WorkingDir))
	}

	log.WithContext(ctx).Info("Walletnode Service uninstalled successfully")
	return nil
}

// Sub Command
func runUninstallSuperNodeService(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Uninstalling Supernode Service")
	if !askToContinue(ctx, config, "Supernode service") {
		return nil
	}

	stopSNServiceSubCommand(ctx, config)
	removeFile(ctx, config.PastelExecDir, constants.SuperNodeExecName[utils.GetOS()])
	if flagPurge {
		if !askToContinue(ctx, config, "all data and configuration of Supernode service") {
			return nil
		}
		removeFile(ctx, "", config.Configurer.GetSuperNodeConfFile(config.WorkingDir))
		removeFile(ctx, "", config.Configurer.GetSuperNodeLogFile(config.WorkingDir))
		removeDir(ctx, config.WorkingDir, "p2pdata")
		removeDir(ctx, config.WorkingDir, "tmp")
	}

	log.WithContext(ctx).Info("Supernode Service uninstalled successfully")
	return nil
}

// Sub Command
func runUninstallHermesService(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Uninstalling Hermes Service")
	if !askToContinue(ctx, config, "Hermes service") {
		return nil
	}

	stopHermesService(ctx, config)
	removeFile(ctx, config.PastelExecDir, constants.HermesExecName[utils.GetOS()])
	if flagPurge {
		if !askToContinue(ctx, config, "all data and configuration of Hermes service") {
			return nil
		}
		removeFile(ctx, "", config.Configurer.GetHermesConfFile(config.WorkingDir))
		removeFile(ctx, "", config.Configurer.GetHermesLogFile(config.WorkingDir))
	}

	log.WithContext(ctx).Info("Hermes Service uninstalled successfully")
	return nil
}

// Sub Command
// func runUninstallBridgeService(ctx context.Context, config *configs.Config) error {
// 	log.WithContext(ctx).Info("Uninstalling Bridge Service")

// 	if ok, _ := AskUserToContinue(ctx, "Do you want to remove Bridge service Node? Y/N"); !ok {
// 		log.WithContext(ctx).Info("Skip Bridge service removal")
// 		return nil
// 	}

// 	stopBridgeService(ctx, config)
// 	removeFile(ctx, config.PastelExecDir, constants.PastelBridge[utils.GetOS()])

// 	log.WithContext(ctx).Info("Bridge Service uninstalled successfully")
// 	return nil
// }

func runRemoteNodeUninstallSubCommand(ctx context.Context, config *configs.Config) error {
	return runRemoteUninstall(ctx, config, "node")
}
func runRemoteWalletNodeUninstallSubCommand(ctx context.Context, config *configs.Config) error {
	return runRemoteUninstall(ctx, config, "walletnode")
}
func runRemoteSuperNodeUninstallSubCommand(ctx context.Context, config *configs.Config) error {
	return runRemoteUninstall(ctx, config, "supernode")
}
func runRemoteRQServiceUninstallSubCommand(ctx context.Context, config *configs.Config) error {
	return runRemoteUninstall(ctx, config, "rq-service")
}
func runRemoteDDServiceUninstallSubCommand(ctx context.Context, config *configs.Config) error {
	return runRemoteUninstall(ctx, config, "dd-service")
}
func runRemoteWNServiceUninstallSubCommand(ctx context.Context, config *configs.Config) error {
	return runRemoteUninstall(ctx, config, "walletnode-service")
}
func runRemoteSNServiceUninstallSubCommand(ctx context.Context, config *configs.Config) error {
	return runRemoteUninstall(ctx, config, "supernode-service")
}
func runRemoteHermesServiceUninstallSubCommand(ctx context.Context, config *configs.Config) error {
	return runRemoteUninstall(ctx, config, "hermes-service")
}

// func runRemoteBridgeServiceUninstallSubCommand(ctx context.Context, config *configs.Config) error {
// 	return runRemoteUninstall(ctx, config, "bridge-service")
// }

func runRemoteUninstall(ctx context.Context, config *configs.Config, tool string) error {
	log.WithContext(ctx).Infof("Uninstall remote %s", tool)

	// Start remote node
	uninstallOptions := tool

	if flagPurge {
		uninstallOptions = fmt.Sprintf("%s --purge", uninstallOptions)
	}
	if config.Force {
		uninstallOptions = fmt.Sprintf("%s --force", uninstallOptions)
	}

	if len(config.PastelExecDir) > 0 {
		uninstallOptions = fmt.Sprintf("%s --dir=%s", uninstallOptions, config.PastelExecDir)
	}
	if len(config.WorkingDir) > 0 {
		uninstallOptions = fmt.Sprintf("%s --work-dir=%s", uninstallOptions, config.WorkingDir)
	}

	uninstallCmd := fmt.Sprintf("%s uninstall %s", constants.RemotePastelupPath, uninstallOptions)
	if _, err := executeRemoteCommandsWithInventory(ctx, config, []string{uninstallCmd}, false, false); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to uninstall %s on remote host", tool)
	}

	log.WithContext(ctx).Infof("Remote %s uninstall successfully", tool)
	return nil
}
