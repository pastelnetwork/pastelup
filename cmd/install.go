package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
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
)

type installCommand uint8

const (
	nodeInstall installCommand = iota
	walletInstall
	superNodeInstall
	remoteInstall
	dupedetectionInstall
	//highLevel
)

func setupSubCommand(config *configs.Config,
	installCommand installCommand,
	f func(context.Context, *configs.Config) error,
) *cli.Command {
	commonFlags := []*cli.Flag{
		cli.NewFlag("force", &config.Force).SetAliases("f").
			SetUsage(green("Optional, Force to overwrite config files and re-download ZKSnark parameters")),
		cli.NewFlag("peers", &config.Peers).SetAliases("p").
			SetUsage(green("Optional, List of peers to add into pastel.conf file, must be in the format - \"ip\" or \"ip:port\"")),
		cli.NewFlag("release", &config.Version).SetAliases("r").
			SetUsage(green("Optional, Pastel version to install")).SetValue("latest"),
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
			cli.NewFlag("dir", &config.RemotePastelExecDir).SetAliases("d").
				SetUsage(green("Optional, Location where to create pastel node directory on the remote computer (default: $HOME/pastel-utility)")),
			cli.NewFlag("work-dir", &config.RemoteWorkingDir).SetAliases("w").
				SetUsage(green("Optional, Location where to create working directory on the remote computer (default: $HOME/pastel-utility)")),
		}
	}

	remoteFlags := []*cli.Flag{
		cli.NewFlag("ssh-ip", &sshIP).
			SetUsage(yellow("Required, SSH address of the remote host")).SetRequired(),
		cli.NewFlag("ssh-port", &sshPort).
			SetUsage(yellow("Optional, SSH port of the remote host, default is 22")).SetValue(22),
		cli.NewFlag("ssh-key", &sshKey).
			SetUsage(yellow("Optional, Path to SSH private key")),
		cli.NewFlag("ssh-dir", &config.RemotePastelUtilityDir).SetAliases("rpud").
			SetUsage(yellow("Required, Location where to copy pastel-utility on the remote computer")).SetRequired(),
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
		commandName = "walletnode"
		commandMessage = "Install walletnode"
	case superNodeInstall:
		commandFlags = append(dirsFlags, commonFlags[:]...)
		commandName = "supernode"
		commandMessage = "Install supernode"
	case remoteInstall:
		commandFlags = append(append(dirsFlags, commonFlags[:]...), remoteFlags[:]...)
		commandName = "remote"
		commandMessage = "Install supernode remote"
	case dupedetectionInstall:
		commandFlags = dupeFlags
		commandName = "dupedetection"
		commandMessage = "Install dupedetection"
	default:
		commandFlags = append(append(dirsFlags, commonFlags[:]...), remoteFlags[:]...)
	}

	subCommand := cli.NewCommand(commandName)
	subCommand.SetUsage(cyan(commandMessage))
	subCommand.AddFlags(commandFlags...)
	if f != nil {
		subCommand.SetActionFunc(func(ctx context.Context, args []string) error {
			ctx, err := configureLogging(ctx, commandMessage, config)
			if err != nil {
				return err
			}

			if installCommand != remoteInstall {
				if err = installPackages(ctx); err != nil {
					return err
				}
			}

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			sys.RegisterInterruptHandler(cancel, func() {
				log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
				os.Exit(0)
			})

			log.WithContext(ctx).Info("Started")
			err = f(ctx, config)
			if err != nil {
				return err
			}
			log.WithContext(ctx).Info("Finished successfully!")
			return nil
		})
	}
	return subCommand
}

