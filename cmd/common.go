package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-errors/errors"
	ps "github.com/mitchellh/go-ps"
	errors2 "github.com/pkg/errors"

	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/pastelup/configs"
	"github.com/pastelnetwork/pastelup/constants"
	"github.com/pastelnetwork/pastelup/services/pastelcore"
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

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "rpcuser=") {
			config.RPCUser = strings.TrimPrefix(line, "rpcuser=")
		}
		if strings.HasPrefix(line, "rpcpassword=") {
			config.RPCPwd = strings.TrimPrefix(line, "rpcpassword=")
		}
		if strings.HasPrefix(line, "rpcport=") {
			config.RPCPort, _ = strconv.Atoi(strings.TrimPrefix(line, "rpcport="))
		}
		if strings.HasPrefix(line, "testnet=") {
			isTestnet, _ := strconv.ParseBool(strings.TrimPrefix(line, "testnet="))
			config.IsTestnet = isTestnet
			if isTestnet {
				config.Network = constants.NetworkTestnet
			}
		}
		if strings.HasPrefix(line, "regtest=") {
			isRegTestnet, _ := strconv.ParseBool(strings.TrimPrefix(line, "regtest="))
			if isRegTestnet {
				config.Network = constants.NetworkRegTest
			}
		}
		if strings.HasPrefix(line, "txindex=") {
			config.TxIndex, _ = strconv.Atoi(strings.TrimPrefix(line, "txindex="))
		}
	}
	// if both testnet=1 and regtest=1 are not set in pastel.conf -- mainnet mode is on
	if config.Network == "" {
		config.Network = constants.NetworkMainnet
	}
	return nil
}

// GetProcessCmdInput gets the arguments of a process. returns true if it was running and false if it wasn't or there was an error
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

	log.WithContext(ctx).Infof("Application %s is not running", toolType)
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

// RunSudoCMD takes in the config and applies user password if set to do so
// else it runs the sudo command and asks the user to input their password
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
func FindRunningProcessPid(ctx context.Context, searchTerm string) (int, error) {

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
	log.WithContext(ctx).Infof("Cannot find running process using search term = %s", searchTerm)

	return 0, nil
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
	} else if config.Network == constants.NetworkRegTest {
		return constants.RegTestPortList
	}
	return constants.MainnetPortList
}

// GetMNSyncInfo gets result of "mnsync status"
func GetMNSyncInfo(ctx context.Context, config *configs.Config) (structure.RPCPastelMNSyncStatus, error) {
	var mnstatus structure.RPCPastelMNSyncStatus
	err := pastelcore.NewClient(config).RunCommandWithArgs(
		pastelcore.MasterNodeSyncCmd,
		[]string{"status"},
		&mnstatus,
	)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to get mnsync status from pasteld")
		return mnstatus, err
	}
	return mnstatus, nil
}

// GetPastelInfo gets result of "getinfo"
func GetPastelInfo(ctx context.Context, config *configs.Config) (structure.RPCGetInfo, error) {
	var info structure.RPCGetInfo
	err := pastelcore.NewClient(config).RunCommand(pastelcore.GetInfoCmd, &info)
	if err != nil {
		log.WithContext(ctx).Warnf("unable to access pastel server (on port %d) to get pastel info... [%v]", config.RPCPort, err)
		return info, err
	}
	// this indicates we got an empty or errored response
	if info.Result.Version == 0 {
		log.WithContext(ctx).Errorf("info response has errors: %v", info.Error)
		return info, fmt.Errorf("info response has errors")
	}
	return info, nil
}

func containsAny(str string, substrings []string) bool {
	for _, substring := range substrings {
		if strings.Contains(str, substring) {
			return true
		}
	}
	return false
}

