package history

import (
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/support/db"
)

func TestOperationQueries(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()
	q := &Q{tt.HorizonSession()}

	// Test OperationByID
	op, transaction, err := q.OperationByID(false, 8589938689)
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(int64(8589938689), op.ID)
	}
	tt.Assert.Nil(transaction)

	// Test Operations()
	ops, transactions, err := q.Operations().
		ForAccount("GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON").
		Fetch()
	if tt.Assert.NoError(err) {
		tt.Assert.Len(ops, 2)
	}
	tt.Assert.Len(transactions, 0)

	// ledger filter works
	ops, transactions, err = q.Operations().ForLedger(2).Fetch()
	if tt.Assert.NoError(err) {
		tt.Assert.Len(ops, 3)
	}
	tt.Assert.Len(transactions, 0)

	// tx filter works
	hash := "2374e99349b9ef7dba9a5db3339b78fda8f34777b1af33ba468ad5c0df946d4d"
	ops, transactions, err = q.Operations().ForTransaction(hash).Fetch()
	if tt.Assert.NoError(err) {
		tt.Assert.Len(ops, 1)
	}
	tt.Assert.Len(transactions, 0)

	// payment filter works
	tt.Scenario("pathed_payment")
	ops, transactions, err = q.Operations().OnlyPayments().Fetch()
	if tt.Assert.NoError(err) {
		tt.Assert.Len(ops, 10)
	}
	tt.Assert.Len(transactions, 0)

	// payment filter includes account merges
	tt.Scenario("account_merge")
	ops, transactions, err = q.Operations().OnlyPayments().Fetch()
	if tt.Assert.NoError(err) {
		tt.Assert.Len(ops, 3)
	}
	tt.Assert.Len(transactions, 0)
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

	q := &Q{tt.HorizonSession()}
	query := q.Operations().
		ForAccount("GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2")

	operations, transactions, err := query.Fetch()
	tt.Assert.NoError(err)
	tt.Assert.Len(transactions, 0)

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

	q := &Q{tt.HorizonSession()}
	query := q.Operations().
		ForAccount("GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2").
		IncludeFailed()

	operations, transactions, err := query.Fetch()
	tt.Assert.NoError(err)
	tt.Assert.Len(transactions, 0)

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

	q := &Q{tt.HorizonSession()}
	query := q.Operations().
		OnlyPayments().
		ForAccount("GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON")

	operations, transactions, err := query.Fetch()
	tt.Assert.NoError(err)
	tt.Assert.Len(transactions, 0)

	tt.Assert.Equal(2, len(operations))

	for _, operation := range operations {
		tt.Assert.True(*operation.TransactionSuccessful)
	}

	sql, _, err := query.sql.ToSql()
	tt.Assert.NoError(err)
	// Note: brackets around `(ht.successful = true OR ht.successful IS NULL)` are critical!
	tt.Assert.Contains(sql, "WHERE hop.type IN (?,?,?,?,?) AND hopp.history_account_id = ? AND (ht.successful = true OR ht.successful IS NULL)")
}

// TestPaymentsIncludeFailed tests `IncludeFailed` method.
func TestPaymentsIncludeFailed(t *testing.T) {
	tt := test.Start(t).Scenario("failed_transactions")
	defer tt.Finish()

	q := &Q{tt.HorizonSession()}
	query := q.Operations().
		OnlyPayments().
		ForAccount("GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON").
		IncludeFailed()

	operations, transactions, err := query.Fetch()
	tt.Assert.NoError(err)
	tt.Assert.Len(transactions, 0)

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
	tt.Assert.Equal("SELECT hop.id, hop.transaction_id, hop.application_order, hop.type, hop.details, hop.source_account, ht.transaction_hash, ht.tx_result, ht.successful as transaction_successful FROM history_operations hop LEFT JOIN history_transactions ht ON ht.id = hop.transaction_id JOIN history_operation_participants hopp ON hopp.history_operation_id = hop.id WHERE hop.type IN (?,?,?,?,?) AND hopp.history_account_id = ?", sql)
}

func TestExtraChecksOperationsTransactionSuccessfulTrueResultFalse(t *testing.T) {
	tt := test.Start(t).Scenario("failed_transactions")
	defer tt.Finish()

	// successful `true` but tx result `false`
	_, err := tt.HorizonDB.Exec(
		`UPDATE history_transactions SET successful = true WHERE transaction_hash = 'aa168f12124b7c196c0adaee7c73a64d37f99428cacb59a91ff389626845e7cf'`,
	)
	tt.Require.NoError(err)

	q := &Q{tt.HorizonSession()}
	query := q.Operations().
		ForAccount("GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2").
		IncludeFailed()

	_, _, err = query.Fetch()
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

	q := &Q{tt.HorizonSession()}
	query := q.Operations().
		ForAccount("GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON").
		IncludeFailed()

	_, _, err = query.Fetch()
	tt.Assert.Error(err)
	tt.Assert.Contains(err.Error(), "Corrupted data! `successful=false` but returned transaction is success")
}

