package cmd

import (
	"context"
	"fmt"

	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/pastelup/configs"
	"github.com/pastelnetwork/pastelup/constants"
	"github.com/pastelnetwork/pastelup/utils"
)

// ToolTypeServices represents the list of tool types that can be enabled as system services
// i.e. systemd services if on linux
var (
	installServiceFlag = []string{
		"node",
		"masternode",
		"supernode",
		"walletnode",
		"dd-service",
		"rq-service",
		"hermes",
		"bridge",
		"dd-img-service",
	}

	installSolutionFlag = []string{
		"supernode",
		"walletnode",
	}

	solutionToTools = map[string][]string{
		"supernode": {
			"masternode",
			"supernode",
			"dd-service",
			"rq-service",
			"hermes",
			"dd-img-service",
		},
		"walletnode": {
			"node",
			"walletnode",
			"rq-service",
			"bridge",
		},
	}

	toolToToolType = map[string]constants.ToolType{
		"node":           constants.PastelD,
		"masternode":     constants.PastelD,
		"supernode":      constants.SuperNode,
		"walletnode":     constants.WalletNode,
		"dd-service":     constants.DDService,
		"rq-service":     constants.RQService,
		"hermes":         constants.Hermes,
		"bridge":         constants.Bridge,
		"dd-img-service": constants.DDImgService,
	}
)

func isToolValid(toolFlag string) error {
	isValid := false
	for _, t := range installServiceFlag {
		if t == toolFlag {
			isValid = true
		}
	}
	if !isValid {
		return fmt.Errorf("tool %v is not a valid tool type to run as a service. Please use one of %+v", toolFlag, installServiceFlag)
	}
	return nil
}

func isSolutionValid(solutionFlag string) error {
	isValid := false
	for _, t := range installSolutionFlag {
		if t == solutionFlag {
			isValid = true
		}
	}
	if !isValid {
		return fmt.Errorf("tool %v is not a valid solution type to run as a service. Please use one of %+v", solutionFlag, installSolutionFlag)
	}
	return nil
}

func toolToProcess(config *configs.Config) ([]string, error) {
	var toolsToInstall []string
	if err := isSolutionValid(config.ServiceSolution); err != nil {
		if err := isToolValid(config.ServiceTool); err != nil {
			return nil, err
		}
		return append(toolsToInstall, config.ServiceTool), nil
	}
	return append(toolsToInstall, solutionToTools[config.ServiceSolution]...), nil
}

// installSystemService installs already installed application as system service. For example, on linux, a user
// may run ./pastelup update install-service --tool node and this would install the systemd service for pasteld
// that can be controlled by systemtctl
func installSystemService(ctx context.Context, config *configs.Config) error {

	toolsToInstall, err := toolToProcess(config)
	if err != nil {
		return err
	}

	sm, err := NewServiceManager(utils.GetOS(), config.Configurer.DefaultHomeDir())
	if err != nil {
		return err // services feature not configured for users OS
	}

	for _, app := range toolsToInstall {

		tool := toolToToolType[app]

		err = sm.RegisterService(ctx, config, tool, app == "masternode")
		if err != nil {
			return err
		}
		log.WithContext(ctx).Infof("System service %s is registered for auto start on boot", app)

		if config.EnableService {
			err := sm.EnableService(ctx, config, tool)
			if err != nil {
				return fmt.Errorf("unable to enable %s service for auto start on boot: %v", app, err)
			}
			log.WithContext(ctx).Infof("System service %s is enabled for auto start on boot", app)
		}

		if config.StartService {
			isRunning, err := sm.StartService(ctx, config, tool)
			if !isRunning || err != nil {
				return fmt.Errorf("unable to start %s as a system service: %v", app, err)
			}
			log.WithContext(ctx).Infof("Started %s as a system service", app)
		}
	}
	return nil
}

// removeSystemService stops and remove an installed system service. For example, on linux, a user
// may run ./pastelup update remove-service --tool node and this would stop and remove the systemd service running via systemtctl
// TODO: REMOVE service part is not yet implemented
//lint:ignore U1000 ignore for now
func removeSystemService(ctx context.Context, config *configs.Config) error {

	toolsToUnInstall, err := toolToProcess(config)
	if err != nil {
		return err
	}

	sm, err := NewServiceManager(utils.GetOS(), config.Configurer.DefaultHomeDir())
	if err != nil {
		return err // services feature not configured for users OS
	}
	for _, app := range toolsToUnInstall {

		tool := toolToToolType[app]

		err = sm.StopService(ctx, config, tool)
		if err != nil {
			return fmt.Errorf("unable to stop %s as a system service: %v", app, err)
		}
		log.WithContext(ctx).Infof("Stopped %s as a system service", app)
	}
	return nil
}
