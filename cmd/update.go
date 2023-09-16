package cmd

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// archiveRetention is the number of archives to preserve before deleting old ones to avoid build up
// these are specific to the type of archive i.e. dd-service archives can have this number of archives alongside workdir archives
const archiveRetention = 3

const (
	updateNode updateCommand = iota
	updateWalletNode
	updateSuperNode
	updateRQService
	updateDDService
	updateWNService
	updateSNService
	updatePastelup
	remoteUpdate
	installService
	removeService
)

var (
	updateCommandName = map[updateCommand]string{
		updateNode:       "node",
		updateWalletNode: "walletnode",
		updateSuperNode:  "supernode",
		updateRQService:  "rq-service",
		updateDDService:  "dd-service",
		updateWNService:  "walletnode-service",
		updateSNService:  "supernode-service",
		updatePastelup:   "pastelup",
		remoteUpdate:     "remote",
		installService:   "install-service",
		removeService:    "remove-service",
	}
	updateCommandMessage = map[updateCommand]string{
		updateNode:       "Update Node",
		updateWalletNode: "Update Walletnode",
		updateSuperNode:  "Update Supernode",
		updateRQService:  "Update RaptorQ service only",
		updateDDService:  "Update DupeDetection service only",
		updateWNService:  "Update Walletnode service only",
		updateSNService:  "Update Supernode service only",
		updatePastelup:   "Update Pastelup",
		remoteUpdate:     "Update on Remote host",
		installService:   "Install managed service",
		removeService:    "Remove managed service",
	}
)

