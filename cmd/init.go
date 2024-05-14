package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/pastelnetwork/pastelup/common/cli"
	"github.com/pastelnetwork/pastelup/common/log"
	"github.com/pastelnetwork/pastelup/configs"
	"github.com/pastelnetwork/pastelup/constants"
)

type initCommand uint8

const (
	superNodeInit initCommand = iota
	coldHotInit
	remoteInit
)

var (
	initCmdName = map[initCommand]string{
		superNodeInit: "supernode",
		coldHotInit:   "coldhot",
		remoteInit:    "remote",
	}
	initCmdMessage = map[initCommand]string{
		superNodeInit: "Initialise local Supernode",
		coldHotInit:   "Initialise Supernode in Cold/Hot mode",
		remoteInit:    "Initialise remote Supernode",
	}
)

type masterNodeConf struct {
	MnAddress  string `json:"mnAddress"`
	MnPrivKey  string `json:"mnPrivKey"`
	Txid       string `json:"txid"`
	OutIndex   string `json:"outIndex"`
	ExtAddress string `json:"extAddress"`
	ExtP2P     string `json:"extP2P"`
	ExtCfg     string `json:"extCfg"`
	ExtKey     string `json:"extKey"`
}

func setupInitSubCommand(config *configs.Config,
	initCommand initCommand, remote bool,
	f func(context.Context, *configs.Config) error,
) *cli.Command {

	commonFlags := []*cli.Flag{
		cli.NewFlag("ip", &config.NodeExtIP).
			SetUsage(green("Optional, WAN IP address of the SuperNode, default - WAN IP address of the host")),
		cli.NewFlag("port", &config.MasterNodePort).
			SetUsage(green("Optional, Port for WAN IP address of the SuperNode, default - 9933 (19933 for Testnet, 29933 for Devnet)")),
		cli.NewFlag("reindex", &config.ReIndex).
			SetUsage(green("Optional, Start with reindex")),
	}

	var dirsFlags []*cli.Flag

	if remote && initCommand != coldHotInit {
		dirsFlags = []*cli.Flag{
			cli.NewFlag("dir", &config.PastelExecDir).SetAliases("d").
				SetUsage(green("Optional, Location of pastel node directory on the remote computer (default: $HOME/pastel)")),
			cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
				SetUsage(green("Optional, Location of working directory on the remote computer (default: $HOME/.pastel)")),
		}
	} else {
		dirsFlags = []*cli.Flag{
			cli.NewFlag("dir", &config.PastelExecDir).SetAliases("d").
				SetUsage(green("Optional, Location of pastel node directory")).SetValue(config.Configurer.DefaultPastelExecutableDir()),
			cli.NewFlag("work-dir", &config.WorkingDir).SetAliases("w").
				SetUsage(green("Optional, location of working directory")).SetValue(config.Configurer.DefaultWorkingDir()),
		}
	}

	superNodeInitFlags := []*cli.Flag{
		cli.NewFlag("name", &config.MasterNodeName).
			SetUsage(red("Required, name of the Masternode to create or update in the masternode.conf")),

		cli.NewFlag("new", &config.CreateNewMasterNodeConf).
			SetUsage(red("Required (if --add is not used), if specified, will create new masternode.conf with new Masternode record in it.")),
		cli.NewFlag("add", &config.AddToMasterNodeConf).
			SetUsage(red("Required (if --new is not used), if specified, will add new Masternode record to the existing masternode.conf.")),

		cli.NewFlag("pkey", &config.MasterNodePrivateKey).
			SetUsage(yellow("Optional, Masternode private key, if omitted, new masternode private key will be created")),

		cli.NewFlag("txid", &config.MasterNodeTxID).
			SetUsage(yellow("Optional, collateral payment txid, transaction id of 5M collateral MN payment")),
		cli.NewFlag("ind", &config.MasterNodeTxInd).
			SetUsage(yellow("Optional, collateral payment output index, output index in the transaction of 5M collateral MN payment")),

		cli.NewFlag("skip-collateral-validation", &config.DontCheckCollateral).
			SetUsage(yellow("Optional (if both txid and ind specified), skip validation of collateral tx on this node")),
		cli.NewFlag("noReindex", &config.DontUseReindex).
			SetUsage(yellow("Optional, disable any default --reindex")),

		cli.NewFlag("pastelid", &config.MasterNodePastelID).
			SetUsage(yellow("Optional, pastelid of the Masternode. If omitted, new pastelid will be created and registered")),
		cli.NewFlag("passphrase", &config.MasterNodePassPhrase).
			SetUsage(yellow("Optional, passphrase to pastelid private key. If omitted, user will be asked interactively")),

		cli.NewFlag("rpc-ip", &config.MasterNodeRPCIP).
			SetUsage(yellow("Optional, SuperNode IP address. If omitted, value passed to --ip will be used")),
		cli.NewFlag("rpc-port", &config.MasterNodeRPCPort).
			SetUsage(yellow("Optional, SuperNode port, default - 4444 (14444 for Testnet and 24444 for Devnet")),
		cli.NewFlag("p2p-ip", &config.MasterNodeP2PIP).
			SetUsage(yellow("Optional, Kademlia IP address, if omitted, value passed to --ip will be used")),
		cli.NewFlag("p2p-port", &config.MasterNodeP2PPort).
			SetUsage(yellow("Optional, Kademlia port, default - 4445 (14445 for Testnet and 24445 for Devnet)")),

		cli.NewFlag("activate", &config.ActivateMasterNode).
			SetUsage(yellow("Optional, if specified, will try to enable node as Masternode (start-alias).")),
	}

	remoteStartFlags := []*cli.Flag{
		cli.NewFlag("ssh-ip", &config.RemoteIP).
			SetUsage(red("Required, SSH address of the remote node")),
		cli.NewFlag("ssh-port", &config.RemotePort).
			SetUsage(yellow("Optional, SSH port of the remote node")).SetValue(22),
		cli.NewFlag("ssh-user", &config.RemoteUser).
			SetUsage(yellow("Optional, SSH user")),
		cli.NewFlag("ssh-key", &config.RemoteSSHKey).
			SetUsage(yellow("Optional, Path to SSH private key")),
	}
	coldhotStartFlags := []*cli.Flag{
		cli.NewFlag("ssh-ip", &config.RemoteIP).
			SetUsage(red("Required, SSH address of the remote HOT node")),
		cli.NewFlag("ssh-port", &config.RemotePort).
			SetUsage(yellow("Optional, SSH port of the remote HOT node")).SetValue(22),
		cli.NewFlag("ssh-user", &config.RemoteUser).
			SetUsage(yellow("Optional, SSH user")),
		cli.NewFlag("ssh-key", &config.RemoteSSHKey).
			SetUsage(yellow("Optional, Path to SSH private key")),
		cli.NewFlag("remote-dir", &config.RemoteHotPastelExecDir).
			SetUsage(yellow("Optional, Location where of pastel node directory on the remote computer (default: $HOME/pastel)")),
		cli.NewFlag("remote-work-dir", &config.RemoteHotWorkingDir).
			SetUsage(yellow("Optional, Location of working directory on the remote computer (default: $HOME/.pastel)")),
		cli.NewFlag("remote-home-dir", &config.RemoteHotHomeDir).
			SetUsage(yellow("Optional, Location of home directory on the remote computer (default: $HOME)")),
	}

	var commandName, commandMessage string
	if remote && initCommand != coldHotInit {
		commandName = initCmdName[remoteInit]
		commandMessage = initCmdMessage[remoteInit]
	} else {
		commandName = initCmdName[initCommand]
		commandMessage = initCmdMessage[initCommand]
	}

	commandFlags := append(superNodeInitFlags, dirsFlags[:]...)
	commandFlags = append(commandFlags, commonFlags[:]...)

	if remote && initCommand != coldHotInit {
		commandFlags = append(commandFlags, remoteStartFlags[:]...)
	}
	if initCommand == coldHotInit {
		commandFlags = append(commandFlags, coldhotStartFlags[:]...)
	}

	subCommand := cli.NewCommand(commandName)
	subCommand.SetUsage(cyan(commandMessage))
	subCommand.AddFlags(commandFlags...)
	addLogFlags(subCommand, config)

	if f != nil {
		subCommand.SetActionFunc(func(ctx context.Context, _ []string) error {
			ctx, err := configureLogging(ctx, commandMessage, config)
			if err != nil {
				return err
			}

			// Register interrupt handler
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt)
			go func() {
				for {
					<-sigCh

					yes, _ := AskUserToContinue(ctx, "Interrupt signal received, do you want to cancel this process? Y/N")
					if yes {
						log.WithContext(ctx).Info("Gracefully shutting down...")
						cancel()
						os.Exit(0)
					}
				}
			}()
			if !remote {
				if err = ParsePastelConf(ctx, config); err != nil {
					return err
				}
			}
			log.WithContext(ctx).Info("Starting")
			err = f(ctx, config)
			if err != nil {
				return err
			}
			log.WithContext(ctx).Info("Finished successfully!")

			return nil
		})
	}
	return subCommand
}

