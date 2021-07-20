package ledgerbackend

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdbfoundation/go/support/log"
)

func TestGenerateConfig(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		appendPath   string
		mode         digitalbitsCoreRunnerMode
		expectedPath string
	}{
		{
			name:         "offline config with no appendix",
			mode:         digitalbitsCoreRunnerModeOffline,
			appendPath:   "",
			expectedPath: filepath.Join("testdata", "expected-offline-core.cfg"),
		},
		{
			name:         "online config with appendix",
			mode:         digitalbitsCoreRunnerModeOnline,
			appendPath:   filepath.Join("testdata", "sample-appendix.cfg"),
			expectedPath: filepath.Join("testdata", "expected-online-core.cfg"),
		},
		{
			name:         "offline config with appendix",
			mode:         digitalbitsCoreRunnerModeOffline,
			appendPath:   filepath.Join("testdata", "sample-appendix.cfg"),
			expectedPath: filepath.Join("testdata", "expected-offline-with-appendix-core.cfg"),
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			digitalbitsCoreRunner, err := newDigitalBitsCoreRunner(CaptiveCoreConfig{
				HTTPPort:           6789,
				HistoryArchiveURLs: []string{"http://localhost:1170"},
				Log:                log.New(),
				ConfigAppendPath:   testCase.appendPath,
				Context:            context.Background(),
				NetworkPassphrase:  "LiveNet Global DigitalBits Network ; February 2021",
			}, testCase.mode)
			assert.NoError(t, err)

			tempDir := digitalbitsCoreRunner.tempDir
			digitalbitsCoreRunner.tempDir = "/test-temp-dir"

			config, err := digitalbitsCoreRunner.generateConfig()
			assert.NoError(t, err)

			expectedByte, err := ioutil.ReadFile(testCase.expectedPath)
			assert.NoError(t, err)

			assert.Equal(t, config, string(expectedByte))

			digitalbitsCoreRunner.tempDir = tempDir
			assert.NoError(t, digitalbitsCoreRunner.close())
		})
	}
}

func TestCloseBeforeStart(t *testing.T) {
	runner, err := newDigitalBitsCoreRunner(CaptiveCoreConfig{
		HistoryArchiveURLs: []string{"http://localhost"},
		Log:                log.New(),
		Context:            context.Background(),
	}, digitalbitsCoreRunnerModeOffline)
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