func setupUpdateSubCommand(config *configs.Config,
	updateCommand updateCommand, remote bool,
	f func(context.Context, *configs.Config) error,
) *cli.Command {

	commonFlags := []*cli.Flag{
		cli.NewFlag("version", &config.Version).SetAliases("v").
			SetUsage(green("Optional, Pastel version to install, default is latest release either mainnet or testnet, depending on the network flag")),
		cli.NewFlag("force", &config.Force).SetAliases("f").
			SetUsage(green("Optional, Force to overwrite config files and re-download ZKSnark parameters")),
		cli.NewFlag("regen-rpc", &config.RegenRPC).
			SetUsage(green("Optional, regenerate the random rpc user, password and chosen port. This will happen automatically if not defined already in your pastel.conf file")),
		cli.NewFlag("ignore-dependencies", &flagIgnoreDependencies).
			SetUsage(green("Optional, ignore checking dependencies and continue installation even if dependencies are not met")),
		cli.NewFlag("clean", &config.Clean).SetAliases("c").
			SetUsage(green("Optional, Clean .pastel folder")),
		cli.NewFlag("no-backup", &config.NoBackup).
			SetUsage(green("Optional, skip backing up existing setup files before updating workspace")),
		cli.NewFlag("backup-all", &config.BackupAll).
			SetUsage(green("Optional, back up ALL existing files before updating workspace")),
		cli.NewFlag("skip-system-update", &config.SkipSystemUpdate).
			SetUsage(green("Optional, Skip System Update skips linux apt-update")).SetValue(true),
		cli.NewFlag("skip-dd-packages-update", &config.SkipDDPackagesUpdate).
			SetUsage(green("Optional, Skip Update of python packages during dd-service update")).SetValue(true),
		cli.NewFlag("skip-dd-supporting-files-update", &config.SkipDDSupportingDilesUpdate).
			SetUsage(green("Optional, Skip Download and Update of dd-service supporting files")).SetValue(true),
	}

	userFlags := []*cli.Flag{
		cli.NewFlag("user-pw", &config.UserPw).
			SetUsage(green("Optional, password of current sudo user - so no sudo password request is prompted")),
	}

	var dirsFlags []*cli.Flag
	var archDirsFlags []*cli.Flag

	if !remote {
		dirsFlags = []*cli.Flag{
			cli.NewFlag("dir", &config.PastelExecDir).SetAliases("d").
				SetUsage(green("Optional, Location of the pastel node directory")).SetValue(config.Configurer.DefaultPastelExecutableDir()),
			cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
				SetUsage(green("Optional, Location of the working directory")).SetValue(config.Configurer.DefaultWorkingDir()),
		}
	} else {
		dirsFlags = []*cli.Flag{
			cli.NewFlag("dir", &config.PastelExecDir).SetAliases("d").
				SetUsage(green("Optional, Location of the pastel node directory on the remote computer (default: $HOME/pastel)")),
			cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
				SetUsage(green("Optional, Location of the working directory on the remote computer (default: $HOME/.pastel)")),
		}
	}

	if !remote {
		archDirsFlags = []*cli.Flag{
			cli.NewFlag("archive-dir", &config.ArchiveDir).
				SetUsage(green("Optional, Location where to store archived backup before update")).SetValue(config.Configurer.DefaultArchiveDir()),
		}
	} else {
		archDirsFlags = []*cli.Flag{
			cli.NewFlag("archive-dir", &config.ArchiveDir).
				SetUsage(green("Optional, Location where to store archived backup before update on the remote computer (default: $HOME/.pastel_archive)")),
		}
	}

	remoteFlags := []*cli.Flag{
		cli.NewFlag("ssh-ip", &config.RemoteIP).
			SetUsage(red("Required (if inventory not used), SSH address of the remote host")),
		cli.NewFlag("ssh-port", &config.RemotePort).
			SetUsage(yellow("Optional, SSH port of the remote host, default is 22")).SetValue(22),
		cli.NewFlag("ssh-user", &config.RemoteUser).
			SetUsage(yellow("Optional, Username of user at remote host")),
		cli.NewFlag("ssh-user-pw", &config.UserPw).
			SetUsage(yellow("Optional, password of remote user - so no sudo password request is prompted")),
		cli.NewFlag("ssh-key", &config.RemoteSSHKey).
			SetUsage(yellow("Optional, Path to SSH private key for SSH Key Authentication")),
		cli.NewFlag("inventory", &config.InventoryFile).
			SetUsage(yellow("Required (if ssh-ip not used), Path to the file with configuration of the remote hosts")),
	}

	systemServiceFlags := []*cli.Flag{
		cli.NewFlag("tool", &config.ServiceTool).
			SetUsage(red("Required (either this or --solution), Name of the Pastel application to set as a system service, " +
				"One of: node, masternode, supernode, walletnode, dd-service, rq-service, hermes, bridge. " +
				"NOTE: flags supernode and walletnode will only set service for corresponding application itself")),
		cli.NewFlag("solution", &config.ServiceSolution).
			SetUsage(red("Required (either this or --tool), Name of the Pastel application set (solution) to set as a system services, " +
				"One of: supernode or walletnode")),
		cli.NewFlag("autostart", &config.EnableService).
			SetUsage(yellow("Optional, Enable service for auto start after OS boot")),
		cli.NewFlag("start", &config.StartService).
			SetUsage(yellow("Optional, Start service right away")),
	}

	var commandName, commandMessage string
	if !remote {
		commandName = updateCommandName[updateCommand]
		commandMessage = updateCommandMessage[updateCommand]
	} else {
		commandName = updateCommandName[remoteUpdate]
		commandMessage = updateCommandMessage[remoteUpdate]
	}

	var commandFlags []*cli.Flag

	if updateCommand == installService || updateCommand == removeService {
		commandFlags = append(systemServiceFlags, dirsFlags[:]...)
	} else {
		commandFlags = append(dirsFlags, archDirsFlags[:]...)
		commandFlags = append(commandFlags, commonFlags[:]...)
	}

	if remote {
		commandFlags = append(commandFlags, remoteFlags[:]...)
	} else {
		commandFlags = append(commandFlags, userFlags[:]...)
	}

	subCommand := cli.NewCommand(commandName)
	subCommand.SetUsage(cyan(commandMessage))
	subCommand.AddFlags(commandFlags...)
	addLogFlags(subCommand, config)

	if f != nil {
		subCommand.SetActionFunc(func(ctx context.Context, _ []string) error {
			ctx, err := configureLogging(ctx, commandMessage, config)
			if err != nil {
				return fmt.Errorf("failed to configure logging option - %v", err)
			}

			sys.RegisterInterruptHandler(func() {
				log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
				os.Exit(0)
			})

			if !remote {
				if err = ParsePastelConf(ctx, config); err != nil {
					return err
				}
			}
			log.WithContext(ctx).Infof("Update started for network mode '%v'...", config.Network)
			if config.Version != "" {
				log.WithContext(ctx).Infof("Version set to '%v", config.Version)
			}
			if err = f(ctx, config); err != nil {
				return err
			}
			log.WithContext(ctx).Info("Finished successfully!")
			return nil
		})
	}

	return subCommand
}

