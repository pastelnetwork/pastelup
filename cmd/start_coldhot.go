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
	"github.com/pastelnetwork/pastelup/configs"
	"github.com/pastelnetwork/pastelup/constants"
	"github.com/pastelnetwork/pastelup/structure"
	"github.com/pastelnetwork/pastelup/utils"
	"gopkg.in/yaml.v2"
)

// TODO: Remove the use of shadowing global variables and decouple
// this part from rest of the code for better maintenance of codebase

// ColdHotRunnerOpts defines opts for ColdHotRunner
type ColdHotRunnerOpts struct {
	testnetOption string
	reIndex       string

	// remote paths
	remotePastelUp  string
	remotePasteld   string
	remotePastelCli string
}

// ColdHotRunner starts sn in coldhot mode
type ColdHotRunner struct {
	sshClient *utils.Client
	config    *configs.Config
	opts      *ColdHotRunnerOpts
}

// Init initiates coldhot runner
func (r *ColdHotRunner) Init(ctx context.Context) error {
	client, err := prepareRemoteSession(ctx, r.config)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to prepare remote session")
		return fmt.Errorf("prepare remote session: %s", err)
	}

	r.sshClient = client

	if err := r.handleArgs(ctx); err != nil {
		return fmt.Errorf("parse args: %s", err)
	}

	if err := r.handleConfigs(ctx); err != nil {
		return fmt.Errorf("parse args: %s", err)
	}

	//get remote pastel.conf
	//remotePastelConfPath := filepath.Join(r.config.RemoteHotWorkingDir, constants.PastelConfName)
	//if err := client.ScpFrom(localPastelupPath, remotePastelUp); err != nil {
	//	return fmt.Errorf("failed to copy pastel.comf from remote %s", err)
	//}

	// Get external IP
	if flagNodeExtIP == "" {
		out, err := client.Cmd(fmt.Sprintf("curl %s", "http://ipinfo.io/ip")).Output()
		if err != nil {
			return fmt.Errorf("failure in getting ext ip of remote %s", err)
		}

		flagNodeExtIP = string(out)
	}

	return nil
}

func (r *ColdHotRunner) handleArgs(ctx context.Context) (err error) {

	if len(r.config.RemoteHotHomeDir) == 0 {
		var out []byte
		if out, err = r.sshClient.Cmd("eval echo ~$USER").Output(); err != nil || len(out) == 0 {
			log.WithContext(ctx).Error("Cannot identify remote HOME directory. Please use '--remote-home-dir'")
			return fmt.Errorf("cannot identify remote HOME directory")
		}
		r.config.RemoteHotHomeDir = strings.TrimSuffix(string(out), "\n")
	}
	if len(r.config.RemoteHotPastelExecDir) == 0 {
		r.config.RemoteHotPastelExecDir = filepath.Join(r.config.RemoteHotHomeDir, "pastel")
	}
	if len(r.config.RemoteHotWorkingDir) == 0 {
		r.config.RemoteHotWorkingDir = filepath.Join(r.config.RemoteHotHomeDir, ".pastel")
	}
	log.WithContext(ctx).Infof("Remote (HOT) HOME directory - %s", r.config.RemoteHotHomeDir)
	log.WithContext(ctx).Infof("Remote (HOT) Installation directory - %s", r.config.RemoteHotPastelExecDir)
	log.WithContext(ctx).Infof("Remote (HOT) Working directory - %s", r.config.RemoteHotWorkingDir)

	r.opts.remotePastelCli = filepath.Join(r.config.RemoteHotPastelExecDir, constants.PastelCliName[utils.GetOS()])
	r.opts.remotePastelCli = strings.ReplaceAll(r.opts.remotePastelCli, "\\", "/")

	r.opts.remotePasteld = filepath.Join(r.config.RemoteHotPastelExecDir, constants.PasteldName[utils.GetOS()])
	r.opts.remotePasteld = strings.ReplaceAll(r.opts.remotePasteld, "\\", "/")

	r.opts.remotePastelUp = constants.RemotePastelupPath

	return nil
}

