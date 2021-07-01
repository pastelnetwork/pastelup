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
		cli.NewFlag("iPath", &config.PastelNodeDir).
			SetUsage(green("Location where to create pastel node directory")).SetValue(defaultExecutableDir),
		cli.NewFlag("work-dir", &config.WorkingDir).
			SetUsage(green("Location where to create working directory")).SetValue(defaultWorkingDir),
		cli.NewFlag("network", &config.Network).SetAliases("n").
			SetUsage(green("Network type, can be - \"mainnet\" or \"testnet\"")).SetValue("mainnet"),
		cli.NewFlag("force", &config.Force).SetAliases("f").
			SetUsage(green("Force to overwrite config files and re-download ZKSnark parameters")),
		cli.NewFlag("peers", &config.Peers).SetAliases("p").
			SetUsage(green("List of peers to add into pastel.conf file, must be in the format - \"ip\" or \"ip:port\"")),
		cli.NewFlag("r", &flagRestart),
	}
	installNodeSubCommand.AddFlags(installNodeFlags...)

	installNodeSubCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "Install Node", config)
		if err != nil {
			return err
		}
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
		return runInstallWalletSubCommand(ctx, config)
	})
	installWalletFlags := []*cli.Flag{
		cli.NewFlag("iPath", &config.PastelWalletDir).
			SetUsage(green("Location where to create pastel wallet node directory")).SetValue(defaultExecutableDir),
		cli.NewFlag("force", &config.Force).SetAliases("f").
			SetUsage(green("Force to overwrite config files and re-download ZKSnark parameters")),
		cli.NewFlag("r", &flagRestart),
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
		return runInstallSuperNodeSubCommand(ctx, config)
	})
	installSuperNodeFlags := []*cli.Flag{
		cli.NewFlag("iPath", &config.PastelSuperNodeDir).
			SetUsage(green("Location where to create pastel wallet node directory")).SetValue(defaultExecutableDir),
		cli.NewFlag("force", &config.Force).SetAliases("f").
			SetUsage(green("Force to overwrite config files and re-download ZKSnark parameters")),
		cli.NewFlag("r", &flagRestart),
	}
	installSuperNodeSubCommand.AddFlags(installSuperNodeFlags...)

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

