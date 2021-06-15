package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/pastelnetwork/gonode/common/cli"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/pastel-utility/configurer"
	"github.com/pastelnetwork/pastel-utility/structure"
)

var (
	errSubCommandRequired     = fmt.Errorf("subcommand is required")
	errMasterNodeNameRequired = fmt.Errorf("required --name, name of the Masternode to start and create in the masternode.conf if `--create` or `--update` are specified")
	errMasterNodeTxIDRequired = fmt.Errorf("required --txid, transaction id of 5M collateral MN payment")
	errMasterNodeINDRequired  = fmt.Errorf("required --ind, output index in the transaction of 5M collateral MN payment")
	errMasterNodePwdRequired  = fmt.Errorf("required --passphrase <passphrase to pastelid private key>, if --pastelid is omitted")
	errSetTestnet             = fmt.Errorf("please initialize pastel.conf as testnet mode")
	errSetMainnet             = fmt.Errorf("please initialize pastel.conf as mainnet mode")
	errGetExternalIP          = fmt.Errorf("cannot get external ip address")
)

var (
	flagInteractiveMode bool
	flagRestart         bool

	flagMasterNodeName       string
	flagMasterNodeIsTestNet  bool
	flagMasterNodeIsCreate   bool
	flagMasterNodeIsUpdate   bool
	flagMasterNodeTxID       string
	flagMasterNodeIND        string
	flagMasterNodeIP         string
	flagMasterNodePort       int
	flagMasterNodePrivateKey string
	flagMasterNodePastelID   string
	flagMasterNodePassPhrase string
	flagMasterNodeRPCIP      string
	flagMasterNodeRPCPort    int
	flagMasterNodeP2PIP      string
	flagMasterNodeP2PPort    int
)

func setupStartCommand() *cli.Command {
	config := configs.New()

	startCommand := cli.NewCommand("start")
	startCommand.SetUsage("usage")
	addLogFlags(startCommand, config)

	superNodeSubcommand := cli.NewCommand("supernode")
	superNodeSubcommand.CustomHelpTemplate = GetColoredSubCommandHeaders()
	superNodeSubcommand.SetUsage(cyan("Starts supernode"))
	superNodeSubcommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "superNodeSubCommand", config)
		if err != nil {
			return err
		}
		return runStartSuperNodeSubCommand(ctx, config)
	})
	superNodeFlags := []*cli.Flag{
		cli.NewFlag("i", &flagInteractiveMode),
		cli.NewFlag("r", &flagRestart),
	}
	superNodeSubcommand.AddFlags(superNodeFlags...)

	masterNodeSubCommand := cli.NewCommand("masternode")
	masterNodeSubCommand.CustomHelpTemplate = GetColoredSubCommandHeaders()
	masterNodeSubCommand.SetUsage(cyan("Starts master node"))
	masterNodeSubCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "masterNodeSubCommand", config)
		if err != nil {
			return err
		}
		return runStartMasterNodeSubCommand(ctx, config)
	})
	masterNodeFlags := []*cli.Flag{
		cli.NewFlag("i", &flagInteractiveMode),
		cli.NewFlag("r", &flagRestart),
		cli.NewFlag("name", &flagMasterNodeName).SetUsage("name of the Master node").SetRequired(),
		cli.NewFlag("testnet", &flagMasterNodeIsTestNet),
		cli.NewFlag("create", &flagMasterNodeIsCreate),
		cli.NewFlag("update", &flagMasterNodeIsUpdate),
		cli.NewFlag("txid", &flagMasterNodeTxID),
		cli.NewFlag("ind", &flagMasterNodeIND),
		cli.NewFlag("ip", &flagMasterNodeIP),
		cli.NewFlag("port", &flagMasterNodePort),
		cli.NewFlag("pkey", &flagMasterNodePrivateKey),
		cli.NewFlag("pastelid", &flagMasterNodePastelID),
		cli.NewFlag("passphrase", &flagMasterNodePassPhrase),
		cli.NewFlag("rpc-ip", &flagMasterNodeRPCIP),
		cli.NewFlag("rpc-port", &flagMasterNodeRPCPort),
		cli.NewFlag("p2p-ip", &flagMasterNodeP2PIP),
		cli.NewFlag("p2p-port", &flagMasterNodeP2PPort),
	}
	masterNodeSubCommand.AddFlags(masterNodeFlags...)

	nodeSubCommand := cli.NewCommand("node")
	nodeSubCommand.CustomHelpTemplate = GetColoredSubCommandHeaders()
	nodeSubCommand.SetUsage(cyan("Starts specified node"))
	nodeSubCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "nodeSubCommand", config)
		if err != nil {
			return err
		}
		return runStartNodeSubCommand(ctx, config)
	})
	nodeFlags := []*cli.Flag{
		cli.NewFlag("i", &flagInteractiveMode),
		cli.NewFlag("r", &flagRestart),
	}
	nodeSubCommand.AddFlags(nodeFlags...)

	walletSubCommand := cli.NewCommand("wallet")
	walletSubCommand.CustomHelpTemplate = GetColoredSubCommandHeaders()
	walletSubCommand.SetUsage(cyan("Starts wallet"))
	walletSubCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "nodeSubCommand", config)
		if err != nil {
			return err
		}
		return runStartWalletSubCommand(ctx, config)
	})
	walletFlags := []*cli.Flag{
		cli.NewFlag("i", &flagInteractiveMode),
		cli.NewFlag("r", &flagRestart),
	}
	walletSubCommand.AddFlags(walletFlags...)

	startCommand.AddSubcommands(
		superNodeSubcommand,
		masterNodeSubCommand,
		nodeSubCommand,
		walletSubCommand,
	)

	startCommand.SetActionFunc(func(ctx context.Context, args []string) error {
		ctx, err := configureLogging(ctx, "startcommand", config)
		if err != nil {
			return err
		}
		if len(args) == 0 {
			return errSubCommandRequired
		}
		return runStart(ctx, config)
	})
	return startCommand
}

