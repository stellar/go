package history

import (
	"context"
	"database/sql"
	"encoding/hex"
	"testing"
	"time"

	"github.com/guregu/null"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

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
			InnerSignatures:  nil,
			MemoType:         "none",
			Memo:             null.NewString("", false),
			Successful:       true,
			TimeBounds:       TimeBounds{Null: true},
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

	tt.Assert.NoError(q.InitEmptyTxSubmissionResult(ctx, hash))

	_, err = q.GetTxSubmissionResult(ctx, hash)
	tt.Assert.Error(err)
	tt.Assert.Equal(err, sql.ErrNoRows)

	transactions, err = q.GetTxSubmissionResults(ctx, []string{hash})
	tt.Assert.NoError(err)
	tt.Assert.Len(transactions, 0)

	// Trying to set the result of a transaction which wasn't initialized
	// doesn't fail
	// TODO: should it?
	toInsertFail := toInsert
	toInsertFail.Result.TransactionHash = xdr.Hash{0x1, 0x2, 0x3, 0x4}
	err = q.SetTxSubmissionResult(ctx, toInsertFail, sequence, ledgerCloseTime)
	tt.Assert.NoError(err)

	// Now insert the valid transaction
	tt.Assert.NoError(q.SetTxSubmissionResult(ctx, toInsert, sequence, ledgerCloseTime))
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
	rowsAffected, err := q.DeleteTxSubmissionResultsOlderThan(ctx, 1)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(1), rowsAffected)

	_, err = q.GetTxSubmissionResult(ctx, hash)
	tt.Assert.Error(err)
	tt.Assert.Equal(err, sql.ErrNoRows)

	transactions, err = q.GetTxSubmissionResults(ctx, []string{hash})
	tt.Assert.NoError(err)
	tt.Assert.Len(transactions, 0)
}
