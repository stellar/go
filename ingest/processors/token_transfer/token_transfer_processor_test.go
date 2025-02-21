package token_transfer

import (
	"github.com/stellar/go/ingest"
	addressProto "github.com/stellar/go/ingest/address"
	assetProto "github.com/stellar/go/ingest/asset"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

var (
	someTxAccount = xdr.MustMuxedAddress(keypair.MustRandom().Address())
	someTxHash    = xdr.Hash{1, 1, 1, 1}

	gAddressAccountA = "GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON"
	gAddressAccountB = "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z"
	accountA         = xdr.MustMuxedAddress(gAddressAccountA)
	accountB         = xdr.MustMuxedAddress(gAddressAccountB)

	memoA            = uint64(123)
	memoB            = uint64(234)
	muxedAccountA    = muxedAccountFromGaddr(gAddressAccountA, memoA)
	muxedAccountB    = muxedAccountFromGaddr(gAddressAccountB, memoB)
	mAddressAccountA = muxedAccountA.Address()
	mAddressAccountB = muxedAccountB.Address()

	hundredUnitsInt64 = xdr.Int64(1000000000)
	hundredUnitsStr   = "100.0000000"

	usdc           = "USDC"
	usdcIssuer     = "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN"
	usdcAccount    = xdr.MustMuxedAddress(usdcIssuer)
	usdcAsset      = xdr.MustNewCreditAsset(usdc, usdcIssuer)
	usdcProtoAsset = assetProto.NewIssuedAsset(usdc, usdcIssuer)

	someLcm = xdr.LedgerCloseMeta{
		V: int32(0),
		V0: &xdr.LedgerCloseMetaV0{
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Header: xdr.LedgerHeader{
					LedgerVersion: 20,
					LedgerSeq:     xdr.Uint32(12345),
					ScpValue:      xdr.StellarValue{CloseTime: xdr.TimePoint(12345 * 100)},
				},
			},
			TxSet:              xdr.TransactionSet{},
			TxProcessing:       nil,
			UpgradesProcessing: nil,
			ScpInfo:            nil,
		},
		V1: nil,
	}

	someTx = ingest.LedgerTransaction{
		Index: 1,
		Envelope: xdr.TransactionEnvelope{
			Type: xdr.EnvelopeTypeEnvelopeTypeTx,
			V1: &xdr.TransactionV1Envelope{
				Tx: xdr.Transaction{
					SourceAccount: someTxAccount,
				},
			},
		},
		Result:        xdr.TransactionResultPair{},
		UnsafeMeta:    xdr.TransactionMeta{},
		LedgerVersion: 1234,
		Ledger:        someLcm,
		Hash:          someTxHash,
	}

	someOperationIndex = uint32(0)
	expectedEventMeta  = NewEventMeta(someTx, &someOperationIndex, nil)

	protoAddress = func(addr string) *addressProto.Address {
		ret := &addressProto.Address{StrKey: addr}
		if strings.HasPrefix(addr, "G") {
			ret.AddressType = addressProto.AddressType_ADDRESS_TYPE_ACCOUNT
		} else {
			ret.AddressType = addressProto.AddressType_ADDRESS_TYPE_MUXED_ACCOUNT
		}
		return ret
	}

	mintEvent = func(to *addressProto.Address, amt string, asset *assetProto.Asset) *TokenTransferEvent {
		ret := &TokenTransferEvent{
			Meta: expectedEventMeta,
			Event: &TokenTransferEvent_Mint{
				Mint: &Mint{
					To:     to,
					Amount: amt,
				},
			},
		}
		if asset != nil {
			ret.Asset = asset
		} else {
			ret.Asset = assetProto.NewNativeAsset()
		}
		return ret
	}

	burnEvent = func(from *addressProto.Address, amt string, asset *assetProto.Asset) *TokenTransferEvent {
		ret := &TokenTransferEvent{
			Meta: expectedEventMeta,
			Event: &TokenTransferEvent_Burn{
				Burn: &Burn{
					From:   from,
					Amount: amt,
				},
			},
		}
		if asset != nil {
			ret.Asset = asset
		} else {
			ret.Asset = assetProto.NewNativeAsset()
		}
		return ret
	}

	transferEvent = func(from *addressProto.Address, to *addressProto.Address, amt string, asset *assetProto.Asset) *TokenTransferEvent {
		ret := &TokenTransferEvent{
			Meta: expectedEventMeta,
			Event: &TokenTransferEvent_Transfer{
				Transfer: &Transfer{
					From:   from,
					To:     to,
					Amount: amt,
				},
			},
		}
		if asset != nil {
			ret.Asset = asset
		} else {
			ret.Asset = assetProto.NewNativeAsset()
		}
		return ret
	}
)

