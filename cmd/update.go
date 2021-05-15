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

func setupUpdateCommand(app *cli.App, config *configs.Config) {

	// define flags here
	var updateFlag string

	updateCommand := cli.NewCommand("update")
	updateCommand.SetUsage("") // TODO write down usage description
	updateCommandFlags := []*cli.Flag{
		cli.NewFlag("flag-name", &updateFlag),
	}
	updateCommand.AddFlags(updateCommandFlags...)
	addLogFlags(updateCommand, config)

	updateCommand.SetActionFunc(func(ctx context.Context, args []string) error {
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

		log.Info("flag-name: ", updateFlag)

		return runUpdate(ctx, config)
	})
	app.AddCommands(updateCommand)
}

func runUpdate(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Update")
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
