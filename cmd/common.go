package cmd

import (
	"context"
	"fmt"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/pastel-utility/constants"
	"github.com/pastelnetwork/pastel-utility/utils"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
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
	return RunCMD("curl", "ipinfo.io/ip")
}

// ParsePastelConf parse configuration of pasteld.
func ParsePastelConf(ctx context.Context, config *configs.Config) error {

	pastelConfPath := filepath.Join(config.PastelExecDir, constants.PastelConfName)
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
func CheckProcessRunning(toolFileName string, execPath string) bool {
	var pID string
	var processID int

	if utils.GetOS() == constants.Windows {
		arg := fmt.Sprintf("IMAGENAME eq %s", toolFileName)
		out, err := RunCMD("tasklist", "/FI", arg)
		cnt := strings.Count(out, ",")
		if err != nil {
			return false
		}
		if strings.Contains(out, "No tasks") || cnt == 2 {
			return false
		}

	} else {
		matches, _ := filepath.Glob("/proc/*/exe")
		for _, file := range matches {
			target, _ := os.Readlink(file)
			if len(target) > 0 {
				if target == execPath {
					split := strings.Split(file, "/")

					pID = split[len(split)-2]
					processID, _ = strconv.Atoi(pID)
					_, err := os.FindProcess(processID)
					if err != nil {
						return false
					}
				}
			}
		}

	}

	return true
}
