package cmd

import (
	"context"
	"fmt"
	"time"

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

	flagToToolType = map[string]constants.ToolType{
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

// installSystemService installs already installed application as system service. For example, on linux, a user
// may run ./pastelup update install-service --tool node and this would install the systemd service for pasteld
// that can be controlled by systemtctl
func installSystemService(ctx context.Context, config *configs.Config) error {
	if err := isToolValid(config.ServiceTool); err != nil {
		return err
	}

	sm, err := NewServiceManager(utils.GetOS(), config.Configurer.DefaultHomeDir())
	if err != nil {
		return err // services feature not configured for users OS
	}
	tool := flagToToolType[config.ServiceTool]
	err = sm.RegisterService(ctx, config, tool, config.ServiceTool == "masternode")
	if err != nil {
		return err
	}

	if config.EnableService {
		err := sm.EnableService(ctx, config, tool)
		if err != nil {
			return fmt.Errorf("unable to enable %v service for auto start on boot: %v", config.ServiceTool, err)
		}
		log.WithContext(ctx).Infof("System service %s is enabled for auto start on boot", config.ServiceTool)
	}

	if config.StartService {
		isRunning, err := sm.StartService(ctx, config, tool)
		if !isRunning || err != nil {
			return fmt.Errorf("unable to start %v as a system service: %v", config.ServiceTool, err)
		}
		log.WithContext(ctx).Infof("Started %s as a system service", config.ServiceTool)
	}
	return nil
}

// removeSystemService stops and remove an installed system service. For example, on linux, a user
// may run ./pastelup update remove-service --tool node and this would stop and remove the systemd service running via systemtctl
// TODO: REMOVE service part is not yet implemented
func removeSystemService(ctx context.Context, config *configs.Config) error {
	if err := isToolValid(config.ServiceTool); err != nil {
		return err
	}
	sm, err := NewServiceManager(utils.GetOS(), config.Configurer.DefaultHomeDir())
	if err != nil {
		return err // services feature not configured for users OS
	}
	tool := flagToToolType[config.ServiceTool]
	err = sm.StopService(ctx, config, tool)
	if err != nil {
		return fmt.Errorf("unable to stop %v as a system service: %v", config.ServiceTool, err)
	}
	log.WithContext(ctx).Infof("Stopped %s as a system service", config.ServiceTool)
	return nil
}

//TODO: redo this method for installing ALL service for specific setups - SuperNode and WalletNode
func installServices(ctx context.Context, apps []constants.ToolType, config *configs.Config) error {
	sm, err := NewServiceManager(utils.GetOS(), config.Configurer.DefaultHomeDir())
	if err != nil {
		log.WithContext(ctx).Warn(err.Error())
		return nil // services aren't implemented for this OS
	}
	for _, app := range apps {
		err = sm.RegisterService(ctx, config, app, false)
		if err != nil {
			log.WithContext(ctx).Errorf("unable to register service %v: %v", app, err)
			return err
		}
		_, err := sm.StartService(ctx, config, app) // if app already running, this will be a noop
		if err != nil {
			log.WithContext(ctx).Errorf("unable to start service %v: %v", app, err)
			return err
		}
	}
	time.Sleep(5 * time.Second) // apply artificial buffer for services to start
	// verify services are up and running
	var nonRunningServices []constants.ToolType
	for _, app := range apps {
		isRunning := sm.IsRunning(ctx, config, app)
		if !isRunning {
			nonRunningServices = append(nonRunningServices, app)
		}
	}
	if len(nonRunningServices) > 0 {
		e := fmt.Errorf("unable to successfully start services: %+v", nonRunningServices)
		log.WithContext(ctx).Error(e.Error())
		return e
	}
	return nil
}
