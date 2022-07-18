package cmd

import (
	"path/filepath"

	"github.com/pastelnetwork/pastelup/configs"
	"github.com/pastelnetwork/pastelup/constants"
	"github.com/pastelnetwork/pastelup/utils"
	"github.com/pkg/errors"
)

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
	})
	if err != nil {
		return "", errors.Errorf("failed to get supernode config: %v", err)
	}

	return toolConfig, nil
}

func GetHermesConfigs(config *configs.Config) (string, error) {
	portList := GetSNPortList(config)

	snTempDirPath := filepath.Join(config.WorkingDir, constants.TempDir)
	ddDirPath := filepath.Join(config.Configurer.DefaultHomeDir(), constants.DupeDetectionServiceDir)

	toolConfig, err := utils.GetServiceConfig(string(constants.Hermes), configs.HermesDefaultConfig, &configs.HermesConfig{
		LogFilePath:    config.Configurer.GetSuperNodeLogFile(config.WorkingDir),
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
