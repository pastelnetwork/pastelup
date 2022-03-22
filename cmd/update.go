package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	cp "github.com/otiai10/copy"
	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pastelnetwork/pastelup/configs"
	"github.com/pastelnetwork/pastelup/constants"
	"github.com/pastelnetwork/pastelup/utils"
)

type updateCommand uint8

const (
	updateNode updateCommand = iota
	updateWalletNode
	updateSuperNode
	updateDDService
	updateRQService
	updateWNService
	updateSNService
	updateRemote
)

var (
	updateCommandName = map[updateCommand]string{
		updateNode:       "node",
		updateWalletNode: "walletnode",
		updateSuperNode:  "supernode",
		updateDDService:  "dd-service",
		updateRQService:  "rq-service",
		updateWNService:  "walletnode-service",
		updateSNService:  "supernode-service",
		updateRemote:     "remote",
	}
	updateCommandMessage = map[updateCommand]string{
		updateNode:       "Update Node",
		updateWalletNode: "Update Walletnode",
		updateSuperNode:  "Update Supernode",
		updateDDService:  "Update DupeDetection service only",
		updateRQService:  "Update RaptorQ service only",
		updateWNService:  "Update Walletnode service only",
		updateSNService:  "Update Supernode service only",
		updateRemote:     "Update on Remote host",
	}
	updateServicesToStop = map[constants.ToolType][]constants.ToolType{
		constants.PastelD: {constants.PastelD},
		constants.WalletNode: {
			constants.PastelD,
			constants.WalletNode,
			constants.RQService,
		},
		constants.SuperNode: {
			constants.PastelD,
			constants.SuperNode,
			constants.DDService,
			constants.DDImgService,
			constants.RQService,
		},
		constants.DDService: {
			constants.DDService,
			constants.DDImgService,
		},
		constants.RQService: {constants.RQService},
	}
)