func runStart(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Start")
	defer log.WithContext(ctx).Info("End")

	configJSON, err := config.String()
	if err != nil {
		return err
	}
	log.WithContext(ctx).Infof("Config: %s", configJSON)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sys.RegisterInterruptHandler(cancel, func() {
		log.WithContext(ctx).Info("Interrupt signal received. Gracefully shutting down...")
		os.Exit(0)
	})

	// actions to run goes here

	return nil
}

func runStartNodeSubCommand(_ context.Context, _ *configs.Config) error {
	// TODO: Implement start node command
	panic("")
}

func runStartSuperNodeSubCommand(_ context.Context, _ *configs.Config) error {
	// TODO: Implement start supper node command
	panic("")
}

func runStartMasterNodeSubCommand(ctx context.Context, config *configs.Config) error {
	// check master node name

	var masternodePrivKey, pastelid, output string
	var err error

	if err := checkStartMasterNodeParams(ctx, config); err != nil {
		return err
	}

	if err := CheckPastelConf(config); err != nil {
		return err
	}
	// If create master node using HOT/HOT wallet
	if flagMasterNodeIsCreate || flagMasterNodeIsUpdate {

		if flagMasterNodeIsCreate {
			if err = backupConfFile(); err != nil { // delete conf file
				return err
			}

			go RunPasteld(fmt.Sprintf("--externalip=%s", flagMasterNodeIP), "--reindex", "--daemon")

			var failCnt = 0
			for {
				if output, err = runPastelCLI("getaccountaddress", ""); err != nil {
					fmt.Printf("Waiting the pasteld to be started ...\n")
					time.Sleep(10000 * time.Millisecond)
					failCnt++
					if failCnt == 10 {
						fmt.Printf("Can not start with pasteld\n")
						return err
					}
				} else {
					fmt.Printf("Hot wallet address = %s\n", output)
					break
				}
			}

			if len(flagMasterNodePrivateKey) == 0 {
				if masternodePrivKey, err = runPastelCLI("masternode", "genkey"); err != nil {
					return err
				}
			} else {
				masternodePrivKey = flagMasterNodePrivateKey
			}
			fmt.Printf("masternode private key = %s\n", masternodePrivKey)
			if _, err = runPastelCLI("stop"); err != nil {
				return err
			}
			time.Sleep(2000 * time.Millisecond)

			// Restart pasteld as a masternode
			go RunPasteld("-masternode", "-txindex=1", "-reindex", fmt.Sprintf("-masternodeprivkey=%s", masternodePrivKey), fmt.Sprintf("--externalip=%s", flagMasterNodeIP))

			if len(flagMasterNodePastelID) == 0 && len(flagMasterNodePassPhrase) != 0 {
				// Check masternode status
				var mnstatus structure.RPCPastelMSStatus

				for {
					if output, err = runPastelCLI("mnsync", "status"); err != nil {
						fmt.Printf("Waiting the pasteld to be started ...\n")
						time.Sleep(10000 * time.Millisecond)
						failCnt++
						if failCnt >= 10 {
							fmt.Printf("Can not start with pasteld\n")
							return err
						}
					} else {
						// Master Node Output
						if err = json.Unmarshal([]byte(output), &mnstatus); err != nil {
							return err
						}
						if mnstatus.AssetName == "Initial" {
							if output, err = runPastelCLI("mnsync", "reset"); err != nil {
								fmt.Printf("master node reset was failed\n")
								return err
							}
							time.Sleep(10000 * time.Millisecond)
						} else {
							if mnstatus.IsSynced == true {
								fmt.Printf("master node was synced!\n")
								break
							} else {
								fmt.Printf("master node was not synced!!!\nWaiting for sync...")
								time.Sleep(10000 * time.Millisecond)
							}
						}
					}
				}

				if output, err = runPastelCLI("pastelid", "newkey", flagMasterNodePassPhrase); err != nil {
					return err
				} // generate a PastelID
				var pastelidSt structure.RPCPastelID
				if err = json.Unmarshal([]byte(output), &pastelidSt); err != nil {
					return err
				}
				pastelid = pastelidSt.Pastelid
			} else {
				pastelid = flagMasterNodePastelID
			}

			fmt.Printf("pastelid = %s\n", pastelid)

			failCnt = 0

			for {
				if output, err = runPastelCLI("getaccountaddress", ""); err != nil {
					fmt.Printf("Waiting the pasteld to be started ...\n")
					time.Sleep(10000 * time.Millisecond)
					failCnt++
					if failCnt == 10 {
						fmt.Printf("Can not start with pasteld\n")
						return err
					}
				} else {
					fmt.Printf("master node address = %s\n", output)
					break
				}
			}

			for {
				if output, err = runPastelCLI("masternode", "outputs"); err != nil {
					fmt.Printf("masternode outputs\n")
					return err
				}
				var recMasterNode map[string]interface{}
				json.Unmarshal([]byte(output), &recMasterNode)

				if len(recMasterNode) != 0 {
					if recMasterNode[flagMasterNodeTxID] != nil && recMasterNode[flagMasterNodeTxID] == flagMasterNodeIND {
						// if receives PSL go to next step
						fmt.Printf("masternode outputs = %s, %s\n", flagMasterNodeTxID, flagMasterNodeIND)
						break
					}
				}

				time.Sleep(10000 * time.Millisecond)
			}

			// Make masternode conf data
			confData := map[string]interface{}{
				flagMasterNodeName: map[string]string{
					"mnAddress":  flagMasterNodeIP + ":" + fmt.Sprintf("%d", flagMasterNodePort),
					"mnPrivKey":  masternodePrivKey,
					"txid":       flagMasterNodeTxID,
					"outIndex":   flagMasterNodeIND,
					"extAddress": flagMasterNodeIP + ":" + fmt.Sprintf("%d", flagMasterNodeRPCPort),
					"p2pAddress": flagMasterNodeP2PIP + ":" + fmt.Sprintf("%d", flagMasterNodeP2PPort),
					"extCfg":     "",
					"extKey":     pastelid,
				},
			}
			data, _ := json.Marshal(confData)

			// Create masternode.conf file
			if err = createConfFile(data); err != nil {
				return err
			}
			fmt.Println(string(data))

			if _, err = runPastelCLI("stop"); err != nil {
				return err
			}
		}

		if flagMasterNodeIsUpdate {
			// Make masternode conf data
			confData := map[string]interface{}{
				flagMasterNodeName: map[string]string{
					"mnAddress":  flagMasterNodeIP + ":" + fmt.Sprintf("%d", flagMasterNodePort),
					"mnPrivKey":  masternodePrivKey,
					"txid":       flagMasterNodeTxID,
					"outIndex":   flagMasterNodeIND,
					"extAddress": flagMasterNodeIP + ":" + fmt.Sprintf("%d", flagMasterNodeRPCPort),
					"p2pAddress": flagMasterNodeP2PIP + ":" + fmt.Sprintf("%d", flagMasterNodeP2PPort),
					"extCfg":     "",
					"extKey":     pastelid,
				},
			}

			// Create masternode.conf file
			if _, err = updateMasternodeConfFile(confData); err != nil {
				return err
			}
		}
	}

	// Get conf data from masternode.conf File
	var nodeName, privKey, extIP = getStartInfo()

	// Start Node as Masternode
	go RunPasteld("-masternode", "-txindex=1", "-reindex", fmt.Sprintf("-masternodeprivkey=%s", privKey), fmt.Sprintf("--externalip=%s", extIP))

	var mnstatus structure.RPCPastelMSStatus
	var failCnt = 0

	for {
		if output, err = runPastelCLI("mnsync", "status"); err != nil {
			fmt.Printf("Waiting the pasteld to be started ...\n")
			time.Sleep(10000 * time.Millisecond)
			failCnt++
			if failCnt >= 10 {
				fmt.Printf("Can not start with pasteld\n")
				return err
			}
		} else {
			// Master Node Output
			if err = json.Unmarshal([]byte(output), &mnstatus); err != nil {
				return err
			}

			if mnstatus.AssetName == "Initial" {
				if output, err = runPastelCLI("mnsync", "reset"); err != nil {
					fmt.Printf("master node reset was failed\n")
					return err
				}
				time.Sleep(10000 * time.Millisecond)
			}
			if mnstatus.IsSynced == true {
				fmt.Printf("master node was synced!\n")
				break
			}
			fmt.Printf("master node was not synced!!!\nWaiting for sync...")
			time.Sleep(10000 * time.Millisecond)
		}
	}

	// Enable Masternode
	failCnt = 0
	for {
		if output, err = runPastelCLI("masternode", "start-alias", nodeName); err != nil {
			fmt.Printf("Waiting the pasteld to be started ...\n")
			time.Sleep(10000 * time.Millisecond)
			failCnt++
			if failCnt == 10 {
				return err
			}
		} else {
			fmt.Printf("The pasteld was started successfully...\n")
			fmt.Printf("masternode alias status = %s\n", output)
			break
		}

	}

	return nil
}

