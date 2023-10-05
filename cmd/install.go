package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/errors"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pastelnetwork/pastelup/configs"
	"github.com/pastelnetwork/pastelup/constants"
	"github.com/pastelnetwork/pastelup/utils"
)

type installCommand uint8

const (
	installNode installCommand = iota
	installWalletNode
	installSuperNode
	installRQService
	installDDService
	installDDImgServer
	installWNService
	installSNService
	installHermesService
	remoteInstall
)

var (
	// node flags
	flagIgnoreDependencies bool
)

// var nonNetworkDependentServices = []constants.ToolType{constants.DDImgService, constants.DDService, constants.RQService}
var nonNetworkDependentServices []constants.ToolType

var (
	installCmdName = map[installCommand]string{
		installNode:          "node",
		installWalletNode:    "walletnode",
		installSuperNode:     "supernode",
		installRQService:     "rq-service",
		installDDService:     "dd-service",
		installDDImgServer:   "imgserver",
		installHermesService: "hermes-service",
		remoteInstall:        "remote",
	}
	installCmdMessage = map[installCommand]string{
		installNode:          "Install node",
		installWalletNode:    "Install Walletnode",
		installSuperNode:     "Install Supernode",
		installRQService:     "Install RaptorQ service",
		installDDService:     "Install Dupe Detection service only",
		installDDImgServer:   "Install Dupe Detection Image Server only",
		installHermesService: "Install hermes-service only",
		remoteInstall:        "Install on Remote host",
	}
)
var appToServiceMap = map[constants.ToolType][]constants.ToolType{
	constants.PastelD: {constants.PastelD},
	constants.WalletNode: {
		constants.PastelD,
		constants.RQService,
		//constants.Bridge,
		constants.WalletNode,
	},
	constants.SuperNode: {
		constants.PastelD,
		constants.RQService,
		constants.DDService,
		constants.SuperNode,
		constants.Hermes,
	},
	constants.RQService:    {constants.RQService},
	constants.DDService:    {constants.DDService},
	constants.DDImgService: {constants.DDImgService},
	constants.Hermes:       {constants.Hermes},
	constants.Bridge:       {constants.Bridge},
}

