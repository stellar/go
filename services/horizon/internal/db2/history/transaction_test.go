package history

import (
	"database/sql"
	"encoding/hex"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/db2/sqx"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/xdr"
)

func TestTransactionQueries(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()
	q := &Q{tt.HorizonSession()}

	// Test TransactionByHash
	var tx Transaction
	real := "2374e99349b9ef7dba9a5db3339b78fda8f34777b1af33ba468ad5c0df946d4d"
	err := q.TransactionByHash(&tx, real)
	tt.Assert.NoError(err)

	fake := "not_real"
	err = q.TransactionByHash(&tx, fake)
	tt.Assert.Equal(err, sql.ErrNoRows)
}

// TestTransactionSuccessfulOnly tests if default query returns successful
// transactions only.
// If it's not enclosed in brackets, it may return incorrect result when mixed
// with `ForAccount` or `ForLedger` filters.
func TestTransactionSuccessfulOnly(t *testing.T) {
	tt := test.Start(t).Scenario("failed_transactions")
	defer tt.Finish()

	var transactions []Transaction

	q := &Q{tt.HorizonSession()}
	query := q.Transactions().
		ForAccount("GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2")

	err := query.Select(&transactions)
	tt.Assert.NoError(err)

	tt.Assert.Equal(3, len(transactions))

	for _, transaction := range transactions {
		tt.Assert.True(*transaction.Successful)
	}

	sql, _, err := query.sql.ToSql()
	tt.Assert.NoError(err)
	// Note: brackets around `(ht.successful = true OR ht.successful IS NULL)` are critical!
	tt.Assert.Contains(sql, "WHERE htp.history_account_id = ? AND (ht.successful = true OR ht.successful IS NULL)")
}

// TestTransactionIncludeFailed tests `IncludeFailed` method.
func TestTransactionIncludeFailed(t *testing.T) {
	tt := test.Start(t).Scenario("failed_transactions")
	defer tt.Finish()

	var transactions []Transaction

	q := &Q{tt.HorizonSession()}
	query := q.Transactions().
		ForAccount("GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2").
		IncludeFailed()

	err := query.Select(&transactions)
	tt.Assert.NoError(err)

	var failed, successful int
	for _, transaction := range transactions {
		if *transaction.Successful {
			successful++
		} else {
			failed++
		}
	}

	tt.Assert.Equal(3, successful)
	tt.Assert.Equal(1, failed)

	sql, _, err := query.sql.ToSql()
	tt.Assert.NoError(err)
	tt.Assert.Equal("SELECT ht.id, ht.transaction_hash, ht.ledger_sequence, ht.application_order, ht.account, ht.account_sequence, ht.max_fee, COALESCE(ht.fee_charged, ht.max_fee) as fee_charged, ht.operation_count, ht.tx_envelope, ht.tx_result, ht.tx_meta, ht.tx_fee_meta, ht.created_at, ht.updated_at, ht.successful, array_to_string(ht.signatures, ',') AS signatures, ht.memo_type, ht.memo, lower(ht.time_bounds) AS valid_after, upper(ht.time_bounds) AS valid_before, hl.closed_at AS ledger_close_time FROM history_transactions ht LEFT JOIN history_ledgers hl ON ht.ledger_sequence = hl.sequence JOIN history_transaction_participants htp ON htp.history_transaction_id = ht.id WHERE htp.history_account_id = ?", sql)
}

func TestExtraChecksTransactionSuccessfulTrueResultFalse(t *testing.T) {
	tt := test.Start(t).Scenario("failed_transactions")
	defer tt.Finish()

	// successful `true` but tx result `false`
	_, err := tt.HorizonDB.Exec(
		`UPDATE history_transactions SET successful = true WHERE transaction_hash = 'aa168f12124b7c196c0adaee7c73a64d37f99428cacb59a91ff389626845e7cf'`,
	)
	tt.Require.NoError(err)

	var transactions []Transaction

	q := &Q{tt.HorizonSession()}
	query := q.Transactions().
		ForAccount("GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2").
		IncludeFailed()

	err = query.Select(&transactions)
	tt.Assert.Error(err)
	tt.Assert.Contains(err.Error(), "Corrupted data! `successful=true` but returned transaction is not success")
}

func TestExtraChecksTransactionSuccessfulFalseResultTrue(t *testing.T) {
	tt := test.Start(t).Scenario("failed_transactions")
	defer tt.Finish()

	// successful `false` but tx result `true`
	_, err := tt.HorizonDB.Exec(
		`UPDATE history_transactions SET successful = false WHERE transaction_hash = 'a2dabf4e9d1642722602272e178a37c973c9177b957da86192a99b3e9f3a9aa4'`,
	)
	tt.Require.NoError(err)

	var transactions []Transaction

	q := &Q{tt.HorizonSession()}
	query := q.Transactions().
		ForAccount("GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON").
		IncludeFailed()

	err = query.Select(&transactions)
	tt.Assert.Error(err)
	tt.Assert.Contains(err.Error(), "Corrupted data! `successful=false` but returned transaction is success")
}

