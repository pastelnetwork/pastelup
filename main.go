package main

import (
	"os"

	"github.com/pastelnetwork/pastelup/cmd"
	"github.com/pastelnetwork/pastelup/common/errors"
	"github.com/pastelnetwork/pastelup/common/log"
	"github.com/pastelnetwork/pastelup/common/sys"
)

const (
	debugModeEnvName = "PASTEL_UTILITY_DEBUG"
)

var (
	debugMode = sys.GetBoolEnv(debugModeEnvName, false)
)

func main() {
	defer errors.Recover(log.FatalAndExit)
	app := cmd.NewApp(os.Args)
	err := app.Run(os.Args)
	log.FatalAndExit(err)
}

func init() {
	log.SetDebugMode(debugMode)
}