func setupSubCommand(config *configs.Config,
	installCommand installCommand,
	remote bool,
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
	}

	networkFlags := []*cli.Flag{
		cli.NewFlag("network", &config.Network).SetAliases("n").
			SetUsage(red("Required, network type, can be - \"mainnet\" or \"testnet\"")),
	}

	pastelFlags := []*cli.Flag{
		cli.NewFlag("peers", &config.Peers).SetAliases("p").
			SetUsage(green("Optional, List of peers to add into pastel.conf file, must be in the format - \"ip\" or \"ip:port\"")),
		cli.NewFlag("legacy", &config.Legacy).
			SetUsage(green("Optional, Install legacy pastel parameters - including sprout proving and verifying keys")).SetValue(false),
	}

	userFlags := []*cli.Flag{
		cli.NewFlag("user-pw", &config.UserPw).
			SetUsage(green("Optional, password of current sudo user - so no sudo password request is prompted")),
	}

	var dirsFlags []*cli.Flag

	if !remote {
		dirsFlags = []*cli.Flag{
			cli.NewFlag("dir", &config.PastelExecDir).SetAliases("d").
				SetUsage(green("Optional, Location where to create pastel node directory")).SetValue(config.Configurer.DefaultPastelExecutableDir()),
			cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
				SetUsage(green("Optional, Location where to create working directory")).SetValue(config.Configurer.DefaultWorkingDir()),
		}
	} else {
		dirsFlags = []*cli.Flag{
			cli.NewFlag("dir", &config.PastelExecDir).SetAliases("d").
				SetUsage(green("Optional, Location where to create pastel node directory on the remote computer (default: $HOME/pastel)")),
			cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
				SetUsage(green("Optional, Location where to create working directory on the remote computer (default: $HOME/.pastel)")),
		}
	}

	remoteFlags := []*cli.Flag{
		cli.NewFlag("ssh-ip", &config.RemoteIP).
			SetUsage(red("Required (if inventory not used), SSH address of the remote host")),
		cli.NewFlag("ssh-port", &config.RemotePort).
			SetUsage(yellow("Optional, SSH port of the remote host, default is 22")).SetValue(22),
		cli.NewFlag("ssh-user", &config.RemoteUser).
			SetUsage(yellow("Optional, SSH user")),
		cli.NewFlag("ssh-user-pw", &config.UserPw).
			SetUsage(yellow("Optional, password of remote user - so no sudo password request is prompted")),
		cli.NewFlag("ssh-key", &config.RemoteSSHKey).
			SetUsage(yellow("Optional, Path to SSH private key")),
		cli.NewFlag("inventory", &config.InventoryFile).
			SetUsage(yellow("Required (if ssh-ip not used), Path to the file with configuration of the remote hosts")),
	}

	ddServiceFlags := []*cli.Flag{
		cli.NewFlag("no-cache", &config.NoCache).
			SetUsage(yellow("Optional, runs the installation of python dependencies with caching turned off")),
	}

	var commandName, commandMessage string
	if !remote {
		commandName = installCmdName[installCommand]
		commandMessage = installCmdMessage[installCommand]
	} else {
		commandName = installCmdName[remoteInstall]
		commandMessage = installCmdMessage[remoteInstall]
	}

	commandFlags := append(networkFlags, dirsFlags[:]...)
	commandFlags = append(commandFlags, commonFlags[:]...)

	if installCommand == installNode ||
		installCommand == installWalletNode ||
		installCommand == installSuperNode {
		commandFlags = append(commandFlags, pastelFlags[:]...)
	}
	if remote {
		commandFlags = append(commandFlags, remoteFlags[:]...)
	} else if installCommand == installSuperNode {
		commandFlags = append(commandFlags, userFlags...)
	}

	if installCommand == installDDService || installCommand == installSuperNode {
		commandFlags = append(commandFlags, ddServiceFlags...)
	}

	subCommand := cli.NewCommand(commandName)
	subCommand.SetUsage(cyan(commandMessage))
	subCommand.AddFlags(commandFlags...)
	addLogFlags(subCommand, config)

	if f != nil {
		subCommand.SetActionFunc(func(ctx context.Context, _ []string) error {
			ctx, err := configureLogging(ctx, commandMessage, config)
			if err != nil {
				//Logger doesn't exist
				return fmt.Errorf("failed to configure logging option - %v", err)
			}

			sys.RegisterInterruptHandler(func() {
				log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
				os.Exit(0)
			})

			log.WithContext(ctx).Infof("Install started for network mode '%v'...", config.Network)
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

func setupInstallCommand(config *configs.Config) *cli.Command {
	config.OpMode = "install"

	installNodeSubCommand := setupSubCommand(config, installNode, false, runInstallNodeSubCommand)
	installWalletNodeSubCommand := setupSubCommand(config, installWalletNode, false, runInstallWalletNodeSubCommand)
	installSuperNodeSubCommand := setupSubCommand(config, installSuperNode, false, runInstallSuperNodeSubCommand)
	installRQSubCommand := setupSubCommand(config, installRQService, false, runInstallRaptorQSubCommand)
	installDDSubCommand := setupSubCommand(config, installDDService, false, runInstallDupeDetectionSubCommand)
	installDDImgServerSubCommand := setupSubCommand(config, installDDImgServer, false, runInstallDupeDetectionImgServerSubCommand)
	installHermesServiceSubCommand := setupSubCommand(config, installHermesService, false, runInstallHermesServiceSubCommand)

	installNodeSubCommand.AddSubcommands(setupSubCommand(config, installNode, true, runRemoteInstallNode))
	installWalletNodeSubCommand.AddSubcommands(setupSubCommand(config, installWalletNode, true, runRemoteInstallWalletNode))
	installSuperNodeSubCommand.AddSubcommands(setupSubCommand(config, installSuperNode, true, runRemoteInstallSuperNode))
	installRQSubCommand.AddSubcommands(setupSubCommand(config, installRQService, true, runRemoteInstallRQService))
	installDDSubCommand.AddSubcommands(setupSubCommand(config, installDDService, true, runRemoteInstallDDService))
	installDDImgServerSubCommand.AddSubcommands(setupSubCommand(config, installDDImgServer, true, runRemoteInstallImgServer))
	installHermesServiceSubCommand.AddSubcommands(setupSubCommand(config, installHermesService, true, runRemoteInstallHermesService))

	installCommand := cli.NewCommand("install")
	installCommand.SetUsage(blue("Performs installation and initialization of the system for both WalletNode and SuperNodes"))
	installCommand.AddSubcommands(installWalletNodeSubCommand)
	installCommand.AddSubcommands(installSuperNodeSubCommand)
	installCommand.AddSubcommands(installNodeSubCommand)
	installCommand.AddSubcommands(installRQSubCommand)
	installCommand.AddSubcommands(installDDSubCommand)
	installCommand.AddSubcommands(installHermesServiceSubCommand)
	installCommand.AddSubcommands(installDDImgServerSubCommand)
	return installCommand
}

func runInstallNodeSubCommand(ctx context.Context, config *configs.Config) (err error) {
	return runServicesInstall(ctx, config, constants.PastelD, true)
}

func runInstallWalletNodeSubCommand(ctx context.Context, config *configs.Config) (err error) {
	return runServicesInstall(ctx, config, constants.WalletNode, true)
}

func runInstallSuperNodeSubCommand(ctx context.Context, config *configs.Config) (err error) {
	if utils.GetOS() != constants.Linux {
		log.WithContext(ctx).Error("Supernode can only be installed on Linux")
		return fmt.Errorf("supernode can only be installed on Linux. You are on: %s", string(utils.GetOS()))
	}
	ReserveSNPorts(ctx, config)
	return runServicesInstall(ctx, config, constants.SuperNode, true)
}

func runInstallRaptorQSubCommand(ctx context.Context, config *configs.Config) error {
	return runServicesInstall(ctx, config, constants.RQService, false)
}

func runInstallDupeDetectionSubCommand(ctx context.Context, config *configs.Config) error {
	if utils.GetOS() != constants.Linux {
		log.WithContext(ctx).Error("Dupe Detection service can only be installed on Linux")
		return fmt.Errorf("dupe Detection service can only be installed on Linux. You are on: %s", string(utils.GetOS()))
	}
	return runServicesInstall(ctx, config, constants.DDService, false)
}

func runInstallHermesServiceSubCommand(ctx context.Context, config *configs.Config) error {
	if utils.GetOS() != constants.Linux {
		log.WithContext(ctx).Error("Hermes service can only be installed on Linux")
		return fmt.Errorf("hermes service can only be installed on Linux. You are on: %s", string(utils.GetOS()))
	}
	return runServicesInstall(ctx, config, constants.Hermes, false)
}

func runInstallDupeDetectionImgServerSubCommand(ctx context.Context, config *configs.Config) error {
	if utils.GetOS() != constants.Linux {
		log.WithContext(ctx).Error("dupe detection image service can only be installed on Linux")
		return fmt.Errorf("dupe detection service image can only be installed on Linux. You are on: %s", string(utils.GetOS()))
	}
	config.ServiceTool = "dd-img-service"
	return installSystemService(ctx, config)
}

func runRemoteInstallNode(ctx context.Context, config *configs.Config) error {
	return runRemoteInstall(ctx, config, "node")
}
func runRemoteInstallWalletNode(ctx context.Context, config *configs.Config) error {
	return runRemoteInstall(ctx, config, "walletnode")
}
func runRemoteInstallSuperNode(ctx context.Context, config *configs.Config) error {
	return runRemoteInstall(ctx, config, "supernode")
}
func runRemoteInstallRQService(ctx context.Context, config *configs.Config) error {
	return runRemoteInstall(ctx, config, "rq-service")
}
func runRemoteInstallDDService(ctx context.Context, config *configs.Config) error {
	return runRemoteInstall(ctx, config, "dd-service")
}
func runRemoteInstallImgServer(ctx context.Context, config *configs.Config) error {
	return runRemoteInstall(ctx, config, "imgserver")
}
func runRemoteInstallHermesService(ctx context.Context, config *configs.Config) error {
	return runRemoteInstall(ctx, config, "hermes-service")
}

func runRemoteInstall(ctx context.Context, config *configs.Config, tool string) (err error) {
	if config.Network != "testnet" && config.Network != "mainnet" {
		log.WithContext(ctx).Fatal("--network or -n parameter is required")
		return fmt.Errorf("--network or -n parameter is required")
	}
	if len(config.RemoteIP) == 0 {
		log.WithContext(ctx).Fatal("remote IP is required")
		return fmt.Errorf("remote IP is required")
	}

	log.WithContext(ctx).Infof("Installing remote %s", tool)

	remoteOptions := tool
	if len(config.PastelExecDir) > 0 {
		remoteOptions = fmt.Sprintf("%s --dir=%s", remoteOptions, config.PastelExecDir)
	}

	if len(config.WorkingDir) > 0 {
		remoteOptions = fmt.Sprintf("%s --work-dir=%s", remoteOptions, config.WorkingDir)
	}

	if config.Force {
		remoteOptions = fmt.Sprintf("%s --force", remoteOptions)
	}

	if len(config.Version) > 0 {
		remoteOptions = fmt.Sprintf("%s --version=%s", remoteOptions, config.Version)
	}

	if len(config.Peers) > 0 {
		remoteOptions = fmt.Sprintf("%s --peers=%s", remoteOptions, config.Peers)
	}

	if config.Network == constants.NetworkTestnet {
		remoteOptions = fmt.Sprintf("%s -n=testnet", remoteOptions)
	} else if config.Network == constants.NetworkRegTest {
		remoteOptions = fmt.Sprintf("%s -n=regtest", remoteOptions)
	}

	if len(config.UserPw) > 0 {
		remoteOptions = fmt.Sprintf("%s --user-pw=%s", remoteOptions, config.UserPw)
	}

	installSuperNodeCmd := fmt.Sprintf("yes Y | %s install %s", constants.RemotePastelupPath, remoteOptions)
	if _, err := executeRemoteCommandsWithInventory(ctx, config, []string{installSuperNodeCmd}, false, false); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to install remote %s", tool)
	}
	log.WithContext(ctx).Infof("Finished remote installation of %s", tool)

	return nil
}

func runServicesInstall(ctx context.Context, config *configs.Config, installCommand constants.ToolType, withDependencies bool) error {

	if config.OpMode == "install" && !utils.ContainsToolType(nonNetworkDependentServices, installCommand) {
		if !utils.IsValidNetworkOpt(config.Network) {
			return fmt.Errorf("invalid --network provided. valid opts: %s", strings.Join(constants.NetworkModes, ","))
		}
		log.WithContext(ctx).Infof("initiating in %s mode", config.Network)
	}

	getDefaultRPCParameters(config)

	if installCommand == constants.PastelD ||
		(installCommand == constants.WalletNode && withDependencies) ||
		(installCommand == constants.SuperNode && withDependencies) {

		// need to stop pasteld else we'll get a text file busy error
		if CheckProcessRunning(constants.PastelD) {
			_, err := GetPastelInfo(ctx, config) // this needed to check if pasteld is running in the same mode
			if err == nil {
				log.WithContext(ctx).Infof("pasteld is already running")
				if yes, _ := AskUserToContinue(ctx,
					"Do you want to stop it and continue? Y/N"); !yes {
					log.WithContext(ctx).Warn("Exiting...")
					return fmt.Errorf("user terminated installation")
				}

				sm, err := NewServiceManager(utils.GetOS(), config.Configurer.DefaultHomeDir())
				if err == nil {
					_ = sm.StopService(ctx, config, constants.PastelD)
				}
				if CheckProcessRunning(constants.PastelD) {
					if err = ParsePastelConf(ctx, config); err != nil {
						return err
					}
					err = stopPatelCLI(ctx, config)
					if err != nil {
						log.WithContext(ctx).Warnf("Encountered error trying to stop pasteld %v, will try to kill it", err)
						_ = KillProcess(ctx, constants.PastelD)
					}
				}
				log.WithContext(ctx).Info("pasteld stopped or was not running")
			}
		}
	}

	// create, if needed, installation directory, example ~/pastel
	if err := checkInstallDir(ctx, config, config.PastelExecDir, config.OpMode); err != nil {
		//error was logged inside checkInstallDir
		return err
	}

	if !flagIgnoreDependencies {
		if err := checkInstalledPackages(ctx, config, installCommand, withDependencies); err != nil {
			log.WithContext(ctx).WithError(err).Error("Missing packages...")
			return err
		}
	}

	// install pasteld and pastel-cli; setup working dir (~/.pastel) and pastel.conf
	if installCommand == constants.PastelD ||
		(installCommand == constants.WalletNode && withDependencies) ||
		(installCommand == constants.SuperNode && withDependencies) {
		if err := installPastelCore(ctx, config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to install Pastel Node")
			return err
		}
	}
	// install rqservice and its config
	if installCommand == constants.RQService ||
		(installCommand == constants.WalletNode && withDependencies) ||
		(installCommand == constants.SuperNode && withDependencies) {
		if err := installRaptorQService(ctx, config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to install RaptorQ service")
			return err
		}
	}
	// install WalletNode and its config
	if installCommand == constants.WalletNode {
		if err := installWalletNodeService(ctx, config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to install WalletNode service")
			return err
		}
	}

	// install SuperNode, dd-service and their configs; open ports
	if installCommand == constants.SuperNode {
		if err := installSuperNodeService(ctx, config, config.OpMode == "install" /*only open ports when full system install*/); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to install WalletNode service")
			return err
		}
	}

	if installCommand == constants.DDService ||
		(installCommand == constants.SuperNode && withDependencies) {
		if err := installDupeDetection(ctx, config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to install dd-service")
			return err
		}
	}

	if installCommand == constants.Hermes {
		if err := installHermes(ctx, config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to install hermes-service")
			return err
		}
	}
	return nil
}

func installPastelUp(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Installing Pastelup tool ...")
	pastelupExecName := constants.PastelUpExecName[utils.GetOS()]
	pastelupName := constants.PastelupName[utils.GetOS()]
	if err := downloadComponents(ctx, config, constants.Pastelup, ""); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to download %s", constants.Pastelup)
		return err
	}
	downloadedExecPath := filepath.Join(config.PastelExecDir, pastelupExecName)
	outputPath := filepath.Join(".", pastelupName)
	if err := os.Rename(downloadedExecPath, outputPath); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to rename %v to %s: %v", downloadedExecPath, outputPath, err)
		return err
	}
	if err := makeExecutable(ctx, ".", pastelupName); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to make %s executable", pastelupName)
		return err
	}
	return nil
}

