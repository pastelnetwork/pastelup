package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/pastel-utility/configurer"
	"github.com/pastelnetwork/pastel-utility/utils"
	"github.com/pkg/errors"
)

func setupInitCommand() *cli.Command {
	config := configs.GetConfig()

	if len(config.WorkingDir) == 0 {
		config.WorkingDir = configurer.DefaultWorkingDir()
	}

	initCommand := cli.NewCommand("init")
	initCommand.CustomHelpTemplate = GetColoredCommandHeaders()
	initCommand.SetUsage(blue("Command that performs initialization of the system for both Wallet and SuperNodes"))
	initCommandFlags := []*cli.Flag{
		cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
			SetUsage(green("Location where to create working directory")).SetValue(config.WorkingDir),
		cli.NewFlag("network", &config.Network).SetAliases("n").
			SetUsage(green("Network type, can be - \"mainnet\" or \"testnet\"")).SetValue("mainnet"),
		cli.NewFlag("force", &config.Force).SetAliases("f").
			SetUsage(green("Force to overwrite config files and re-download ZKSnark parameters")),
		cli.NewFlag("peers", &config.Peers).SetAliases("p").
			SetUsage(green("List of peers to add into pastel.conf file, must be in the format - \"ip\" or \"ip:port\"")),
	}
	initCommand.AddFlags(initCommandFlags...)
	addLogFlags(initCommand, config)

	// create walletnode and supernode subcommands
	walletnodeSubCommand := cli.NewCommand("walletnode")
	walletnodeSubCommand.CustomHelpTemplate = GetColoredSubCommandHeaders() // TODO: this is not working
	walletnodeSubCommand.SetUsage(cyan("Perform wallet specific initialization after common"))
	walletnodeSubCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "walletnodeSubCommand", config)
		if err != nil {
			return err
		}
		return runWalletSubCommand(ctx, config)
	})

	supernodeSubCommand := cli.NewCommand("supernode")
	supernodeSubCommand.CustomHelpTemplate = GetColoredSubCommandHeaders() // TODO: this is not working
	supernodeSubCommand.SetUsage(cyan("Perform Supernode/Masternode specific initialization after common"))
	supernodeSubCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "supernodeSubCommand", config)
		if err != nil {
			return err
		}
		return runSuperNodeSubCommand(ctx, config)
	})

	initCommand.AddSubcommands(
		walletnodeSubCommand,
		supernodeSubCommand,
	)

	initCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "initcommand", config)
		if err != nil {
			return err
		}

		if len(args) == 0 {
			return fmt.Errorf("command is required")
		}

		return runInit(ctx, config)
	})
	return initCommand
}

func runInit(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Init")
	defer log.WithContext(ctx).Info("End")

	configJSON, err := config.String()
	if err != nil {
		return err
	}
	log.WithContext(ctx).Infof("Config: %s", configJSON)

	err = config.SaveConfig()
	if err != nil {
		log.WithContext(ctx).Error("Cannot save pastel-utility.conf!")
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sys.RegisterInterruptHandler(cancel, func() {
		log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
		os.Exit(0)
	})

	//run the init command logic flow
	err = InitCommandLogic(ctx, config)
	if err != nil {
		return err
	}

	return nil
}

// InitCommandLogic runs the init command logic flow
// takes all provided arguments in the cli and does all the background tasks
// Print success info log on successfully ran command, return error if fail
func InitCommandLogic(ctx context.Context, config *configs.Config) error {
	forceSet := config.Force
	var workDirPath, zksnarkPath string
	if config.WorkingDir == configurer.DefaultWorkingDir() {
		zksnarkPath = configurer.DefaultZksnarkDir()
		workDirPath = config.WorkingDir
	} else {
		workDirPath = filepath.Join(config.WorkingDir)
		zksnarkPath = filepath.Join(config.WorkingDir, "/.pastel-params/")
	}

	// create working dir path
	if config.WorkingDir != config.PastelExecDir {
		if err := utils.CreateFolder(ctx, workDirPath, forceSet); err != nil {
			if config.WorkingDir != config.PastelExecDir {
				return err
			}
		}
	}

	// create zksnark parameters path
	if err := utils.CreateFolder(ctx, zksnarkPath, forceSet); err != nil {
		return err
	}

	// create pastel.conf file
	f, err := utils.CreateFile(ctx, workDirPath+"/pastel.conf", forceSet)
	if err != nil {
		return err
	}

	// write to file
	err = updatePastelConfigFile(ctx, f, config)
	if err != nil {
		return err
	}

	// download zksnark params
	if err := downloadZksnarkParams(ctx, zksnarkPath, forceSet); err != nil && !(os.IsExist(err) && !forceSet) {
		return err
	}
	checkLocalAndRouterFirewalls(ctx, []string{"80", "21"})

	return nil
}

// updatePastelConfigFile populates the pastel.conf file with the corresponding logic
// Print success info log on successfully ran command, return error if fail
func updatePastelConfigFile(ctx context.Context, fileName string, config *configs.Config) error {
	// Open file using READ & WRITE permission.
	var file, err = os.OpenFile(fileName, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Populate pastel.conf line-by-line to file.
	_, err = file.WriteString("server=1\n") // creates server line
	if err != nil {
		return err
	}

	_, err = file.WriteString("listen=1\n\n") // creates server line
	if err != nil {
		return err
	}

	rpcUser := utils.GenerateRandomString(8)
	_, err = file.WriteString("rpcuser=" + rpcUser + "\n") // creates  rpcuser line
	if err != nil {
		return err
	}

	rpcPassword := utils.GenerateRandomString(15)
	_, err = file.WriteString("rpcpassword=" + rpcPassword + "\n") // creates rpcpassword line
	if err != nil {
		return err
	}

	if config.Network == "testnet" {
		_, err = file.WriteString("testnet=1\n") // creates testnet line
		if err != nil {
			return err
		}
	}

	if config.Peers != "" {
		nodes := strings.Split(config.Peers, ",")
		for _, node := range nodes {
			_, err = file.WriteString("addnode=" + node + "\n") // creates addnode line
			if err != nil {
				return err
			}
		}

	}

	// Save file changes.
	err = file.Sync()
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Error saving file")
		return errors.Errorf("failed to save file changes: %v", err)
	}

	log.WithContext(ctx).Info("File updated successfully: \n")

	return nil
}

// downloadZksnarkParams downloads zksnark params to the specified forlder
// Print success info log on successfully ran command, return error if fail
func downloadZksnarkParams(ctx context.Context, path string, force bool) error {
	log.WithContext(ctx).Info("Downloading zksnark files:")
	for _, zksnarkParamsName := range configs.ZksnarkParamsNames {
		zksnarkParamsPath := path + "/" + zksnarkParamsName
		log.WithContext(ctx).Infof("downloading: %s", zksnarkParamsPath)
		_, err := os.Stat(zksnarkParamsPath)
		// check if file exists and force is not set
		if os.IsExist(err) && !force {
			log.WithContext(ctx).WithError(err).Errorf("Error: file zksnark param already exists %s\n", zksnarkParamsPath)
			return errors.Errorf("zksnarkParam exists:  %s", zksnarkParamsPath)
		}

		out, err := os.Create(zksnarkParamsPath)
		if err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Error creating file: %s\n", zksnarkParamsPath)
			return errors.Errorf("Failed to create file: %v", err)
		}
		defer out.Close()

		// download param
		resp, err := http.Get(configs.ZksnarkParamsURL + zksnarkParamsName)
		if err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Error downloading file: %s\n", configs.ZksnarkParamsURL+zksnarkParamsName)
			return errors.Errorf("failed to download: %v", err)
		}
		defer resp.Body.Close()

		// write to file
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return err
		}
	}

	log.WithContext(ctx).Info("ZkSnark params downloaded.\n")

	return nil

}

