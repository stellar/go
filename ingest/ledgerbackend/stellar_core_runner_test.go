package ledgerbackend

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/support/log"
)

func TestCloseBeforeStart(t *testing.T) {
	captiveCoreToml, err := NewCaptiveCoreToml(CaptiveCoreTomlParams{})
	assert.NoError(t, err)

	runner, err := newStellarCoreRunner(CaptiveCoreConfig{
		HistoryArchiveURLs: []string{"http://localhost"},
		Log:                log.New(),
		Context:            context.Background(),
		Toml:               captiveCoreToml,
	}, stellarCoreRunnerModeOffline)
	assert.NoError(t, err)

	tempDir := runner.storagePath
	info, err := os.Stat(tempDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())

	assert.NoError(t, runner.close())

	// Directory no longer cleaned up on shutdown (perf. bump in v2.5.0)
	_, err = os.Stat(tempDir)
	assert.NoError(t, err)
}
