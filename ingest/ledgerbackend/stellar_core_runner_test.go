package ledgerbackend

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/support/log"
)

func TestGenerateConfig(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		appendPath   string
		mode         stellarCoreRunnerMode
		expectedPath string
	}{
		{
			name:         "offline config with no appendix",
			mode:         stellarCoreRunnerModeOffline,
			appendPath:   "",
			expectedPath: filepath.Join("testdata", "expected-offline-core.cfg"),
		},
		{
			name:         "online config with appendix",
			mode:         stellarCoreRunnerModeOnline,
			appendPath:   filepath.Join("testdata", "sample-appendix.cfg"),
			expectedPath: filepath.Join("testdata", "expected-online-core.cfg"),
		},
		{
			name:         "offline config with appendix",
			mode:         stellarCoreRunnerModeOffline,
			appendPath:   filepath.Join("testdata", "sample-appendix.cfg"),
			expectedPath: filepath.Join("testdata", "expected-offline-with-appendix-core.cfg"),
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			stellarCoreRunner, err := newStellarCoreRunner(CaptiveCoreConfig{
				HTTPPort:           6789,
				HistoryArchiveURLs: []string{"http://localhost:1170"},
				Log:                log.New(),
				ConfigAppendPath:   testCase.appendPath,
				StoragePath:        "./test-temp-dir",
				PeerPort:           12345,
				Context:            context.Background(),
				NetworkPassphrase:  "Public Global Stellar Network ; September 2015",
			}, testCase.mode)
			assert.NoError(t, err)

			config, err := stellarCoreRunner.generateConfig()
			assert.NoError(t, err)

			expectedByte, err := ioutil.ReadFile(testCase.expectedPath)
			assert.NoError(t, err)

			assert.Equal(t, config, string(expectedByte))

			assert.NoError(t, stellarCoreRunner.close())
		})
	}
}

func TestCloseBeforeStart(t *testing.T) {
	runner, err := newStellarCoreRunner(CaptiveCoreConfig{
		HistoryArchiveURLs: []string{"http://localhost"},
		Log:                log.New(),
		Context:            context.Background(),
	}, stellarCoreRunnerModeOffline)
	assert.NoError(t, err)

	tempDir := runner.storagePath
	info, err := os.Stat(tempDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())

	assert.NoError(t, runner.close())

	_, err = os.Stat(tempDir)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}
