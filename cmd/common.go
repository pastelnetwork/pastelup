package cmd

import (
	"bufio"
	"context"
	"fmt"
	"github.com/go-errors/errors"
	ps "github.com/mitchellh/go-ps"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/pastel-utility/constants"
	"github.com/pastelnetwork/pastel-utility/utils"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func checkPastelFilePath(ctx context.Context, dirPath string, filePath string) (fullPath string, err error) {

	fullPath = filepath.Join(dirPath, filePath)
	if _, err = os.Stat(fullPath); os.IsNotExist(err) {
		log.WithContext(ctx).Errorf("could not find path - %s", fullPath)
		return "", fmt.Errorf("could not find path - %s", fullPath)
	} else if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("failed to check path = %s", fullPath)
		return "", err
	}

	return fullPath, err
}

// GetExternalIPAddress runs shell command and returns external IP address
func GetExternalIPAddress() (externalIP string, err error) {

	resp, err := http.Get(constants.IPCheckURL)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// ParsePastelConf parse configuration of pasteld.
func ParsePastelConf(ctx context.Context, config *configs.Config) error {

	pastelConfPath := filepath.Join(config.WorkingDir, constants.PastelConfName)
	if _, err := os.Stat(pastelConfPath); os.IsNotExist(err) {
		log.WithContext(ctx).WithError(err).Errorf("Could not find pastel config - %s", pastelConfPath)
		return err
	} else if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to find pastel config - %s", pastelConfPath)
		return err
	}

	var file, err = os.OpenFile(pastelConfPath, os.O_RDWR, 0644)
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Could not open pastel config - %s", pastelConfPath)
		return err
	}
	defer file.Close()

	configure, err := ioutil.ReadAll(file)
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Could not read pastel config - %s", pastelConfPath)
		return err
	}

	if strings.Contains(string(configure), "testnet=1") {
		config.Network = "testnet"
	} else {
		config.Network = "mainnet"
	}

	return nil
}

// CheckProcessRunning checks if the process is running
func CheckProcessRunning(toolType constants.ToolType) bool {
	if pid, err := GetRunningProcessPid(toolType); pid != 0 && err == nil {
		return true
	}
	return false
}

// GetRunningProcessPid returns process id, if the pastel service is running
func GetRunningProcessPid(toolType constants.ToolType) (int, error) {
	execName := constants.ServiceName[toolType][utils.GetOS()]
	proc, err := ps.Processes()
	if err != nil {
		return 0, errors.Errorf("failed to get list process: %v", err)
	}
	pid := 0
	for _, p := range proc {
		length := len(p.Executable())
		if length > (len(execName)) {
			length = len(execName)
		}
		nameForTest := execName[:length]
		if nameForTest == p.Executable() {
			pid = p.Pid()
			break
		}
	}

	return pid, nil
}

// KillProcess kills pastel service if it is running
func KillProcess(ctx context.Context, toolType constants.ToolType) error {

	if pid, err := GetRunningProcessPid(toolType); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to check running processes")
		return err
	} else if pid != 0 {
		process, err := os.FindProcess(pid)
		if err != nil {
			return errors.Errorf("failed to kill process - %d: %v", pid, err)
		}
		return process.Kill()
	}

	log.WithContext(ctx).Infof("Service %s is not running", toolType)
	return nil
}

// AskUserToContinue ask user interactively  Yes or No question
func AskUserToContinue(ctx context.Context, question string) bool {

	log.WithContext(ctx).Warn(red(question))

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Something went wrong...")
		return false
	}

	return strings.EqualFold(strings.TrimSpace(line), "y")
}

// RunPastelCLI runs pastel-cli commands
func RunPastelCLI(ctx context.Context, config *configs.Config, args ...string) (output string, err error) {
	var pastelCliPath string

	if pastelCliPath, err = checkPastelFilePath(ctx, config.PastelExecDir, constants.PastelCliName[utils.GetOS()]); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Could not find pastel-cli at %s (PastelExecDir is %s)", pastelCliPath, config.PastelExecDir)
		return "", err
	}

	args = append([]string{fmt.Sprintf("--datadir=%s", config.WorkingDir)}, args...)

	return RunCMD(pastelCliPath, args...)
}
