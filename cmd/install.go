package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/pastel-utility/configurer"
	"github.com/pastelnetwork/pastel-utility/constants"
	"github.com/pastelnetwork/pastel-utility/utils"
)

var (
	sshIP   string
	sshPort int
	sshKey string
)

type InstallCommand uint8
const (
	nodeInstall InstallCommand = iota
	walletInstall
	superNodeInstall
	remoteInstall
	highLevel
)

func setupSubCommand(config *configs.Config,
					 installCommand InstallCommand,
					 f func(context.Context, *configs.Config) error,
					) *cli.Command {


	defaultWorkingDir := configurer.DefaultWorkingDir()
	defaultExecutableDir := configurer.DefaultPastelExecutableDir()

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
				SetUsage(green("Optional, Location where to create pastel node directory")).SetValue(defaultExecutableDir),
			cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
				SetUsage(green("Optional, Location where to create working directory")).SetValue(defaultWorkingDir),
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

	var commandName, commandMessage string
	var commandFlags []*cli.Flag

	switch installCommand {
	case nodeInstall:
		{
			commandFlags = append(dirsFlags, commonFlags[:]...)
			commandName = "node"
			commandMessage = "Install node"
		}
	case walletInstall:
		{
			commandFlags = append(dirsFlags, commonFlags[:]...)
			commandName = "walletnode"
			commandMessage = "Install walletnode"
		}
	case superNodeInstall:
		{
			commandFlags = append(dirsFlags, commonFlags[:]...)
			commandName = "supernode"
			commandMessage = "Install supernode"
		}
	case remoteInstall:
		{
			commandFlags = append(append(dirsFlags, commonFlags[:]...), remoteFlags[:]...)
			commandName = "remote"
			commandMessage = "Install supernode remote"
		}
	default:
		{
			commandFlags = append(append(dirsFlags, commonFlags[:]...), remoteFlags[:]...)
		}
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
				if utils.GetOS() == constants.Linux {
					installPackages()
				}
			}

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			sys.RegisterInterruptHandler(cancel, func() {
				log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
				os.Exit(0)
			})
			return f(ctx, config)
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

	installCommand := cli.NewCommand("install")
	installCommand.SetUsage("Performs installation and initialization of the system for both WalletNode and SuperNodes")
	installCommand.AddSubcommands(installNodeSubCommand)
	installCommand.AddSubcommands(installWalletSubCommand)
	installCommand.AddSubcommands(installSuperNodeSubCommand)
	//installCommand := setupSubCommand(config, highLevel, nil)

	return installCommand
}

func runInstallNodeSubCommand(ctx context.Context, config *configs.Config) (err error) {

	log.WithContext(ctx).Info("Start install node")
	defer log.WithContext(ctx).Info("End install node")

	if err = InitializeFunc(ctx, config); err != nil {
		return err
	}

	if _, err = initNodeDownloadPath(ctx, config, config.PastelExecDir); err != nil {
		return err
	}

	var PastelExecArchiveName string
	var PastelDownloadURL string
	if config.Version == "latest" {
		PastelExecArchiveName = constants.PastelExecArchiveName[utils.GetOS()]
		PastelDownloadURL = constants.PastelDownloadURL[utils.GetOS()]
	} else {
		rcVersion := strings.Split(config.Version, "-")[1] // get rc version
		PastelExecArchiveName = fmt.Sprintf("%s%s%s", constants.PastelDownloadReleaseFileName[utils.GetOS()], rcVersion, constants.PastelDownloadReleaseFileExtension[utils.GetOS()])
		PastelDownloadURL = fmt.Sprintf("%s%s/%s", constants.PastelDownloadReleaseURL, config.Version, PastelExecArchiveName)
	}

	err = utils.DownloadFile(ctx,
		filepath.Join(config.PastelExecDir, PastelExecArchiveName),
		PastelDownloadURL)

	if err != nil {
		log.WithContext(ctx).Error("Failed to download pastel executables.")
		return err
	}

	log.WithContext(ctx).Info("Installing...")

	log.WithContext(ctx).Debug("Extract archive files")
	if err = uncompressNodeArchive(ctx, config.PastelExecDir, filepath.Join(config.PastelExecDir, PastelExecArchiveName)); err != nil {
		log.WithContext(ctx).Error("Failed to extract archive files")
		return err
	}

	log.WithContext(ctx).Debug("Delete archive files")
	if err = utils.DeleteFile(filepath.Join(config.PastelExecDir, PastelExecArchiveName)); err != nil {
		log.WithContext(ctx).Error("Failed to delete archive files")
		return err
	}

	if err = InitCommandLogic(ctx, config); err != nil {
		log.WithContext(ctx).Error("Initialize the node")
		return err
	}

	if utils.GetOS() == constants.Linux {
		err = installChrome(ctx, config)
		if err != nil {
			return err
		}
	}

	log.WithContext(ctx).Info("Node install was finished successfully")

	return nil
}