func setupUpdateSubCommand(config *configs.Config,
	updateCmd updateCommand, remote bool,
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

		cli.NewFlag("clean", &config.Clean).SetAliases("c").
			SetUsage(green("Optional, Clean .pastel folder")),
	}

	var dirsFlags []*cli.Flag

	if !remote {
		dirsFlags = []*cli.Flag{
			cli.NewFlag("dir", &config.PastelExecDir).SetAliases("d").
				SetUsage(green("Optional, Location where to create pastel node directory")).SetValue(config.Configurer.DefaultPastelExecutableDir()),
			cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
				SetUsage(green("Optional, Location where to create working directory")).SetValue(config.Configurer.DefaultWorkingDir()),
			cli.NewFlag("archive-dir", &config.ArchiveDir).
				SetUsage(green("Optional, Location where to store archived backup before update")).SetValue(config.Configurer.DefaultArchiveDir()),
		}
	} else {
		dirsFlags = []*cli.Flag{
			cli.NewFlag("dir", &config.PastelExecDir).SetAliases("d").
				SetUsage(green("Optional, Location where to create pastel node directory on the remote computer (default: $HOME/pastel)")),
			cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
				SetUsage(green("Optional, Location where to create working directory on the remote computer (default: $HOME/.pastel)")),
			cli.NewFlag("archive-dir", &config.ArchiveDir).
				SetUsage(green("Optional, Location where to store archived backup before update on the remote computer (default: $HOME/.pastel_archive)")),
		}
	}

	userFlags := []*cli.Flag{
		cli.NewFlag("user-pw", &config.UserPw).
			SetUsage(green("Optional, password of current sudo user - so no sudo password request is prompted")),
	}

	superNodeFlags := []*cli.Flag{
		cli.NewFlag("name", &flagMasterNodeName).
			SetUsage(red("Required, name of the Masternode to start (and create or update in the masternode.conf if --create or --update are specified)")).SetRequired(),
	}

	remoteFlags := []*cli.Flag{
		cli.NewFlag("ssh-ip", &config.RemoteIP).
			SetUsage(red("Required, SSH address of the remote host")).SetRequired(),
		cli.NewFlag("ssh-port", &config.RemotePort).
			SetUsage(yellow("Optional, SSH port of the remote host, default is 22")).SetValue(22),
		cli.NewFlag("ssh-user", &config.RemoteUser).
			SetUsage(yellow("Optional, Username of user at remote host")),
		cli.NewFlag("ssh-user-pw", &config.UserPw).
			SetUsage(red("Required, password of remote user - so no sudo password request is prompted")).SetRequired(),
		cli.NewFlag("ssh-key", &config.RemoteSSHKey).
			SetUsage(yellow("Optional, Path to SSH private key for SSH Key Authentication")),
		cli.NewFlag("bin", &config.BinComponentPath).SetRequired().
			SetUsage(red("Required, local path to the local binary (pasteld, pastel-cli, rq-service, supernode) file or a folder of binary to remote host")),
	}

	var commandName, commandMessage string
	if !remote {
		commandName = updateCommandName[updateCmd]
		commandMessage = updateCommandMessage[updateCmd]
	} else {
		commandName = updateCommandName[updateRemote]
		commandMessage = updateCommandMessage[updateRemote]
	}

	commandFlags := append(dirsFlags, commonFlags[:]...)
	if updateCmd == updateSuperNode {
		commandFlags = append(commandFlags, superNodeFlags...)
	}
	if remote {
		commandFlags = append(commandFlags, remoteFlags[:]...)
	} else if updateCmd == updateSuperNode {
		commandFlags = append(commandFlags, userFlags...)
	}

	subCommand := cli.NewCommand(commandName)
	subCommand.SetUsage(cyan(commandMessage))
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

	updateNodeSubCommand := setupUpdateSubCommand(config, updateNode, false, runUpdateNodeSubCommand)
	updateWalletNodeSubCommand := setupUpdateSubCommand(config, updateWalletNode, false, runUpdateWalletNodeSubCommand)
	updateSuperNodeSubCommand := setupUpdateSubCommand(config, updateSuperNode, false, runUpdateSuperNodeSubCommand)
	updateRQServiceSubCommand := setupUpdateSubCommand(config, updateRQService, false, runUpdateRQServiceSubCommand)
	updateDDServiceSubCommand := setupUpdateSubCommand(config, updateDDService, false, runUpdateDDServiceSubCommand)
	updateWNServiceSubCommand := setupUpdateSubCommand(config, updateWNService, false, runUpdateWNServiceSubCommand)
	updateSNServiceSubCommand := setupUpdateSubCommand(config, updateSNService, false, runUpdateSNServiceSubCommand)

	updateNodeSubCommand.AddSubcommands(setupUpdateSubCommand(config, updateNode, true, runUpdateRemoteSubCommand))
	updateWalletNodeSubCommand.AddSubcommands(setupUpdateSubCommand(config, updateWalletNode, true, runUpdateRemoteSubCommand))
	updateSuperNodeSubCommand.AddSubcommands(setupUpdateSubCommand(config, updateSuperNode, true, runUpdateRemoteSubCommand))
	updateRQServiceSubCommand.AddSubcommands(setupUpdateSubCommand(config, updateRQService, true, runUpdateRemoteSubCommand))
	updateDDServiceSubCommand.AddSubcommands(setupUpdateSubCommand(config, updateDDService, true, runUpdateRemoteSubCommand))
	updateWNServiceSubCommand.AddSubcommands(setupUpdateSubCommand(config, updateWNService, true, runUpdateRemoteSubCommand))
	updateSNServiceSubCommand.AddSubcommands(setupUpdateSubCommand(config, updateSNService, true, runUpdateRemoteSubCommand))

	// Add update command
	updateCommand := cli.NewCommand("update")
	updateCommand.SetUsage(blue("Perform update components for each service: Node, Walletnode and Supernode"))

	updateCommand.AddSubcommands(updateNodeSubCommand)
	updateCommand.AddSubcommands(updateWalletNodeSubCommand)
	updateCommand.AddSubcommands(updateSuperNodeSubCommand)
	updateCommand.AddSubcommands(updateRQServiceSubCommand)
	updateCommand.AddSubcommands(updateDDServiceSubCommand)
	updateCommand.AddSubcommands(updateWNServiceSubCommand)
	updateCommand.AddSubcommands(updateSNServiceSubCommand)

	return updateCommand
}

