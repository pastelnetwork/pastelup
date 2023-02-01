package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/pastelup/configs"
	"github.com/pastelnetwork/pastelup/constants"
	"github.com/pastelnetwork/pastelup/services/pastelcore"
	"github.com/pastelnetwork/pastelup/utils"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// GetSNConfigs returns SN configs
func GetSNConfigs(config *configs.Config) (string, error) {
	portList := GetSNPortList(config)

	snTempDirPath := filepath.Join(config.WorkingDir, constants.TempDir)
	rqWorkDirPath := filepath.Join(config.WorkingDir, constants.RQServiceDir)
	p2pDataPath := filepath.Join(config.WorkingDir, constants.P2PDataDir)
	mdlDataPath := filepath.Join(config.WorkingDir, constants.MDLDataDir)
	ddDirPath := filepath.Join(config.Configurer.DefaultHomeDir(), constants.DupeDetectionServiceDir)

	toolConfig, err := utils.GetServiceConfig(string(constants.SuperNode), configs.SupernodeDefaultConfig, &configs.SuperNodeConfig{
		LogFilePath:                     config.Configurer.GetSuperNodeLogFile(config.WorkingDir),
		LogCompress:                     constants.LogConfigDefaultCompress,
		LogMaxSizeMB:                    constants.LogConfigDefaultMaxSizeMB,
		LogMaxAgeDays:                   constants.LogConfigDefaultMaxAgeDays,
		LogMaxBackups:                   constants.LogConfigDefaultMaxBackups,
		LogLevelCommon:                  constants.SuperNodeDefaultCommonLogLevel,
		LogLevelP2P:                     constants.SuperNodeDefaultP2PLogLevel,
		LogLevelMetadb:                  constants.SuperNodeDefaultMetaDBLogLevel,
		LogLevelDD:                      constants.SuperNodeDefaultDDLogLevel,
		SNTempDir:                       snTempDirPath,
		SNWorkDir:                       config.WorkingDir,
		RQDir:                           rqWorkDirPath,
		DDDir:                           ddDirPath,
		SuperNodePort:                   portList[constants.SNPort],
		P2PPort:                         portList[constants.P2PPort],
		P2PDataDir:                      p2pDataPath,
		MDLPort:                         portList[constants.MDLPort],
		RAFTPort:                        portList[constants.RAFTPort],
		MDLDataDir:                      mdlDataPath,
		RaptorqPort:                     constants.RQServiceDefaultPort,
		NumberOfChallengeReplicas:       constants.NumberOfChallengeReplicas,
		StorageChallengeExpiredDuration: constants.StorageChallengeExpiredDuration,
		DDServerPort:                    constants.DDServerDefaultPort,
	})
	if err != nil {
		return "", errors.Errorf("failed to get supernode config: %v", err)
	}

	return toolConfig, nil
}

// GetHermesConfigs returns hermes configs
func GetHermesConfigs(config *configs.Config) (string, error) {
	portList := GetSNPortList(config)

	snTempDirPath := filepath.Join(config.WorkingDir, constants.TempDir)
	ddDirPath := filepath.Join(config.Configurer.DefaultHomeDir(), constants.DupeDetectionServiceDir)

	toolConfig, err := utils.GetServiceConfig(string(constants.Hermes), configs.HermesDefaultConfig, &configs.HermesConfig{
		LogFilePath:    config.Configurer.GetHermesLogFile(config.WorkingDir),
		LogCompress:    constants.LogConfigDefaultCompress,
		LogMaxSizeMB:   constants.LogConfigDefaultMaxSizeMB,
		LogMaxAgeDays:  constants.LogConfigDefaultMaxAgeDays,
		LogMaxBackups:  constants.LogConfigDefaultMaxBackups,
		LogLevelCommon: constants.SuperNodeDefaultCommonLogLevel,
		LogLevelP2P:    constants.SuperNodeDefaultP2PLogLevel,
		LogLevelMetadb: constants.SuperNodeDefaultMetaDBLogLevel,
		LogLevelDD:     constants.SuperNodeDefaultDDLogLevel,
		SNTempDir:      snTempDirPath,
		SNWorkDir:      config.WorkingDir,
		DDDir:          ddDirPath,
		SNHost:         "localhost",
		SNPort:         portList[constants.SNPort],
	})
	if err != nil {
		return "", errors.Errorf("failed to get supernode config: %v", err)
	}

	return toolConfig, nil
}

