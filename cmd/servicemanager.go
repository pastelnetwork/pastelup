package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/pastelup/configs"
	"github.com/pastelnetwork/pastelup/constants"
	"github.com/pastelnetwork/pastelup/utils"
)

// ServiceManager handles registering, starting and stopping system processes on the clients respective OS system manager (i.e. linux -> systemctl)
type ServiceManager interface {
	RegisterService(context.Context, constants.ToolType, RegistrationParams) error
	StartService(context.Context, *configs.Config, constants.ToolType) (bool, error)
	StopService(context.Context, *configs.Config, constants.ToolType) error
	EnableService(context.Context, *configs.Config, constants.ToolType) error
	DisableService(context.Context, *configs.Config, constants.ToolType) error
	IsRunning(context.Context, *configs.Config, constants.ToolType) bool
	IsRegistered(context.Context, *configs.Config, constants.ToolType) bool
	ServiceName(constants.ToolType) string
}

/*type systemdCmd string

const (
	start   systemdCmd = "start"
	stop    systemdCmd = "stop"
	enable  systemdCmd = "enable"
	disable systemdCmd = "disable"
	status  systemdCmd = "status"
)
*/

// NewServiceManager returns a new serviceManager, if the OS does not have one configured, the error will be set and Noop Manager will be returned
func NewServiceManager(os constants.OSType, homeDir string) (ServiceManager, error) {
	switch os {
	case constants.Linux:
		return LinuxSystemdManager{
			homeDir: homeDir,
		}, nil
	}
	// if you don't want to check error, we return a noop manager that will do nothing since
	// the user's system is not supported for system management
	return NoopManager{}, fmt.Errorf("services are not comptabile with your OS (%v)", os)
}

// NoopManager can be used to do nothing if the OS doesnt have a system manager configured
type NoopManager struct{}

// RegisterService registers a service with the OS system manager
func (nm NoopManager) RegisterService(context.Context, constants.ToolType, RegistrationParams) error {
	return nil
}

// StartService starts the given service as long as it is registered
func (nm NoopManager) StartService(context.Context, *configs.Config, constants.ToolType) (bool, error) {
	return false, nil
}

// StopService stops a running service, if it isn't running it is a no-op
func (nm NoopManager) StopService(context.Context, *configs.Config, constants.ToolType) error {
	return nil
}

// IsRunning checks to see if the service is running
func (nm NoopManager) IsRunning(context.Context, *configs.Config, constants.ToolType) bool {
	return false
}

// EnableService checks to see if the service is running
func (nm NoopManager) EnableService(context.Context, *configs.Config, constants.ToolType) error {
	return nil
}

// DisableService checks to see if the service is running
func (nm NoopManager) DisableService(context.Context, *configs.Config, constants.ToolType) error {
	return nil
}

// IsRegistered checks if the associated app's system command file exists, if it does it returns true, else it returns false
// if err is not nil, there was an error checking the existence of the file
func (nm NoopManager) IsRegistered(context.Context, *configs.Config, constants.ToolType) bool {
	return false
}

// ServiceName returns the formatted service name given a tooltype
func (nm NoopManager) ServiceName(constants.ToolType) string {
	return ""
}

// LinuxSystemdManager is a service manager for linux based OS
type LinuxSystemdManager struct {
	homeDir string
}

// RegistrationParams additional flags to pass during service registration
type RegistrationParams struct {
	Force       bool
	FlagDevMode bool
	Config      *configs.Config
}

