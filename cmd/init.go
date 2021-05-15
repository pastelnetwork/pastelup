package cmd

import (
	"context"
	"io/ioutil"

	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/log/hooks"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pkg/errors"
)

func setupInitCommand(app *cli.App, config *configs.Config) {

	// define flags here
	var workDirectoryFlag string

	initCommand := cli.NewCommand()
	initCommand.Name = "init"
	initCommand.Usage = "Command that performs initialization of the system for both Wallet and SuperNodes"
	initCommandFlags := []*cli.Flag{
		cli.NewFlag("work-dir", &workDirectoryFlag),
	}
	initCommand.AddFlags(initCommandFlags...)
	addLogFlags(initCommand, config)

	// create walletnode and supernode subcommands
	walletnodeSubCommand := cli.NewCommand()
	walletnodeSubCommand.Name = green.Sprint("walletnode")
	walletnodeSubCommand.Usage = cyan.Sprint("Perform wallet specific initialization after common")

	supernodeSubCommand := cli.NewCommand()
	supernodeSubCommand.Name = green.Sprint("supernode")
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

		log.Info("flag-name: ", workDirectoryFlag)

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
