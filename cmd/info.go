package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/cloudfoundry/gosigar"
	"github.com/olekukonko/tablewriter"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pastelnetwork/pastelup/configs"
	"github.com/pastelnetwork/pastelup/constants"
	"github.com/pastelnetwork/pastelup/utils"
)

var (
	flagHostInfo   bool
	flagPastelInfo bool
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

type processInfo struct {
	Path string
}

func setupInfoSubCommand(config *configs.Config,
	infoCmd infoCommand, remote bool,
	f func(context.Context, *configs.Config) error,
) *cli.Command {

	infoFlags := []*cli.Flag{
		cli.NewFlag("host", &flagHostInfo).
			SetUsage(green("Get Host info (Host name, OS version, ")).SetValue(true),
		cli.NewFlag("pastel", &flagPastelInfo).
			SetUsage(green("Get Pastel info (Working Directory, Executables Directory")).SetValue(true),
	}

	remoteFlags := []*cli.Flag{
		cli.NewFlag("ssh-ip", &config.RemoteIP).
			SetUsage(red("Required, SSH address of the remote host")).SetRequired(),
		cli.NewFlag("ssh-port", &config.RemotePort).
			SetUsage(yellow("Optional, SSH port of the remote host, default is 22")).SetValue(22),
		cli.NewFlag("ssh-user", &config.RemoteUser).
			SetUsage(yellow("Optional, Username of user at remote host")),
		cli.NewFlag("ssh-key", &config.RemoteSSHKey).
			SetUsage(yellow("Optional, Path to SSH private key for SSH Key Authentication")),
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
	}

	subCommand := cli.NewCommand(commandName)
	subCommand.SetUsage(cyan(commandMessage))
	subCommand.AddFlags(commandFlags...)

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

func setupInfoCommand() *cli.Command {
	config := configs.InitConfig()

	// Add info command
	infoCommand := setupInfoSubCommand(config, infoLocal, false, runInfoSubCommand)
	infoCommand.AddSubcommands(setupInfoSubCommand(config, infoLocal, true, runRemoteInfoSubCommand))

	return infoCommand
}

func runInfoSubCommand( /*ctx*/ _ context.Context, config *configs.Config) error {

	pastelProcNames := make(map[string]processInfo)
	if flagHostInfo {
		log.Infof(green("=== System info ==="))
		host, _ := os.Hostname()
		log.Infof("HostName: %s", host)
		log.Infof("OS: %s", utils.GetOS())

		for _, tool := range pastelTools {
			pastelProcNames[constants.ServiceName[tool][utils.GetOS()]] = processInfo{}
		}

		getMemoryInfo()
		getPastelProcessesInfo(&pastelProcNames)
	}

	if flagPastelInfo {
		log.Infof(blue("=== Pastel info ==="))
		//for _, tool := range pastelTools {
		//}

		log.Infof("Working Directory: %s", config.WorkingDir)
		log.Infof("Pastel Exec Directory: %s", pastelProcNames[string(constants.PastelD)].Path)
	}

	return nil
}

func runRemoteInfoSubCommand( /*ctx*/ _ context.Context /*config*/, _ *configs.Config) error {
	return nil
}

func format(val uint64) uint64 {
	return val / 1024
}

func getPastelProcessesInfo(pastelProcNames *map[string]processInfo) {
	pids := sigar.ProcList{}
	pids.Get()

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Service name", "Process ID", "CPU %", "Virtual Memory", "Resident Memory", "Starting Time", "Running Time", "Path"})
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

	for _, pid := range pids.List {

		state := sigar.ProcState{}
		if err := state.Get(pid); err != nil {
			continue
		}

		var pinfo processInfo
		found := false
		if pinfo, found = (*pastelProcNames)[state.Name]; !found {
			continue
		}

		exe := sigar.ProcExe{}
		if err := exe.Get(pid); err != nil {
			continue
		}
		pinfo.Path = exe.Cwd
		(*pastelProcNames)[state.Name] = pinfo

		mem := sigar.ProcMem{}
		if err := mem.Get(pid); err != nil {
			continue
		}

		time := sigar.ProcTime{}
		if err := time.Get(pid); err != nil {
			continue
		}

		cpu := sigar.ProcCpu{}
		if err := cpu.Get(pid); err != nil {
			continue
		}

		table.Append([]string{
			state.Name,
			strconv.Itoa(pid),
			strconv.Itoa(int(cpu.Percent)),
			strconv.Itoa(int(mem.Size / 1024)),
			strconv.Itoa(int(mem.Resident / 1024)),
			time.FormatStartTime(),
			time.FormatTotal(),
			exe.Cwd,
		})
	}
	table.Render()
}

func getMemoryInfo() {
	mem := sigar.Mem{}
	swap := sigar.Swap{}

	mem.Get()
	swap.Get()

	fmt.Fprintf(os.Stdout, "%18s %10s %10s\n",
		"total", "used", "free")

	fmt.Fprintf(os.Stdout, "Mem:    %10d %10d %10d\n",
		format(mem.Total), format(mem.Used), format(mem.Free))

	fmt.Fprintf(os.Stdout, "-/+ buffers/cache: %10d %10d\n",
		format(mem.ActualUsed), format(mem.ActualFree))

	fmt.Fprintf(os.Stdout, "Swap:   %10d %10d %10d\n",
		format(swap.Total), format(swap.Used), format(swap.Free))
}