// RegisterService registers the service and starts it
func (sm LinuxSystemdManager) RegisterService(ctx context.Context, app constants.ToolType, params RegistrationParams) error {
	log.WithContext(ctx).Infof("Installing %v as systemd service", app)

	if isRegistered := sm.IsRegistered(ctx, params.Config, app); isRegistered {
		return nil // already registered
	}

	var systemdFile string
	var err error
	var execCmd, execPath, workDir string

	// Service file - will be installed at /etc/systemd/system
	appServiceFileName := sm.ServiceName(app)
	appServiceFilePath := filepath.Join(constants.SystemdSystemDir, appServiceFileName)
	appServiceTempFilePath := filepath.Join("/tmp/", appServiceFileName)

	username, err := RunCMD("whoami")
	if err != nil {
		return fmt.Errorf("unable to get own user name (%v): %v", app, err)
	}

	pastelConfigPath := filepath.Join(params.Config.WorkingDir, constants.PastelConfName)

	switch app {
	case constants.DDImgService:
		appBaseDir := filepath.Join(sm.homeDir, constants.DupeDetectionServiceDir)
		appServiceWorkDirPath := filepath.Join(appBaseDir, "img_server")
		execCmd = "python3 -m  http.server 8000"
		workDir = appServiceWorkDirPath
	case constants.PastelD:
		var extIP string
		// Get pasteld path
		execPath = filepath.Join(params.Config.PastelExecDir, constants.PasteldName[utils.GetOS()])
		if exists := utils.CheckFileExist(execPath); !exists {
			log.WithContext(ctx).WithError(err).Error(fmt.Sprintf("Could not find %v executable file", app))
			return err
		}
		// Get external IP
		if extIP, err = utils.GetExternalIPAddress(); err != nil {
			log.WithContext(ctx).WithError(err).Error("Could not get external IP address")
			return err
		}
		execCmd = execPath + " --datadir=" + params.Config.WorkingDir + " --externalip=" + extIP + " --reindex"
		workDir = params.Config.PastelExecDir
	case constants.RQService:
		execPath = filepath.Join(params.Config.PastelExecDir, constants.PastelRQServiceExecName[utils.GetOS()])
		if exists := utils.CheckFileExist(execPath); !exists {
			log.WithContext(ctx).WithError(err).Error(fmt.Printf("Could not find %v executable file", app))
			return err
		}
		rqServiceArgs := fmt.Sprintf("--config-file=%s", params.Config.Configurer.GetRQServiceConfFile(params.Config.WorkingDir))
		execCmd = execPath + " " + rqServiceArgs
		workDir = params.Config.PastelExecDir
	case constants.DDService:
		execPath = filepath.Join(params.Config.PastelExecDir, utils.GetDupeDetectionExecName())
		if exists := utils.CheckFileExist(execPath); !exists {
			log.WithContext(ctx).WithError(err).Error(fmt.Printf("Could not find %v executable file", app))
			return err
		}
		ddConfigFilePath := filepath.Join(sm.homeDir,
			constants.DupeDetectionServiceDir,
			constants.DupeDetectionSupportFilePath,
			constants.DupeDetectionConfigFilename)
		execCmd = "python3 " + execPath + " " + ddConfigFilePath
		workDir = params.Config.PastelExecDir
	case constants.SuperNode:
		execPath = filepath.Join(params.Config.PastelExecDir, constants.SuperNodeExecName[utils.GetOS()])
		if exists := utils.CheckFileExist(execPath); !exists {
			log.WithContext(ctx).WithError(err).Error(fmt.Sprintf("Could not find %v executable file", app))
			return err
		}
		supernodeConfigPath := params.Config.Configurer.GetSuperNodeConfFile(params.Config.WorkingDir)
		execCmd = execPath + " --config-file=" + supernodeConfigPath + " --pastel-config-file=" + pastelConfigPath
		workDir = params.Config.PastelExecDir
	case constants.Hermes:
		execPath = filepath.Join(params.Config.PastelExecDir, constants.HermesExecName[utils.GetOS()])
		if exists := utils.CheckFileExist(execPath); !exists {
			log.WithContext(ctx).WithError(err).Error(fmt.Sprintf("Could not find %v executable file", app))
			return err
		}

		hermesConfigPath := params.Config.Configurer.GetHermesConfFile(params.Config.WorkingDir)
		execCmd = execPath + " --config-file=" + hermesConfigPath + " --pastel-config-file=" + pastelConfigPath
		workDir = params.Config.PastelExecDir
	case constants.WalletNode:
		execPath = filepath.Join(params.Config.PastelExecDir, constants.WalletNodeExecName[utils.GetOS()])
		if exists := utils.CheckFileExist(execPath); !exists {
			log.WithContext(ctx).WithError(err).Error(fmt.Sprintf("Could not find %v executable file", app))
			return err
		}
		walletnodeConfigFile := params.Config.Configurer.GetWalletNodeConfFile(params.Config.WorkingDir)
		execCmd = execPath + " --config-file=" + walletnodeConfigFile + " --pastel-config-file=" + pastelConfigPath
		if params.FlagDevMode {
			execCmd += " --swagger"
		}
		workDir = params.Config.PastelExecDir
	case constants.Bridge:
		execPath = filepath.Join(params.Config.PastelExecDir, constants.BridgeExecName[utils.GetOS()])
		if exists := utils.CheckFileExist(execPath); !exists {
			log.WithContext(ctx).WithError(err).Error(fmt.Sprintf("Could not find %v executable file", app))
			return err
		}

		bridgeConfigPath := params.Config.Configurer.GetBridgeConfFile(params.Config.WorkingDir)
		execCmd = execPath + " --config-file=" + bridgeConfigPath + " --pastel-config-file=" + pastelConfigPath
		workDir = params.Config.PastelExecDir
	default:
		return nil
	}

	// Create systemd file
	systemdFile, err = utils.GetServiceConfig(string(app), configs.SystemdService,
		&configs.SystemdServiceScript{
			Desc:    fmt.Sprintf("%v daemon", app),
			ExecCmd: execCmd,
			WorkDir: workDir,
			User:    username,
		})
	if err != nil {
		e := fmt.Errorf("unable ot create service file for (%v): %v", app, err)
		log.WithContext(ctx).WithError(err).Error(e.Error())
		return e
	}

	// write systemdFile to SystemdUserDir with mode 0644
	if err := ioutil.WriteFile(appServiceTempFilePath, []byte(systemdFile), 0644); err != nil {
		log.WithContext(ctx).WithError(err).Error("unable to write " + appServiceFileName + " file")
	}

	_, err = RunSudoCMD(params.Config, "cp", appServiceTempFilePath, appServiceFilePath)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to update")
		return err
	}

	// reload systemctl daemon
	_, err = RunSudoCMD(params.Config, "systemctl", "daemon-reload")
	if err != nil {
		return fmt.Errorf("unable to reload systemctl daemon (%v): %v", app, err)
	}

	// Enable service
	log.WithContext(ctx).Info("Setting service for auto start on boot")
	if out, err := RunSudoCMD(params.Config, "systemctl", "enable", appServiceFileName); err != nil {
		log.WithContext(ctx).WithFields(log.Fields{"message": out}).
			WithError(err).Error("unable to enable " + appServiceFileName + " service")
		return fmt.Errorf("err enabling "+appServiceFileName+" - err: %s", err)
	}
	return nil
}

