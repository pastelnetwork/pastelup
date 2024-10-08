package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	sigar "github.com/cloudfoundry/gosigar"
	"github.com/olekukonko/tablewriter"

	"github.com/pastelnetwork/pastelup/common/cli"
	"github.com/pastelnetwork/pastelup/common/log"
	"github.com/pastelnetwork/pastelup/common/sys"
	"github.com/pastelnetwork/pastelup/configs"
	"github.com/pastelnetwork/pastelup/constants"
	"github.com/pastelnetwork/pastelup/services/pastelcore"
	"github.com/pastelnetwork/pastelup/structure"
	"github.com/pastelnetwork/pastelup/utils"
)

var (
	flagHostInfo   bool
	flagPastelInfo bool
	flagOutput     string
)

var (
	pastelTools = []constants.ToolType{
		constants.PastelD,
		constants.SuperNode,
		constants.WalletNode,
		constants.DDService,
		constants.RQService,
		constants.DDImgService,
	}
)

type hostInfo struct {
	Hostname string
	OS       string
}

type processInfo struct {
	Process   string
	Pid       string
	CPU       string
	Virtmem   string
	Rmem      string
	Starttime string
	Runtime   string
	Path      string
	Args      []string
	Version   string
	Hash      string
}

type memoryInfo struct {
	Memory string
	Total  string
	Used   string
	Free   string
}

type filesystemInfo struct {
	Filesystem string
	Size       string
	Used       string
	Avail      string
	Use        string
	Mounted    string
}

type systemInfo struct {
	HostInfo hostInfo
	MemInfo  []memoryInfo
	FsInfo   []filesystemInfo
	ProcInfo []processInfo
}

type pastelInfo struct {
	Args       []string
	WorkingDir string
	ExecDir    string
	GetInfo    structure.GetInfoResult
	MNStatus   structure.MNStatusResult
	MNConfig   structure.MasternodeConfResult
}

type allInfo struct {
	SystemInfo systemInfo
	PastelInfo pastelInfo
}

type infoCommand uint8

const (
	infoLocal infoCommand = iota
	infoRemote
)

var (
	infoCommandName = map[infoCommand]string{
		infoLocal:  "info",
		infoRemote: "remote",
	}
	infoCommandMessage = map[infoCommand]string{
		infoLocal:  "Information about Pastel system and the host",
		infoRemote: "Information about Remote Pastel system and the host",
	}
)