func runStartWalletSubCommand(_ context.Context, _ *configs.Config) error {
	// TODO: Implement wallet command
	panic("")
}

func checkStartMasterNodeParams(_ context.Context, _ *configs.Config) error {
	if len(flagMasterNodeName) == 0 {
		return errMasterNodeNameRequired
	}

	if flagMasterNodeIsCreate || flagMasterNodeIsUpdate {
		if len(flagMasterNodeTxID) == 0 {
			return errMasterNodeTxIDRequired
		}

		if len(flagMasterNodeIND) == 0 {
			return errMasterNodeINDRequired
		}

		if len(flagMasterNodeIP) == 0 {
			externalIP, err := GetExternalIPAddress()

			if err != nil {
				return errGetExternalIP
			}
			flagMasterNodeIP = externalIP
		}

		if len(flagMasterNodePastelID) == 0 {
			if len(flagMasterNodePassPhrase) == 0 {
				return errMasterNodePwdRequired
			}
		}
	}

	flagMasterNodeRPCIP = func() string {
		if len(flagMasterNodeRPCIP) == 0 {
			return flagMasterNodeIP
		}
		return flagMasterNodeRPCIP
	}()
	flagMasterNodeP2PIP = func() string {
		if len(flagMasterNodeP2PIP) == 0 {
			return flagMasterNodeIP
		}
		return flagMasterNodeP2PIP
	}()

	if flagMasterNodeIsTestNet {
		flagMasterNodePort = func() int {
			if flagMasterNodePort == 0 {
				return 19933
			}
			return flagMasterNodePort
		}()
		flagMasterNodeRPCPort = func() int {
			if flagMasterNodeRPCPort == 0 {
				return 14444
			}
			return flagMasterNodeRPCPort
		}()
		flagMasterNodeP2PPort = func() int {
			if flagMasterNodeP2PPort == 0 {
				return 14445
			}
			return flagMasterNodeP2PPort
		}()
	} else {
		flagMasterNodePort = func() int {
			if flagMasterNodePort == 0 {
				return 9933
			}
			return flagMasterNodePort
		}()
		flagMasterNodeRPCPort = func() int {
			if flagMasterNodeRPCPort == 0 {
				return 4444
			}
			return flagMasterNodeRPCPort
		}()
		flagMasterNodeP2PPort = func() int {
			if flagMasterNodeP2PPort == 0 {
				return 4445
			}
			return flagMasterNodeP2PPort
		}()
	}
	return nil
}

