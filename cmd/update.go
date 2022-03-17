package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

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
	updateSuperNodeRemote
	updateRQService
	updateDDService
)

var (
	updateCommandName = map[updateCommand]string{
		updateNode:            "node",
		updateWalletNode:      "walletnode",
		updateSuperNode:       "supernode",
		updateSuperNodeRemote: "remote",
		updateRQService:       "rq-service",
		updateDDService:       "dd-service",
	}
	updateDependencies = map[constants.ToolType][]constants.ToolType{
		constants.SuperNode: {
			constants.PastelD,
			constants.SuperNode,
			constants.RQService,
			constants.DDService,
			constants.WalletNode,
		},
		constants.WalletNode: {
			constants.PastelD,
			constants.RQService,
			constants.WalletNode,
		},
		constants.PastelD: {constants.PastelD},
	}
	updateServicesToStop = map[constants.ToolType][]constants.ToolType{
		constants.SuperNode: {
			constants.PastelD,
			constants.SuperNode,
			constants.DDService,
			constants.DDImgService,
			constants.RQService,
		},
		constants.WalletNode: {
			constants.PastelD,
			constants.WalletNode,
			constants.RQService,
		},
		constants.DDImgService: {
			constants.DDService,
			constants.DDImgService,
		},
		constants.RQService: {constants.RQService},
		constants.PastelD:   {constants.PastelD},
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

		cli.NewFlag("clean", &config.Clean).SetAliases("c").
			SetUsage(green("Optional, Clean .pastel folder")),
	}

	if updateCmd == updateSuperNodeRemote || updateCmd == updateSuperNode {
		commonFlags = append(commonFlags,
			cli.NewFlag("user-pw", &config.UserPw).
				SetUsage(green("Optional, password of current sudo user - so no sudo password request is prompted")),
		)
	}

	if updateCmd == updateSuperNodeRemote || updateCmd == updateSuperNode || updateCmd == updateWalletNode {
		commonFlags = append(commonFlags,
			cli.NewFlag("name", &flagMasterNodeName).
				SetUsage(red("Required, name of the Masternode to start (and create or update in the masternode.conf if --create or --update are specified)")).SetRequired(),
		)
	}

	remoteFlags := []*cli.Flag{
		cli.NewFlag("ssh-ip", &config.RemoteIP).
			SetUsage(red("Required, SSH address of the remote host")).SetRequired(),
		cli.NewFlag("ssh-port", &config.RemotePort).
			SetUsage(yellow("Optional, SSH port of the remote host, default is 22")).SetValue(22),
		cli.NewFlag("ssh-user", &config.RemoteUser).
			SetUsage(yellow("Optional, Username of user at remote host")),
		cli.NewFlag("ssh-key", &config.RemoteSSHKey).
			SetUsage(yellow("Optional, Path to SSH private key for SSH Key Authentication")),
		cli.NewFlag("bin", &config.BinComponentPath).SetRequired().
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
	updateRQServiceSubCommand := setupUpdateSubCommand(config, updateRQService, runUpdateRQServiceSubCommand)
	updateDDServiceSubCommand := setupUpdateSubCommand(config, updateDDService, runUpdateDDServiceSubCommand)

	// Add update command
	updateCommand := cli.NewCommand("update")
	updateCommand.SetUsage(blue("Perform update components for each service: Node, Walletnode and Supernode"))

	updateCommand.AddSubcommands(updateNodeSubCommand)
	updateCommand.AddSubcommands(updateWalletNnodeSubCommand)
	updateCommand.AddSubcommands(updateSuperNodeSubCommand)
	updateCommand.AddSubcommands(updateRQServiceSubCommand)
	updateCommand.AddSubcommands(updateDDServiceSubCommand)

	return updateCommand
}

func runUpdateSuperNodeRemoteSubCommand(ctx context.Context, config *configs.Config) (err error) {

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

func runUpdateSuperNodeSubCommand(ctx context.Context, config *configs.Config) (err error) {
	log.WithContext(ctx).Info("Updating SuperNode component ...")
	err = stopAndUpdateService(ctx, constants.SuperNode, config)
	if err != nil {
		return err
	}
	log.WithContext(ctx).Info("Successfully updated SuperNode component and its dependencies")
	return nil
}

func runUpdateNodeSubCommand(ctx context.Context, config *configs.Config) (err error) {
	log.WithContext(ctx).Info("Updating node component ...")
	err = stopAndUpdateService(ctx, constants.PastelD, config)
	if err != nil {
		return err
	}
	log.WithContext(ctx).Info("Successfully updated Node component and its dependencies")
	return nil
}

func runUpdateWalletNodeSubCommand(ctx context.Context, config *configs.Config) (err error) {
	log.WithContext(ctx).Info("Updating WalletNode component ...")
	err = stopAndUpdateService(ctx, constants.WalletNode, config)
	if err != nil {
		return err
	}
	log.WithContext(ctx).Info("Successfully updated WalletNode component and its dependencies")
	return nil
}

func stopAndUpdateService(ctx context.Context, service constants.ToolType, config *configs.Config) error {
	servicesToStop := updateServicesToStop[constants.WalletNode]
	err := stopServicesWithConfirmation(ctx, config, servicesToStop)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to stop dependent services")
		return err
	}
	err = archiveWorkDir(ctx, config)
	if err != nil {
		return err
	}
	servicesToUpdate := updateDependencies[constants.WalletNode]
	for _, service := range servicesToUpdate {
		err = updateService(ctx, config, service)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error(fmt.Printf("Failed to update dependent service '%v': %v", service, err))
			return err
		}
	}
	return nil
}