func setupInstallCommand() *cli.Command {
	config := configs.GetConfig()

	installNodeSubCommand := setupSubCommand(config, nodeInstall, runInstallNodeSubCommand)
	installWalletSubCommand := setupSubCommand(config, walletInstall, runInstallWalletSubCommand)
	installSuperNodeSubCommand := setupSubCommand(config, superNodeInstall, runInstallSuperNodeSubCommand)
	installSuperNodeRemoteSubCommand := setupSubCommand(config, remoteInstall, runInstallSuperNodeRemoteSubCommand)
	installSuperNodeSubCommand.AddSubcommands(installSuperNodeRemoteSubCommand)
	installDupeDetecionSubCommand := setupSubCommand(config, dupedetectionInstall, runInstallDupeDetectionSubCommand)

	installCommand := cli.NewCommand("install")
	installCommand.SetUsage(blue("Performs installation and initialization of the system for both WalletNode and SuperNodes"))
	installCommand.AddSubcommands(installNodeSubCommand)
	installCommand.AddSubcommands(installWalletSubCommand)
	installCommand.AddSubcommands(installSuperNodeSubCommand)
	installCommand.AddSubcommands(installDupeDetecionSubCommand)
	//installCommand := setupSubCommand(config, highLevel, nil)

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

	if err = InitializeFunc(ctx, config); err != nil {
		return err
	}

	var client *utils.Client
	log.WithContext(ctx).Infof("Connecting to SSH Hot node wallet -> %s:%d...", sshIP, sshPort)
	if len(sshKey) == 0 {
		username, password, _ := credentials(true)
		client, err = utils.DialWithPasswd(fmt.Sprintf("%s:%d", sshIP, sshPort), username, password)
	} else {
		username, _, _ := credentials(false)
		client, err = utils.DialWithKey(fmt.Sprintf("%s:%d", sshIP, sshPort), username, sshKey)
	}
	if err != nil {
		return err
	}

	defer client.Close()

	log.WithContext(ctx).Info("Connected successfully")

	pastelUtilityPath := filepath.Join(config.RemotePastelUtilityDir, "pastel-utility")
	pastelUtilityPath = strings.ReplaceAll(pastelUtilityPath, "\\", "/")
	pastelUtilityDownloadPath := constants.PastelUtilityDownloadURL

	_, err = client.Cmd(fmt.Sprintf("rm -r -f %s", pastelUtilityPath)).Output()
	if err != nil {
		log.WithContext(ctx).Error("Failed to delete pastel-utility file")
		return err
	}

	log.WithContext(ctx).Info("Downloading Pastel-Utility Executable...")
	_, err = client.Cmd(fmt.Sprintf("wget -O %s %s", pastelUtilityPath, pastelUtilityDownloadPath)).Output()

	log.WithContext(ctx).Debugf("wget -O %s  %s", pastelUtilityPath, pastelUtilityDownloadPath)
	if err != nil {
		log.WithContext(ctx).Error("Failed to download pastel-utility")
		return err
	}

	log.WithContext(ctx).Info("Finished Downloading Pastel-Utility Successfully")

	_, err = client.Cmd(fmt.Sprintf("chmod 777 /%s", pastelUtilityPath)).Output()
	if err != nil {
		log.WithContext(ctx).Error("Failed to change permission of pastel-utility")
		return err
	}

	_, err = client.Cmd(fmt.Sprintf("%s stop supernode ", pastelUtilityPath)).Output()
	if err != nil {
		log.WithContext(ctx).Errorf("failed to stop supernode, err: %s", err)
		return err
	}

	log.WithContext(ctx).Info("Installing Supernode ...")

	log.WithContext(ctx).Debugf("pastel-utility path: %s", pastelUtilityPath)

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

	stdin := bytes.NewBufferString(fmt.Sprintf("/%s install supernode%s", pastelUtilityPath, remoteOptions))
	var stdout, stderr io.Writer

	return client.Shell().SetStdio(stdin, stdout, stderr).Start()
}

func runInstallDupeDetectionSubCommand(ctx context.Context, config *configs.Config) error {
	if err := installDupeDetection(ctx, config); err != nil {
		return err
	}
	return nil
}

func initNodeDownloadPath(ctx context.Context, config *configs.Config, nodeInstallPath string) (nodePath string, err error) {
	defer log.WithContext(ctx).Infof("Node install path is %s", nodeInstallPath)

	if err = utils.CreateFolder(ctx, nodeInstallPath, config.Force); os.IsExist(err) {
		reader := bufio.NewReader(os.Stdin)
		log.WithContext(ctx).Warnf("%s. Do you want continue to install? Y/N", err.Error())
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			return "", readErr
		}

		if strings.TrimSpace(line) == "Y" || strings.TrimSpace(line) == "y" {
			config.Force = true
			if err = InitializeFunc(ctx, config); err != nil {
				return "", err
			}
			if err = utils.CreateFolder(ctx, nodeInstallPath, config.Force); err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}

	return "", nil
}