// GetExternalIPAddress runs shell command and returns external IP address
func GetExternalIPAddress() (externalIP string, err error) {
	return RunCMD("curl", "ipinfo.io/ip")
}

// RunPasteld runs pasteld
func RunPasteld(args ...string) (output string, err error) {
	if flagMasterNodeIsTestNet {
		args = append(args, "--testnet")
		output, err = RunCMD("./pasteld", args...)
	} else {
		output, err = RunCMD("./pasteld", args...)
	}
	return output, err
}

// Run pastel-cli
func runPastelCLI(args ...string) (output string, err error) {
	return RunCMD("./pastel-cli", args...)
}

// Create or Update masternode.conf File
func createConfFile(confData []byte) (err error) {
	workDirPath := configurer.DefaultWorkingDir()
	var masternodeConfPath, masternodeConfPathBackup string

	if flagMasterNodeIsTestNet {
		masternodeConfPath = workDirPath + "/testnet3/masternode.conf"
		masternodeConfPathBackup = workDirPath + "/testnet3/masternode_%s.conf"
	} else {
		masternodeConfPath = workDirPath + "/masternode.conf"
		masternodeConfPathBackup = workDirPath + "/masternode_%s.conf"
	}
	if _, err := os.Stat(masternodeConfPath); err == nil { // if masternode.conf File exists , backup
		oldFileName := masternodeConfPath
		currentTime := time.Now()
		backupFileName := fmt.Sprintf(masternodeConfPathBackup, currentTime.Format("2021-01-01 23:59:59"))
		err := os.Rename(oldFileName, backupFileName)
		if err != nil {
			return err
		}

	}

	confFile, err := os.Create(masternodeConfPath)
	confFile.Write(confData)
	if err != nil {
		return err
	}
	defer confFile.Close()

	return nil
}

