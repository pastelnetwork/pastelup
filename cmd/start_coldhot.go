package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
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
	remotePasteld   string
	remotePastelCli string
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

	/*// TBD --- Not sure When to register..
	if err := r.registerTicketPastelID(ctx); err != nil {
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
	if err = runPastelServiceRemote(ctx, r.config, constants.RQService, r.sshClient); err != nil {
		return err
	}

	// ***************  7. Start supernode  **************
	err = runSuperNodeRemote(ctx, r.config, r.sshClient /*, extIP, pastelid*/)
	if err != nil {
		return err
	}

	return nil
}

func (r *ColdHotRunner) runRemoteNodeAsMasterNode(ctx context.Context) error {
	cmdLine := fmt.Sprintf("%s --masternode --txindex=1 --reindex --masternodeprivkey=%s --externalip=%s %s --daemon",
		r.opts.remotePasteld, flagMasterNodePrivateKey, flagNodeExtIP, r.opts.testnetOption)
	log.WithContext(ctx).Infof("%s\n", cmdLine)
	go r.sshClient.Cmd(cmdLine).Run()

	// TODO (Matee): check pastelD running on remote instead of wait
	time.Sleep(10 * time.Second)

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

	// TODO (Matee): check pastelD running on remote instead of wait
	time.Sleep(10 * time.Second)

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

	// TODO (Matee): check pastelD NOT running on remote instead of wait
	time.Sleep(5 * time.Second)

	if flagMasterNodeIsCreate {
		if _, err = backupConfFile(ctx, r.config); err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to backup masternode.conf")
			return err
		}
	}

	return nil
}

// Not yet tested
func runSuperNodeRemote(ctx context.Context, config *configs.Config, client *utils.Client /*, extIP string, pastelid string*/) (err error) {
	log.WithContext(ctx).Info("Remote:::Starting supernode")
	log.WithContext(ctx).Debug("Remote:::Configure supernode setting")

	log.WithContext(ctx).Info("Remote:::Configuring supernode was finished")
	log.WithContext(ctx).Info("Remote:::Start supernode")

	remoteWorkDirPath, remotePastelExecPath, remoteOsType, err := getRemoteInfo(config, client)
	if err != nil {
		return err
	}

	remoteSuperNodeConfigFilePath := config.Configurer.GetSuperNodeConfFile(string(remoteWorkDirPath))

	var remoteSupernodeExecFile string

	remoteSuperNodeConfigFilePath = strings.ReplaceAll(remoteSuperNodeConfigFilePath, "\\", "/")
	remoteSupernodeExecFile = filepath.Join(string(remotePastelExecPath), constants.SuperNodeExecName[constants.OSType(string(remoteOsType))])
	remoteSupernodeExecFile = strings.ReplaceAll(remoteSupernodeExecFile, "\\", "/")

	time.Sleep(5 * time.Second)

	log.WithContext(ctx).Infof("Remote:::Start supernode command : %s", fmt.Sprintf("%s %s", remoteSupernodeExecFile, fmt.Sprintf("--config-file=%s", remoteSuperNodeConfigFilePath)))

	go client.Cmd(fmt.Sprintf("%s %s", remoteSupernodeExecFile,
		fmt.Sprintf("--config-file=%s", remoteSuperNodeConfigFilePath))).Run()

	defer client.Close()

	log.WithContext(ctx).Info("Remote:::Waiting for supernode started...")
	time.Sleep(5 * time.Second)

	log.WithContext(ctx).Info("Remote:::Supernode was started successfully")
	return nil
}

// not yet tested
func runPastelServiceRemote(ctx context.Context, config *configs.Config, tool constants.ToolType, client *utils.Client) (err error) {
	commandName := filepath.Base(string(tool))
	log.WithContext(ctx).Infof("Remote:::Starting %s", commandName)

	remoteWorkDirPath, remotePastelExecPath, remoteOsType, err := getRemoteInfo(config, client)
	if err != nil {
		return err
	}

	switch tool {
	case constants.RQService:
		remoteRQServiceConfigFilePath := config.Configurer.GetRQServiceConfFile(string(remoteWorkDirPath))

		remoteRQServiceConfigFilePath = strings.ReplaceAll(remoteRQServiceConfigFilePath, "\\", "/")

		pastelRqServicePath := filepath.Join(string(remotePastelExecPath), constants.PastelRQServiceExecName[constants.OSType(string(remoteOsType))])
		pastelRqServicePath = strings.ReplaceAll(pastelRqServicePath, "\\", "/")

		go client.Cmd(fmt.Sprintf("%s %s", pastelRqServicePath, fmt.Sprintf("--config-file=%s", remoteRQServiceConfigFilePath))).Run()

		time.Sleep(10 * time.Second)

	}

	log.WithContext(ctx).Infof("Remote:::The %s started succesfully!", commandName)
	return nil
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
