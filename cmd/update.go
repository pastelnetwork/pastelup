package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/pastel-utility/constants"
	"github.com/pastelnetwork/pastel-utility/utils"
)

type updateCommand uint8

const (
	updateNode updateCommand = iota
	updateWalletNode
	updateSuperNode
	updateSuperNodeRemote
)

var (
	updateCommandName = map[updateCommand]string{
		updateNode:            "node",
		updateWalletNode:      "walletnode",
		updateSuperNode:       "supernode",
		updateSuperNodeRemote: "remote",
	}

	// updateConfigFlags struct to hold all the flags settings
	updateConfigFlags struct {
		MasterNodeName   string
		RemoteIP         string
		RemotePort       int
		RemoteUser       string
		SSHKey           string
		BinUtilityPath   string
		BinComponentPath string
	}
)

func setupUpdateSubCommand(config *configs.Config,
	updateCmd updateCommand,
	f func(context.Context, *configs.Config) error,
) *cli.Command {

	commonFlags := []*cli.Flag{
		cli.NewFlag("network", &config.Network).SetAliases("n").
			SetUsage(green("Optional, network type, can be - \"mainnet\" or \"testnet\"")).SetValue("mainnet"),
		cli.NewFlag("force", &config.Force).SetAliases("f").
			SetUsage(green("Optional, Force to overwrite config files and re-download ZKSnark parameters")),
		cli.NewFlag("peers", &config.Peers).SetAliases("p").
			SetUsage(green("Optional, List of peers to add into pastel.conf file, must be in the format - \"ip\" or \"ip:port\"")),
		cli.NewFlag("release", &config.Version).SetAliases("r").
			SetUsage(green("Optional, Pastel version to install")).SetValue("beta"),

		cli.NewFlag("dir", &config.PastelExecDir).SetAliases("d").
			SetUsage(green("Optional, Location where to create pastel node directory")).SetValue(config.Configurer.DefaultPastelExecutableDir()),
		cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
			SetUsage(green("Optional, Location where to create working directory")).SetValue(config.Configurer.DefaultWorkingDir()),
	}

	if updateCmd == updateSuperNodeRemote || updateCmd == updateSuperNode {
		commonFlags = append(commonFlags,
			cli.NewFlag("user-pw", &config.UserPw).
				SetUsage(green("Optional, password of current sudo user - so no sudo password request is prompted")),
		)
	}

	remoteFlags := []*cli.Flag{
		cli.NewFlag("name", &updateConfigFlags.MasterNodeName).
			SetUsage(red("Required, name of the Masternode to start (and create or update in the masternode.conf if --create or --update are specified)")).SetRequired(),
		cli.NewFlag("ssh-ip", &updateConfigFlags.RemoteIP).
			SetUsage(red("Required, SSH address of the remote host")).SetRequired(),
		cli.NewFlag("ssh-port", &updateConfigFlags.RemotePort).
			SetUsage(yellow("Optional, SSH port of the remote host, default is 22")).SetValue(22),
		cli.NewFlag("ssh-user", &updateConfigFlags.RemoteUser).
			SetUsage(yellow("Optional, Username of user at remote host")),
		cli.NewFlag("ssh-key", &updateConfigFlags.SSHKey).
			SetUsage(yellow("Optional, Path to SSH private key for SSH Key Authentication")),
		cli.NewFlag("utility-path", &updateConfigFlags.BinUtilityPath).SetRequired().
			SetUsage(red("Required, local path of pastel-utility file ")),
		cli.NewFlag("bin", &updateConfigFlags.BinComponentPath).SetRequired().
			SetUsage(red("Required, local path to the local binary (pasteld, pastel-cli, rq-service, supernode) file  or a folder of binary to remote host")),
	}

	commandMessage := "Update " + string(updateCmd)

	commandFlags := commonFlags
	if updateCmd == updateSuperNodeRemote {
		commandFlags = append(commandFlags, remoteFlags...)
	}

	subCommand := cli.NewCommand(updateCommandName[updateCmd])
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
	config.OpMode = "update"

	updateNodeSubCommand := setupUpdateSubCommand(config, updateNode, runUpdateNodeSubCommand)
	updateWalletNnodeSubCommand := setupUpdateSubCommand(config, updateWalletNode, runUpdateWalletNodeSubCommand)

	updateSuperNodeRemoteSubCommand := setupUpdateSubCommand(config, updateSuperNodeRemote, runUpdateSuperNodeRemoteSubCommand)
	updateSuperNodeSubCommand := setupUpdateSubCommand(config, updateSuperNode, runUpdateSuperNodeSubCommand)
	updateSuperNodeSubCommand.AddSubcommands(updateSuperNodeRemoteSubCommand)

	// Add update command
	updateCommand := cli.NewCommand("update")
	updateCommand.SetUsage(blue("Perform update components for each service: Node, Walletnode and Supernode"))

	updateCommand.AddSubcommands(updateNodeSubCommand)
	updateCommand.AddSubcommands(updateWalletNnodeSubCommand)
	updateCommand.AddSubcommands(updateSuperNodeSubCommand)

	return updateCommand
}