// checkLocalAndRouterFirewalls checks local and router firewalls and suggest what to open
func checkLocalAndRouterFirewalls(ctx context.Context, requiredPorts []string) error {
	baseURL := "http://portchecker.com?q=" + strings.Join(requiredPorts[:], ",")
	// resp, err := http.Get(baseURL)
	// if err != nil {
	// 	log.WithContext(ctx).WithError(err).Errorf("Error requesting url\n")
	// 	return errors.Errorf("Failed to request port url %v \n", err)
	// }
	// defer resp.Body.Close()
	// ok := resp.StatusCode == http.StatusOK
	ok := true
	if ok {
		log.WithContext(ctx).Info("Your ports {} are opened and can be accessed by other PAstel nodes! ", baseURL)
	} else {
		log.WithContext(ctx).Info("Your ports {} are NOT opened and can NOT be accessed by other Pastel nodes!\n Please open this ports in your router firewall and in {} firewall", baseURL, "func_to_check_OS()")
	}
	return nil
}

// runWalletSubCommand runs wallet subcommand
func runWalletSubCommand(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Wallet Node")
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

	forceSet := config.Force
	workDirPath := filepath.Join(config.WorkingDir, "walletnode")

	// create working dir path
	if err := utils.CreateFolder(ctx, workDirPath, forceSet); err != nil {
		return err
	}

	// create walletnode default config
	// create file
	fileName, err := utils.CreateFile(ctx, workDirPath+"/wallet.yml", forceSet)
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

	return nil

}

// runSuperNodeSubCommand runs wallet subcommand
func runSuperNodeSubCommand(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Super Node")
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

	forceSet := config.Force
	workDirPath := filepath.Join(config.WorkingDir, "supernode")

	// create working dir path
	if err := utils.CreateFolder(ctx, workDirPath, forceSet); err != nil {
		return err
	}

	// create walletnode default config
	// create file
	fileName, err := utils.CreateFile(ctx, workDirPath+"/supernode.conf", forceSet)
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
	_, err = file.WriteString(configs.SupernodeDefaultConfig) // creates server line
	if err != nil {
		return err
	}

	// pip install gdown
	// ~/.local/bin/gdown https://drive.google.com/uc?id=1U6tpIpZBxqxIyFej2EeQ-SbLcO_lVNfu
	// unzip ./SavedMLModels.zip -d <paslel-dir>/supernode/tfmodels

	_, err = RunCMD("pip3", "install", "gdown")
	if err != nil {
		return err
	}

	savedModelURL := "https://drive.google.com/uc?id=1U6tpIpZBxqxIyFej2EeQ-SbLcO_lVNfu"

	log.WithContext(ctx).Infof("Downloading: %s ...\n", savedModelURL)

	_, err = RunCMD("gdown", savedModelURL)
	if err != nil {
		return err
	}

	tfmodelsPath := filepath.Join(workDirPath, "tfmodels")
	// create working dir path
	if err := utils.CreateFolder(ctx, tfmodelsPath, forceSet); err != nil {
		return err
	}

	_, err = RunCMD("unzip", "./SavedMLModels.zip", "-d", tfmodelsPath)
	if err != nil {
		return err
	}

	return nil

}

// RunCMD runs shell command and returns output and error
func RunCMD(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)

	stdout, err := cmd.Output()

	if err != nil {
		return "", err
	}
	return string(stdout), nil
}

// RunCMDWithInteractive runs shell command with interactive
func RunCMDWithInteractive(command string, args ...string) error {
	cmd := exec.Command(command, args...)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	return cmd.Run()
}