func setupInitCommand(config *configs.Config) *cli.Command {

	initSuperNodeSubCommand := setupInitSubCommand(config, superNodeInit, false, runInitSuperNodeSubCommand)
	initColdHotSuperNodeSubCommand := setupInitSubCommand(config, coldHotInit, true, runInitColdHotSuperNodeSubCommand)
	initRemoteSuperNodeSubCommand := setupInitSubCommand(config, superNodeInit, true, runInitRemoteSuperNodeSubCommand)

	initSuperNodeSubCommand.AddSubcommands(initColdHotSuperNodeSubCommand)
	initSuperNodeSubCommand.AddSubcommands(initRemoteSuperNodeSubCommand)

	startCommand := cli.NewCommand("init")
	startCommand.SetUsage(blue("Initialise SuperNode either locally, remotely or in cold-hot mode"))
	startCommand.AddSubcommands(initSuperNodeSubCommand)

	return startCommand

}

///// Top level start commands

// Sub Command
func runInitSuperNodeSubCommand(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Info("Initialising local supernode")

	if !config.DontUseReindex {
		config.ReIndex = true // init means first start, reindex is required
	}
	log.WithContext(ctx).Infof("DontUseReindex: %v", config.DontUseReindex)
	log.WithContext(ctx).Infof("ReIndex: %v", config.ReIndex)

	if !config.CreateNewMasterNodeConf && !config.AddToMasterNodeConf {
		log.WithContext(ctx).Error("Either 'new' or 'add' flag must be provided")
		return errors.New("either 'new' or 'add' flag is missing")
	}
	if config.CreateNewMasterNodeConf && config.MasterNodeName == "" {
		log.WithContext(ctx).Error("flag 'new' provided but name is missing")
		return errors.New("flag 'new' provided but name is missing")
	}

	if err := runStartSuperNode(ctx, config, true); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to initialize local supernode")
		return err
	}
	log.WithContext(ctx).Info("Local supernode Initialised successfully")
	return nil
}