// WaitingForPastelDToStart whether pasteld is running
func WaitingForPastelDToStart(ctx context.Context, config *configs.Config) bool {
	log.WithContext(ctx).Info("Waiting the pasteld to start...")
	var attempts = 0
	var maxAttempts = 30
	for attempts <= maxAttempts {
		info, err := GetPastelInfo(ctx, config)
		if err == nil {
			log.WithContext(ctx).Info("pasteld started successfully")
			return true
		}
		if info.Error != nil {
			errorMap, ok := info.Error.(map[string]interface{})
			if ok {
				//errorCode, _ := errorMap["code"].(int)
				errorMessage, _ := errorMap["message"].(string)
				subMessages := []string{"Rescanning", "Activating best chain", "Loading block index"}
				if containsAny(errorMessage, subMessages) {
					if attempts == maxAttempts-1 {
						maxAttempts += 30
					}
				}
			}
		}

		time.Sleep(10 * time.Second)
		attempts++
	}
	return false
}

// StopPastelDAndWait sends stop command to pasteld and waits 10 seconds
func StopPastelDAndWait(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Stopping local pasteld...")
	var resp map[string]interface{}
	err := pastelcore.NewClient(config).RunCommand(pastelcore.StopCmd, &resp)
	if err != nil {
		log.WithContext(ctx).Errorf("unable to stop pastel: %v", err)
		return err
	}
	time.Sleep(10 * time.Second)
	log.WithContext(ctx).Infof("Stopped local pasteld: %+v", resp)
	return nil
}

// CheckMasterNodeSync checks and waits until mnsync is "Finished", return number of synced blocks
func CheckMasterNodeSync(ctx context.Context, config *configs.Config) (int, error) {
	var getinfo structure.RPCGetInfo
	var err error
	t := time.Now()
	fmt.Println() // add some space for loading line
	for {
		getinfo, err = GetPastelInfo(ctx, config)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("master node getinfo call has failed")
			return 0, err
		}
		// Checking mnsync status
		mnstatus, err := GetMNSyncInfo(ctx, config)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("master node mnsync status call has failed")
			return 0, err
		}
		// line overwrites itself to avoid abundant loggin
		log.WithContext(ctx).Infof("Waiting for sync... Loading blocks - block #%d; Node has %d connection; mnstatus=%v, isSynced=%v (elapsed: %v)\r", getinfo.Result.Blocks, getinfo.Result.Connections, mnstatus.Result.AssetName, mnstatus.Result.IsSynced, time.Since(t))
		if mnstatus.Result.AssetName == "Initial" {
			var output interface{}
			err := pastelcore.NewClient(config).RunCommandWithArgs(pastelcore.MasterNodeSyncCmd, []string{"reset"}, &output)
			if err != nil {
				log.WithContext(ctx).WithError(err).Error("\nmaster node reset has failed")
				return 0, err
			}
			time.Sleep(10 * time.Second)
		}
		if mnstatus.Result.IsSynced {
			log.WithContext(ctx).Info("\nmasternodes lists are synced!")
			break
		}
		time.Sleep(10 * time.Second)
	}
	return getinfo.Result.Blocks, nil
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

