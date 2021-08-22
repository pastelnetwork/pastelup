package cmd

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/log/hooks"
	"github.com/pastelnetwork/gonode/common/version"
	"github.com/pastelnetwork/pastel-utility/configs"
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
func NewApp() *cli.App {

	app := cli.NewApp(appName)
	AppWriter = app.Writer
	app.SetUsage(appUsage)
	app.SetVersion(version.Version())

	app.HideHelp = false
	app.HideHelpCommand = false
	app.AddCommands(
		setupInstallCommand(),
		setupStartCommand(),
		setupStopCommand(),
		setupShowCommand(),
		setupUpdateCommand(),
		setupInfoCommand(),
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

// RunCMD runs shell command and returns output and error
func RunCMD(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)

	var stdBuffer bytes.Buffer
	mw := io.MultiWriter(os.Stdout, &stdBuffer)

	cmd.Stdout = mw
	cmd.Stderr = mw

	// Execute the command
	if err := cmd.Run(); err != nil {
		return stdBuffer.String(), err
	}

	return stdBuffer.String(), nil
}

// RunCMDWithInteractive runs shell command with interactive
func RunCMDWithInteractive(command string, args ...string) error {
	cmd := exec.Command(command, args...)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

// CreateUtilityConfigFile - Initialize the function
func CreateUtilityConfigFile(ctx context.Context, config *configs.Config) (err error) {
	configJSON, err := config.String()
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Bad config")
		return err
	}

	if err = config.SaveConfig(); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to save pastel-utility config file")
		return err
	}

	log.WithContext(ctx).Infof("Config: %s", configJSON)

	return nil
}
