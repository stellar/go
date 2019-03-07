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
	want := "SELECT hop.id, hop.transaction_id, hop.application_order, hop.type, hop.details, hop.source_account, ht.transaction_hash, ht.tx_result, ht.successful as transaction_successful FROM history_operations hop LEFT JOIN history_transactions ht ON ht.id = hop.transaction_id JOIN history_operation_participants hopp ON hopp.history_operation_id = hop.id WHERE hopp.history_account_id = ? AND hopp.history_operation_id > ? ORDER BY hopp.history_operation_id asc LIMIT 10"
	tt.Assert.EqualValues(want, got)

	opsQ = q.Operations().ForLedger(2).Page(db2.PageQuery{Cursor: "8589938689", Order: "asc", Limit: 10})
	tt.Assert.NoError(opsQ.Err)
	got, _, err = opsQ.sql.ToSql()
	tt.Assert.NoError(err)

	// Other operation queries will use hop.id in their predicates.
	want = "SELECT hop.id, hop.transaction_id, hop.application_order, hop.type, hop.details, hop.source_account, ht.transaction_hash, ht.tx_result, ht.successful as transaction_successful FROM history_operations hop LEFT JOIN history_transactions ht ON ht.id = hop.transaction_id WHERE hop.id >= ? AND hop.id < ? AND hop.id > ? ORDER BY hop.id asc LIMIT 10"
	tt.Assert.EqualValues(want, got)
}

// TestOperationSuccessfulOnly tests if default query returns operations in
// successful transactions only.
// If it's not enclosed in brackets, it may return incorrect result when mixed
// with `ForAccount` or `ForLedger` filters.
func TestOperationSuccessfulOnly(t *testing.T) {
	tt := test.Start(t).Scenario("failed_transactions")
	defer tt.Finish()

	var operations []Operation

	q := &Q{tt.HorizonSession()}
	query := q.Operations().
		ForAccount("GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2")

	err := query.Select(&operations)
	tt.Assert.NoError(err)

	tt.Assert.Equal(3, len(operations))

	for _, operation := range operations {
		tt.Assert.True(*operation.TransactionSuccessful)
	}

	sql, _, err := query.sql.ToSql()
	tt.Assert.NoError(err)
	// Note: brackets around `(ht.successful = true OR ht.successful IS NULL)` are critical!
	tt.Assert.Contains(sql, "WHERE hopp.history_account_id = ? AND (ht.successful = true OR ht.successful IS NULL)")
}

// TestOperationIncludeFailed tests `IncludeFailed` method.
func TestOperationIncludeFailed(t *testing.T) {
	tt := test.Start(t).Scenario("failed_transactions")
	defer tt.Finish()

	var operations []Operation

	q := &Q{tt.HorizonSession()}
	query := q.Operations().
		ForAccount("GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2").
		IncludeFailed()

	err := query.Select(&operations)
	tt.Assert.NoError(err)

	var failed, successful int
	for _, operation := range operations {
		if *operation.TransactionSuccessful {
			successful++
		} else {
			failed++
		}
	}

	tt.Assert.Equal(3, successful)
	tt.Assert.Equal(1, failed)

	sql, _, err := query.sql.ToSql()
	tt.Assert.NoError(err)
	tt.Assert.Equal("SELECT hop.id, hop.transaction_id, hop.application_order, hop.type, hop.details, hop.source_account, ht.transaction_hash, ht.tx_result, ht.successful as transaction_successful FROM history_operations hop LEFT JOIN history_transactions ht ON ht.id = hop.transaction_id JOIN history_operation_participants hopp ON hopp.history_operation_id = hop.id WHERE hopp.history_account_id = ?", sql)
}