func checkBridgeConfigPastelID(ctx context.Context, config *configs.Config, confPath string) error {
	bridgeConfFile, err := ioutil.ReadFile(confPath)
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to open existing bridge.yml file at - %s", confPath)
		return err
	}

	bridgeConf := make(map[string]interface{})
	if err = yaml.Unmarshal(bridgeConfFile, &bridgeConf); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to parse existing hermes.yml file at - %s", confPath)
		return err
	}

	download := bridgeConf["download"].(map[interface{}]interface{})

	var pastelid, passphrase string
	if download["pastel_id"] != nil {
		pastelid = download["pastel_id"].(string)
	}
	if download["passphrase"] != nil {
		passphrase = download["passphrase"].(string)
	}

	if pastelid != "" && passphrase != "" {
		log.WithContext(ctx).Info("Bridge service found pastelid & pass")
		return nil
	}

	log.WithContext(ctx).Warn(red("Bridge service config file is missing PastelID and passphrase."))
	log.WithContext(ctx).Warn(red("This is probably the first time you're starting bridge service."))

	pastelIDRegistered := false
	var res map[string]interface{}
	err = pastelcore.NewClient(config).RunCommandWithArgs(
		pastelcore.PastelIDCmd,
		[]string{"list"},
		&res,
	)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to list existing pastelid keys")
	} else {
		keys := res["result"].([]interface{})
		if keys != nil && len(keys) > 0 {
			log.WithContext(ctx).Warn(red("You pastel has some PastelID, You can choose one of them to use for bridge service.\n"))
			n := 0
			var arr []string
			for _, item := range keys {
				keyPair := item.(map[string]interface{})
				key := keyPair["PastelID"].(string)
				log.WithContext(ctx).Warn(red(fmt.Sprintf("%d - %s", n, key)))
				arr = append(arr, key)
				n++
			}
			_, strNum := AskUserToContinue(ctx, "Enter number of PastelID to use, 'Y' to generate new one or 'N' to skip...")
			if strings.EqualFold(strNum, "N") {
				return nil
			} else if !strings.EqualFold(strNum, "Y") {
				dNum, err := strconv.Atoi(strNum)
				if err != nil || dNum < 0 || dNum >= n {
					err = fmt.Errorf("wrong input - no PastelID selected")
				} else {
					pastelid = arr[dNum]
					log.WithContext(ctx).Info(green("You selected PastelID: " + pastelid))
					_, passphrase = AskUserToContinue(ctx, "Enter its Passphrase...")
					//TODO: check if pastelID registered or not
					pastelIDRegistered = true
				}
			}
		}
	}

	if pastelid == "" || passphrase == "" {
		if pastelidOK, _ := AskUserToContinue(ctx, "Do you want to create new PastelID? Y/N"); pastelidOK {
			_, passphrase = AskUserToContinue(ctx, "Enter Passphrase for new pastelid")
			var resp map[string]interface{}
			err = pastelcore.NewClient(config).RunCommandWithArgs(
				pastelcore.PastelIDCmd,
				[]string{"newkey", passphrase},
				&resp,
			)
			if err != nil {
				log.WithContext(ctx).WithError(err).Error("Failed to generate new pastelid key")
				return err
			}

			res := resp["result"].(map[string]interface{})
			pastelid = res["pastelid"].(string)

			log.WithContext(ctx).WithField("pastelid", pastelid).WithField("legroastKey", res["legRoastKey"].(string)).Info("Please save these generated keys")
		}
	}

	doRegisterPastelID := false
	if pastelid != "" && passphrase != "" && !pastelIDRegistered {
		log.WithContext(ctx).Warn(red("New PastelID must be registered in the network before it can be used."))
		log.WithContext(ctx).Warn(red("You can do it later after all services started by command `pastel-cli tickets register id ..."))
		log.WithContext(ctx).Warn(red("Or we can try register it now, but that can take long time (mostly waiting for network sync).\n"))
		doRegisterPastelID, _ = AskUserToContinue(ctx, "Do you want to try register it now? Y/N")
	}

	if doRegisterPastelID {
		var regResp map[string]interface{}
		_, address := AskUserToContinue(ctx, "Enter Address to register pastelid against - please make sure it has enough balance.\nLeave Empty(Press Enter) to generate a new address")

		if address == "" {
			out, err := RunPastelCLI(ctx, config, "getnewaddress")
			if err != nil {
				log.WithContext(ctx).WithError(err).WithField("out", string(out)).
					Error("Failed to generate new address")

				return err
			}

			address = out
			log.WithContext(ctx).WithField("address", address).Info("Please save the newly generated address")
			log.WithContext(ctx).WithField("address", address).Warn("And send some Pastel coins to it")
		}

		ok, err := handleWaitForBalance(ctx, config, address)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("error handle wait for balance")
			return err
		}

		if ok {
			err = pastelcore.NewClient(config).RunCommandWithArgs(
				pastelcore.TicketsCmd,
				[]string{"register", "id", pastelid, passphrase, address},
				&regResp,
			)
			if err != nil {
				log.WithContext(ctx).WithError(err).Error("Failed to register new pastelid key")
				return err
			}

			regRes := regResp["result"].(map[string]interface{})
			txid := regRes["txid"].(string)
			if txid == "" {
				log.WithContext(ctx).WithError(err).WithField("res", regRes).Error("Failed to register new pastelid: txid not found")
				return err
			}
			pastelIDRegistered = true
		}
	}
	if !pastelIDRegistered {
		_, _ = AskUserToContinue(ctx, "Ignoring ticket registration. Please register pastelid on network for proper functioning of Walletnode & Bridge service\nPress Enter to continue")
	}

	if pastelid == "" && passphrase == "" {
		return nil
	}

	download["pastel_id"] = pastelid
	download["passphrase"] = passphrase
	bridgeConf["download"] = download

	var bridgeConfFileUpdated []byte
	if bridgeConfFileUpdated, err = yaml.Marshal(&bridgeConf); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to unparse yml for bridge.yml file at - %s", confPath)
		return err
	}

	if ioutil.WriteFile(confPath, bridgeConfFileUpdated, 0644) != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to update bridge.yml file at - %s", confPath)
		return err
	}

	log.WithContext(ctx).Info("bridge conf updated")
	return nil
}

