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

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/errors"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/pastel-utility/constants"
	"github.com/pastelnetwork/pastel-utility/utils"
)

var (
	sshIP   string
	sshPort int
	sshKey  string
	sshUser string
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
		cli.NewFlag("started-remote", &config.StartedRemote).
			SetUsage(green("Optional, means that this command is executed remotely via ssh shell")),
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
				SetUsage(green("Optional, Location where to create pastel node directory on the remote computer (default: $HOME/.pastel)")),
			cli.NewFlag("remote-work-dir", &config.RemoteWorkingDir).SetAliases("w").
				SetUsage(green("Optional, Location where to create working directory on the remote computer (default: $HOME/pastel-utility)")),
		}
	}

	remoteFlags := []*cli.Flag{
		cli.NewFlag("ssh-ip", &sshIP).
			SetUsage(red("Required, SSH address of the remote host")).SetRequired(),
		cli.NewFlag("ssh-port", &sshPort).
			SetUsage(yellow("Optional, SSH port of the remote host, default is 22")).SetValue(22),
		cli.NewFlag("ssh-user", &sshUser).
			SetUsage(yellow("Optional, SSH user")),
		cli.NewFlag("ssh-key", &sshKey).
			SetUsage(yellow("Optional, Path to SSH private key")),
		cli.NewFlag("ssh-dir", &config.RemotePastelUtilityDir).SetAliases("rpud").
			SetUsage(yellow("Required, Location where to copy pastel-utility on the remote computer")).SetRequired(),
		cli.NewFlag("utility-path-to-copy", &config.CopyUtilityPath).
			SetUsage(yellow("Optional, path to the local pastel-utility file to copy to remote host")),
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
	if len(sshIP) == 0 {
		return fmt.Errorf("--ssh-ip IP address - Required, SSH address of the remote host")
	}

	if len(config.RemotePastelUtilityDir) == 0 {
		return fmt.Errorf("--ssh-dir RemotePastelUtilityDir - Required, pastel-utility path of the remote host")
	}

	var client *utils.Client
	log.WithContext(ctx).Infof("Connecting to remote host -> %s:%d...", sshIP, sshPort)
	if len(sshKey) == 0 {
		username, password, _ := utils.Credentials(sshUser, true)
		client, err = utils.DialWithPasswd(fmt.Sprintf("%s:%d", sshIP, sshPort), username, password)
	} else {
		username, _, _ := utils.Credentials(sshUser, false)
		client, err = utils.DialWithKey(fmt.Sprintf("%s:%d", sshIP, sshPort), username, sshKey)
	}
	if err != nil {
		return err
	}

	defer client.Close()

	log.WithContext(ctx).Info("Connected successfully")

	pastelUtilityFile := "pastel-utility"
	pastelUtilityPath := filepath.Join(config.RemotePastelUtilityDir, pastelUtilityFile)
	pastelUtilityPath = strings.ReplaceAll(pastelUtilityPath, "\\", "/")

	err = client.ShellCmd(ctx, fmt.Sprintf("rm -r -f %s", pastelUtilityPath))
	if err != nil {
		log.WithContext(ctx).Error("Failed to delete pastel-utility file")
		return err
	}

	// Transfer pastel utility to remote
	copyUtilityPath := config.CopyUtilityPath
	if len(copyUtilityPath) == 0 {
		copyUtilityPath = os.Args[0]
	}

	// scp pastel-utility to remote
	log.WithContext(ctx).Infof("Copying local pastel-utility executable to remote host - %s", copyUtilityPath)

	if err := client.Scp(copyUtilityPath, pastelUtilityPath); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to copy pastel-utility executable to remote host")
		return err
	}

	log.WithContext(ctx).Info("Successfully copied pastel-utility executable to remote host")

	if err = client.ShellCmd(ctx, fmt.Sprintf("chmod 755 %s", pastelUtilityPath)); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to change permission of pastel-utility")
		return err
	}

	checkIfRunningCommand := "ps afx | grep -E 'pasteld|rq-service|dd-service|supernode' | grep -v grep"
	if out, _ := client.Cmd(checkIfRunningCommand).Output(); len(out) != 0 {
		log.WithContext(ctx).Info("Supernode is running on remote host")

		if yes, _ := AskUserToContinue(ctx,
			"Do you want to stop it and continue? Y/N"); !yes {
			log.WithContext(ctx).Warn("Exiting...")
			return fmt.Errorf("user terminated installation")
		}

		log.WithContext(ctx).Info("Stopping supernode services...")
		stopSuperNodeCmd := fmt.Sprintf("%s stop supernode ", pastelUtilityPath)
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
	log.WithContext(ctx).Infof("pastel-utility path: %s", pastelUtilityPath)

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

	// disable config ports by tool, need do it manually due to having to enter
	// FIXME: add port config via ssh later
	remoteOptions = fmt.Sprintf("%s --started-remote", remoteOptions)

	installSuperNodeCmd := fmt.Sprintf("%s install supernode%s", pastelUtilityPath, remoteOptions)
	err = client.ShellCmd(ctx, installSuperNodeCmd)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to Installing Supernode")
		return err
	}

	log.WithContext(ctx).Info("Finished Installing Supernode Successfully, but not at all ^^")
	log.WithContext(ctx).Warn("Please manualy install chrome & config ports by yourself  as following:")
	showUserInstallGuideline(ctx, config)
	return nil
}

