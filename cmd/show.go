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

func setupShowCommand(app *cli.App, config *configs.Config) {

	// define flags here
	var showFlag string

	showCommand := cli.NewCommand()
	showCommand.Name = "show"
	showCommand.Usage = "" // TODO write down usage description
	showCommandFlags := []*cli.Flag{
		cli.NewFlag("flag-name", &showFlag),
	}
	showCommand.AddFlags(showCommandFlags...)
	addLogFlags(showCommand, config)

	showCommand.SetActionFunc(func(ctx context.Context, args []string) error {
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

		log.Info("flag-name: ", showFlag)

		return runShow(ctx, config)
	})
	app.AddCommands(showCommand)
}

func runShow(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Show")
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
