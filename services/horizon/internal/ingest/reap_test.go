package ingest

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
)

func TestDeleteUnretainedHistory(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	tt.Scenario("kahuna")

	db := tt.HorizonSession()

	reaper := NewReaper(ReapConfig{
		RetentionCount: 0,
		ReapBatchSize:  50,
	}, db)

	// Disable sleeps for this.
	sleep = 0

	var (
		prev int
		cur  int
	)
	err := db.GetRaw(tt.Ctx, &prev, `SELECT COUNT(*) FROM history_ledgers`)
	tt.Require.NoError(err)

	err = reaper.DeleteUnretainedHistory(tt.Ctx)
	if tt.Assert.NoError(err) {
		err = db.GetRaw(tt.Ctx, &cur, `SELECT COUNT(*) FROM history_ledgers`)
		tt.Require.NoError(err)
		tt.Assert.Equal(prev, cur, "Ledgers deleted when RetentionCount == 0")
	}

	reaper.config.RetentionCount = 10
	err = reaper.DeleteUnretainedHistory(tt.Ctx)
	if tt.Assert.NoError(err) {
		err = db.GetRaw(tt.Ctx, &cur, `SELECT COUNT(*) FROM history_ledgers`)
		tt.Require.NoError(err)
		tt.Assert.Equal(10, cur)
	}

	reaper.config.RetentionCount = 1
	err = reaper.DeleteUnretainedHistory(tt.Ctx)
	if tt.Assert.NoError(err) {
		err = db.GetRaw(tt.Ctx, &cur, `SELECT COUNT(*) FROM history_ledgers`)
		tt.Require.NoError(err)
		tt.Assert.Equal(1, cur)
	}
}