func setupInfoSubCommand(config *configs.Config,
	infoCmd infoCommand, remote bool,
	f func(context.Context, *configs.Config) error,
) *cli.Command {

	infoFlags := []*cli.Flag{
		cli.NewFlag("host", &flagHostInfo).
			SetUsage(green("Get Host info (Host name, OS version, ")).SetValue(true),
		cli.NewFlag("pastel", &flagPastelInfo).
			SetUsage(green("Get Pastel info (Working Directory, Executables Directory")).SetValue(true),
		cli.NewFlag("output", &flagOutput).
			SetUsage(green("How to present information. Available choices are: 'console' and 'json'")).SetValue("console"),
	}

	var dirsFlags []*cli.Flag
	if !remote {
		dirsFlags = []*cli.Flag{
			cli.NewFlag("dir", &config.PastelExecDir).SetAliases("d").
				SetUsage(green("Optional, Location where to create pastel node directory")).SetValue(config.Configurer.DefaultPastelExecutableDir()),
			cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
				SetUsage(green("Optional, Location where to create working directory")).SetValue(config.Configurer.DefaultWorkingDir()),
		}
	}

	remoteFlags := []*cli.Flag{
		cli.NewFlag("ssh-ip", &config.RemoteIP).
			SetUsage(red("Required (if `inventory` is not used), SSH address of the remote host")),
		cli.NewFlag("ssh-port", &config.RemotePort).
			SetUsage(yellow("Optional, SSH port of the remote host, default is 22")).SetValue(22),
		cli.NewFlag("ssh-user", &config.RemoteUser).
			SetUsage(yellow("Optional, Username of user at remote host")),
		cli.NewFlag("ssh-key", &config.RemoteSSHKey).
			SetUsage(yellow("Optional, Path to SSH private key for SSH Key Authentication")),
		cli.NewFlag("inventory", &config.InventoryFile).
			SetUsage(red("Optional, Path to the file with configuration of the remote hosts")),
		cli.NewFlag("in-parallel", &config.AsyncRemote).
			SetUsage(green("Optional, When using inventory file run remote tasks in parallel")),
		cli.NewFlag("filter", &config.InventoryFilter).
			SetUsage(green("Optional, use only specified host groups from the inventory file, comma separated list")),
	}

	var commandName, commandMessage string
	if !remote {
		commandName = infoCommandName[infoCmd]
		commandMessage = infoCommandMessage[infoCmd]
	} else {
		commandName = infoCommandName[infoRemote]
		commandMessage = infoCommandMessage[infoRemote]
	}

	commandFlags := infoFlags
	if remote {
		commandFlags = append(commandFlags, remoteFlags[:]...)
	} else {
		commandFlags = append(commandFlags, dirsFlags[:]...)
	}

	subCommand := cli.NewCommand(commandName)
	subCommand.SetUsage(cyan(commandMessage))
	subCommand.AddFlags(commandFlags...)
	addLogFlags(subCommand, config)

	if f != nil {
		subCommand.SetActionFunc(func(ctx context.Context, _ []string) error {
			ctx, err := configureLogging(ctx, commandMessage, config)
			if err != nil {
				return fmt.Errorf("failed to configure logging option - %v", err)
			}

			sys.RegisterInterruptHandler(func() {
				log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
				os.Exit(0)
			})

			if !remote {
				if err = ParsePastelConf(ctx, config); err != nil {
					return err
				}
			}
			log.WithContext(ctx).Info("Started")
			if err = f(ctx, config); err != nil {
				return err
			}
			log.WithContext(ctx).Info("Finished successfully!")
			return nil
		})
	}

	return subCommand
}

func setupInfoCommand(config *configs.Config) *cli.Command {
	infoCommand := setupInfoSubCommand(config, infoLocal, false, runInfoSubCommand)
	infoCommand.AddSubcommands(setupInfoSubCommand(config, infoLocal, true, runRemoteInfoSubCommand))
	return infoCommand
}

func formatMemory(val uint64) string {
	return strconv.Itoa(int(val / 1024))
}
func formatSize(size uint64) string {
	return sigar.FormatSize(size * 1024)
}

