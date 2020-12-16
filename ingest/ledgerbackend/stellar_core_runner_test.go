package ledgerbackend

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/support/log"
)

func TestGenerateConfig(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "test-generate-config")
	assert.NoError(t, err)
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	r := stellarCoreRunner{
		configAppendPath: tmpFile.Name(),
	}

	tmpFile.WriteString(`[[HOME_DOMAINS]]
HOME_DOMAIN="testnet.stellar.org"
QUALITY="HIGH"

[[VALIDATORS]]
NAME="sdf_testnet_1"
HOME_DOMAIN="testnet.stellar.org"
PUBLIC_KEY="GDKXE2OZMJIPOSLNA6N6F2BVCI3O777I2OOC4BV7VOYUEHYX7RTRYA7Y"
ADDRESS="core-testnet1.stellar.org"
`)
	_, err = r.generateConfig()
	assert.NoError(t, err)
}

func TestCloseBeforeStart(t *testing.T) {
	runner, err := newStellarCoreRunner(CaptiveCoreConfig{
		HistoryArchiveURLs: []string{"http://localhost"},
		Log:                log.New(),
		Context:            context.Background(),
	}, stellarCoreRunnerModeOffline)
	assert.NoError(t, err)

	tempDir := runner.tempDir
	info, err := os.Stat(tempDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())

	assert.NoError(t, runner.close())

	_, err = os.Stat(tempDir)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}
