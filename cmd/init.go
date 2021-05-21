package cmd

import (
	"context"
	"github.com/fatih/color"
	"io/ioutil"

	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/log/hooks"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pkg/errors"
)

func setupInitCommand(app *cli.App, config *configs.Config) {
	//define colors
	cyan := color.New(color.FgCyan)
	// define flags here
	var initFlag string

	initCommand := cli.NewCommand()
	initCommand.Name = "init"
	initCommand.Usage = "Command that performs initialization of the system for both Wallet and SuperNodes"
	initCommandFlags := []*cli.Flag{
		cli.NewFlag("flag-name", &initFlag),
	}

	// create walletnode and supernode subcommands
	walletnodeSubCommand := cli.NewCommand()
	walletnodeSubCommand.Name = "walletnode"
	walletnodeSubCommand.Usage = cyan.Sprint("Perform wallet specific initialization after common")

	supernodeSubCommand := cli.NewCommand()
	supernodeSubCommand.Name = "supernode"
	supernodeSubCommand.Usage = cyan.Sprint("Perform Supernode/Masternode specific initialization after common")
	initSubCommands := []*cli.Command{
		walletnodeSubCommand,
		supernodeSubCommand,
	}
	initCommand.AddSubcommands(initSubCommands...)
	initCommand.AddFlags(initCommandFlags...)
	addLogFlags(initCommand, config)

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

		log.Info("flag-name: ", initFlag)

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

	return nil

}
