package ledgerbackend

import (
	"testing"

	"github.com/stellar/go/support/log"
	logpkg "github.com/stellar/go/support/log"
	"github.com/stretchr/testify/assert"
)

// TODO: test frame decoding
// TODO: test from static base64-encoded data

func TestCaptiveCore(t *testing.T) {
	log.SetLevel(logpkg.InfoLevel)
	c := NewCaptive("Public Global Stellar Network ; September 2015",
		[]string{"http://history.stellar.org/prd/core-live/core_live_001"})
	seq, e := c.GetLatestLedgerSequence()
	assert.NoError(t, e)
	assert.Greater(t, seq, uint32(0))
	ok, lcm, e := c.GetLedger(seq - 200)
	assert.NoError(t, e)
	assert.Equal(t, true, ok)
	assert.Equal(t, uint32(lcm.LedgerHeader.Header.LedgerSeq), seq-200)
	assert.DirExists(t, c.getTmpDir())
	e = c.Close()
	assert.NoError(t, e)
}
