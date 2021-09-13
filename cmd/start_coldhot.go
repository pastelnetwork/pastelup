package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/pastel-utility/configs"
	"github.com/pastelnetwork/pastel-utility/constants"
	"github.com/pastelnetwork/pastel-utility/utils"
)

// TODO: Remove the use of shadowing global variables and decouple
// this part from rest of the code for better maintenance of codebase

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

type ColdHotRunner struct {
	sshClient *utils.Client
	config    *configs.Config
	opts      *ColdHotRunnerOpts
}

func (r *ColdHotRunner) Init(ctx context.Context) error {
	if err := r.HandleArgs(); err != nil {
		return fmt.Errorf("parse args: %s", err)
	}

	if err := r.HandleConfigs(ctx); err != nil {
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

func (r *ColdHotRunner) HandleArgs() (err error) {
	if len(r.config.RemotePastelUtilityDir) == 0 {
		return fmt.Errorf("cannot find remote pastel-utility dir")
	}

	if len(r.config.RemotePastelExecDir) == 0 {
		r.config.RemotePastelExecDir = r.config.Configurer.DefaultPastelExecutableDir()
	}

	r.opts.remotePastelCli = filepath.Join(r.config.RemotePastelExecDir, constants.PastelCliName[utils.GetOS()])
	r.opts.remotePasteld = filepath.Join(r.config.RemotePastelExecDir, constants.PasteldName[utils.GetOS()])
	r.opts.remotePastelUtility = filepath.Join(r.config.RemotePastelUtilityDir, "pastel-utility")
	return nil
}

func (r *ColdHotRunner) HandleConfigs(ctx context.Context) error {
	log.WithContext(ctx).Infof("reading pastel.conf")
	// Check pastel config for testnet option and set config.Network
	if err := ParsePastelConf(ctx, r.config); err != nil {
		return fmt.Errorf("parse pastel.conf: %s", err)
	}

	if r.config.Network == constants.NetworkTestnet {
		r.opts.testnetOption = " --testnet"
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

func (r *ColdHotRunner) Run(ctx context.Context) (err error) {
	// ***************  1. Start the local Pastel Network Node ***************
	log.WithContext(ctx).Infof("Starting pasteld")
	if err = runPastelNode(ctx, r.config, true, flagNodeExtIP, ""); err != nil {
		log.WithContext(ctx).WithError(err).Error("pasteld failed to start")
		return err
	}

	// ***************  2. If flag --create or --update is provided ***************
	if flagMasterNodeIsCreate || flagMasterNodeIsUpdate {
		log.WithContext(ctx).Info("Prepare mastenode parameters")
		if err := r.handleCreateUpdateStartColdHot(ctx, r.config, r.sshClient); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to validate and prepare masternode parameters")
			return err
		}
		if err := createOrUpdateMasternodeConf(ctx, r.config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to create or update masternode.conf")
			return err

		}
	}

	// ***************  3. Execute following commands over SSH on the remote node (using ssh-ip and ssh-port)  ***************

	if err = remoteHotNodeCtrl(ctx, r.config, r.sshClient); err != nil {
		log.WithContext(ctx).WithError(err).Error("failed on remoteHotNodeCtrl")
		return err
	}
	log.WithContext(ctx).Info("The hot wallet node has been successfully launched!")

	//Get conf data from masternode.conf File
	privkey, _, _, err := getMasternodeConfData(ctx, r.config, flagMasterNodeName)
	if err != nil {
		return err
	}
	flagMasterNodePrivateKey = privkey

	if err := r.runRemoteNodeAsMasterNode(ctx); err != nil {
		log.WithContext(ctx).WithError(err).Error("unable to run remote as masternode")
		return fmt.Errorf("run remote as masternode: %s", err)
	}

	// (TBD) when to run register ticket step
	/*if err := r.registerTicketPastelID(ctx); err != nil {
		log.WithContext(ctx).WithError(err).Error("unable to register pastelID ticket")
		return err
	}*/

	// ***************  4. If --activate are provided, ***************
	if flagMasterNodeIsActivate {
		log.WithContext(ctx).Info("--activate is on, activating mn...")
		if err = runStartAliasMasternode(ctx, r.config, flagMasterNodeName); err != nil {
			return err
		}
	}

	log.WithContext(ctx).Info("stopping cold node..")
	// ***************  5. Stop Cold Node  ***************
	if _, err = RunPastelCLI(ctx, r.config, "stop"); err != nil {
		return err
	}

	// *************  6. Start rq-servce    *************
	if err = r.runServiceRemote(ctx, string(constants.RQService)); err != nil {
		return fmt.Errorf("failed to start rq-service on hot node: %s", err)
	}

	// ***************  7. Start supernode  **************
	snService := fmt.Sprintf("%s-%s", string(constants.SuperNode), "service")
	if err = r.runServiceRemote(ctx, snService); err != nil {
		return fmt.Errorf("failed to start supernode-service on hot node: %s", err)
	}

	return nil
}

func (r *ColdHotRunner) runRemoteNodeAsMasterNode(ctx context.Context) error {
	cmdLine := fmt.Sprintf("%s --masternode --txindex=1 --reindex --masternodeprivkey=%s --externalip=%s %s --daemon",
		r.opts.remotePasteld, flagMasterNodePrivateKey, flagNodeExtIP, r.opts.testnetOption)
	log.WithContext(ctx).Infof("%s\n", cmdLine)
	go r.sshClient.Cmd(cmdLine).Run()

	if !CheckPastelDRunningRemote(ctx, r.sshClient, r.opts.remotePastelCli, true) {
		err := fmt.Errorf("unable to start pasteld on remote")
		log.WithContext(ctx).WithError(err).Error("run remote as master failed")
		return err
	}

	if err := checkMasterNodeSyncRemote(ctx, r.config, r.sshClient, r.opts.remotePastelCli); err != nil {
		log.WithContext(ctx).Error("Remote::Master node sync failed")
		return err
	}

	return nil
}
func (r *ColdHotRunner) handleCreateUpdateStartColdHot(ctx context.Context, config *configs.Config, client *utils.Client) (err error) {
	if err := checkCollateral(ctx, r.config); err != nil {
		log.WithContext(ctx).WithError(err).Error("Missing collateral transaction")
		return err
	}

	if err := checkPassphrase(ctx); err != nil {
		log.WithContext(ctx).WithError(err).Error("Missing passphrase")
		return err
	}

	go func() {
		if err := r.sshClient.Cmd(fmt.Sprintf("%s --reindex --externalip=%s --daemon %s",
			r.opts.remotePasteld, flagNodeExtIP, r.opts.testnetOption)).Run(); err != nil {
			fmt.Println("pasteld run err: ", err.Error())
		}
	}()

	if !CheckPastelDRunningRemote(ctx, r.sshClient, r.opts.remotePastelCli, true) {
		return fmt.Errorf("unable to start pasteld on remote: %s", err)
	}

	if err := checkMasternodePrivKey(ctx, r.config, r.sshClient); err != nil {
		log.WithContext(ctx).WithError(err).Error("Missing masternode private key")
		return err
	}

	if err := checkPastelID(ctx, r.config, r.sshClient); err != nil {
		log.WithContext(ctx).WithError(err).Error("Missing masternode PastelID")
		return err
	}

	if _, err := r.sshClient.Cmd(fmt.Sprintf("%s stop", r.opts.remotePastelCli)).Output(); err != nil {
		log.WithContext(ctx).Error("Error - stopping on remote pasteld")
		return err
	}

	if !CheckPastelDRunningRemote(ctx, r.sshClient, r.opts.remotePastelCli, false) {
		return fmt.Errorf("unable to start pasteld on remote: %s", err)
	}

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

func (r *ColdHotRunner) registerTicketPastelID(ctx context.Context) (err error) {
	cmd := fmt.Sprintf("%s %s %s %s", r.opts.remotePastelCli, "tickets register mnid",
		flagMasterNodePastelID, flagMasterNodePassPhrase)
	fmt.Println("run cmd for reg: ", cmd)
	out, err := r.sshClient.Cmd(cmd).Output()
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to register ticket mnid")
		return err
	}

	/*var pastelidSt structure.RPCPastelID
	if err = json.Unmarshal([]byte(pastelid), &pastelidSt); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to parse pastelid json")
		return err
	}
	flagMasterNodePastelID = pastelidSt.Pastelid*/

	log.WithContext(ctx).Infof("Register ticket pastelid result = %s", string(out))
	return nil
}

// CheckPastelDRunningRemote whether pasteld is running
func CheckPastelDRunningRemote(ctx context.Context, client *utils.Client, cliPath string, want bool) (ret bool) {
	var failCnt = 0
	var err error

	log.WithContext(ctx).Info("Waiting the pasteld to be started...")

	for {
		if _, err = client.Cmd(fmt.Sprintf("%s %s", cliPath, "getinfo")).Output(); err != nil {
			if !want {
				log.WithContext(ctx).Info("remote pasteld stopped.")
				return true
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

	log.WithContext(ctx).Info("remote pasteld started successfully")
	return true
}