// TestPaymentsSuccessfulOnly tests if default query returns payments in
// successful transactions only.
// If it's not enclosed in brackets, it may return incorrect result when mixed
// with `ForAccount` or `ForLedger` filters.
func TestPaymentsSuccessfulOnly(t *testing.T) {
	tt := test.Start(t).Scenario("failed_transactions")
	defer tt.Finish()

	var operations []Operation

	q := &Q{tt.HorizonSession()}
	query := q.Operations().
		OnlyPayments().
		ForAccount("GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON")

	err := query.Select(&operations)
	tt.Assert.NoError(err)

	tt.Assert.Equal(2, len(operations))

	for _, operation := range operations {
		tt.Assert.True(*operation.TransactionSuccessful)
	}

	sql, _, err := query.sql.ToSql()
	tt.Assert.NoError(err)
	// Note: brackets around `(ht.successful = true OR ht.successful IS NULL)` are critical!
	tt.Assert.Contains(sql, "WHERE hop.type IN (?,?,?,?) AND hopp.history_account_id = ? AND (ht.successful = true OR ht.successful IS NULL)")
}

// TestPaymentsIncludeFailed tests `IncludeFailed` method.
func TestPaymentsIncludeFailed(t *testing.T) {
	tt := test.Start(t).Scenario("failed_transactions")
	defer tt.Finish()

	var operations []Operation

	q := &Q{tt.HorizonSession()}
	query := q.Operations().
		OnlyPayments().
		ForAccount("GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON").
		IncludeFailed()

	err := query.Select(&operations)
	tt.Assert.NoError(err)

	var failed, successful int
	for _, operation := range operations {
		if *operation.TransactionSuccessful {
			successful++
		} else {
			failed++
		}
	}

	tt.Assert.Equal(2, successful)
	tt.Assert.Equal(1, failed)

	sql, _, err := query.sql.ToSql()
	tt.Assert.NoError(err)
	tt.Assert.Equal("SELECT hop.id, hop.transaction_id, hop.application_order, hop.type, hop.details, hop.source_account, ht.transaction_hash, ht.tx_result, ht.successful as transaction_successful FROM history_operations hop LEFT JOIN history_transactions ht ON ht.id = hop.transaction_id JOIN history_operation_participants hopp ON hopp.history_operation_id = hop.id WHERE hop.type IN (?,?,?,?) AND hopp.history_account_id = ?", sql)
}

func TestExtraChecksOperationsTransactionSuccessfulTrueResultFalse(t *testing.T) {
	tt := test.Start(t).Scenario("failed_transactions")
	defer tt.Finish()

	// successful `true` but tx result `false`
	_, err := tt.HorizonDB.Exec(
		`UPDATE history_transactions SET successful = true WHERE transaction_hash = 'aa168f12124b7c196c0adaee7c73a64d37f99428cacb59a91ff389626845e7cf'`,
	)
	tt.Require.NoError(err)

	var operations []Operation

	q := &Q{tt.HorizonSession()}
	query := q.Operations().
		ForAccount("GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2").
		IncludeFailed()

	err = query.Select(&operations)
	tt.Assert.Error(err)
	tt.Assert.Contains(err.Error(), "Corrupted data! `successful=true` but returned transaction is not success")
}

func TestExtraChecksOperationsTransactionSuccessfulFalseResultTrue(t *testing.T) {
	tt := test.Start(t).Scenario("failed_transactions")
	defer tt.Finish()

	// successful `false` but tx result `true`
	_, err := tt.HorizonDB.Exec(
		`UPDATE history_transactions SET successful = false WHERE transaction_hash = 'a2dabf4e9d1642722602272e178a37c973c9177b957da86192a99b3e9f3a9aa4'`,
	)
	tt.Require.NoError(err)

	var operations []Operation

	q := &Q{tt.HorizonSession()}
	query := q.Operations().
		ForAccount("GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON").
		IncludeFailed()

	err = query.Select(&operations)
	tt.Assert.Error(err)
	tt.Assert.Contains(err.Error(), "Corrupted data! `successful=false` but returned transaction is success")
}
