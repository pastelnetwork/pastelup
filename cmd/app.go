package cmd

import (
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

	app := cli.NewApp(appName)
	app.SetUsage(appUsage)
	app.SetVersion(version.Version())

	//setup start command
	//setup install command
	//setup init command

	return app
}
