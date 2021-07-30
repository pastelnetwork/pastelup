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
)

func setupInstallCommand() *cli.Command {
	config := configs.GetConfig()

	defaultWorkingDir := configurer.DefaultWorkingDir()
	defaultExecutableDir := configurer.DefaultPastelExecutableDir()

	installCommand := cli.NewCommand("install")
	installCommand.SetUsage("usage")

	installNodeSubCommand := cli.NewCommand("node")
	installNodeSubCommand.SetUsage(cyan("Install node"))

	installNodeFlags := []*cli.Flag{
		cli.NewFlag("dir", &config.PastelExecDir).SetAliases("d").
			SetUsage(green("Location where to create pastel node directory")).SetValue(defaultExecutableDir),
		cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
			SetUsage(green("Location where to create working directory")).SetValue(defaultWorkingDir),
		cli.NewFlag("force", &config.Force).SetAliases("f").
			SetUsage(green("Force to overwrite config files and re-download ZKSnark parameters")),
		cli.NewFlag("peers", &config.Peers).SetAliases("p").
			SetUsage(green("List of peers to add into pastel.conf file, must be in the format - \"ip\" or \"ip:port\"")),
		cli.NewFlag("release", &config.Version).SetAliases("r").SetValue("latest"),
	}
	installNodeSubCommand.AddFlags(installNodeFlags...)

	installNodeSubCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "Install Node", config)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		sys.RegisterInterruptHandler(cancel, func() {
			log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
			os.Exit(0)
		})
		return runInstallNodeSubCommand(ctx, config)
	})

	installWalletSubCommand := cli.NewCommand("walletnode")
	installWalletSubCommand.SetUsage(cyan("Install walletnode"))
	installWalletSubCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "Install walletnode", config)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		sys.RegisterInterruptHandler(cancel, func() {
			log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
			os.Exit(0)
		})
		return runInstallWalletSubCommand(ctx, config)
	})

	installWalletFlags := []*cli.Flag{
		cli.NewFlag("dir", &config.PastelExecDir).SetAliases("d").
			SetUsage(green("Location where to create pastel node directory")).SetValue(defaultExecutableDir),
		cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
			SetUsage(green("Location where to create working directory")).SetValue(defaultWorkingDir),
		cli.NewFlag("force", &config.Force).SetAliases("f").
			SetUsage(green("Force to overwrite config files and re-download ZKSnark parameters")),
		cli.NewFlag("peers", &config.Peers).SetAliases("p").
			SetUsage(green("List of peers to add into pastel.conf file, must be in the format - \"ip\" or \"ip:port\"")),
		cli.NewFlag("release", &config.Version).SetAliases("r").SetValue("latest"),
	}
	installWalletSubCommand.AddFlags(installWalletFlags...)

	installSuperNodeSubCommand := cli.NewCommand("supernode")
	installSuperNodeSubCommand.SetUsage(cyan("Install supernode"))
	installSuperNodeSubCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "Install supernode", config)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		sys.RegisterInterruptHandler(cancel, func() {
			log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
			os.Exit(0)
		})

		return runInstallSuperNodeSubCommand(ctx, config)
	})
	installSuperNodeFlags := []*cli.Flag{
		cli.NewFlag("dir", &config.PastelExecDir).SetAliases("d").
			SetUsage(green("Location where to create pastel node directory")).SetValue(defaultExecutableDir),
		cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
			SetUsage(green("Location where to create working directory")).SetValue(defaultWorkingDir),
		cli.NewFlag("force", &config.Force).SetAliases("f").
			SetUsage(green("Force to overwrite config files and re-download ZKSnark parameters")),
		cli.NewFlag("peers", &config.Peers).SetAliases("p").
			SetUsage(green("List of peers to add into pastel.conf file, must be in the format - \"ip\" or \"ip:port\"")),
		cli.NewFlag("release", &config.Version).SetAliases("r").SetValue("latest"),
	}
	installSuperNodeSubCommand.AddFlags(installSuperNodeFlags...)

	installSuperNodeRemoteSubCommand := cli.NewCommand("remote")
	installSuperNodeRemoteSubCommand.SetUsage(cyan("Install supernode remote"))
	installSuperNodeRemoteSubCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "Install supernode remote", config)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		sys.RegisterInterruptHandler(cancel, func() {
			log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
			os.Exit(0)
		})

		return runInstallSuperNodeRemoteSubCommand(ctx, config)
	})
	installSuperNodeRemoteFlags := []*cli.Flag{
		cli.NewFlag("dir", &config.RemotePastelExecDir).SetAliases("d").
			SetUsage(green("Location where to create pastel node directory")),
		cli.NewFlag("work-dir", &config.RemoteWorkingDir).SetAliases("w").
			SetUsage(green("Location where to create working directory")),
		cli.NewFlag("force", &config.Force).SetAliases("f").
			SetUsage(green("Force to overwrite config files and re-download ZKSnark parameters")),
		cli.NewFlag("peers", &config.Peers).SetAliases("p").
			SetUsage(green("List of peers to add into pastel.conf file, must be in the format - \"ip\" or \"ip:port\"")),
		cli.NewFlag("release", &config.Version).SetAliases("r").SetValue("latest"),
		cli.NewFlag("ssh-ip", &sshIP).SetUsage(green("Required,IP address - Required, SSH address of the remote host")).SetRequired(),
		cli.NewFlag("ssh-port", &sshPort).SetUsage(green("port - Optional, SSH port of the remote host, default is 22")).SetValue(22),
		cli.NewFlag("ssh-dir", &config.RemotePastelUtilityDir).SetAliases("rpud").
			SetUsage(green("Required, Location where to create pastel-utility directory")).SetRequired(),
	}
	installSuperNodeRemoteSubCommand.AddFlags(installSuperNodeRemoteFlags...)

	installSuperNodeSubCommand.AddSubcommands(installSuperNodeRemoteSubCommand)

	installCommand.AddSubcommands(installNodeSubCommand)
	installCommand.AddSubcommands(installWalletSubCommand)
	installCommand.AddSubcommands(installSuperNodeSubCommand)

	installFlags := []*cli.Flag{
		cli.NewFlag("dir", &config.RemotePastelExecDir).SetAliases("d").
			SetUsage(green("Optional, location where to create pastel node directory")),
		cli.NewFlag("work-dir", &config.RemoteWorkingDir).SetAliases("w").
			SetUsage(green("Optional, location where to create working directory")),
		cli.NewFlag("force", &config.Force).SetAliases("f").
			SetUsage(green("Optional, force to overwrite config files and re-download ZKSnark parameters")),
		cli.NewFlag("peers", &config.Peers).SetAliases("p").
			SetUsage(green("Optional, list of peers to add into pastel.conf file, must be in the format - \"ip\" or \"ip:port\"")),
		cli.NewFlag("release", &config.Version).
			SetUsage(green("Optional, release version to install")).SetAliases("r").SetValue("latest"),
		cli.NewFlag("ssh-ip", &sshIP).SetUsage(yellow("Supernode specific : IP address - Required , SSH address of the remote host")).SetRequired(),
		cli.NewFlag("ssh-port", &sshPort).SetUsage(yellow("Supernode specific : Optional, SSH port of the remote host, default is 22")).SetValue(22),
		cli.NewFlag("ssh-dir", &config.RemotePastelUtilityDir).SetAliases("rpud").
			SetUsage(yellow("Supernode specific : Required, Location where to create pastel-utility directory")).SetRequired(),
	}
	installCommand.AddFlags(installFlags...)

	installCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "installcommand", config)
		if err != nil {
			return err
		}

		return runInstall(ctx, config)
	})
	return installCommand
}