func copyPastelUpToRemote(ctx context.Context, config *configs.Config, client *utils.Client, remotePastelUp string) error {
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
		if err := client.Scp(ctx, localPastelupPath, remotePastelUp, "0777"); err != nil {
			return fmt.Errorf("failed to copy pastelup to remote %s", err)
		}

	} else {
		log.WithContext(ctx).Infof("current OS is not linux, skipping pastelup copy")

		// Download PastelUpExecName from remote and save to remotePastelUp
		downloadURL := fmt.Sprintf("%s/%s/pastelup/%s", constants.DownloadBaseURL, constants.GetVersionSubURL("", ""), constants.PastelUpExecName["Linux"])
		log.WithContext(ctx).Infof("downloading pastelup from %s", downloadURL)

		cmd := fmt.Sprintf("wget %s -O %s", downloadURL, remotePastelUp)
		if _, err := client.Cmd(cmd).Output(); err != nil {
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

func checkHotAndColdNodesNetworkMode(ctx context.Context, client *utils.Client, config *configs.Config) error {
	tempConfFilePath := filepath.Join(config.PastelExecDir, "tempconf")
	defer func() {
		if _, err := os.Stat(tempConfFilePath); errors.Is(err, os.ErrNotExist) {
			log.WithContext(ctx).Error("file does not exist")
		} else {
			err := os.Remove(tempConfFilePath)
			if err != nil {
				log.Error(err)
			}
		}
	}()

	// Copy pastel.conf from remote
	log.WithContext(ctx).Info("copying pastel.conf from remote...")
	remotePastelConfPath := filepath.Join(config.RemoteHotWorkingDir, constants.PastelConfName)
	if err := client.ScpFrom(ctx, remotePastelConfPath, tempConfFilePath); err != nil {
		return fmt.Errorf("failed to copy pastel.conf from remote %s", err)
	}
	log.WithContext(ctx).Info("pastel.conf copied from remote...")

	log.WithContext(ctx).Info("getting network mode from remote.conf...")
	remoteNetwork, err := getNetworkModeFromRemote(ctx, tempConfFilePath)
	if err != nil {
		return fmt.Errorf("failed to parse pastel.conf copied from remote %s", err)
	}
	log.WithContext(ctx).Info("network mode retrieved from remote.conf...")

	if config.Network != remoteNetwork {
		err = errors.New("hot and cold nodes are operating in different network modes")
		log.WithContext(ctx).WithError(err).Errorf("hot node network:%s , cold node network: %s", remoteNetwork, config.Network)
		return err
	}

	return nil
}

func getNetworkModeFromRemote(ctx context.Context, confFilePath string) (remoteNetwork string, err error) {
	log.WithContext(ctx).Info("opening the copied remote.conf...")
	file, err := os.OpenFile(confFilePath, os.O_RDWR, 0644)
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Could not open remote pastel config - %s", confFilePath)
		return remoteNetwork, err
	}
	log.WithContext(ctx).Info("File opened")
	defer file.Close()

	log.WithContext(ctx).Info("Parsing network mode from remote.conf")

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "testnet=") {
			isTestnet, err := strconv.ParseBool(strings.TrimPrefix(line, "testnet="))
			if err != nil {
				log.WithContext(ctx).WithError(err).Errorf("Could not parse testnet= bool from file: - %s", confFilePath)
				return remoteNetwork, err
			}

			if isTestnet {
				log.WithContext(ctx).Infof("remote node is operating in testnet")
				return constants.NetworkTestnet, nil
			}
		}
		if strings.HasPrefix(line, "regtest=") {
			isRegTestnet, err := strconv.ParseBool(strings.TrimPrefix(line, "regtest="))
			if err != nil {
				log.WithContext(ctx).WithError(err).Errorf("Could not parse regtest= bool from file: - %s", confFilePath)
				return remoteNetwork, err
			}

			if isRegTestnet {
				log.WithContext(ctx).Infof("remote node is operating in regtest")
				return constants.NetworkRegTest, nil
			}
		}
	}

	return constants.NetworkMainnet, nil
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
	if err := copyPastelUpToRemote(ctx, config, client, constants.RemotePastelupPath); err != nil {
		log.WithContext(ctx).Errorf("Failed to copy pastelup to remote at %s - %v", constants.RemotePastelupPath, err)
		client.Close()
		return nil, fmt.Errorf("failed to install pastelup at %s - %v", constants.RemotePastelupPath, err)
	}
	log.WithContext(ctx).Info("successfully install pastelup executable to remote host")

	return client, nil
}

func executeRemoteCommandsWithInventory(ctx context.Context, config *configs.Config, commands []string, tryStop bool, needOutput bool) ([][]byte, error) {
	if len(config.InventoryFile) > 0 {
		var inv Inventory
		err := inv.ReadAnsibleYamlInventory(config.InventoryFile)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to load inventory file")
			return nil, err
		}
		outs, err := inv.ExecuteCommands(ctx, config, commands, needOutput)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to execute command on remote host from inventory")
			return nil, err
		}
		return outs, nil
	}
	out, err := executeRemoteCommands(ctx, config, commands, tryStop, needOutput)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to execute command on remote host")
		return nil, err
	}
	return [][]byte{out}, nil
}