func installPastelCore(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Installing PastelCore service...")

	pasteldName := constants.PasteldName[utils.GetOS()]
	pastelCliName := constants.PastelCliName[utils.GetOS()]

	if err := downloadComponents(ctx, config, constants.PastelD, ""); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to download %s", constants.PastelD)
		return err
	}
	if err := makeExecutable(ctx, config.PastelExecDir, pasteldName); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to make %s executable", pasteldName)
		return err
	}
	if err := makeExecutable(ctx, config.PastelExecDir, pastelCliName); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to make %s executable", pastelCliName)
		return err
	}
	if err := setupBasePastelWorkingEnvironment(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to install Pastel Node")
		return err
	}
	return nil
}

func installRaptorQService(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Installing rq-service...")

	toolPath := constants.PastelRQServiceExecName[utils.GetOS()]
	rqWorkDirPath := filepath.Join(config.WorkingDir, constants.RQServiceDir)

	toolConfig, err := utils.GetServiceConfig(string(constants.RQService), configs.RQServiceDefaultConfig, &configs.RQServiceConfig{
		HostName: "127.0.0.1",
		Port:     constants.RQServiceDefaultPort,
	})
	if err != nil {
		return errors.Errorf("failed to get rqservice config: %v", err)
	}

	if err = downloadComponents(ctx, config, constants.RQService, ""); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to download %s", toolPath)
		return err
	}
	if err = makeExecutable(ctx, config.PastelExecDir, toolPath); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to make %s executable", toolPath)
		return err
	}

	if err := utils.CreateFolder(ctx, rqWorkDirPath, config.Force); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to create folder %s", rqWorkDirPath)
		return err
	}

	if err = setupComponentConfigFile(ctx, config,
		string(constants.RQService),
		config.Configurer.GetRQServiceConfFile(config.WorkingDir),
		toolConfig,
	); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to setup %s", toolPath)
		return err
	}
	return nil
}