func runInfoSubCommand(ctx context.Context, config *configs.Config) error {
	//var err error
	var sysInfo systemInfo
	if flagHostInfo {
		if flagOutput == "console" {
			log.WithContext(ctx).Info(green("\n=== System info ==="))
		}

		host, _ := os.Hostname()
		sysInfo.HostInfo = hostInfo{
			Hostname: host,
			OS:       string(utils.GetOS()),
		}

		pastelProcNames := make(map[string]bool)
		pastelProcNamesShort := make(map[string]bool)
		for _, tool := range pastelTools {
			name := constants.ServiceName[tool][utils.GetOS()]
			pastelProcNames[name] = true

			short := int(math.Min(15, float64(len(name))))
			shortName := name[:short]
			pastelProcNamesShort[shortName] = true
		}
		// for old SN installations
		pastelProcNames["supernode-ubunt"] = true
		pastelProcNames["rq-service-ubun"] = true

		//dd and img-server
		pastelProcNames["python3"] = true //TODO - get command line parameters and check for `dupe_detection_server.py`
		pastelProcNames["start_dd_img_se"] = true

		sysInfo.MemInfo = getMemoryInfo()
		sysInfo.FsInfo = getFSInfo()
		sysInfo.ProcInfo = getPastelProcessesInfo(&pastelProcNames, &pastelProcNamesShort)

		if flagOutput == "console" {
			printHostInfo(sysInfo.HostInfo)
			printMemoryInfo(sysInfo.MemInfo)
			printFSInfo(sysInfo.FsInfo)
			printProcessInfo(sysInfo.ProcInfo)
		}
	}

	pastelInfo := pastelInfo{}

	if flagPastelInfo {
		if flagOutput == "console" {
			log.WithContext(ctx).Info(blue("\n=== Pastel info ==="))
		}

		for _, process := range sysInfo.ProcInfo {
			if strings.HasPrefix(process.Process, "pasteld") {
				config.PastelExecDir = process.Path
				config.WorkingDir = config.Configurer.DefaultWorkingDir()
				if len(process.Args) > 0 {
					for _, arg := range process.Args[1:] {
						if strings.Contains(arg, "--datadir") {
							datadir := strings.Split(arg, "=")
							if len(datadir) == 1 {
								datadir = strings.Split(arg, " ")
							}
							if len(datadir) == 2 {
								config.WorkingDir = datadir[1]
							}
						}
					}
				}
				pastelInfo.Args = process.Args
				pastelInfo.WorkingDir = config.WorkingDir
				pastelInfo.ExecDir = config.PastelExecDir

				var info structure.RPCGetInfo
				err := pastelcore.NewClient(config).RunCommand(pastelcore.GetInfoCmd, &info)
				if err != nil {
					log.WithContext(ctx).Errorf("unable to get pastel info: %v", err)
				} else {
					pastelInfo.GetInfo = info.Result
				}

				var mnStatus structure.RPCPastelMNStatus
				err = pastelcore.NewClient(config).RunCommandWithArgs(pastelcore.MasterNodeCmd, []string{"status"}, &mnStatus)
				if err != nil {
					log.WithContext(ctx).Errorf("unable to get masternode status: %v", err)
				} else {
					pastelInfo.MNStatus = mnStatus.Result
				}

				var mnConfig structure.RPCMasternodeConf
				err = pastelcore.NewClient(config).RunCommandWithArgs(pastelcore.MasterNodeCmd, []string{"list-conf"}, &mnConfig)
				if err != nil {
					log.WithContext(ctx).Errorf("unable to get masternode config: %v", err)
				} else {
					pastelInfo.MNConfig = mnConfig.Result
				}
			}
		}

		if flagOutput == "console" {
			printPastelInfo(pastelInfo)
		}
	}

	if flagOutput == "json" {
		allInfo := allInfo{
			SystemInfo: sysInfo,
			PastelInfo: pastelInfo,
		}
		//data, _ := json.MarshalIndent(allInfo, "", "  ")
		data, _ := json.Marshal(allInfo)
		//if err != nil {
		//	fmt.Printf("Error %v\n", err)
		//}
		fmt.Println(string(data))
	}
	return nil
}

func runRemoteInfoSubCommand(ctx context.Context, config *configs.Config) error {
	infoOptions := ""
	if flagHostInfo {
		infoOptions = " --host"
	}
	if flagPastelInfo {
		infoOptions = fmt.Sprintf("%s --pastel", infoOptions)
	}
	if len(flagOutput) > 0 {
		infoOptions = fmt.Sprintf("%s --output %s", infoOptions, flagOutput)
	}
	if config.Quiet && flagOutput == "json" {
		infoOptions = fmt.Sprintf("%s -q", infoOptions)
	}
	if len(config.LogLevel) > 0 {
		infoOptions = fmt.Sprintf("%s --log-level %s", infoOptions, config.LogLevel)
	}
	infoCmd := fmt.Sprintf("%s info %s", constants.RemotePastelupPath, infoOptions)
	if outs, err := executeRemoteCommandsWithInventory(ctx, config, []string{infoCmd}, false, flagOutput == "json"); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to get info from remote hosts")
	} else {
		if flagOutput == "json" {
			sliceOfStrings := make([]json.RawMessage, len(outs))
			for i, byteSlice := range outs {
				sliceOfStrings[i] = byteSlice
			}
			jsonData, err := json.MarshalIndent(sliceOfStrings, "", "  ")
			if err != nil {
				fmt.Println("Error:", err)
				log.WithContext(ctx).WithError(err).Error("Failed to format responses as JSON")
				return err
			}
			fmt.Printf("Info from remote hosts: %s\n", string(jsonData))
		}
	}
	return nil
}

