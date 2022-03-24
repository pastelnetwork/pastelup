package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	errors2 "github.com/pkg/errors"

	"github.com/go-errors/errors"
	ps "github.com/mitchellh/go-ps"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/pastelup/configs"
	"github.com/pastelnetwork/pastelup/constants"
	"github.com/pastelnetwork/pastelup/structure"
	"github.com/pastelnetwork/pastelup/utils"
)

// @todo move all invocations to use utils.CheckFileExists
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

// ParsePastelConf parse configuration of pasteld.
func ParsePastelConf(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Infof("parsing pastel conf at %s", config.WorkingDir)

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

	log.WithContext(ctx).WithField("pastel.conf", string(configure)).Info("pastel conf")
	if strings.Contains(string(configure), "testnet=1") {
		config.Network = constants.NetworkTestnet
	} else if strings.Contains(string(configure), "regtest=1") {
		config.Network = constants.NetworkRegTest
	} else {
		config.Network = constants.NetworkMainnet
	}

	return nil
}

// GetProcessCmdInput gets the arguments of a process. returns true if it was running and false if it wasnt or there was an error
func GetProcessCmdInput(toolType constants.ToolType) (bool, []string) {
	var cmdArgs []string
	pid, err := GetRunningProcessPid(toolType)
	if err != nil {
		return false, cmdArgs
	}
	output, err := RunCMD("bash", "-c", fmt.Sprintf("ps -o args= -f -p %v", pid))
	if err != nil {
		return false, cmdArgs
	}
	args := strings.Split(output, " ")
	for i, arg := range args {
		if i != 0 && arg != "" {
			cmdArgs = append(cmdArgs, strings.Trim(arg, "\n"))
		}
	}
	return true, cmdArgs
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
func AskUserToContinue(ctx context.Context, question string) (bool, string) {

	log.WithContext(ctx).Warn(red(question))

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Something went wrong...")
		return false, ""
	}

	input := strings.TrimSpace(line)
	return strings.EqualFold(input, "y"), input
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

func RunSudoCMD(config *configs.Config, args ...string) (string, error) {
	if len(config.UserPw) > 0 {
		return RunCMD("bash", "-c", "echo "+config.UserPw+" | sudo -S "+strings.Join(args, " "))
	}
	return RunCMD("sudo", args...)
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
	if config.Network == constants.NetworkRegTest {
		return constants.TestnetPortList
	} else if config.Network == constants.NetworkTestnet {
		return constants.RegTestPortList
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
func CheckPastelDRunning(ctx context.Context, config *configs.Config) bool {
	var failCnt = 0
	var err error

	log.WithContext(ctx).Info("Waiting the pasteld to be started...")

	for {
		if _, err = RunPastelCLI(ctx, config, "getinfo"); err != nil {
			time.Sleep(10 * time.Second)
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
	log.WithContext(ctx).Info("Stopping local pasteld...")
	if _, err = RunPastelCLI(ctx, config, "stop"); err != nil {
		return err
	}

	time.Sleep(10 * time.Second)
	log.WithContext(ctx).Info("Stopped local pasteld")
	return nil
}

// CheckMasterNodeSync checks and waits until mnsync is "Finished", return number of synced blocks
func CheckMasterNodeSync(ctx context.Context, config *configs.Config) (int, error) {
	var getinfo structure.RPCGetInfo
	var err error

	for {
		// Checking getinfo
		log.WithContext(ctx).Info("Waiting for sync...")

		getinfo, err = GetPastelInfo(ctx, config)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("master node getinfo call has failed")
			return 0, err
		}
		log.WithContext(ctx).Infof("Loading blocks - block #%d; Node has %d connection", getinfo.Blocks, getinfo.Connections)

		// Checking mnsync status
		mnstatus, err := GetMNSyncInfo(ctx, config)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("master node mnsync status call has failed")
			return 0, err
		}

		if mnstatus.AssetName == "Initial" {
			if _, err = RunPastelCLI(ctx, config, "mnsync", "reset"); err != nil {
				log.WithContext(ctx).WithError(err).Error("master node reset has failed")
				return 0, err
			}
			time.Sleep(10 * time.Second)
		}
		if mnstatus.IsSynced {
			log.WithContext(ctx).Info("master node was synced!")
			break
		}

		time.Sleep(10 * time.Second)
	}

	return getinfo.Blocks, nil
}

// CheckZksnarkParams validates Zksnark files
func CheckZksnarkParams(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Checking pastel param files...")

	zksnarkPath := config.Configurer.DefaultZksnarkDir()

	zkParams := configs.ZksnarkParamsNamesV2
	if config.Legacy {
		zkParams = append(zkParams, configs.ZksnarkParamsNamesV1...)
	}

	for _, zksnarkParamsName := range zkParams {
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

func copyPastelUpToRemote(ctx context.Context, client *utils.Client, remotePastelUp string) error {
	// Check if the current os is linux
	if runtime.GOOS == "linux" {
		log.WithContext(ctx).Infof("copying pastelup to remote")
		var localPastelupPath string

		// Get local pastelup path
		ex, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get path of executable file %s", err)
		}

		// Check if localPastelupPath is a symlink file
		if localPastelupPath, err = filepath.EvalSymlinks(ex); err != nil {
			return fmt.Errorf("local pastelup is symbol link:  %s", err)
		}

		// Copy pastelup to remote
		if err := client.Scp(localPastelupPath, remotePastelUp); err != nil {
			return fmt.Errorf("failed to copy pastelup to remote %s", err)
		}

	} else {
		log.WithContext(ctx).Infof("current OS is not linux, skipping pastelup copy")

		// Download PastelUpExecName from remote and save to remotePastelUp
		log.WithContext(ctx).Infof("downloading pastelup from Pastel download portal ...")
		version := "beta"
		downloadURL := fmt.Sprintf("%s/%s/%s", constants.DownloadBaseURL, version, constants.PastelUpExecName["Linux"])

		if _, err := client.Cmd(fmt.Sprintf("wget %s -O %s", downloadURL, remotePastelUp)).Output(); err != nil {
			return fmt.Errorf("failed to download pastelup from remote: %s", err.Error())
		}
		log.WithContext(ctx).Infof("pastelup downloaded from Pastel download portal")
	}

	// chmod +x remote pastelup
	if _, err := client.Cmd(fmt.Sprintf("chmod +x %s", remotePastelUp)).Output(); err != nil {
		return fmt.Errorf("failed to chmod +x pastelup at remote: %s", err.Error())
	}

	return nil
}

func prepareRemoteSession(ctx context.Context, config *configs.Config) (*utils.Client, error) {
	var err error

	// Validate config
	if len(config.RemoteIP) == 0 {
		log.WithContext(ctx).Fatal("remote IP is required")
		return nil, fmt.Errorf("remote IP is required")
	}

	// Connect to remote node
	log.WithContext(ctx).Infof("connecting to remote host -> %s:%d...", config.RemoteIP, config.RemotePort)

	var client *utils.Client

	if len(config.RemoteSSHKey) == 0 {
		username, password, _ := utils.Credentials(config.RemoteUser, true)
		client, err = utils.DialWithPasswd(fmt.Sprintf("%s:%d", config.RemoteIP, config.RemotePort), username, password)
	} else {
		username, _, _ := utils.Credentials(config.RemoteUser, false)
		client, err = utils.DialWithKey(fmt.Sprintf("%s:%d", config.RemoteIP, config.RemotePort), username, config.RemoteSSHKey)
	}
	if err != nil {
		return nil, err
	}

	log.WithContext(ctx).Info("connected successfully")

	// Transfer pastelup to remote
	log.WithContext(ctx).Info("installing pastelup to remote host...")

	if err := copyPastelUpToRemote(ctx, client, constants.RemotePastelupPath); err != nil {
		log.WithContext(ctx).Errorf("Failed to copy pastelup to remote at %s - %v", constants.RemotePastelupPath, err)
		client.Close()
		return nil, fmt.Errorf("failed to install pastelup at %s - %v", constants.RemotePastelupPath, err)
	}
	log.WithContext(ctx).Info("successfully install pastelup executable to remote host")

	return client, nil
}
