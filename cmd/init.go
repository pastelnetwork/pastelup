package cmd

import (
	"context"
	"fmt"
	"github.com/pastelnetwork/pastelup/constants"
	"os"
	"os/signal"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/pastelup/configs"
)

var (
	// masternode flags
	flagMasterNodeIsActivate bool
	flagMasterNodeName       string
	flagMasterNodeIsCreate   bool
	flagMasterNodeIsUpdate   bool
	flagMasterNodeTxID       string
	flagMasterNodeInd        string
	flagMasterNodePort       int
	flagMasterNodePrivateKey string
	flagMasterNodePastelID   string
	flagMasterNodePassPhrase string
	flagMasterNodeRPCIP      string
	flagMasterNodeRPCPort    int
	flagMasterNodeP2PIP      string
	flagMasterNodeP2PPort    int
)

type initCommand uint8

const (
	superNodeInit initCommand = iota
	coldHotInit
	remoteInit
)

var (
	initCmdName = map[initCommand]string{
		superNodeInit: "supernode",
		coldHotInit:   "coldhot",
		remoteInit:    "remote",
	}
	initCmdMessage = map[initCommand]string{
		superNodeInit: "Initialise local Supernode",
		coldHotInit:   "Initialise Supernode in Cold/Hot mode",
		remoteInit:    "Initialise remote Supernode",
	}
)