func insertTransaction(
	tt *test.T, q *Q, tableName string, transaction io.LedgerTransaction, sequence int32,
) {
	m, err := transactionToMap(transaction, uint32(sequence))
	tt.Assert.NoError(err)
	insertSQL := sq.Insert(tableName).SetMap(m)
	_, err = q.Exec(insertSQL)
	tt.Assert.NoError(err)
}

type testTransaction struct {
	index         uint32
	envelopeXDR   string
	resultXDR     string
	feeChangesXDR string
	metaXDR       string
	hash          string
}

func buildLedgerTransaction(tt *test.T, tx testTransaction) io.LedgerTransaction {
	transaction := io.LedgerTransaction{
		Index:      tx.index,
		Envelope:   xdr.TransactionEnvelope{},
		Result:     xdr.TransactionResultPair{},
		FeeChanges: xdr.LedgerEntryChanges{},
		Meta:       xdr.TransactionMeta{},
	}

	err := xdr.SafeUnmarshalBase64(tx.envelopeXDR, &transaction.Envelope)
	tt.Assert.NoError(err)
	err = xdr.SafeUnmarshalBase64(tx.resultXDR, &transaction.Result.Result)
	tt.Assert.NoError(err)
	err = xdr.SafeUnmarshalBase64(tx.metaXDR, &transaction.Meta)
	tt.Assert.NoError(err)
	err = xdr.SafeUnmarshalBase64(tx.feeChangesXDR, &transaction.FeeChanges)
	tt.Assert.NoError(err)

	_, err = hex.Decode(transaction.Result.TransactionHash[:], []byte(tx.hash))
	tt.Assert.NoError(err)

	return transaction
}

