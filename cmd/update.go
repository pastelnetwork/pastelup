package cmd

import (
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
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

// ARCHIVE_RETENTION is the number of archives to preserve before deleting old ones to avoid build up
// these are specific to the type of archive i.e. dd-service archives can have this number of archives along side workdir archives
const ARCHIVE_RETENTION = 3

const (
	updateNode updateCommand = iota
	updateWalletNode
	updateSuperNode
	updateDDService
	updateRQService
	updateWNService
	updateSNService
	updateRemote
	updatePastelup
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
		updatePastelup:   "pastelup",
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
		updatePastelup:   "Update Pastelup",
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
			SetUsage(green("Optional, Pastel version to install")),
		cli.NewFlag("clean", &config.Clean).SetAliases("c").
			SetUsage(green("Optional, Clean .pastel folder")),
		cli.NewFlag("user-pw", &config.UserPw).
			SetUsage(green("Optional, password of current sudo user - so no sudo password request is prompted")),
		cli.NewFlag("no-backup", &config.NoBackup).
			SetUsage(green("Optional, skip backing up configuration files before updating workspace")),
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
		cli.NewFlag("inventory", &config.InventoryFile).
			SetUsage(red("Optional, Path to the file with configuration of the remote hosts")),
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
	if remote {
		commandFlags = append(commandFlags, remoteFlags[:]...)
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

			if config.Version == "" {
				log.WithContext(ctx).
					WithError(constants.NoVersionSetErr{}).
					Error("Failed to process update command")
				return err
			}
			log.WithContext(ctx).Infof("Started update...release set to '%v' ", config.Version)
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
		updateOptions = fmt.Sprintf("--user-pw %s", config.UserPw)
	}

	if len(config.Version) > 0 {
		updateOptions = fmt.Sprintf("%s --release=%s", updateOptions, config.Version)
	}

	updateSuperNodeCmd := fmt.Sprintf("yes Y | %s update %s", constants.RemotePastelupPath, updateOptions)
	if err := executeRemoteCommandsWithInventory(ctx, config, []string{updateSuperNodeCmd}, false); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to update %s on remote host", tool)
	}
	log.WithContext(ctx).Infof("Remote %s updated", tool)

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
	if backUpWorkDir && !config.NoBackup {
		err = archiveWorkDir(ctx, config)
		if err != nil {
			return err
		}
	}
	if backUpDDDir && !config.NoBackup {
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
	// if we have more than ARCHIVE_RETENTION amount of archives, delete old ones to avoid build up
	matchingArchives := []fs.FileInfo{}
	files, err := ioutil.ReadDir(archiveBaseDir)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.IsDir() && strings.HasPrefix(f.Name(), archivePrefix+"_archive_") {
			matchingArchives = append(matchingArchives, f)
		}
	}
	if len(matchingArchives) > ARCHIVE_RETENTION {
		sort.Slice(matchingArchives, func(i, j int) bool {
			return matchingArchives[i].ModTime().Before(matchingArchives[j].ModTime())
		})
		archivesToDelete := len(matchingArchives) - ARCHIVE_RETENTION
		i := 0
		for i < archivesToDelete {
			fp := filepath.Join(archiveBaseDir, matchingArchives[i].Name())
			log.WithContext(ctx).Info(fmt.Sprintf("Deleting old arvhive %v to avoid build up: created at %v", fp, matchingArchives[i].ModTime().Format(time.RFC3339)))
			err = utils.ClearDir(ctx, fp, []string{})
			if err != nil {
				return err
			}
			err = os.Remove(fp)
			if err != nil {
				return err
			}
			i++
		}
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
	}
	log.WithContext(ctx).Info(fmt.Sprintf("%v directory archived", dirToArchive))
	return nil
}