func runInstallDupeDetectionSubCommand(ctx context.Context, config *configs.Config) error {
	return installDupeDetection(ctx, config)
}

func runInstallDupeDetectionImgServerSubCommand(ctx context.Context, config *configs.Config) error {
	return íntallAppService(ctx, "dd-img-server", config)
}

func runComponentsInstall(ctx context.Context, config *configs.Config, installCommand constants.ToolType) error {
	if !utils.IsValidNetworkOpt(config.Network) {
		return fmt.Errorf("invalid --network provided. valid opts: %s", strings.Join(constants.NetworkModes, ","))
	}
	log.WithContext(ctx).Infof("initiaing in %s mode", config.Network)

	// create installation directory, example ~/pastel
	if err := createInstallDir(ctx, config, config.PastelExecDir); err != nil {
		//error was logged inside createInstallDir
		return err
	}

	if err := checkInstalledPackages(ctx, installCommand); err != nil {
		log.WithContext(ctx).WithError(err).Error("Missing packages...")
		return err
	}

	// install pasteld and pastel-cli; setup working dir (~/.pastel) and pastel.conf
	if installCommand == constants.PastelD ||
		installCommand == constants.WalletNode ||
		installCommand == constants.SuperNode {

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
		installCommand == constants.SuperNode {

		toolPath := constants.PastelRQServiceExecName[utils.GetOS()]
		toolConfig, err := utils.GetServiceConfig(string(constants.RQService), configs.RQServiceDefaultConfig, &configs.RQServiceConfig{
			HostName: "127.0.0.1",
			Port:     constants.RRServiceDefaultPort,
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
			RaptorqPort: constants.RRServiceDefaultPort,
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
			RaptorqPort:   constants.RRServiceDefaultPort,
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

		// Open ports
		if !config.StartedRemote {
			if err = openPorts(ctx, portList); err != nil {
				log.WithContext(ctx).WithError(err).Error("Failed to open ports")
				return err
			}
		} else {
			log.WithContext(ctx).Warn("Please open ports by manually!")
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

func checkInstalledPackages(ctx context.Context, tool constants.ToolType) (err error) {
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

	return installMissingReqPackagesLinux(ctx, notInstall)
}

func installMissingReqPackagesLinux(ctx context.Context, pkgs []string) error {
	log.WithContext(ctx).WithField("packages", strings.Join(pkgs, ",")).
		Info("system will now install missing packages")

	for _, pkg := range pkgs {
		out, err := RunCMD("sudo", "apt", "install", "-y", pkg)
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

func openPorts(ctx context.Context, portList []int) (err error) {
	// only open ports on SuperNode and this is only on Linux!!!
	var out string
	for k := range portList {
		log.WithContext(ctx).Infof("Opening port: %d", portList[k])

		portStr := fmt.Sprintf("%d", portList[k])
		switch utils.GetOS() {
		case constants.Linux:
			out, err = RunCMD("sudo", "ufw", "allow", portStr)
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

func showOpenPortGuideline(ctx context.Context, portList []int) {
	log.WithContext(ctx).Warn(" - Open ports:")

	for k := range portList {
		switch utils.GetOS() {
		case constants.Linux:
			log.WithContext(ctx).Warnf("   sudo ufw allow %d", portList[k])
		case constants.Windows:
			log.WithContext(ctx).Warnf("   netsh advfirewall firewall add rule name=TCP Port %d dir=in action=allow protocol=TCP localport=%d", portList[k], portList[k])
		case constants.Mac:
			log.WithContext(ctx).Warnf("   sudo ipfw allow tcp from any to any dest-port %d", portList[k])
		}
	}
}

func installDupeDetection(ctx context.Context, config *configs.Config) (err error) {
	log.WithContext(ctx).Info("Installing dd-service...")

	if err := checkInstalledPackages(ctx, constants.DDService); err != nil {
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

	// need to install manual by user
	if !config.StartedRemote {
		if err = installChrome(ctx, config); err != nil {
			return err
		}
	}

	ddBaseDir := filepath.Join(config.Configurer.DefaultHomeDir(), constants.DupeDetectionServiceDir)
	var pathList []interface{}
	for _, configItem := range constants.DupeDetectionConfigs {
		dupeDetectionDirPath := filepath.Join(ddBaseDir, configItem)
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

	if err = íntallAppService(ctx, "dd-img-server", config); err != nil {
		return err
	}

	log.WithContext(ctx).Info("Installing DupeDetection finished successfully")
	return nil
}

func íntallAppService(ctx context.Context, appName string, config *configs.Config) error {

	log.WithContext(ctx).Info("Installing" + appName)

	var systemdFile, serviceStartScript string
	var appServiceStartDir, appServiceStartFilePath string
	var err error

	// Service file - will be installed at /etc/systemd/system
	appServiceFileName := appName + ".service"
	systemdDir := "/etc/systemd/system"
	appServiceFilePath := filepath.Join(systemdDir, appServiceFileName)
	appServiceTmpFilePath := filepath.Join(config.PastelExecDir, appServiceFileName)

	// Executable script - called by systemd service
	appServiceStartFile := "start_" + appName + ".sh"

	switch appName {
	case "dd-img-server":
		appServiceStartDir = filepath.Join(config.PastelExecDir, "dd-service")
		appServiceStartFilePath = filepath.Join(appServiceStartDir, appServiceStartFile)

		// Systemd content
		systemdFile, err = utils.GetServiceConfig(appName, configs.DDImgServerService,
			&configs.DDImgServerServiceScript{
				DDImgServerStartScript: appServiceStartFilePath,
			})

		if err != nil {
			log.WithContext(ctx).WithError(err).Error("unable to create content of dd_img_server file")
			return fmt.Errorf("unable to create content of dd_img_server file - err: %s", err)
		}

		// Startup script content
		appBaseDir := filepath.Join(config.Configurer.DefaultHomeDir(), constants.DupeDetectionServiceDir)
		appServiceWorkDirPath := filepath.Join(appBaseDir, "img_server")

		serviceStartScript, err = utils.GetServiceConfig("dd_img_server_start", configs.DDImgServerStart,
			&configs.DDImgServerStartScript{
				DDImgServerDir: appServiceWorkDirPath,
			})
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("unable to create content of dd_img_server_start file")
			return fmt.Errorf("unable to create content of dd_img_server_start file - err: %s", err)
		}
	case "pasteld":

	case "supernode":

	case "rq-server":

	case "dd-server":

	default:
	}

	// create service file
	if err := utils.CreateAndWrite(ctx, config.Force, appServiceTmpFilePath, systemdFile); err != nil {
		return err
	}

	if _, err := RunCMD("sudo", "mv", appServiceTmpFilePath, appServiceFilePath); err != nil {
		log.WithContext(ctx).Error("Failed to move service file to systemd folder")
		return err
	}

	if _, err := RunCMD("sudo", "chmod", "644", appServiceFilePath); err != nil {
		log.WithContext(ctx).Errorf("Failed to make %s as executable", appServiceFilePath)
		return err
	}

	// create start script file
	if err := utils.CreateAndWrite(ctx, config.Force, appServiceStartFilePath, serviceStartScript); err != nil {
		return err
	}

	if err := makeExecutable(ctx, appServiceStartDir, appServiceStartFile); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to make %s executable", appServiceStartFilePath)
		return err
	}

	// Auto start service at boot
	log.WithContext(ctx).Info("Setting service for auto start on boot")
	if out, err := RunCMD("sudo", "systemctl", "enable", appName); err != nil {
		log.WithContext(ctx).WithFields(log.Fields{"message": out}).
			WithError(err).Error("unable to enable " + appName + " service")

		return fmt.Errorf("err enabling "+appName+" service - err: %s", err)
	}

	// Start script
	log.WithContext(ctx).Info("Starting service")
	if out, err := RunCMD("sudo", "systemctl", "start", appName); err != nil {
		log.WithContext(ctx).WithFields(log.Fields{"message": out}).
			WithError(err).Error("unable to start " + appName + " service")

		return fmt.Errorf("err starting "+appName+" service - err: %s", err)
	}

	log.WithContext(ctx).Info(appName + " installed successfully")
	return nil
}

func installChrome(ctx context.Context, config *configs.Config) (err error) {
	if utils.GetOS() == constants.Linux {
		log.WithContext(ctx).Infof("Downloading Chrome to install: %s \n", constants.ChromeDownloadURL[utils.GetOS()])

		err = utils.DownloadFile(ctx, filepath.Join(config.PastelExecDir, constants.ChromeExecFileName[utils.GetOS()]), constants.ChromeDownloadURL[utils.GetOS()])
		if err != nil {
			return err
		}

		if _, err = RunCMD("chmod", "777",
			filepath.Join(config.PastelExecDir, constants.ChromeExecFileName[utils.GetOS()])); err != nil {
			log.WithContext(ctx).Error("Failed to make chrome-install as executable")
			return err
		}

		log.WithContext(ctx).Infof("Installing Chrome : %s \n", filepath.Join(config.PastelExecDir, constants.ChromeExecFileName[utils.GetOS()]))

		RunCMDWithInteractive("sudo", "dpkg", "-i", filepath.Join(config.PastelExecDir, constants.ChromeExecFileName[utils.GetOS()]))

		utils.DeleteFile(filepath.Join(config.PastelExecDir, constants.ChromeExecFileName[utils.GetOS()]))
	}
	return nil
}

func showInstallChromeGuideline(ctx context.Context, _ *configs.Config) {
	if utils.GetOS() == constants.Linux {
		log.WithContext(ctx).Warn(" - Install chrome:")
		log.WithContext(ctx).Warnf("   wget %s", constants.ChromeDownloadURL[utils.GetOS()])
		log.WithContext(ctx).Warnf("   sudo dpkg -i %s", constants.ChromeExecFileName[utils.GetOS()])
	}
}

func showUserInstallGuideline(ctx context.Context, config *configs.Config) {
	showInstallChromeGuideline(ctx, config)
	portList := GetSNPortList(config)
	showOpenPortGuideline(ctx, portList)
}