func TestCheckExpTransactions(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	transaction := buildLedgerTransaction(
		tt,
		testTransaction{
			index:         1,
			envelopeXDR:   "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAZAEXUhsAADDGAAAAAQAAAAAAAAAAAAAAAF3v3WAAAAABAAAACjEwOTUzMDMyNTAAAAAAAAEAAAAAAAAAAQAAAAAOr5CG1ax6qG2fBEgXJlF0sw5W0irOS6N/NRDbavBm4QAAAAAAAAAAE32fwAAAAAAAAAABf/7fqwAAAEAkWgyAgV5tF3m1y1TIDYkNXP8pZLAwcxhWEi4f3jcZJK7QrKSXhKoawVGrp5NNs4y9dgKt8zHZ8KbJreFBUsIB",
			resultXDR:     "AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=",
			feeChangesXDR: "AAAAAgAAAAMBnyVBAAAAAAAAAAAokk0ZqR+mxwuhJJ2uXvNqIhmObygxBFIJKvQgf/7fqwAALNdjj1x7ARdSGwAAMMUAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEBnye/AAAAAAAAAAAokk0ZqR+mxwuhJJ2uXvNqIhmObygxBFIJKvQgf/7fqwAALNdjj1wXARdSGwAAMMUAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==",
			metaXDR:       "AAAAAAAAAAMAAAACAAAAAAAAAAMAAAAAAAAAABbxCy3mLg3hiTqX4VUEEp60pFOrJNxYM1JtxXTwXhY2AAAAAAvrwgAAAAADAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAMAAAAAAAAAAAGUcmKO5465JxTSLQOQljwk2SfqAJmZSG6JH6wtqpwhDeC2s5t4PNQAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAwAAAAEAAAADAAAAAAAAAAABlHJijueOuScU0i0DkJY8JNkn6gCZmUhuiR+sLaqcIQAAAAAL68IAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAMAAAADAAAAAAAAAAAW8Qst5i4N4Yk6l+FVBBKetKRTqyTcWDNSbcV08F4WNgAAAAAL68IAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAADAAAAAAAAAAAW8Qst5i4N4Yk6l+FVBBKetKRTqyTcWDNSbcV08F4WNg3gtrObeDzUAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAABAAAAAwAAAAAAAAAAAZRyYo7njrknFNItA5CWPCTZJ+oAmZlIbokfrC2qnCEAAAAAC+vCAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
			hash:          "ea1e96dd4aa8a16e357842b0fcdb66c1ab03fade7dcd3a99f88ed28ea8c30f6a",
		},
	)

	otherTransaction := buildLedgerTransaction(
		tt,
		testTransaction{
			index:         2,
			envelopeXDR:   "AAAAAAGUcmKO5465JxTSLQOQljwk2SfqAJmZSG6JH6wtqpwhAAABLAAAAAAAAAABAAAAAAAAAAEAAAALaGVsbG8gd29ybGQAAAAAAwAAAAAAAAAAAAAAABbxCy3mLg3hiTqX4VUEEp60pFOrJNxYM1JtxXTwXhY2AAAAAAvrwgAAAAAAAAAAAQAAAAAW8Qst5i4N4Yk6l+FVBBKetKRTqyTcWDNSbcV08F4WNgAAAAAN4Lazj4x61AAAAAAAAAAFAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABLaqcIQAAAEBKwqWy3TaOxoGnfm9eUjfTRBvPf34dvDA0Nf+B8z4zBob90UXtuCqmQqwMCyH+okOI3c05br3khkH0yP4kCwcE",
			resultXDR:     "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
			feeChangesXDR: "AAAAAgAAAAMBnyVBAAAAAAAAAAAokk0ZqR+mxwuhJJ2uXvNqIhmObygxBFIJKvQgf/7fqwAALNdjj1x7ARdSGwAAMMUAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEBnye/AAAAAAAAAAAokk0ZqR+mxwuhJJ2uXvNqIhmObygxBFIJKvQgf/7fqwAALNdjj1wXARdSGwAAMMUAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==",
			metaXDR:       "AAAAAAAAAAMAAAACAAAAAAAAHqEAAAAAAAAAAB+lHtRjj4+h2/0Tj8iBQiaUDzLo4oRCLyUnytFHzAyIAAAAAAvrwgAAAB6hAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAHqEAAAAAAAAAABbxCy3mLg3hiTqX4VUEEp60pFOrJNxYM1JtxXTwXhY2DeC2s4+MeHwAAAADAAAAAgAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAgAAAAAAAB6hAAAAAAAAAACzMOD+8iU8qo+qbTYewT8lxKE/s1cE3FOCVWxsqJ74GwAAAAAL68IAAAAeoQAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAB6hAAAAAAAAAAAW8Qst5i4N4Yk6l+FVBBKetKRTqyTcWDNSbcV08F4WNg3gtrODoLZ8AAAAAwAAAAIAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAIAAAAAAAAeoQAAAAAAAAAASZcLtOTqf+cdbsq8HmLMkeqU06LN94UTWXuSBem5Z88AAAAAC+vCAAAAHqEAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAeoQAAAAAAAAAAFvELLeYuDeGJOpfhVQQSnrSkU6sk3FgzUm3FdPBeFjYN4Lazd7T0fAAAAAMAAAACAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAA=",
			hash:          "3389e9f0f1a65f19736cacf544c2e825313e8447f569233bb8db39aa607c8889",
		},
	)

	sequence := int32(123)
	valid, err := q.CheckExpTransactions(sequence)
	tt.Assert.True(valid)
	tt.Assert.NoError(err)

	insertTransaction(tt, q, "exp_history_transactions", transaction, sequence)
	insertTransaction(tt, q, "exp_history_transactions", otherTransaction, sequence)
	insertTransaction(tt, q, "history_transactions", transaction, sequence)

	ledger := Ledger{
		Sequence:                   sequence,
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
	_, err = q.Exec(sq.Insert("history_ledgers").SetMap(ledgerToMap(ledger)))
	tt.Assert.NoError(err)
	_, err = q.Exec(sq.Insert("exp_history_ledgers").SetMap(ledgerToMap(ledger)))
	tt.Assert.NoError(err)

	valid, err = q.CheckExpTransactions(sequence)
	tt.Assert.False(valid)
	tt.Assert.NoError(err)

	insertTransaction(tt, q, "history_transactions", otherTransaction, sequence)

	valid, err = q.CheckExpTransactions(sequence)
	tt.Assert.True(valid)
	tt.Assert.NoError(err)

	for fieldName, value := range map[string]interface{}{
		"id":               999,
		"transaction_hash": "hash",
		"account":          "account",
		"account_sequence": "999",
		"max_fee":          999,
		"fee_charged":      999,
		"operation_count":  999,
		"tx_envelope":      "envelope",
		"tx_result":        "result",
		"tx_meta":          "meta",
		"tx_fee_meta":      "fee_meta",
		"signatures":       sqx.StringArray([]string{"sig1", "sig2"}),
		"time_bounds":      sq.Expr("int8range(?,?)", 123, 456),
		"memo_type":        "invalid",
		"memo":             "invalid-memo",
		"successful":       false,
	} {
		updateSQL := sq.Update("history_transactions").
			Set(fieldName, value).
			Where(
				"ledger_sequence = ? AND application_order = ?",
				sequence, otherTransaction.Index,
			)
		_, err = q.Exec(updateSQL)
		tt.Assert.NoError(err)

		valid, err = q.CheckExpTransactions(sequence)
		tt.Assert.NoError(err)
		tt.Assert.False(valid)

		_, err = q.Exec(sq.Delete("history_transactions").
			Where(
				"ledger_sequence = ? AND application_order = ?",
				sequence, otherTransaction.Index,
			))
		tt.Assert.NoError(err)

		insertTransaction(tt, q, "history_transactions", otherTransaction, sequence)

		valid, err := q.CheckExpTransactions(ledger.Sequence)
		tt.Assert.NoError(err)
		tt.Assert.True(valid)
	}
}

func TestInsertExpTransactionDoesNotAllowDuplicateIndex(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	sequence := uint32(123)
	insertBuilder := q.NewTransactionBatchInsertBuilder(0)

	firstTransaction := buildLedgerTransaction(tt, testTransaction{
		index:         1,
		envelopeXDR:   "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAyAEXUhsAADDRAAAAAAAAAAAAAAABAAAAAAAAAAsBF1IbAABX4QAAAAAAAAAA",
		resultXDR:     "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
		feeChangesXDR: "AAAAAA==",
		metaXDR:       "AAAAAQAAAAAAAAAA",
		hash:          "19aaa18db88605aedec04659fb45e06f240b022eb2d429e05133e4d53cd945ba",
	})
	secondTransaction := buildLedgerTransaction(tt, testTransaction{
		index:         1,
		envelopeXDR:   "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAyAEXUhsAADDRAAAAAAAAAAIAAAAAAAAAewAAAAEAAAAAAAAACwEXUhsAAFfhAAAAAAAAAAA=",
		resultXDR:     "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
		feeChangesXDR: "AAAAAA==",
		metaXDR:       "AAAAAQAAAAAAAAAA",
		hash:          "7e2def20d5a21a56be2a457b648f702ee1af889d3df65790e92a05081e9fabf1",
	})

	tt.Assert.NoError(insertBuilder.Add(firstTransaction, sequence))
	tt.Assert.NoError(insertBuilder.Exec())

	tt.Assert.NoError(insertBuilder.Add(secondTransaction, sequence))
	tt.Assert.EqualError(
		insertBuilder.Exec(),
		"error adding values while inserting to exp_history_transactions: "+
			"exec failed: pq: duplicate key value violates unique constraint "+
			"\"exp_history_transactions_id_idx\"",
	)

	ledger := Ledger{
		Sequence:                   int32(sequence),
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
	_, err := q.Exec(sq.Insert("exp_history_ledgers").SetMap(ledgerToMap(ledger)))
	tt.Assert.NoError(err)

	var transactions []Transaction
	tt.Assert.NoError(q.Select(&transactions, selectExpTransaction))
	tt.Assert.Len(transactions, 1)
	tt.Assert.Equal(
		"19aaa18db88605aedec04659fb45e06f240b022eb2d429e05133e4d53cd945ba",
		transactions[0].TransactionHash,
	)
}

func TestInsertExpTransaction(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	sequence := uint32(123)
	ledger := Ledger{
		Sequence:                   int32(sequence),
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
	_, err := q.Exec(sq.Insert("history_ledgers").SetMap(ledgerToMap(ledger)))
	tt.Assert.NoError(err)
	_, err = q.Exec(sq.Insert("exp_history_ledgers").SetMap(ledgerToMap(ledger)))
	tt.Assert.NoError(err)

	insertBuilder := q.NewTransactionBatchInsertBuilder(0)
	success := new(bool)
	*success = true

	for _, testCase := range []struct {
		name     string
		toInsert io.LedgerTransaction
		expected Transaction
	}{
		{
			"successful transaction without signatures",
			buildLedgerTransaction(tt, testTransaction{
				index:         1,
				envelopeXDR:   "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAyAEXUhsAADDRAAAAAAAAAAAAAAABAAAAAAAAAAsBF1IbAABX4QAAAAAAAAAA",
				resultXDR:     "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
				feeChangesXDR: "AAAAAA==",
				metaXDR:       "AAAAAQAAAAAAAAAA",
				hash:          "19aaa18db88605aedec04659fb45e06f240b022eb2d429e05133e4d53cd945ba",
			}),
			Transaction{
				TotalOrderID:     TotalOrderID{528280981504},
				TransactionHash:  "19aaa18db88605aedec04659fb45e06f240b022eb2d429e05133e4d53cd945ba",
				LedgerSequence:   ledger.Sequence,
				LedgerCloseTime:  ledger.ClosedAt,
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
				SignatureString:  "",
				MemoType:         "none",
				Memo:             null.NewString("", false),
				ValidAfter:       null.NewInt(0, false),
				ValidBefore:      null.NewInt(0, false),
				Successful:       success,
			},
		},
		{
			"successful transaction with multiple signatures",
			buildLedgerTransaction(tt, testTransaction{
				index:         1,
				envelopeXDR:   "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAyAEXUhsAADDRAAAAAAAAAAAAAAABAAAAAAAAAAsBF1IbAABX4QAAAAAAAAACQmz0pAAAAEAwgPyQg4s//ITKYLKFTsD9h5WmdVMPMCqzNa2/xz7+oDDnjYFopbEInGX+OyCBjSYX5JHxfRu9Ze88GDJhwNkPto+xlgAAAEAnQnypOwpERbb0YCZkxdcFNWRgqQZs6TUBQ9RqVsiIN0ON3yakgh0xK2qj3D6TkDxQpFOb+I7ZX+uJDXgIX0gC",
				resultXDR:     "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
				feeChangesXDR: "AAAAAA==",
				metaXDR:       "AAAAAQAAAAAAAAAA",
				hash:          "19aaa18db88605aedec04659fb45e06f240b022eb2d429e05133e4d53cd945ba",
			}),
			Transaction{
				TotalOrderID:     TotalOrderID{528280981504},
				TransactionHash:  "19aaa18db88605aedec04659fb45e06f240b022eb2d429e05133e4d53cd945ba",
				LedgerSequence:   ledger.Sequence,
				LedgerCloseTime:  ledger.ClosedAt,
				ApplicationOrder: 1,
				Account:          "GAUJETIZVEP2NRYLUESJ3LS66NVCEGMON4UDCBCSBEVPIID773P2W6AY",
				AccountSequence:  "78621794419880145",
				MaxFee:           200,
				FeeCharged:       300,
				OperationCount:   1,
				TxEnvelope:       "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAyAEXUhsAADDRAAAAAAAAAAAAAAABAAAAAAAAAAsBF1IbAABX4QAAAAAAAAACQmz0pAAAAEAwgPyQg4s//ITKYLKFTsD9h5WmdVMPMCqzNa2/xz7+oDDnjYFopbEInGX+OyCBjSYX5JHxfRu9Ze88GDJhwNkPto+xlgAAAEAnQnypOwpERbb0YCZkxdcFNWRgqQZs6TUBQ9RqVsiIN0ON3yakgh0xK2qj3D6TkDxQpFOb+I7ZX+uJDXgIX0gC",
				TxResult:         "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
				TxFeeMeta:        "AAAAAA==",
				TxMeta:           "AAAAAQAAAAAAAAAA",
				SignatureString:  "MID8kIOLP/yEymCyhU7A/YeVpnVTDzAqszWtv8c+/qAw542BaKWxCJxl/jsggY0mF+SR8X0bvWXvPBgyYcDZDw==,J0J8qTsKREW29GAmZMXXBTVkYKkGbOk1AUPUalbIiDdDjd8mpIIdMStqo9w+k5A8UKRTm/iO2V/riQ14CF9IAg==",
				MemoType:         "none",
				Memo:             null.NewString("", false),
				ValidAfter:       null.NewInt(0, false),
				ValidBefore:      null.NewInt(0, false),
				Successful:       success,
			},
		},
		{
			"failed transaction",
			buildLedgerTransaction(tt, testTransaction{
				index:         1,
				envelopeXDR:   "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAyAEXUhsAADDRAAAAAAAAAAAAAAABAAAAAAAAAAsBF1IbAABX4QAAAAAAAAABQmz0pAAAAEAwgPyQg4s//ITKYLKFTsD9h5WmdVMPMCqzNa2/xz7+oDDnjYFopbEInGX+OyCBjSYX5JHxfRu9Ze88GDJhwNkP",
				resultXDR:     "AAAAAAAAAHv////6AAAAAA==",
				feeChangesXDR: "AAAAAA==",
				metaXDR:       "AAAAAQAAAAAAAAAA",
				hash:          "19aaa18db88605aedec04659fb45e06f240b022eb2d429e05133e4d53cd945ba",
			}),
			Transaction{
				TotalOrderID:     TotalOrderID{528280981504},
				TransactionHash:  "19aaa18db88605aedec04659fb45e06f240b022eb2d429e05133e4d53cd945ba",
				LedgerSequence:   ledger.Sequence,
				LedgerCloseTime:  ledger.ClosedAt,
				ApplicationOrder: 1,
				Account:          "GAUJETIZVEP2NRYLUESJ3LS66NVCEGMON4UDCBCSBEVPIID773P2W6AY",
				AccountSequence:  "78621794419880145",
				MaxFee:           200,
				FeeCharged:       123,
				OperationCount:   1,
				TxEnvelope:       "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAyAEXUhsAADDRAAAAAAAAAAAAAAABAAAAAAAAAAsBF1IbAABX4QAAAAAAAAABQmz0pAAAAEAwgPyQg4s//ITKYLKFTsD9h5WmdVMPMCqzNa2/xz7+oDDnjYFopbEInGX+OyCBjSYX5JHxfRu9Ze88GDJhwNkP",
				TxResult:         "AAAAAAAAAHv////6AAAAAA==",
				TxFeeMeta:        "AAAAAA==",
				TxMeta:           "AAAAAQAAAAAAAAAA",
				SignatureString:  "MID8kIOLP/yEymCyhU7A/YeVpnVTDzAqszWtv8c+/qAw542BaKWxCJxl/jsggY0mF+SR8X0bvWXvPBgyYcDZDw==",
				MemoType:         "none",
				Memo:             null.NewString("", false),
				ValidAfter:       null.NewInt(0, false),
				ValidBefore:      null.NewInt(0, false),
				Successful:       new(bool),
			},
		},
		{
			"transaction with text memo",
			buildLedgerTransaction(tt, testTransaction{
				index:         1,
				envelopeXDR:   "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAyAEXUhsAADDRAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAABAAAACXRlc3QgbWVtbwAAAAAAAAEAAAAAAAAACwEXUhsAAFfhAAAAAAAAAAA=",
				resultXDR:     "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
				feeChangesXDR: "AAAAAA==",
				metaXDR:       "AAAAAQAAAAAAAAAA",
				hash:          "edba3051b2f2d9b713e8a08709d631eccb72c59864ff3c564c68792271bb24a7",
			}),
			Transaction{
				TotalOrderID:     TotalOrderID{528280981504},
				TransactionHash:  "edba3051b2f2d9b713e8a08709d631eccb72c59864ff3c564c68792271bb24a7",
				LedgerSequence:   ledger.Sequence,
				LedgerCloseTime:  ledger.ClosedAt,
				ApplicationOrder: 1,
				Account:          "GAUJETIZVEP2NRYLUESJ3LS66NVCEGMON4UDCBCSBEVPIID773P2W6AY",
				AccountSequence:  "78621794419880145",
				MaxFee:           200,
				FeeCharged:       300,
				OperationCount:   1,
				TxEnvelope:       "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAyAEXUhsAADDRAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAABAAAACXRlc3QgbWVtbwAAAAAAAAEAAAAAAAAACwEXUhsAAFfhAAAAAAAAAAA=",
				TxResult:         "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
				TxFeeMeta:        "AAAAAA==",
				TxMeta:           "AAAAAQAAAAAAAAAA",
				SignatureString:  "",
				MemoType:         "text",
				Memo:             null.NewString("test memo", true),
				ValidAfter:       null.NewInt(0, true),
				ValidBefore:      null.NewInt(0, false),
				Successful:       success,
			},
		},
		{
			"transaction with id memo",
			buildLedgerTransaction(tt, testTransaction{
				index:         1,
				envelopeXDR:   "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAyAEXUhsAADDRAAAAAAAAAAIAAAAAAAAAewAAAAEAAAAAAAAACwEXUhsAAFfhAAAAAAAAAAA=",
				resultXDR:     "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
				feeChangesXDR: "AAAAAA==",
				metaXDR:       "AAAAAQAAAAAAAAAA",
				hash:          "7e2def20d5a21a56be2a457b648f702ee1af889d3df65790e92a05081e9fabf1",
			}),
			Transaction{
				TotalOrderID:     TotalOrderID{528280981504},
				TransactionHash:  "7e2def20d5a21a56be2a457b648f702ee1af889d3df65790e92a05081e9fabf1",
				LedgerSequence:   ledger.Sequence,
				LedgerCloseTime:  ledger.ClosedAt,
				ApplicationOrder: 1,
				Account:          "GAUJETIZVEP2NRYLUESJ3LS66NVCEGMON4UDCBCSBEVPIID773P2W6AY",
				AccountSequence:  "78621794419880145",
				MaxFee:           200,
				FeeCharged:       300,
				OperationCount:   1,
				TxEnvelope:       "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAyAEXUhsAADDRAAAAAAAAAAIAAAAAAAAAewAAAAEAAAAAAAAACwEXUhsAAFfhAAAAAAAAAAA=",
				TxResult:         "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
				TxFeeMeta:        "AAAAAA==",
				TxMeta:           "AAAAAQAAAAAAAAAA",
				SignatureString:  "",
				MemoType:         "id",
				Memo:             null.NewString("123", true),
				ValidAfter:       null.NewInt(0, false),
				ValidBefore:      null.NewInt(0, false),
				Successful:       success,
			},
		},
		{
			"transaction with hash memo",
			buildLedgerTransaction(tt, testTransaction{
				index:         1,
				envelopeXDR:   "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAyAEXUhsAADDRAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAADfi3vINWiGla+KkV7ZI9wLuGviJ099leQ6SoFCB6fq/EAAAABAAAAAAAAAAsBF1IbAABX4QAAAAAAAAAA",
				resultXDR:     "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
				feeChangesXDR: "AAAAAA==",
				metaXDR:       "AAAAAQAAAAAAAAAA",
				hash:          "8aba253c2adc4083f35830cec37d9c6226b757ab3a94f15a12d6c22973fd5f3f",
			}),
			Transaction{
				TotalOrderID:     TotalOrderID{528280981504},
				TransactionHash:  "8aba253c2adc4083f35830cec37d9c6226b757ab3a94f15a12d6c22973fd5f3f",
				LedgerSequence:   ledger.Sequence,
				LedgerCloseTime:  ledger.ClosedAt,
				ApplicationOrder: 1,
				Account:          "GAUJETIZVEP2NRYLUESJ3LS66NVCEGMON4UDCBCSBEVPIID773P2W6AY",
				AccountSequence:  "78621794419880145",
				MaxFee:           200,
				FeeCharged:       300,
				OperationCount:   1,
				TxEnvelope:       "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAyAEXUhsAADDRAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAADfi3vINWiGla+KkV7ZI9wLuGviJ099leQ6SoFCB6fq/EAAAABAAAAAAAAAAsBF1IbAABX4QAAAAAAAAAA",
				TxResult:         "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
				TxFeeMeta:        "AAAAAA==",
				TxMeta:           "AAAAAQAAAAAAAAAA",
				SignatureString:  "",
				MemoType:         "hash",
				Memo:             null.NewString("fi3vINWiGla+KkV7ZI9wLuGviJ099leQ6SoFCB6fq/E=", true),
				ValidAfter:       null.NewInt(0, true),
				ValidBefore:      null.NewInt(0, false),
				Successful:       success,
			},
		},
		{
			"transaction with return memo",
			buildLedgerTransaction(tt, testTransaction{
				index:         1,
				envelopeXDR:   "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAyAEXUhsAADDRAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAEzdjArlILa/LNv7o7lo/qv5+fVVPNl0yPgZQWB6u+gL4AAAABAAAAAAAAAAsBF1IbAABX4QAAAAAAAAAA",
				resultXDR:     "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
				feeChangesXDR: "AAAAAA==",
				metaXDR:       "AAAAAQAAAAAAAAAA",
				hash:          "9e932a86cea43239aed62d8cd3b6b5ad2d8eb0a63247355e4ab44f2994209f2a",
			}),
			Transaction{
				TotalOrderID:     TotalOrderID{528280981504},
				TransactionHash:  "9e932a86cea43239aed62d8cd3b6b5ad2d8eb0a63247355e4ab44f2994209f2a",
				LedgerSequence:   ledger.Sequence,
				LedgerCloseTime:  ledger.ClosedAt,
				ApplicationOrder: 1,
				Account:          "GAUJETIZVEP2NRYLUESJ3LS66NVCEGMON4UDCBCSBEVPIID773P2W6AY",
				AccountSequence:  "78621794419880145",
				MaxFee:           200,
				FeeCharged:       300,
				OperationCount:   1,
				TxEnvelope:       "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAyAEXUhsAADDRAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAEzdjArlILa/LNv7o7lo/qv5+fVVPNl0yPgZQWB6u+gL4AAAABAAAAAAAAAAsBF1IbAABX4QAAAAAAAAAA",
				TxResult:         "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
				TxFeeMeta:        "AAAAAA==",
				TxMeta:           "AAAAAQAAAAAAAAAA",
				SignatureString:  "",
				MemoType:         "return",
				Memo:             null.NewString("zdjArlILa/LNv7o7lo/qv5+fVVPNl0yPgZQWB6u+gL4=", true),
				ValidAfter:       null.NewInt(0, true),
				ValidBefore:      null.NewInt(0, false),
				Successful:       success,
			},
		},
		{
			"transaction with min time bound",
			buildLedgerTransaction(tt, testTransaction{
				index:         1,
				envelopeXDR:   "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAZAAAAAAAAeJAAAAAAQAAAABd8tcbAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAALAAAAAAAS1ocAAAAAAAAAAA==",
				resultXDR:     "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
				feeChangesXDR: "AAAAAA==",
				metaXDR:       "AAAAAQAAAAAAAAAA",
				hash:          "2a9ec3733989aa9a542ed6d0adbcc664915b1c3a72a019e6e23c2860f1ab342b",
			}),
			Transaction{
				TotalOrderID:     TotalOrderID{528280981504},
				TransactionHash:  "2a9ec3733989aa9a542ed6d0adbcc664915b1c3a72a019e6e23c2860f1ab342b",
				LedgerSequence:   ledger.Sequence,
				LedgerCloseTime:  ledger.ClosedAt,
				ApplicationOrder: 1,
				Account:          "GAUJETIZVEP2NRYLUESJ3LS66NVCEGMON4UDCBCSBEVPIID773P2W6AY",
				AccountSequence:  "123456",
				MaxFee:           100,
				FeeCharged:       300,
				OperationCount:   1,
				TxEnvelope:       "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAZAAAAAAAAeJAAAAAAQAAAABd8tcbAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAALAAAAAAAS1ocAAAAAAAAAAA==",
				TxResult:         "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
				TxFeeMeta:        "AAAAAA==",
				TxMeta:           "AAAAAQAAAAAAAAAA",
				SignatureString:  "",
				MemoType:         "none",
				Memo:             null.NewString("", false),
				ValidAfter:       null.NewInt(1576195867, true),
				ValidBefore:      null.NewInt(0, false),
				Successful:       success,
			},
		},
		{
			"transaction with max time bound",
			buildLedgerTransaction(tt, testTransaction{
				index:         1,
				envelopeXDR:   "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAZAAAAAAAAeJAAAAAAQAAAAAAAAAAAAAAAF3y1xsAAAAAAAAAAQAAAAAAAAALAAAAAAAS1ocAAAAAAAAAAA==",
				resultXDR:     "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
				feeChangesXDR: "AAAAAA==",
				metaXDR:       "AAAAAQAAAAAAAAAA",
				hash:          "5858709ae02992431f98f7410be3d3586c5a83e9e7105d270ce1ddc2cf45358a",
			}),
			Transaction{
				TotalOrderID:     TotalOrderID{528280981504},
				TransactionHash:  "5858709ae02992431f98f7410be3d3586c5a83e9e7105d270ce1ddc2cf45358a",
				LedgerSequence:   ledger.Sequence,
				LedgerCloseTime:  ledger.ClosedAt,
				ApplicationOrder: 1,
				Account:          "GAUJETIZVEP2NRYLUESJ3LS66NVCEGMON4UDCBCSBEVPIID773P2W6AY",
				AccountSequence:  "123456",
				MaxFee:           100,
				FeeCharged:       300,
				OperationCount:   1,
				TxEnvelope:       "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAZAAAAAAAAeJAAAAAAQAAAAAAAAAAAAAAAF3y1xsAAAAAAAAAAQAAAAAAAAALAAAAAAAS1ocAAAAAAAAAAA==",
				TxResult:         "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
				TxFeeMeta:        "AAAAAA==",
				TxMeta:           "AAAAAQAAAAAAAAAA",
				SignatureString:  "",
				MemoType:         "none",
				Memo:             null.NewString("", false),
				ValidAfter:       null.NewInt(0, true),
				ValidBefore:      null.NewInt(1576195867, true),
				Successful:       success,
			},
		},
		{
			"transaction with min and max time bound",
			buildLedgerTransaction(tt, testTransaction{
				index:         1,
				envelopeXDR:   "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAZAAAAAAAAeJAAAAAAQAAAABd8VB7AAAAAF3y1xsAAAAAAAAAAQAAAAAAAAALAAAAAAAS1ocAAAAAAAAAAA==",
				resultXDR:     "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
				feeChangesXDR: "AAAAAA==",
				metaXDR:       "AAAAAQAAAAAAAAAA",
				hash:          "7aa3419a833fb14e312ae47a98e565f668a72f23c39e0cf79f598d3d3e793b2d",
			}),
			Transaction{
				TotalOrderID:     TotalOrderID{528280981504},
				TransactionHash:  "7aa3419a833fb14e312ae47a98e565f668a72f23c39e0cf79f598d3d3e793b2d",
				LedgerSequence:   ledger.Sequence,
				LedgerCloseTime:  ledger.ClosedAt,
				ApplicationOrder: 1,
				Account:          "GAUJETIZVEP2NRYLUESJ3LS66NVCEGMON4UDCBCSBEVPIID773P2W6AY",
				AccountSequence:  "123456",
				MaxFee:           100,
				FeeCharged:       300,
				OperationCount:   1,
				TxEnvelope:       "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAZAAAAAAAAeJAAAAAAQAAAABd8VB7AAAAAF3y1xsAAAAAAAAAAQAAAAAAAAALAAAAAAAS1ocAAAAAAAAAAA==",
				TxResult:         "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
				TxFeeMeta:        "AAAAAA==",
				TxMeta:           "AAAAAQAAAAAAAAAA",
				SignatureString:  "",
				MemoType:         "none",
				Memo:             null.NewString("", false),
				ValidAfter:       null.NewInt(1576095867, true),
				ValidBefore:      null.NewInt(1576195867, true),
				Successful:       success,
			},
		},
		{
			"transaction with multiple operations",
			buildLedgerTransaction(tt, testTransaction{
				index:         1,
				envelopeXDR:   "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAyAAAAAAAAeJAAAAAAAAAAAAAAAACAAAAAAAAAAsAAAAAABLWhwAAAAAAAAALAAAAAAAS1ogAAAAAAAAAAA==",
				resultXDR:     "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
				feeChangesXDR: "AAAAAA==",
				metaXDR:       "AAAAAQAAAAAAAAAA",
				hash:          "6a3698a409141c6e45cb254aaaf94254c36a34323146a58214ce47b9f921174c",
			}),
			Transaction{
				TotalOrderID:     TotalOrderID{528280981504},
				TransactionHash:  "6a3698a409141c6e45cb254aaaf94254c36a34323146a58214ce47b9f921174c",
				LedgerSequence:   ledger.Sequence,
				LedgerCloseTime:  ledger.ClosedAt,
				ApplicationOrder: 1,
				Account:          "GAUJETIZVEP2NRYLUESJ3LS66NVCEGMON4UDCBCSBEVPIID773P2W6AY",
				AccountSequence:  "123456",
				MaxFee:           200,
				FeeCharged:       300,
				OperationCount:   2,
				TxEnvelope:       "AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAyAAAAAAAAeJAAAAAAAAAAAAAAAACAAAAAAAAAAsAAAAAABLWhwAAAAAAAAALAAAAAAAS1ogAAAAAAAAAAA==",
				TxResult:         "AAAAAAAAASwAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAFAAAAAAAAAAA=",
				TxFeeMeta:        "AAAAAA==",
				TxMeta:           "AAAAAQAAAAAAAAAA",
				SignatureString:  "",
				MemoType:         "none",
				Memo:             null.NewString("", false),
				ValidAfter:       null.NewInt(0, false),
				ValidBefore:      null.NewInt(0, false),
				Successful:       success,
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			tt.Assert.NoError(insertBuilder.Add(testCase.toInsert, sequence))
			tt.Assert.NoError(insertBuilder.Exec())

			var transactions []Transaction
			tt.Assert.NoError(q.Select(&transactions, selectExpTransaction))
			tt.Assert.Len(transactions, 1)

			transaction := transactions[0]

			// ignore created time and updated time
			transaction.CreatedAt = testCase.expected.CreatedAt
			transaction.UpdatedAt = testCase.expected.UpdatedAt

			// compare ClosedAt separately because reflect.DeepEqual does not handle time.Time
			expClosedAt := transaction.LedgerCloseTime
			transaction.LedgerCloseTime = testCase.expected.LedgerCloseTime

			tt.Assert.True(expClosedAt.Equal(testCase.expected.LedgerCloseTime))
			tt.Assert.Equal(transaction, testCase.expected)

			_, err = q.Exec(sq.Delete("exp_history_transactions"))
			tt.Assert.NoError(err)
		})
	}
}
