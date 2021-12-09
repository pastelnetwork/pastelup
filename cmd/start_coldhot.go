package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pastelnetwork/gonode/common/errors"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/pastel-utility/constants"
	"github.com/pastelnetwork/pastel-utility/structure"
	"github.com/pastelnetwork/pastel-utility/utils"
	"gopkg.in/yaml.v2"
)

// TODO: Remove the use of shadowing global variables and decouple
// this part from rest of the code for better maintenance of codebase

// ColdHotRunnerOpts defines opts for ColdHotRunner
type ColdHotRunnerOpts struct {
	// ssh params
	sshUser string
	sshIP   string
	sshPort int
	sshKey  string

	testnetOption string

	// remote paths
	remotePastelUtility string
	remotePasteld       string
	remotePastelCli     string
}

// ColdHotRunner starts sn in coldhot mode
type ColdHotRunner struct {
	sshClient *utils.Client
	config    *configs.Config
	opts      *ColdHotRunnerOpts
}

// Init initiates coldhot runner
func (r *ColdHotRunner) Init(ctx context.Context) error {
	if err := r.handleArgs(); err != nil {
		return fmt.Errorf("parse args: %s", err)
	}

	if err := r.handleConfigs(ctx); err != nil {
		return fmt.Errorf("parse args: %s", err)
	}

	client, err := connectSSH(ctx, r.opts.sshUser, r.opts.sshIP, r.opts.sshPort, r.opts.sshKey)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to connect with remote via SSH")
		return fmt.Errorf("ssh connection failure: %s", err)
	}
	r.sshClient = client

	// ***************** get external ip addr ***************
	if flagNodeExtIP == "" {
		out, err := client.Cmd(fmt.Sprintf("curl %s", "http://ipinfo.io/ip")).Output()
		if err != nil {
			return fmt.Errorf("failure in getting ext ip of remote %s", err)
		}

		flagNodeExtIP = string(out)
	}

	return nil
}

func (r *ColdHotRunner) handleArgs() (err error) {
	if len(r.config.RemotePastelUtilityDir) == 0 {
		return fmt.Errorf("cannot find remote pastel-utility dir")
	}

	if len(r.config.RemotePastelExecDir) == 0 {
		r.config.RemotePastelExecDir = r.config.Configurer.DefaultPastelExecutableDir()
	}

	r.opts.remotePastelCli = filepath.Join(r.config.RemotePastelExecDir, constants.PastelCliName[utils.GetOS()])
	r.opts.remotePastelCli = strings.ReplaceAll(r.opts.remotePastelCli, "\\", "/")

	r.opts.remotePasteld = filepath.Join(r.config.RemotePastelExecDir, constants.PasteldName[utils.GetOS()])
	r.opts.remotePasteld = strings.ReplaceAll(r.opts.remotePasteld, "\\", "/")

	r.opts.remotePastelUtility = filepath.Join(r.config.RemotePastelUtilityDir, "pastel-utility")
	r.opts.remotePastelUtility = strings.ReplaceAll(r.opts.remotePastelUtility, "\\", "/")

	return nil
}

func (r *ColdHotRunner) handleConfigs(ctx context.Context) error {
	log.WithContext(ctx).Infof("reading pastel.conf")
	// Check pastel config for testnet option and set config.Network
	if err := ParsePastelConf(ctx, r.config); err != nil {
		return fmt.Errorf("parse pastel.conf: %s", err)
	}

	if r.config.Network == constants.NetworkTestnet {
		r.opts.testnetOption = " --testnet"
	}
	if r.config.Network == constants.NetworkRegTest {
		r.opts.testnetOption = " --regtest"
	}
	log.WithContext(ctx).Infof("Finished Reading pastel.conf! Starting node in %s mode", r.config.Network)

	log.WithContext(ctx).Infof("checking masternode start params")
	// Check masternode params like sshIP, mnName, extIP, masternode conf, assign ports
	if err := checkStartMasterNodeParams(ctx, r.config, true); err != nil {
		return fmt.Errorf("checkStartMasterNodeParams: %s", err)
	}
	log.WithContext(ctx).Infof("finished checking masternode start params")

	return nil
}