func (r *ColdHotRunner) handleConfigs(ctx context.Context) error {
	log.WithContext(ctx).Infof("reading pastel.conf")
	// Check pastel config for testnet option and set config.Network
	if err := ParsePastelConf(ctx, r.config); err != nil {
		return fmt.Errorf("parse pastel.conf: %s", err)
	}

	if r.config.ReIndex {
		r.opts.reIndex = " --reindex"
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

	defer r.sshClient.Close()

	// ***************  1. Start the local Pastel Node (if it is not already running) and ensure it is fully synced ***************
	// 1.A Try to start pasteld (it will not start again if it is already running)
	if _, err = RunPastelCLI(ctx, r.config, "getinfo"); err == nil {
		log.WithContext(ctx).Info("Pasteld service is already running!")
		isPasteldAlreadyRunning = true
	} else {
		log.WithContext(ctx).Infof("Starting pasteld")

		mmnConfFile := getMasternodeConfPath(r.config, r.config.WorkingDir, "masternode.conf")
		txIndexOne := utils.CheckFileExist(mmnConfFile)

		if err = runPastelNode(ctx, r.config, txIndexOne, r.config.ReIndex, "", ""); err != nil {
			log.WithContext(ctx).WithError(err).Error("pasteld failed to start")
			return err
		}
	}
	// 1.B Wait the local node to be synced (this will start )
	log.WithContext(ctx).Infof("Waiting for local node to be synced")
	if numOfSyncedBlocks, err = CheckMasterNodeSync(ctx, r.config); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to wait for local node to fully sync")
		return err
	}

	// ***************  2. Start the remote Pastel Node (if it is not already running) and ensure it is fully synced ***************
	// 2.A Check if the remote node is already running and is fully synced
	remoteIsRunning := false
	remoteSynced := false
	if CheckPastelDRunningRemote(ctx, r.sshClient, r.opts.remotePastelCli, false) {
		remoteIsRunning = true
		log.WithContext(ctx).Infof("remote pasteld is already running")
		if yes, _ := AskUserToContinue(ctx,
			"Do you want to stop it and restart as SuperNode? Y/N"); !yes {
			log.WithContext(ctx).Warn("Exiting...")
			return fmt.Errorf("user terminated installation")
		}

		if err := r.checkMasterNodeSyncRemote(ctx, numOfSyncedBlocks, 0); err != nil {
			log.WithContext(ctx).WithError(err).Warn("Remote::unable to sync running node, will stop and re-try")
			if err := stopRemoteNode(ctx, r.sshClient, r.opts.remotePastelCli); err != nil {
				log.WithContext(ctx).WithError(err).Error("Remote::unable to stop pasteld")
				return err
			}
		} else {
			log.WithContext(ctx).Info("Remote::node is fully synced")
			remoteSynced = true
		}
	}
	// 2.B if remote Node is not running OR is not synced - start and wait for sync
	if !remoteSynced {
		// Run pasteld at remote side, wait for it to be synced and STOP pasteld
		log.WithContext(ctx).Infof("Starting pasteld at remote node and wait for it to be synced")
		if err = r.startAndSyncRemoteNode(ctx, numOfSyncedBlocks); err != nil {
			log.WithContext(ctx).WithError(err).Error("failed on startAndSyncRemoteNode")
			return err
		}
		log.WithContext(ctx).Infof("Remote::pasteld is fully synced")
	}

	// *************** 3. Prepare the remote node for coldhot mode (this is always - `init supernode coldhot` call) ***************
	if flagMasterNodeConfNew || flagMasterNodeConfAdd {
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

		// Even though in COLD/HOT mode masternode/conf only required on the COLD (local) node,
		//we copy it to the HOT (remote) node, so `start supernode remote` will work (it parses that file for start parameters)
		if err := r.copyMasterNodeConToRemote(ctx); err != nil {
			return fmt.Errorf("failed to copy masternode.conf to remote %s", err)
		}
	}
	// Now remote node need to be stopped, so it can be re-started as masternode
	if remoteIsRunning {
		log.WithContext(ctx).Info("Remote::Stopping pasteld ...")
		if err := stopRemoteNode(ctx, r.sshClient, r.opts.remotePastelCli); err != nil {
			log.WithContext(ctx).WithError(err).Error("Remote::unable to stop pasteld")
			return err
		}
		log.WithContext(ctx).Info("Remote::pasteld is stopped")
	}

	// *************** 4. Start remote node as masternode ***************
	//Get conf data from masternode.conf File
	privkey, _, _, err := getMasternodeConfData(ctx, r.config, flagMasterNodeName, flagNodeExtIP)
	if err != nil {
		return err
	}
	flagMasterNodePrivateKey = privkey

	if err := r.runRemoteNodeAsMasterNode(ctx, numOfSyncedBlocks); err != nil {
		log.WithContext(ctx).WithError(err).Error("unable to run remote as masternode")
		return fmt.Errorf("run remote as masternode: %s", err)
	}
	log.WithContext(ctx).Info("remote node started as masternode successfully..")

	// ***************  5. Start/Stop local (if needed) ***************
	// 5.A Restart local cold node pasteld - this is required if masternode.conf was created/modified, so local node can re-read it
	// but only in case it was already running OR we need to activate remote (HOT) "masternode" (--activate is provided)
	if (flagMasterNodeConfNew || flagMasterNodeConfAdd) && (isPasteldAlreadyRunning || flagMasterNodeIsActivate) {
		log.WithContext(ctx).Infof("Stopping pasteld at local node")
		if err = StopPastelDAndWait(ctx, r.config); err != nil {
			log.WithContext(ctx).WithError(err).Error("failed to stop pasteld")
			return err
		}
		log.WithContext(ctx).Infof("Starting pasteld at local node")
		if err = runPastelNode(ctx, r.config, true, true, "", ""); err != nil {
			log.WithContext(ctx).WithError(err).Error("failed to start pasteld")
			return err
		}
	}
	// 5.B activate HOT node
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

		if flagMasterNodeConfNew {
			log.WithContext(ctx).Info("registering pastelID ticket...")
			if err := r.registerTicketPastelID(ctx); err != nil {
				log.WithContext(ctx).WithError(err).Error("unable to register pastelID ticket")
			}
		}
	}
	// 5.C Stop Cold Node
	if isPasteldAlreadyRunning {
		log.WithContext(ctx).Info("As pasteld was running before starting this operation (init supernode coldhot), it will be kept running!")
	} else {
		log.WithContext(ctx).Info("Stopping code node ... ")
		if err = StopPastelDAndWait(ctx, r.config); err != nil {
			log.WithContext(ctx).WithError(err).Error("unable to stop local node")
			return err
		}
	}

	// *************  6. Start rq-servce    *************
	withErrors := false
	log.WithContext(ctx).Info("starting rq-service..")
	if err = r.runServiceRemote(ctx, string(constants.RQService)); err != nil {
		log.WithContext(ctx).WithError(err).Error("failed to start rq-service on hot node")
		withErrors = true
	} else {
		log.WithContext(ctx).Info("rq-service started successfully")
	}

	// *************  7. Start dd-servce    *************
	log.WithContext(ctx).Info("starting dd-service..")
	if err = r.runServiceRemote(ctx, string(constants.DDService)); err != nil {
		log.WithContext(ctx).WithError(err).Error("failed to start dd-service on hot node")
		withErrors = true
	} else {
		log.WithContext(ctx).Info("dd-service started successfully")
	}

	// ***************  8. Start supernode  **************

	if err := r.createAndCopyRemoteSuperNodeConfig(ctx); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to update supernode.yml")
		return err
	}

	if err := r.createAndCopyRemoteHermesConfig(ctx); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to update hermes.yml")
		return err
	}

	log.WithContext(ctx).Info("starting supernode-service & hermes-service..")
	snService := fmt.Sprintf("%s-%s", string(constants.SuperNode), "service")
	if err = r.runServiceRemote(ctx, snService); err != nil {
		return fmt.Errorf("failed to start supernode-service or hermes-service on hot node: %s", err)
	}

	if withErrors {
		log.WithContext(ctx).Warn("some services was not started, please see log above.\n\tYou can try to restart them manually: see command 'start <service> remote'")
	}

	return nil
}

