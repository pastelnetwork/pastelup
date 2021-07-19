package cmd

import (
	"context"
	"os"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pastelnetwork/pastel-utility/configs"
)

func setupUpdateCommand() *cli.Command {
	config := configs.New()

	// define flags here
	var updateFlag string

	updateCommand := cli.NewCommand("update")
	updateCommand.SetUsage("usage")
	updateCommandFlags := []*cli.Flag{
		cli.NewFlag("flag-name", &updateFlag),
	}
	updateCommand.AddFlags(updateCommandFlags...)
	addLogFlags(updateCommand, config)

	updateCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "updatecommand", config)
		if err != nil {
			return err
		}

		log.Info("flag-name: ", updateFlag)

		return runUpdate(ctx, config)
	})
	return updateCommand
}

func runUpdate(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Update")
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