func runUpdateRemoteSubCommand(ctx context.Context, config *configs.Config) (err error) {

	// Connect to remote
	client, err := prepareRemoteSession(ctx, config)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to prepare remote session")
		return
	}
	defer client.Close()

	// in case config.BinComponentPath empty then execute command at remote host to upgrade supernode
	if len(config.BinComponentPath) == 0 {
		log.WithContext(ctx).Info("Upgrading supernode at remote host ...")

		updateOptions := ""

		if len(config.PastelExecDir) > 0 {
			updateOptions = fmt.Sprintf("--dir %s", config.PastelExecDir)
		}

		if len(config.WorkingDir) > 0 {
			updateOptions = fmt.Sprintf("%s --work-dir %s", updateOptions, config.WorkingDir)
		}

		if config.Force {
			updateOptions = fmt.Sprintf("%s --force", updateOptions)
		}

		if len(config.UserPw) > 0 {
			updateOptions = fmt.Sprintf("--user-pw %s", config.UserPw)
		}

		if len(flagMasterNodeName) > 0 {
			updateOptions = fmt.Sprintf("%s --name %s", updateOptions, flagMasterNodeName)
		}

		if len(config.Version) > 0 {
			updateOptions = fmt.Sprintf("%s --release=%s", updateOptions, config.Version)
		}

		updateSuperNodeCmd := fmt.Sprintf("yes Y | %s update supernode %s", constants.RemotePastelupPath, updateOptions)
		err = client.ShellCmd(ctx, updateSuperNodeCmd)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to update Supernode services")
			return err
		}

	} else {
		/* Stop supernode services using pastelup */
		log.WithContext(ctx).Info("Stopping Supernode service ...")

		remoteOptions := ""

		if len(config.PastelExecDir) > 0 {
			remoteOptions = fmt.Sprintf("--dir %s", config.PastelExecDir)
		}

		if len(config.WorkingDir) > 0 {
			remoteOptions = fmt.Sprintf("%s --work-dir %s", remoteOptions, config.WorkingDir)
		}

		stopSuperNodeCmd := fmt.Sprintf("%s stop supernode %s", constants.RemotePastelupPath, remoteOptions)
		err = client.ShellCmd(ctx, stopSuperNodeCmd)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to stop Supernode services")
			return err
		}

		log.WithContext(ctx).Info("Successfully stop supernode at remote host")

		/* Copy the binary (pastel-cli, pasteld, pastel-cli, rq-service, supernode) from local folder to remote location to overwrite binary */
		fileInfo, err := os.Stat(config.BinComponentPath)
		if err != nil {
			return err
		}

		if fileInfo.IsDir() {
			log.WithContext(ctx).Infof("Copying all files in %s to remote host %s", config.BinComponentPath, config.PastelExecDir)
			files, err := ioutil.ReadDir(config.BinComponentPath)
			if err != nil {
				log.WithContext(ctx).WithError(err).Error("Failed to read directory ", config.BinComponentPath)
				return err
			}

			for _, file := range files {
				log.WithContext(ctx).Infof("Copying %s to remote host %s", file.Name(), config.PastelExecDir)
				sourceBin := filepath.Join(config.BinComponentPath, file.Name())
				destBin := filepath.Join(config.PastelExecDir, file.Name())

				if err := client.Scp(sourceBin, destBin); err != nil {
					log.WithContext(ctx).WithError(err).Error("Failed to copy file ", file.Name())
					return err
				}

				// chmod +x for the copied file
				if err := client.ShellCmd(ctx, destBin); err != nil {
					log.WithContext(ctx).WithError(err).Error("Failed to chmod +x file ", file.Name())
					return err
				}
			}
		} else {
			log.WithContext(ctx).Infof("Copying file %s to %s at remote host", config.BinComponentPath, config.PastelExecDir)
			destBin := filepath.Join(config.PastelExecDir, fileInfo.Name())
			if err := client.Scp(config.BinComponentPath, destBin); err != nil {
				log.WithContext(ctx).WithError(err).Error("Failed to copy file ", fileInfo.Name())
				return err
			}

			// chmod +x copied file
			if err := client.ShellCmd(ctx, fmt.Sprintf("chmod +x %s", destBin)); err != nil {
				log.WithContext(ctx).WithError(err).Error("Failed to chmod +x file ", fileInfo.Name())
				return err
			}
		}
		log.WithContext(ctx).Info("Successfully copied app binary executable to remote host")

		/* Start service supernode again */
		log.WithContext(ctx).Info("Starting Supernode service ...")

		if len(flagMasterNodeName) > 0 {
			remoteOptions = fmt.Sprintf("--name=%s", flagMasterNodeName)
		}

		startSuperNodeCmd := fmt.Sprintf("%s start supernode %s", constants.RemotePastelupPath, remoteOptions)

		err = client.ShellCmd(ctx, startSuperNodeCmd)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to start Supernode services")
			return err
		}
	}

	return nil
}

