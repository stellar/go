package history

import (
	"context"
	"database/sql"
	"encoding/hex"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/xdr"
)

func TestInitIdempotent(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}
	hash := xdr.Hash{0x1, 0x2, 0x3, 0x4}

	// first invocation, creates row
	ctx := context.Background()
	tt.Assert.NoError(q.InitEmptyTxSubmissionResult(ctx, hash.HexString(), ""))

	// nth invocations on same hash, should be idempotent, if already a row, no-op, no error
	tt.Assert.NoError(q.InitEmptyTxSubmissionResult(ctx, hash.HexString(), ""))
}

func TestInvalidSerializedTX(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}
	hash := xdr.Hash{0x1, 0x2, 0x3, 0x4}

	// first invocation, creates row
	ctx := context.Background()
	tt.Assert.NoError(q.InitEmptyTxSubmissionResult(ctx, hash.HexString(), ""))

	// put invalid encoded bytes for tx result
	sql := sq.Update(txSubResultTableName).
		Set(txSubResultColumnName, "garbage").
		Where(sq.Eq{txSubResultHashColumnName: hash.HexString()})
	result, err := q.Exec(ctx, sql)
	rows, _ := result.RowsAffected()
	tt.Assert.Equal(rows, int64(1))
	tt.Assert.NoError(err)

	// should get err when retrieving due to invalid bytes
	_, err = q.GetTxSubmissionResults(ctx, []string{hash.HexString()})
	tt.Assert.Error(err)
}

func TestTxSubResult(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	sequence := uint32(123)
	toInsert := buildLedgerTransaction(tt.T, testTransaction{
		index:         1,
		envelopeXDR:   "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAyAEXUhsAADDRAAAAAAAAAAAAAAABAAAAAAAAAAsBF1IbAABX4QAAAAAAAAAA",
		resultXDR:     "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
		feeChangesXDR: "AAAAAA==",
		metaXDR:       "AAAAAQAAAAAAAAAA",
		hash:          "19aaa18db88605aedec04659fb45e06f240b022eb2d429e05133e4d53cd945ba",
	})
	ledgerCloseTime := time.Now().UTC().Truncate(time.Second)
	expected := Transaction{
		LedgerCloseTime: ledgerCloseTime,
		TransactionWithoutLedger: TransactionWithoutLedger{
			TotalOrderID:     TotalOrderID{528280981504},
			TransactionHash:  "19aaa18db88605aedec04659fb45e06f240b022eb2d429e05133e4d53cd945ba",
			LedgerSequence:   int32(sequence),
			ApplicationOrder: 1,
			Account:          "GAUJETIZVEP2NRYLUESJ3LS66NVCEGMON4UDCBCSBEVPIID773P2W6AY",
			AccountSequence:  "78621794419880145",
			MaxFee:           200,
			FeeCharged:       300,
			OperationCount:   1,
			TxEnvelope:       "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAyAEXUhsAADDRAAAAAAAAAAAAAAABAAAAAAAAAAsBF1IbAABX4QAAAAAAAAAA",
			TxResult:         "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
			TxFeeMeta:        "AAAAAA==",
			TxMeta:           "AAAAAQAAAAAAAAAA",
			Signatures:       []string{},
			ExtraSigners:     nil,
			InnerSignatures:  nil,
			MemoType:         "none",
			Memo:             null.NewString("", false),
			Successful:       true,
			TimeBounds:       TimeBounds{Null: true},
			LedgerBounds:     LedgerBounds{Null: true},
		},
	}

	hash := hex.EncodeToString(toInsert.Result.TransactionHash[:])
	ctx := context.Background()

	_, err := q.GetTxSubmissionResult(ctx, hash)
	tt.Assert.Error(err)
	tt.Assert.Equal(err, sql.ErrNoRows)
	transactions, err := q.GetTxSubmissionResults(ctx, []string{hash})
	tt.Assert.NoError(err)
	tt.Assert.Len(transactions, 0)

	tt.Assert.NoError(q.InitEmptyTxSubmissionResult(ctx, hash, ""))

	_, err = q.GetTxSubmissionResult(ctx, hash)
	tt.Assert.Error(err)
	tt.Assert.Equal(err, sql.ErrNoRows)

	transactions, err = q.GetTxSubmissionResults(ctx, []string{hash})
	tt.Assert.NoError(err)
	tt.Assert.Len(transactions, 0)

	// Trying to set the result of a transaction which wasn't initialized
	// doesn't fail
	toInsertFail := toInsert
	toInsertFail.Result.TransactionHash = xdr.Hash{0x1, 0x2, 0x3, 0x4}
	affectedRows, err := q.SetTxSubmissionResults(ctx, []ingest.LedgerTransaction{toInsertFail}, sequence, ledgerCloseTime)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(0), affectedRows)

	// Now insert the valid transaction
	affectedRows, err = q.SetTxSubmissionResults(ctx, []ingest.LedgerTransaction{toInsert}, sequence, ledgerCloseTime)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(1), affectedRows)
	transaction, err := q.GetTxSubmissionResult(ctx, hash)
	tt.Assert.NoError(err)

	// ignore created time and updated time
	transaction.CreatedAt = expected.CreatedAt
	transaction.UpdatedAt = expected.UpdatedAt

	// compare ClosedAt separately because reflect.DeepEqual does not handle time.Time
	closedAt := transaction.LedgerCloseTime
	transaction.LedgerCloseTime = expected.LedgerCloseTime

	tt.Assert.True(closedAt.Equal(expected.LedgerCloseTime))
	tt.Assert.Equal(transaction, expected)

	transactions, err = q.GetTxSubmissionResults(ctx, []string{hash})
	tt.Assert.NoError(err)
	tt.Assert.Len(transactions, 1)

	time.Sleep(2 * time.Second)
	affectedRows, err = q.DeleteTxSubmissionResultsOlderThan(ctx, 1)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(1), affectedRows)

	_, err = q.GetTxSubmissionResult(ctx, hash)
	tt.Assert.Error(err)
	tt.Assert.Equal(err, sql.ErrNoRows)

	transactions, err = q.GetTxSubmissionResults(ctx, []string{hash})
	tt.Assert.NoError(err)
	tt.Assert.Len(transactions, 0)

	// test querying by inner hash
	innerHash := "lambada"
	tt.Assert.NoError(q.InitEmptyTxSubmissionResult(ctx, hash, innerHash))
	_, err = q.SetTxSubmissionResults(ctx, []ingest.LedgerTransaction{toInsert}, sequence, ledgerCloseTime)
	tt.Assert.NoError(err)
	_, err = q.GetTxSubmissionResult(ctx, innerHash)
	tt.Assert.NoError(err)

}

func TestSetTxSubResultBatching(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	transactionLen := db.PostgresQueryMaxParams + 3
	transactions := make([]ingest.LedgerTransaction, transactionLen, transactionLen)
	for i := range transactions {
		transactions[i] = buildLedgerTransaction(tt.T, testTransaction{
			index:         1,
			envelopeXDR:   "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAyAEXUhsAADDRAAAAAAAAAAAAAAABAAAAAAAAAAsBF1IbAABX4QAAAAAAAAAA",
			resultXDR:     "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
			feeChangesXDR: "AAAAAA==",
			metaXDR:       "AAAAAQAAAAAAAAAA",
			hash:          "19aaa18db88605aedec04659fb45e06f240b022eb2d429e05133e4d53cd945ba",
		})
	}

	ctx := context.Background()
	updatedRows, err := q.SetTxSubmissionResults(ctx, transactions, 0, time.Now())
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(0), updatedRows)
}
