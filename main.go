package main

import (
	"os"

	"github.com/edentech88/pastel-utility/cmd"
	"github.com/pastelnetwork/gonode/common/errors"
	"github.com/pastelnetwork/gonode/common/log"
	"github.com/pastelnetwork/gonode/common/sys"
)

const (
	debugModeEnvName = "PASTEL_UTILITY_DEBUG"
)

var (
	debugMode = sys.GetBoolEnv(debugModeEnvName, false)
)

func main() {
	defer errors.Recover(log.FatalAndExit)

	app := cmd.NewApp()
	err := app.Run(os.Args)

	log.FatalAndExit(err)

}

func init() {
	log.SetDebugMode(debugMode)
}
