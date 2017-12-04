package ingest

import (
	"testing"

	"github.com/stellar/go/network"
	"github.com/stellar/go/services/horizon/internal/test"
)

func TestClearAll(t *testing.T) {
	tt := test.Start(t).Scenario("kahuna")
	defer tt.Finish()
	is := sys(tt)

	err := is.ClearAll()

	tt.Require.NoError(err)

	// ensure no ledgers
	var found int
	err = tt.HorizonSession().GetRaw(&found, "SELECT COUNT(*) FROM history_ledgers")
	tt.Require.NoError(err)
	tt.Assert.Equal(0, found)
}

func TestValidation(t *testing.T) {
	tt := test.Start(t).ScenarioWithoutHorizon("kahuna")
	defer tt.Finish()

	sys := New(network.TestNetworkPassphrase, "", tt.CoreSession(), tt.HorizonSession())

	// intact chain
	for i := int32(2); i <= 57; i++ {
		tt.Assert.NoError(sys.validateLedgerChain(i))
	}
	_, err := tt.CoreSession().ExecRaw(
		`DELETE FROM ledgerheaders WHERE ledgerseq = ?`, 5,
	)
	tt.Require.NoError(err)

	// missing cur
	err = sys.validateLedgerChain(5)
	tt.Assert.Error(err)
	tt.Assert.Contains(err.Error(), "failed to load cur ledger")

	// missing prev
	err = sys.validateLedgerChain(6)
	tt.Assert.Error(err)
	tt.Assert.Contains(err.Error(), "failed to load prev ledger")

	// mismatched header
	_, err = tt.CoreSession().ExecRaw(`
		UPDATE ledgerheaders
		SET ledgerhash = ?
		WHERE ledgerseq = ?`, "00000", 8)
	tt.Require.NoError(err)

	err = sys.validateLedgerChain(9)
	tt.Assert.Error(err)
	tt.Assert.Contains(err.Error(), "cur and prev ledger hashes don't match")
}

// TestSystem_newTickSession tests the ledger that newTickSession picks to start
// ingestion from in various scenarios.
func TestSystem_newTickSession(t *testing.T) {
	tt := test.Start(t).ScenarioWithoutHorizon("kahuna")
	defer tt.Finish()

	sys := New(network.TestNetworkPassphrase, "", tt.CoreSession(), tt.HorizonSession())

	sess, err := sys.newTickSession()
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(int32(1), sess.Cursor.FirstLedger)
		tt.Assert.Equal(int32(57), sess.Cursor.LastLedger)
	}

	// when HistoryRetentionCount is set, start with the first importable ledger
	sys.HistoryRetentionCount = 10

	sess, err = sys.newTickSession()
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(int32(48), sess.Cursor.FirstLedger)
		tt.Assert.Equal(int32(57), sess.Cursor.LastLedger)
	}

	// when a gap exists where the first importable ledger should be, pick the
	// newest after the gap
	_, err = tt.CoreSession().ExecRaw(`
		DELETE FROM ledgerheaders
		WHERE ledgerseq BETWEEN 35 AND 50`)
	tt.Require.NoError(err)

	sess, err = sys.newTickSession()
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(int32(51), sess.Cursor.FirstLedger)
		tt.Assert.Equal(int32(57), sess.Cursor.LastLedger)
	}

	// when the history database is populated, start at the end of ingested
	// history
	sess = sys.Tick()
	tt.Require.NoError(sess.Err)
	tt.UpdateLedgerState()

	sess, err = sys.newTickSession()
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(int32(57), sess.Cursor.FirstLedger)
		tt.Assert.Equal(int32(57), sess.Cursor.LastLedger)
	}

	// sanity test: ensure no error when re-ticking with a synced horizon db.
	sess = sys.Tick()
	tt.Assert.NoError(sess.Err)

	// prep for next scenario
	err = sys.ClearAll()
	tt.Require.NoError(err)
	tt.UpdateLedgerState()

	// establish a reingestion start point
	err = sys.ReingestSingle(int32(52))
	tt.Require.NoError(err)
	tt.UpdateLedgerState()

	sess, err = sys.newTickSession()
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(int32(53), sess.Cursor.FirstLedger)
		tt.Assert.Equal(int32(57), sess.Cursor.LastLedger)
	}
}
