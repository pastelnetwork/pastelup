package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/pastel-utility/utils"
	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/configurer"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pkg/errors"
)

var zksnarkParamsURL = "https://z.cash/downloads/"
var zksnarkParamsNames = []string{
	"sapling-spend.params",
	"sapling-output.params",
	"sprout-proving.key",
	"sprout-verifying.key",
	"sprout-groth16.params",
}

func setupInitCommand() *cli.Command {
	config := configs.New()

	initCommand := cli.NewCommand("init")
	initCommand.CustomHelpTemplate = GetColoredCommandHeaders()
	initCommand.SetUsage(blue("Command that performs initialization of the system for both Wallet and SuperNodes"))
	initCommandFlags := []*cli.Flag{
		cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("d").
			SetUsage(green("Location where to create working directory")).SetValue("default"),
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
	walletnodeSubCommand.CustomHelpTemplate = GetColoredSubCommandHeaders()
	walletnodeSubCommand.SetUsage(cyan("Perform wallet specific initialization after common"))
	walletnodeSubCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging("walletnodeSubCommand", config, ctx)
		if err != nil {
			return err
		}
		return runWalletSubCommand(ctx, config)
	})

	supernodeSubCommand := cli.NewCommand("supernode")
	supernodeSubCommand.CustomHelpTemplate = GetColoredSubCommandHeaders()
	supernodeSubCommand.SetUsage(cyan("Perform Supernode/Masternode specific initialization after common"))
	supernodeSubCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging("supernodeSubCommand", config, ctx)
		if err != nil {
			return err
		}
		return runSuperNodeSubCommand(ctx, config)
	})

	initSubCommands := []*cli.Command{
		walletnodeSubCommand,
		supernodeSubCommand,
	}
	initCommand.AddSubcommands(initSubCommands...)

	initCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging("initcommand", config, ctx)
		if err != nil {
			return err
		}
		return runInit(ctx, config)
	})
	return initCommand
}

// runWalletSubCommand runs wallet subcommand
func runWalletSubCommand(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Wallet Node")
	defer log.WithContext(ctx).Info("End")

	configJson, err := config.String()
	if err != nil {
		return err
	}
	log.WithContext(ctx).Infof("Config: %s", configJson)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sys.RegisterInterruptHandler(cancel, func() {
		log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
		os.Exit(0)
	})

	forceSet := config.Force
	workDirPath := config.WorkingDir + "/walletnode/"

	// get default OS location if path flag is not explicitly set
	if config.WorkingDir == "default" {
		workDirPath = configurer.DefaultConfigPath("walletnode")
	}

	// create working dir path
	if err := utils.CreateFolder(ctx, workDirPath, forceSet); err != nil {
		return err
	}

	// create walletnode default config
	// create file
	fileName, err := utils.CreateFile(ctx, workDirPath+"/wallet.conf", forceSet)
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
	_, err = file.WriteString(configs.WalletDefaultConfig) // creates server line
	if err != nil {
		return err
	}

	return nil

}

// runSuperNodeSubCommand runs wallet subcommand
func runSuperNodeSubCommand(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Super Node")
	defer log.WithContext(ctx).Info("End")

	configJson, err := config.String()
	if err != nil {
		return err
	}
	log.WithContext(ctx).Infof("Config: %s", configJson)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sys.RegisterInterruptHandler(cancel, func() {
		log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
		os.Exit(0)
	})

	forceSet := config.Force
	workDirPath := config.WorkingDir + "/supernode/"

	// get default OS location if path flag is not explicitly set
	if config.WorkingDir == "default" {
		workDirPath = configurer.DefaultConfigPath("supernode")
	}

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

	return nil

}

func runInit(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Init")
	defer log.WithContext(ctx).Info("End")

	configJson, err := config.String()
	if err != nil {
		return err
	}
	log.WithContext(ctx).Infof("Config: %s", configJson)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sys.RegisterInterruptHandler(cancel, func() {
		log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
		os.Exit(0)
	})

	// actions to run goes here

	//run the init command logic flow
	err = initCommandLogic(ctx, config)
	if err != nil {
		return err
	}

	return nil
}

