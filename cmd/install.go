package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/errors"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pastelnetwork/pastelup/configs"
	"github.com/pastelnetwork/pastelup/constants"
	"github.com/pastelnetwork/pastelup/servicemanager"
	"github.com/pastelnetwork/pastelup/utils"
)

type installCommand uint8

const (
	nodeInstall installCommand = iota
	walletNodeInstall
	superNodeInstall
	remoteInstall
	rqServiceInstall
	ddServiceInstall
	ddServiceImgServerInstall
	snServiceInstall
	wnServiceInstall
)

var (
	installCmdName = map[installCommand]string{
		nodeInstall:               "node",
		walletNodeInstall:         "walletnode",
		superNodeInstall:          "supernode",
		rqServiceInstall:          "rq-service",
		ddServiceInstall:          "dd-service",
		ddServiceImgServerInstall: "imgserver",
		remoteInstall:             "remote",
	}
	installCmdMessage = map[installCommand]string{
		nodeInstall:               "Install node",
		walletNodeInstall:         "Install Walletnode",
		superNodeInstall:          "Install Supernode",
		rqServiceInstall:          "Install RaptorQ service",
		ddServiceInstall:          "Install Dupe Detection service only",
		ddServiceImgServerInstall: "Install Dupe Detection Image Server only",
		remoteInstall:             "Install on Remote host",
	}
)
var appToServiceMap = map[constants.ToolType][]constants.ToolType{
	constants.PastelD: {constants.PastelD},
	constants.WalletNode: {
		constants.PastelD,
		constants.RQService,
		constants.WalletNode,
	},
	constants.SuperNode: {
		constants.PastelD,
		constants.RQService,
		constants.DDService,
		constants.SuperNode,
	},
	constants.RQService:    {constants.RQService},
	constants.DDService:    {constants.DDService},
	constants.DDImgService: {constants.DDImgService},
}