func updateMasternodeConfFile(confData map[string]interface{}) (result bool, err error) {
	workDirPath := configurer.DefaultWorkingDir()
	var masternodeConfPath string

	if flagMasterNodeIsTestNet {
		masternodeConfPath = workDirPath + "/testnet3/masternode.conf"
	} else {
		masternodeConfPath = workDirPath + "/masternode.conf"
	}

	// Read ConfData from masternode.conf
	confFile, err := ioutil.ReadFile(masternodeConfPath)
	if err != nil {
		return false, err
	}

	var conf map[string]interface{}

	json.Unmarshal([]byte(confFile), &conf)

	for k := range confData {
		if conf[k] != nil {
			confDataValue := confData[k].(map[string]string)
			confValue := conf[k].(map[string]interface{})
			for itemKey := range confDataValue {
				if len(confDataValue[itemKey]) != 0 {
					confValue[itemKey] = confDataValue[itemKey]

				}
			}
		}
	}

	var updatedConf []byte
	if updatedConf, err = json.Marshal(conf); err != nil {
		fmt.Printf("updated conf = %s", updatedConf)
		return false, err
	}

	if ioutil.WriteFile(masternodeConfPath, updatedConf, 0644) != nil {
		return false, err
	}

	return true, nil
}

func backupConfFile() (err error) {
	workDirPath := configurer.DefaultWorkingDir()
	var masternodeConfPath, masternodeConfPathBackup string

	if flagMasterNodeIsTestNet {
		masternodeConfPath = workDirPath + "/testnet3/masternode.conf"
		masternodeConfPathBackup = workDirPath + "/testnet3/masternode_%s.conf"
	} else {
		masternodeConfPath = workDirPath + "/masternode.conf"
		masternodeConfPathBackup = workDirPath + "/masternode_%s.conf"
	}
	if _, err := os.Stat(masternodeConfPath); err == nil { // if masternode.conf File exists , backup
		oldFileName := masternodeConfPath
		currentTime := time.Now()
		backupFileName := fmt.Sprintf(masternodeConfPathBackup, currentTime.Format("2021-01-01 23:59:59"))
		if err := os.Rename(oldFileName, backupFileName); err != nil {
			return err
		}

	}
	if _, err := os.Stat(masternodeConfPath); err == nil { // if masternode.conf File exists , backup
		if err = os.Remove(masternodeConfPath); err != nil {
			return err
		}

	}

	return nil
}

