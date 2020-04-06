package history

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/network"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

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

type testTransaction struct {
	index         uint32
	envelopeXDR   string
	resultXDR     string
	feeChangesXDR string
	metaXDR       string
	hash          string
}

func buildLedgerTransaction(t *testing.T, tx testTransaction) io.LedgerTransaction {
	transaction := io.LedgerTransaction{
		Index:      tx.index,
		Envelope:   xdr.TransactionEnvelope{},
		Result:     xdr.TransactionResultPair{},
		FeeChanges: xdr.LedgerEntryChanges{},
		Meta:       xdr.TransactionMeta{},
	}

	tt := assert.New(t)

	err := xdr.SafeUnmarshalBase64(tx.envelopeXDR, &transaction.Envelope)
	tt.NoError(err)
	err = xdr.SafeUnmarshalBase64(tx.resultXDR, &transaction.Result.Result)
	tt.NoError(err)
	err = xdr.SafeUnmarshalBase64(tx.metaXDR, &transaction.Meta)
	tt.NoError(err)
	err = xdr.SafeUnmarshalBase64(tx.feeChangesXDR, &transaction.FeeChanges)
	tt.NoError(err)

	_, err = hex.Decode(transaction.Result.TransactionHash[:], []byte(tx.hash))
	tt.NoError(err)

	return transaction
}

// FeeBumpFixture contains the data inserted into the database
// when running FeeBumpScenario
type FeeBumpFixture struct {
	Ledger      Ledger
	Envelope    xdr.TransactionEnvelope
	Transaction Transaction
	OuterHash   string
	InnerHash   string
}

