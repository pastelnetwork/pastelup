package cmd

import (
	"context"
	"fmt"
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/log/hooks"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"runtime"
)

func setupInitCommand(app *cli.App, config *configs.Config) {
	// define flags here
	var dirFlag string
	var networkFlag string
	var forceFlag bool
	var peerFlag string

	initCommand := cli.NewCommand("init")
	// initCommand.CustomAppHelpTemplate = getColoredHeaders(cyan)
	initCommand.CustomHelpTemplate = GetColoredHeaders(cyan)
	initCommand.SetUsage("Command that performs initialization of the system for both Wallet and SuperNodes")
	initCommandFlags := []*cli.Flag{
		cli.NewFlag("work-dir", &dirFlag).SetAliases("d").
			SetUsage("Location where to create working directory").SetValue("default"),
		cli.NewFlag("network", &networkFlag).SetAliases("n").
			SetUsage("Network type, can be - \"mainnet\" or \"testnet\"").SetValue("mainnet"),
		cli.NewFlag("force", &forceFlag).SetAliases("f").
			SetUsage("Force to overwrite config files and re-download ZKSnark parameters"),
		cli.NewFlag("peers", &peerFlag).SetAliases("p").
			SetUsage("List of peers to add into pastel.conf file, must be in the format - \"ip\" or \"ip:port\""),
	}
	initCommand.AddFlags(initCommandFlags...)
	addLogFlags(initCommand, config)

	// create walletnode and supernode subcommands
	walletnodeSubCommandName := green.Sprint("walletnode")
	walletnodeSubCommand := cli.NewCommand(walletnodeSubCommandName)
	walletnodeSubCommand.Usage = cyan.Sprint("Perform wallet specific initialization after common")

	supernodeSubCommandName := green.Sprint("supernode")
	supernodeSubCommand := cli.NewCommand(supernodeSubCommandName)
	supernodeSubCommand.Usage = cyan.Sprint("Perform Supernode/Masternode specific initialization after common")

	initSubCommands := []*cli.Command{
		walletnodeSubCommand,
		supernodeSubCommand,
	}
	initCommand.AddSubcommands(initSubCommands...)

	initCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx = log.ContextWithPrefix(ctx, "app")

		if config.Quiet {
			log.SetOutput(ioutil.Discard)
		} else {
			log.SetOutput(app.Writer)
		}

		if config.LogFile != "" {
			fileHook := hooks.NewFileHook(config.LogFile)
			log.AddHook(fileHook)
		}

		if err := log.SetLevelName(config.LogLevel); err != nil {
			return errors.Errorf("--log-level %q, %v", config.LogLevel, err)
		}

		log.Info("flag-name: ", dirFlag)
		config.WorkingDir = dirFlag
		config.Network = networkFlag
		config.Force = forceFlag

		return runInit(ctx, config)
	})
	app.AddCommands(initCommand)
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
	})

	// actions to run goes here

	//create directory
	err = createDirectory(config)
	if err != nil {
		return err
	}

	return nil
}

func getDetaultOsLocation(os string) string {
	switch os {
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
		return "~/.pastel"
	default:
		return ""
	}
}

func createDirectory(config *configs.Config) error {
	forceSet := config.Force
	path := config.WorkingDir

	// get default OS location if path flag is not explicitly set
	if path == "default" {
		path = getDetaultOsLocation(runtime.GOOS)
	}

	fmt.Printf("PATH: %s", path)

	if forceSet {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			return err
		}
	} else {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf("err %v", err)

			err := os.Mkdir(path, os.ModePerm)
			fmt.Printf("err %v", err)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