func runComponentsInstall(ctx context.Context, config *configs.Config, installCommand constants.ToolType) (err error) {
	if err = InitializeFunc(ctx, config); err != nil {
		return err
	}

	if _, err = initNodeDownloadPath(ctx, config, config.PastelExecDir); err != nil {
		return err
	}

	switch installCommand {
	case constants.PastelD:
		if err = installComponent(ctx, config, constants.PastelD, config.Version); err != nil {
			return err
		}
	case constants.WalletNode:
		if err = installComponent(ctx, config, constants.PastelD, "latest"); err != nil {
			return err
		}

		if err = installComponent(ctx, config, constants.WalletNode, config.Version); err != nil {
			return err
		}

		if err = installComponent(ctx, config, constants.RQService, config.Version); err != nil {
			return err
		}
	case constants.SuperNode:
		if err = installComponent(ctx, config, constants.PastelD, "latest"); err != nil {
			return err
		}

		if err = installComponent(ctx, config, constants.SuperNode, config.Version); err != nil {
			return err
		}

		if err = installComponent(ctx, config, constants.RQService, config.Version); err != nil {
			return err
		}

		// Open ports
		openErr := openPort(ctx, constants.PortList)
		if openErr != nil {
			return openErr
		}

		log.WithContext(ctx).Info("Installing dd-service...")
		if err = installDupeDetection(ctx, config); err != nil {
			log.WithContext(ctx).Error("Installing dd-service executable failed")
			return err
		}
		log.WithContext(ctx).Info("The dd-service Installed Successfully")
	}

	return nil
}

func installComponent(ctx context.Context, config *configs.Config, installCommand constants.ToolType, version string) (err error) {
	commandName := strings.Split(string(installCommand), "/")[len(strings.Split(string(installCommand), "/"))-1]
	log.WithContext(ctx).Infof("Installing %s executable...", commandName)

	downloadURL, execArchiveName, err := config.Configurer.GetDownloadURL(version, installCommand)
	if err != nil {
		return errors.Errorf("failed to get download url, err: %s", err)
	}

	if err = installExecutable(ctx, config, downloadURL.String(), execArchiveName, installCommand); err != nil {
		log.WithContext(ctx).Errorf("Install %s executable failed", commandName)
		return err
	}

	log.WithContext(ctx).Infof("%s executable installed successfully", commandName)

	if installCommand == constants.PastelD {
		if err = InitCommandLogic(ctx, config); err != nil {
			log.WithContext(ctx).Error("Initialize the node")
			return err
		}
	}

	return nil
}

func uncompressNodeArchive(ctx context.Context, dstFolder string, archiveFile string) error {
	file, err := os.Open(archiveFile)
	if err != nil {
		log.WithContext(ctx).Error("Not found archive file!!!")
		return err
	}
	defer file.Close()

	_, err = utils.Unzip(archiveFile, dstFolder)

	if err != nil {
		log.WithContext(ctx).Error("Extracting pastel executables Error!!!")
		return err
	}

	return nil
}

func uncompressArchive(ctx context.Context, dstFolder string, archiveFile string) error {
	file, err := os.Open(archiveFile)

	if err != nil {
		log.WithContext(ctx).Error("Not found archive file!!!")
		return err
	}
	defer file.Close()

	_, err = utils.Unzip(archiveFile, dstFolder)

	if err != nil {
		return err
	}
	return nil
}

// InitializeFunc - Initialize the function
func InitializeFunc(ctx context.Context, config *configs.Config) (err error) {
	configJSON, err := config.String()
	if err != nil {
		return err
	}

	if err = config.SaveConfig(); err != nil {
		return err
	}

	log.WithContext(ctx).Infof("Config: %s", configJSON)

	return nil
}