func getStartInfo() (nodeName string, privKey string, extIP string) {
	workDirPath := configurer.DefaultWorkingDir()
	var masternodeConfPath string

	if flagMasterNodeIsTestNet {
		masternodeConfPath = workDirPath + "/testnet3/masternode.conf"
	} else {
		masternodeConfPath = workDirPath + "/masternode.conf"
	}

	// Read ConfData from masternode.conf
	confFile, err := ioutil.ReadFile(masternodeConfPath)
	if err != nil {
		return "", "", ""
	}

	var conf map[string]interface{}
	json.Unmarshal([]byte(confFile), &conf)

	for key := range conf {
		nodeName = key // get Node Name
		fmt.Println(key)
	}
	confData := conf[nodeName].(map[string]interface{})
	extAddr := strings.Split(confData["mnAddress"].(string), ":") // get Ext IP
	fmt.Println(extAddr[0])
	return nodeName, confData["mnPrivKey"].(string), extAddr[0]

}

// CheckPastelConf check configuration of pastel settings.
func CheckPastelConf(_ *configs.Config) (err error) {
	workDirPath := configurer.DefaultWorkingDir()

	if _, err := os.Stat(workDirPath); os.IsNotExist(err) {
		return err
	}

	if _, err := os.Stat(workDirPath + "/pastel.conf"); os.IsNotExist(err) {
		return err
	}

	if flagMasterNodeIsTestNet {
		var file, err = os.OpenFile(workDirPath+"/pastel.conf", os.O_RDWR, 0644)
		if err != nil {
			return err
		}
		defer file.Close()

		configure, err := ioutil.ReadAll(file)

		if err != nil {
			return err
		}

		if !strings.Contains(string(configure), "testnet=1") {
			return errSetTestnet
		}
	} else {
		var file, err = os.OpenFile(workDirPath+"/pastel.conf", os.O_RDWR, 0644)
		if err != nil {
			return err
		}
		defer file.Close()

		configure, err := ioutil.ReadAll(file)
		if err != nil {
			return err
		}

		if strings.Contains(string(configure), "testnet=1") {
			return errSetMainnet
		}
	}

	return nil
}