func setupUpdateCommand(config *configs.Config) *cli.Command {
	config.OpMode = "update"

	updatePastelupSubCommand := setupUpdateSubCommand(config, updatePastelup, false, runUpdatePastelup)
	updateNodeSubCommand := setupUpdateSubCommand(config, updateNode, false, runUpdateNodeSubCommand)
	updateWalletNodeSubCommand := setupUpdateSubCommand(config, updateWalletNode, false, runUpdateWalletNodeSubCommand)
	updateSuperNodeSubCommand := setupUpdateSubCommand(config, updateSuperNode, false, runUpdateSuperNodeSubCommand)
	updateRQServiceSubCommand := setupUpdateSubCommand(config, updateRQService, false, runUpdateRQServiceSubCommand)
	updateDDServiceSubCommand := setupUpdateSubCommand(config, updateDDService, false, runUpdateDDServiceSubCommand)
	updateWNServiceSubCommand := setupUpdateSubCommand(config, updateWNService, false, runUpdateWNServiceSubCommand)
	updateSNServiceSubCommand := setupUpdateSubCommand(config, updateSNService, false, runUpdateSNServiceSubCommand)

	updateNodeSubCommand.AddSubcommands(setupUpdateSubCommand(config, updateNode, true, runUpdateRemoteNode))
	updateWalletNodeSubCommand.AddSubcommands(setupUpdateSubCommand(config, updateWalletNode, true, runUpdateRemoteWalletNode))
	updateSuperNodeSubCommand.AddSubcommands(setupUpdateSubCommand(config, updateSuperNode, true, runUpdateRemoteSuperNode))
	updateRQServiceSubCommand.AddSubcommands(setupUpdateSubCommand(config, updateRQService, true, runUpdateRemoteRQService))
	updateDDServiceSubCommand.AddSubcommands(setupUpdateSubCommand(config, updateDDService, true, runUpdateRemoteDDService))
	updateWNServiceSubCommand.AddSubcommands(setupUpdateSubCommand(config, updateWNService, true, runUpdateRemoteWNService))
	updateSNServiceSubCommand.AddSubcommands(setupUpdateSubCommand(config, updateSNService, true, runUpdateRemoteSNService))

	installServiceSubCommand := setupUpdateSubCommand(config, installService, false, installSystemService)
	installServiceSubCommand.AddSubcommands(setupUpdateSubCommand(config, installService, true, installSystemServiceRemote))
	//removeServiceSubCommand := setupUpdateSubCommand(config, removeService, false, removeSystemService)

	// Add update command
	updateCommand := cli.NewCommand("update")
	updateCommand.SetUsage(blue("Perform update components for each service: Node, Walletnode and Supernode"))
	updateCommand.AddSubcommands(updatePastelupSubCommand)
	updateCommand.AddSubcommands(updateNodeSubCommand)
	updateCommand.AddSubcommands(updateWalletNodeSubCommand)
	updateCommand.AddSubcommands(updateSuperNodeSubCommand)
	updateCommand.AddSubcommands(updateRQServiceSubCommand)
	updateCommand.AddSubcommands(updateDDServiceSubCommand)
	updateCommand.AddSubcommands(updateWNServiceSubCommand)
	updateCommand.AddSubcommands(updateSNServiceSubCommand)
	updateCommand.AddSubcommands(installServiceSubCommand)
	//updateCommand.AddSubcommands(removeServiceSubCommand)

	return updateCommand
}