func runInitColdHotSuperNodeSubCommand(ctx context.Context, config *configs.Config) (err error) {

	if !config.CreateNewMasterNodeConf && !config.AddToMasterNodeConf {
		log.WithContext(ctx).Error("Either 'new' or 'add' flag must be provided")
		return errors.New("either 'new' or 'add' flag is missing")
	}
	if config.CreateNewMasterNodeConf && config.MasterNodeName == "" {
		log.WithContext(ctx).Error("flag 'new' provided but name is missing")
		return errors.New("flag 'new' provided but name is missing")
	}

	runner := &ColdHotRunner{
		config: config,
		opts:   &ColdHotRunnerOpts{},
	}

	log.WithContext(ctx).Info("Initialising supernode in coldhot mode")
	if err := runner.Init(ctx); err != nil {
		log.WithContext(ctx).WithError(err).Error("init coldhot runner failed.")
		return err
	}

	log.WithContext(ctx).Info("running supernode coldhot runner")
	if err := runner.Run(ctx); err != nil {
		log.WithContext(ctx).WithError(err).Error("run coldhot runner failed.")
		return err
	}
	log.WithContext(ctx).Info("Supernode in coldhot mode initialised successfully")

	return nil
}

func runInitRemoteSuperNodeSubCommand(ctx context.Context, config *configs.Config) error {
	log.WithContext(ctx).Infof("Initializing remote supernode")

	if !config.CreateNewMasterNodeConf && !config.AddToMasterNodeConf {
		log.WithContext(ctx).Error("Either 'new' or 'add' flag must be provided")
		return errors.New("either 'new' or 'add' flag is missing")
	}
	if config.CreateNewMasterNodeConf && config.MasterNodeName == "" {
		log.WithContext(ctx).Error("flag 'new' provided but name is missing")
		return errors.New("flag 'new' provided but name is missing")
	}

	startOptions := ""

	if len(config.MasterNodeName) > 0 {
		startOptions = fmt.Sprintf("--name=%s", config.MasterNodeName)
	}

	if config.ActivateMasterNode {
		startOptions = fmt.Sprintf("%s --activate", startOptions)
	}

	if len(config.MasterNodePrivateKey) > 0 {
		startOptions = fmt.Sprintf("%s --pkey=%s", startOptions, config.MasterNodePrivateKey)
	}

	if config.CreateNewMasterNodeConf {
		startOptions = fmt.Sprintf("%s --new", startOptions)
	} else if config.AddToMasterNodeConf {
		startOptions = fmt.Sprintf("%s --add", startOptions)
	}

	if len(config.MasterNodeTxID) > 0 {
		startOptions = fmt.Sprintf("%s --txid=%s", startOptions, config.MasterNodeTxID)
	}

	if len(config.MasterNodeTxInd) > 0 {
		startOptions = fmt.Sprintf("%s --ind=%s", startOptions, config.MasterNodeTxInd)
	}

	if config.DontCheckCollateral {
		startOptions = fmt.Sprintf("%s --skip-collateral-validation", startOptions)
	}

	if len(config.MasterNodePastelID) > 0 {
		startOptions = fmt.Sprintf("%s --pastelid=%s", startOptions, config.MasterNodePastelID)
	}

	if len(config.MasterNodePassPhrase) > 0 {
		startOptions = fmt.Sprintf("%s --passphrase=%s", startOptions, config.MasterNodePassPhrase)
	}

	if config.MasterNodePort > 0 {
		startOptions = fmt.Sprintf("%s --port=%d", startOptions, config.MasterNodePort)
	}

	if len(config.MasterNodeRPCIP) > 0 {
		startOptions = fmt.Sprintf("%s --rpc-ip=%s", startOptions, config.MasterNodeRPCIP)
	}

	if config.MasterNodeRPCPort > 0 {
		startOptions = fmt.Sprintf("%s --rpc-port=%d", startOptions, config.MasterNodeRPCPort)
	}

	if len(config.MasterNodeP2PIP) > 0 {
		startOptions = fmt.Sprintf("%s --p2p-ip=%s", startOptions, config.MasterNodeP2PIP)
	}

	if config.MasterNodeP2PPort > 0 {
		startOptions = fmt.Sprintf("%s --p2p-port=%d", startOptions, config.MasterNodeP2PPort)
	}

	if len(config.NodeExtIP) > 0 {
		startOptions = fmt.Sprintf("%s --ip=%s", startOptions, config.NodeExtIP)
	}

	if config.ReIndex {
		startOptions = fmt.Sprintf("%s --reindex", startOptions)
	}

	if len(config.PastelExecDir) > 0 {
		startOptions = fmt.Sprintf("%s --dir=%s", startOptions, config.PastelExecDir)
	}

	if len(config.WorkingDir) > 0 {
		startOptions = fmt.Sprintf("%s --work-dir=%s", startOptions, config.WorkingDir)
	}

	initSuperNodeCmd := fmt.Sprintf("%s init supernode %s", constants.RemotePastelupPath, startOptions)
	if _, err := executeRemoteCommands(ctx, config, []string{initSuperNodeCmd}, false, false); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to init remote Supernode services")
		return err
	}
	log.WithContext(ctx).Infof("Remote supernode initialized")

	return nil
}