// Helper functions for testing
func muxedAccountFromGaddr(gAddress string, memoId uint64) xdr.MuxedAccount {
	res, _ := xdr.MuxedAccountFromAccountId(gAddress, memoId)
	return res
}

func buildLedgerTransaction(sourceAccount string) ingest.LedgerTransaction {
	return ingest.LedgerTransaction{
		Index: 1,
		Envelope: xdr.TransactionEnvelope{
			Type: xdr.EnvelopeTypeEnvelopeTypeTx,
			V1: &xdr.TransactionV1Envelope{
				Tx: xdr.Transaction{
					SourceAccount: xdr.MustMuxedAddress(sourceAccount),
				},
			},
		},
		Result:        xdr.TransactionResultPair{},
		UnsafeMeta:    xdr.TransactionMeta{},
		LedgerVersion: 1234,
		Ledger:        someLcm,
		Hash:          someTxHash,
	}
}

type testFixture struct {
	name     string
	tx       ingest.LedgerTransaction
	opIndex  uint32
	op       xdr.Operation
	opResult xdr.OperationResult
	expected []*TokenTransferEvent
	wantErr  bool
}

func TestAccountCreateEvents(t *testing.T) {
	tests := []testFixture{
		{
			name:    "successful account creation",
			tx:      someTx,
			opIndex: 0,
			op: xdr.Operation{
				SourceAccount: &accountA,
				Body: xdr.OperationBody{
					Type: xdr.OperationTypeCreateAccount,
					CreateAccountOp: &xdr.CreateAccountOp{
						Destination:     accountB.ToAccountId(),
						StartingBalance: hundredUnitsInt64,
					},
				},
			},
			opResult: xdr.OperationResult{},
			expected: []*TokenTransferEvent{
				{
					Meta:  expectedEventMeta,
					Asset: assetProto.NewNativeAsset(),

					Event: &TokenTransferEvent_Transfer{
						Transfer: &Transfer{
							From:   &addressProto.Address{AddressType: addressProto.AddressType_ADDRESS_TYPE_ACCOUNT, StrKey: gAddressAccountA},
							To:     &addressProto.Address{AddressType: addressProto.AddressType_ADDRESS_TYPE_ACCOUNT, StrKey: gAddressAccountB},
							Amount: hundredUnitsStr,
						},
					},
				},
			},
		},
	}

	for _, fixture := range tests {
		t.Run(fixture.name, func(t *testing.T) {
			events, err := ProcessTokenTransferEventsFromOperation(fixture.tx, fixture.opIndex, fixture.op, fixture.opResult)
			if fixture.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, fixture.expected, events)
		})
	}
}

func TestMergeAccountEvents(t *testing.T) {
	mergeAccountOp :=
		xdr.Operation{
			SourceAccount: &accountA,
			Body: xdr.OperationBody{
				Type:        xdr.OperationTypeAccountMerge,
				Destination: &accountB,
			},
		}

	tests := []testFixture{
		{
			name:    "successful account merge",
			tx:      someTx,
			opIndex: 0,
			op:      mergeAccountOp,
			opResult: xdr.OperationResult{
				Code: xdr.OperationResultCodeOpInner,
				Tr: &xdr.OperationResultTr{
					Type: xdr.OperationTypeAccountMerge,
					AccountMergeResult: &xdr.AccountMergeResult{
						Code:                 xdr.AccountMergeResultCodeAccountMergeSuccess,
						SourceAccountBalance: &hundredUnitsInt64,
					},
				},
			},
			expected: []*TokenTransferEvent{
				transferEvent(protoAddress(gAddressAccountA), protoAddress(gAddressAccountB), hundredUnitsStr, nil),
			},
		},
		{
			name:    "empty account merge - no events",
			tx:      someTx,
			opIndex: 0,
			op:      mergeAccountOp,
			opResult: xdr.OperationResult{
				Code: xdr.OperationResultCodeOpInner,
				Tr: &xdr.OperationResultTr{
					Type: xdr.OperationTypeAccountMerge,
					AccountMergeResult: &xdr.AccountMergeResult{
						Code:                 xdr.AccountMergeResultCodeAccountMergeSuccess,
						SourceAccountBalance: nil,
					},
				},
			},
			expected: nil,
		},
	}

	for _, fixture := range tests {
		t.Run(fixture.name, func(t *testing.T) {
			events, err := ProcessTokenTransferEventsFromOperation(fixture.tx, fixture.opIndex, fixture.op, fixture.opResult)
			if fixture.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, fixture.expected, events)
		})
	}
}