// FeeBumpScenario creates a ledger containing a fee bump transaction,
// an operation, and an effect
func FeeBumpScenario(tt *test.T, q *Q, successful bool) FeeBumpFixture {
	fixture := FeeBumpFixture{}
	sequence := uint32(123)
	fixture.Ledger = Ledger{
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
	*fixture.Ledger.SuccessfulTransactionCount = 1
	*fixture.Ledger.FailedTransactionCount = 0
	_, err := q.Exec(sq.Insert("history_ledgers").SetMap(ledgerToMap(fixture.Ledger)))
	tt.Assert.NoError(err)

	fixture.Envelope = xdr.TransactionEnvelope{
		Type: xdr.EnvelopeTypeEnvelopeTypeTxFeeBump,
		FeeBump: &xdr.FeeBumpTransactionEnvelope{
			Tx: xdr.FeeBumpTransaction{
				FeeSource: xdr.AccountId{
					Type:    xdr.PublicKeyTypePublicKeyTypeEd25519,
					Ed25519: &xdr.Uint256{2, 2, 2},
				},
				Fee: 776,
				InnerTx: xdr.FeeBumpTransactionInnerTx{
					Type: xdr.EnvelopeTypeEnvelopeTypeTx,
					V1: &xdr.TransactionV1Envelope{
						Tx: xdr.Transaction{
							SourceAccount: xdr.AccountId{
								Type: xdr.PublicKeyTypePublicKeyTypeEd25519,
								Ed25519: &xdr.Uint256{
									3, 3, 3,
								},
							},
							Fee: 99,
							Memo: xdr.Memo{
								Type: xdr.MemoTypeMemoNone,
							},
							SeqNum: 97,
							TimeBounds: &xdr.TimeBounds{
								MinTime: 2,
								MaxTime: 4,
							},
							Operations: []xdr.Operation{
								xdr.Operation{
									Body: xdr.OperationBody{
										Type: xdr.OperationTypeBumpSequence,
										BumpSequenceOp: &xdr.BumpSequenceOp{
											BumpTo: 98,
										},
									},
								},
							},
						},
						Signatures: []xdr.DecoratedSignature{
							xdr.DecoratedSignature{
								Hint:      xdr.SignatureHint{2, 2, 2, 2},
								Signature: xdr.Signature{20, 20, 20},
							},
						},
					},
				},
			},
			Signatures: []xdr.DecoratedSignature{
				xdr.DecoratedSignature{
					Hint:      xdr.SignatureHint{3, 3, 3, 3},
					Signature: xdr.Signature{30, 30, 30},
				},
			},
		},
	}
	envelopeXDR, err := xdr.MarshalBase64(fixture.Envelope)
	tt.Assert.NoError(err)

	innerHash, err := network.HashTransaction(
		&fixture.Envelope.FeeBump.Tx.InnerTx.V1.Tx,
		"Test SDF Network ; September 2015",
	)
	tt.Assert.NoError(err)
	fixture.InnerHash = hex.EncodeToString(innerHash[:])

	outerHash, err := network.HashFeeBumpTransaction(
		&fixture.Envelope.FeeBump.Tx,
		"Test SDF Network ; September 2015",
	)
	tt.Assert.NoError(err)
	fixture.OuterHash = hex.EncodeToString(outerHash[:])

	tt.Assert.NotEqual(fixture.InnerHash, fixture.OuterHash)

	resultPair := xdr.TransactionResultPair{
		TransactionHash: xdr.Hash(outerHash),
		Result: xdr.TransactionResult{
			FeeCharged: 123,
			Result: xdr.TransactionResultResult{
				Code: xdr.TransactionResultCodeTxFeeBumpInnerSuccess,
				InnerResultPair: &xdr.InnerTransactionResultPair{
					TransactionHash: xdr.Hash(innerHash),
					Result: xdr.InnerTransactionResult{
						Result: xdr.InnerTransactionResultResult{
							Code: xdr.TransactionResultCodeTxSuccess,
							Results: &[]xdr.OperationResult{
								xdr.OperationResult{
									Tr: &xdr.OperationResultTr{
										Type: xdr.OperationTypeBumpSequence,
										BumpSeqResult: &xdr.BumpSequenceResult{
											Code: xdr.BumpSequenceResultCodeBumpSequenceSuccess,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if !successful {
		resultPair.Result.Result.Code = xdr.TransactionResultCodeTxFeeBumpInnerFailed
		resultPair.Result.Result.InnerResultPair.Result.Result.Code = xdr.TransactionResultCodeTxBadAuth
	}

	resultXDR, err := xdr.MarshalBase64(resultPair.Result)
	tt.Assert.NoError(err)

	feeBumpTransaction := buildLedgerTransaction(tt.T, testTransaction{
		index:         1,
		envelopeXDR:   envelopeXDR,
		resultXDR:     resultXDR,
		feeChangesXDR: "AAAAAA==",
		metaXDR:       "AAAAAQAAAAAAAAAA",
		hash:          fixture.OuterHash,
	})

	insertBuilder := q.NewTransactionBatchInsertBuilder(0)
	tt.Assert.NoError(insertBuilder.Add(feeBumpTransaction, sequence))
	tt.Assert.NoError(insertBuilder.Exec())

	account := fixture.Envelope.SourceAccount()
	feeBumpAccount := fixture.Envelope.FeeBumpAccount()

	opBuilder := q.NewOperationBatchInsertBuilder(1)
	details, err := json.Marshal(map[string]string{
		"bump_to": "98",
	})
	tt.Assert.NoError(err)

	tt.Assert.NoError(opBuilder.Add(
		toid.New(fixture.Ledger.Sequence, 1, 1).ToInt64(),
		toid.New(fixture.Ledger.Sequence, 1, 0).ToInt64(),
		1,
		xdr.OperationTypeBumpSequence,
		details,
		account.Address(),
	))
	tt.Assert.NoError(opBuilder.Exec())

	effectBuilder := q.NewEffectBatchInsertBuilder(2)
	details, err = json.Marshal(map[string]interface{}{"new_seq": 98})
	tt.Assert.NoError(err)

	accounIDs, err := q.CreateAccounts([]string{account.Address()}, 1)
	tt.Assert.NoError(err)

	err = effectBuilder.Add(
		accounIDs[account.Address()],
		toid.New(fixture.Ledger.Sequence, 1, 1).ToInt64(),
		1,
		EffectSequenceBumped,
		details,
	)
	tt.Assert.NoError(err)
	tt.Assert.NoError(effectBuilder.Exec())

	fixture.Transaction = Transaction{
		TotalOrderID:     TotalOrderID{528280981504},
		TransactionHash:  fixture.OuterHash,
		LedgerSequence:   fixture.Ledger.Sequence,
		ApplicationOrder: 1,
		Account:          account.Address(),
		AccountSequence:  "97",
		MaxFee:           int64(fixture.Envelope.Fee()),
		FeeCharged:       int64(resultPair.Result.FeeCharged),
		OperationCount:   1,
		TxEnvelope:       envelopeXDR,
		TxResult:         resultXDR,
		TxFeeMeta:        "AAAAAA==",
		TxMeta:           "AAAAAQAAAAAAAAAA",
		MemoType:         "none",
		Memo:             null.NewString("", false),
		ValidAfter:       null.IntFrom(2),
		ValidBefore:      null.IntFrom(4),
		Successful:       successful,
		SignatureString: strings.Join(
			signatures(fixture.Envelope.FeeBumpSignatures()), ",",
		),
		InnerSignatureString: null.StringFrom(strings.Join(
			signatures(fixture.Envelope.Signatures()), ",",
		)),
		NewMaxFee:            null.IntFrom(int64(fixture.Envelope.FeeBumpFee())),
		InnerTransactionHash: null.StringFrom(fixture.InnerHash),
		FeeAccount:           null.StringFrom(feeBumpAccount.Address()),
	}

	results, err := q.TransactionsByIDs(fixture.Transaction.ID)
	tt.Assert.NoError(err)
	dbTx := results[fixture.Transaction.ID]
	fixture.Transaction.CreatedAt = dbTx.CreatedAt
	fixture.Transaction.UpdatedAt = dbTx.UpdatedAt
	fixture.Transaction.LedgerCloseTime = dbTx.LedgerCloseTime

	return fixture
}