// /// masternode.conf helpers
func createOrUpdateMasternodeConf(ctx context.Context, config *configs.Config) error {

	// this function must only be called when --create or --update
	if !config.CreateNewMasterNodeConf && !config.AddToMasterNodeConf {
		return nil
	}

	var err error
	var conf map[string]masterNodeConf

	if config.AddToMasterNodeConf {
		conf, err = loadMasternodeConfFile(ctx, config)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to load existing masternode.conf file")
			return err
		}
	} else {
		conf = make(map[string]masterNodeConf)
	}
	log.WithContext(ctx).Infof("CONFIG: %+v", config)

	conf[config.MasterNodeName] = masterNodeConf{
		MnAddress:  config.NodeExtIP + ":" + fmt.Sprintf("%d", config.MasterNodePort),
		MnPrivKey:  config.MasterNodePrivateKey,
		Txid:       config.MasterNodeTxID,
		OutIndex:   config.MasterNodeTxInd,
		ExtAddress: config.NodeExtIP + ":" + fmt.Sprintf("%d", config.MasterNodeRPCPort),
		ExtP2P:     config.MasterNodeP2PIP + ":" + fmt.Sprintf("%d", config.MasterNodeP2PPort),
		ExtCfg:     "",
		ExtKey:     config.MasterNodePastelID,
	}

	// Create masternode.conf file
	if err := writeMasterNodeConfFile(ctx, config, conf); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to create new masternode.conf file")
		return err
	}
	log.WithContext(ctx).Info("masternode.conf updated")

	return nil
}

