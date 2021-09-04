package configurer

import (
	"fmt"
	"testing"

	"github.com/pastelnetwork/pastel-utility/constants"
	"github.com/pastelnetwork/pastel-utility/utils"

	"github.com/tj/assert"
)

func TestGetDownloadUrl(t *testing.T) {
	testConfigurer, _ := NewConfigurer()
	downloadUrl, fileName, _ := testConfigurer.GetDownloadURL("beta", constants.WalletNode)
	assert.Equal(t, constants.WalletNodeExecName[utils.GetOS()], fileName)
	assert.Equal(t, fmt.Sprintf("/beta/gonode/%s", constants.WalletNodeExecName[utils.GetOS()]), downloadUrl.Path)
}

func TestGetDownloadGitURL(t *testing.T) {
	testConfigurer, _ := NewConfigurer()
	downloadUrl, fileName, _ := testConfigurer.GetDownloadGitURL("v1.0", constants.WalletNode)
	assert.Equal(t, constants.WalletNodeExecName[utils.GetOS()], fileName)
	assert.Equal(t, fmt.Sprintf("/pastelnetwork/gonode/releases/download/v1.0/%s", constants.WalletNodeExecName[utils.GetOS()]), downloadUrl.Path)
	fmt.Println(fileName)
}

func TestGetDownloadGitChecksumURL(t *testing.T) {
	testConfigurer, _ := NewConfigurer()
	downloadUrl, fileName, _ := testConfigurer.GetDownloadGitChecksumURL("v1.0", constants.WalletNode)
	assert.Equal(t, constants.WalletNodeExecChecksumName[utils.GetOS()], fileName)
	assert.Equal(t, fmt.Sprintf("/pastelnetwork/gonode/releases/download/v1.0/%s", constants.WalletNodeExecChecksumName[utils.GetOS()]), downloadUrl.Path)
	fmt.Println(fileName)
}
