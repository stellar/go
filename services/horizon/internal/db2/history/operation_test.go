package history

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/test"
)

func TestOperationQueries(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()
	q := &Q{tt.HorizonSession()}

	// Test OperationByID
	var op Operation
	err := q.OperationByID(&op, 8589938689)
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(int64(8589938689), op.ID)
	}

	// Test Operations()
	ops := []Operation{}
	err = q.Operations().
		ForAccount("GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON").
		Select(&ops)
	if tt.Assert.NoError(err) {
		tt.Assert.Len(ops, 2)
	}

	// ledger filter works
	ops = []Operation{}
	err = q.Operations().ForLedger(2).Select(&ops)
	if tt.Assert.NoError(err) {
		tt.Assert.Len(ops, 3)
	}

	// tx filter works
	hash := "2374e99349b9ef7dba9a5db3339b78fda8f34777b1af33ba468ad5c0df946d4d"
	ops = []Operation{}
	err = q.Operations().ForTransaction(hash).Select(&ops)
	if tt.Assert.NoError(err) {
		tt.Assert.Len(ops, 1)
	}

	// payment filter works
	tt.Scenario("pathed_payment")
	ops = []Operation{}
	err = q.Operations().OnlyPayments().Select(&ops)
	if tt.Assert.NoError(err) {
		tt.Assert.Len(ops, 10)
	}

	// payment filter includes account merges
	tt.Scenario("account_merge")
	ops = []Operation{}
	err = q.Operations().OnlyPayments().Select(&ops)
	if tt.Assert.NoError(err) {
		tt.Assert.Len(ops, 3)
	}
}

func TestOperationQueryBuilder(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()
	q := &Q{tt.HorizonSession()}

	opsQ := q.Operations().ForAccount("GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON").Page(db2.PageQuery{Cursor: "8589938689", Order: "asc", Limit: 10})
	tt.Assert.NoError(opsQ.Err)
	got, _, err := opsQ.sql.ToSql()
	tt.Assert.NoError(err)

	// Operations for account queries will use hopp.history_operation_id in their predicates.
	want := "SELECT hop.id, hop.transaction_id, hop.application_order, hop.type, hop.details, hop.source_account, ht.transaction_hash, ht.successful as transaction_successful FROM history_operations hop LEFT JOIN history_transactions ht ON ht.id = hop.transaction_id JOIN history_operation_participants hopp ON hopp.history_operation_id = hop.id WHERE hopp.history_account_id = ? AND hopp.history_operation_id > ? ORDER BY hopp.history_operation_id asc LIMIT 10"
	tt.Assert.EqualValues(want, got)

	opsQ = q.Operations().ForLedger(2).Page(db2.PageQuery{Cursor: "8589938689", Order: "asc", Limit: 10})
	tt.Assert.NoError(opsQ.Err)
	got, _, err = opsQ.sql.ToSql()
	tt.Assert.NoError(err)

	// Other operation queries will use hop.id in their predicates.
	want = "SELECT hop.id, hop.transaction_id, hop.application_order, hop.type, hop.details, hop.source_account, ht.transaction_hash, ht.successful as transaction_successful FROM history_operations hop LEFT JOIN history_transactions ht ON ht.id = hop.transaction_id WHERE hop.id >= ? AND hop.id < ? AND hop.id > ? ORDER BY hop.id asc LIMIT 10"
	tt.Assert.EqualValues(want, got)
}

// TestOperationSuccessfulOnly tests SuccessfulOnly() method. It specifically
// tests the `successful = true OR successful IS NULL` condition in a query.
// If it's not enclosed in brackets, it may return incorrect result when mixed
// with `ForAccount` or `ForLedger` filters.
func TestOperationSuccessfulOnly(t *testing.T) {
	tt := test.Start(t).Scenario("failed_transactions")
	defer tt.Finish()

	q := &Q{tt.HorizonSession()}
	query := q.Operations().
		ForAccount("GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU").
		SuccessfulOnly()

	sql, _, err := query.sql.ToSql()
	tt.Assert.NoError(err)
	// Note: brackets around `(ht.successful = true OR ht.successful IS NULL)` are critical!
	tt.Assert.Contains(sql, "WHERE hopp.history_account_id = ? AND (ht.successful = true OR ht.successful IS NULL)")
}
