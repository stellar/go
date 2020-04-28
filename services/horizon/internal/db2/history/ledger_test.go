package history

import (
	"database/sql"
	"encoding/hex"
	"testing"
	"time"

	"github.com/guregu/null"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/xdr"
)

func TestLedgerQueries(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()
	q := &Q{tt.HorizonSession()}

	// Test LedgerBySequence
	var l Ledger
	err := q.LedgerBySequence(&l, 3)
	tt.Assert.NoError(err)

	err = q.LedgerBySequence(&l, 100000)
	tt.Assert.Equal(err, sql.ErrNoRows)

	// Test Ledgers()
	ls := []Ledger{}
	err = q.Ledgers().Select(&ls)

	if tt.Assert.NoError(err) {
		tt.Assert.Len(ls, 3)
	}

	// LedgersBySequence
	err = q.LedgersBySequence(&ls, 1, 2, 3)

	if tt.Assert.NoError(err) {
		tt.Assert.Len(ls, 3)

		foundSeqs := make([]int32, len(ls))
		for i := range ls {
			foundSeqs[i] = ls[i].Sequence
		}

		tt.Assert.Contains(foundSeqs, int32(1))
		tt.Assert.Contains(foundSeqs, int32(2))
		tt.Assert.Contains(foundSeqs, int32(3))
	}
}

func TestInsertLedger(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	expectedLedger := Ledger{
		Sequence:                   69859,
		LedgerHash:                 "4db1e4f145e9ee75162040d26284795e0697e2e84084624e7c6c723ebbf80118",
		PreviousLedgerHash:         null.NewString("4b0b8bace3b2438b2404776ce57643966855487ba6384724a3c664c7aa4cd9e4", true),
		TotalOrderID:               TotalOrderID{toid.New(int32(69859), 0, 0).ToInt64()},
		ImporterVersion:            123,
		TransactionCount:           12,
		SuccessfulTransactionCount: new(int32),
		FailedTransactionCount:     new(int32),
		OperationCount:             23,
		TotalCoins:                 23451,
		FeePool:                    213,
		BaseReserve:                687,
		MaxTxSetSize:               345,
		ProtocolVersion:            12,
		BaseFee:                    100,
		ClosedAt:                   time.Now().UTC().Truncate(time.Second),
	}
	*expectedLedger.SuccessfulTransactionCount = 12
	*expectedLedger.FailedTransactionCount = 3

	var ledgerHash, previousLedgerHash xdr.Hash

	written, err := hex.Decode(ledgerHash[:], []byte(expectedLedger.LedgerHash))
	tt.Assert.NoError(err)
	tt.Assert.Equal(len(ledgerHash), written)

	written, err = hex.Decode(previousLedgerHash[:], []byte(expectedLedger.PreviousLedgerHash.String))
	tt.Assert.NoError(err)
	tt.Assert.Equal(len(previousLedgerHash), written)

	ledgerEntry := xdr.LedgerHeaderHistoryEntry{
		Hash: ledgerHash,
		Header: xdr.LedgerHeader{
			LedgerVersion:      12,
			PreviousLedgerHash: previousLedgerHash,
			LedgerSeq:          xdr.Uint32(expectedLedger.Sequence),
			TotalCoins:         xdr.Int64(expectedLedger.TotalCoins),
			FeePool:            xdr.Int64(expectedLedger.FeePool),
			BaseFee:            xdr.Uint32(expectedLedger.BaseFee),
			BaseReserve:        xdr.Uint32(expectedLedger.BaseReserve),
			MaxTxSetSize:       xdr.Uint32(expectedLedger.MaxTxSetSize),
			ScpValue: xdr.StellarValue{
				CloseTime: xdr.TimePoint(expectedLedger.ClosedAt.Unix()),
			},
		},
	}
	ledgerHeaderBase64, err := xdr.MarshalBase64(ledgerEntry.Header)
	tt.Assert.NoError(err)
	expectedLedger.LedgerHeaderXDR = null.NewString(ledgerHeaderBase64, true)

	rowsAffected, err := q.InsertLedger(
		ledgerEntry,
		12,
		3,
		23,
		int(expectedLedger.ImporterVersion),
	)
	tt.Assert.NoError(err)
	tt.Assert.Equal(rowsAffected, int64(1))

	var ledgerFromDB Ledger
	err = q.LedgerBySequence(&ledgerFromDB, 69859)
	tt.Assert.NoError(err)

	expectedLedger.CreatedAt = ledgerFromDB.CreatedAt
	expectedLedger.UpdatedAt = ledgerFromDB.UpdatedAt
	tt.Assert.True(ledgerFromDB.CreatedAt.After(expectedLedger.ClosedAt))
	tt.Assert.True(ledgerFromDB.UpdatedAt.After(expectedLedger.ClosedAt))
	tt.Assert.True(ledgerFromDB.CreatedAt.Before(expectedLedger.ClosedAt.Add(time.Hour)))
	tt.Assert.True(ledgerFromDB.UpdatedAt.Before(expectedLedger.ClosedAt.Add(time.Hour)))

	tt.Assert.True(expectedLedger.ClosedAt.Equal(ledgerFromDB.ClosedAt))
	expectedLedger.ClosedAt = ledgerFromDB.ClosedAt

	tt.Assert.Equal(expectedLedger, ledgerFromDB)
}