// openPort opens port
func openPort(ctx context.Context, portList []string) (err error) {
	var out string
	for k := range portList {
		log.WithContext(ctx).Infof("Opening port: %s", portList[k])

		switch utils.GetOS() {
		case constants.Linux:
			out, err = RunCMD("sudo", "ufw", "allow", portList[k])
		case constants.Windows:
			out, err = RunCMD("netsh", "advfirewall", "firewall", "add", "rule", "name=TCP Port "+portList[k], "dir=in", "action=allow", "protocol=TCP", "localport="+portList[k])
		case constants.Mac:
			out, err = RunCMD("sudo", "ipfw", "allow", "tcp", "from", "any", "to", "any", "dst-port", portList[k])
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

func installPackages(ctx context.Context) (err error) {
	if utils.GetOS() == constants.Linux {
		installedCmd := utils.GetInstalledCommand(ctx)
		var notInstall []string
		for _, p := range constants.DependenciesPackages {
			if _, ok := installedCmd[p]; !ok {
				notInstall = append(notInstall, p)
			}
		}

		if len(notInstall) > 0 {
			return errors.New(strings.Join(notInstall, ", ") + " is missing from your OS, which is required for running, please install them")
		}
	}
	return nil
}

func installExecutable(ctx context.Context, config *configs.Config, downloadURL string, archiveName string, toolType constants.ToolType) (err error) {
	err = utils.DownloadFile(ctx,
		filepath.Join(config.PastelExecDir, archiveName),
		downloadURL)

	if err != nil {
		log.WithContext(ctx).Errorf(fmt.Sprintf("Failed to download pastel executable file : %s", downloadURL))
		return err
	}

	log.WithContext(ctx).Info("Installing...")

	log.WithContext(ctx).Debug("Extracting archive files")

	switch toolType {
	case constants.PastelD:
		err = uncompressNodeArchive(ctx, config.PastelExecDir, filepath.Join(config.PastelExecDir, archiveName))
		if err == nil {
			if utils.GetOS() == constants.Linux {
				if _, err = RunCMD("chmod", "777",
					filepath.Join(config.PastelExecDir, constants.PasteldName[utils.GetOS()])); err != nil {
					log.WithContext(ctx).Error("Failed to make pasteld as executable")
					return err
				}
				if _, err = RunCMD("chmod", "777",
					filepath.Join(config.PastelExecDir, constants.PastelCliName[utils.GetOS()])); err != nil {
					log.WithContext(ctx).Error("Failed to make pastel-cli as executable")
					return err
				}
			}
		}
	case constants.WalletNode:
		err = uncompressArchive(ctx, config.PastelExecDir, filepath.Join(config.PastelExecDir, archiveName))
		if err == nil {
			if utils.GetOS() == constants.Linux {
				if _, err = RunCMD("chmod", "777",
					filepath.Join(config.PastelExecDir, constants.WalletNodeExecName[utils.GetOS()])); err != nil {
					log.WithContext(ctx).Error("Failed to make walletnode as executable")
					return err
				}
			}
			log.WithContext(ctx).Info("Initialize the walletnode")

			workDirPath := filepath.Join(config.WorkingDir, "walletnode")

			if err := utils.CreateFolder(ctx, workDirPath, config.Force); err != nil {
				return err
			}

			fileName, err := utils.CreateFile(ctx, filepath.Join(workDirPath, "wallet.yml"), config.Force)
			if err != nil {
				return err
			}

			if err = utils.WriteFile(fileName, configs.WalletMainNetConfig); err != nil {
				return err
			}
		}
	case constants.SuperNode:
		err = uncompressArchive(ctx, config.PastelExecDir, filepath.Join(config.PastelExecDir, archiveName))
		if err == nil {
			if utils.GetOS() == constants.Linux {
				if _, err = RunCMD("chmod", "777",
					filepath.Join(config.PastelExecDir, constants.SuperNodeExecName[utils.GetOS()])); err != nil {
					log.WithContext(ctx).Error("Failed to make supernode as executable")
					return err
				}
			}

			log.WithContext(ctx).Info("Initialize the supernode")
			workDirPath := filepath.Join(config.WorkingDir, "supernode")
			if err := utils.CreateFolder(ctx, workDirPath, config.Force); err != nil {
				return err
			}

			fileName, err := utils.CreateFile(ctx, filepath.Join(workDirPath, "supernode.yml"), config.Force)
			if err != nil {
				return err
			}

			if err = utils.WriteFile(fileName, fmt.Sprintf(configs.SupernodeDefaultConfig, "some-value", "127.0.0.1", "4444")); err != nil {
				return err
			}
		}
	case constants.RQService:
		err = uncompressArchive(ctx, config.PastelExecDir, filepath.Join(config.PastelExecDir, archiveName))
		if err == nil {
			if utils.GetOS() == constants.Linux {
				if _, err = RunCMD("chmod", "777",
					filepath.Join(config.PastelExecDir, constants.PastelRQServiceExecName[utils.GetOS()])); err != nil {
					log.WithContext(ctx).Error("Failed to make rqservice as executable")
					return err
				}
			}
			log.WithContext(ctx).Info("Initialize the rqservice")

			workDirPath := filepath.Join(config.WorkingDir, "rqservice")

			if err := utils.CreateFolder(ctx, workDirPath, config.Force); err != nil {
				log.WithContext(ctx).Error("Failed to create rqservice folder")
				return err
			}

			var fileName string
			if fileName, err = utils.CreateFile(ctx, filepath.Join(workDirPath, "rqservice.toml"), config.Force); err != nil {
				log.WithContext(ctx).Error("Failed to create rqservice.toml file")
				return err
			}

			if err = utils.WriteFile(fileName, fmt.Sprintf(configs.RQServiceConfig, "127.0.0.1", "50051")); err != nil {
				log.WithContext(ctx).Error("Failed to write rqservice.toml file")
				return err
			}
		}
	default:
		log.WithContext(ctx).Warn("Please select correct tool type!")
		return nil
	}

	if err != nil {
		log.WithContext(ctx).Errorf("Failed to extract archive file : %s", filepath.Join(config.PastelExecDir, archiveName))
		return err
	}

	log.WithContext(ctx).Debug("Delete archive files")
	if err = utils.DeleteFile(filepath.Join(config.PastelExecDir, archiveName)); err != nil {
		log.WithContext(ctx).Errorf("Failed to delete archive file : %s", filepath.Join(config.PastelExecDir, archiveName))
		return err
	}
	return nil
}

func installDupeDetection(ctx context.Context, config *configs.Config) (err error) {
	subCmd := []string{"-m", "pip", "install"}
	subCmd = append(subCmd, constants.DependenciesDupeDetectionPackages...)

	log.WithContext(ctx).Info("Installing Pip...")
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

	if err = installChrome(ctx, config); err != nil {
		return err
	}

	homeDir := config.Configurer.GetHomeDir()
	homeDir = filepath.Join(homeDir, "pastel_dupe_detection_service")
	var pathList []interface{}
	for index := range constants.DupeDetectionConfigs {
		dupeDetectionDirPath := filepath.Join(homeDir, constants.DupeDetectionConfigs[index])
		if err = utils.CreateFolder(ctx, dupeDetectionDirPath, config.Force); err != nil {
			return err
		}
		pathList = append(pathList, dupeDetectionDirPath)
	}

	targetDir := filepath.Join(homeDir, constants.DupeDetectionSupportFilePath)
	for index := range constants.DupeDetectionSupportDownloadURL {
		if err = utils.DownloadFile(ctx,
			filepath.Join(targetDir, "temp.zip"),
			constants.DupeDetectionSupportDownloadURL[index]); err != nil {
			log.WithContext(ctx).Errorf("Failed to download archive file : %s", constants.DupeDetectionSupportDownloadURL[index])
			return err
		}

		log.WithContext(ctx).Infof("Extracting archive file : %s", filepath.Join(targetDir, "temp.zip"))
		if err = uncompressArchive(ctx, targetDir, filepath.Join(targetDir, "temp.zip")); err != nil {
			log.WithContext(ctx).Errorf("Failed to extract archive file : %s", filepath.Join(targetDir, "temp.zip"))
			return err
		}

		if err = utils.DeleteFile(filepath.Join(targetDir, "temp.zip")); err != nil {
			log.WithContext(ctx).Errorf("Failed to delete archive file : %s", filepath.Join(targetDir, "temp.zip"))
			return err
		}
	}

	targetDir = filepath.Join(homeDir, constants.DupeDetectionSupportFilePath)
	fileName, err := utils.CreateFile(ctx, filepath.Join(targetDir, "config.ini"), config.Force)
	if err != nil {
		return err
	}

	if err = utils.WriteFile(fileName, fmt.Sprintf(configs.DupeDetectionConfig, pathList...)); err != nil {
		return err
	}

	configPath := filepath.Join(targetDir, "config.ini")

	if utils.GetOS() == constants.Linux {
		RunCMDWithInteractive("export", "DUPEDETECTIONCONFIGPATH=%s", configPath)
	}

	log.WithContext(ctx).Info("Installing DupeDetection finished successfully")
	return nil
}
