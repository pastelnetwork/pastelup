package cmd

import (
	"context"
	"os"

	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
)

func setupStopCommand() *cli.Command {
	config := configs.New()

	// define flags here
	var stopFlag string

	stopCommand := cli.NewCommand("stop")
	stopCommand.SetUsage("usage")
	stopCommandFlags := []*cli.Flag{
		cli.NewFlag("flag-name", &stopFlag),
	}
	stopCommand.AddFlags(stopCommandFlags...)
	addLogFlags(stopCommand, config)

	stopCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "stopcommand", config)
		if err != nil {
			return err
		}

		log.Info("flag-name: ", stopFlag)

		return runStop(ctx, config)
	})
	return stopCommand
}

func runStop(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Stop")
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

	// actions to run goes here

	return nil

}