func runInstallWalletSubCommand(ctx context.Context, config *configs.Config) (err error) {
	log.WithContext(ctx).Info("Start install walletnode")
	defer log.WithContext(ctx).Info("End install walletnode")

	if err = InitializeFunc(ctx, config); err != nil {
		return err
	}

	if _, err = initNodeDownloadPath(ctx, config, config.PastelExecDir); err != nil {
		return err
	}


	var PastelWalletDonwloadURL string

	PastelExecArchiveName := constants.PastelExecArchiveName[utils.GetOS()]
	WalletExecArchiveName := constants.WalletNodeExecArchiveName[utils.GetOS()]
	PastelDownloadURL := constants.PastelDownloadURL[utils.GetOS()]
	if config.Version == "latest" {
		PastelWalletDonwloadURL = constants.PastelWalletDownloadURL[utils.GetOS()]
	} else {
		PastelWalletDonwloadURL = fmt.Sprintf("%s%s/%s", constants.PastelWalletSuperReleaseDownloadURL, config.Version, constants.WalletNodeExecArchiveName[utils.GetOS()])
	}

	err = utils.DownloadFile(ctx,
		filepath.Join(config.PastelExecDir, PastelExecArchiveName),
		PastelDownloadURL)
	if err != nil {
		log.WithContext(ctx).Error("Failed to download pastel executables.")
		return err
	}

	log.WithContext(ctx).Info("Installing...")

	log.WithContext(ctx).Debug("Extract archive files")
	if err = uncompressNodeArchive(ctx, config.PastelExecDir, filepath.Join(config.PastelExecDir, PastelExecArchiveName)); err != nil {
		log.WithContext(ctx).Error("Failed to extract archive files")
		return err
	}

	log.WithContext(ctx).Debug("Delete archive files")
	if err = utils.DeleteFile(filepath.Join(config.PastelExecDir, PastelExecArchiveName)); err != nil {
		log.WithContext(ctx).Error("Failed to delete archive files")
		return err
	}

	if err = InitCommandLogic(ctx, config); err != nil {
		log.WithContext(ctx).Error("Initialize the node")
		return err
	}

	err = utils.DownloadFile(ctx,
		fmt.Sprintf("%s/%s", config.PastelExecDir, constants.WalletNodeExecArchiveName[utils.GetOS()]),
		PastelWalletDonwloadURL)

	if err != nil {
		log.WithContext(ctx).Error("Failed to download wallet executable archive.")
		return err
	}

	log.WithContext(ctx).Debug("Extracting wallet archive file")
	if err = uncompressArchive(ctx, config.PastelExecDir, filepath.Join(config.PastelExecDir, WalletExecArchiveName), "wallet"); err != nil {
		log.WithContext(ctx).Error("Failed to extract archive files")
		return err
	}

	log.WithContext(ctx).Debug("Delete archive files")
	if err = utils.DeleteFile(filepath.Join(config.PastelExecDir, WalletExecArchiveName)); err != nil {
		log.WithContext(ctx).Error("Failed to delete archive files")
		return err
	}

	log.WithContext(ctx).Infof("Wallet dir path -> %s", filepath.Join(config.PastelExecDir, constants.PastelWalletExecName[utils.GetOS()]))
	if utils.GetOS() == constants.Linux {
		if _, err = RunCMD("chmod", "777",
			fmt.Sprintf("%s/%s", config.PastelExecDir, constants.PastelWalletExecName[utils.GetOS()])); err != nil {
			log.WithContext(ctx).Error("Failed to make wallet node as executable")
			return err
		}
	}

	log.WithContext(ctx).Info("Initialize the walletnode")

	workDirPath := filepath.Join(config.WorkingDir, "walletnode")

	// create working dir path
	if err := utils.CreateFolder(ctx, workDirPath, config.Force); err != nil {
		return err
	}

	// create walletnode default config
	// create file
	fileName, err := utils.CreateFile(ctx, filepath.Join(workDirPath, "wallet.yml"), config.Force)
	if err != nil {
		return err
	}

	if err = utils.WriteFile(fileName, configs.WalletMainNetConfig); err != nil {
		return err
	}

	if fileName, err = utils.CreateFile(ctx, filepath.Join(config.WorkingDir, "rqservice"), config.Force); err != nil {
		return err
	}

	if err = utils.WriteFile(fileName, fmt.Sprintf(configs.RQServiceConfig, "127.0.0.1", "50051")); err != nil {
		return err
	}

	if utils.GetOS() == constants.Linux {
		err = installChrome(ctx, config)
		if err != nil {
			return err
		}
	}

	log.WithContext(ctx).Info("Wallet node install was finished successfully")

	return nil
}

