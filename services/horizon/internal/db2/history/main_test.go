package history

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/services/horizon/internal/test"
)

func TestLatestLedger(t *testing.T) {
	tt := test.Start(t)
	tt.Scenario("base")
	defer tt.Finish()
	q := &Q{tt.HorizonSession()}

	var seq int
	err := q.LatestLedger(tt.Ctx, &seq)

	if tt.Assert.NoError(err) {
		tt.Assert.Equal(3, seq)
	}
}

func TestLatestLedgerSequenceClosedAt(t *testing.T) {
	tt := test.Start(t)
	tt.Scenario("base")
	defer tt.Finish()
	q := &Q{tt.HorizonSession()}

	sequence, closedAt, err := q.LatestLedgerSequenceClosedAt(tt.Ctx)
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(int32(3), sequence)
		tt.Assert.Equal("2019-10-31T13:19:46Z", closedAt.Format(time.RFC3339))
	}

	test.ResetHorizonDB(t, tt.HorizonDB)

	sequence, closedAt, err = q.LatestLedgerSequenceClosedAt(tt.Ctx)
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(int32(0), sequence)
		tt.Assert.Equal("0001-01-01T00:00:00Z", closedAt.Format(time.RFC3339))
	}
}

func TestGetLatestHistoryLedgerEmptyDB(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	value, err := q.GetLatestHistoryLedger(tt.Ctx)
	tt.Assert.NoError(err)
	tt.Assert.Equal(uint32(0), value)
}

func TestElderLedger(t *testing.T) {
	tt := test.Start(t)
	tt.Scenario("base")
	defer tt.Finish()
	q := &Q{tt.HorizonSession()}

	var seq int
	err := q.ElderLedger(tt.Ctx, &seq)

	if tt.Assert.NoError(err) {
		tt.Assert.Equal(1, seq)
	}
}

func TestConstructReapLookupTablesQuery(t *testing.T) {
	query := constructReapLookupTablesQuery(
		"history_accounts",
		[]tableObjectFieldPair{
			{
				name:        "history_transaction_participants",
				objectField: "history_account_id",
			},
			{
				name:        "history_effects",
				objectField: "history_account_id",
			},
			{
				name:        "history_operation_participants",
				objectField: "history_account_id",
			},
			{
				name:        "history_trades",
				objectField: "base_account_id",
			},
			{
				name:        "history_trades",
				objectField: "counter_account_id",
			},
		},
		10,
		0,
	)

	assert.Equal(t,
		"DELETE FROM history_accounts WHERE id IN ("+
			"SELECT e1.id FROM ("+
			"SELECT id FROM history_accounts WHERE id >= 0 ORDER BY id LIMIT 10) e1 "+
			"LEFT JOIN LATERAL ( "+
			"SELECT 1 as row FROM history_transaction_participants WHERE history_transaction_participants.history_account_id = e1.id LIMIT 1"+
			") e2 ON true LEFT JOIN LATERAL ( "+
			"SELECT 1 as row FROM history_effects WHERE history_effects.history_account_id = e1.id LIMIT 1"+
			") e3 ON true LEFT JOIN LATERAL ( "+
			"SELECT 1 as row FROM history_operation_participants WHERE history_operation_participants.history_account_id = e1.id LIMIT 1"+
			") e4 ON true LEFT JOIN LATERAL ( "+
			"SELECT 1 as row FROM history_trades WHERE history_trades.base_account_id = e1.id LIMIT 1"+
			") e5 ON true LEFT JOIN LATERAL ( "+
			"SELECT 1 as row FROM history_trades WHERE history_trades.counter_account_id = e1.id LIMIT 1"+
			") e6 ON true "+
			"WHERE e2.row IS NULL AND e3.row IS NULL AND e4.row IS NULL AND e5.row IS NULL AND e6.row IS NULL);", query)
}
