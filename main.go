package main

import (
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

	log.FatalAndExit(err)

}

func init() {
	log.SetDebugMode(debugMode)
}