func runInstallSuperNodeSubCommand(ctx context.Context, config *configs.Config) (err error) {

	log.WithContext(ctx).Info("Start install supernode")
	defer log.WithContext(ctx).Info("End install supernode")
	if err = InitializeFunc(ctx, config); err != nil {
		return err
	}

	if _, err = initNodeDownloadPath(ctx, config, config.PastelExecDir); err != nil {
		return err
	}

	var PastelSuperDownloadURL string

	PastelExecArchiveName := constants.PastelExecArchiveName[utils.GetOS()]
	SuperExecArchiveName := constants.SupperNodeExecArchiveName[utils.GetOS()]
	PastelDownloadURL := constants.PastelDownloadURL[utils.GetOS()]

	if config.Version == "latest" {
		PastelSuperDownloadURL = constants.PastelSuperNodeDownloadURL[utils.GetOS()]
	} else {
		PastelSuperDownloadURL = fmt.Sprintf("%s%s/%s", constants.PastelWalletSuperReleaseDownloadURL, config.Version, constants.SupperNodeExecArchiveName[utils.GetOS()])
	}

	err = utils.DownloadFile(ctx,
		filepath.Join(config.PastelExecDir, PastelExecArchiveName),
		PastelDownloadURL)
	if err != nil {
		log.WithContext(ctx).Error("Failed to download pastel executables.")
		return err
	}

	log.WithContext(ctx).Info("Installing...")

	log.WithContext(ctx).Debug("Extracting archive files...")
	if err = uncompressNodeArchive(ctx, config.PastelExecDir, filepath.Join(config.PastelExecDir, PastelExecArchiveName)); err != nil {
		log.WithContext(ctx).Error("Failed to extract archive files")
		return err
	}

	log.WithContext(ctx).Debug("Deleting archive files...")
	if err = utils.DeleteFile(filepath.Join(config.PastelExecDir, PastelExecArchiveName)); err != nil {
		log.WithContext(ctx).Error("Failed to delete archive files")
		return err
	}

	if err = InitCommandLogic(ctx, config); err != nil {
		log.WithContext(ctx).Error("Initialize the node")
		return err
	}

	err = utils.DownloadFile(ctx,
		fmt.Sprintf("%s/%s", config.PastelExecDir, SuperExecArchiveName),
		PastelSuperDownloadURL)

	if err != nil {
		log.WithContext(ctx).Error("Failed to download pastel executables.")
		return err
	}

	log.WithContext(ctx).Debug("Extracting supernode archive file")
	if err = uncompressArchive(ctx, config.PastelExecDir, filepath.Join(config.PastelExecDir, SuperExecArchiveName), "supernode"); err != nil {
		log.WithContext(ctx).Error("Failed to extract archive files")
		return err
	}

	log.WithContext(ctx).Debug("Delete archive files")
	if err = utils.DeleteFile(filepath.Join(config.PastelExecDir, SuperExecArchiveName)); err != nil {
		log.WithContext(ctx).Error("Failed to delete archive files")
		return err
	}

	log.WithContext(ctx).Infof("Supernode dir path -> %s/%s", config.PastelExecDir, constants.PastelSuperNodeExecName[utils.GetOS()])
	if utils.GetOS() == constants.Linux {
		if _, err = RunCMD("chmod", "777",
			fmt.Sprintf("%s/%s", config.PastelExecDir, constants.PastelSuperNodeExecName[utils.GetOS()])); err != nil {
			log.WithContext(ctx).Error("Failed to make wallet node as executable")
			return err
		}
	}

	log.WithContext(ctx).Info("Initialize the supernode")

	workDirPath := filepath.Join(config.WorkingDir, "supernode")

	// create working dir path
	if err := utils.CreateFolder(ctx, workDirPath, config.Force); err != nil {
		return err
	}

	// create walletnode default config
	// create file
	fileName, err := utils.CreateFile(ctx, filepath.Join(workDirPath, "supernode.yml"), config.Force)
	if err != nil {
		return err
	}

	if err = utils.WriteFile(fileName, fmt.Sprintf(configs.SupernodeDefaultConfig, "some-value", "127.0.0.1", "4444", config.WorkingDir, constants.DupeDetectionImageFingerPrintDataBase)); err != nil {
		return err
	}

	if fileName, err = utils.CreateFile(ctx, filepath.Join(config.WorkingDir, "rqservice"), config.Force); err != nil {
		return err
	}

	if err = utils.WriteFile(fileName, fmt.Sprintf(configs.RQServiceConfig, "127.0.0.1", "50051")); err != nil {
		return err
	}

	openErr := openPort(ctx, constants.PortList)
	if openErr != nil {
		return openErr
	}

	// create dupe-detection dir path
	dupeDetectionDirPath := filepath.Join(config.WorkingDir, "dupe_detection_je")
	if err = utils.CreateFolder(ctx, dupeDetectionDirPath, config.Force); err != nil {
		return err
	}

	// create dupe-detection sub dirs path
	dupeDetectionInputDirPath := filepath.Join(dupeDetectionDirPath, "input")
	if err = utils.CreateFolder(ctx, dupeDetectionInputDirPath, config.Force); err != nil {
		return err
	}
	dupeDetectionOutputDirPath := filepath.Join(dupeDetectionDirPath, "output")
	if err = utils.CreateFolder(ctx, dupeDetectionOutputDirPath, config.Force); err != nil {
		return err
	}
	dupeDetectionRarenessDirPath := filepath.Join(dupeDetectionDirPath, "rareness")
	if err = utils.CreateFolder(ctx, dupeDetectionRarenessDirPath, config.Force); err != nil {
		return err
	}

	requirementURL := "https://download.pastel.network/machine-learning/requirements.txt"
	err = utils.DownloadFile(ctx, filepath.Join(config.WorkingDir, constants.PipRequirmentsFileName), requirementURL)
	if err == nil {
		log.WithContext(ctx).Info("Installing Pip...")
		if utils.GetOS() == constants.Windows {
			RunCMDWithInteractive("python", "-m", "pip", "install", "-r", filepath.Join(config.WorkingDir, constants.PipRequirmentsFileName))
		} else {
			RunCMDWithInteractive("python3", "-m", "pip", "install", "-r", filepath.Join(config.WorkingDir, constants.PipRequirmentsFileName))

		}

		log.WithContext(ctx).Info("Pip install finished")
	} else {
		log.WithContext(ctx).Info("Can not download requirement file to install Pip.")
	}

	if utils.GetOS() == constants.Linux {
		err = installChrome(ctx, config)
		if err != nil {
			return err
		}
	}

	log.WithContext(ctx).Info("Supernode install was finished successfully")

	return nil
}