func runUpdatePastelup(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Downloading latest version of pastelup tool ...")
	return installPastelUp(ctx, config)
}
func runUpdateRemoteNode(ctx context.Context, config *configs.Config) (err error) {
	return runRemoteUpdate(ctx, config, "node")
}
func runUpdateRemoteWalletNode(ctx context.Context, config *configs.Config) (err error) {
	return runRemoteUpdate(ctx, config, "walletnode")
}
func runUpdateRemoteSuperNode(ctx context.Context, config *configs.Config) (err error) {
	return runRemoteUpdate(ctx, config, "supernode")
}
func runUpdateRemoteRQService(ctx context.Context, config *configs.Config) (err error) {
	return runRemoteUpdate(ctx, config, "rq-service")
}
func runUpdateRemoteDDService(ctx context.Context, config *configs.Config) (err error) {
	return runRemoteUpdate(ctx, config, "dd-service")
}
func runUpdateRemoteWNService(ctx context.Context, config *configs.Config) (err error) {
	return runRemoteUpdate(ctx, config, "walletnode-service")
}
func runUpdateRemoteSNService(ctx context.Context, config *configs.Config) (err error) {
	return runRemoteUpdate(ctx, config, "supernode-service")
}

func runRemoteUpdate(ctx context.Context, config *configs.Config, tool string) (err error) {
	if len(config.RemoteIP) == 0 {
		log.WithContext(ctx).Fatal("remote IP is required")
		return fmt.Errorf("remote IP is required")
	}
	log.WithContext(ctx).Infof("Updating remote %s", tool)

	updateOptions := tool

	if len(config.PastelExecDir) > 0 {
		updateOptions = fmt.Sprintf("%s --dir %s", updateOptions, config.PastelExecDir)
	}

	if len(config.WorkingDir) > 0 {
		updateOptions = fmt.Sprintf("%s --work-dir %s", updateOptions, config.WorkingDir)
	}

	if config.Force {
		updateOptions = fmt.Sprintf("%s --force", updateOptions)
	}

	if len(config.UserPw) > 0 {
		updateOptions = fmt.Sprintf("%s --user-pw %s", updateOptions, config.UserPw)
	}

	if len(config.Version) > 0 {
		updateOptions = fmt.Sprintf("%s --version=%s", updateOptions, config.Version)
	}

	updateSuperNodeCmd := fmt.Sprintf("yes Y | %s update %s", constants.RemotePastelupPath, updateOptions)
	if _, err := executeRemoteCommandsWithInventory(ctx, config, []string{updateSuperNodeCmd}, false, false); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to update %s on remote host", tool)
	}
	log.WithContext(ctx).Infof("Remote %s updated", tool)

	return nil
}