func assertOperationMatchesTransaction(tt *test.T, operation Operation, transaction Transaction) {
	tt.Assert.Equal(operation.TransactionID, transaction.ID)
	tt.Assert.Equal(operation.TransactionHash, transaction.TransactionHash)
	tt.Assert.Equal(operation.TxResult, transaction.TxResult)
	tt.Assert.Equal(operation.IsTransactionSuccessful(), transaction.IsSuccessful())
}

// TestOperationIncludeTransactions tests that transactions are included when fetching records from the db.
func TestOperationIncludeTransactions(t *testing.T) {
	tt := test.Start(t).Scenario("failed_transactions")
	defer tt.Finish()

	accountID := "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2"

	q := &Q{tt.HorizonSession()}
	query := q.Operations().
		IncludeTransactions().
		ForAccount(accountID)

	operations, transactions, err := query.Fetch()
	tt.Assert.NoError(err)
	tt.Assert.Len(transactions, 3)
	tt.Assert.Len(transactions, len(operations))

	for i := range operations {
		operation := operations[i]
		transaction := transactions[i]
		assertOperationMatchesTransaction(tt, operation, transaction)
	}

	withoutTransactionsQuery := (&Q{tt.HorizonSession()}).Operations().
		ForAccount(accountID)

	var expectedTransactions []Transaction
	err = (&Q{tt.HorizonSession()}).Transactions().ForAccount(accountID).Select(&expectedTransactions)
	tt.Assert.NoError(err)

	expectedOperations, _, err := withoutTransactionsQuery.Fetch()
	tt.Assert.NoError(err)

	tt.Assert.Equal(operations, expectedOperations)
	tt.Assert.Equal(transactions, expectedTransactions)

	op, transaction, err := q.OperationByID(true, expectedOperations[0].ID)
	tt.Assert.NoError(err)
	tt.Assert.Equal(op, expectedOperations[0])
	tt.Assert.Equal(*transaction, expectedTransactions[0])
	assertOperationMatchesTransaction(tt, op, *transaction)
}

