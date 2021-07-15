package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/pastel-utility/utils"
)

var (
	flagOSVersion bool
)

func setupInfoCommand() *cli.Command {
	config := configs.GetConfig()

	infoCommand := cli.NewCommand("info")
	infoCommand.SetUsage("usage")

	infoFlags := []*cli.Flag{
		cli.NewFlag("os-version", &flagOSVersion).SetAliases("ov").
			SetUsage(green("Get OS version of running machine")),
	}
	infoCommand.AddFlags(infoFlags...)

	infoCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "Get Info", config)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		sys.RegisterInterruptHandler(cancel, func() {
			log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
			os.Exit(0)
		})

		return runInfoSubCommand(ctx, config)
	})

	return infoCommand
}

func runInfoSubCommand(ctx context.Context, config *configs.Config) error {

	if flagOSVersion {
		fmt.Print(utils.GetOS())
	}
	return nil

}