func runUpdateNodeSubCommand(ctx context.Context, config *configs.Config) (err error) {
	return stopAndUpdateService(ctx, config, constants.PastelD, true, false, true)
}

func runUpdateSuperNodeSubCommand(ctx context.Context, config *configs.Config) (err error) {
	return stopAndUpdateService(ctx, config, constants.SuperNode, true, true, true)
}

func runUpdateWalletNodeSubCommand(ctx context.Context, config *configs.Config) (err error) {
	return stopAndUpdateService(ctx, config, constants.WalletNode, true, false, true)
}

func runUpdateRQServiceSubCommand(ctx context.Context, config *configs.Config) (err error) {
	return stopAndUpdateService(ctx, config, constants.RQService, false, false, false)
}

func runUpdateDDServiceSubCommand(ctx context.Context, config *configs.Config) (err error) {
	return stopAndUpdateService(ctx, config, constants.DDService, false, true, false)
}

func runUpdateWNServiceSubCommand(ctx context.Context, config *configs.Config) (err error) {
	return stopAndUpdateService(ctx, config, constants.WalletNode, false, false, false)
}

func runUpdateSNServiceSubCommand(ctx context.Context, config *configs.Config) (err error) {
	return stopAndUpdateService(ctx, config, constants.SuperNode, false, false, false)
}

func stopAndUpdateService(ctx context.Context, config *configs.Config, updateCommand constants.ToolType,
	backUpWorkDir bool, backUpDDDir bool, withDependencies bool) error {

	log.WithContext(ctx).Infof("Updating %s component ...", string(updateCommand))

	var servicesToStop []constants.ToolType
	if withDependencies {
		servicesToStop = appToServiceMap[updateCommand]
	} else {
		servicesToStop = append(servicesToStop, updateCommand)
	}

	err := stopServicesWithConfirmation(ctx, config, servicesToStop)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to stop dependent services")
		return err
	}
	if backUpWorkDir {
		err = archiveWorkDir(ctx, config)
		if err != nil {
			return err
		}
	}
	if backUpDDDir {
		if err = archiveDDDir(ctx, config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to run extra step")
			return err
		}
	}
	if err = updateSolution(ctx, config, updateCommand, withDependencies); err != nil {
		log.WithContext(ctx).WithError(err).Error(fmt.Printf("Failed to update '%v': %v", updateCommand, err))
		return err
	}
	log.WithContext(ctx).Infof("Successfully updated %s component and its dependencies", string(updateCommand))

	log.WithContext(ctx).Info("Updated services need to be restarted:")
	if updateCommand == constants.WalletNode ||
		updateCommand == constants.SuperNode {
		log.WithContext(ctx).Infof(blue("\t\t'pastelup start %s'"), string(updateCommand))
	} else {
		for _, srv := range servicesToStop {
			var cmdStr string
			if string(srv) == "pasteld" {
				cmdStr = "node"
			} else {
				cmdStr = string(srv)
			}
			log.WithContext(ctx).Infof(blue("\t\t'pastelup start %s'"), cmdStr)
		}
	}
	return nil
}