func setupInitSubCommand(config *configs.Config,
	initCommand initCommand, remote bool,
	f func(context.Context, *configs.Config) error,
) *cli.Command {

	commonFlags := []*cli.Flag{
		cli.NewFlag("ip", &flagNodeExtIP).
			SetUsage(green("Optional, WAN address of the host")),
		cli.NewFlag("reindex", &flagReIndex).SetAliases("r").
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

	superNodeInitFlags := []*cli.Flag{
		cli.NewFlag("name", &flagMasterNodeName).
			SetUsage(red("Required, name of the Masternode to create or update in the masternode.conf")).SetRequired(),

		cli.NewFlag("new", &flagMasterNodeIsCreate).
			SetUsage(red("Required (if --update is not used), if specified, will add new Masternode record in the masternode.conf.")),
		cli.NewFlag("update", &flagMasterNodeIsUpdate).
			SetUsage(red("Required (if --new is not used), if specified, will update Masternode record in the masternode.conf.")),

		cli.NewFlag("pkey", &flagMasterNodePrivateKey).
			SetUsage(green("Optional, Masternode private key, if omitted, new masternode private key will be created")),
		cli.NewFlag("txid", &flagMasterNodeTxID).
			SetUsage(yellow("Required (only if --update or --create specified), collateral payment txid , transaction id of 5M collateral MN payment")),
		cli.NewFlag("ind", &flagMasterNodeInd).
			SetUsage(yellow("Required (only if --update or --create specified), collateral payment output index , output index in the transaction of 5M collateral MN payment")),

		cli.NewFlag("pastelid", &flagMasterNodePastelID).
			SetUsage(green("Optional, pastelid of the Masternode. If omitted, new pastelid will be created and registered")),
		cli.NewFlag("passphrase", &flagMasterNodePassPhrase).
			SetUsage(yellow("Optional, passphrase to pastelid private key. If omitted, user will be asked interactively")),
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

		cli.NewFlag("activate", &flagMasterNodeIsActivate).
			SetUsage(green("Optional, if specified, will try to enable node as Masternode (start-alias).")),
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
	}
	coldhotStartFlags := []*cli.Flag{
		cli.NewFlag("ssh-ip", &config.RemoteIP).
			SetUsage(red("Required, SSH address of the remote HOT node")),
		cli.NewFlag("ssh-port", &config.RemotePort).
			SetUsage(green("Optional, SSH port of the remote HOT node")).SetValue(22),
		cli.NewFlag("ssh-user", &config.RemoteUser).
			SetUsage(yellow("Optional, SSH user")),
		cli.NewFlag("ssh-key", &config.RemoteSSHKey).
			SetUsage(yellow("Optional, Path to SSH private key")),
	}

	var commandName, commandMessage string
	if remote && initCommand != coldHotInit {
		commandName = initCmdName[remoteInit]
		commandMessage = initCmdMessage[remoteInit]
	} else {
		commandName = initCmdName[initCommand]
		commandMessage = initCmdMessage[initCommand]
	}

	commandFlags := append(dirsFlags, commonFlags[:]...)
	commandFlags = append(commandFlags, superNodeInitFlags[:]...)

	if remote && initCommand != coldHotInit {
		commandFlags = append(commandFlags, remoteStartFlags[:]...)
	}
	if initCommand == coldHotInit {
		commandFlags = append(commandFlags, coldhotStartFlags[:]...)
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

func setupInitCommand() *cli.Command {
	config := configs.InitConfig()

	initSuperNodeSubCommand := setupInitSubCommand(config, superNodeInit, false, runInitSuperNodeSubCommand)
	initColdHotSuperNodeSubCommand := setupInitSubCommand(config, coldHotInit, true, runInitColdHotSuperNodeSubCommand)
	initRemoteSuperNodeSubCommand := setupInitSubCommand(config, superNodeInit, true, runInitRemoteSuperNodeSubCommand)

	initSuperNodeSubCommand.AddSubcommands(initColdHotSuperNodeSubCommand)
	initSuperNodeSubCommand.AddSubcommands(initRemoteSuperNodeSubCommand)

	startCommand := cli.NewCommand("init")
	startCommand.SetUsage(blue("Initialise SuperNode either locally, remotely or in cold-hot mode"))
	startCommand.AddSubcommands(initSuperNodeSubCommand)

	return startCommand

}

///// Top level start commands

// Sub Command
func runInitSuperNodeSubCommand(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Initialising local supernode")
	if err := runStartSuperNode(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to initialize local supernode")
		return err
	}
	log.WithContext(ctx).Info("Local supernode Initialised successfully")
	return nil
}

func runInitColdHotSuperNodeSubCommand(ctx context.Context, config *configs.Config) (err error) {
	runner := &ColdHotRunner{
		config: config,
		opts:   &ColdHotRunnerOpts{},
	}

	log.WithContext(ctx).Info("Initialising supernode in coldhot mode")
	if err := runner.Init(ctx); err != nil {
		log.WithContext(ctx).WithError(err).Error("init coldhot runner failed.")
		return err
	}

	log.WithContext(ctx).Info("running supernode coldhot runner")
	if err := runner.Run(ctx); err != nil {
		log.WithContext(ctx).WithError(err).Error("run coldhot runner failed.")
		return err
	}
	log.WithContext(ctx).Info("Supernode in coldhot mode initialised successfully")

	return nil
}

func runInitRemoteSuperNodeSubCommand(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Infof("Initializing remote supernode")

	startOptions := ""

	if len(flagMasterNodeName) > 0 {
		startOptions = fmt.Sprintf("--name=%s", flagMasterNodeName)
	}

	if flagMasterNodeIsActivate {
		startOptions = fmt.Sprintf("%s --activate", startOptions)
	}

	if len(flagMasterNodePrivateKey) > 0 {
		startOptions = fmt.Sprintf("%s --pkey=%s", startOptions, flagMasterNodePrivateKey)
	}

	if flagMasterNodeIsCreate {
		startOptions = fmt.Sprintf("%s --new", startOptions)
	} else if flagMasterNodeIsUpdate {
		startOptions = fmt.Sprintf("%s --update", startOptions)
	}

	if len(flagMasterNodeTxID) > 0 {
		startOptions = fmt.Sprintf("%s --txid=%s", startOptions, flagMasterNodeTxID)
	}

	if len(flagMasterNodeInd) > 0 {
		startOptions = fmt.Sprintf("%s --ind=%s", startOptions, flagMasterNodeInd)
	}

	if len(flagMasterNodePastelID) > 0 {
		startOptions = fmt.Sprintf("%s --pastelid=%s", startOptions, flagMasterNodePastelID)
	}

	if len(flagMasterNodePassPhrase) > 0 {
		startOptions = fmt.Sprintf("%s --passphrase=%s", startOptions, flagMasterNodePassPhrase)
	}

	if flagMasterNodePort > 0 {
		startOptions = fmt.Sprintf("%s --port=%d", startOptions, flagMasterNodePort)
	}

	if len(flagMasterNodeRPCIP) > 0 {
		startOptions = fmt.Sprintf("%s --rpc-ip=%s", startOptions, flagMasterNodeRPCIP)
	}

	if flagMasterNodeRPCPort > 0 {
		startOptions = fmt.Sprintf("%s --rpc-port=%d", startOptions, flagMasterNodeRPCPort)
	}

	if len(flagMasterNodeP2PIP) > 0 {
		startOptions = fmt.Sprintf("%s --p2p-ip=%s", startOptions, flagMasterNodeP2PIP)
	}

	if flagMasterNodeP2PPort > 0 {
		startOptions = fmt.Sprintf("%s --p2p-port=%d", startOptions, flagMasterNodeP2PPort)
	}

	if len(flagNodeExtIP) > 0 {
		startOptions = fmt.Sprintf("%s --ip=%s", startOptions, flagNodeExtIP)
	}

	if flagReIndex {
		startOptions = fmt.Sprintf("%s --reindex", startOptions)
	}

	if len(config.PastelExecDir) > 0 {
		startOptions = fmt.Sprintf("%s --dir=%s", startOptions, config.PastelExecDir)
	}

	if len(config.WorkingDir) > 0 {
		startOptions = fmt.Sprintf("%s --work-dir=%s", startOptions, config.WorkingDir)
	}

	initSuperNodeCmd := fmt.Sprintf("%s init supernode %s", constants.RemotePastelupPath, startOptions)
	if err := executeRemoteCommands(ctx, config, []string{initSuperNodeCmd}, false); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to init remote Supernode services")
		return err
	}
	log.WithContext(ctx).Infof("Remote supernode initialized")

	return nil
}