// Run starts coldhot runner
func (r *ColdHotRunner) Run(ctx context.Context) (err error) {
	isPasteldAlreadyRunning := false
	var numOfSyncedBlocks int

	// ***************  1. Start the local Pastel Network Node ***************
	// Check if pasteld is already running
	if _, err = RunPastelCLI(ctx, r.config, "getinfo"); err == nil {
		log.WithContext(ctx).Info("Pasteld service is already running!")
		isPasteldAlreadyRunning = true
	} else {
		log.WithContext(ctx).Infof("Starting pasteld")
		if err = runPastelNode(ctx, r.config, true, "", ""); err != nil {
			log.WithContext(ctx).WithError(err).Error("pasteld failed to start")
			return err
		}
	}

	// Wait the local node to be synced
	log.WithContext(ctx).Infof("Waiting for local node to be synced")
	if numOfSyncedBlocks, err = CheckMasterNodeSync(ctx, r.config); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to wait for local node to fully sync")
		return err
	}

	// ***************  2. If flag --create or --update is provided ***************
	if flagMasterNodeIsCreate || flagMasterNodeIsUpdate {
		log.WithContext(ctx).Info("Prepare mastenode parameters")
		if err := r.handleCreateUpdateStartColdHot(ctx); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to validate and prepare masternode parameters")
			return err
		}
		if flagMasterNodeP2PIP == "" {
			flagMasterNodeP2PIP = flagNodeExtIP
		}
		if err := createOrUpdateMasternodeConf(ctx, r.config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to create or update masternode.conf")
			return err
		}
	}

	// ***************  3. Execute following commands over SSH on the remote node (using ssh-ip and ssh-port)  ***************

	// Run pasteld at remote side and wait for it to be synced
	log.WithContext(ctx).Infof("Starting pasteld at remote node and wait for it to be synced")
	if err = r.runRemoteNode(ctx, numOfSyncedBlocks); err != nil {
		log.WithContext(ctx).WithError(err).Error("failed on runRemoteNode")
		return err
	}
	log.WithContext(ctx).Infof("Remote::pasteld is fully synced")

	//Get conf data from masternode.conf File
	privkey, _, _, err := getMasternodeConfData(ctx, r.config, flagMasterNodeName)
	if err != nil {
		return err
	}
	flagMasterNodePrivateKey = privkey

	if err := r.runRemoteNodeAsMasterNode(ctx, numOfSyncedBlocks); err != nil {
		log.WithContext(ctx).WithError(err).Error("unable to run remote as masternode")
		return fmt.Errorf("run remote as masternode: %s", err)
	}
	log.WithContext(ctx).Info("remote node started as masternode successfully..")

	// Restart pasteld at local node (cold node)
	log.WithContext(ctx).Infof("Stopping pasteld at local node")
	if err = StopPastelDAndWait(ctx, r.config); err != nil {
		log.WithContext(ctx).WithError(err).Error("failed to stop pasteld")
		return err
	}

	log.WithContext(ctx).Infof("Starting pasteld at local node")
	if err = runPastelNode(ctx, r.config, true, "", ""); err != nil {
		log.WithContext(ctx).WithError(err).Error("failed to start pasteld")
		return err
	}

	// ***************  4. If --activate are provided, ***************
	if flagMasterNodeIsActivate {
		log.WithContext(ctx).Info("found --activate flag, checking local node sync..")
		if _, err = CheckMasterNodeSync(ctx, r.config); err != nil {
			log.WithError(err).Error("local Masternode sync failure")
			return err
		}

		log.WithContext(ctx).Info("now activating mn...")
		if err = runStartAliasMasternode(ctx, r.config, flagMasterNodeName); err != nil {
			log.WithError(err).Error("Masternode activation failure")
			return fmt.Errorf("masternode activation failed: %s", err)
		}

		if flagMasterNodeIsCreate {
			log.WithContext(ctx).Info("registering pastelID ticket...")
			if err := r.registerTicketPastelID(ctx); err != nil {
				log.WithContext(ctx).WithError(err).Error("unable to register pastelID ticket")
			}
		}
	}

	// ***************  5. Stop Cold Node  ***************
	if isPasteldAlreadyRunning {
		log.WithContext(ctx).Info("As pasteld is running before starting supernode-coldstart, keep it running!")
	} else {
		log.WithContext(ctx).Info("Stopping code node ... ")
		if err = StopPastelDAndWait(ctx, r.config); err != nil {
			log.WithContext(ctx).WithError(err).Error("unable to stop local node")
			return err
		}
	}

	// *************  6. Start rq-servce    *************
	log.WithContext(ctx).Info("starting rq-service..")
	if err = r.runServiceRemote(ctx, string(constants.RQService)); err != nil {
		return fmt.Errorf("failed to start rq-service on hot node: %s", err)
	}
	log.WithContext(ctx).Info("rq-service started successfully")

	// *************  Start dd-servce    *************
	log.WithContext(ctx).Info("starting dd-service..")
	if err = r.runServiceRemote(ctx, string(constants.DDService)); err != nil {
		return fmt.Errorf("failed to start dd-service on hot node: %s", err)
	}
	log.WithContext(ctx).Info("dd-service started successfully")

	// ***************  7. Start supernode  **************

	if err := r.createAndCopyRemoteSuperNodeConfig(ctx, r.config); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to update supernode.yml")
		return err
	}

	log.WithContext(ctx).Info("starting supernode-service..")
	snService := fmt.Sprintf("%s-%s", string(constants.SuperNode), "service")
	if err = r.runServiceRemote(ctx, snService); err != nil {
		return fmt.Errorf("failed to start supernode-service on hot node: %s", err)
	}
	log.WithContext(ctx).Info("started supernode-service successfully..")

	return nil
}

