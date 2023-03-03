package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	sigar "github.com/cloudfoundry/gosigar"
	"github.com/olekukonko/tablewriter"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
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
	MemInfo  []memoryInfo
	FsInfo   []filesystemInfo
	ProcInfo []processInfo
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
		cli.NewFlag("filter", &config.InventoryFilter).
			SetUsage(green("Optional, use only specified host groups from the inventory file, comma separated list")),
		cli.NewFlag("release", &config.Version).SetAliases("r").
			SetUsage(green("Optional, Version of pastelup to download to remote " +
				"host if different local and remote OS's")),
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

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			sys.RegisterInterruptHandler(cancel, func() {
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
		fmt.Println(green("\n=== System info ==="))
		host, _ := os.Hostname()
		fmt.Printf("HostName: %s\n", host)
		fmt.Printf("OS: %s\n", utils.GetOS())

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
			printMemoryInfo(sysInfo.MemInfo)
			printFSInfo(sysInfo.FsInfo)
			printProcessInfo(sysInfo.ProcInfo)
		}
	}

	if flagPastelInfo {
		fmt.Println(blue("\n=== Pastel info ==="))
		for _, process := range sysInfo.ProcInfo {
			if strings.HasPrefix(process.Process, "pasteld") {
				config.WorkingDir = config.Configurer.DefaultWorkingDir()
				if len(process.Args) > 0 {
					fmt.Printf(red("pasteld") + " was started with the following parameters:\n")
					for _, arg := range process.Args[1:] {
						fmt.Printf(cyan("\t%s\n"), arg)
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
				} else {
					fmt.Print(blue("pasteld was started without parameters\n"))
				}
				config.PastelExecDir = process.Path

				fmt.Printf("Blockchain info on the host:\n")
				var info structure.RPCGetInfo
				err := pastelcore.NewClient(config).RunCommand(pastelcore.GetInfoCmd, &info)
				if err != nil {
					log.WithContext(ctx).Errorf("unable to get pastel info: %v", err)
				}
				fmt.Println(info.String() + "\n")

				fmt.Printf("Masternode status of the host:\n")
				var mnStatus structure.RPCPastelMNStatus
				err = pastelcore.NewClient(config).RunCommandWithArgs(pastelcore.MasterNodeCmd, []string{"status"}, &mnStatus)
				if err != nil {
					log.WithContext(ctx).Errorf("unable to get masternode status: %v", err)
				}
				fmt.Printf("%+v\n", mnStatus)
			}
		}
		fmt.Printf("Working Directory: %s\n", config.WorkingDir)
	}

	if flagOutput == "json" {
		data, _ := json.Marshal(sysInfo)
		//if err != nil {
		//	fmt.Printf("Error %v\n", err)
		//}
		fmt.Printf("%s\n", string(data))
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
	if config.Quiet {
		infoOptions = fmt.Sprintf("%s -q", infoOptions)
	}
	if len(config.LogLevel) > 0 {
		infoOptions = fmt.Sprintf("%s --log-level %s", infoOptions, config.LogLevel)
	}
	infoCmd := fmt.Sprintf("%s info %s", constants.RemotePastelupPath, infoOptions)
	if err := executeRemoteCommandsWithInventory(ctx, config, []string{infoCmd}, false); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to get info from remote hosts")
	}
	return nil
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
		})
	}
	return procInfo
}
func printProcessInfo(info []processInfo) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Process", "Pid", "CPU%", "VirtMem", "RMem", "StartTime", "RunTime", "Path"})
	table.SetColumnColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
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
