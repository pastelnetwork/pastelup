package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/pastel-utility/constants"
	"github.com/pastelnetwork/pastel-utility/utils"
)

type updateCommand uint8

const (
	updateWalletNode updateCommand = iota
	updateSuperNode
	updateSuperNodeRemote
)

func setupUpdateSubCommand(config *configs.Config,
	updateCmd updateCommand,
	f func(context.Context, *configs.Config) error,
) *cli.Command {

	commonFlags := []*cli.Flag{
		cli.NewFlag("user-pw", &config.UserPw).
			SetUsage(green("Optional, password of current sudo user - so no sudo password request is prompted")),
	}

	var dirsFlags []*cli.Flag

	if updateCmd != updateSuperNodeRemote {
		dirsFlags = []*cli.Flag{
			cli.NewFlag("dir", &config.PastelExecDir).SetAliases("d").
				SetUsage(green("Optional, Location where to create pastel node directory")).SetValue(config.Configurer.DefaultPastelExecutableDir()),
			cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
				SetUsage(green("Optional, Location where to create working directory")).SetValue(config.Configurer.DefaultWorkingDir()),
		}
	} else {
		dirsFlags = []*cli.Flag{
			cli.NewFlag("remote-dir", &config.RemotePastelExecDir).SetAliases("d").
				SetUsage(green("Optional, Location where to create pastel node directory on the remote computer (default: $HOME/pastel)")),
			cli.NewFlag("remote-work-dir", &config.RemoteWorkingDir).SetAliases("w").
				SetUsage(green("Optional, Location where to create working directory on the remote computer (default: $HOME/.pastel)")),
		}
	}

	remoteFlags := []*cli.Flag{
		cli.NewFlag("ssh-ip", &config.RemoteIP).
			SetUsage(red("Required, SSH address of the remote host")).SetRequired(),
		cli.NewFlag("ssh-port", &config.RemotePort).
			SetUsage(yellow("Optional, SSH port of the remote host, default is 22")).SetValue(22),
		cli.NewFlag("ssh-user", &config.RemoteUser).
			SetUsage(yellow("Optional, SSH user")),
		cli.NewFlag("ssh-user-pw", &config.RemoteUserPw).
			SetUsage(red("Required, password of remote user - so no sudo request is promoted")).SetRequired(),
		cli.NewFlag("ssh-key", &config.SSHKey).
			SetUsage(yellow("Optional, Path to SSH private key")),
		cli.NewFlag("utility-path", &config.BinUtilityPath).SetRequired().
			SetUsage(red("Optional, path to the local binary pastel-utility file to copy to remote host")),
		cli.NewFlag("bin", &config.BinComponentPath).SetRequired().
			SetUsage(red("Optional, path to the local binary (pasteld, pastel-cli, rq-service, supernode) file to copy to remote host")),
	}

	var commandName, commandMessage string
	var commandFlags []*cli.Flag

	switch updateCmd {
	case updateWalletNode:
		commandFlags = append(dirsFlags, commonFlags[:]...)
		commandName = string(constants.WalletNode)
		commandMessage = "Update walletnode"
	case updateSuperNode:
		commandFlags = append(dirsFlags, commonFlags[:]...)
		commandName = string(constants.SuperNode)
		commandMessage = "Update supernode"
	case updateSuperNodeRemote:
		commandFlags = append(append(dirsFlags, commonFlags[:]...), remoteFlags[:]...)
		commandName = "remote"
		commandMessage = "Update supernode remote"
	default:
		commandFlags = append(append(dirsFlags, commonFlags[:]...), remoteFlags[:]...)
	}

	// Assign sub-cmd flag
	subCommand := cli.NewCommand(commandName)
	subCommand.AddFlags(commandFlags...)

	if f != nil {
		subCommand.SetActionFunc(func(ctx context.Context, _ []string) error {
			ctx, err := configureLogging(ctx, commandMessage, config)
			if err != nil {
				return fmt.Errorf("failed to configure logging option - %v", err)
			}

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			sys.RegisterInterruptHandler(cancel, func() {
				log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
				os.Exit(0)
			})

			log.WithContext(ctx).Info("Started")
			if err = f(ctx, config); err != nil {
				return err
			}
			log.WithContext(ctx).Info("Finished successfully!")
			return nil
		})
	}

	return subCommand
}

func setupUpdateCommand() *cli.Command {
	config := configs.InitConfig()

	updateSuperNodeRemoteSubCommand := setupUpdateSubCommand(config, updateSuperNodeRemote, runUpdateSuperNodeRemoteSubCommand)
	updateSuperNodeSubCommand := setupUpdateSubCommand(config, updateSuperNode, runUpdateSuperNodeSubCommand)
	updateSuperNodeSubCommand.AddSubcommands(updateSuperNodeRemoteSubCommand)

	// Add update command
	updateCommand := cli.NewCommand("update")
	updateCommand.SetUsage(blue("Perform update components for each service: WalletNode and SuperNode"))

	updateCommand.AddSubcommands(updateSuperNodeSubCommand)

	return updateCommand
}

func runUpdateSuperNodeRemoteSubCommand(ctx context.Context, config *configs.Config) (err error) {

	// Validate config
	if len(config.RemoteIP) == 0 {
		return fmt.Errorf("--ssh-ip IP address - Required, SSH address of the remote host")
	}

	// Connect to remote
	var client *utils.Client
	log.WithContext(ctx).Infof("Connecting to remote host -> %s:%d...", config.RemoteIP, config.RemotePort)

	if len(config.SSHKey) == 0 {
		username, password, _ := utils.Credentials(config.RemoteUser, true)
		client, err = utils.DialWithPasswd(fmt.Sprintf("%s:%d", config.RemoteIP, config.RemotePort), username, password)
	} else {
		username, _, _ := utils.Credentials(config.RemoteUser, false)
		client, err = utils.DialWithKey(fmt.Sprintf("%s:%d", config.RemoteIP, config.RemotePort), username, config.SSHKey)
	}
	if err != nil {
		return err
	}

	defer client.Close()

	log.WithContext(ctx).Info("Connected successfully")

	/* Upload pastel-utility to remote at /tmp */
	pastelUtilityPath := "/tmp/pastel-utility"

	BinUtilityPath := config.BinUtilityPath
	if len(BinUtilityPath) == 0 {
		BinUtilityPath = os.Args[0]
	}

	log.WithContext(ctx).Infof("Copying local pastel-utility from  executable to remote host - %s", BinUtilityPath)

	if err := client.Scp(BinUtilityPath, pastelUtilityPath); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to copy pastel-utility executable to remote host")
		return err
	}

	log.WithContext(ctx).Info("Successfully copied pastel-utility executable to remote host")

	/* Stop supernode services using pastel-utility */
	log.WithContext(ctx).Info("Stopping Supernode service ...")

	remoteOptions := ""

	if len(config.RemoteUserPw) > 0 {
		remoteOptions += " --user-pw " + config.RemoteUserPw
	}

	stopSuperNodeCmd := fmt.Sprintf("%s stop supernode %s", pastelUtilityPath, remoteOptions)
	err = client.ShellCmd(ctx, stopSuperNodeCmd)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to Installing Supernode")
		return err
	}

	log.WithContext(ctx).Info("Successfully stop supernode at remote host")

	/* Copy the binary (pastel-cli, pasteld, pastel-cli, rq-service, supernode) to remote location to overwrite binary */

	/* Restart again supernode service */

	return nil
}

func runUpdateSuperNodeSubCommand(ctx context.Context, config *configs.Config) (err error) {

	return nil
}
