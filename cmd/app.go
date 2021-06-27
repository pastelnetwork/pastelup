package cmd

import (
	"context"
	"io"
	"io/ioutil"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/log/hooks"
	"github.com/pastelnetwork/gonode/common/version"
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pkg/errors"
)

const (
	appName  = "Pastel-Utility"
	appUsage = `This is a cli utility for Pastel Network.` // TODO: Write a clear description.

	// defaultConfigFile = ""
)

// AppWriter writer for logging
var AppWriter io.Writer

// NewApp inits a new command line interface.
func NewApp() *cli.App {

	app := cli.NewApp(appName)
	AppWriter = app.Writer
	app.SetUsage(appUsage)
	app.SetVersion(version.Version())
	app.SetCustomAppHelpTemplate(GetColoredHeaders())

	app.HideHelp = false
	app.HideHelpCommand = false
	app.AddCommands(
		setupInitCommand(),
		setupInstallCommand(),
		setupStartCommand(),
		setupStopCommand(),
		setupShowCommand(),
		setupUpdateCommand(),
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
