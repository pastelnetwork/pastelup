package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/pastel-utility/configurer"
	"github.com/pastelnetwork/pastel-utility/constants"
	"github.com/pastelnetwork/pastel-utility/utils"
)

func setupStopCommand() *cli.Command {
	config := configs.GetConfig()

	defaultWorkingDir := configurer.DefaultWorkingDir()
	defaultExecutableDir := configurer.DefaultPastelExecutableDir()

	stopCommand := cli.NewCommand("stop")
	stopCommand.SetUsage("usage")

	allSubCommand := cli.NewCommand("all")
	allSubCommand.CustomHelpTemplate = GetColoredSubCommandHeaders()
	allSubCommand.SetUsage(cyan("Stop all"))
	allSubCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "allSubCommand", config)
		if err != nil {
			return err
		}
		return runStopAllSubCommand(ctx, config)
	})
	nodeFlags := []*cli.Flag{
		cli.NewFlag("dir", &config.PastelExecDir).SetAliases("d").
			SetUsage(green("Location where to create pastel node directory")).SetValue(defaultExecutableDir),
		cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
			SetUsage(green("Location where to create working directory")).SetValue(defaultWorkingDir),
	}
	allSubCommand.AddFlags(nodeFlags...)

	nodeSubCommand := cli.NewCommand("node")
	nodeSubCommand.CustomHelpTemplate = GetColoredSubCommandHeaders()
	nodeSubCommand.SetUsage(cyan("Stop specified node"))
	nodeSubCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "nodeSubCommand", config)
		if err != nil {
			return err
		}
		return runStopNodeSubCommand(ctx, config)
	})

	nodeSubCommand.AddFlags(nodeFlags...)

	walletSubCommand := cli.NewCommand("walletnode")
	walletSubCommand.CustomHelpTemplate = GetColoredSubCommandHeaders()
	walletSubCommand.SetUsage(cyan("Stop wallet"))
	walletSubCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "walletnodeSubCommand", config)
		if err != nil {
			return err
		}
		return runStopWalletSubCommand(ctx, config)
	})

	walletSubCommand.AddFlags(nodeFlags...)

	superNodeSubCommand := cli.NewCommand("supernode")
	superNodeSubCommand.CustomHelpTemplate = GetColoredSubCommandHeaders()
	superNodeSubCommand.SetUsage(cyan("Stop supernode"))
	superNodeSubCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "superNodeSubCommand", config)
		if err != nil {
			return err
		}
		return runStopSuperNodeSubCommand(ctx, config)
	})

	superNodeSubCommand.AddFlags(nodeFlags...)

	stopCommand.AddSubcommands(
		superNodeSubCommand,
		nodeSubCommand,
		walletSubCommand,
		allSubCommand,
	)

	stopCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "stopcommand", config)
		if err != nil {
			return err
		}

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

	return nil

}

func runStopAllSubCommand(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info(fmt.Sprintf("Stop all on %s", utils.GetOS()))
	defer log.WithContext(ctx).Info("End successfully")

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

	// *************  Kill process super node  *************
	log.WithContext(ctx).Info("Supernode process kill starting.")
	if _, err = processKillSuperNode(ctx, config); err != nil {
		return err
	}
	log.WithContext(ctx).Info("Supernode process ended.")

	// *************  Kill process wallet node  *************
	log.WithContext(ctx).Info("Walletnode process kill starting.")
	if _, err = processKillWalletNode(ctx, config); err != nil {
		return err
	}
	log.WithContext(ctx).Info("Walletnode process ended.")

	// TODO: Implement Stop node command
	log.WithContext(ctx).Info("Pasteld process kill starting.")
	if _, err = stopPatelCLI(ctx, config); err != nil {
		return err
	}
	log.WithContext(ctx).Info("Pasteld process ended.")

	time.Sleep(10000 * time.Millisecond)

	return nil
}

func runStopNodeSubCommand(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info(fmt.Sprintf("Stop node on %s", utils.GetOS()))
	defer log.WithContext(ctx).Info("End successfully")

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

	// TODO: Implement Stop node command
	log.WithContext(ctx).Info("Pasteld process kill starting.")
	if _, err = stopPatelCLI(ctx, config); err != nil {
		return err
	}
	log.WithContext(ctx).Info("Pasteld process ended.")

	return nil
}