func installDupeDetection(ctx context.Context, config *configs.Config) (err error) {
	log.WithContext(ctx).Info("Installing dd-service...")
	// Download dd-service
	if err = downloadComponents(ctx, config, constants.DDService, constants.DupeDetectionSubFolder); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to download %s", constants.DDService)
		return err
	}

	if config.OpMode == "update" && config.SkipDDPackagesUpdate {
		log.WithContext(ctx).Info("Skipping dd-service packages update")
	} else {
		pythonCmd := "python3"
		if utils.GetOS() == constants.Windows {
			pythonCmd = "python"
		}
		venv := filepath.Join(config.PastelExecDir, constants.DupeDetectionSubFolder, "venv")
		if err := RunCMDWithInteractive(pythonCmd, "-m", "venv", venv); err != nil {
			return err
		}
		cmd := fmt.Sprintf("source %v/bin/activate && %v -m pip install --upgrade pip", venv, pythonCmd)
		if err := RunCMDWithInteractive("bash", "-c", cmd); err != nil {
			return err
		}

		requirementsFile := filepath.Join(config.PastelExecDir, constants.DupeDetectionSubFolder, constants.PipRequirmentsFileName)
		// b/c the commands get run as forked sub processes, we need to run the venv and install in one command
		cmd = fmt.Sprintf("source %v/bin/activate && pip install --upgrade -r %v", venv, requirementsFile)
		if config.NoCache {
			cmd += " --no-cache-dir"
		}
		if err := RunCMDWithInteractive("bash", "-c", cmd); err != nil {
			return err
		}
		log.WithContext(ctx).Info("Pip install finished")
	}
	appBaseDir := filepath.Join(config.Configurer.DefaultHomeDir(), constants.DupeDetectionServiceDir)
	var pathList []interface{}
	for _, configItem := range constants.DupeDetectionConfigs {
		dupeDetectionDirPath := filepath.Join(appBaseDir, configItem)
		if err = utils.CreateFolder(ctx, dupeDetectionDirPath, config.Force); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to create directory : %s", dupeDetectionDirPath)
			return err
		}
		pathList = append(pathList, dupeDetectionDirPath)
	}

	ddSupportFilesDir := filepath.Join(appBaseDir, constants.DupeDetectionSupportFilePath)
	if config.OpMode == "update" && config.SkipDDSupportingDilesUpdate {
		log.WithContext(ctx).Info("Skipping dd-service supporting files update")
	} else {
		tmpDir := filepath.Join(ddSupportFilesDir, "temp.zip")
		for _, url := range constants.DupeDetectionSupportDownloadURL {
			// Get ddSupportContent and cal checksum
			ddSupportContent := path.Base(url)
			ddSupportPath := filepath.Join(ddSupportFilesDir, ddSupportContent)
			fileInfo, err := os.Stat(ddSupportPath)
			if err == nil {
				log.WithContext(ctx).Infof("Checking checksum of DupeDetection Support: %s", ddSupportContent)
				var checksum string
				if fileInfo.IsDir() {
					checksum, err = utils.CalChecksumOfFolder(ctx, ddSupportPath)
				} else {
					checksum, err = utils.GetChecksum(ctx, ddSupportPath)
				}
				if err != nil {
					log.WithContext(ctx).WithError(err).Errorf("Failed to get checksum: %s", ddSupportPath)
					return err
				}
				log.WithContext(ctx).Infof("Checksum of DupeDetection Support: %s is %s", ddSupportContent, checksum)
				if checksum == constants.DupeDetectionSupportChecksum[ddSupportContent] {
					log.WithContext(ctx).Infof("DupeDetection Support file: %s is already exists and checksum matched, so skipping download.", ddSupportPath)
					continue
				}
			}
			if !strings.Contains(url, ".zip") {
				if err = utils.DownloadFile(ctx, filepath.Join(ddSupportFilesDir, path.Base(url)), url); err != nil {
					log.WithContext(ctx).WithError(err).Errorf("Failed to download file: %s", url)
					return err
				}
				continue
			}
			if err = utils.DownloadFile(ctx, tmpDir, url); err != nil {
				log.WithContext(ctx).WithError(err).Errorf("Failed to download archive file: %s", url)
				return err
			}

			log.WithContext(ctx).Infof("Extracting archive file : %s", tmpDir)
			if err = processArchive(ctx, ddSupportFilesDir, tmpDir); err != nil {
				log.WithContext(ctx).WithError(err).Errorf("Failed to extract archive file : %s", tmpDir)
				return err
			}
		}
	}
	if config.OpMode == "install" {
		ddConfigPath := filepath.Join(ddSupportFilesDir, constants.DupeDetectionConfigFilename)
		ddConfigSamplePath := filepath.Join(config.PastelExecDir, constants.DupeDetectionSubFolder, "config.ini.sample")
		//check if config.ini.sample exists
		if utils.CheckFileExist(ddConfigSamplePath) {
			// if exists copy from sample
			if err = utils.CopyFile(ctx, ddConfigSamplePath, ddSupportFilesDir, constants.DupeDetectionConfigFilename); err != nil {
				log.WithContext(ctx).Errorf("Failed to copy config.ini.sample for dd-service : %s", ddConfigPath)
				return err
			}
		} else {
			// if not create from scratch
			err = utils.CreateFile(ctx, ddConfigPath, config.Force)
			if err != nil {
				log.WithContext(ctx).Errorf("Failed to create config.ini for dd-service : %s", ddConfigPath)
				return err
			}
			if err = utils.WriteFile(ddConfigPath, fmt.Sprintf(configs.DupeDetectionConfig, pathList...)); err != nil {
				return err
			}
		}
		_ = os.Setenv("DUPEDETECTIONCONFIGPATH", ddConfigPath)
	}
	log.WithContext(ctx).Info("Installing DupeDetection finished successfully")
	return nil
}