func getBalance(ctx context.Context, config *configs.Config, address string) (balance float64, err error) {
	out, err := RunPastelCLI(ctx, config, "z_getbalance", address)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithField("out", string(out)).
			Error("Failed to get balance")

		return 0.0, err
	}
	strBalance := strings.TrimSpace(strings.Trim(out, "\n"))

	return strconv.ParseFloat(strBalance, 64)
}

func handleWaitForBalance(ctx context.Context, config *configs.Config, address string) (bool, error) {
	i := 0
	firstTime := true
	for {
		fmt.Println("checking for balance...")
		balance, err := getBalance(ctx, config, address)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("unable to get balance")
			return false, fmt.Errorf("handleWaitForBalance: getBalance: %s", err)
		}

		if balance >= minBalanceForTicketReg {
			return true, nil
		}
		if firstTime {
			if ok, _ := AskUserToContinue(ctx, `Not enough balance on the address. 
			Would you like to wait (It might take some time)? Y/N`); !ok {
				return false, nil
			}
			firstTime = false
		}
		time.Sleep(6 * time.Second)

		if i == 9 {
			log.WithContext(ctx).Warn(yellow("Enough balance not received yet"))

			mnstatus, err := GetMNSyncInfo(ctx, config)
			if err != nil {
				log.WithContext(ctx).WithError(err).Error("mnsync status call has failed")
				return false, err
			}

			if !mnstatus.Result.IsSynced {
				log.WithContext(ctx).Warn(red("It seems you node is not syncing"))
				getinfo, err := GetPastelInfo(ctx, config)
				if err != nil {
					log.WithContext(ctx).WithError(err).Error("getinfo call has failed")
					return false, err
				}
				log.WithContext(ctx).Warnf("You have %d blocks and %d connections", getinfo.Result.Blocks, getinfo.Result.Connections)

				//if getinfo.Result.Connections == 0 {
				//	log.WithContext(ctx).WithError(err).Warn(red("And you are not connected to any node"))
				//	if ok, _ := AskUserToContinue(ctx, `Do you want to try to connect and wait? Y/N`); ok {
				//		return false, nil
				//	}
				//	var resp interface{}
				//	err = pastelcore.NewClient(config).RunCommandWithArgs(
				//		pastelcore.AddNode,
				//		[]string{"list", networkSpecificAddress, "onetry"},
				//		&resp,
				//	)
				//	if err != nil {
				//		log.WithContext(ctx).WithError(err).Error("Failed to call addnode")
				//	}
				//}
			}
			if ok, _ := AskUserToContinue(ctx, `Would you like to continue to wait? Y/N`); !ok {
				return false, nil
			}
			i = 0
		}
		i++
	}
}

func checkBridgeEnabled(ctx context.Context, confPath string) (bool, error) {
	walletConfFile, err := ioutil.ReadFile(confPath)
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to open existing bridge.yml file at - %s", confPath)
		return false, err
	}

	walletConf := make(map[string]interface{})
	if err = yaml.Unmarshal(walletConfFile, &walletConf); err != nil {
		log.WithContext(ctx).WithError(err).Errorf("Failed to parse existing hermes.yml file at - %s", confPath)
		return false, err
	}

	if _, ok := walletConf["bridge"]; !ok {
		return false, nil
	}

	bridge := walletConf["bridge"].(map[interface{}]interface{})
	bridgeOn, ok := bridge["switch"].(bool)
	if !ok {
		return false, nil
	}

	return bridgeOn, nil
}