func printHostInfo(info hostInfo) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"", ""})
	table.SetColumnColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgBlackColor},
	)
	table.Append([]string{
		"Hostname",
		info.Hostname,
	})
	table.Append([]string{
		"OS",
		info.OS,
	})
	table.Render()
}

func calcFileHash(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	hashInBytes := hash.Sum(nil)[:20]
	return hex.EncodeToString(hashInBytes), nil
}

func getPastelProcessesInfo(procNames *map[string]bool, procNamesShort *map[string]bool) []processInfo {
	pids := sigar.ProcList{}
	pids.Get()

	var procInfo []processInfo

	for _, pid := range pids.List {
		state := sigar.ProcState{}
		if err := state.Get(pid); err != nil {
			continue
		}
		found := false
		if _, found = (*procNames)[state.Name]; !found {
			if _, found = (*procNamesShort)[state.Name]; !found {
				continue
			}
		}

		//fmt.Printf("%s\n", state.Name)

		var dir, file, vmem, rmem, stime, rtime, cpup string
		var pasteldArgs []string

		if strings.HasPrefix(state.Name, "python") ||
			strings.HasPrefix(state.Name, "start_dd_img_se") ||
			strings.HasPrefix(state.Name, "pasteld") {
			args := sigar.ProcArgs{}
			args.Get(pid)
			if strings.HasPrefix(state.Name, "pasteld") {
				pasteldArgs = args.List
				goto foundDDorISorPD
			}
			for _, arg := range args.List {
				if strings.Contains(arg, "dupe_detection_server.py") ||
					strings.Contains(arg, "start_dd_img_server.sh") {
					dir, file = filepath.Split(arg)
					goto foundDDorISorPD
				}
			}
			continue
		}

	foundDDorISorPD:
		if len(dir) == 0 && len(file) == 0 {
			file = state.Name
			exe := sigar.ProcExe{}
			if err := exe.Get(pid); err == nil {
				dir, file = filepath.Split(exe.Name)
			}
		}

		mem := sigar.ProcMem{}
		if err := mem.Get(pid); err == nil {
			vmem = strconv.Itoa(int(mem.Size / 1024))
			rmem = strconv.Itoa(int(mem.Resident / 1024))
		}
		time := sigar.ProcTime{}
		if err := time.Get(pid); err == nil {
			stime = time.FormatStartTime()
			rtime = time.FormatTotal()
		}
		cpu := sigar.ProcCpu{}
		if err := cpu.Get(pid); err == nil {
			cpup = strconv.Itoa(int(cpu.Percent))
		}

		hash, err := calcFileHash(dir + file)
		if err != nil {
			log.WithContext(context.Background()).WithError(err).Errorf("Failed to calculate hash of %s", dir+file)
		}

		var version = ""
		//if file != "dupe_detection_server.py" && file != "start_dd_img_server.sh" {
		//	version, err := RunCMD(dir+file, "--version")
		//	if err != nil {
		//		log.WithContext(context.Background()).WithError(err).Errorf("Failed to get version of %s", dir+file)
		//	}
		//	if file == "pasteld" || file == "pastel-cli" {
		//		lines := strings.SplitN(version, "\n", 2)
		//		// Extract the first line
		//		if len(lines) > 0 {
		//			version = lines[0]
		//		} else {
		//			version = ""
		//		}
		//	}
		//}

		procInfo = append(procInfo, processInfo{
			Pid:       strconv.Itoa(pid),
			Process:   file,
			Path:      dir,
			Virtmem:   vmem,
			Rmem:      rmem,
			Starttime: stime,
			Runtime:   rtime,
			CPU:       cpup,
			Args:      pasteldArgs,
			Hash:      hash,
			Version:   version,
		})
	}
	return procInfo
}
func printProcessInfo(info []processInfo) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Process", "Pid", "CPU%", "VirtMem", "RMem", "StartTime", "RunTime", "Path", "Version", "Hash"})
	table.SetColumnColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgBlackColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgBlackColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgBlackColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgBlackColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgBlackColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgBlackColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgBlackColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgBlackColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgBlackColor},
	)
	for _, process := range info {
		table.Append([]string{
			process.Process,
			process.Pid,
			process.CPU,
			process.Virtmem,
			process.Rmem,
			process.Starttime,
			process.Runtime,
			process.Path,
			process.Version,
			process.Hash,
		})
	}
	table.Render()
}