func installWalletNodeService(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Installing WalletNode service...")
	//installBridge, _ := AskUserToContinue(ctx, "Install Bridge Service? Y/N")
	installBridge := false

	wnPath := constants.WalletNodeExecName[utils.GetOS()]
	bridgePath := constants.BridgeExecName[utils.GetOS()]
	burnAddress := constants.BurnAddressMainnet
	if config.Network == constants.NetworkTestnet {
		burnAddress = constants.BurnAddressTestnet
	} else if config.Network == constants.NetworkRegTest {
		burnAddress = constants.BurnAddressTestnet
	}
	wnTempDirPath := filepath.Join(config.WorkingDir, constants.TempDir)
	rqWorkDirPath := filepath.Join(config.WorkingDir, constants.RQServiceDir)
	wnConfig, err := utils.GetServiceConfig(string(constants.WalletNode), configs.WalletDefaultConfig, &configs.WalletNodeConfig{
		LogLevel:      constants.WalletNodeDefaultLogLevel,
		LogFilePath:   config.Configurer.GetWalletNodeLogFile(config.WorkingDir),
		LogCompress:   constants.LogConfigDefaultCompress,
		LogMaxSizeMB:  constants.LogConfigDefaultMaxSizeMB,
		LogMaxAgeDays: constants.LogConfigDefaultMaxAgeDays,
		LogMaxBackups: constants.LogConfigDefaultMaxBackups,
		WNTempDir:     wnTempDirPath,
		WNWorkDir:     config.WorkingDir,
		RQDir:         rqWorkDirPath,
		BurnAddress:   burnAddress,
		RaptorqPort:   constants.RQServiceDefaultPort,
		BridgePort:    constants.BridgeServiceDefaultPort,
		BridgeOn:      installBridge,
	})
	if err != nil {
		return errors.Errorf("failed to get walletnode config: %v", err)
	}

	if err = downloadComponents(ctx, config, constants.WalletNode, ""); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to download %s", wnPath)
		return err
	}

	if installBridge {
		bridgeConfig, err := utils.GetServiceConfig(string(constants.Bridge),
			configs.BridgeDefaultConfig, &configs.BridgeConfig{
				LogLevel:           constants.WalletNodeDefaultLogLevel,
				LogFilePath:        config.Configurer.GetBridgeLogFile(config.WorkingDir),
				LogCompress:        constants.LogConfigDefaultCompress,
				LogMaxSizeMB:       constants.LogConfigDefaultMaxSizeMB,
				LogMaxAgeDays:      constants.LogConfigDefaultMaxAgeDays,
				LogMaxBackups:      constants.LogConfigDefaultMaxBackups,
				WNTempDir:          wnTempDirPath,
				WNWorkDir:          config.WorkingDir,
				BurnAddress:        burnAddress,
				ConnRefreshTimeout: 300,
				Connections:        10,
				ListenAddress:      "127.0.0.1",
				Port:               constants.BridgeServiceDefaultPort,
			})
		if err != nil {
			return errors.Errorf("failed to get bridge config: %v", err)
		}

		if err = downloadComponents(ctx, config, constants.Bridge, ""); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to download %s", bridgePath)
			return err
		}

		if err = makeExecutable(ctx, config.PastelExecDir, bridgePath); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to make %s executable", bridgePath)
			return err
		}

		if err = setupComponentConfigFile(ctx, config, string(constants.Bridge),
			config.Configurer.GetBridgeConfFile(config.WorkingDir), bridgeConfig); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to setup %s", bridgePath)
			return err
		}
	}

	if err = makeExecutable(ctx, config.PastelExecDir, wnPath); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to make %s executable", wnPath)
		return err
	}

	if err = setupComponentConfigFile(ctx, config, string(constants.WalletNode),
		config.Configurer.GetWalletNodeConfFile(config.WorkingDir), wnConfig); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to setup %s", wnPath)
		return err
	}

	return nil
}

func installHermes(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Installing Hermes service...")

	hermesConfig, err := GetHermesConfigs(config)
	if err != nil {
		return errors.Errorf("failed to get hermes config: %v", err)
	}

	hermesPath := constants.HermesExecName[utils.GetOS()]

	if err = downloadComponents(ctx, config, constants.Hermes, ""); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to download %s", hermesPath)
		return err
	}

	if err = makeExecutable(ctx, config.PastelExecDir, hermesPath); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to make %s executable", hermesPath)
		return err
	}

	if err = setupComponentConfigFile(ctx, config, string(constants.Hermes),
		config.Configurer.GetHermesConfFile(config.WorkingDir), hermesConfig); err != nil {

		log.WithContext(ctx).WithError(err).Errorf("Failed to setup %s", hermesPath)
		return err
	}

	return nil
}

func installSuperNodeService(ctx context.Context, config *configs.Config, tryOpenPorts bool) error {
	log.WithContext(ctx).Info("Installing SuperNode service...")

	snTempDirPath := filepath.Join(config.WorkingDir, constants.TempDir)
	p2pDataPath := filepath.Join(config.WorkingDir, constants.P2PDataDir)
	mdlDataPath := filepath.Join(config.WorkingDir, constants.MDLDataDir)

	snConfig, err := GetSNConfigs(config)
	if err != nil {
		return errors.Errorf("failed to get supernode config: %v", err)
	}

	hermesConfig, err := GetHermesConfigs(config)
	if err != nil {
		return errors.Errorf("failed to get hermes config: %v", err)
	}

	snPath := constants.SuperNodeExecName[utils.GetOS()]
	hermesPath := constants.HermesExecName[utils.GetOS()]

	if err = downloadComponents(ctx, config, constants.SuperNode, ""); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to download %s", snPath)
		return err
	}

	if err = downloadComponents(ctx, config, constants.Hermes, ""); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to download %s", hermesPath)
		return err
	}

	if err = makeExecutable(ctx, config.PastelExecDir, snPath); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to make %s executable", snPath)
		return err
	}

	if err = makeExecutable(ctx, config.PastelExecDir, hermesPath); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to make %s executable", hermesPath)
		return err
	}

	if err = setupComponentConfigFile(ctx, config, string(constants.SuperNode),
		config.Configurer.GetSuperNodeConfFile(config.WorkingDir), snConfig); err != nil {

		log.WithContext(ctx).WithError(err).Errorf("Failed to setup %s", snPath)
		return err
	}

	if err = setupComponentConfigFile(ctx, config, string(constants.Hermes),
		config.Configurer.GetHermesConfFile(config.WorkingDir), hermesConfig); err != nil {

		log.WithContext(ctx).WithError(err).Errorf("Failed to setup %s", hermesPath)
		return err
	}

	if err := utils.CreateFolder(ctx, snTempDirPath, config.Force); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to create folder %s", snTempDirPath)
		return err
	}

	if err := utils.CreateFolder(ctx, p2pDataPath, config.Force); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to create folder %s", p2pDataPath)
		return err
	}

	if err := utils.CreateFolder(ctx, mdlDataPath, config.Force); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to create folder %s", mdlDataPath)
		return err
	}

	if tryOpenPorts {
		// Open ports
		if err = openPorts(ctx, config, GetSNPortList(config)); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to open ports")
			return err
		}

		increaseOpenFilesOnLinux(ctx, config)
	}

	return nil
}

