package cmd

import (
	"context"
	"os"

	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
)

func setupStartCommand() *cli.Command {
	config := configs.New()

	// define flags here
	var startFlag string

	startCommand := cli.NewCommand("start")
	startCommand.SetUsage("usage")
	startCommandFlags := []*cli.Flag{
		cli.NewFlag("flag-name", &startFlag),
	}
	startCommand.AddFlags(startCommandFlags...)
	addLogFlags(startCommand, config)

	startCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging("startcommand", config, ctx)
		if err != nil {
			return err
		}

		log.Info("flag-name: ", startFlag)

		return runStart(ctx, config)
	})
	return startCommand
}

func runStart(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Start")
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

	return nil

}
