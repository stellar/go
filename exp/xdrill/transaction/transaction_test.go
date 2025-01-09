package transaction

import (
	"testing"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestTransaction(t *testing.T) {
	ledger := ledgerTestInput()
	transaction := transactionTestInput()

	assert.Equal(t, "1122330000000000000000000000000000000000000000000000000000000000", Hash(transaction))
	assert.Equal(t, uint32(1), Index(transaction))
	assert.Equal(t, int64(131335723340009472), ID(transaction, ledger))

	var err error
	var account string
	account, err = Account(transaction)
	assert.Equal(t, nil, err)
	assert.Equal(t, "GAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABCAK", account)

	assert.Equal(t, int64(30578981), AccountSequence(transaction))
	assert.Equal(t, uint32(4560), MaxFee(transaction))

	feeCharged := FeeCharged(transaction, ledger)
	assert.Equal(t, int64(789), *feeCharged)

	assert.Equal(t, uint32(3), OperationCount(transaction))
	assert.Equal(t, "test memo", Memo(transaction))
	assert.Equal(t, "MemoTypeMemoText", MemoType(transaction))

	var timeBounds *string
	timeBounds, err = TimeBounds(transaction)
	assert.Equal(t, nil, err)
	assert.Equal(t, "[1,10)", *timeBounds)

	ledgerBounds := LedgerBounds(transaction)
	assert.Equal(t, "[2,20)", *ledgerBounds)

	minSequence := MinSequence(transaction)
	assert.Equal(t, int64(123), *minSequence)

	minSequenceAge := MinSequenceAge(transaction)
	assert.Equal(t, int64(456), *minSequenceAge)

	minSequenceLedgerGap := MinSequenceLedgerGap(transaction)
	assert.Equal(t, int64(789), *minSequenceLedgerGap)

	sorobanResourceFee := SorobanResourceFee(transaction)
	assert.Equal(t, int64(1234), *sorobanResourceFee)

	sorobanResourcesInstructions := SorobanResourcesInstructions(transaction)
	assert.Equal(t, uint32(123), *sorobanResourcesInstructions)

	sorobanResourcesReadBytes := SorobanResourcesReadBytes(transaction)
	assert.Equal(t, uint32(456), *sorobanResourcesReadBytes)

	sorobanResourcesWriteBytes := SorobanResourcesWriteBytes(transaction)
	assert.Equal(t, uint32(789), *sorobanResourcesWriteBytes)

	inclusionFeeBid := InclusionFeeBid(transaction)
	assert.Equal(t, int64(3326), *inclusionFeeBid)

	sorobanInclusionFeeCharged := SorobanInclusionFeeCharged(transaction)
	assert.Equal(t, int64(-1234), *sorobanInclusionFeeCharged)

	sorobanResourceFeeRefund := SorobanResourceFeeRefund(transaction)
	assert.Equal(t, int64(0), *sorobanResourceFeeRefund)

	sorobanTotalNonRefundableResourceFeeCharged := SorobanTotalNonRefundableResourceFeeCharged(transaction)
	assert.Equal(t, int64(321), *sorobanTotalNonRefundableResourceFeeCharged)

	sorobanTotalRefundableResourceFeeCharged := SorobanTotalRefundableResourceFeeCharged(transaction)
	assert.Equal(t, int64(123), *sorobanTotalRefundableResourceFeeCharged)

	sorobanRentFeeCharged := SorobanRentFeeCharged(transaction)
	assert.Equal(t, int64(456), *sorobanRentFeeCharged)

	assert.Equal(t, "TransactionResultCodeTxSuccess", ResultCode(transaction))
	assert.Equal(t, []string{"GAISFR7R"}, Signers(transaction))

	accountMuxed := AccountMuxed(transaction)
	assert.Equal(t, "MAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPMJ2I", *accountMuxed)

	feeAccount := FeeAccount(transaction)
	assert.Equal(t, (*string)(nil), feeAccount)

	feeAccountMuxed := FeeAccountMuxed(transaction)
	assert.Equal(t, (*string)(nil), feeAccountMuxed)

	innerTransactionHash := InnerTransactionHash(transaction)
	assert.Equal(t, (*string)(nil), innerTransactionHash)

	assert.Equal(t, (*uint32)(nil), NewMaxFee(transaction))
	assert.Equal(t, true, Successful(transaction))
}

func ledgerTestInput() (lcm xdr.LedgerCloseMeta) {
	lcm = xdr.LedgerCloseMeta{
		V: 1,
		V1: &xdr.LedgerCloseMetaV1{
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Header: xdr.LedgerHeader{
					LedgerSeq:     30578981,
					LedgerVersion: 22,
				},
			},
		},
	}

	return lcm
}

