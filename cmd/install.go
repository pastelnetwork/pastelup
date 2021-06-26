package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

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

	defaultExecutableDir := configurer.DefaultPastelExecutableDir()

	installCommand := cli.NewCommand("install")
	installCommand.SetUsage("usage")

	installNodeSubCommand := cli.NewCommand("node")
	installNodeSubCommand.CustomHelpTemplate = GetColoredSubCommandHeaders()
	installNodeSubCommand.SetUsage(cyan("Install node"))

	installNodeFlags := []*cli.Flag{
		cli.NewFlag("iPath", &config.PastelNodeDir).SetUsage(green("Location where to create pastel node directory")).SetValue(defaultExecutableDir),
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

	installWalletSubCommand := cli.NewCommand("wallet")
	installWalletSubCommand.CustomHelpTemplate = GetColoredSubCommandHeaders()
	installWalletSubCommand.SetUsage(cyan("Install wallet"))
	installWalletSubCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "Install wallet", config)
		if err != nil {
			return err
		}
		return runInstallWalletSubCommand(ctx, config)
	})
	installWalletFlags := []*cli.Flag{
		cli.NewFlag("iPath", &config.PastelWalletDir).SetUsage(green("Location where to create pastel wallet node directory")).SetValue(defaultExecutableDir),
		cli.NewFlag("r", &flagRestart),
	}
	installWalletSubCommand.AddFlags(installWalletFlags...)

	installCommand.AddSubcommands(installNodeSubCommand)
	installCommand.AddSubcommands(installWalletSubCommand)

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
	log.WithContext(ctx).Debug("Copy config file")

	if err = copyFile(ctx,
		fmt.Sprintf("%s/%s", config.PastelNodeDir, constants.PASTEL_CONF_NAME),
		configurer.DefaultWorkingDir(),
		constants.PASTEL_CONF_NAME); err != nil {
		log.WithContext(ctx).Error("Failed to copy pastel.conf file")
		return err
	}

	log.WithContext(ctx).Info("Node install was finished successfully")
	return nil
}

func runInstallWalletSubCommand(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info(fmt.Sprintf("Install wallet on %s", utils.GetOS()))
	defer log.WithContext(ctx).Info("End install wallet")

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

	log.WithContext(ctx).Info("Wallet node install was finished successfully")
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
	return fmt.Errorf("not supported OS!!!")
}

func copyFile(ctx context.Context, src string, dstFolder string, dstFileName string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		log.WithContext(ctx).Error(fmt.Sprintf("%s file not exist!!!", src))
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		log.WithContext(ctx).Error(fmt.Sprintf("%s is not a regular file", src))
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		log.WithContext(ctx).Error(fmt.Sprintf("%s file cannot be opened!!!", src))
		return err
	}
	defer source.Close()

	if _, err := os.Stat(dstFolder); os.IsNotExist(err) {
		if err = utils.CreateFolder(ctx, dstFolder, true); err != nil {
			log.WithContext(ctx).Error(fmt.Sprintf("Could not create folder on this %s", dstFolder))
			return utils.CreateFolder(ctx, dstFolder, true)
		}
	}

	destination, err := os.Create(fmt.Sprintf("%s/%s", dstFolder, dstFileName))
	if err != nil {
		log.WithContext(ctx).Error(fmt.Sprintf("Could not copy file to %s", dstFolder))
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)

	return err
}
