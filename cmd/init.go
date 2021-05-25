package cmd

import (
	"context"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pastelnetwork/pastel-utility/configs"
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
	initCommand.CustomHelpTemplate = GetColoredHeaders(cyan)
	initCommand.SetUsage("Command that performs initialization of the system for both Wallet and SuperNodes")
	initCommandFlags := []*cli.Flag{
		cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("d").
			SetUsage("Location where to create working directory").SetValue("default"),
		cli.NewFlag("network", &config.Network).SetAliases("n").
			SetUsage("Network type, can be - \"mainnet\" or \"testnet\"").SetValue("mainnet"),
		cli.NewFlag("force", &config.Force).SetAliases("f").
			SetUsage("Force to overwrite config files and re-download ZKSnark parameters"),
		cli.NewFlag("peers", &config.Peers).SetAliases("p").
			SetUsage("List of peers to add into pastel.conf file, must be in the format - \"ip\" or \"ip:port\""),
	}
	initCommand.AddFlags(initCommandFlags...)
	addLogFlags(initCommand, config)

	// create walletnode and supernode subcommands
	// walletnodeSubCommandName := green.Sprint("walletnode")
	walletnodeSubCommand := cli.NewCommand("walletnode")
	walletnodeSubCommand.Usage = cyan.Sprint("Perform wallet specific initialization after common")
	walletnodeSubCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging("walletnodeSubCommand", config, ctx)
		if err != nil {
			return err
		}
		return runWalletSubCommand(ctx, config)
	})

	// supernodeSubCommandName := green.Sprint("supernode")
	supernodeSubCommand := cli.NewCommand("supernode")
	supernodeSubCommand.Usage = cyan.Sprint("Perform Supernode/Masternode specific initialization after common")
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
	if err := createFolder(ctx, workDirPath, forceSet); err != nil {
		return err
	}

	// create walletnode default config
	// create file
	fileName, err := createFile(ctx, workDirPath+"/wallet.conf", forceSet)
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
	if err := createFolder(ctx, workDirPath, forceSet); err != nil {
		return err
	}

	// create walletnode default config
	// create file
	fileName, err := createFile(ctx, workDirPath+"/supernode.conf", forceSet)
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

// getDefaultOsLocation returns the pre defined directory creation path
// for the given Operating System
// returns `path` string
func getDefaultOsLocation(system string) string {
	switch system {
	case "windows":
		// TODO: check the Windows major version (something like the w32 api library)
		// if Vista or newer use C:\Users\Username\AppData\Roaming\Pastel
		// for older versions use C:\Documents and Settings\Username\Application Data\Pastel
		winVer := 10
		path := "C:\\Documents and Settings\\Username\\Application Data\\Pastel"
		if winVer >= 6 {
			path = "C:\\Users\\Username\\AppData\\Roaming\\Pastel"
		}
		return path
	case "darwin":
		return "~/Library/Application Support/Pastel"
	case "linux":
		homeDir, _ := os.UserHomeDir()
		return homeDir + "/.pastel"
	default:
		return ""
	}
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
	if err := createFolder(ctx, workDirPath, forceSet); err != nil {
		return err
	}

	// create zksnark parameters path
	if err := createFolder(ctx, zksnarkPath, forceSet); err != nil {
		return err
	}

	// create file
	f, err := createFile(ctx, workDirPath+"/pastel.conf", forceSet)
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

// createFolder creates the folder in the specified `path`
// Print success info log on successfully ran command, return error if fail
func createFolder(ctx context.Context, path string, force bool) error {
	if force {
		err := os.MkdirAll(path, 0755)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Error creating directory")
			return errors.Errorf("Failed to create directory: %v \n", err)
		}
		log.WithContext(ctx).Infof("Directory created on %s \n", path)
	} else {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			err := os.MkdirAll(path, 0755)
			if err != nil {
				log.WithContext(ctx).WithError(err).Error("Error creating directory")
				return errors.Errorf("Failed to create directory: %v \n", err)
			}
			log.WithContext(ctx).Infof("Directory created on %s \n", path)
		} else {
			log.WithContext(ctx).WithError(err).Error("Directory already exists \n")
			return errors.Errorf("Directory already exists \n")
		}
	}

	return nil
}

// createFile creates pastel.conf file
// Print success info log on successfully ran command, return error if fail
func createFile(ctx context.Context, fileName string, force bool) (string, error) {

	if force {
		var file, err = os.Create(fileName)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Error creating file")
			return "", errors.Errorf("Failed to create file: %v \n", err)
		}
		defer file.Close()
	} else {
		// check if file exists
		var _, err = os.Stat(fileName)

		// create file if not exists
		if os.IsNotExist(err) {
			var file, err = os.Create(fileName)
			if err != nil {
				log.WithContext(ctx).WithError(err).Error("Error creating file")
				return "", errors.Errorf("Failed to create file: %v \n", err)
			}
			defer file.Close()
		} else {
			log.WithContext(ctx).WithError(err).Error("File already exists \n")
			return "", errors.Errorf("File already exists \n")
		}
	}

	log.WithContext(ctx).Infof("File created: %s \n", fileName)

	return fileName, nil
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

	rpcUser := generateRandomString(8)
	_, err = file.WriteString("*rpcuser=" + rpcUser + "* \n \n") // creates  rpcuser line
	if err != nil {
		return err
	}

	rpcPassword := generateRandomString(15)
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

// generateRandomString is a helper func for generating
// random string of the given input length
// returns the generated string
func generateRandomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789")
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	str := b.String()

	return str
}