func transactionTestInput() ingest.LedgerTransaction {
	ed25519 := xdr.Uint256([32]byte{0x11, 0x22, 0x33})
	muxedAccount := xdr.MuxedAccount{
		Type:    256,
		Ed25519: &ed25519,
		Med25519: &xdr.MuxedAccountMed25519{
			Id:      xdr.Uint64(123),
			Ed25519: ed25519,
		},
	}

	memoText := "test memo"
	minSeqNum := xdr.SequenceNumber(123)

	transaction := ingest.LedgerTransaction{
		Index: 1,
		Envelope: xdr.TransactionEnvelope{
			Type: xdr.EnvelopeTypeEnvelopeTypeTx,
			V1: &xdr.TransactionV1Envelope{
				Signatures: []xdr.DecoratedSignature{
					{
						Signature: []byte{0x11, 0x22},
					},
				},
				Tx: xdr.Transaction{
					SourceAccount: muxedAccount,
					SeqNum:        xdr.SequenceNumber(30578981),
					Fee:           xdr.Uint32(4560),
					Operations: []xdr.Operation{
						{
							SourceAccount: &muxedAccount,
							Body:          xdr.OperationBody{},
						},
						{
							SourceAccount: &muxedAccount,
							Body:          xdr.OperationBody{},
						},
						{
							SourceAccount: &muxedAccount,
							Body:          xdr.OperationBody{},
						},
					},
					Memo: xdr.Memo{
						Type: xdr.MemoTypeMemoText,
						Text: &memoText,
					},
					Cond: xdr.Preconditions{
						Type: 2,
						V2: &xdr.PreconditionsV2{
							TimeBounds: &xdr.TimeBounds{
								MinTime: xdr.TimePoint(1),
								MaxTime: xdr.TimePoint(10),
							},
							LedgerBounds: &xdr.LedgerBounds{
								MinLedger: 2,
								MaxLedger: 20,
							},
							MinSeqNum:       &minSeqNum,
							MinSeqAge:       456,
							MinSeqLedgerGap: 789,
						},
					},
					Ext: xdr.TransactionExt{
						V: 1,
						SorobanData: &xdr.SorobanTransactionData{
							Resources: xdr.SorobanResources{
								Instructions: 123,
								ReadBytes:    456,
								WriteBytes:   789,
							},
							ResourceFee: 1234,
						},
					},
				},
			},
		},
		Result: xdr.TransactionResultPair{
			TransactionHash: xdr.Hash{0x11, 0x22, 0x33},
			Result: xdr.TransactionResult{
				FeeCharged: xdr.Int64(789),
				Result: xdr.TransactionResultResult{
					Code: 0,
				},
			},
		},
		FeeChanges: xdr.LedgerEntryChanges{
			{
				Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
				State: &xdr.LedgerEntry{
					Data: xdr.LedgerEntryData{
						Type: xdr.LedgerEntryTypeAccount,
						Account: &xdr.AccountEntry{
							AccountId: xdr.AccountId{
								Type:    0,
								Ed25519: &ed25519,
							},
							Balance: 1000,
						},
					},
				},
			},
			{},
		},
		UnsafeMeta: xdr.TransactionMeta{
			V: 3,
			V3: &xdr.TransactionMetaV3{
				TxChangesAfter: xdr.LedgerEntryChanges{},
				SorobanMeta: &xdr.SorobanTransactionMeta{
					Ext: xdr.SorobanTransactionMetaExt{
						V: 1,
						V1: &xdr.SorobanTransactionMetaExtV1{
							TotalNonRefundableResourceFeeCharged: 321,
							TotalRefundableResourceFeeCharged:    123,
							RentFeeCharged:                       456,
						},
					},
				},
			},
		},
		LedgerVersion: 22,
		Ledger:        xdr.LedgerCloseMeta{},
		Hash:          xdr.Hash{},
	}

	return transaction
}
