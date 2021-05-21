package cmd

import (
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/version"
)

const (
	appName  = "Pastel-Utility"
	appUsage = `Set up usage here` // TODO: Write a clear description.

	defaultConfigFile = ""
)

// NewApp inits a new command line interface.
func NewApp() *cli.App {
	config := configs.New()

	app := cli.NewApp(appName)
	app.SetUsage(appUsage)
	app.SetVersion(version.Version())

	//setup start command
	//setup install command
	//setup init command

	setupInitCommand(app, config)
	setupStartCommand(app, config)
    setupStopCommand(app, config)
	setupShowCommand(app, config)
	setupUpdateCommand(app, config)

	return app
}

func addLogFlags(command *cli.Command, config *configs.Config) {
	command.AddFlags(
		// Main
		cli.NewFlag("log-level", &config.LogLevel).SetUsage("Set the log `level`.").SetValue(config.LogLevel),
		cli.NewFlag("log-file", &config.LogFile).SetUsage("The log `file` to write to."),
		cli.NewFlag("quiet", &config.Quiet).SetUsage("Disallows log output to stdout.").SetAliases("q"),
	)
}