func runInstallSuperNodeRemoteSubCommand(ctx context.Context, config *configs.Config) (err error) {
	if len(sshIP) == 0 {
		return fmt.Errorf("--ssh-ip IP address - Required, SSH address of the remote host")
	}

	if len(config.RemotePastelUtilityDir) == 0 {
		return fmt.Errorf("--ssh-dir RemotePastelUtilityDir - Required, pastel-utility path of the remote host")
	}

	//if len(config.Peers) == 0 {
	//	return fmt.Errorf("--peers - Required, list of peers to add into pastel.conf file, must be in the format - “ip” or “ip:port")
	//}

	log.WithContext(ctx).Info("Start install supernode on remote")
	defer log.WithContext(ctx).Info("End install supernode on remote")
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

	_, err = client.Cmd(fmt.Sprintf("rm -r -f %s", pastelUtilityPath)).Output()
	if err != nil {
		fmt.Println("rm Err")
		fmt.Println(err.Error())
		return err
	}
	log.WithContext(ctx).Info("Downloading Pastel-Utility Executable...")
	_, err = client.Cmd(fmt.Sprintf("wget -O %s https://github.com/pastelnetwork/pastel-utility/releases/download/v0.5.8/pastel-utility-linux-amd64", pastelUtilityPath)).Output()
	fmt.Printf("wget -O %s  https://github.com/pastelnetwork/pastel-utility/releases/download/v0.5.8/pastel-utility-linux-amd64\n", pastelUtilityPath)
	if err != nil {
		fmt.Println("download Err")
		fmt.Println(err.Error())
		return err
	}
	log.WithContext(ctx).Info("Finished Downloading Pastel-Utility Successfully")

	log.WithContext(ctx).Info(fmt.Sprintf("Downloading  %s...", pastelUtilityPath))

	_, err = client.Cmd(fmt.Sprintf("chmod 777 /%s", pastelUtilityPath)).Output()
	if err != nil {
		fmt.Printf("chmod 777 /%s\n", pastelUtilityPath)
		fmt.Println("chmod Err")
		fmt.Println(err.Error())
		return err
	}

	_, err = client.Cmd(fmt.Sprintf("%s stop supernode ", pastelUtilityPath)).Output()
	if err != nil {
		fmt.Println("Stop supernode Err1")
	}

	log.WithContext(ctx).Info("Installing Supernode ...")

	fmt.Println(pastelUtilityPath)
	if len(config.RemotePastelExecDir) > 0 && len(config.RemoteWorkingDir) > 0 {
		_, err = client.Cmd(fmt.Sprintf("/%s install supernode --dir=%s –work-dir=%s --force --peers=%s", pastelUtilityPath, config.RemotePastelExecDir, config.RemoteWorkingDir, config.Peers)).Output()
		if err != nil {
			fmt.Println("install supernode Err1")
			fmt.Println(err.Error())
			return err
		}

	} else if len(config.RemotePastelExecDir) > 0 && len(config.RemoteWorkingDir) == 0 {
		_, err = client.Cmd(fmt.Sprintf("/%s install supernode --dir=%s --force --peers=%s", pastelUtilityPath, config.RemotePastelExecDir, config.Peers)).Output()
		if err != nil {
			fmt.Println("install supernode Err2")
			fmt.Println(err.Error())
			return err
		}
	} else if len(config.RemoteWorkingDir) > 0 && len(config.RemotePastelExecDir) == 0 {
		_, err = client.Cmd(fmt.Sprintf("/%s install supernode –work-dir=%s --force --peers=%s", pastelUtilityPath, config.RemoteWorkingDir, config.Peers)).Output()
		if err != nil {
			fmt.Println("install supernode Err3")
			fmt.Println(err.Error())
			return err
		}
	} else {
		_, err = client.Cmd(fmt.Sprintf("/%s install supernode --force --peers=%s", pastelUtilityPath, config.Peers)).Output()
		if err != nil {
			fmt.Printf("%s install supernode --force --peers=%s\n", pastelUtilityPath, config.Peers)
			fmt.Println("install supernode Err4")
			fmt.Println(err.Error())
			return err
		}
	}

	if utils.GetOS() == constants.Linux {
		err = installChrome(ctx, config)
		if err != nil {
			return err
		}
	}

	log.WithContext(ctx).Info("Finished Install Supernode successfully")

	return nil
}