func TestCheckExpOperations(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	sequence := int32(56)

	// first transaction
	transactionResult := "AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA="
	transactionMeta := "AAAAAQAAAAIAAAADAAAAOAAAAAAAAAAAGlyOIRNnS6HDEkmGAE9gpZJUnLZNnYsjzt+pXVxR/9QAAAACVAvjnAAAADcAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAOAAAAAAAAAAAGlyOIRNnS6HDEkmGAE9gpZJUnLZNnYsjzt+pXVxR/9QAAAACVAvjnAAAADcAAAABAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAAA=="
	transactionHash := "2a805712c6d10f9e74bb0ccf54ae92a2b4b1e586451fe8133a2433816f6b567c"

	transaction, err := buildTransaction(
		1,
		"AAAAABpcjiETZ0uhwxJJhgBPYKWSVJy2TZ2LI87fqV1cUf/UAAAAZAAAADcAAAABAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAGlyOIRNnS6HDEkmGAE9gpZJUnLZNnYsjzt+pXVxR/9QAAAAAAAAAAAX14QAAAAAAAAAAAVxR/9QAAABAK6pcXYMzAEmH08CZ1LWmvtNDKauhx+OImtP/Lk4hVTMJRVBOebVs5WEPj9iSrgGT0EswuDCZ2i5AEzwgGof9Ag==",
		&transactionResult,
		&transactionMeta,
		&transactionHash,
	)
	tt.Assert.NoError(err)

	transactionMeta = "AAAAAQAAAAIAAAADAAAAOQAAAAAAAAAAYvwdC9CRsrYcDdZWNGsqaNfTR8bywsjubQRHAlb8BfcLGrZY9dZxbAAAAAAAAAAZAAAAAAAAAAEAAAAAYvwdC9CRsrYcDdZWNGsqaNfTR8bywsjubQRHAlb8BfcAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAOQAAAAAAAAAAYvwdC9CRsrYcDdZWNGsqaNfTR8bywsjubQRHAlb8BfcLGrZY9dZxbAAAAAAAAAAaAAAAAAAAAAEAAAAAYvwdC9CRsrYcDdZWNGsqaNfTR8bywsjubQRHAlb8BfcAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAAwAAAAMAAAA5AAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9wsatlj11nFsAAAAAAAAABoAAAAAAAAAAQAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9wAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAA5AAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9wsatlahyo1sAAAAAAAAABoAAAAAAAAAAQAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9wAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAA5AAAAAAAAAAChl+0U/eniJ9tObqFB2J6i27dnD0TTka2hKHkwHK0SlAAAAAJUC+QAAAAAOQAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA=="
	transactionHash = "0e5bd332291e3098e49886df2cdb9b5369a5f9e0a9973f0d9e1a9489c6581ba2"
	// second transaction
	secondTransaction, err := buildTransaction(
		2,
		"AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAAaAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAoZftFP3p4ifbTm6hQdieotu3Zw9E05GtoSh5MBytEpQAAAACVAvkAAAAAAAAAAABVvwF9wAAAEDHU95E9wxgETD8TqxUrkgC0/7XHyNDts6Q5huRHfDRyRcoHdv7aMp/sPvC3RPkXjOMjgbKJUX7SgExUeYB5f8F",
		&transactionResult,
		&transactionMeta,
		&transactionHash,
	)
	tt.Assert.NoError(err)

	// third transaction
	transactionMeta = "AAAAAQAAAAIAAAADAAAANQAAAAAAAAAAYvwdC9CRsrYcDdZWNGsqaNfTR8bywsjubQRHAlb8BfcLGrZdne46/AAAAAAAAAAWAAAAAAAAAAEAAAAAYvwdC9CRsrYcDdZWNGsqaNfTR8bywsjubQRHAlb8BfcAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAANQAAAAAAAAAAYvwdC9CRsrYcDdZWNGsqaNfTR8bywsjubQRHAlb8BfcLGrZdne46/AAAAAAAAAAXAAAAAAAAAAEAAAAAYvwdC9CRsrYcDdZWNGsqaNfTR8bywsjubQRHAlb8BfcAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAAwAAAAMAAAA1AAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9wsatl2d7jr8AAAAAAAAABcAAAAAAAAAAQAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9wAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAA1AAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9wsatltJ4lb8AAAAAAAAABcAAAAAAAAAAQAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9wAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAA1AAAAAAAAAAAEl5OYpHWqv4JmgjKMB8bGshdn+0jVUh3Y59mRGjAPPgAAAAJUC+QAAAAANQAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA=="
	transactionHash = "df5f0e8b3b533dd9cda0ff7540bef3e9e19369060f8a4b0414b0e3c1b4315b1c"
	thirdTransaction, err := buildTransaction(
		3,
		"AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAAXAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAABJeTmKR1qr+CZoIyjAfGxrIXZ/tI1VId2OfZkRowDz4AAAACVAvkAAAAAAAAAAABVvwF9wAAAEDyHwhW9GXQVXG1qibbeqSjxYzhv5IC08K2vSkxzYTwJykvQ8l0+e4M4h2guoK89s8HUfIqIOzDmoGsNTaLcYUG",
		&transactionResult,
		&transactionMeta,
		&transactionHash,
	)

	tt.Assert.NoError(err)

	insertTransaction(tt, q, "exp_history_transactions", transaction, sequence)
	insertTransaction(tt, q, "exp_history_transactions", secondTransaction, sequence)
	insertTransaction(tt, q, "exp_history_transactions", thirdTransaction, sequence)
	insertTransaction(tt, q, "history_transactions", transaction, sequence)
	insertTransaction(tt, q, "history_transactions", secondTransaction, sequence)
	insertTransaction(tt, q, "history_transactions", thirdTransaction, sequence)

	operationBatch := q.NewOperationBatchInsertBuilder(100)

	txs := []io.LedgerTransaction{
		transaction,
		secondTransaction,
		thirdTransaction,
	}

	for _, t := range txs {
		err = operationBatch.Add(t, uint32(sequence))
		tt.Assert.NoError(err)
	}

	err = operationBatch.Exec()
	tt.Assert.NoError(err)

	batchBuilder := operationBatchInsertBuilder{
		builder: db.BatchInsertBuilder{
			Table:        q.GetTable("history_operations"),
			MaxBatchSize: 100,
		},
	}

	for _, t := range txs {
		err = batchBuilder.Add(t, uint32(sequence))
		tt.Assert.NoError(err)
	}

	err = batchBuilder.Exec()
	tt.Assert.NoError(err)

	valid, err := q.CheckExpOperations(sequence)
	tt.Assert.True(valid)
	tt.Assert.NoError(err)

	operation := transactionOperationWrapper{
		index:          0,
		transaction:    thirdTransaction,
		operation:      thirdTransaction.Envelope.Tx.Operations[0],
		ledgerSequence: uint32(sequence),
	}

	for fieldName, value := range map[string]interface{}{
		"application_order": 100,
		"type":              13,
		"details":           "{}",
		"source_account":    "source_account",
	} {
		updateSQL := sq.Update("history_operations").
			Set(fieldName, value).
			Where(
				"id = ?",
				operation.ID(),
			)
		_, err = q.Exec(updateSQL)
		tt.Assert.NoError(err)

		valid, err = q.CheckExpOperations(sequence)
		tt.Assert.NoError(err)
		tt.Assert.False(valid)

		_, err = q.Exec(sq.Delete("history_operations").
			Where("id = ?", operation.ID()))
		tt.Assert.NoError(err)

		err = batchBuilder.Add(operation.transaction, operation.ledgerSequence)
		tt.Assert.NoError(err)
		err = batchBuilder.Exec()
		tt.Assert.NoError(err)

		valid, err := q.CheckExpOperations(sequence)
		tt.Assert.NoError(err)
		tt.Assert.True(valid)
	}
}
