package ledgerbackend

import (
	"io"
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
	resetTmpFile := func() {
		err := tmpFile.Truncate(0)
		assert.NoError(t, err)
		_, err = tmpFile.Seek(0, io.SeekStart)
		assert.NoError(t, err)

	}

	r := stellarCoreRunner{
		quorumConfigPath: tmpFile.Name(),
	}

	tmpFile.WriteString(`[QUORUM_SET]
THRESHOLD_PERCENT=100
VALIDATORS=["GCZBOIAY4HLKAJVNJORXZOZRAY2BJDBZHKPBHZCRAIUR5IHC2UHBGCQR"]
`)
	_, err = r.generateConfig()
	assert.NoError(t, err)

	resetTmpFile()
	tmpFile.WriteString(`QUORUM_SET=foo
`)
	_, err = r.generateConfig()
	assert.Error(t, err)

	resetTmpFile()
	tmpFile.WriteString(`[QUORUM_SET.foo]
THRESHOLD_PERCENT=100
VALIDATORS=["GCZBOIAY4HLKAJVNJORXZOZRAY2BJDBZHKPBHZCRAIUR5IHC2UHBGCQR"]

[QUORUM_SET.bar]
THRESHOLD_PERCENT=100
VALIDATORS=["GCZBOIAY4HLKAJVNJORXZOZRAY2BJDBZHKPBHZCRAIUR5IHC2UHBGCQR"]
`)
	_, err = r.generateConfig()
	assert.NoError(t, err)

	resetTmpFile()
	tmpFile.WriteString(`[FOO]
THRESHOLD_PERCENT=100
VALIDATORS=["GCZBOIAY4HLKAJVNJORXZOZRAY2BJDBZHKPBHZCRAIUR5IHC2UHBGCQR"]
`)
	_, err = r.generateConfig()
	assert.Error(t, err)
}