func runUpdateRQServiceSubCommand(ctx context.Context, config *configs.Config) (err error) {
	log.WithContext(ctx).Info("Updating RQ service...")
	servicesToStop := updateServicesToStop[constants.RQService]
	err = stopServicesWithConfirmation(ctx, config, servicesToStop)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to stop dependent services")
		return err
	}
	err = updateService(ctx, config, constants.RQService)
	if err != nil {
		return err
	}
	log.WithContext(ctx).Info("Successfully updated rq-service component")
	return nil
}

func runUpdateDDServiceSubCommand(ctx context.Context, config *configs.Config) (err error) {
	log.WithContext(ctx).Info("Updating DD service...")
	servicesToStop := updateServicesToStop[constants.DDService]
	err = stopServicesWithConfirmation(ctx, config, servicesToStop)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to stop dependent services")
		return err
	}
	homeDir := config.Configurer.DefaultHomeDir()
	dirToArchive := filepath.Join(homeDir, constants.DupeDetectionServiceDir)
	if archiveName, err := archiveDir(homeDir, dirToArchive, constants.DupeDetectionServiceDir); err != nil {
		log.WithContext(ctx).Error(fmt.Sprintf("Failed to archive %v directory: %v", dirToArchive, err))
	} else {
		log.WithContext(ctx).Info(fmt.Sprintf("Archived %v directory as %v", dirToArchive, archiveName))
	}
	err = updateService(ctx, config, constants.DDService)
	if err != nil {
		return err
	}
	log.WithContext(ctx).Info("Successfully updated dd-service component")
	return nil
}

// updateService does the actial install of the latest image of the specified service
func updateService(ctx context.Context, config *configs.Config, service constants.ToolType) error {
	log.WithContext(ctx).Info(fmt.Sprintf("Downloading latest version of %v component ...", service))
	if err := runComponentsInstall(ctx, config, service); err != nil {
		log.WithContext(ctx).WithError(err).Error(fmt.Sprintf("Failed to update %v component", service))
		return err
	}
	return nil
}

// archiveWorkDir runs archive dir on the users work dir (i.e. ~/.pastel if on linux)
func archiveWorkDir(ctx context.Context, config *configs.Config) error {
	homeDir := config.Configurer.DefaultHomeDir()
	dirToArchive := config.Configurer.DefaultWorkingDir()
	workDir := config.Configurer.WorkDir()
	archiveName, err := archiveDir(homeDir, dirToArchive, workDir)
	if err != nil {
		log.WithContext(ctx).Error(fmt.Sprintf("Failed to archive %v directory: %v", dirToArchive, err))
		return err
	}
	if config.Clean {
		pathToClean := path.Join(homeDir, workDir)
		log.WithContext(ctx).Infof("Clean flag set, cleaning work dir (%v)", pathToClean)
		filesToPreserve := []string{"pastel.conf", "wallet.dat", "masternode.conf"}
		err = utils.ClearDir(ctx, pathToClean, filesToPreserve)
		if err != nil {
			log.WithContext(ctx).Error(fmt.Sprintf("Failed to clean directory:  %v", err))
			return err
		}
	}
	log.WithContext(ctx).Info(fmt.Sprintf("Archived %v directory as %v", dirToArchive, archiveName))
	return nil
}

// archiveDir is makes a copy of the specified dir to a new dir in ~/.pastel_archives dir
func archiveDir(homeDir, dirToArchive, archiveSource string) (string, error) {
	now := time.Now().Unix()
	archiveBaseDir := homeDir + "/.pastel_archives"
	if exists := utils.CheckFileExist(archiveBaseDir); !exists {
		err := os.Mkdir(archiveBaseDir, 0755)
		if err != nil {
			return "", err
		}
	}
	archiveName := fmt.Sprintf("%s_archive_%v", archiveSource, now)
	archivePath := archiveBaseDir + "/" + archiveName
	cmd := fmt.Sprintf("cp -R %v %v", dirToArchive, archivePath)
	_, err := RunCMD("bash", "-c", cmd)
	if err != nil {
		return "", err
	}
	return archivePath, nil
}
