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
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/pastel-utility/constants"
	"github.com/pastelnetwork/pastel-utility/utils"
)

type installCommand uint8

const (
	nodeInstall installCommand = iota
	walletInstall
	superNodeInstall
	remoteInstall
	dupedetectionInstall
	dupedetectionImgServerInstall
	//highLevel
)

func setupSubCommand(config *configs.Config,
	installCommand installCommand,
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
	}

	if installCommand == superNodeInstall || installCommand == remoteInstall {
		serviceFlags := []*cli.Flag{
			cli.NewFlag("enable-service", &config.EnableService).
				SetUsage(green("Optional, start all apps automatically as systemd service")),
			cli.NewFlag("user-pw", &config.UserPw).
				SetUsage(green("Optional, password of current sudo user - so no sudo password request is prompted")),
			cli.NewFlag("sn-only", &config.InstallSNOnly).
				SetUsage(green("Optional, installl supernode only")),
		}
		commonFlags = append(commonFlags, serviceFlags...)
	}

	var dirsFlags []*cli.Flag

	if installCommand != remoteInstall {
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
		cli.NewFlag("ssh-user-pw", &config.UserPw).
			SetUsage(red("Required, password of remote user - so no sudo request is promoted")).SetRequired(),
		cli.NewFlag("ssh-key", &config.RemoteSSHKey).
			SetUsage(yellow("Optional, Path to SSH private key")),
	}

	dupeFlags := []*cli.Flag{
		cli.NewFlag("force", &config.Force).SetAliases("f").
			SetUsage(green("Optional, Force to overwrite config files and re-download ZKSnark parameters")),
	}

	var commandName, commandMessage string
	var commandFlags []*cli.Flag

	switch installCommand {
	case nodeInstall:
		commandFlags = append(dirsFlags, commonFlags[:]...)
		commandName = "node"
		commandMessage = "Install node"
	case walletInstall:
		commandFlags = append(dirsFlags, commonFlags[:]...)
		commandName = string(constants.WalletNode)
		commandMessage = "Install walletnode"
	case superNodeInstall:
		commandFlags = append(dirsFlags, commonFlags[:]...)
		commandName = string(constants.SuperNode)
		commandMessage = "Install supernode"
	case remoteInstall:
		commandFlags = append(append(dirsFlags, commonFlags[:]...), remoteFlags[:]...)
		commandName = "remote"
		commandMessage = "Install supernode remote"
	case dupedetectionInstall:
		commandFlags = append(dirsFlags, dupeFlags[:]...)
		commandName = "dupedetection"
		commandMessage = "Install dupedetection"
	case dupedetectionImgServerInstall:
		commandFlags = append(dirsFlags, dupeFlags[:]...)
		commandName = "imgserver"
		commandMessage = "Install dupedetection image server"
	default:
		commandFlags = append(append(dirsFlags, commonFlags[:]...), remoteFlags[:]...)
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

	installNodeSubCommand := setupSubCommand(config, nodeInstall, runInstallNodeSubCommand)
	installWalletSubCommand := setupSubCommand(config, walletInstall, runInstallWalletSubCommand)
	installSuperNodeSubCommand := setupSubCommand(config, superNodeInstall, runInstallSuperNodeSubCommand)
	installSuperNodeRemoteSubCommand := setupSubCommand(config, remoteInstall, runInstallSuperNodeRemoteSubCommand)
	installSuperNodeSubCommand.AddSubcommands(installSuperNodeRemoteSubCommand)
	installDupeDetecionSubCommand := setupSubCommand(config, dupedetectionInstall, runInstallDupeDetectionSubCommand)
	installDupeDetecionImgServerSubCommand := setupSubCommand(config, dupedetectionImgServerInstall, runInstallDupeDetectionImgServerSubCommand)

	installCommand := cli.NewCommand("install")
	installCommand.SetUsage(blue("Performs installation and initialization of the system for both WalletNode and SuperNodes"))
	installCommand.AddSubcommands(installNodeSubCommand)
	installCommand.AddSubcommands(installWalletSubCommand)
	installCommand.AddSubcommands(installSuperNodeSubCommand)
	installCommand.AddSubcommands(installDupeDetecionSubCommand)
	installCommand.AddSubcommands(installDupeDetecionImgServerSubCommand)

	return installCommand
}