func setupSubCommand(config *configs.Config,
	installCommand installCommand,
	remote bool,
	f func(context.Context, *configs.Config) error,
) *cli.Command {
	commonFlags := []*cli.Flag{
		cli.NewFlag("force", &config.Force).SetAliases("f").
			SetUsage(green("Optional, Force to overwrite config files and re-download ZKSnark parameters")),
		cli.NewFlag("release", &config.Version).SetAliases("r").
			SetUsage(green("Optional, Pastel version to install")).SetValue("beta"),
		cli.NewFlag("enable-service", &config.EnableService).
			SetUsage(green("Optional, start all apps automatically as system service (i.e. for linux OS, systemd)")),
	}

	pastelFlags := []*cli.Flag{
		cli.NewFlag("network", &config.Network).SetAliases("n").
			SetUsage(green("Optional, network type, can be - \"mainnet\" or \"testnet\"")).SetValue("mainnet"),
		cli.NewFlag("peers", &config.Peers).SetAliases("p").
			SetUsage(green("Optional, List of peers to add into pastel.conf file, must be in the format - \"ip\" or \"ip:port\"")),
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
			SetUsage(red("Required, SSH address of the remote host")).SetRequired(),
		cli.NewFlag("ssh-port", &config.RemotePort).
			SetUsage(yellow("Optional, SSH port of the remote host, default is 22")).SetValue(22),
		cli.NewFlag("ssh-user", &config.RemoteUser).
			SetUsage(yellow("Optional, SSH user")),
		cli.NewFlag("ssh-user-pw", &config.UserPw).
			SetUsage(red("Required, password of remote user - so no sudo password request is prompted")).SetRequired(),
		cli.NewFlag("ssh-key", &config.RemoteSSHKey).
			SetUsage(yellow("Optional, Path to SSH private key")),
	}

	var commandName, commandMessage string
	if !remote {
		commandName = installCmdName[installCommand]
		commandMessage = installCmdMessage[installCommand]
	} else {
		commandName = installCmdName[remoteInstall]
		commandMessage = installCmdMessage[remoteInstall]
	}

	commandFlags := append(dirsFlags, commonFlags[:]...)
	if installCommand == nodeInstall ||
		installCommand == walletNodeInstall ||
		installCommand == superNodeInstall {
		commandFlags = append(commandFlags, pastelFlags[:]...)
	}
	if remote {
		commandFlags = append(commandFlags, remoteFlags[:]...)
	} else if installCommand == superNodeInstall {
		commandFlags = append(commandFlags, userFlags...)
	}

	subCommand := cli.NewCommand(commandName)
	subCommand.SetUsage(cyan(commandMessage))
	subCommand.AddFlags(commandFlags...)
	if f != nil {
		subCommand.SetActionFunc(func(ctx context.Context, _ []string) error {
			ctx, err := configureLogging(ctx, commandMessage, config)
			if err != nil {
				//Logger doesn't exist
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

func setupInstallCommand() *cli.Command {
	config := configs.InitConfig()
	config.OpMode = "install"

	installNodeSubCommand := setupSubCommand(config, nodeInstall, false, runInstallNodeSubCommand)
	installWalletNodeSubCommand := setupSubCommand(config, walletNodeInstall, false, runInstallWalletNodeSubCommand)
	installSuperNodeSubCommand := setupSubCommand(config, superNodeInstall, false, runInstallSuperNodeSubCommand)
	installRQSubCommand := setupSubCommand(config, rqServiceInstall, false, runInstallRaptorQSubCommand)
	installDDSubCommand := setupSubCommand(config, ddServiceInstall, false, runInstallDupeDetectionSubCommand)
	installDDImgServerSubCommand := setupSubCommand(config, ddServiceImgServerInstall, false, runInstallDupeDetectionImgServerSubCommand)

	installNodeSubCommand.AddSubcommands(setupSubCommand(config, nodeInstall, true, runRemoteInstallSubCommand))
	installWalletNodeSubCommand.AddSubcommands(setupSubCommand(config, walletNodeInstall, true, runRemoteInstallSubCommand))
	installSuperNodeSubCommand.AddSubcommands(setupSubCommand(config, superNodeInstall, true, runRemoteInstallSubCommand))
	installRQSubCommand.AddSubcommands(setupSubCommand(config, rqServiceInstall, true, runRemoteInstallSubCommand))
	installDDSubCommand.AddSubcommands(setupSubCommand(config, ddServiceInstall, true, runRemoteInstallSubCommand))
	installDDImgServerSubCommand.AddSubcommands(setupSubCommand(config, ddServiceImgServerInstall, true, runRemoteInstallSubCommand))

	installCommand := cli.NewCommand("install")
	installCommand.SetUsage(blue("Performs installation and initialization of the system for both WalletNode and SuperNodes"))
	installCommand.AddSubcommands(installWalletNodeSubCommand)
	installCommand.AddSubcommands(installSuperNodeSubCommand)
	installCommand.AddSubcommands(installNodeSubCommand)
	installCommand.AddSubcommands(installRQSubCommand)
	installCommand.AddSubcommands(installDDSubCommand)
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
		return fmt.Errorf("Supernode can only be installed on Linux. You are on: %s", string(utils.GetOS()))
	}
	return runServicesInstall(ctx, config, constants.SuperNode, true)
}

func runInstallRaptorQSubCommand(ctx context.Context, config *configs.Config) error {
	return runServicesInstall(ctx, config, constants.RQService, false)
}

func runInstallDupeDetectionSubCommand(ctx context.Context, config *configs.Config) error {
	if utils.GetOS() != constants.Linux {
		log.WithContext(ctx).Error("Dupe Detection service can only be installed on Linux")
		return fmt.Errorf("Dupe Detection service can only be installed on Linux. You are on: %s", string(utils.GetOS()))
	}
	return runServicesInstall(ctx, config, constants.DDService, false)
}

func runInstallDupeDetectionImgServerSubCommand(ctx context.Context, config *configs.Config) error {
	if !config.EnableService {
		return nil
	}
	return installServices(ctx, appToServiceMap[constants.DDImgService], config)
}

func runRemoteInstallSubCommand(ctx context.Context, config *configs.Config) (err error) {
	// Connect to remote
	client, err := prepareRemoteSession(ctx, config)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to prepare remote session")
		return
	}
	defer client.Close()

	// Validate running services
	checkIfRunningCommand := "ps afx | grep -E 'pasteld|rq-service|dd-service|supernode' | grep -v grep"
	if out, _ := client.Cmd(checkIfRunningCommand).Output(); len(out) != 0 {
		log.WithContext(ctx).Info("Supernode is running on remote host")

		if yes, _ := AskUserToContinue(ctx,
			"Do you want to stop it and continue? Y/N"); !yes {
			log.WithContext(ctx).Warn("Exiting...")
			return fmt.Errorf("user terminated installation")
		}

		log.WithContext(ctx).Info("Stopping supernode services...")
		stopSuperNodeCmd := fmt.Sprintf("%s stop supernode ", constants.RemotePastelupPath)
		err = client.ShellCmd(ctx, stopSuperNodeCmd)
		if err != nil {
			if config.Force {
				log.WithContext(ctx).WithError(err).Warnf("failed to stop supernode: %v", err)
			} else {
				log.WithContext(ctx).WithError(err).Errorf("failed to stop supernode: %v", err)
				return err
			}
		} else {
			log.WithContext(ctx).Info("Supernode stopped")
		}
	}

	log.WithContext(ctx).Info("Installing Supernode ...")

	remoteOptions := ""
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
		remoteOptions = fmt.Sprintf("%s --release=%s", remoteOptions, config.Version)
	}

	if len(config.Peers) > 0 {
		remoteOptions = fmt.Sprintf("%s --peers=%s", remoteOptions, config.Peers)
	}

	if config.Network == constants.NetworkTestnet {
		remoteOptions = fmt.Sprintf("%s -n=testnet", remoteOptions)
	} else if config.Network == constants.NetworkRegTest {
		remoteOptions = fmt.Sprintf("%s -n=regtest", remoteOptions)
	}

	if config.EnableService {
		remoteOptions = fmt.Sprintf("%s --enable-service", remoteOptions)
	}

	if len(config.UserPw) > 0 {
		remoteOptions = fmt.Sprintf("%s --user-pw=%s", remoteOptions, config.UserPw)
	}

	installSuperNodeCmd := fmt.Sprintf("yes Y | %s install supernode%s", constants.RemotePastelupPath, remoteOptions)
	err = client.ShellCmd(ctx, installSuperNodeCmd)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to Installing Supernode")
		return err
	}

	log.WithContext(ctx).Info("Finished Installing Supernode Successfully")
	return nil
}

func runServicesInstall(ctx context.Context, config *configs.Config, installCommand constants.ToolType, withDependencies bool) error {
	if config.OpMode == "install" {
		if !utils.IsValidNetworkOpt(config.Network) {
			return fmt.Errorf("invalid --network provided. valid opts: %s", strings.Join(constants.NetworkModes, ","))
		}
		log.WithContext(ctx).Infof("initiating in %s mode", config.Network)
	}

	if installCommand == constants.PastelD ||
		(installCommand == constants.WalletNode && withDependencies) ||
		(installCommand == constants.SuperNode && withDependencies) {

		// need to stop pasteld else we'll get a text file busy error
		possibleCliPath := filepath.Join(config.PastelExecDir, constants.PastelCliName[utils.GetOS()])
		if utils.CheckFileExist(possibleCliPath) {
			log.WithContext(ctx).Info("Trying to stop pasteld...")
			sm, _ := servicemanager.New(utils.GetOS(), config.Configurer.DefaultHomeDir())
			sm.StopService(ctx, constants.PastelD)
			RunPastelCLI(ctx, config, "stop")
			time.Sleep(10 * time.Second) // buffer period to stop
			log.WithContext(ctx).Info("pasteld stopped or was not running")
		}
	}

	// create, if needed, installation directory, example ~/pastel
	if err := checkInstallDir(ctx, config, config.PastelExecDir, config.OpMode); err != nil {
		//error was logged inside checkInstallDir
		return err
	}

	if err := checkInstalledPackages(ctx, config, installCommand, withDependencies); err != nil {
		log.WithContext(ctx).WithError(err).Error("Missing packages...")
		return err
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
		if err := installRQService(ctx, config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to install RaptorQ service")
			return err
		}
	}
	// install WalletNode and its config
	if installCommand == constants.WalletNode {
		if err := installWNService(ctx, config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to install WalletNode service")
			return err
		}
	}

	// install SuperNode, dd-service and their configs; open ports
	if installCommand == constants.SuperNode {
		if err := installSNService(ctx, config, withDependencies /*only open ports when full system install*/); err != nil {
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

	// do service installation if enabled
	if config.EnableService {
		var serviceApps []constants.ToolType
		if withDependencies {
			serviceApps = appToServiceMap[installCommand]
		} else {
			serviceApps = append(serviceApps, installCommand)
		}
		err := installServices(ctx, serviceApps, config)
		if err != nil {
			return err
		}
	}
	return nil
}

func installPastelCore(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Installing PastelCore service...")

	pasteldName := constants.PasteldName[utils.GetOS()]
	pastelCliName := constants.PastelCliName[utils.GetOS()]

	if err := downloadComponents(ctx, config, constants.PastelD, config.Version, ""); err != nil {
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
	if err := setupBasePasteWorkingEnvironment(ctx, config); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to install Pastel Node")
		return err
	}
	return nil
}

func installRQService(ctx context.Context, config *configs.Config) error {
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

	if err = downloadComponents(ctx, config, constants.RQService, config.Version, ""); err != nil {
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

	if err = setupComponentWorkingEnvironment(ctx, config,
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
	if err = downloadComponents(ctx, config, constants.DDService, config.Version, constants.DupeDetectionSubFolder); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to download %s", constants.DDService)
		return err
	}

	// Install pip pkg from requirements.in
	subCmd := []string{"-m", "pip", "install", "-r"}
	subCmd = append(subCmd, filepath.Join(config.PastelExecDir, constants.DupeDetectionSubFolder, constants.PipRequirmentsFileName))

	log.WithContext(ctx).Info("Installing Pip: ", subCmd)
	if utils.GetOS() == constants.Windows {
		if err := RunCMDWithInteractive("python", subCmd...); err != nil {
			return err
		}
	} else {
		if err := RunCMDWithInteractive("python3", subCmd...); err != nil {
			return err
		}
	}

	log.WithContext(ctx).Info("Pip install finished")

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

	targetDir := filepath.Join(appBaseDir, constants.DupeDetectionSupportFilePath)
	tmpDir := filepath.Join(targetDir, "temp.zip")
	for _, url := range constants.DupeDetectionSupportDownloadURL {
		// Get ddSupportContent and cal checksum
		ddSupportContent := constants.DupeDetectionSupportContents[path.Base(url)]
		if len(ddSupportContent) > 0 {
			ddSupportPath := filepath.Join(targetDir, ddSupportContent)
			fileInfo, err := os.Stat(ddSupportPath)
			if err == nil {
				var checksum string
				log.WithContext(ctx).Infof("Checking checksum of DupeDetection Support: %s", ddSupportContent)

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

				// Compare checksum
				if checksum == constants.DupeDetectionSupportChecksum[ddSupportContent] {
					log.WithContext(ctx).Infof("DupeDetection Support file: %s is already exists and checkum matched, so skipping download.", ddSupportPath)
					continue
				}
			}
		}

		if !strings.Contains(url, ".zip") {
			if err = utils.DownloadFile(ctx, filepath.Join(targetDir, path.Base(url)), url); err != nil {
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
		if err = processArchive(ctx, targetDir, tmpDir); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to extract archive file : %s", tmpDir)
			return err
		}
	}

	if config.OpMode == "install" {
		ddConfigPath := filepath.Join(targetDir, constants.DupeDetectionConfigFilename)
		err = utils.CreateFile(ctx, ddConfigPath, config.Force)
		if err != nil {
			log.WithContext(ctx).Errorf("Failed to create config.ini for dd-service : %s", ddConfigPath)
			return err
		}
		if err = utils.WriteFile(ddConfigPath, fmt.Sprintf(configs.DupeDetectionConfig, pathList...)); err != nil {
			return err
		}
		os.Setenv("DUPEDETECTIONCONFIGPATH", ddConfigPath)
	}
	log.WithContext(ctx).Info("Installing DupeDetection finished successfully")
	return nil
}

func installWNService(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Installing WalletNode service...")

	toolPath := constants.WalletNodeExecName[utils.GetOS()]
	burnAddress := constants.BurnAddressMainnet
	if config.Network == constants.NetworkTestnet {
		burnAddress = constants.BurnAddressTestnet
	} else if config.Network == constants.NetworkRegTest {
		burnAddress = constants.BurnAddressTestnet
	}
	wnTempDirPath := filepath.Join(config.WorkingDir, constants.TempDir)
	rqWorkDirPath := filepath.Join(config.WorkingDir, constants.RQServiceDir)
	toolConfig, err := utils.GetServiceConfig(string(constants.WalletNode), configs.WalletDefaultConfig, &configs.WalletNodeConfig{
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
	})
	if err != nil {
		return errors.Errorf("failed to get walletnode config: %v", err)
	}
	if err = downloadComponents(ctx, config, constants.WalletNode, config.Version, ""); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to download %s", toolPath)
		return err
	}
	if err = makeExecutable(ctx, config.PastelExecDir, toolPath); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to make %s executable", toolPath)
		return err
	}
	if err = setupComponentWorkingEnvironment(ctx, config,
		string(constants.WalletNode),
		config.Configurer.GetWalletNodeConfFile(config.WorkingDir),
		toolConfig,
	); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to setup %s", toolPath)
		return err
	}
	return nil
}

func installSNService(ctx context.Context, config *configs.Config, tryOpenPorts bool) error {
	log.WithContext(ctx).Info("Installing SuperNode service...")

	portList := GetSNPortList(config)

	snTempDirPath := filepath.Join(config.WorkingDir, constants.TempDir)
	rqWorkDirPath := filepath.Join(config.WorkingDir, constants.RQServiceDir)
	p2pDataPath := filepath.Join(config.WorkingDir, constants.P2PDataDir)
	mdlDataPath := filepath.Join(config.WorkingDir, constants.MDLDataDir)

	toolConfig, err := utils.GetServiceConfig(string(constants.SuperNode), configs.SupernodeDefaultConfig, &configs.SuperNodeConfig{
		LogFilePath:                     config.Configurer.GetSuperNodeLogFile(config.WorkingDir),
		LogCompress:                     constants.LogConfigDefaultCompress,
		LogMaxSizeMB:                    constants.LogConfigDefaultMaxSizeMB,
		LogMaxAgeDays:                   constants.LogConfigDefaultMaxAgeDays,
		LogMaxBackups:                   constants.LogConfigDefaultMaxBackups,
		LogLevelCommon:                  constants.SuperNodeDefaultCommonLogLevel,
		LogLevelP2P:                     constants.SuperNodeDefaultP2PLogLevel,
		LogLevelMetadb:                  constants.SuperNodeDefaultMetaDBLogLevel,
		LogLevelDD:                      constants.SuperNodeDefaultDDLogLevel,
		SNTempDir:                       snTempDirPath,
		SNWorkDir:                       config.WorkingDir,
		RQDir:                           rqWorkDirPath,
		DDDir:                           filepath.Join(config.Configurer.DefaultHomeDir(), constants.DupeDetectionServiceDir),
		SuperNodePort:                   portList[constants.SNPort],
		P2PPort:                         portList[constants.P2PPort],
		P2PDataDir:                      p2pDataPath,
		MDLPort:                         portList[constants.MDLPort],
		RAFTPort:                        portList[constants.RAFTPort],
		MDLDataDir:                      mdlDataPath,
		RaptorqPort:                     constants.RQServiceDefaultPort,
		NumberOfChallengeReplicas:       constants.NumberOfChallengeReplicas,
		StorageChallengeExpiredDuration: constants.StorageChallengeExpiredDuration,
	})
	if err != nil {
		return errors.Errorf("failed to get supernode config: %v", err)
	}

	toolPath := constants.SuperNodeExecName[utils.GetOS()]

	if err = downloadComponents(ctx, config, constants.SuperNode, config.Version, ""); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to download %s", toolPath)
		return err
	}
	if err = makeExecutable(ctx, config.PastelExecDir, toolPath); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to make %s executable", toolPath)
		return err
	}
	if err = setupComponentWorkingEnvironment(ctx, config,
		string(constants.SuperNode),
		config.Configurer.GetSuperNodeConfFile(config.WorkingDir),
		toolConfig); err != nil {

		log.WithContext(ctx).WithError(err).Errorf("Failed to setup %s", toolPath)
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
		if err = openPorts(ctx, config, portList); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to open ports")
			return err
		}
	}

	return nil
}