func installSystemServiceRemote(ctx context.Context, config *configs.Config) (err error) {

	//--tool value
	//--solution value
	//--autostart
	//--start

	serviceInstallOptions := "install-service"

	if len(config.ServiceTool) > 0 {
		serviceInstallOptions = fmt.Sprintf("%s --tool %s", serviceInstallOptions, config.ServiceTool)
		log.WithContext(ctx).Infof("Installing %s as systemd services on remote host(s)", config.ServiceTool)
	} else if len(config.ServiceSolution) > 0 {
		serviceInstallOptions = fmt.Sprintf("%s --solution %s", serviceInstallOptions, config.ServiceSolution)
		log.WithContext(ctx).Infof("Installing %s as systemd services on remote host(s)", config.ServiceSolution)
	}

	if config.EnableService {
		serviceInstallOptions = fmt.Sprintf("%s --autostart", serviceInstallOptions)
	}
	if config.StartService {
		serviceInstallOptions = fmt.Sprintf("%s --start", serviceInstallOptions)
	}

	if len(config.PastelExecDir) > 0 {
		serviceInstallOptions = fmt.Sprintf("%s --dir %s", serviceInstallOptions, config.PastelExecDir)
	}

	if len(config.WorkingDir) > 0 {
		serviceInstallOptions = fmt.Sprintf("%s --work-dir %s", serviceInstallOptions, config.WorkingDir)
	}

	if config.Force {
		serviceInstallOptions = fmt.Sprintf("%s --force", serviceInstallOptions)
	}

	if len(config.UserPw) > 0 {
		serviceInstallOptions = fmt.Sprintf("%s --user-pw %s", serviceInstallOptions, config.UserPw)
	}

	updateSuperNodeCmd := fmt.Sprintf("yes Y | %s update %s", constants.RemotePastelupPath, serviceInstallOptions)
	if _, err := executeRemoteCommandsWithInventory(ctx, config, []string{updateSuperNodeCmd}, false, false); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to set systemd services on remote host")
	}
	log.WithContext(ctx).Infof("Remote systemd services set")

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
	backUpPastelWorkDir bool, backUpDDWorkDir bool, withDependencies bool) error {

	if config.NoBackup && config.BackupAll {
		log.WithContext(ctx).Error("cannot use --no-backup and --backup-all together")
		return fmt.Errorf("cannot use --no-backup and --backup-all together")
	}

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
	// backup work dir is only done for Node (pasteld), SuperNode and WalletNode updates
	if backUpPastelWorkDir && !config.NoBackup {
		err = backUpWorkDir(ctx, config)
		if err != nil {
			return err
		}
	}
	// backup dd dir is only done for SuperNode and dd-service updates
	if backUpDDWorkDir && config.BackupAll {
		if err = backUpDDDir(ctx, config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to run extra step")
			return err
		}
	}
	if err = updateSolution(ctx, config, updateCommand, withDependencies); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to update '%v': %v", updateCommand, err)
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

// backUpWorkDir runs archive dir on the users work dir (i.e. ~/.pastel if on linux)
func backUpWorkDir(ctx context.Context, config *configs.Config) error {
	archivePrefix := config.Configurer.WorkDir()

	var whatToBackUp []string
	if !config.BackupAll {
		whatToBackUp = []string{"pastel.conf", "supernode.yml", "hermes.yml", "walletnode.yml", "bridge.yml",
			"wallet.dat", "masternode.conf", "mncache.dat", "mnpayments.dat", "blocks", "chainstate", "pastelkeys"}
	}
	if err := backUpDir(ctx, config, config.WorkingDir, archivePrefix, whatToBackUp); err != nil {
		log.WithContext(ctx).Error(fmt.Sprintf("Failed to archive %v directory: %v", config.WorkingDir, err))
		return err
	}
	if config.Clean {
		//pathToClean := path.Join(homeDir, workDir)
		log.WithContext(ctx).Infof("Clean flag set, cleaning work dir (%v)", config.WorkingDir)
		filesToPreserve := []string{
			"pastel.conf", "wallet.dat", "masternode.conf",
			"supernode.yml", "hermes.yml",
			"walletnode.yml", "bridge.yml"}
		dirsToPreserve := []string{"pastelkeys"}
		if _, err := utils.ClearDir(ctx, config.WorkingDir, filesToPreserve, dirsToPreserve, config.IsTestnet); err != nil {
			log.WithContext(ctx).Error(fmt.Sprintf("Failed to clean directory:  %v", err))
			return err
		}
	}
	log.WithContext(ctx).Info(fmt.Sprintf("Working directory %v archived", config.WorkingDir))
	return nil
}

// backUpDir is makes a copy of the specified dir to a new dir in ~/.pastel_archives dir
func backUpDir(ctx context.Context, config *configs.Config, dirToArchive, archivePrefix string, whatToBackUp []string) error {
	now := time.Now().Unix()
	archiveBaseDir := config.ArchiveDir
	archiveName := fmt.Sprintf("%s_archive_%v", archivePrefix, now)
	log.WithContext(ctx).Info(fmt.Sprintf("Archiving %v directory to %v as %v", dirToArchive, archiveBaseDir, archiveName))

	if exists := utils.CheckFileExist(archiveBaseDir); !exists {
		err := os.Mkdir(archiveBaseDir, 0755)
		if err != nil {
			log.WithContext(ctx).Error(fmt.Sprintf("Failed to create %v directory: %v", archiveBaseDir, err))
			return err
		}
	}

	archivePath := filepath.Join(archiveBaseDir, archiveName)

	if whatToBackUp == nil {
		err := cp.Copy(dirToArchive, archivePath)
		if err != nil {
			return err
		}
	} else {
		err := os.Mkdir(archivePath, 0755)
		if err != nil {
			log.WithContext(ctx).Error(fmt.Sprintf("Failed to create %v directory: %v", archiveBaseDir, err))
			return err
		}

		for _, name := range whatToBackUp {
			srcPath := filepath.Join(dirToArchive, name)
			dstPath := filepath.Join(archivePath, name)
			if exists := utils.CheckFileExist(srcPath); exists {
				err := cp.Copy(srcPath, dstPath)
				if err != nil {
					return err
				}
			}
		}
		if config.IsTestnet {
			archivePathTestnetSubDir := filepath.Join(archivePath, "testnet3")
			err := os.Mkdir(archivePathTestnetSubDir, 0755)
			if err != nil {
				log.WithContext(ctx).Error(fmt.Sprintf("Failed to create %v directory: %v", archivePathTestnetSubDir, err))
				return err
			}
			for _, name := range whatToBackUp {
				srcPath := filepath.Join(dirToArchive, "testnet3", name)
				dstPath := filepath.Join(archivePathTestnetSubDir, name)
				if exists := utils.CheckFileExist(srcPath); exists {
					err := cp.Copy(srcPath, dstPath)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	err := deleteOldArchives(ctx, archiveBaseDir, archivePrefix)
	if err != nil {
		log.WithContext(ctx).Error(fmt.Sprintf("Failed to remove old archives:  %v", err))
		return err
	}
	log.WithContext(ctx).Info(fmt.Sprintf("Archived %v directory as %v", config.WorkingDir, archiveName))
	return nil
}

func deleteOldArchives(ctx context.Context, archiveBaseDir string, archivePrefix string) error {
	// if we have more than ARCHIVE_RETENTION amount of archives, delete old ones to avoid build up
	var matchingArchives []fs.FileInfo
	files, err := os.ReadDir(archiveBaseDir)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.IsDir() && strings.HasPrefix(f.Name(), archivePrefix+"_archive_") {
			info, err := f.Info()
			if err != nil {
				return err
			}
			matchingArchives = append(matchingArchives, info)
		}
	}
	if len(matchingArchives) > archiveRetention {
		sort.Slice(matchingArchives, func(i, j int) bool {
			return matchingArchives[i].ModTime().Before(matchingArchives[j].ModTime())
		})
		archivesToDelete := len(matchingArchives) - archiveRetention
		i := 0
		for i < archivesToDelete {
			directoryToRemove := filepath.Join(archiveBaseDir, matchingArchives[i].Name())
			log.WithContext(ctx).Info(fmt.Sprintf("Deleting old arvhive %v to avoid build up: created at %v", directoryToRemove, matchingArchives[i].ModTime().Format(time.RFC3339)))
			err = os.RemoveAll(directoryToRemove)
			if err != nil {
				return err
			}
			i++
		}
	}
	return nil
}

func backUpDDDir(ctx context.Context, config *configs.Config) error {
	homeDir := config.Configurer.DefaultHomeDir()
	dirToArchive := filepath.Join(homeDir, constants.DupeDetectionServiceDir)
	if err := backUpDir(ctx, config, dirToArchive, constants.DupeDetectionServiceDir, []string{}); err != nil {
		log.WithContext(ctx).Error(fmt.Sprintf("Failed to archive %v directory: %v", dirToArchive, err))
		return err
	}
	log.WithContext(ctx).Info(fmt.Sprintf("%v directory archived", dirToArchive))
	return nil
}
