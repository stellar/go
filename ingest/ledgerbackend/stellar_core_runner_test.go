package ledgerbackend

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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