func runUpdateSuperNodeRemoteSubCommand(ctx context.Context, config *configs.Config) (err error) {

	// Validate config
	if len(updateConfigFlags.RemoteIP) == 0 {
		return fmt.Errorf("--ssh-ip IP address - Required, SSH address of the remote host")
	}

	// Connect to remote
	var client *utils.Client
	log.WithContext(ctx).Infof("Connecting to remote host -> %s:%d...", updateConfigFlags.RemoteIP, updateConfigFlags.RemotePort)

	if len(updateConfigFlags.SSHKey) == 0 {
		username, password, _ := utils.Credentials(updateConfigFlags.RemoteUser, true)
		client, err = utils.DialWithPasswd(fmt.Sprintf("%s:%d", updateConfigFlags.RemoteIP, updateConfigFlags.RemotePort), username, password)
	} else {
		username, _, _ := utils.Credentials(updateConfigFlags.RemoteUser, false)
		client, err = utils.DialWithKey(fmt.Sprintf("%s:%d", updateConfigFlags.RemoteIP, updateConfigFlags.RemotePort), username, updateConfigFlags.SSHKey)
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

	if len(config.PastelExecDir) > 0 {
		remoteOptions = fmt.Sprintf("--dir %s", config.PastelExecDir)
	}

	if len(config.WorkingDir) > 0 {
		remoteOptions = fmt.Sprintf("%s --work-dir %s", remoteOptions, config.WorkingDir)
	}

	if len(config.UserPw) > 0 {
		remoteOptions = fmt.Sprintf("%s --user-pw %s", remoteOptions, config.UserPw)
	}

	stopSuperNodeCmd := fmt.Sprintf("%s stop supernode %s", pastelUtilityPath, remoteOptions)
	err = client.ShellCmd(ctx, stopSuperNodeCmd)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to stop Supernode services")
		return err
	}

	log.WithContext(ctx).Info("Successfully stop supernode at remote host")

	/* Copy the binary (pastel-cli, pasteld, pastel-cli, rq-service, supernode) from local folder to remote location to overwrite binary */
	fileInfo, err := os.Stat(updateConfigFlags.BinComponentPath)
	if err != nil {
		return err
	}

	if fileInfo.IsDir() {
		log.WithContext(ctx).Infof("Copying all files in %s to remote host %s", updateConfigFlags.BinComponentPath, config.PastelExecDir)
		files, err := ioutil.ReadDir(updateConfigFlags.BinComponentPath)
		if err != nil {
			return err
		}

		for _, file := range files {
			log.WithContext(ctx).Infof("Copying %s to remote host %s", file.Name(), config.PastelExecDir)
			if err := client.Scp(filepath.Join(updateConfigFlags.BinComponentPath, file.Name()), filepath.Join(config.PastelExecDir, file.Name())); err != nil {
				return err
			}
		}
	} else {
		log.WithContext(ctx).Infof("Copying file %s to %s at remote host", updateConfigFlags.BinComponentPath, config.PastelExecDir)
		if err := client.Scp(updateConfigFlags.BinComponentPath, filepath.Join(config.PastelExecDir, fileInfo.Name())); err != nil {
			return err
		}
	}

	log.WithContext(ctx).Info("Successfully copied app binary executable to remote host")

	/* Start service supernode again */
	log.WithContext(ctx).Info("Starting Supernode service ...")

	if len(updateConfigFlags.MasterNodeName) > 0 {
		remoteOptions = fmt.Sprintf("--name=%s", updateConfigFlags.MasterNodeName)
	}

	startSuperNodeCmd := fmt.Sprintf("%s start supernode %s", pastelUtilityPath, remoteOptions)

	err = client.ShellCmd(ctx, startSuperNodeCmd)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to start Supernode services")
		return err
	}

	return nil
}

func runUpdateSuperNodeSubCommand(ctx context.Context, config *configs.Config) (err error) {

	log.WithContext(ctx).Info("Stopping SuperNode service ...")
	runStopSuperNodeSubCommand(ctx, config)

	log.WithContext(ctx).Info("Updating SuperNode component ...")
	if err = runComponentsInstall(ctx, config, constants.SuperNode); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to update supernode component")
		return err
	}

	log.WithContext(ctx).Info("Starting SuperNode service ...")
	runLocalSuperNodeSubCommand(ctx, config)

	return nil
}

func runUpdateNodeSubCommand(ctx context.Context, config *configs.Config) (err error) {

	log.WithContext(ctx).Info("Stopping Node service ...")
	runStopNodeSubCommand(ctx, config)

	log.WithContext(ctx).Info("Updating node component ...")
	if err = runComponentsInstall(ctx, config, constants.PastelD); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to update node component")
		return err
	}

	log.WithContext(ctx).Info("Starting Node service ...")
	runStartNodeSubCommand(ctx, config)

	return nil
}

func runUpdateWalletNodeSubCommand(ctx context.Context, config *configs.Config) (err error) {

	log.WithContext(ctx).Info("Stopping Wallet Node service ...")
	runStopWalletSubCommand(ctx, config)

	log.WithContext(ctx).Info("Updating walletnode component ...")
	if err = runComponentsInstall(ctx, config, constants.WalletNode); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to update wallet node component")
		return err
	}

	log.WithContext(ctx).Info("Starting Wallet Node service ...")
	runStartWalletSubCommand(ctx, config)

	return nil
}