func initNodeDownloadPath(ctx context.Context, config *configs.Config, nodeInstallPath string) (nodePath string, err error) {
	defer log.WithContext(ctx).Infof("Node install path is %s", nodeInstallPath)

	if err = utils.CreateFolder(ctx, nodeInstallPath, config.Force); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			reader := bufio.NewReader(os.Stdin)

			fmt.Printf("%s.Do you want continue to install? Y/N\n", err.Error())
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

	}

	return "", nil
}

func uncompressNodeArchive(ctx context.Context, dstFolder string, archiveFile string) error {
	osType := utils.GetOS()

	file, err := os.Open(archiveFile)

	if err != nil {
		log.WithContext(ctx).Error("Not found archive file!!!")
		return err
	}
	defer file.Close()

	var fileReader io.ReadCloser = file

	switch osType {
	case constants.Linux:
		return utils.Untar(dstFolder, fileReader, filepath.Join(dstFolder, constants.PasteldName[utils.GetOS()]), filepath.Join(dstFolder, constants.PastelCliName[utils.GetOS()]))
	case constants.Mac:
		return utils.Untar(dstFolder, fileReader, filepath.Join(dstFolder, constants.PasteldName[utils.GetOS()]), filepath.Join(dstFolder, constants.PastelCliName[utils.GetOS()]))
	case constants.Windows:
		_, err = utils.Unzip(archiveFile, dstFolder, filepath.Join(dstFolder, constants.PasteldName[utils.GetOS()]), filepath.Join(dstFolder, constants.PastelCliName[utils.GetOS()]))
		return err
	default:
		log.WithContext(ctx).Error("Not supported OS!!!")
	}
	return fmt.Errorf("not supported OS")
}