func checkInstallDir(ctx context.Context, config *configs.Config, installPath string, opMode string) error {
	defer log.WithContext(ctx).Infof("Install path is %s", installPath)

	if utils.CheckFileExist(config.PastelExecDir) {
		if config.OpMode == "install" {
			log.WithContext(ctx).Infof("Directory %s already exists...", installPath)
		}

		if yes, _ := AskUserToContinue(ctx, fmt.Sprintf("%s will overwrite content of %s. Do you want continue? Y/N", opMode, installPath)); !yes {
			log.WithContext(ctx).Info("Operation canceled by user. Exiting...")
			return fmt.Errorf("user terminated installation...")
		}
		config.Force = true
		return nil
	} else if config.OpMode == "update" {
		log.WithContext(ctx).Infof("Previous installation doesn't exist at %s. Noting to update. Exiting...", config.PastelExecDir)
		return fmt.Errorf("nothing to update. Exiting...")
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
	packagesRequired := []string{}
	for _, item := range packagesRequiredDirty {
		if _, value := keyGuard[item]; !value {
			keyGuard[item] = true
			packagesRequired = append(packagesRequired, item)
		}
	}

	if utils.GetOS() != constants.Linux {
		reqPkgsStr := strings.Join(packagesRequired, ",")
		log.WithContext(ctx).WithField("required-packages", reqPkgsStr).
			WithError(errors.New("install/update required pkgs")).
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

	if config.OpMode == "update" &&
		utils.GetOS() == constants.Linux &&
		len(packagesToUpdate) != 0 {

		pkgsUpdStr := strings.Join(packagesToUpdate, ",")

		if yes, _ := AskUserToContinue(ctx, "Some system packages ["+pkgsUpdStr+"] required for "+string(tool)+" need to be updated. Do you want to update them? Y/N"); !yes {
			log.WithContext(ctx).Warn("Exiting...")
			return fmt.Errorf("user terminated installation")
		}

		if err := installOrUpgradePackagesLinux(ctx, config, "upgrade", packagesToUpdate); err != nil {
			log.WithContext(ctx).WithField("packages-update", packagesToUpdate).
				WithError(err).
				Errorf("failed to update required pkgs - %s", pkgsUpdStr)
			return err
		}
	}

	if len(packagesMissing) == 0 {
		return nil
	}

	pkgsMissStr := strings.Join(packagesMissing, ",")

	if yes, _ := AskUserToContinue(ctx, "The system misses some packages ["+pkgsMissStr+"] required for "+string(tool)+". Do you want to install them? Y/N"); !yes {
		log.WithContext(ctx).Warn("Exiting...")
		return fmt.Errorf("user terminated installation")
	}

	return installOrUpgradePackagesLinux(ctx, config, "install", packagesMissing)
}

func installOrUpgradePackagesLinux(ctx context.Context, config *configs.Config, what string, pkgs []string) error {
	var out string
	var err error

	log.WithContext(ctx).WithField("packages", strings.Join(pkgs, ",")).
		Infof("system will now %s packages", what)

	var sudoStr string
	if len(config.UserPw) > 0 {
		sudoStr = "echo" + config.UserPw + "| sudo -S"
	} else {
		sudoStr = "sudo"
	}

	// Update repo
	_, err = RunCMD(sudoStr, "apt", "update")
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to update")
		return err
	}

	for _, pkg := range pkgs {
		log.WithContext(ctx).Infof("%sing package %s", what, pkg)

		if pkg == "google-chrome-stable" {
			if err := addGoogleRepo(ctx, config); err != nil {
				log.WithContext(ctx).WithError(err).Errorf("Failed to update pkg %s", pkg)
				return err
			}
			_, err = RunCMD(sudoStr, "apt", "update")
			if err != nil {
				log.WithContext(ctx).WithError(err).Error("Failed to update")
				return err
			}
		}

		out, err = RunCMD(sudoStr, "apt", "-y", what, pkg) //"install" or "upgrade"
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

	var sudoStr string
	if len(config.UserPw) > 0 {
		sudoStr = "echo" + config.UserPw + "| sudo -S"
	} else {
		sudoStr = "sudo"
	}

	_, err = RunCMD(sudoStr, "apt-key", "add", "/tmp/google-key.pub")
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

	_, err = RunCMD(sudoStr, "mv", "/tmp/google-chrome.list", constants.UbuntuSourceListPath)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to move /tmp/google-chrome.list to " + constants.UbuntuSourceListPath)
		return err
	}

	log.WithContext(ctx).Info("Added google ppa repo")
	return nil
}

func downloadComponents(ctx context.Context, config *configs.Config, installCommand constants.ToolType, version string, dstFolder string) (err error) {
	if _, err := os.Stat(config.PastelExecDir); os.IsNotExist(err) {
		if err := utils.CreateFolder(ctx, config.PastelExecDir, config.Force); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("create folder: %s", config.PastelExecDir)
			return err
		}
	}

	commandName := filepath.Base(string(installCommand))
	log.WithContext(ctx).Infof("Downloading %s...", commandName)

	downloadURL, archiveName, err := config.Configurer.GetDownloadURL(version, installCommand)
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

func setupComponentWorkingEnvironment(ctx context.Context, config *configs.Config,
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

func setupBasePasteWorkingEnvironment(ctx context.Context, config *configs.Config) error {
	// Ignore if not in "install" mode
	if config.OpMode != "install" {
		return nil
	}

	// create working dir
	if err := utils.CreateFolder(ctx, config.WorkingDir, config.Force); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to create folder %s", config.WorkingDir)
		return err
	}

	portList := GetSNPortList(config)
	config.RPCPort = portList[constants.NodePort]

	config.RPCUser = utils.GenerateRandomString(8)
	config.RPCPwd = utils.GenerateRandomString(15)

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
	if err := downloadZksnarkParams(ctx, config.Configurer.DefaultZksnarkDir(), config.Force, config.Version); err != nil &&
		!(os.IsExist(err) && !config.Force) {
		log.WithContext(ctx).WithError(err).Errorf("Failed to download Zksnark parameters into folder %s", config.Configurer.DefaultZksnarkDir())
		return fmt.Errorf("failed to download Zksnark parameters into folder %s - %v", config.Configurer.DefaultZksnarkDir(), err)
	}

	return nil
}

func updatePastelConfigFile(ctx context.Context, filePath string, config *configs.Config) error {
	cfgBuffer := bytes.Buffer{}

	// Populate pastel.conf line-by-line to file.
	cfgBuffer.WriteString("server=1\n")                          // creates server line
	cfgBuffer.WriteString("listen=1\n\n")                        // creates server line
	cfgBuffer.WriteString("rpcuser=" + config.RPCUser + "\n")    // creates  rpcuser line
	cfgBuffer.WriteString("rpcpassword=" + config.RPCPwd + "\n") // creates rpcpassword line

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
	err := ioutil.WriteFile(filePath, cfgBuffer.Bytes(), 0644)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Error saving file")
		return errors.Errorf("failed to save file changes: %v", err)
	}

	log.WithContext(ctx).Info("File updated successfully")

	return nil
}

