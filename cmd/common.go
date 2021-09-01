package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	errors2 "github.com/pkg/errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-errors/errors"
	ps "github.com/mitchellh/go-ps"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/pastel-utility/constants"
	"github.com/pastelnetwork/pastel-utility/structure"
	"github.com/pastelnetwork/pastel-utility/utils"
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
		config.Network = constants.NetworkTestnet
	} else {
		config.Network = constants.NetworkMainnet
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

// KillProcessByPid kills process by its pid
func KillProcessByPid(ctx context.Context, pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to find process by pid = %d", pid)
		return err
	}
	if err := process.Kill(); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to kill process with pid = %d", pid)
		return err
	}
	return nil
}

// KillProcess kills pastel service if it is running
func KillProcess(ctx context.Context, toolType constants.ToolType) error {

	if pid, err := GetRunningProcessPid(toolType); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to check running processes")
		return err
	} else if pid != 0 {
		if err := KillProcessByPid(ctx, pid); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to kill service - %s", toolType)
			return err
		}
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

// RunCMD runs shell command and returns output and error
func RunCMD(command string, args ...string) (string, error) {
	return RunCMDWithEnvVariable(command, "", "", args...)
}

// RunCMDWithEnvVariable runs shell command with environmental variable and returns output and error
func RunCMDWithEnvVariable(command string, evName string, evValue string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)

	if len(evName) != 0 && len(evValue) != 0 {
		additionalEnv := fmt.Sprintf("%s=%s", evName, evValue)
		cmd.Env = append(os.Environ(), additionalEnv)
	}

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

// FindRunningProcessPid search in the ps list using search term
func FindRunningProcessPid(searchTerm string) (int, error) {

	if output, err := FindRunningProcess(searchTerm); len(output) != 0 {
		output = strings.Trim(output, " ")
		items := strings.Split(output, " ")
		for i := 0; i < 3; i++ {
			if len(items) > i {
				if pid, err := strconv.Atoi(items[i]); err == nil {
					return pid, nil
				}
			}
		}
	} else if err != nil {
		return 0, err
	}

	return 0, errors.Errorf("Cannot find running process using search term = %s", searchTerm)
}

// FindRunningProcess search in the ps list using search term
func FindRunningProcess(searchTerm string) (string, error) {

	c1 := exec.Command("ps", "afx")
	c2 := exec.Command("grep", searchTerm)
	c3 := exec.Command("grep", "-v", "grep")

	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	c1.Stdout = w1
	c2.Stdin = r1
	c2.Stdout = w2
	c3.Stdin = r2

	var stdBuffer bytes.Buffer
	mw := io.MultiWriter(os.Stdout, &stdBuffer)
	c3.Stdout = mw
	c1.Stderr = mw
	c2.Stderr = mw
	c3.Stderr = mw

	var err error
	if err = c1.Start(); err == nil {
		if err = c2.Start(); err == nil {
			if err = c3.Start(); err == nil {
				if err = c1.Wait(); err == nil {
					if err = w1.Close(); err == nil {
						if err = c2.Wait(); err == nil {
							if err = w2.Close(); err == nil {
								if err = c3.Wait(); err == nil {
									return stdBuffer.String(), nil
								}
							}
						}
					}
				}
			}
		}
	}

	return "", err
}

// GetSNPortList returns array of SuperNode ports for network
func GetSNPortList(config *configs.Config) []int {
	if config.Network == constants.NetworkTestnet {
		return constants.TestnetPortList
	}
	return constants.MainnetPortList
}

// GetMNSyncInfo gets result of "mnsync status"
func GetMNSyncInfo(ctx context.Context, config *configs.Config) (structure.RPCPastelMSStatus, error) {
	var mnstatus structure.RPCPastelMSStatus

	output, err := RunPastelCLI(ctx, config, "mnsync", "status")
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to get mnsync status from pasteld")
		return mnstatus, err
	}
	if err := json.Unmarshal([]byte(output), &mnstatus); err != nil {
		return mnstatus, err
	}
	return mnstatus, nil
}

// GetPastelInfo gets result of "getinfo"
func GetPastelInfo(ctx context.Context, config *configs.Config) (structure.RPCGetInfo, error) {
	var getifno structure.RPCGetInfo

	output, err := RunPastelCLI(ctx, config, "getinfo")
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to getinfo from pasteld")
		return getifno, err
	}

	// Master Node Output
	if err = json.Unmarshal([]byte(output), &getifno); err != nil {
		return getifno, err
	}
	return getifno, nil
}

// CheckPastelDRunning whether pasteld is running
func CheckPastelDRunning(ctx context.Context, config *configs.Config) (ret bool) {
	var failCnt = 0
	var err error

	log.WithContext(ctx).Info("Waiting the pasteld to be started...")

	for {
		if _, err = RunPastelCLI(ctx, config, "getinfo"); err != nil {
			time.Sleep(10000 * time.Millisecond)
			failCnt++
			if failCnt == 10 {
				return false
			}
		} else {
			break
		}
	}

	log.WithContext(ctx).Info("pasteld was started successfully")
	return true
}

// StopPastelDAndWait sends stop command to pasteld and waits 10 seconds
func StopPastelDAndWait(ctx context.Context, config *configs.Config) (err error) {
	log.WithContext(ctx).Info("Stopping pasteld...")
	if _, err = RunPastelCLI(ctx, config, "stop"); err != nil {
		return err
	}

	time.Sleep(10000 * time.Millisecond)
	log.WithContext(ctx).Info("Stopped pasteld")
	return nil
}

// CheckMasterNodeSync checks and waits until mnsync is "Finished"
func CheckMasterNodeSync(ctx context.Context, config *configs.Config) (err error) {
	for {

		mnstatus, err := GetMNSyncInfo(ctx, config)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("master node mnsync status call has failed")
			return err
		}

		if mnstatus.AssetName == "Initial" {
			if _, err = RunPastelCLI(ctx, config, "mnsync", "reset"); err != nil {
				log.WithContext(ctx).WithError(err).Error("master node reset has failed")
				return err
			}
			time.Sleep(10000 * time.Millisecond)
		}
		if mnstatus.IsSynced {
			log.WithContext(ctx).Info("master node was synced!")
			break
		}
		log.WithContext(ctx).Info("Waiting for sync...")

		getinfo, err := GetPastelInfo(ctx, config)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("master node getinfo call has failed")
			return err
		}
		log.WithContext(ctx).Infof("Loading blocks - block #%d; Node has %d connection", getinfo.Blocks, getinfo.Connections)

		time.Sleep(10000 * time.Millisecond)
	}

	return nil
}

// CheckZksnarkParams validates Zksnark files
func CheckZksnarkParams(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Checking pastel param files...")

	zksnarkPath := config.Configurer.DefaultZksnarkDir()

	for _, zksnarkParamsName := range configs.ZksnarkParamsNames {
		zksnarkParamsPath := filepath.Join(zksnarkPath, zksnarkParamsName)

		log.WithContext(ctx).Infof("Checking pastel param file : %s", zksnarkParamsPath)
		checkSum, err := utils.GetChecksum(ctx, zksnarkParamsPath)
		if err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to check param file : %s", zksnarkParamsPath)
			return err
		} else if checkSum != constants.PastelParamsCheckSums[zksnarkParamsName] {
			log.WithContext(ctx).Errorf("Wrong checksum of the pastel param file: %s", zksnarkParamsPath)
			return errors2.Errorf("Wrong checksum of the pastel param file: %s", zksnarkParamsPath)
		}
	}

	return nil
}