func executeRemoteCommands(ctx context.Context, config *configs.Config, commands []string, tryStop bool, needOutput bool) ([]byte, error) {
	// Connect to remote
	client, err := prepareRemoteSession(ctx, config)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to prepare remote session")
		return nil, fmt.Errorf("failed to prepare remote session: %v", err)
	}
	defer client.Close()

	if tryStop {
		if err = checkAndStopRemoteServices(ctx, config, client); err != nil {
			return nil, err
		}
	}

	var outs []byte
	for _, command := range commands {
		if needOutput {
			out, err := client.Cmd(command).Output()
			if err != nil {
				log.WithContext(ctx).WithField("out", string(out)).WithField("cmd", command).
					WithError(err).Error("failed to execute remote command")
				return nil, fmt.Errorf("failed to execute remote command: %s", err.Error())
			}
			outs = append(outs, out...)
		} else {
			err = client.ShellCmd(ctx, command)
			if err != nil {
				log.WithContext(ctx).WithError(err).Error("Failed while executing remote command")
				return nil, err
			}
		}
	}
	return outs, nil
}

func checkAndStopRemoteServices(ctx context.Context, config *configs.Config, client *utils.Client) error {

	checkIfRunningCommand := "ps afx | grep -E 'pasteld|rq-service|dd-service|supernode' | grep -v grep"
	if out, _ := client.Cmd(checkIfRunningCommand).Output(); len(out) != 0 {
		log.WithContext(ctx).Info("Supernode is running on remote host")

		if yes, _ := AskUserToContinue(ctx,
			"Do you want to stop it and continue? Y/N"); !yes {
			log.WithContext(ctx).Warn("Exiting...")
			return fmt.Errorf("user terminated installation")
		}

		log.WithContext(ctx).Info("Stopping supernode services...")
		stopSuperNodeCmd := fmt.Sprintf("%s stop supernode ", constants.RemotePastelupPath)
		err := client.ShellCmd(ctx, stopSuperNodeCmd)
		if err != nil {
			if config.Force {
				log.WithContext(ctx).WithError(err).Warnf("failed to stop supernode: %v", err)
			} else {
				log.WithContext(ctx).WithError(err).Errorf("failed to stop supernode: %v", err)
				return err
			}
		} else {
			log.WithContext(ctx).Info("Supernode stopped")
		}
	}
	return nil
}

func getMasternodeConfPath(config *configs.Config, workDirPath string, fileName string) string {
	var masternodeConfPath string
	if config.Network == constants.NetworkTestnet {
		masternodeConfPath = filepath.Join(workDirPath, "testnet3", fileName)
	} else if config.Network == constants.NetworkRegTest {
		masternodeConfPath = filepath.Join(workDirPath, "regtest", fileName)
	} else {
		masternodeConfPath = filepath.Join(workDirPath, fileName)
	}
	return masternodeConfPath
}

// ReserveSNPorts reserves ports for supernode
func ReserveSNPorts(ctx context.Context, config *configs.Config) {
	portList := GetSNPortList(config)
	cmd := fmt.Sprintf("net.ipv4.ip_local_reserved_ports=%d,%d,%d,%d,%d,%d",
		portList[constants.NodeRPCPort],
		portList[constants.SNPort],
		portList[constants.NodePort],
		portList[constants.P2PPort],
		constants.RQServiceDefaultPort,
		constants.DDServerDefaultPort)
	log.WithContext(ctx).Infof("Running: sysctl -w %s", cmd)
	out, err := RunSudoCMD(config, "sysctl", "-w", cmd)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to reserve ports for SuperNode")
	}
	log.WithContext(ctx).Info(out)
}
