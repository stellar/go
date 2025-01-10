package ledger

import (
	"testing"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestLedger(t *testing.T) {
	ledger := Ledger{
		Ledger: ledgerTestInput(),
	}

	assert.Equal(t, uint32(30578981), ledger.Sequence())
	assert.Equal(t, int64(131335723340005376), ledger.ID())
	assert.Equal(t, "26932dc4d84b5fabe9ae744cb43ce4c6daccf98c86a991b2a14945b1adac4d59", ledger.Hash())
	assert.Equal(t, "f63c15d0eaf48afbd751a4c4dfade54a3448053c47c5a71d622668ae0cc2a208", ledger.PreviousHash())
	assert.Equal(t, int64(1594584547), ledger.CloseTime())
	assert.Equal(t, time.Time(time.Date(2020, time.July, 12, 20, 9, 7, 0, time.UTC)), ledger.ClosedAt())
	assert.Equal(t, int64(1054439020873472865), ledger.TotalCoins())
	assert.Equal(t, int64(18153766209161), ledger.FeePool())
	assert.Equal(t, uint32(100), ledger.BaseFee())
	assert.Equal(t, uint32(5000000), ledger.BaseReserve())
	assert.Equal(t, uint32(1000), ledger.MaxTxSetSize())
	assert.Equal(t, uint32(13), ledger.LedgerVersion())

	var ok bool
	var freeWrite int64
	freeWrite, ok = ledger.SorobanFeeWrite1Kb()
	assert.Equal(t, true, ok)
	assert.Equal(t, int64(12), freeWrite)

	var bucketSize uint64
	bucketSize, ok = ledger.TotalByteSizeOfBucketList()
	assert.Equal(t, true, ok)
	assert.Equal(t, uint64(56), bucketSize)

	var nodeID string
	nodeID, ok = ledger.NodeID()
	assert.Equal(t, true, ok)
	assert.Equal(t, "GARAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA76O", nodeID)

	var signature string
	signature, ok = ledger.Signature()
	assert.Equal(t, true, ok)
	assert.Equal(t, "9g==", signature)

	var success int32
	var failed int32
	success, failed, ok = ledger.TransactionCounts()
	assert.Equal(t, true, ok)
	assert.Equal(t, int32(1), success)
	assert.Equal(t, int32(1), failed)

	success, failed, ok = ledger.OperationCounts()
	assert.Equal(t, true, ok)
	assert.Equal(t, int32(1), success)
	assert.Equal(t, int32(13), failed)

}

func ledgerTestInput() (lcm xdr.LedgerCloseMeta) {
	lcm = xdr.LedgerCloseMeta{
		V: 1,
		V1: &xdr.LedgerCloseMetaV1{
			Ext: xdr.LedgerCloseMetaExt{
				V: 1,
				V1: &xdr.LedgerCloseMetaExtV1{
					SorobanFeeWrite1Kb: xdr.Int64(12),
				},
			},
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Hash: xdr.Hash{0x26, 0x93, 0x2d, 0xc4, 0xd8, 0x4b, 0x5f, 0xab, 0xe9, 0xae, 0x74, 0x4c, 0xb4, 0x3c, 0xe4, 0xc6, 0xda, 0xcc, 0xf9, 0x8c, 0x86, 0xa9, 0x91, 0xb2, 0xa1, 0x49, 0x45, 0xb1, 0xad, 0xac, 0x4d, 0x59},
				Header: xdr.LedgerHeader{
					LedgerSeq:          30578981,
					TotalCoins:         1054439020873472865,
					FeePool:            18153766209161,
					BaseFee:            100,
					BaseReserve:        5000000,
					MaxTxSetSize:       1000,
					LedgerVersion:      13,
					PreviousLedgerHash: xdr.Hash{0xf6, 0x3c, 0x15, 0xd0, 0xea, 0xf4, 0x8a, 0xfb, 0xd7, 0x51, 0xa4, 0xc4, 0xdf, 0xad, 0xe5, 0x4a, 0x34, 0x48, 0x5, 0x3c, 0x47, 0xc5, 0xa7, 0x1d, 0x62, 0x26, 0x68, 0xae, 0xc, 0xc2, 0xa2, 0x8},
					ScpValue: xdr.StellarValue{
						Ext: xdr.StellarValueExt{
							V: 1,
							LcValueSignature: &xdr.LedgerCloseValueSignature{
								NodeId: xdr.NodeId{
									Type:    0,
									Ed25519: &xdr.Uint256{34},
								},
								Signature: []byte{0xf6},
							},
						},
						CloseTime: 1594584547,
					},
				},
			},
			TotalByteSizeOfBucketList: xdr.Uint64(56),
			TxSet: xdr.GeneralizedTransactionSet{
				V: 0,
				V1TxSet: &xdr.TransactionSetV1{
					Phases: []xdr.TransactionPhase{
						{
							V: 0,
							V0Components: &[]xdr.TxSetComponent{
								{
									Type: 0,
									TxsMaybeDiscountedFee: &xdr.TxSetComponentTxsMaybeDiscountedFee{
										Txs: []xdr.TransactionEnvelope{
											createSampleTx(3),
											createSampleTx(10),
										},
									},
								},
							},
						},
					},
				},
			},
			TxProcessing: []xdr.TransactionResultMeta{
				{
					Result: xdr.TransactionResultPair{
						Result: xdr.TransactionResult{
							Result: xdr.TransactionResultResult{
								Code: xdr.TransactionResultCodeTxSuccess,
								Results: &[]xdr.OperationResult{
									{
										Code: xdr.OperationResultCodeOpInner,
										Tr: &xdr.OperationResultTr{
											Type: xdr.OperationTypeCreateAccount,
											CreateAccountResult: &xdr.CreateAccountResult{
												Code: 0,
											},
										},
									},
								},
							},
						},
					},
				},
				{
					Result: xdr.TransactionResultPair{
						Result: xdr.TransactionResult{
							Result: xdr.TransactionResultResult{
								Code: xdr.TransactionResultCodeTxFailed,
								Results: &[]xdr.OperationResult{
									{
										Code: xdr.OperationResultCodeOpInner,
										Tr: &xdr.OperationResultTr{
											Type: xdr.OperationTypeCreateAccount,
											CreateAccountResult: &xdr.CreateAccountResult{
												Code: 0,
											},
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

	return lcm
}

func createSampleTx(operationCount int) xdr.TransactionEnvelope {
	kp, err := keypair.Random()
	panicOnError(err)

	operations := []txnbuild.Operation{}
	operationType := &txnbuild.BumpSequence{
		BumpTo: 0,
	}
	for i := 0; i < operationCount; i++ {
		operations = append(operations, operationType)
	}

	sourceAccount := txnbuild.NewSimpleAccount(kp.Address(), int64(0))
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount: &sourceAccount,
			Operations:    operations,
			BaseFee:       txnbuild.MinBaseFee,
			Preconditions: txnbuild.Preconditions{TimeBounds: txnbuild.NewInfiniteTimeout()},
		},
	)
	panicOnError(err)

	env := tx.ToXDR()
	return env
}

// PanicOnError is a function that panics if the provided error is not nil
func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