func (r *ColdHotRunner) runRemoteNodeAsMasterNode(ctx context.Context, numOfSyncedBlocks int) error {

	log.WithContext(ctx).Info("Running remote node as masternode ...")
	go func() {
		cmdLine := fmt.Sprintf("%s --masternode --txindex=1 --reindex --masternodeprivkey=%s --externalip=%s  --data-dir=%s %s --daemon ",
			r.opts.remotePasteld, flagMasterNodePrivateKey, flagNodeExtIP, r.config.RemoteHotWorkingDir, r.opts.testnetOption)

		log.WithContext(ctx).Infof("start remote node as masternode - %s\n", cmdLine)

		if err := r.sshClient.Cmd(cmdLine).Run(); err != nil {
			fmt.Printf("pasteld run err: %s\n", err.Error())
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

	// Check Collateral on COLD node
	if err = checkCollateral(ctx, r.config); err != nil {
		log.WithContext(ctx).WithError(err).Error("Missing collateral transaction")
		return err
	}

	if err = checkPassphrase(ctx); err != nil {
		log.WithContext(ctx).WithError(err).Error("Missing passphrase")
		return err
	}

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

	return nil
}

func (r *ColdHotRunner) runServiceRemote(ctx context.Context, service string) (err error) {
	log.WithContext(ctx).WithField("service", service).Info("starting service on remote")

	cmd := fmt.Sprintf("%s %s %s", r.opts.remotePastelUp, "start", service)
	if r.config.RemoteHotWorkingDir != "" {
		cmd = fmt.Sprintf("%s --work-dir=%s", cmd, r.config.RemoteHotWorkingDir)
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
func CheckPastelDRunningRemote(ctx context.Context, client *utils.Client, cliPath string, wait bool) (ret bool) {
	var failCnt = 0
	var err error

	log.WithContext(ctx).Info("Remote::checking if pasteld is running ...")

	for {
		if _, err = client.Cmd(fmt.Sprintf("%s %s", cliPath, "getinfo")).Output(); err != nil {
			if !wait {
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

	log.WithContext(ctx).Info("remote pasteld is stopped ...")
	return nil
}

func (r *ColdHotRunner) startAndSyncRemoteNode(ctx context.Context, numOfSyncedBlocks int) error {
	startRemotePasteld := func() {
		cmd := fmt.Sprintf("%s %s --externalip=%s --data-dir=%s --daemon %s",
			r.opts.remotePasteld, r.opts.reIndex, flagNodeExtIP, r.config.RemoteHotWorkingDir, r.opts.testnetOption)
		log.WithContext(ctx).Infof("Remote::node starting pasteld - %s", cmd)
		if err := r.sshClient.Cmd(cmd).Run(); err != nil {
			fmt.Printf("pasteld run err: %s\n", err.Error())
		}
	}

	go startRemotePasteld()

	if !CheckPastelDRunningRemote(ctx, r.sshClient, r.opts.remotePastelCli, true) {
		if r.opts.reIndex != "--reindex" {
			yes, _ := AskUserToContinue(ctx, "pasteld failed to start, starting it with --reindex might help. "+
				"But it will take 20-30 minutes longer? Do you wan to proceed? Y/N")
			if !yes {
				log.WithContext(ctx).Error("User terminated - exiting")
				return fmt.Errorf("user terminated - exiting")
			}
			r.opts.reIndex = "--reindex"
			go startRemotePasteld()
			if !CheckPastelDRunningRemote(ctx, r.sshClient, r.opts.remotePastelCli, true) {
				return fmt.Errorf("unable to start pasteld on remote")
			}
		} else {
			return fmt.Errorf("unable to start pasteld on remote")
		}
	}

	if err := r.checkMasterNodeSyncRemote(ctx, numOfSyncedBlocks, 0); err != nil {
		log.WithContext(ctx).Error("Remote:: node sync failed")
		return err
	}
	log.WithContext(ctx).Info("Remote::node is fully synced")

	//log.WithContext(ctx).Info("Remote::Stopping pasteld ...")
	//if err := stopRemoteNode(ctx, r.sshClient, r.opts.remotePastelCli); err != nil {
	//	log.WithContext(ctx).WithError(err).Error("Remote::unable to stop pasteld")
	//	return err
	//}
	//log.WithContext(ctx).Info("Remote::pasteld is stopped")
	return nil
}

func (r *ColdHotRunner) checkMasterNodeSyncRemote(ctx context.Context, numOfSyncedBlocks int, retryCount int) (err error) {
	// when running cmds against pastel-cli and not RPC server,
	// the output is not wrapped in a Result thus set the output catchers
	// to the underlying Result object instead of base object
	var mnstatus structure.MNSyncStatusResult
	var getinfo structure.GetInfoResult
	var output []byte
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
func (r *ColdHotRunner) createAndCopyRemoteSuperNodeConfig(ctx context.Context) error {

	supernodeConfigPath := "supernode.yml"
	log.WithContext(ctx).Infof("Creating remote supernode config - %s", supernodeConfigPath)

	if _, err := os.Stat(supernodeConfigPath); os.IsNotExist(err) {
		// create new
		if err = utils.CreateFile(ctx, supernodeConfigPath, true); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to create new supernode.yml file at - %s", supernodeConfigPath)
			return err
		}

		portList := GetSNPortList(r.config)

		snTempDirPath := filepath.Join(r.config.RemoteHotWorkingDir, constants.TempDir)
		rqWorkDirPath := filepath.Join(r.config.RemoteHotWorkingDir, constants.RQServiceDir)
		p2pDataPath := filepath.Join(r.config.RemoteHotWorkingDir, constants.P2PDataDir)
		mdlDataPath := filepath.Join(r.config.RemoteHotWorkingDir, constants.MDLDataDir)
		ddDirPath := filepath.Join(r.config.RemoteHotHomeDir, constants.DupeDetectionServiceDir)

		toolConfig, err := utils.GetServiceConfig(string(constants.SuperNode), configs.SupernodeDefaultConfig, &configs.SuperNodeConfig{
			LogFilePath:                     r.config.Configurer.GetSuperNodeLogFile(r.config.RemoteHotWorkingDir),
			LogCompress:                     constants.LogConfigDefaultCompress,
			LogMaxSizeMB:                    constants.LogConfigDefaultMaxSizeMB,
			LogMaxAgeDays:                   constants.LogConfigDefaultMaxAgeDays,
			LogMaxBackups:                   constants.LogConfigDefaultMaxBackups,
			LogLevelCommon:                  constants.SuperNodeDefaultCommonLogLevel,
			LogLevelP2P:                     constants.SuperNodeDefaultP2PLogLevel,
			LogLevelMetadb:                  constants.SuperNodeDefaultMetaDBLogLevel,
			LogLevelDD:                      constants.SuperNodeDefaultDDLogLevel,
			SNTempDir:                       snTempDirPath,
			SNWorkDir:                       r.config.RemoteHotWorkingDir,
			RQDir:                           rqWorkDirPath,
			DDDir:                           ddDirPath,
			SuperNodePort:                   portList[constants.SNPort],
			P2PPort:                         portList[constants.P2PPort],
			P2PDataDir:                      p2pDataPath,
			MDLPort:                         portList[constants.MDLPort],
			RAFTPort:                        portList[constants.RAFTPort],
			MDLDataDir:                      mdlDataPath,
			RaptorqPort:                     constants.RQServiceDefaultPort,
			DDServerPort:                    constants.DDServerDefaultPort,
			NumberOfChallengeReplicas:       constants.NumberOfChallengeReplicas,
			StorageChallengeExpiredDuration: constants.StorageChallengeExpiredDuration,
			PasteID:                         flagMasterNodePastelID,
			Passphrase:                      flagMasterNodePassPhrase,
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
		node["storage_challenge_expired_duration"] = constants.StorageChallengeExpiredDuration
		node["number_of_challenge_replicas"] = constants.NumberOfChallengeReplicas

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

	remoteSnConfigPath := r.config.Configurer.GetSuperNodeConfFile(r.config.RemoteHotWorkingDir)
	remoteSnConfigPath = strings.ReplaceAll(remoteSnConfigPath, "\\", "/")

	log.WithContext(ctx).Info("copying supernode config..")
	if err := r.sshClient.Scp(supernodeConfigPath, remoteSnConfigPath, "0644"); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to copy pastelup executable to remote host")
		return err
	}

	if err := utils.DeleteFile(supernodeConfigPath); err != nil {
		log.WithContext(ctx).Errorf("Failed to delete archive file : %s", supernodeConfigPath)
		return err
	}

	if err := r.sshClient.ShellCmd(ctx, fmt.Sprintf("chmod 755 %s", remoteSnConfigPath)); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to change permission of pastelup")
		return err
	}

	return nil
}

func (r *ColdHotRunner) copyMasterNodeConToRemote(ctx context.Context) error {
	mnConf := make(map[string]masterNodeConf)

	mnConf[flagMasterNodeName] = masterNodeConf{
		MnAddress:  flagNodeExtIP + ":" + fmt.Sprintf("%d", flagMasterNodePort),
		MnPrivKey:  flagMasterNodePrivateKey,
		Txid:       flagMasterNodeTxID,
		OutIndex:   flagMasterNodeInd,
		ExtAddress: flagNodeExtIP + ":" + fmt.Sprintf("%d", flagMasterNodeRPCPort),
		ExtP2P:     flagMasterNodeP2PIP + ":" + fmt.Sprintf("%d", flagMasterNodeP2PPort),
		ExtCfg:     "",
		ExtKey:     flagMasterNodePastelID,
	}
	confData, err := json.Marshal(mnConf)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Invalid new masternode conf data")
		return err
	}

	tmpMNConfPath := "/tmp/mn.conf"
	if err := ioutil.WriteFile(tmpMNConfPath, confData, 0644); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to create and write temporary masternode.conf file at '/tmp/mn.conf'")
		return err
	}

	hotMasternodeConfPath := getMasternodeConfPath(r.config, r.config.RemoteHotWorkingDir, "masternode.conf")
	if err := r.sshClient.Scp(tmpMNConfPath, hotMasternodeConfPath, "0644"); err != nil {
		return fmt.Errorf("failed to copy masternode.conf to remote %s", err)
	}
	return nil
}

///// hermes.yml helpers
func (r *ColdHotRunner) createAndCopyRemoteHermesConfig(ctx context.Context) error {

	hermesConfigPath := "hermes.yml"
	log.WithContext(ctx).Infof("Creating remote hermes config - %s", hermesConfigPath)

	if _, err := os.Stat(hermesConfigPath); os.IsNotExist(err) {
		// create new
		if err = utils.CreateFile(ctx, hermesConfigPath, true); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to create new hermes.yml file at - %s", hermesConfigPath)
			return err
		}

		portList := GetSNPortList(r.config)

		snTempDirPath := filepath.Join(r.config.RemoteHotWorkingDir, constants.TempDir)
		ddDirPath := filepath.Join(r.config.RemoteHotHomeDir, constants.DupeDetectionServiceDir)

		toolConfig, err := utils.GetServiceConfig(string(constants.Hermes), configs.HermesDefaultConfig, &configs.HermesConfig{
			LogFilePath:    r.config.Configurer.GetSuperNodeLogFile(r.config.RemoteHotWorkingDir),
			LogCompress:    constants.LogConfigDefaultCompress,
			LogMaxSizeMB:   constants.LogConfigDefaultMaxSizeMB,
			LogMaxAgeDays:  constants.LogConfigDefaultMaxAgeDays,
			LogMaxBackups:  constants.LogConfigDefaultMaxBackups,
			LogLevelCommon: constants.SuperNodeDefaultCommonLogLevel,
			LogLevelP2P:    constants.SuperNodeDefaultP2PLogLevel,
			LogLevelMetadb: constants.SuperNodeDefaultMetaDBLogLevel,
			LogLevelDD:     constants.SuperNodeDefaultDDLogLevel,
			SNTempDir:      snTempDirPath,
			SNWorkDir:      r.config.RemoteHotWorkingDir,
			DDDir:          ddDirPath,
			PastelID:       flagMasterNodePastelID,
			Passphrase:     flagMasterNodePassPhrase,
			SNHost:         "localhost",
			SNPort:         portList[constants.SNPort],
		})
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to get hermes config")
			return err
		}
		if err = utils.WriteFile(hermesConfigPath, toolConfig); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to update new hermes.yml file at - %s", hermesConfigPath)
			return err
		}

	} else if err == nil {
		//update existing
		var hermesConfFile []byte
		hermesConfFile, err = ioutil.ReadFile(hermesConfigPath)
		if err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to open existing hermes.yml file at - %s", hermesConfigPath)
			return err
		}
		hermesConf := make(map[string]interface{})
		if err = yaml.Unmarshal(hermesConfFile, &hermesConf); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to parse existing hermes.yml file at - %s", hermesConfigPath)
			return err
		}

		hermesConf["pastel_id"] = flagMasterNodePastelID
		hermesConf["pass_phrase"] = flagMasterNodePassPhrase

		var hermesConfFileUpdated []byte
		if hermesConfFileUpdated, err = yaml.Marshal(&hermesConf); err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to unparse yml for hermes.yml file at - %s", hermesConfigPath)
			return err
		}
		if ioutil.WriteFile(hermesConfigPath, hermesConfFileUpdated, 0644) != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to update hermes.yml file at - %s", hermesConfigPath)
			return err
		}
	} else {
		log.WithContext(ctx).WithError(err).Errorf("Failed to update or create hermes.yml file at - %s", hermesConfigPath)
		return err
	}

	log.WithContext(ctx).Info("Hermes config updated")

	remoteHermesConfigPath := r.config.Configurer.GetHermesConfFile(r.config.RemoteHotWorkingDir)
	remoteHermesConfigPath = strings.ReplaceAll(remoteHermesConfigPath, "\\", "/")

	log.WithContext(ctx).Info("copying hermes config..")
	if err := r.sshClient.Scp(hermesConfigPath, remoteHermesConfigPath, "0644"); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to copy pastelup executable to remote host")
		return err
	}

	if err := utils.DeleteFile(hermesConfigPath); err != nil {
		log.WithContext(ctx).Errorf("Failed to delete archive file : %s", hermesConfigPath)
		return err
	}

	if err := r.sshClient.ShellCmd(ctx, fmt.Sprintf("chmod 755 %s", remoteHermesConfigPath)); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to change permission of pastelup")
		return err
	}

	return nil
}