func runInstall(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Install")
	defer log.WithContext(ctx).Info("End")

	configJSON, err := config.String()
	if err != nil {
		return err
	}
	log.WithContext(ctx).Infof("Config: %s", configJSON)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sys.RegisterInterruptHandler(cancel, func() {
		log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
		os.Exit(0)
	})

	// actions to run goes here

	return nil

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

func runInstallSuperNodeRemoteSubCommand(ctx context.Context, config *configs.Config) (err error) {
	if len(sshIP) == 0 {
		return fmt.Errorf("--ssh-ip IP address - Required, SSH address of the remote host")
	}

	if len(config.RemotePastelUtilityDir) == 0 {
		return fmt.Errorf("--ssh-dir RemotePastelUtilityDir - Required, pastel-utility path of the remote host")
	}

	log.WithContext(ctx).Info("Start install supernode on remote")
	defer log.WithContext(ctx).Info("End install supernode on remote")
	if err = InitializeFunc(ctx, config); err != nil {
		return err
	}

	username, password, _ := credentials()

	log.WithContext(ctx).Infof("Connecting to SSH Hot node wallet -> %s:%d...", flagMasterNodeSSHIP, flagMasterNodeSSHPort)
	client, err := utils.DialWithPasswd(fmt.Sprintf("%s:%d", sshIP, sshPort), username, password)
	if err != nil {
		return err
	}
	defer client.Close()

	log.WithContext(ctx).Info("Connected successfully")

	pastelUtilityPath := filepath.Join(config.RemotePastelUtilityDir, "pastel-utility-linux-amd64")
	pastelUtilityPath = strings.ReplaceAll(pastelUtilityPath, "\\", "/")

	_, err = client.Cmd(fmt.Sprintf("rm -r -f %s", pastelUtilityPath)).Output()
	if err != nil {
		fmt.Println("rm Err")
		fmt.Println(err.Error())
		return err
	}
	log.WithContext(ctx).Info("Downloading Pastel-Utility Executable...")
	_, err = client.Cmd(fmt.Sprintf("wget -P %s https://github.com/pastelnetwork/pastel-utility/releases/download/v0.5.5/pastel-utility-linux-amd64", config.RemotePastelUtilityDir)).Output()
	if err != nil {
		fmt.Println("download Err")
		fmt.Println(err.Error())
		return err
	}
	log.WithContext(ctx).Info("Finished Downloading Pastel-Utility Successfully")

	log.WithContext(ctx).Info(fmt.Sprintf("Downloading  %s...", pastelUtilityPath))

	_, err = client.Cmd(fmt.Sprintf("chmod 777 %s", pastelUtilityPath)).Output()
	if err != nil {
		fmt.Println("chmod Err")
		fmt.Println(err.Error())
		return err
	}

	log.WithContext(ctx).Info("Installing Supernode ...")

	fmt.Println(pastelUtilityPath)
	if len(config.RemotePastelExecDir) > 0 && len(config.RemoteWorkingDir) > 0 {
		_, err = client.Cmd(fmt.Sprintf("%s install supernode --dir=%s –work-dir=%s --force --peers=%s", pastelUtilityPath, config.RemotePastelExecDir, config.RemoteWorkingDir, config.Peers)).Output()
		if err != nil {
			fmt.Println("install supernode Err1")
			fmt.Println(err.Error())
			return err
		}

	} else if len(config.RemotePastelExecDir) > 0 && len(config.RemoteWorkingDir) == 0 {
		_, err = client.Cmd(fmt.Sprintf("%s install supernode --dir=%s --force --peers=%s", pastelUtilityPath, config.RemotePastelExecDir, config.Peers)).Output()
		if err != nil {
			fmt.Println("install supernode Err2")
			fmt.Println(err.Error())
			return err
		}
	} else if len(config.RemoteWorkingDir) > 0 && len(config.RemotePastelExecDir) == 0 {
		_, err = client.Cmd(fmt.Sprintf("%s install supernode –work-dir=%s --force --peers=%s", pastelUtilityPath, config.RemoteWorkingDir, config.Peers)).Output()
		if err != nil {
			fmt.Println("install supernode Err3")
			fmt.Println(err.Error())
			return err
		}
	} else {
		_, err = client.Cmd(fmt.Sprintf("%s install supernode --force --peers=%s", pastelUtilityPath, config.Peers)).Output()
		if err != nil {
			fmt.Println("install supernode Err4")
			fmt.Println(err.Error())
			return err
		}
	}

	log.WithContext(ctx).Info("Finished Install Supernode successfully")

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

	if utils.GetOS() == constants.Linux {
		err = installChrome(ctx, config)
		if err != nil {
			return err
		}
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

	log.WithContext(ctx).Info("Supernode install was finished successfully")

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
