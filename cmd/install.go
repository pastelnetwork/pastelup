package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
	installNodeSubCommand.CustomHelpTemplate = GetColoredSubCommandHeaders()
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
	installWalletSubCommand.CustomHelpTemplate = GetColoredSubCommandHeaders()
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
	installSuperNodeSubCommand.CustomHelpTemplate = GetColoredSubCommandHeaders()
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
	installSuperNodeRemoteSubCommand.CustomHelpTemplate = GetColoredSubCommandHeaders()
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
		cli.NewFlag("ssh-ip", &sshIP).SetUsage(green("IP address - Required, SSH address of the remote host")),
		cli.NewFlag("ssh-port", &sshPort).SetUsage(green("port - Optional, SSH port of the remote host, default is 22")).SetValue(22),
		cli.NewFlag("ssh-dir", &config.RemotePastelUtilityDir).SetAliases("rpud").
			SetUsage(green("Location where to create pastel-utility directory")),
	}
	installSuperNodeRemoteSubCommand.AddFlags(installSuperNodeRemoteFlags...)

	installSuperNodeSubCommand.AddSubcommands(installSuperNodeRemoteSubCommand)

	installCommand.AddSubcommands(installNodeSubCommand)
	installCommand.AddSubcommands(installWalletSubCommand)
	installCommand.AddSubcommands(installSuperNodeSubCommand)

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

	err = utils.DownloadFile(ctx, fmt.Sprintf("%s/%s", config.PastelExecDir, constants.PastelExecArchiveName[utils.GetOS()]), constants.PastelDownloadURL[utils.GetOS()])
	if err != nil {
		log.WithContext(ctx).Error("Failed to download pastel executables.")
		return err
	}

	log.WithContext(ctx).Info("Installing...")

	log.WithContext(ctx).Debug("Extract archive files")
	if err = uncompressNodeArchive(ctx, config.PastelExecDir, fmt.Sprintf("%s/%s", config.PastelExecDir, constants.PastelExecArchiveName[utils.GetOS()])); err != nil {
		log.WithContext(ctx).Error("Failed to extract archive files")
		return err
	}

	log.WithContext(ctx).Debug("Delete archive files")
	if err = utils.DeleteFile(fmt.Sprintf("%s/%s", config.PastelExecDir, constants.PastelExecArchiveName[utils.GetOS()])); err != nil {
		log.WithContext(ctx).Error("Failed to delete archive files")
		return err
	}

	if err = InitCommandLogic(ctx, config); err != nil {
		log.WithContext(ctx).Error("Initialize the node")
		return err
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

	err = utils.DownloadFile(ctx, filepath.Join(config.PastelExecDir, constants.PastelExecArchiveName[utils.GetOS()]), constants.PastelDownloadURL[utils.GetOS()])
	if err != nil {
		log.WithContext(ctx).Error("Failed to download pastel executables.")
		return err
	}

	log.WithContext(ctx).Info("Installing...")

	log.WithContext(ctx).Debug("Extract archive files")
	if err = uncompressNodeArchive(ctx, config.PastelExecDir, filepath.Join(config.PastelExecDir, constants.PastelExecArchiveName[utils.GetOS()])); err != nil {
		log.WithContext(ctx).Error("Failed to extract archive files")
		return err
	}

	log.WithContext(ctx).Debug("Delete archive files")
	if err = utils.DeleteFile(filepath.Join(config.PastelExecDir, constants.PastelExecArchiveName[utils.GetOS()])); err != nil {
		log.WithContext(ctx).Error("Failed to delete archive files")
		return err
	}

	if err = InitCommandLogic(ctx, config); err != nil {
		log.WithContext(ctx).Error("Initialize the node")
		return err
	}

	err = utils.DownloadFile(ctx,
		fmt.Sprintf("%s/%s", config.PastelExecDir, constants.PastelWalletExecName[utils.GetOS()]),
		constants.PastelWalletDownloadURL[utils.GetOS()])
	if err != nil {
		log.WithContext(ctx).Error("Failed to download pastel executables.")
		return err
	}

	log.WithContext(ctx).Info(fmt.Sprintf("Wallet dir path -> %s", filepath.Join(config.PastelExecDir, constants.PastelWalletExecName[utils.GetOS()])))
	if _, err = RunCMD("chmod", "777",
		fmt.Sprintf("%s/%s", config.PastelExecDir, constants.PastelWalletExecName[utils.GetOS()])); err != nil {
		log.WithContext(ctx).Error("Failed to make wallet node as executable")
		return err
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

	log.WithContext(ctx).Info("Wallet node install was finished successfully")
	return nil
}

func runInstallSuperNodeRemoteSubCommand(ctx context.Context, config *configs.Config) (err error) {
	log.WithContext(ctx).Info("Start install supernode on remote")
	defer log.WithContext(ctx).Info("End install supernode on remote")
	if err = InitializeFunc(ctx, config); err != nil {
		return err
	}

	if len(sshIP) == 0 {
		return fmt.Errorf("--ssh-ip IP address - Required, SSH address of the remote host")
	}

	//username, password, _ := credentials()
	username := "root"
	password := "vST5utLHaX7feEmh"

	log.WithContext(ctx).Info(fmt.Sprintf("Connecting to SSH Hot node wallet -> %s:%d...", flagMasterNodeSSHIP, flagMasterNodeSSHPort))
	client, err := utils.DialWithPasswd(fmt.Sprintf("%s:%d", sshIP, sshPort), username, password)
	if err != nil {
		return err
	}
	defer client.Close()

	log.WithContext(ctx).Info("Connected successfully")

	_, err = client.Cmd(fmt.Sprintf("wget -P %s https://github.com/pastelnetwork/pastel-utility/releases/download/v0.5.1/pastel-utility-linux-amd64", config.RemotePastelUtilityDir)).Output()
	if err != nil {
		return err
	}

	_, err = client.Cmd(fmt.Sprintf("chmod 777 %s", filepath.Join(config.RemotePastelUtilityDir, "pastel-utility-linux-amd64"))).Output()
	if err != nil {
		return err
	}

	// client.Terminal(nil).Start()
	if len(config.RemotePastelExecDir) > 0 && len(config.RemoteWorkingDir) > 0 {
		_, err = client.Cmd(fmt.Sprintf("%s install supernode --dir=%s –work-dir=%s --force", filepath.Join(config.RemotePastelUtilityDir, "pastel-utility-linux-amd64"), config.RemotePastelExecDir, config.RemoteWorkingDir)).Output()
		if err != nil {
			return err
		}

	} else if len(config.RemotePastelExecDir) > 0 && len(config.RemoteWorkingDir) == 0 {
		_, err = client.Cmd(fmt.Sprintf("%s install supernode --dir=%s --force", filepath.Join(config.RemotePastelUtilityDir, "pastel-utility-linux-amd64"), config.RemotePastelExecDir)).Output()
		if err != nil {
			return err
		}
	} else if len(config.RemoteWorkingDir) > 0 && len(config.RemotePastelExecDir) == 0 {
		_, err = client.Cmd(fmt.Sprintf("%s install supernode –work-dir=%s --force", filepath.Join(config.RemotePastelUtilityDir, "pastel-utility-linux-amd64"), config.RemoteWorkingDir)).Output()
		if err != nil {
			return err
		}
	} else {
		_, err = client.Cmd(fmt.Sprintf("%s install supernode --force", filepath.Join(config.RemotePastelUtilityDir, "pastel-utility-linux-amd64"))).Output()
		if err != nil {
			return err
		}
	}

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

	err = utils.DownloadFile(ctx, filepath.Join(config.PastelExecDir, constants.PastelExecArchiveName[utils.GetOS()]), constants.PastelDownloadURL[utils.GetOS()])
	if err != nil {
		log.WithContext(ctx).Error("Failed to download pastel executables.")
		return err
	}

	log.WithContext(ctx).Info("Installing...")

	log.WithContext(ctx).Debug("Extract archive files")
	if err = uncompressNodeArchive(ctx, config.PastelExecDir, filepath.Join(config.PastelExecDir, constants.PastelExecArchiveName[utils.GetOS()])); err != nil {
		log.WithContext(ctx).Error("Failed to extract archive files")
		return err
	}

	log.WithContext(ctx).Debug("Delete archive files")
	if err = utils.DeleteFile(filepath.Join(config.PastelExecDir, constants.PastelExecArchiveName[utils.GetOS()])); err != nil {
		log.WithContext(ctx).Error("Failed to delete archive files")
		return err
	}

	if err = InitCommandLogic(ctx, config); err != nil {
		log.WithContext(ctx).Error("Initialize the node")
		return err
	}

	err = utils.DownloadFile(ctx,
		fmt.Sprintf("%s/%s", config.PastelExecDir, constants.PastelSuperNodeExecName[utils.GetOS()]),
		constants.PastelSuperNodeDownloadURL[utils.GetOS()])

	if err != nil {
		log.WithContext(ctx).Error("Failed to download pastel executables.")
		return err
	}

	log.WithContext(ctx).Info("Installing...")

	log.WithContext(ctx).Info(fmt.Sprintf("Supernode dir path -> %s/%s", config.PastelExecDir, constants.PastelSuperNodeExecName[utils.GetOS()]))
	if _, err = RunCMD("chmod", "777",
		fmt.Sprintf("%s/%s", config.PastelExecDir, constants.PastelSuperNodeExecName[utils.GetOS()])); err != nil {
		log.WithContext(ctx).Error("Failed to make wallet node as executable")
		return err
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

	if err = utils.WriteFile(fileName, fmt.Sprintf(configs.SupernodeDefaultConfig, "some-value", "127.0.0.1", "4444")); err != nil {
		return err
	}

	if fileName, err = utils.CreateFile(ctx, filepath.Join(config.WorkingDir, "rqservice"), config.Force); err != nil {
		return err
	}

	if err = utils.WriteFile(fileName, fmt.Sprintf(configs.RQServiceConfig, "127.0.0.1", "50051")); err != nil {
		return err
	}

	tfmodelsPath := filepath.Join(workDirPath, "tfmodels")
	// create working dir path
	if err := utils.CreateFolder(ctx, tfmodelsPath, config.Force); err != nil {
		return err
	}

	savedModelURL := "https://drive.google.com/u/0/uc?export=download&confirm=Lq3g&id=1U6tpIpZBxqxIyFej2EeQ-SbLcO_lVNfu"

	log.WithContext(ctx).Infof("Downloading: %s ...\n", savedModelURL)

	err = utils.DownloadFile(ctx, filepath.Join(workDirPath, "SavedMLModels.zip"), savedModelURL)
	err = utils.DownloadFile(ctx, filepath.Join(workDirPath, "SavedMLModels.zip"), savedModelURL)
	if err != nil {
		_, err = RunCMD("pip3", "install", "gdown")
		if err != nil {
			return err
		}
		_, err = RunCMD("gdown", savedModelURL)
		if err != nil {
			return err
		}

		_, err = RunCMD("unzip", "./SavedMLModels.zip", "-d", tfmodelsPath)
		if err != nil {
			return err
		}
	} else {
		_, err = RunCMD("unzip", filepath.Join(workDirPath, "SavedMLModels.zip"), "-d", tfmodelsPath)
		if err != nil {
			return err
		}
	}

	log.WithContext(ctx).Info("Supernode install was finished successfully")
	return nil
}

func initNodeDownloadPath(ctx context.Context, config *configs.Config, nodeInstallPath string) (nodePath string, err error) {
	defer log.WithContext(ctx).Infof("Node install path is %s", nodeInstallPath)

	if err = utils.CreateFolder(ctx, nodeInstallPath, config.Force); err != nil {
		return "", err
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
