package history

import (
	"database/sql"
	"encoding/hex"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
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

	rowsAffected, err := q.InsertExpLedger(
		ledgerEntry,
		12,
		3,
		23,
		int(expectedLedger.ImporterVersion),
	)
	tt.Assert.NoError(err)
	tt.Assert.Equal(rowsAffected, int64(1))

	ledgerFromDB, err := q.expLedgerBySequence(69859)
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

func ledgerToMap(ledger Ledger) map[string]interface{} {
	return map[string]interface{}{
		"importer_version":             ledger.ImporterVersion,
		"id":                           ledger.TotalOrderID.ID,
		"sequence":                     ledger.Sequence,
		"ledger_hash":                  ledger.LedgerHash,
		"previous_ledger_hash":         ledger.PreviousLedgerHash,
		"total_coins":                  ledger.TotalCoins,
		"fee_pool":                     ledger.FeePool,
		"base_fee":                     ledger.BaseFee,
		"base_reserve":                 ledger.BaseReserve,
		"max_tx_set_size":              ledger.MaxTxSetSize,
		"closed_at":                    ledger.ClosedAt,
		"created_at":                   ledger.CreatedAt,
		"updated_at":                   ledger.UpdatedAt,
		"transaction_count":            ledger.SuccessfulTransactionCount,
		"successful_transaction_count": ledger.SuccessfulTransactionCount,
		"failed_transaction_count":     ledger.FailedTransactionCount,
		"operation_count":              ledger.OperationCount,
		"protocol_version":             ledger.ProtocolVersion,
		"ledger_header":                ledger.LedgerHeaderXDR,
	}
}

func TestCheckExpLedger(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	ledger := Ledger{
		Sequence:                   69859,
		LedgerHash:                 "4db1e4f145e9ee75162040d26284795e0697e2e84084624e7c6c723ebbf80118",
		PreviousLedgerHash:         null.NewString("4b0b8bace3b2438b2404776ce57643966855487ba6384724a3c664c7aa4cd9e4", true),
		TotalOrderID:               TotalOrderID{toid.New(int32(69859), 0, 0).ToInt64()},
		ImporterVersion:            321,
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
		LedgerHeaderXDR:            null.NewString("temp", true),
	}
	*ledger.SuccessfulTransactionCount = 12
	*ledger.FailedTransactionCount = 3

	_, err := q.CheckExpLedger(ledger.Sequence)
	tt.Assert.Equal(err, sql.ErrNoRows)

	insertSQL := sq.Insert("exp_history_ledgers").SetMap(ledgerToMap(ledger))
	_, err = q.Exec(insertSQL)
	tt.Assert.NoError(err)

	_, err = q.CheckExpLedger(ledger.Sequence)
	tt.Assert.Equal(err, sql.ErrNoRows)

	ledger.CreatedAt = time.Now()
	ledger.UpdatedAt = time.Now()
	ledger.ImporterVersion = 123

	insertSQL = sq.Insert("history_ledgers").SetMap(ledgerToMap(ledger))
	_, err = q.Exec(insertSQL)
	tt.Assert.NoError(err)

	valid, err := q.CheckExpLedger(ledger.Sequence)
	tt.Assert.NoError(err)
	tt.Assert.True(valid)

	for fieldName, value := range map[string]interface{}{
		"closed_at":                    time.Now().Add(time.Minute).UTC().Truncate(time.Second),
		"ledger_hash":                  "hash",
		"previous_ledger_hash":         "previous",
		"id":                           999,
		"total_coins":                  9999,
		"fee_pool":                     9999,
		"base_fee":                     9999,
		"base_reserve":                 9999,
		"max_tx_set_size":              9999,
		"transaction_count":            9999,
		"successful_transaction_count": 9999,
		"failed_transaction_count":     9999,
		"operation_count":              9999,
		"protocol_version":             9999,
		"ledger_header":                "ledger header",
	} {
		updateSQL := sq.Update("history_ledgers").
			Set(fieldName, value).
			Where("sequence = ?", ledger.Sequence)
		_, err = q.Exec(updateSQL)
		tt.Assert.NoError(err)

		valid, err = q.CheckExpLedger(ledger.Sequence)
		tt.Assert.NoError(err)
		tt.Assert.False(valid)

		_, err = q.Exec(sq.Delete("history_ledgers").Where("sequence = ?", ledger.Sequence))
		tt.Assert.NoError(err)

		insertSQL = sq.Insert("history_ledgers").SetMap(ledgerToMap(ledger))
		_, err = q.Exec(insertSQL)
		tt.Assert.NoError(err)

		valid, err := q.CheckExpLedger(ledger.Sequence)
		tt.Assert.NoError(err)
		tt.Assert.True(valid)
	}
}