func runStopWalletSubCommand(ctx context.Context, config *configs.Config) error {

	log.WithContext(ctx).Info(fmt.Sprintf("Stop wallet node on %s", utils.GetOS()))
	defer log.WithContext(ctx).Info("End successfully")

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

	// *************  Kill process wallet node  *************
	log.WithContext(ctx).Info("Walletnode process kill starting.")
	if _, err = processKillWalletNode(ctx, config); err != nil {
		return err
	}
	log.WithContext(ctx).Info("Walletnode process ended.")

	// *************  Stop pasteld  *************
	log.WithContext(ctx).Info("Pasteld process kill starting.")
	if _, err = stopPatelCLI(ctx, config); err != nil {
		return err
	}
	log.WithContext(ctx).Info("Pasteld process ended.")

	return nil
}

func runStopSuperNodeSubCommand(ctx context.Context, config *configs.Config) error {
	var err error

	log.WithContext(ctx).Info("Checking parameters...")
	log.WithContext(ctx).Info("Finished checking parameters!")
	log.WithContext(ctx).Info("Checking pastel config...")
	if err := CheckPastelConf(config); err != nil {
		log.WithContext(ctx).Error("pastel.conf was not correct!")
		return err
	}
	log.WithContext(ctx).Info("Finished checking pastel config!")

	// *************  Kill process super node  *************
	log.WithContext(ctx).Info("Supernode process kill starting.")
	if _, err = processKillSuperNode(ctx, config); err != nil {
		return err
	}
	log.WithContext(ctx).Info("Supernode process ended.")

	// *************  Stop pastel super node   *************
	log.WithContext(ctx).Info("Pasteld process kill starting.")
	if _, err = stopPatelCLI(ctx, config); err != nil {
		return err
	}
	log.WithContext(ctx).Info("Pasteld process ended.")

	return nil
}

func stopPatelCLI(ctx context.Context, config *configs.Config) (output string, err error) {
	var pasteldPath string

	if _, pasteldPath, _, _, err = checkPastelInstallPath(ctx, config, ""); err != nil {
		return pasteldPath, errNotFoundPastelPath
	}

	matches, err := filepath.Glob("/proc/*/exe")
	for _, file := range matches {
		target, _ := os.Readlink(file)
		if len(target) > 0 {
			if target == pasteldPath {
				if _, err = runPastelCLI(ctx, config, "stop"); err != nil {
					return "", err
				}
				break
			}
		}
	}

	return
}

func processKillWalletNode(ctx context.Context, config *configs.Config) (output string, err error) {
	var pastelWalletNodePath string
	var pID string
	var processID int

	if _, _, _, pastelWalletNodePath, err = checkPastelInstallPath(ctx, config, "wallet"); err != nil {
		return pastelWalletNodePath, errNotFoundPastelPath
	}

	matches, err := filepath.Glob("/proc/*/exe")
	for _, file := range matches {
		target, _ := os.Readlink(file)
		if len(target) > 0 {
			if target == pastelWalletNodePath {
				split := strings.Split(file, "/")

				pID = split[len(split)-2]
				processID, err = strconv.Atoi(pID)
				proc, err := os.FindProcess(processID)
				if err != nil {
					log.Println(err)
				}
				// Kill the process
				proc.Kill()

				break
			}
		}
	}

	return
}

func processKillSuperNode(ctx context.Context, config *configs.Config) (output string, err error) {
	var pID string
	var processID int
	var pastelSuperNodePath string

	if _, err = os.Stat(filepath.Join(config.PastelExecDir, constants.PastelSuperNodeExecName[utils.GetOS()])); os.IsNotExist(err) {
		log.WithContext(ctx).Error("could not find super node path")
		return "", fmt.Errorf("could not find super node path")
	}
	pastelSuperNodePath = filepath.Join(config.PastelExecDir, constants.PastelSuperNodeExecName[utils.GetOS()])

	matches, err := filepath.Glob("/proc/*/exe")
	for _, file := range matches {
		target, _ := os.Readlink(file)
		if len(target) > 0 {
			if target == pastelSuperNodePath {
				split := strings.Split(file, "/")

				pID = split[len(split)-2]
				processID, err = strconv.Atoi(pID)
				proc, err := os.FindProcess(processID)
				if err != nil {
					log.Println(err)
				}
				// Kill the process
				proc.Kill()

				break
			}
		}
	}

	return
}