func checkInstallDir(ctx context.Context, config *configs.Config, installPath string, opMode string) error {
	defer log.WithContext(ctx).Infof("Install path is %s", installPath)

	if utils.CheckFileExist(config.PastelExecDir) {
		if config.OpMode == "install" {
			log.WithContext(ctx).Infof("Directory %s already exists...", installPath)
		}
		if !config.Force {
			if yes, _ := AskUserToContinue(ctx, fmt.Sprintf("%s will overwrite content of %s. Do you want continue? Y/N", opMode, installPath)); !yes {
				log.WithContext(ctx).Info("operation canceled by user. Exiting...")
				return fmt.Errorf("user terminated installation")
			}
		}
		config.Force = true
		return nil
	} else if config.OpMode == "update" {
		log.WithContext(ctx).Infof("Previous installation doesn't exist at %s. Noting to update. Exiting...", config.PastelExecDir)
		return fmt.Errorf("nothing to update. Exiting")
	}

	if err := utils.CreateFolder(ctx, installPath, config.Force); err != nil {
		log.WithContext(ctx).WithError(err).Error("Exiting...")
		return fmt.Errorf("failed to create install directory - %s (%v)", installPath, err)
	}

	return nil
}

func checkInstalledPackages(ctx context.Context, config *configs.Config, tool constants.ToolType, withDependencies bool) (err error) {

	var packagesRequiredDirty []string

	var appServices []constants.ToolType
	if withDependencies {
		appServices = appToServiceMap[tool]
	} else {
		appServices = append(appServices, tool)
	}

	for _, srv := range appServices {
		packagesRequiredDirty = append(packagesRequiredDirty, constants.DependenciesPackages[srv][utils.GetOS()]...)
	}
	if len(packagesRequiredDirty) == 0 {
		return nil
	}
	//remove duplicates
	keyGuard := make(map[string]bool)
	var packagesRequired []string
	for _, item := range packagesRequiredDirty {
		if _, value := keyGuard[item]; !value {
			keyGuard[item] = true
			packagesRequired = append(packagesRequired, item)
		}
	}

	if utils.GetOS() != constants.Linux {
		reqPackagesStr := strings.Join(packagesRequired, ",")
		log.WithContext(ctx).WithField("required-packages", reqPackagesStr).
			WithError(errors.New("install/update required packages")).
			Error("automatic install/update for required packages only set up for linux")
	}

	packagesInstalled := utils.GetInstalledPackages(ctx)
	var packagesMissing []string
	var packagesToUpdate []string
	for _, p := range packagesRequired {
		if _, ok := packagesInstalled[p]; !ok {
			packagesMissing = append(packagesMissing, p)
		} else {
			packagesToUpdate = append(packagesToUpdate, p)
		}
	}

	if config.OpMode == "update" && utils.GetOS() == constants.Linux && len(packagesToUpdate) != 0 {
		packagesUpdStr := strings.Join(packagesToUpdate, ",")
		if !config.Force {
			if yes, _ := AskUserToContinue(ctx, "Some system packages ["+packagesUpdStr+"] required for "+string(tool)+" need to be updated. Do you want to update them? Y/N"); !yes {
				log.WithContext(ctx).Warn("Exiting...")
				return fmt.Errorf("user terminated installation")
			}
		}
		if config.SkipSystemUpdate {
			log.WithContext(ctx).Info("Skipping system update")
		} else {
			if err := installOrUpgradePackagesLinux(ctx, config, "upgrade", packagesToUpdate); err != nil {
				log.WithContext(ctx).WithField("packages-update", packagesToUpdate).
					WithError(err).
					Errorf("failed to update required packages - %s", packagesUpdStr)
				return err
			}
		}
	}

	if len(packagesMissing) == 0 {
		return nil
	}

	packagesMissStr := strings.Join(packagesMissing, ",")
	if !config.Force {
		if yes, _ := AskUserToContinue(ctx, "The system misses some packages ["+packagesMissStr+"] required for "+string(tool)+". Do you want to install them? Y/N"); !yes {
			log.WithContext(ctx).Warn("Exiting...")
			return fmt.Errorf("user terminated installation")
		}
	}
	return installOrUpgradePackagesLinux(ctx, config, "install", packagesMissing)
}

func installOrUpgradePackagesLinux(ctx context.Context, config *configs.Config, what string, packages []string) error {
	var out string
	var err error

	log.WithContext(ctx).WithField("packages", strings.Join(packages, ",")).
		Infof("system will now %s packages", what)

	// Update repo
	_, err = RunSudoCMD(config, "apt", "update")
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to update")
		return err
	}

	for _, pkg := range packages {
		log.WithContext(ctx).Infof("%sing package %s", what, pkg)

		if pkg == "google-chrome-stable" {
			if err := addGoogleRepo(ctx, config); err != nil {
				log.WithContext(ctx).WithError(err).Errorf("Failed to update pkg %s", pkg)
				return err
			}
			_, err = RunSudoCMD(config, "apt", "update")
			if err != nil {
				log.WithContext(ctx).WithError(err).Error("Failed to update")
				return err
			}
		}
		out, err = RunSudoCMD(config, "apt", "-y", what, pkg) //"install" or "upgrade"
		if err != nil {
			log.WithContext(ctx).WithFields(log.Fields{"message": out, "package": pkg}).
				WithError(err).Errorf("unable to %s package", what)
			return err
		}
	}
	log.WithContext(ctx).Infof("Packages %sed", what)
	return nil
}

