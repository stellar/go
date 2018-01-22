// +build integration

package database

import (
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stellar/go/services/bifrost/queue"
	"github.com/stellar/go/services/bifrost/testdata/fixtures"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsEmptyWithPendingTransactions(t *testing.T) {
	testDB := OpenTestDB(t)
	defer testDB.Close()
	dbQueue := &PostgresDatabase{session: &db.Session{DB: testDB}}
	require.NoError(t, dbQueue.QueueAdd(fixtures.Transaction()))
	// when
	result, err := dbQueue.IsEmpty()
	// then
	require.NoError(t, err)
	assert.False(t, result)
}

func TestWithQueuedTransactionShouldCallCallback(t *testing.T) {
	testDB := OpenTestDB(t)
	defer testDB.Close()
	dbQueue := &PostgresDatabase{session: &db.Session{DB: testDB}}
	require.NoError(t, dbQueue.QueueAdd(fixtures.Transaction()))

	// when
	var callbackExecuted bool
	myHandler := func(transaction queue.Transaction) error {
		callbackExecuted = true
		return nil
	}
	err := dbQueue.WithQueuedTransaction(myHandler)
	// then
	require.NoError(t, err)
	assert.True(t, callbackExecuted)
}

func TestTransactionsAreLockedWhileProcessed(t *testing.T) {
	testDB := OpenTestDB(t)
	defer testDB.Close()
	// prepare DB: update legacy entries to make new one first
	_, _ = testDB.Exec("update " + transactionsQueueTableName + " set failure_count=failure_count+1 where pooled=false;")

	dbQueue := &PostgresDatabase{session: &db.Session{DB: testDB}}
	myTransaction := fixtures.Transaction()
	require.NoError(t, dbQueue.QueueAdd(myTransaction))

	var blockCallback, waitForCallback sync.WaitGroup
	blockCallback.Add(1)
	waitForCallback.Add(1)
	defer blockCallback.Done()

	blockingHandler := func(transaction queue.Transaction) error {
		waitForCallback.Done()
		blockCallback.Wait()
		return nil
	}
	// when
	go dbQueue.WithQueuedTransaction(blockingHandler)
	waitForCallback.Wait()

	// then
	row := loadQueuedTransaction(t, testDB, myTransaction)
	assert.True(t, row.LockedUntil.After(time.Now()))
	assert.NotEmpty(t, row.LockedToken)
}

func TestTransactionsAreUpdatedWhenProcessed(t *testing.T) {
	testDB := OpenTestDB(t)
	defer testDB.Close()
	// prepare DB: update legacy entries to make new one first
	_, _ = testDB.Exec("update " + transactionsQueueTableName + " set failure_count=failure_count+1 where pooled=false;")

	dbQueue := &PostgresDatabase{session: &db.Session{DB: testDB}}
	myTransaction := fixtures.Transaction()
	require.NoError(t, dbQueue.QueueAdd(myTransaction))

	blockingHandler := func(transaction queue.Transaction) error {
		return nil
	}
	// when
	err := dbQueue.WithQueuedTransaction(blockingHandler)
	require.NoError(t, err)

	// then
	row := loadQueuedTransaction(t, testDB, myTransaction)
	assert.True(t, row.Pooled)
	assert.Nil(t, row.LockedUntil)
	assert.Empty(t, row.LockedToken)
}

func TestTransactionsFailureCountUpdated(t *testing.T) {
	testDB := OpenTestDB(t)
	defer testDB.Close()
	// prepare DB: update legacy entries to make new one first
	_, _ = testDB.Exec("update " + transactionsQueueTableName + " set failure_count=failure_count+1 where pooled=false;")

	dbQueue := &PostgresDatabase{session: &db.Session{DB: testDB}}
	myTransaction := fixtures.Transaction()
	require.NoError(t, dbQueue.QueueAdd(myTransaction))

	myErr := errors.New("test, please ignore")
	blockingHandler := func(transaction queue.Transaction) error {
		return myErr
	}
	// when
	err := dbQueue.WithQueuedTransaction(blockingHandler)

	// then
	require.Error(t, err)
	assert.Equal(t, myErr, errors.Cause(err))

	row := loadQueuedTransaction(t, testDB, myTransaction)
	assert.False(t, row.Pooled)
	assert.Equal(t, 1, row.FailureCount)
	assert.Nil(t, row.LockedUntil)
	assert.Empty(t, row.LockedToken)
}

type xTransactionsQueueRow struct {
	transactionsQueueRow
	ID          int64      `db:"id"`
	LockedUntil *time.Time `db:"locked_until"`
	LockedToken *string    `db:"locked_token"`
	Pooled      bool       `db:"pooled"`
}

func loadQueuedTransaction(t *testing.T, testDB *sqlx.DB, source queue.Transaction) xTransactionsQueueRow {
	t.Helper()
	var row xTransactionsQueueRow
	err := testDB.Get(&row, "select * from "+transactionsQueueTableName+
		" where transaction_id = $1 and asset_code = $2 limit 1", source.TransactionID, source.AssetCode)
	require.NoError(t, err)
	return row
}