func getMemoryInfo() []memoryInfo {
	mem := sigar.Mem{}
	mem.Get()

	swap := sigar.Swap{}
	swap.Get()

	return []memoryInfo{
		{
			Memory: "RAM",
			Total:  formatMemory(mem.Total),
			Used:   formatMemory(mem.Used),
			Free:   formatMemory(mem.Free),
		},
		{
			Memory: "-/+ buffers/cache",
			Total:  "",
			Used:   formatMemory(mem.ActualUsed),
			Free:   formatMemory(mem.ActualFree),
		},
		{
			Memory: "Swap",
			Total:  formatMemory(swap.Total),
			Used:   formatMemory(swap.Used),
			Free:   formatMemory(swap.Free),
		},
	}
}
func printMemoryInfo(info []memoryInfo) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Memory", "Total", "Used", "Free"})
	table.SetColumnColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiGreenColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgWhiteColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgWhiteColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgWhiteColor},
	)
	for _, memInfo := range info {
		table.Append([]string{
			memInfo.Memory,
			memInfo.Total,
			memInfo.Used,
			memInfo.Free,
		})
	}
	table.Render()
}

func getFSInfo() []filesystemInfo {
	fsList := sigar.FileSystemList{}
	fsList.Get()

	var fsInfo []filesystemInfo
	for _, fs := range fsList.List {

		if strings.HasPrefix(fs.DevName, "/dev/loop") ||
			!strings.HasPrefix(fs.DevName, "/dev/") {
			continue
		}

		dirName := fs.DirName

		usage := sigar.FileSystemUsage{}
		usage.Get(dirName)

		fsInfo = append(fsInfo, filesystemInfo{
			Filesystem: fs.DevName,
			Size:       formatSize(usage.Total),
			Used:       formatSize(usage.Used),
			Avail:      formatSize(usage.Avail),
			Use:        sigar.FormatPercent(usage.UsePercent()),
			Mounted:    dirName,
		})
	}
	return fsInfo
}
func printFSInfo(info []filesystemInfo) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Filesystem", "Size", "Used", "Avail", "Use%", "Mounted on"})
	table.SetColumnColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiGreenColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgWhiteColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgWhiteColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgWhiteColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgWhiteColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgWhiteColor},
	)
	for _, fs := range info {
		table.Append([]string{
			fs.Filesystem,
			fs.Size,
			fs.Used,
			fs.Avail,
			fs.Use,
			fs.Mounted,
		})
	}
	table.Render()
}

func printPastelInfo(info pastelInfo) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetRowLine(true)
	table.SetHeader([]string{"", ""})
	table.SetColumnColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiGreenColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgWhiteColor},
	)

	cmdArgs := ""
	if len(info.Args) > 0 {
		for _, arg := range info.Args[1:] {
			cmdArgs = fmt.Sprintf("%s\t%s\n", cmdArgs, arg)
		}
	}
	table.Append([]string{
		"Command line arguments",
		cmdArgs,
	})
	table.Append([]string{
		"Working Directory",
		info.WorkingDir,
	})
	table.Append([]string{
		"Installation Directory",
		info.ExecDir,
	})
	table.Render()
	fmt.Printf("Blockchain info:\n%s\n", info.GetInfo)
	fmt.Printf("Masternode status:\n%s\n", info.MNStatus)
	fmt.Printf("Masternode config:\n%s\n", info.MNConfig)
}