func runInstallNodeSubCommand(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info(fmt.Sprintf("Install node on %s", utils.GetOS()))
	defer log.WithContext(ctx).Info("End install node")

	configJSON, err := config.String()
	if err != nil {
		return err
	}

	if err = config.SaveConfig(); err != nil {
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

	if _, err = initNodeDownloadPath(ctx, config, config.PastelNodeDir); err != nil {
		log.WithContext(ctx).Error("Failed to initialize install path!!!")
		return err
	}

	err = utils.DownloadFile(ctx, fmt.Sprintf("%s/%s", config.PastelNodeDir, constants.PastelExecArchiveName[utils.GetOS()]), constants.PastelDownloadURL[utils.GetOS()])
	if err != nil {
		log.WithContext(ctx).Error("Failed to download pastel executables.")
		return err
	}

	log.WithContext(ctx).Info("Download was finished successfully")
	log.WithContext(ctx).Info("Installing...")

	log.WithContext(ctx).Debug("Extract archive files")
	if err = uncompressNodeArchive(ctx, config.PastelNodeDir, fmt.Sprintf("%s/%s", config.PastelNodeDir, constants.PastelExecArchiveName[utils.GetOS()])); err != nil {
		log.WithContext(ctx).Error("Failed to extract archive files")
		return err
	}
	log.WithContext(ctx).Debug("Delete archive files")
	if err = utils.DeleteFile(fmt.Sprintf("%s/%s", config.PastelNodeDir, constants.PastelExecArchiveName[utils.GetOS()])); err != nil {
		log.WithContext(ctx).Error("Failed to delete archive files")
		return err
	}

	// start node initialize
	log.WithContext(ctx).Info("Initialize the node")

	if err = InitCommandLogic(ctx, config); err != nil {
		log.WithContext(ctx).Error("Initialize the node")
		return err
	}

	log.WithContext(ctx).Info("Node install was finished successfully")
	return nil
}

func runInstallWalletSubCommand(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info(fmt.Sprintf("Install walletnode on %s", utils.GetOS()))
	defer log.WithContext(ctx).Info("End install walletnode")

	configJSON, err := config.String()
	if err != nil {
		return err
	}

	if err = config.SaveConfig(); err != nil {
		return err
	}

	log.WithContext(ctx).Infof("Config: %s", configJSON)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sys.RegisterInterruptHandler(cancel, func() {
		log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
		os.Exit(0)
	})

	if _, err = initNodeDownloadPath(ctx, config, config.PastelWalletDir); err != nil {
		log.WithContext(ctx).Error("Failed to initialize install path!!!")
		return err
	}

	err = utils.DownloadFile(ctx,
		fmt.Sprintf("%s/%s", config.PastelWalletDir, constants.PastelWalletExecName[utils.GetOS()]),
		constants.PastelWalletDownloadURL[utils.GetOS()])

	if err != nil {
		log.WithContext(ctx).Error("Failed to download pastel executables.")
		return err
	}

	log.WithContext(ctx).Info("Download was finished successfully")

	log.WithContext(ctx).Info("Installing...")

	log.WithContext(ctx).Info(fmt.Sprintf("Wallet dir path -> %s/%s", config.PastelWalletDir, constants.PastelWalletExecName[utils.GetOS()]))
	if _, err = RunCMD("chmod", "777",
		fmt.Sprintf("%s/%s", config.PastelWalletDir, constants.PastelWalletExecName[utils.GetOS()])); err != nil {
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
	fileName, err := utils.CreateFile(ctx, workDirPath+"/wallet.yml", config.Force)
	if err != nil {
		return err
	}

	// write to file
	file, err := os.OpenFile(fileName, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Populate pastel.conf line-by-line to file.
	if config.Network == "mainnet" {
		_, err = file.WriteString(configs.WalletMainNetConfig) // creates server line
	} else if config.Network == "testnet" {
		_, err = file.WriteString(configs.WalletTestNetConfig) // creates server line
	} else {
		_, err = file.WriteString(configs.WalletLocalNetConfig) // creates server line
	}

	if err != nil {
		return err
	}

	log.WithContext(ctx).Info("Wallet node install was finished successfully")
	return nil
}

func runInstallSuperNodeSubCommand(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info(fmt.Sprintf("Install supernode on %s", utils.GetOS()))
	defer log.WithContext(ctx).Info("End install supernode")

	configJSON, err := config.String()
	if err != nil {
		return err
	}

	if err = config.SaveConfig(); err != nil {
		return err
	}

	log.WithContext(ctx).Infof("Config: %s", configJSON)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sys.RegisterInterruptHandler(cancel, func() {
		log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
		os.Exit(0)
	})

	if _, err = initNodeDownloadPath(ctx, config, config.PastelSuperNodeDir); err != nil {
		log.WithContext(ctx).Error("Failed to initialize install path!!!")
		return err
	}

	if config.Force {
		err = utils.DownloadFile(ctx,
			fmt.Sprintf("%s/%s", config.PastelSuperNodeDir, constants.PastelSuperNodeExecName[utils.GetOS()]),
			constants.PastelSuperNodeDownloadURL[utils.GetOS()])

		if err != nil {
			log.WithContext(ctx).Error("Failed to download pastel executables.")
			return err
		}
		log.WithContext(ctx).Info("Download was finished successfully")
	} else {
		if !utils.CheckFileExist(fmt.Sprintf("%s/%s", config.PastelSuperNodeDir, constants.PastelSuperNodeExecName[utils.GetOS()])) {
			err = utils.DownloadFile(ctx,
				fmt.Sprintf("%s/%s", config.PastelSuperNodeDir, constants.PastelSuperNodeExecName[utils.GetOS()]),
				constants.PastelSuperNodeDownloadURL[utils.GetOS()])

			if err != nil {
				log.WithContext(ctx).Error("Failed to download pastel executables.")
				return err
			}
			log.WithContext(ctx).Info("Download was finished successfully")
		} else {
			log.WithContext(ctx).Info("Supernode was already exist")
		}
	}

	log.WithContext(ctx).Info("Installing...")

	log.WithContext(ctx).Info(fmt.Sprintf("Supernode dir path -> %s/%s", config.PastelSuperNodeDir, constants.PastelSuperNodeExecName[utils.GetOS()]))
	if _, err = RunCMD("chmod", "777",
		fmt.Sprintf("%s/%s", config.PastelSuperNodeDir, constants.PastelSuperNodeExecName[utils.GetOS()])); err != nil {
		log.WithContext(ctx).Error("Failed to make wallet node as executable")
		return err
	}

	log.WithContext(ctx).Info("Initialize the supernode")

	workDirPath := filepath.Join(config.WorkingDir, "supernode")

	// create working dir path
	if err := utils.CreateFolder(ctx, workDirPath, config.Force); err != nil && err.Error() != constants.DirectoryExist {
		return err
	}

	// create walletnode default config
	// create file
	fileName, err := utils.CreateFile(ctx, filepath.Join(workDirPath, "supernode.yml"), config.Force)
	if err != nil && err.Error() != constants.FileExist {
		return err
	}

	// write to file
	file, err := os.OpenFile(fileName, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Populate pastel.conf line-by-line to file.
	_, err = file.WriteString(fmt.Sprintf(configs.SupernodeDefaultConfig, "some-value", "127.0.0.1", "4444")) // creates server line
	if err != nil {
		return err
	}

	tfmodelsPath := filepath.Join(workDirPath, "tfmodels")
	// create working dir path
	if err := utils.CreateFolder(ctx, tfmodelsPath, config.Force); err != nil && err.Error() != constants.DirectoryExist {
		return err
	}

	/*
		savedModelURL := "https://drive.google.com/uc?id=1U6tpIpZBxqxIyFej2EeQ-SbLcO_lVNfu"

		log.WithContext(ctx).Infof("Downloading: %s ...\n", savedModelURL)

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
	*/
	log.WithContext(ctx).Info("Supernode install was finished successfully")
	return nil
}

func initNodeDownloadPath(ctx context.Context, _ *configs.Config, nodeInstallPath string) (nodePath string, err error) {
	log.WithContext(ctx).Info("Check install node path")
	defer log.WithContext(ctx).Info("Initialized install node path")

	if _, err = os.Stat(nodeInstallPath); os.IsNotExist(err) {
		if err = utils.CreateFolder(ctx, nodeInstallPath, true); err != nil {
			return "", err
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
		return utils.Untar(dstFolder, fileReader)
	case constants.Mac:
		return utils.Untar(dstFolder, fileReader)
	case constants.Windows:
		_, err = utils.Unzip(archiveFile, dstFolder)
		return err
	default:
		log.WithContext(ctx).Error("Not supported OS!!!")
	}
	return fmt.Errorf("not supported OS")
}