func addGoogleRepo(ctx context.Context, config *configs.Config) error {
	var err error

	log.WithContext(ctx).Info("Adding google ssl key ...")

	_, err = RunCMD("bash", "-c", "wget -q -O - "+constants.GooglePubKeyURL+" > /tmp/google-key.pub")
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Write /tmp/google-key.pub failed")
		return err
	}

	_, err = RunSudoCMD(config, "apt-key", "add", "/tmp/google-key.pub")
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to add google ssl key")
		return err
	}
	log.WithContext(ctx).Info("Added google ssl key")

	// Add google repo: /etc/apt/sources.list.d/google-chrome.list
	log.WithContext(ctx).Info("Adding google ppa repo ...")
	_, err = RunCMD("bash", "-c", "echo '"+constants.GooglePPASourceList+"' | tee /tmp/google-chrome.list")
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to create /tmp/google-chrome.list")
		return err
	}

	_, err = RunSudoCMD(config, "mv", "/tmp/google-chrome.list", constants.UbuntuSourceListPath)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to move /tmp/google-chrome.list to " + constants.UbuntuSourceListPath)
		return err
	}

	log.WithContext(ctx).Info("Added google ppa repo")
	return nil
}

func downloadComponents(ctx context.Context, config *configs.Config, installCommand constants.ToolType, dstFolder string) (err error) {
	if _, err := os.Stat(config.PastelExecDir); os.IsNotExist(err) {
		if err := utils.CreateFolder(ctx, config.PastelExecDir, config.Force); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("create folder: %s", config.PastelExecDir)
			return err
		}
	}

	commandName := filepath.Base(string(installCommand))
	log.WithContext(ctx).Infof("Downloading %s...", commandName)

	network := config.Network
	version := config.Version
	if installCommand == constants.Pastelup { // Pastelup is in the root of 'latest-release' folder
		network = ""
		version = ""
	}

	downloadURL, archiveName, err := config.Configurer.GetDownloadURL(network, version, installCommand)
	if err != nil {
		return errors.Errorf("failed to get download url: %v", err)
	}

	if err = utils.DownloadFile(ctx, filepath.Join(config.PastelExecDir, archiveName), downloadURL.String()); err != nil {
		return errors.Errorf("failed to download executable file %s: %v", downloadURL.String(), err)
	}

	if strings.Contains(archiveName, ".zip") {
		if err = processArchive(ctx, filepath.Join(config.PastelExecDir, dstFolder), filepath.Join(config.PastelExecDir, archiveName)); err != nil {
			//Error was logged in processArchive
			return errors.Errorf("failed to process downloaded file: %v", err)
		}
	}

	log.WithContext(ctx).Infof("%s downloaded successfully", commandName)

	return nil
}

func processArchive(ctx context.Context, dstFolder string, archivePath string) error {
	log.WithContext(ctx).Debugf("Extracting archive files from %s to %s", archivePath, dstFolder)

	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		log.WithContext(ctx).WithError(err).Errorf("Not found archive file - %s", archivePath)
		return err
	}

	if _, err := utils.Unzip(archivePath, dstFolder); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to extract executables from %s", archivePath)
		return err
	}
	log.WithContext(ctx).Debug("Delete archive files")
	if err := utils.DeleteFile(archivePath); err != nil {
		log.WithContext(ctx).Errorf("Failed to delete archive file : %s", archivePath)
		return err
	}

	return nil
}

func makeExecutable(ctx context.Context, dirPath string, fileName string) error {
	if utils.GetOS() == constants.Linux ||
		utils.GetOS() == constants.Mac {
		filePath := filepath.Join(dirPath, fileName)
		if _, err := RunCMD("chmod", "755", filePath); err != nil {
			log.WithContext(ctx).Errorf("Failed to make %s as executable", filePath)
			return err
		}
	}
	return nil
}

func setupComponentConfigFile(ctx context.Context, config *configs.Config,
	toolName string, configFilePath string, toolConfig string) error {

	// Ignore if not in "install" mode
	if config.OpMode != "install" {
		return nil
	}

	log.WithContext(ctx).Infof("Initialize working environment for %s", toolName)
	err := utils.CreateFile(ctx, configFilePath, config.Force)
	if err != nil {
		log.WithContext(ctx).Errorf("Failed to create %s file", configFilePath)
		return err
	}

	if err = utils.WriteFile(configFilePath, toolConfig); err != nil {
		log.WithContext(ctx).Errorf("Failed to write config to %s file", configFilePath)
		return err
	}

	return nil
}

func setupBasePastelWorkingEnvironment(ctx context.Context, config *configs.Config) error {
	// Ignore if not in "install" mode
	if config.OpMode != "install" {
		return nil
	}

	// create working dir
	if err := utils.CreateFolder(ctx, config.WorkingDir, config.Force); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to create folder %s", config.WorkingDir)
		return err
	}

	// create pastel.conf file
	pastelConfigPath := filepath.Join(config.WorkingDir, constants.PastelConfName)
	err := utils.CreateFile(ctx, pastelConfigPath, config.Force)
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to create %s", pastelConfigPath)
		return fmt.Errorf("failed to create %s - %v", pastelConfigPath, err)
	}
	// write to file
	if err = updatePastelConfigFile(ctx, pastelConfigPath, config); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to update %s", pastelConfigPath)
		return fmt.Errorf("failed to update %s - %v", pastelConfigPath, err)
	}

	// create zksnark parameters path
	if err := utils.CreateFolder(ctx, config.Configurer.DefaultZksnarkDir(), config.Force); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to update folder %s", config.Configurer.DefaultZksnarkDir())
		return fmt.Errorf("failed to update folder %s - %v", config.Configurer.DefaultZksnarkDir(), err)
	}

	// download zksnark params
	if err := downloadZksnarkParams(ctx, config.Configurer.DefaultZksnarkDir(), config.Force, config.Legacy); err != nil &&
		!(os.IsExist(err) && !config.Force) {
		log.WithContext(ctx).WithError(err).Errorf("Failed to download Zksnark parameters into folder %s", config.Configurer.DefaultZksnarkDir())
		return fmt.Errorf("failed to download Zksnark parameters into folder %s - %v", config.Configurer.DefaultZksnarkDir(), err)
	}

	return nil
}

func getDefaultRPCParameters(config *configs.Config) {
	if config.RPCPort == 0 || config.RegenRPC {
		portList := GetSNPortList(config)
		config.RPCPort = portList[constants.NodeRPCPort]
	}
	if config.RPCUser == "" || config.RegenRPC {
		config.RPCUser = utils.GenerateRandomString(8)
	}
	if config.RPCPwd == "" || config.RegenRPC {
		config.RPCPwd = utils.GenerateRandomString(15)
	}
}

