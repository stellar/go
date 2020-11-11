package reap

import (
	"github.com/stellar/go/services/horizon/internal/ledger"
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
)

func TestDeleteUnretainedHistory(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	ledgerCache := &ledger.Cache{}
	ledgerCache.SetState(tt.Scenario("kahuna"))

	db := tt.HorizonSession()

	sys := New(0, db, ledgerCache)

	var (
		prev int
		cur  int
	)
	err := db.GetRaw(&prev, `SELECT COUNT(*) FROM history_ledgers`)
	tt.Require.NoError(err)

	err = sys.DeleteUnretainedHistory()
	if tt.Assert.NoError(err) {
		err = db.GetRaw(&cur, `SELECT COUNT(*) FROM history_ledgers`)
		tt.Require.NoError(err)
		tt.Assert.Equal(prev, cur, "Ledgers deleted when RetentionCount == 0")
	}

	ledgerCache.SetState(tt.LoadLedgerState())
	sys.RetentionCount = 10
	err = sys.DeleteUnretainedHistory()
	if tt.Assert.NoError(err) {
		err = db.GetRaw(&cur, `SELECT COUNT(*) FROM history_ledgers`)
		tt.Require.NoError(err)
		tt.Assert.Equal(10, cur)
	}

	ledgerCache.SetState(tt.LoadLedgerState())
	sys.RetentionCount = 1
	err = sys.DeleteUnretainedHistory()
	if tt.Assert.NoError(err) {
		err = db.GetRaw(&cur, `SELECT COUNT(*) FROM history_ledgers`)
		tt.Require.NoError(err)
		tt.Assert.Equal(1, cur)
	}
}