func (r *ColdHotRunner) runRemoteNodeAsMasterNode(ctx context.Context, numOfSyncedBlocks int) error {

	log.WithContext(ctx).Info("Running remote node as masternode ...")
	go func() {
		cmdLine := fmt.Sprintf("%s --masternode --txindex=1 --reindex --masternodeprivkey=%s --externalip=%s  --data-dir=%s %s --daemon ",
			r.opts.remotePasteld, flagMasterNodePrivateKey, flagNodeExtIP, r.config.RemoteWorkingDir, r.opts.testnetOption)

		log.WithContext(ctx).Infof("start remote node as masternode%s\n", cmdLine)

		if err := r.sshClient.Cmd(cmdLine).Run(); err != nil {
			fmt.Println("pasteld run err: ", err.Error())
		}
	}()

	if !CheckPastelDRunningRemote(ctx, r.sshClient, r.opts.remotePastelCli, true) {
		err := fmt.Errorf("unable to start pasteld on remote")
		log.WithContext(ctx).WithError(err).Error("run remote as master failed")
		return err
	}

	if err := r.checkMasterNodeSyncRemote(ctx, numOfSyncedBlocks, 0); err != nil {
		log.WithContext(ctx).Error("Remote::Master node sync failed")
		return err
	}

	log.WithContext(ctx).Info("Remote::Master node is started and synced sucessfully")

	return nil
}

