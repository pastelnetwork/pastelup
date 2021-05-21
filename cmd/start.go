package cmd

import (
	"context"
	"io/ioutil"

	"github.com/edentech88/pastel-utility/configs"
	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/log/hooks"
	"github.com/pkg/errors"
)

func setupStartCommand(app *cli.App, config *configs.Config) {

	// define flags here
	var startFlag string

	startCommand := cli.NewCommand()
	startCommand.Name = "start"
	app.SetUsage("usage")
	startCommandFlags := []*cli.Flag{
		cli.NewFlag("flag-name", &startFlag),
	}
	startCommand.AddFlags(startCommandFlags...)

	startCommand.SetActionFunc(func(ctx context.Context, args []string) error {
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

		log.Info("flag-name: ", startFlag)

		return nil
	})
	app.AddCommands(startCommand)
}