func TestPaymentEvents(t *testing.T) {

	paymentOp := func(src *xdr.MuxedAccount, dst xdr.MuxedAccount, asset *xdr.Asset) xdr.Operation {
		op := xdr.Operation{
			SourceAccount: src,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypePayment,
				PaymentOp: &xdr.PaymentOp{
					Destination: dst,
					Amount:      hundredUnitsInt64,
				},
			},
		}
		if asset != nil {
			op.Body.PaymentOp.Asset = *asset
		} else {
			op.Body.PaymentOp.Asset = xdr.Asset{Type: xdr.AssetTypeAssetTypeNative}
		}
		return op
	}

	tests := []testFixture{
		{
			name:    "G account to G account - XLM transfer",
			tx:      someTx,
			opIndex: 0,
			op:      paymentOp(&accountA, accountB, nil),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddress(gAddressAccountA), protoAddress(gAddressAccountB), hundredUnitsStr, nil),
			},
		},
		{
			name:    "G account to G account - USDC transfer",
			tx:      someTx,
			opIndex: 0,
			op:      paymentOp(&accountA, accountB, &usdcAsset),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddress(gAddressAccountA), protoAddress(gAddressAccountB), hundredUnitsStr, usdcProtoAsset),
			},
		},
		{
			name:    "G account to M Account - USDC transfer",
			tx:      someTx,
			opIndex: 0,
			op:      paymentOp(&accountA, muxedAccountB, &usdcAsset),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddress(gAddressAccountA), protoAddress(mAddressAccountB), hundredUnitsStr, usdcProtoAsset),
			},
		},
		{
			name:    "M account to G Account - USDC transfer",
			tx:      someTx,
			opIndex: 0,
			op:      paymentOp(&muxedAccountA, accountB, &usdcAsset),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddress(mAddressAccountA), protoAddress(gAddressAccountB), hundredUnitsStr, usdcProtoAsset),
			},
		},
		{
			name:    "G (issuer account) to G account - USDC mint",
			tx:      someTx,
			opIndex: 0,
			op:      paymentOp(&usdcAccount, accountB, &usdcAsset),
			expected: []*TokenTransferEvent{
				mintEvent(protoAddress(gAddressAccountB), hundredUnitsStr, usdcProtoAsset),
			},
		},
		{
			name:    "G (issuer account) to M account - USDC mint",
			tx:      someTx,
			opIndex: 0,
			op:      paymentOp(&usdcAccount, muxedAccountB, &usdcAsset),
			expected: []*TokenTransferEvent{
				mintEvent(protoAddress(mAddressAccountB), hundredUnitsStr, usdcProtoAsset),
			},
		},
		{
			name:    "G account to G (issuer account) - USDC burn",
			tx:      someTx,
			opIndex: 0,
			op:      paymentOp(&accountA, usdcAccount, &usdcAsset),
			expected: []*TokenTransferEvent{
				burnEvent(protoAddress(gAddressAccountA), hundredUnitsStr, usdcProtoAsset),
			},
		},
		{
			name:    "M account to G (issuer account) - USDC burn",
			tx:      someTx,
			opIndex: 0,
			op:      paymentOp(&muxedAccountA, usdcAccount, &usdcAsset),
			expected: []*TokenTransferEvent{
				burnEvent(protoAddress(mAddressAccountA), hundredUnitsStr, usdcProtoAsset),
			},
		},
	}

	for _, fixture := range tests {
		t.Run(fixture.name, func(t *testing.T) {
			events, err := ProcessTokenTransferEventsFromOperation(fixture.tx, fixture.opIndex, fixture.op, fixture.opResult)
			if fixture.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, fixture.expected, events)
		})
	}
}