func (r *ColdHotRunner) handleCreateUpdateStartColdHot(ctx context.Context) error {
	var err error

	// Check Collateral for cold node
	if err = checkCollateral(ctx, r.config); err != nil {
		log.WithContext(ctx).WithError(err).Error("Missing collateral transaction")
		return err
	}

	if err = checkPassphrase(ctx); err != nil {
		log.WithContext(ctx).WithError(err).Error("Missing passphrase")
		return err
	}

	// Check if cold node is already running, if yes stop it
	if err := stopRemoteNode(ctx, r.sshClient, r.opts.remotePastelCli); err != nil {
		log.WithContext(ctx).WithError(err).Error("remote unable to stop pasteld")
		return err
	}

	go func() {
		cmdLine := fmt.Sprintf("%s --reindex --externalip=%s --data-dir=%s --daemon %s",
			r.opts.remotePasteld, flagNodeExtIP, r.config.RemoteWorkingDir, r.opts.testnetOption)
		log.WithContext(ctx).Infof("starting pasteld on the remote node - %s\n", cmdLine)
		if err := r.sshClient.Cmd(cmdLine).Run(); err != nil {
			log.WithContext(ctx).WithError(err).Error("unable to start pasteld on remote")
		}
	}()

	if !CheckPastelDRunningRemote(ctx, r.sshClient, r.opts.remotePastelCli, true) {
		return errors.New("unable to start pasteld on remote")
	}

	if err = checkMasternodePrivKey(ctx, r.config, r.sshClient); err != nil {
		log.WithContext(ctx).WithError(err).Error("Missing masternode private key")
		return err
	}

	if err = checkPastelID(ctx, r.config, r.sshClient); err != nil {
		log.WithContext(ctx).WithError(err).Error("Missing masternode PastelID")
		return err
	}

	if _, err = r.sshClient.Cmd(fmt.Sprintf("%s stop", r.opts.remotePastelCli)).Output(); err != nil {
		log.WithContext(ctx).Error("Error - stopping on remote pasteld")
		return err
	}

	time.Sleep(5 * time.Second)

	if flagMasterNodeIsCreate {
		if _, err = backupConfFile(ctx, r.config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to backup masternode.conf")
			return err
		}
	}

	return nil
}

func (r *ColdHotRunner) runServiceRemote(ctx context.Context, service string) (err error) {
	log.WithContext(ctx).WithField("service", service).Info("starting service on remote")

	cmd := fmt.Sprintf("%s %s %s", r.opts.remotePastelUtility, "start", service)
	if r.config.RemoteWorkingDir != "" {
		cmd = fmt.Sprintf("%s --work-dir=%s", cmd, r.config.RemoteWorkingDir)
	}

	out, err := r.sshClient.Cmd(cmd).Output()
	if err != nil {
		log.WithContext(ctx).WithField("service", service).WithField("out", string(out)).WithField("cmd", cmd).
			WithError(err).Error("failed to start service on remote")
		return fmt.Errorf("failed to start service on remote: %s", err.Error())
	}

	return err
}

// CheckPastelDRunningRemote whether pasteld is running
func CheckPastelDRunningRemote(ctx context.Context, client *utils.Client, cliPath string, want bool) (ret bool) {
	var failCnt = 0
	var err error

	log.WithContext(ctx).Info("Remote::checking if pasteld is running ...")

	for {
		if _, err = client.Cmd(fmt.Sprintf("%s %s", cliPath, "getinfo")).Output(); err != nil {
			if !want {
				log.WithContext(ctx).Info("remote pasteld is not running")
				return false
			}

			time.Sleep(5 * time.Second)
			failCnt++
			if failCnt == 12 {
				return false
			}
		} else {
			break
		}
	}

	log.WithContext(ctx).Info("Remote::Pasteld is running")
	return true
}

// stopRemoteNode - just stop pasteld and check if it is really stopped
func stopRemoteNode(ctx context.Context, client *utils.Client, cliPath string) error {
	log.WithContext(ctx).Info("remote stopping pasteld if it is running ...")

	client.Cmd(fmt.Sprintf("%s %s", cliPath, "stop")).Output()

	time.Sleep(10 * time.Second)
	if _, err := client.Cmd(fmt.Sprintf("%s %s", cliPath, "getinfo")).Output(); err == nil {
		return fmt.Errorf("failed to stop remote node")
	}

	return nil
}