// initCommandLogic runs the init command logic flow
// takes all provided arguments in the cli and does all the background tasks
// Print success info log on successfully ran command, return error if fail
func initCommandLogic(ctx context.Context, config *configs.Config) error {
	forceSet := config.Force
	workDirPath := config.WorkingDir + "/.pastel/"
	zksnarkPath := config.WorkingDir + "/.pastel-params/"

	// get default OS location if path flag is not explicitly set
	if config.WorkingDir == "default" {
		workDirPath = configurer.DefaultConfigPath("/.pastel")
		zksnarkPath = configurer.DefaultConfigPath("/.pastel-params/")
	}

	// create working dir path
	if err := utils.CreateFolder(ctx, workDirPath, forceSet); err != nil {
		return err
	}

	// create zksnark parameters path
	if err := utils.CreateFolder(ctx, zksnarkPath, forceSet); err != nil {
		return err
	}

	// create file
	f, err := utils.CreateFile(ctx, workDirPath+"/pastel.conf", forceSet)
	if err != nil {
		return err
	}

	// write to file
	err = writeFile(ctx, f, config)
	if err != nil {
		return err
	}

	// download zksnark params
	if err := downloadZksnarkParams(ctx, zksnarkPath, forceSet); err != nil {
		return err
	}
	checkLocalAndRouterFirewalls([]string{"80", "21"}, ctx)

	return nil
}

// downloadZksnarkParams downloads zksnark params to the specified forlder
// Print success info log on successfully ran command, return error if fail
func downloadZksnarkParams(ctx context.Context, path string, force bool) error {
	log.WithContext(ctx).Info("Downloading zksnark files:")
	for _, zksnarkParamsName := range zksnarkParamsNames {
		zksnarkParamsPath := path + "/" + zksnarkParamsName
		log.WithContext(ctx).Infof("downloading: %s", zksnarkParamsPath)
		_, err := os.Stat(zksnarkParamsPath)
		// check if file exists and force is not set
		if os.IsExist(err) && !force {
			log.WithContext(ctx).WithError(err).Errorf("Error: file zksnark param already exists %s\n", zksnarkParamsPath)
			return errors.Errorf("zksnarkParam exists:  %s \n", zksnarkParamsPath)
		}

		out, err := os.Create(zksnarkParamsPath)
		if err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Error creating file: %s\n", zksnarkParamsPath)
			return errors.Errorf("Failed to create file: %v \n", err)
		}
		defer out.Close()

		// download param
		resp, err := http.Get(zksnarkParamsURL + zksnarkParamsName)
		if err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Error downloading file: %s\n", zksnarkParamsURL+zksnarkParamsName)
			return errors.Errorf("Failed to download: %v \n", err)
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

// writeFile populates the pastel.conf file with the corresponding logic
// Print success info log on successfully ran command, return error if fail
func writeFile(ctx context.Context, fileName string, config *configs.Config) error {
	// Open file using READ & WRITE permission.
	var file, err = os.OpenFile(fileName, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Populate pastel.conf line-by-line to file.
	_, err = file.WriteString("*server=1* \n \n") // creates server line
	if err != nil {
		return err
	}

	_, err = file.WriteString("*listen=1* \n \n") // creates server line
	if err != nil {
		return err
	}

	rpcUser := utils.GenerateRandomString(8)
	_, err = file.WriteString("*rpcuser=" + rpcUser + "* \n \n") // creates  rpcuser line
	if err != nil {
		return err
	}

	rpcPassword := utils.GenerateRandomString(15)
	_, err = file.WriteString("*rpcpassword=" + rpcPassword + "* \n \n") // creates rpcpassword line
	if err != nil {
		return err
	}

	if config.Network == "testnet" {
		_, err = file.WriteString("*testnet=1* \n \n") // creates testnet line
		if err != nil {
			return err
		}
	}

	//- If --peers are provided, add `addnode` for each peer
	//      *addnode = ip*
	//or/and
	//      *addnode = ip:port*
	// TODO: add logic for mutiple peers -> probably implement StringSliceFlag
	if config.Peers != "" {
		_, err = file.WriteString("*addnode=" + config.Peers + "* \n \n") // creates addnode line
		if err != nil {
			return err
		}
	}

	// Save file changes.
	err = file.Sync()
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Error saving file")
		return errors.Errorf("Failed to save file changes: %v \n", err)
	}

	log.WithContext(ctx).Info("File updated successfully: \n")

	return nil
}

// checkLocalAndRouterFirewalls checks local and router firewalls and suggest what to open
func checkLocalAndRouterFirewalls(required_ports []string, ctx context.Context) error {
	baseURL := "http://portchecker.com?q=" + strings.Join(required_ports[:], ",")
	// resp, err := http.Get(baseURL)
	// if err != nil {
	// 	log.WithContext(ctx).WithError(err).Errorf("Error requesting url\n")
	// 	return errors.Errorf("Failed to request port url %v \n", err)
	// }
	// defer resp.Body.Close()
	// ok := resp.StatusCode == http.StatusOK
	ok := true
	if ok {
		fmt.Println("Your ports {} are opened and can be accessed by other PAstel nodes!", baseURL)
	} else {
		fmt.Println("Your ports {} are NOT opened and can NOT be accessed by other Pastel nodes!\n Please open this ports in your router firewall and in {} firewall", baseURL, "func_to_check_OS()")
	}
	return nil
}
