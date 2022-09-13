package cmd

import (
	"context"
	"io"
	"io/ioutil"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/log/hooks"
	"github.com/pastelnetwork/pastelup/common/version"
	"github.com/pastelnetwork/pastelup/configs"
	"github.com/pkg/errors"
)

const (
	appName  = "Pastel-Utility"
	appUsage = `This is a tool for installation, configuration and running of Pastel network nodes both - SuperNode and WalletNode.`

	// defaultConfigFile = ""
)

// AppWriter writer for logging
var AppWriter io.Writer

// NewApp inits a new command line interface.
func NewApp(args []string) *cli.App {

	app := cli.NewApp(appName)
	AppWriter = app.Writer
	app.SetUsage(appUsage)
	app.SetVersion(version.Version())

	app.HideHelp = false
	app.HideHelpCommand = false
	/*
	 * @todo we need to use different configs just b/c we set the Operation.
	 * we should persist the cmd line input into the config so we can determine operation by args[1]
	 * and remove the need to re-instatiate a new conffig for each operation.
	 */
	app.AddCommands(
		setupInstallCommand(configs.InitConfig(args)),
		setupUpdateCommand(configs.InitConfig(args)),
		setupStartCommand(configs.InitConfig(args)),
		setupInitCommand(configs.InitConfig(args)),
		setupStopCommand(configs.InitConfig(args)),
		setupShowCommand(configs.InitConfig(args)),
		setupInfoCommand(configs.InitConfig(args)),
		setupPingCommand(configs.InitConfig(args)),
	)
	return app
}

func addLogFlags(command *cli.Command, config *configs.Config) {
	command.AddFlags(
		// Main
		cli.NewFlag("log-level", &config.LogLevel).SetUsage(green("Set the log `level`.")).SetValue(config.LogLevel),
		cli.NewFlag("log-file", &config.LogFile).SetUsage(green("The log `file` to write to.")),
		cli.NewFlag("quiet", &config.Quiet).SetUsage(green("Disallows log output to stdout.")).SetAliases("q"),
	)
}

func configureLogging(ctx context.Context, logPrefix string, config *configs.Config) (context.Context, error) {
	ctx = log.ContextWithPrefix(ctx, logPrefix)

	if config.Quiet {
		log.SetOutput(ioutil.Discard)
	} else {
		log.SetOutput(AppWriter)
	}

	if config.LogFile != "" {
		fileHook := hooks.NewFileHook(config.LogFile)
		log.AddHook(fileHook)
	}

	if err := log.SetLevelName(config.LogLevel); err != nil {
		return nil, errors.Errorf("--log-level %q, %v", config.LogLevel, err)
	}
	return ctx, nil
}