func (r *ColdHotRunner) runRemoteNode(ctx context.Context, numOfSyncedBlocks int) error {
	go func() {
		if err := r.sshClient.Cmd(fmt.Sprintf("%s --reindex --externalip=%s --data-dir=%s --daemon %s",
			r.opts.remotePasteld, flagNodeExtIP, r.config.RemoteWorkingDir, r.opts.testnetOption)).Run(); err != nil {
			fmt.Println("pasteld run err: ", err.Error())
		}
	}()

	if !CheckPastelDRunningRemote(ctx, r.sshClient, r.opts.remotePastelCli, true) {
		return fmt.Errorf("unable to start pasteld on remote")
	}

	if err := r.checkMasterNodeSyncRemote(ctx, numOfSyncedBlocks, 0); err != nil {
		log.WithContext(ctx).Error("Remote:: node sync failed")
		return err
	}
	log.WithContext(ctx).Info("Remote::node is fully synced")

	log.WithContext(ctx).Info("Remote::Stopping pasteld ...")
	if err := stopRemoteNode(ctx, r.sshClient, r.opts.remotePastelCli); err != nil {
		log.WithContext(ctx).WithError(err).Error("Remote::unable to stop pasteld")
		return err
	}
	log.WithContext(ctx).Info("Remote::pasteld is stopped")

	return nil
}

func (r *ColdHotRunner) checkMasterNodeSyncRemote(ctx context.Context, numOfSyncedBlocks int, retryCount int) (err error) {
	var mnstatus structure.RPCPastelMSStatus
	var output []byte
	var getinfo structure.RPCGetInfo

	for {
		// Get mnsync status
		if output, err = r.sshClient.Cmd(fmt.Sprintf("%s mnsync status", r.opts.remotePastelCli)).Output(); err != nil {
			log.WithContext(ctx).WithField("out", string(output)).WithError(err).
				Error("Remote:::failed to get mnsync status")
			if retryCount == 0 {
				log.WithContext(ctx).WithError(err).Error("retrying mynsyc staus...")
				time.Sleep(5 * time.Second)

				return r.checkMasterNodeSyncRemote(ctx, numOfSyncedBlocks, 1)
			}

			return err
		}

		if err = json.Unmarshal([]byte(output), &mnstatus); err != nil {
			log.WithContext(ctx).WithField("payload", string(output)).WithError(err).
				Error("Remote:::failed to unmarshal mnsync status")

			return err
		}

		if mnstatus.AssetName == "Initial" {
			if out, err := r.sshClient.Cmd(fmt.Sprintf("%s mnsync reset", r.opts.remotePastelCli)).Output(); err != nil {
				log.WithContext(ctx).WithField("out", string(out)).WithError(err).
					Error("Remote:::master node reset was failed")

				return err
			}
			time.Sleep(10 * time.Second)
		}
		if mnstatus.IsSynced {
			log.WithContext(ctx).Info("Remote:::master node was synced!")
			break
		}

		// Get blockcount at remote: `pasteld getinfo`
		if output, err = r.sshClient.Cmd(fmt.Sprintf("%s getinfo", r.opts.remotePastelCli)).Output(); err != nil {
			log.WithContext(ctx).WithField("out", string(output)).WithError(err).
				Error("Remote:::pasteld failed to get getinfo")
			return err
		}

		if err = json.Unmarshal([]byte(output), &getinfo); err != nil {
			log.WithContext(ctx).WithField("payload", string(output)).WithError(err).
				Error("Remote:::pasteld failed to unmarshal getinfo")
			return err
		}

		log.WithContext(ctx).Infof("Remote:::Waiting for sync... (%d from %d)", getinfo.Blocks, numOfSyncedBlocks)
		time.Sleep(10 * time.Second)
	}
	return nil
}

