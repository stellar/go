package expingest

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stretchr/testify/assert"
)

func TestCheckVerifyStateVersion(t *testing.T) {
	assert.Equal(
		t,
		history.CurrentExpIngestVersion,
		stateVerifierExpectedIngestionVersion,
		"State verifier is outdated, update it, then update stateVerifierExpectedIngestionVersion value",
	)
}