func updatePastelConfigFile(ctx context.Context, filePath string, config *configs.Config) error {
	cfgBuffer := bytes.Buffer{}

	// Populate pastel.conf line-by-line to file.
	cfgBuffer.WriteString("server=1\n")                                     // creates server line
	cfgBuffer.WriteString("listen=1\n\n")                                   // creates server line
	cfgBuffer.WriteString("rpcuser=" + config.RPCUser + "\n")               // creates rpcuser line
	cfgBuffer.WriteString("rpcpassword=" + config.RPCPwd + "\n")            // creates rpcpassword line
	cfgBuffer.WriteString("rpcport=" + strconv.Itoa(config.RPCPort) + "\n") // creates rpcport line
	cfgBuffer.WriteString("maxmempool=20000\n")
	cfgBuffer.WriteString("rpcworkqueue=512\n")

	if config.Network == constants.NetworkTestnet {
		cfgBuffer.WriteString("testnet=1\n") // creates testnet line
	}

	if config.Network == constants.NetworkRegTest {
		cfgBuffer.WriteString("regtest=1\n") // creates testnet line
	}

	if config.Peers != "" {
		nodes := strings.Split(config.Peers, ",")
		for _, node := range nodes {
			cfgBuffer.WriteString("addnode=" + node + "\n") // creates addnode line
		}
	}

	// Save file changes.
	err := os.WriteFile(filePath, cfgBuffer.Bytes(), 0644)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Error saving file")
		return errors.Errorf("failed to save file changes: %v", err)
	}

	log.WithContext(ctx).Info("File updated successfully")

	return nil
}

func downloadZksnarkParams(ctx context.Context, path string, force bool, legacy bool) error {
	log.WithContext(ctx).Info("Downloading pastel-param files:")

	zkParams := configs.ZksnarkParamsNamesV2
	if legacy {
		zkParams = append(zkParams, configs.ZksnarkParamsNamesV1...)
	}

	for _, zksnarkParamsName := range zkParams {
		checkSum := ""
		zksnarkParamsPath := filepath.Join(path, zksnarkParamsName)
		log.WithContext(ctx).Infof("Downloading: %s", zksnarkParamsPath)
		_, err := os.Stat(zksnarkParamsPath)
		// check if file exists and force is not set
		if err == nil && !force {
			log.WithContext(ctx).WithError(err).Errorf("Pastel param file already exists %s", zksnarkParamsPath)
			return errors.Errorf("pastel-param exists:  %s", zksnarkParamsPath)

		} else if err == nil {

			checkSum, err = utils.GetChecksum(ctx, zksnarkParamsPath)
			if err != nil {
				log.WithContext(ctx).WithError(err).Errorf("Checking pastel param file failed: %s", zksnarkParamsPath)
				return err
			}
		}

		if checkSum != constants.PastelParamsCheckSums[zksnarkParamsName] {
			err := utils.DownloadFile(ctx, zksnarkParamsPath, configs.ZksnarkParamsURL+zksnarkParamsName)
			if err != nil {
				log.WithContext(ctx).WithError(err).Errorf("Failed to download file: %s", configs.ZksnarkParamsURL+zksnarkParamsName)
				return err
			}
		} else {
			log.WithContext(ctx).Infof("Pastel param file %s already exists and checksum matched, so skipping download.", zksnarkParamsName)
		}
	}
	log.WithContext(ctx).Info("Pastel params downloaded.\n")
	return nil
}

func openPorts(ctx context.Context, config *configs.Config, portList []int) (err error) {
	if config.OpMode != "install" {
		return nil
	}

	// only open ports on SuperNode and this is only on Linux!!!
	var out string
	for k := range portList {
		log.WithContext(ctx).Infof("Opening port: %d", portList[k])

		portStr := fmt.Sprintf("%d", portList[k])
		switch utils.GetOS() {
		case constants.Linux:
			out, err = RunSudoCMD(config, "ufw", "allow", portStr)
			/*		case constants.Windows:
						out, err = RunCMD("netsh", "advfirewall", "firewall", "add", "rule", "name=TCP Port "+portStr, "dir=in", "action=allow", "protocol=TCP", "localport="+portStr)
					case constants.Mac:
						out, err = RunCMD("sudo", "ipfw", "allow", "tcp", "from", "any", "to", "any", "dst-port", portStr)
			*/
		}

		if err != nil {
			if utils.GetOS() == constants.Windows {
				log.WithContext(ctx).Error("Please run as administrator to open ports!")
			}
			log.WithContext(ctx).Error(err.Error())
			return err
		}
		log.WithContext(ctx).Info(out)
	}

	return nil
}

func increaseOpenFilesOnLinux(ctx context.Context, config *configs.Config) {
	log.WithContext(ctx).Info("Trying to increase number of open file in the system...")

	systemWideLimit := "fs.file-max = 100000"
	userLimit := "* soft nofile 4096\n* hard nofile 65535"
	defaultLimit := "DefaultLimitNOFILE=65536"

	// Increase system-wide file descriptor limit
	err := writeToFile(config, "/etc/sysctl.conf", systemWideLimit)
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("failed to write to sysctl.conf")
	}

	// Increase per-user limit for file descriptors
	err = writeToFile(config, "/etc/security/limits.conf", userLimit)
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("failed to write to limits.conf")
	}

	// Set default limit for the whole system
	err = writeToFile(config, "/etc/systemd/system.conf", defaultLimit)
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("failed to write to system.conf")
	}
	err = writeToFile(config, "/etc/systemd/user.conf", defaultLimit)
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("failed to write to user.conf")
	}

	// Reload systemd and sysctl
	_, err = RunSudoCMD(config, "systemctl", "daemon-reload")
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("failed to reload systemd")
	}
	_, err = RunSudoCMD(config, "sysctl", "-p")
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("failed to reload sysctl")
	}
}

func writeToFile(config *configs.Config, filename, content string) error {
	// Read the original file
	origData, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	// Write to a temporary file first
	tmpFilename := "/tmp/tempfile"
	tmpFile, err := os.OpenFile(tmpFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	// Write original data to the temporary file
	if _, err := tmpFile.Write(origData); err != nil {
		tmpFile.Close()
		return err
	}

	// Append new content to the temporary file
	if _, err := tmpFile.WriteString(content + "\n"); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	// Move the temporary file to the intended location
	_, err = RunSudoCMD(config, "mv", tmpFilename, filename)
	return err
}