///// supernode.yml helpers
func (r *ColdHotRunner) createAndCopyRemoteSuperNodeConfig(ctx context.Context, config *configs.Config) error {

	supernodeConfigPath := "supernode.yml"
	log.WithContext(ctx).Infof("Creating remote supernode config - %s", supernodeConfigPath)

	if _, err := os.Stat(supernodeConfigPath); os.IsNotExist(err) {
		// create new
		if err = utils.CreateFile(ctx, supernodeConfigPath, true); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to create new supernode.yml file at - %s", supernodeConfigPath)
			return err
		}

		portList := GetSNPortList(config)

		snTempDirPath := filepath.Join(config.RemoteWorkingDir, constants.TempDir)
		rqWorkDirPath := filepath.Join(config.RemoteWorkingDir, constants.RQServiceDir)
		p2pDataPath := filepath.Join(config.RemoteWorkingDir, constants.P2PDataDir)
		mdlDataPath := filepath.Join(config.RemoteWorkingDir, constants.MDLDataDir)

		toolConfig, err := utils.GetServiceConfig(string(constants.SuperNode), configs.SupernodeDefaultConfig, &configs.SuperNodeConfig{
			LogLevel:      constants.SuperNodeDefaultLogLevel,
			LogFilePath:   config.Configurer.GetSuperNodeLogFile(config.RemoteWorkingDir),
			SNTempDir:     snTempDirPath,
			SNWorkDir:     config.RemoteWorkingDir,
			RQDir:         rqWorkDirPath,
			DDDir:         filepath.Join(config.Configurer.DefaultHomeDir(), constants.DupeDetectionServiceDir),
			SuperNodePort: portList[constants.SNPort],
			P2PPort:       portList[constants.P2PPort],
			P2PDataDir:    p2pDataPath,
			MDLPort:       portList[constants.MDLPort],
			RAFTPort:      portList[constants.RAFTPort],
			MDLDataDir:    mdlDataPath,
			RaptorqPort:   constants.RQServiceDefaultPort,
			DDServerPort:  constants.DDServerDefaultPort,
			PasteID:       flagMasterNodePastelID,
			Passphrase:    flagMasterNodePassPhrase,
		})
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to get supernode config")
			return err
		}
		if err = utils.WriteFile(supernodeConfigPath, toolConfig); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to update new supernode.yml file at - %s", supernodeConfigPath)
			return err
		}

	} else if err == nil {
		//update existing
		var snConfFile []byte
		snConfFile, err = ioutil.ReadFile(supernodeConfigPath)
		if err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to open existing supernode.yml file at - %s", supernodeConfigPath)
			return err
		}
		snConf := make(map[string]interface{})
		if err = yaml.Unmarshal(snConfFile, &snConf); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to parse existing supernode.yml file at - %s", supernodeConfigPath)
			return err
		}

		node := snConf["node"].(map[interface{}]interface{})

		node["pastel_id"] = flagMasterNodePastelID
		node["pass_phrase"] = flagMasterNodePassPhrase

		var snConfFileUpdated []byte
		if snConfFileUpdated, err = yaml.Marshal(&snConf); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to unparse yml for supernode.yml file at - %s", supernodeConfigPath)
			return err
		}
		if ioutil.WriteFile(supernodeConfigPath, snConfFileUpdated, 0644) != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to update supernode.yml file at - %s", supernodeConfigPath)
			return err
		}
	} else {
		log.WithContext(ctx).WithError(err).Errorf("Failed to update or create supernode.yml file at - %s", supernodeConfigPath)
		return err
	}

	log.WithContext(ctx).Info("Supernode config updated")

	remoteSnConfigPath := r.config.Configurer.GetSuperNodeConfFile(r.config.RemoteWorkingDir)
	remoteSnConfigPath = strings.ReplaceAll(remoteSnConfigPath, "\\", "/")

	log.WithContext(ctx).Info("copying supernode config..")
	if err := r.sshClient.Scp(supernodeConfigPath, remoteSnConfigPath); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to copy pastel-utility executable to remote host")
		return err
	}

	if err := utils.DeleteFile(supernodeConfigPath); err != nil {
		log.WithContext(ctx).Errorf("Failed to delete archive file : %s", supernodeConfigPath)
		return err
	}

	if err := r.sshClient.ShellCmd(ctx, fmt.Sprintf("chmod 755 %s", remoteSnConfigPath)); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to change permission of pastel-utility")
		return err
	}

	return nil
}