// updateSolution does the actual installation of the latest updateSolution - node, walletnode, supernode
func updateSolution(ctx context.Context, config *configs.Config, installCommand constants.ToolType, withDependencies bool) error {
	log.WithContext(ctx).Info(fmt.Sprintf("Downloading latest version of %v component ...", installCommand))
	if err := runServicesInstall(ctx, config, installCommand, withDependencies); err != nil {
		log.WithContext(ctx).WithError(err).Error(fmt.Sprintf("Failed to update %v component", installCommand))
		return err
	}
	return nil
}

// archiveWorkDir runs archive dir on the users work dir (i.e. ~/.pastel if on linux)
func archiveWorkDir(ctx context.Context, config *configs.Config) error {
	if err := archiveDir(ctx, config, config.WorkingDir, config.Configurer.WorkDir()); err != nil {
		log.WithContext(ctx).Error(fmt.Sprintf("Failed to archive %v directory: %v", config.WorkingDir, err))
		return err
	}
	if config.Clean {
		//pathToClean := path.Join(homeDir, workDir)
		log.WithContext(ctx).Infof("Clean flag set, cleaning work dir (%v)", config.WorkingDir)
		filesToPreserve := []string{"pastel.conf", "wallet.dat", "masternode.conf"}
		if err := utils.ClearDir(ctx, config.WorkingDir, filesToPreserve); err != nil {
			log.WithContext(ctx).Error(fmt.Sprintf("Failed to clean directory:  %v", err))
			return err
		}
	}
	log.WithContext(ctx).Info(fmt.Sprintf("Working directory %v archived", config.WorkingDir))
	return nil
}

// archiveDir is makes a copy of the specified dir to a new dir in ~/.pastel_archives dir
func archiveDir(ctx context.Context, config *configs.Config, dirToArchive, archivePrefix string) error {
	now := time.Now().Unix()
	archiveBaseDir := config.ArchiveDir
	archiveName := fmt.Sprintf("%s_archive_%v", archivePrefix, now)
	log.WithContext(ctx).Info(fmt.Sprintf("Archiving %v directory to %v as %v", dirToArchive, archiveBaseDir, archiveName))

	if exists := utils.CheckFileExist(archiveBaseDir); !exists {
		err := os.Mkdir(archiveBaseDir, 0755)
		if err != nil {
			return err
		}
	}

	archivePath := filepath.Join(archiveBaseDir, archiveName)
	err := cp.Copy(dirToArchive, archivePath)
	if err != nil {
		return err
	}

	log.WithContext(ctx).Info(fmt.Sprintf("Archived %v directory as %v", config.WorkingDir, archiveName))
	return nil
}

func archiveDDDir(ctx context.Context, config *configs.Config) error {
	homeDir := config.Configurer.DefaultHomeDir()
	dirToArchive := filepath.Join(homeDir, constants.DupeDetectionServiceDir)
	if err := archiveDir(ctx, config, dirToArchive, constants.DupeDetectionServiceDir); err != nil {
		log.WithContext(ctx).Error(fmt.Sprintf("Failed to archive %v directory: %v", dirToArchive, err))
		return err
	} else {
		log.WithContext(ctx).Info(fmt.Sprintf("%v directory archived", dirToArchive))
		return nil
	}
}