func uncompressArchive(ctx context.Context, dstFolder string, archiveFile string, flag string) error {

	file, err := os.Open(archiveFile)

	if err != nil {
		log.WithContext(ctx).Error("Not found archive file!!!")
		return err
	}
	defer file.Close()

	var execName string
	if flag == "wallet" {
		execName = constants.PastelWalletExecName[utils.GetOS()]
	} else {
		execName = constants.PastelSuperNodeExecName[utils.GetOS()]
	}
	_, err = utils.Unzip(archiveFile, dstFolder, filepath.Join(dstFolder, execName))

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

	for k := range portList {
		fmt.Println("Opening port:", portList[k])
		if utils.GetOS() == constants.Linux {
			out, err := RunCMD("sudo", "ufw", "allow", portList[k])
			log.WithContext(ctx).Info(out)

			if err != nil {
				log.WithContext(ctx).Error(err.Error())
				return err
			}
		}

		if utils.GetOS() == constants.Windows {
			out, err := RunCMD("netsh", "advfirewall", "firewall", "add", "rule", "name=TCP Port "+portList[k], "dir=in", "action=allow", "protocol=TCP", "localport="+portList[k])
			fmt.Println(out)
			if err != nil {
				log.WithContext(ctx).Error("Please Run as administrator to open ports!")
				log.WithContext(ctx).Error(err.Error())
				return err
			}
		}

		if utils.GetOS() == constants.Mac {
			out, err := RunCMD("sudo", "ipfw", "allow", "tcp", "from", "any", "to", "any", "dst-port", portList[k])
			fmt.Println(out)
			if err != nil {
				log.WithContext(ctx).Error(err.Error())
				return err
			}
		}

	}

	return nil

}

func installChrome(ctx context.Context, config *configs.Config) (err error) {

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

	return nil

}

func installPackages() (err error) {

	fmt.Printf("Installing Packages: %s \n", constants.ChromeDownloadURL[utils.GetOS()])

	RunCMDWithInteractive("sudo", "apt-get", "update")
	RunCMDWithInteractive("sudo", "apt-get", "install", "-y", "wget")
	RunCMDWithInteractive("sudo", "apt-get", "-qq", "-y", "install", "curl")
	RunCMDWithInteractive("sudo", "apt-get", "install", "-y", "libgomp1")
	RunCMDWithInteractive("sudo", "apt-get", "install", "-y", "python3-pip")
	RunCMDWithInteractive("sudo", "apt", "install", "--fix-broken")

	return nil

}