func downloadZksnarkParams(ctx context.Context, path string, force bool, version string) error {
	log.WithContext(ctx).Info("Downloading pastel-param files:")

	zkParams := configs.ZksnarkParamsNamesV2
	if version != "beta" { //@TODO remove after Cezanne release
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
			if len(config.UserPw) > 0 {
				out, err = RunCMD("bash", "-c", "echo "+config.UserPw+" | sudo -S ufw allow "+portStr)
			} else {
				out, err = RunCMD("sudo", "ufw", "allow", portStr)
			}

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

func installServices(ctx context.Context, apps []constants.ToolType, config *configs.Config) error {
	sm, err := servicemanager.New(utils.GetOS(), config.Configurer.DefaultHomeDir())
	if err != nil {
		log.WithContext(ctx).Warn(err.Error())
		return nil // services aren't implemented for this OS
	}
	for _, app := range apps {
		err = sm.RegisterService(ctx, app, servicemanager.ResgistrationParams{
			Config:      config,
			Force:       config.Force,
			FlagDevMode: flagDevMode,
		})
		if err != nil {
			log.WithContext(ctx).Errorf("unable to register service %v: %v", app, err)
			return err
		}
		_, err := sm.StartService(ctx, app) // if app already running, this will be a noop
		if err != nil {
			log.WithContext(ctx).Errorf("unable to start service %v: %v", app, err)
			return err
		}
	}
	time.Sleep(5 * time.Second) // apply artificial buffer for services to start
	// verify services are up and running
	var nonRunningServices []constants.ToolType
	for _, app := range apps {
		isRunning := sm.IsRunning(ctx, app)
		if !isRunning {
			nonRunningServices = append(nonRunningServices, app)
		}
	}
	if len(nonRunningServices) > 0 {
		e := fmt.Errorf("unable to successfully start services: %+v", nonRunningServices)
		log.WithContext(ctx).Error(e.Error())
		return e
	}
	return nil
}