func runInstallNodeSubCommand(ctx context.Context, config *configs.Config) (err error) {
	return runComponentsInstall(ctx, config, constants.PastelD)
}

func runInstallWalletSubCommand(ctx context.Context, config *configs.Config) (err error) {
	return runComponentsInstall(ctx, config, constants.WalletNode)
}

func runInstallSuperNodeSubCommand(ctx context.Context, config *configs.Config) (err error) {
	return runComponentsInstall(ctx, config, constants.SuperNode)
}

func runInstallSuperNodeRemoteSubCommand(ctx context.Context, config *configs.Config) (err error) {
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
	if len(config.RemotePastelExecDir) > 0 {
		remoteOptions = fmt.Sprintf("%s --dir=%s", remoteOptions, config.RemotePastelExecDir)
	}

	if len(config.RemoteWorkingDir) > 0 {
		remoteOptions = fmt.Sprintf("%s --work-dir=%s", remoteOptions, config.RemoteWorkingDir)
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

func runInstallDupeDetectionSubCommand(ctx context.Context, config *configs.Config) error {
	return installDupeDetection(ctx, config)
}

func runInstallDupeDetectionImgServerSubCommand(ctx context.Context, config *configs.Config) error {
	return installAppService(ctx, string(constants.DDImgService), config)
}

func runComponentsInstall(ctx context.Context, config *configs.Config, installCommand constants.ToolType) error {
	if config.OpMode == "install" {
		if !utils.IsValidNetworkOpt(config.Network) {
			return fmt.Errorf("invalid --network provided. valid opts: %s", strings.Join(constants.NetworkModes, ","))
		}
		log.WithContext(ctx).Infof("initiating in %s mode", config.Network)
	}

	// create installation directory, example ~/pastel
	if err := createInstallDir(ctx, config, config.PastelExecDir); err != nil {
		//error was logged inside createInstallDir
		return err
	}

	if err := checkInstalledPackages(ctx, config, installCommand); err != nil {
		log.WithContext(ctx).WithError(err).Error("Missing packages...")
		return err
	}

	// install pasteld and pastel-cli; setup working dir (~/.pastel) and pastel.conf
	if installCommand == constants.PastelD ||
		installCommand == constants.WalletNode ||
		(installCommand == constants.SuperNode) {

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
	}
	// install rqservice and its config
	if installCommand == constants.WalletNode ||
		(installCommand == constants.SuperNode && !config.InstallSNOnly) {

		toolPath := constants.PastelRQServiceExecName[utils.GetOS()]
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

		if err = setupComponentWorkingEnvironment(ctx, config,
			string(constants.RQService),
			config.Configurer.GetRQServiceConfFile(config.WorkingDir),
			toolConfig); err != nil {

			log.WithContext(ctx).WithError(err).Errorf("Failed to setup %s", toolPath)
			return err
		}
	}
	// install WalletNode and its config
	if installCommand == constants.WalletNode {
		toolPath := constants.WalletNodeExecName[utils.GetOS()]

		burnAddress := constants.BurnAddressMainnet
		if config.Network == constants.NetworkTestnet {
			burnAddress = constants.BurnAddressTestnet
		}

		wnTempDirPath := filepath.Join(config.WorkingDir, constants.TempDir)
		rqWorkDirPath := filepath.Join(config.WorkingDir, constants.RQServiceDir)

		toolConfig, err := utils.GetServiceConfig(string(constants.WalletNode), configs.WalletDefaultConfig, &configs.WalletNodeConfig{
			LogLevel:    constants.WalletNodeDefaultLogLevel,
			LogFilePath: config.Configurer.GetWalletNodeLogFile(config.WorkingDir),
			WNTempDir:   wnTempDirPath,
			WNWorkDir:   config.WorkingDir,
			RQDir:       rqWorkDirPath,
			BurnAddress: burnAddress,
			RaptorqPort: constants.RQServiceDefaultPort,
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
			toolConfig); err != nil {

			log.WithContext(ctx).WithError(err).Errorf("Failed to setup %s", toolPath)
			return err
		}

		// Start all wallet nodes apps as service
		if config.EnableService {
			appServiceNames := []string{
				string(constants.PastelD),
				string(constants.RQService),
				string(constants.WalletNode),
			}

			for _, appName := range appServiceNames {
				if err = installAppService(ctx, appName, config); err != nil {
					log.WithContext(ctx).WithError(err).Error("Failed to install " + appName + " service")
					return err
				}
			}
		}
	}

	if installCommand == constants.SuperNode {
		// install SuperNode, dd-service and their configs; open ports
		portList := GetSNPortList(config)

		snTempDirPath := filepath.Join(config.WorkingDir, constants.TempDir)
		rqWorkDirPath := filepath.Join(config.WorkingDir, constants.RQServiceDir)
		p2pDataPath := filepath.Join(config.WorkingDir, constants.P2PDataDir)
		mdlDataPath := filepath.Join(config.WorkingDir, constants.MDLDataDir)

		toolConfig, err := utils.GetServiceConfig(string(constants.SuperNode), configs.SupernodeDefaultConfig, &configs.SuperNodeConfig{
			LogLevel:      constants.SuperNodeDefaultLogLevel,
			LogFilePath:   config.Configurer.GetSuperNodeLogFile(config.WorkingDir),
			SNTempDir:     snTempDirPath,
			SNWorkDir:     config.WorkingDir,
			RQDir:         rqWorkDirPath,
			DDDir:         filepath.Join(config.Configurer.DefaultHomeDir(), constants.DupeDetectionServiceDir),
			SuperNodePort: portList[constants.SNPort],
			P2PPort:       portList[constants.P2PPort],
			P2PDataDir:    p2pDataPath,
			MDLPort:       portList[constants.MDLPort],
			RAFTPort:      portList[constants.RAFTPort],
			MDLDataDir:    mdlDataPath,
			RaptorqPort:   constants.RQServiceDefaultPort,
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

		if !config.InstallSNOnly {
			if err = installDupeDetection(ctx, config); err != nil {
				log.WithContext(ctx).WithError(err).Error("Failed to install dd-service")
				return err
			}

			if err := utils.CreateFolder(ctx, snTempDirPath, config.Force); err != nil {
				log.WithContext(ctx).WithError(err).Errorf("Failed to create folder %s", snTempDirPath)
				return err
			}

			if err := utils.CreateFolder(ctx, rqWorkDirPath, config.Force); err != nil {
				log.WithContext(ctx).WithError(err).Errorf("Failed to create folder %s", rqWorkDirPath)
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
		}

		// Open ports
		if err = openPorts(ctx, config, portList); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to open ports")
			return err
		}

		// installAppsAsService - pasteld, supernode, rq-server, dd-server
		if config.EnableService {
			appServiceNames := []string{
				string(constants.PastelD),
				string(constants.RQService),
				string(constants.DDService),
				string(constants.SuperNode),
			}

			if !config.InstallSNOnly {
				appServiceNames = []string{
					string(constants.PastelD),
					string(constants.SuperNode),
				}
			}

			for _, appName := range appServiceNames {
				if err = installAppService(ctx, appName, config); err != nil {
					log.WithContext(ctx).WithError(err).Error("Failed to install " + appName + " service")
					return err
				}
			}
		}
	}

	// Install node as service
	if (installCommand == constants.PastelD) && (config.EnableService) {
		if err := installAppService(ctx, string(constants.PastelD), config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to install " + appName + " service")
			return err
		}
	}

	return nil
}

func createInstallDir(ctx context.Context, config *configs.Config, installPath string) error {
	defer log.WithContext(ctx).Infof("Install path is %s", installPath)

	if err := utils.CreateFolder(ctx, installPath, config.Force); os.IsExist(err) {

		if yes, _ := AskUserToContinue(ctx, fmt.Sprintf("%s - %s. Do you want continue to install? Y/N",
			err.Error(), installPath)); !yes {

			log.WithContext(ctx).Warn("Exiting...")
			return fmt.Errorf("user terminated installation")
		}

		config.Force = true
		if err = utils.CreateFolder(ctx, installPath, config.Force); err != nil {
			log.WithContext(ctx).WithError(err).Error("Exiting...")
			return fmt.Errorf("failed to create install directory - %s (%v)", installPath, err)
		}
	} else if err != nil {
		log.WithContext(ctx).WithError(err).Error("Exiting...")
		return fmt.Errorf("failed to create install directory - %s (%v)", installPath, err)
	}

	return nil
}

func checkInstalledPackages(ctx context.Context, config *configs.Config, tool constants.ToolType) (err error) {
	// TODO: 1) must offer to install missing packages
	installedCmd := utils.GetInstalledPackages(ctx)
	var notInstall []string
	for _, p := range constants.DependenciesPackages[tool][utils.GetOS()] {
		if _, ok := installedCmd[p]; !ok {
			notInstall = append(notInstall, p)
		}
	}

	if len(notInstall) == 0 {
		return nil
	}

	pkgsStr := strings.Join(notInstall, ",")
	// TODO: devise a mechanism for installing pkgs for mac & windows
	if utils.GetOS() != constants.Linux {
		log.WithContext(ctx).WithField("missing-packages", pkgsStr).
			WithError(errors.New("missing required pkgs")).
			Error("automatic install for required pkgs only set up for linux")

		return fmt.Errorf("missing required pkgs: %s", pkgsStr)
	}

	return installMissingReqPackagesLinux(ctx, config, notInstall)
}

func installMissingReqPackagesLinux(ctx context.Context, config *configs.Config, pkgs []string) error {
	var out string
	var err error

	log.WithContext(ctx).WithField("packages", strings.Join(pkgs, ",")).
		Info("system will now install missing packages")

	// Add google ssl key
	log.WithContext(ctx).Info("Adding google ssl key ...")
	_, err = RunCMD("bash", "-c", "wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | sudo apt-key add - 2>/dev/null")
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to add google ssl key")
		return err
	}
	log.WithContext(ctx).Info("Added google ssl key")

	// Add google repo
	log.WithContext(ctx).Info("Adding google ppa repo ...")
	_, err = RunCMD("bash", "-c", "echo 'deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main' | sudo tee /etc/apt/sources.list.d/google-chrome.list")
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to add google repo")
		return err
	}
	log.WithContext(ctx).Info("Added google ppa repo")

	// Update
	_, err = RunCMD("bash", "-c", "sudo apt-get update")
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to update")
		return err
	}

	for _, pkg := range pkgs {

		if len(config.UserPw) > 0 {
			out, err = RunCMD("bash", "-c", "echo "+config.UserPw+" | sudo apt-get install  -y "+pkg)
		} else {
			out, err = RunCMD("sudo", "apt-get", "install", "-y", pkg)
		}

		if err != nil {
			log.WithContext(ctx).WithFields(log.Fields{"message": out, "package": pkg}).
				WithError(err).Error("unable to install required package")

			return fmt.Errorf("err installing required pkg : %s - err: %s", pkg, err)
		}
	}

	return nil
}

func downloadComponents(ctx context.Context, config *configs.Config, installCommand constants.ToolType, version string, dstFolder string) (err error) {
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
	if err := downloadZksnarkParams(ctx, config.Configurer.DefaultZksnarkDir(), config.Force); err != nil &&
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

func downloadZksnarkParams(ctx context.Context, path string, force bool) error {
	log.WithContext(ctx).Info("Downloading pastel-param files:")
	for _, zksnarkParamsName := range configs.ZksnarkParamsNames {
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

func installDupeDetection(ctx context.Context, config *configs.Config) (err error) {
	log.WithContext(ctx).Info("Installing dd-service...")

	if err := checkInstalledPackages(ctx, config, constants.DDService); err != nil {
		log.WithContext(ctx).WithError(err).Error("Missing packages...")
		return err
	}

	// Download dd-service
	if config.Version == "" {
		config.Version = "beta"
	}
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

		if err = installAppService(ctx, string(constants.DDImgService), config); err != nil {
			return err
		}
	}

	log.WithContext(ctx).Info("Installing DupeDetection finished successfully")
	return nil
}

func installAppService(ctx context.Context, appName string, config *configs.Config) error {

	log.WithContext(ctx).Info("Installing " + appName + " as service")

	// Get home dir
	SystemdUserDir := filepath.Join(config.Configurer.DefaultHomeDir(), constants.SystemdUserDir)

	if err := utils.CreateFolder(ctx, SystemdUserDir, config.Force); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to create directory : %s", SystemdUserDir)
		return err
	}

	var systemdFile string
	var err error
	var execCmd, execPath, workDir string

	// Service file - will be installed at /etc/systemd/system
	appServiceFileName := constants.SystemdServicePrefix + appName + ".service"
	appServiceFilePath := filepath.Join(SystemdUserDir, appServiceFileName)

	switch appName {
	case string(constants.DDImgService):

		appBaseDir := filepath.Join(config.Configurer.DefaultHomeDir(), constants.DupeDetectionServiceDir)
		appServiceWorkDirPath := filepath.Join(appBaseDir, "img_server")

		execCmd = "python3 -m  http.server 8080"
		workDir = appServiceWorkDirPath

	case string(constants.PastelD):
		var extIP string

		// Get pasteld path
		if execPath, err = checkPastelFilePath(ctx, config.PastelExecDir, constants.PasteldName[utils.GetOS()]); err != nil {
			log.WithContext(ctx).WithError(err).Error("Could not find" + appName + " executable file")
			return err
		}

		// Get external IP
		if extIP, err = GetExternalIPAddress(); err != nil {
			log.WithContext(ctx).WithError(err).Error("Could not get external IP address")
			return err
		}

		execCmd = execPath + " --datadir=" + config.WorkingDir + " --externalip=" + extIP
		workDir = config.PastelExecDir

	case string(constants.RQService):
		if execPath, err = checkPastelFilePath(ctx, config.PastelExecDir, constants.PastelRQServiceExecName[utils.GetOS()]); err != nil {
			log.WithContext(ctx).WithError(err).Error("Could not find" + appName + " executable file")
			return err
		}
		rqServiceArgs := fmt.Sprintf("--config-file=%s", config.Configurer.GetRQServiceConfFile(config.WorkingDir))
		execCmd = execPath + " " + rqServiceArgs
		workDir = config.PastelExecDir

	case string(constants.DDService):
		if execPath, err = checkPastelFilePath(ctx, config.PastelExecDir, utils.GetDupeDetectionExecName()); err != nil {
			log.WithContext(ctx).WithError(err).Error("Could not find" + appName + " executable file")
			return err
		}

		ddConfigFilePath := filepath.Join(config.Configurer.DefaultHomeDir(),
			constants.DupeDetectionServiceDir,
			constants.DupeDetectionSupportFilePath,
			constants.DupeDetectionConfigFilename)

		execCmd = "python3 " + execPath + " " + ddConfigFilePath
		workDir = config.PastelExecDir

	case string(constants.SuperNode):
		if execPath, err = checkPastelFilePath(ctx, config.PastelExecDir, constants.SuperNodeExecName[utils.GetOS()]); err != nil {
			log.WithContext(ctx).WithError(err).Error("Could not find" + appName + " executable file")
			return err
		}

		supernodeConfigPath := config.Configurer.GetSuperNodeConfFile(config.WorkingDir)

		execCmd = execPath + " --config-file=" + supernodeConfigPath
		workDir = config.PastelExecDir

	case string(constants.WalletNode):
		if execPath, err = checkPastelFilePath(ctx, config.PastelExecDir, constants.WalletNodeExecName[utils.GetOS()]); err != nil {
			log.WithContext(ctx).WithError(err).Error("Could not find" + appName + " executable file")
			return err
		}

		walletnodeConfigFile := config.Configurer.GetWalletNodeConfFile(config.WorkingDir)

		execCmd = execPath + " --config-file=" + walletnodeConfigFile
		if flagDevMode {
			execCmd += " --swagger"
		}
		workDir = config.PastelExecDir

	default:
		return nil
	}

	// Create systemd file
	systemdFile, err = utils.GetServiceConfig(appName, configs.SystemdService,
		&configs.SystemdServiceScript{
			Desc:    appName + " daemon",
			ExecCmd: execCmd,
			WorkDir: workDir,
		})

	if err != nil {
		log.WithContext(ctx).WithError(err).Error("unable to create content of " + appServiceFileName + " file")
		return fmt.Errorf("unable to create content of "+appServiceFileName+" file - err: %s", err)
	}

	// write systemdFile to SystemdUserDir with mode 0644
	if err := ioutil.WriteFile(appServiceFilePath, []byte(systemdFile), 0644); err != nil {
		log.WithContext(ctx).WithError(err).Error("unable to write " + appServiceFileName + " file")
	}

	// Enable service
	log.WithContext(ctx).Info("Setting service for auto start on boot")
	if out, err := RunCMD("systemctl", "--user", "enable", appServiceFileName); err != nil {
		log.WithContext(ctx).WithFields(log.Fields{"message": out}).
			WithError(err).Error("unable to enable " + appServiceFileName + " service")

		return fmt.Errorf("err enabling "+appServiceFileName+" - err: %s", err)
	}

	// Start the service
	if err := startSystemdService(ctx, appName, config); err != nil {
		log.WithContext(ctx).Errorf("Failed to start %s", appServiceFilePath)
	}

	// Check if service is already running
	time.Sleep(3 * time.Second)
	checkServiceRunning(appName)

	log.WithContext(ctx).Info(appName + " installed successfully")

	return nil
}

func checkServiceInstalled(appName string) error {
	appServiceFileName := constants.SystemdServicePrefix + appName + ".service"

	if _, err := os.Stat(filepath.Join(constants.SystemdUserDir, appServiceFileName)); os.IsNotExist(err) {
		return fmt.Errorf(appServiceFileName + " is not yet installed")
	}

	return nil
}

func checkServiceRunning(appName string) error {
	appServiceFileName := constants.SystemdServicePrefix + appName + ".service"

	_, err := RunCMD("systemctl", "--user", "is-active", "--quiet", appServiceFileName)

	return err
}

// Check if app is installed as service - if yes, then start it
func startSystemdService(ctx context.Context, appName string, _ *configs.Config) error {

	if err := checkServiceInstalled(appName); err != nil {
		return fmt.Errorf("Service " + appName + " is not installed as service")
	}

	// Start app, if it is not running
	err := checkServiceRunning(appName)
	if err == nil {
		log.WithContext(ctx).Info(appName + " is already running!")

	} else {
		appServiceFileName := constants.SystemdServicePrefix + appName + ".service"

		// Start service
		log.WithContext(ctx).Info("Starting service " + appServiceFileName)

		out, err := RunCMD("systemctl", "--user", "start", appServiceFileName)

		if err != nil {
			log.WithContext(ctx).WithFields(log.Fields{"message": out}).
				WithError(err).Error("unable to start " + appServiceFileName)

			return fmt.Errorf("err starting "+appServiceFileName+" - err: %s", err)
		}
	}

	return nil
}

// Stop systemd service if it is running, return nil if the service is found
func stopSystemdService(ctx context.Context, appName string, _ *configs.Config) error {

	if err := checkServiceInstalled(appName); err != nil {
		return fmt.Errorf("Service " + appName + " is not installed as service")
	}

	appServiceFileName := constants.SystemdServicePrefix + appName + ".service"

	if err := checkServiceRunning(appName); err == nil {

		out, err := RunCMD("systemctl", "--user", "stop", appServiceFileName)

		if err != nil {
			log.WithContext(ctx).WithFields(log.Fields{"message": out}).
				WithError(err).Error("unable to stop " + appServiceFileName)

			return fmt.Errorf("err stopping "+appServiceFileName+" - err: %s", err)
		}

		log.WithContext(ctx).Infof("Service %s stopped", appServiceFileName)
	} else {
		log.WithContext(ctx).Infof("Service %s is not running", appServiceFileName)
	}

	return nil
}