// StartService starts the given service as long as it is registered
func (sm LinuxSystemdManager) StartService(ctx context.Context, config *configs.Config, app constants.ToolType) (bool, error) {
	isRegistered := sm.IsRegistered(ctx, config, app)
	if !isRegistered {
		log.WithContext(ctx).Infof("skipping start service because %v is not a registered service", app)
		return false, nil
	}
	isRunning := sm.IsRunning(ctx, config, app)
	if isRunning {
		log.WithContext(ctx).Infof("service %v is already running: noop", app)
		return true, nil
	}
	_, err := RunSudoCMD(config, "systemctl", "start", sm.ServiceName(app))
	if err != nil {
		return false, fmt.Errorf("unable to start service (%v): %v", app, err)
	}
	return true, nil
}

// StopService stops a running service, it isn't running it is a no-op
func (sm LinuxSystemdManager) StopService(ctx context.Context, config *configs.Config, app constants.ToolType) error {
	isRunning := sm.IsRunning(ctx, config, app) // if not registered, this will be false
	if !isRunning {
		return nil // service isnt running, no need to stop
	}
	_, err := RunSudoCMD(config, "systemctl", "stop", sm.ServiceName(app))
	if err != nil {
		return fmt.Errorf("unable to stop service (%v): %v", app, err)
	}
	return nil
}

// EnableService enables a systemd service
func (sm LinuxSystemdManager) EnableService(ctx context.Context, config *configs.Config, app constants.ToolType) error {
	appServiceFileName := sm.ServiceName(app)
	log.WithContext(ctx).Info("Enabling service for auto-start")
	if out, err := RunSudoCMD(config, "systemctl", "enable", appServiceFileName); err != nil {
		log.WithContext(ctx).WithFields(log.Fields{"message": out}).
			WithError(err).Error("unable to enable " + appServiceFileName + " service")
		return fmt.Errorf("err enabling "+appServiceFileName+" - err: %s", err)
	}
	return nil
}

// DisableService disables a systemd service
func (sm LinuxSystemdManager) DisableService(ctx context.Context, config *configs.Config, app constants.ToolType) error {
	appServiceFileName := sm.ServiceName(app)
	log.WithContext(ctx).Info("Disabling service", appServiceFileName)
	if out, err := RunSudoCMD(config, "systemctl", "disable", appServiceFileName); err != nil {
		log.WithContext(ctx).WithFields(log.Fields{"message": out}).
			WithError(err).Error("unable to disable " + appServiceFileName + " service")
		return fmt.Errorf("err enabling "+appServiceFileName+" - err: %s", err)
	}
	return nil
}

// IsRunning checks to see if the service is running
func (sm LinuxSystemdManager) IsRunning(ctx context.Context, config *configs.Config, app constants.ToolType) bool {
	res, _ := RunSudoCMD(config, "systemctl", "is-active", sm.ServiceName(app))
	res = strings.TrimSpace(res)
	log.WithContext(ctx).Infof("%v is-active status: %v", sm.ServiceName(app), res)
	return res == "active" || res == "activating"
}

// IsRegistered checks if the associated app's system command file exists, if it does, it returns true, else it returns false
func (sm LinuxSystemdManager) IsRegistered(ctx context.Context, config *configs.Config, app constants.ToolType) bool {
	res, _ := RunSudoCMD(config, "systemctl", "list-unit-files", sm.ServiceName(app))
	res = strings.TrimSpace(res)
	log.WithContext(ctx).Infof("%v list-unit-files status: %v", sm.ServiceName(app), res)

	return !strings.Contains(res, "0 unit files listed.")
}

// ServiceName returns the formatted service name given a tooltype
func (sm LinuxSystemdManager) ServiceName(app constants.ToolType) string {
	return fmt.Sprintf("%v%v.service", constants.SystemdServicePrefix, app)
}