func writeMasterNodeConfFile(ctx context.Context, config *configs.Config, conf map[string]masterNodeConf) error {

	masternodeConfPath := getMasternodeConfPath(config, config.WorkingDir, "masternode.conf")

	confData, err := json.Marshal(conf)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Invalid new masternode conf data")
		return err
	}

	if err := backupMasterNodeConfFile(ctx, config, masternodeConfPath); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to backup previous masternode.conf file")
		return err
	}

	if err := os.WriteFile(masternodeConfPath, confData, 0644); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to create and write new masternode.conf file")
		return err
	}
	log.WithContext(ctx).Info("Created masternode config file at path:", masternodeConfPath)
	log.WithContext(ctx).Infof("masternode.conf = %s", string(confData))

	return nil
}

func backupMasterNodeConfFile(ctx context.Context, config *configs.Config, masternodeConfPath string) error {

	masternodeConfPathBackup := getMasternodeConfPath(config, config.WorkingDir, "masternode_%s.conf")
	if _, err := os.Stat(masternodeConfPath); err == nil { // if masternode.conf File exists , backup

		if yes, _ := AskUserToContinue(ctx, fmt.Sprintf("Previous masternode.conf found at - %s. "+
			"Do you want to back it up and continue? Y/N", masternodeConfPath)); !yes {

			log.WithContext(ctx).WithError(err).Error("masternode.conf already exists - exiting")
			return fmt.Errorf("masternode.conf already exists - %s", masternodeConfPath)
		}

		currentTime := time.Now()
		backupFileName := fmt.Sprintf(masternodeConfPathBackup, currentTime.Format("2021-01-01-23-59-59"))
		if err := os.Rename(masternodeConfPath, backupFileName); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to rename %s to %s", masternodeConfPath, backupFileName)
			return err
		}
		if _, err := os.Stat(masternodeConfPath); err == nil { // delete after back up if still exist
			if err = os.Remove(masternodeConfPath); err != nil {
				log.WithContext(ctx).WithError(err).Errorf("Failed to remove %s", masternodeConfPath)
				return err
			}
		}
	}

	return nil
}

func loadMasternodeConfFile(ctx context.Context, config *configs.Config) (map[string]masterNodeConf, error) {

	masternodeConfPath := getMasternodeConfPath(config, config.WorkingDir, "masternode.conf")

	// Read ConfData from masternode.conf
	confFile, err := os.ReadFile(masternodeConfPath)
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to read existing masternode.conf file - %s", masternodeConfPath)
		return nil, err
	}

	var conf map[string]masterNodeConf
	err = json.Unmarshal(confFile, &conf)
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Invalid existing masternode.conf file - %s", masternodeConfPath)
		return nil, err
	}
	return conf, nil
}

func getMasternodeConfData(ctx context.Context, config *configs.Config, mnName string, extIP string) (string, string, string, error) {
	conf, err := loadMasternodeConfFile(ctx, config)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to load existing masternode.conf file")
		return "", "", "", err
	}
	mnNode, ok := conf[mnName]
	if !ok {
		// if mnName is not set or doesn't have a match, lookup by extIP
		log.WithContext(ctx).Infof("Attempting to load existing masternode.conf file using external IP address...")
		for mnName, mnConf := range conf {
			extAddrPort := strings.Split(mnConf.MnAddress, ":")
			extAddr := extAddrPort[0] // get Ext IP and Port
			extPort := extAddrPort[1] // get Ext IP and Port
			if extAddr == extIP {
				log.WithContext(ctx).Infof("Loading masternode.conf file using %s conf", mnName)
				return mnConf.MnPrivKey, extAddr, extPort, nil
			}
		}
		err := errors.Errorf("masternode.conf doesn't have node with name - %s or external IP %v", mnName, extIP)
		log.WithContext(ctx).WithError(err).Errorf("Invalid masternode.conf json: %v", conf)
		return "", "", "", err
	}
	privateKey := mnNode.MnPrivKey
	extAddrPort := strings.Split(mnNode.MnAddress, ":")
	extAddr := extAddrPort[0] // get Ext IP and Port
	extPort := extAddrPort[1] // get Ext IP and Port
	return privateKey, extAddr, extPort, nil
}
