package token_transfer

import (
	"github.com/stellar/go/ingest"
	addressProto "github.com/stellar/go/ingest/address"
	assetProto "github.com/stellar/go/ingest/asset"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

var (
	someTxAccount = "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3"
	someTxHash    = xdr.Hash{1, 1, 1, 1}

	accountA = "GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON"
	accountB = "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z"
	memoA    = uint64(123)
	memoB    = uint64(234)

	hundredUnitsInt64 = xdr.Int64(1000000000)
	hundredUnitsStr   = "100.0000000"

	usdc           = "USDC"
	usdcIssuer     = "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN"
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
					SourceAccount: xdr.MustMuxedAddress(someTxAccount),
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
)

// Helper functions for testing
func muxedAddrFromGaddr(gAddress string, memoId uint64) string {
	addr, _ := xdr.MuxedAccountFromAccountId(gAddress, memoId)
	mAddress := addr.Address()
	return mAddress
}

func muxedAccountFromGaddr(gAddress string, memoId uint64) xdr.MuxedAccount {
	res, _ := xdr.MuxedAccountFromAccountId(gAddress, memoId)
	return res
}

func muxedAccountPtrFromGaddr(gAddress string, memoId uint64) *xdr.MuxedAccount {
	res := muxedAccountFromGaddr(gAddress, memoId)
	return &res
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
				SourceAccount: xdr.MustMuxedAddressPtr(accountA),
				Body: xdr.OperationBody{
					Type: xdr.OperationTypeCreateAccount,
					CreateAccountOp: &xdr.CreateAccountOp{
						Destination:     xdr.MustAddress(accountB),
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
							From:   &addressProto.Address{AddressType: addressProto.AddressType_ADDRESS_TYPE_ACCOUNT, StrKey: accountA},
							To:     &addressProto.Address{AddressType: addressProto.AddressType_ADDRESS_TYPE_ACCOUNT, StrKey: accountB},
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
	tests := []testFixture{
		{
			name:    "successful account merge",
			tx:      someTx,
			opIndex: 0,
			op: xdr.Operation{
				SourceAccount: xdr.MustMuxedAddressPtr(accountA),
				Body: xdr.OperationBody{
					Type:        xdr.OperationTypeAccountMerge,
					Destination: xdr.MustMuxedAddressPtr(accountB),
				},
			},
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
				{
					Meta:  expectedEventMeta,
					Asset: assetProto.NewNativeAsset(),
					Event: &TokenTransferEvent_Transfer{
						Transfer: &Transfer{
							From:   &addressProto.Address{AddressType: addressProto.AddressType_ADDRESS_TYPE_ACCOUNT, StrKey: accountA},
							To:     &addressProto.Address{AddressType: addressProto.AddressType_ADDRESS_TYPE_ACCOUNT, StrKey: accountB},
							Amount: hundredUnitsStr,
						},
					},
				},
			},
		},
		{
			name:    "empty account merge - no events",
			tx:      someTx,
			opIndex: 0,
			op: xdr.Operation{
				SourceAccount: xdr.MustMuxedAddressPtr(accountA),
				Body: xdr.OperationBody{
					Type:        xdr.OperationTypeAccountMerge,
					Destination: xdr.MustMuxedAddressPtr(accountB),
				},
			},
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
	tests := []testFixture{
		{
			name:    "G account to G account - XLM transfer",
			tx:      someTx,
			opIndex: 0,
			op: xdr.Operation{
				SourceAccount: xdr.MustMuxedAddressPtr(accountA),
				Body: xdr.OperationBody{
					Type: xdr.OperationTypePayment,
					PaymentOp: &xdr.PaymentOp{
						Destination: xdr.MustMuxedAddress(accountB),
						Asset:       xdr.Asset{Type: xdr.AssetTypeAssetTypeNative},
						Amount:      hundredUnitsInt64,
					},
				},
			},
			expected: []*TokenTransferEvent{
				{
					Meta:  expectedEventMeta,
					Asset: assetProto.NewNativeAsset(),
					Event: &TokenTransferEvent_Transfer{
						Transfer: &Transfer{
							From:   &addressProto.Address{AddressType: addressProto.AddressType_ADDRESS_TYPE_ACCOUNT, StrKey: accountA},
							To:     &addressProto.Address{AddressType: addressProto.AddressType_ADDRESS_TYPE_ACCOUNT, StrKey: accountB},
							Amount: hundredUnitsStr,
						},
					},
				},
			},
		},
		{
			name:    "G account to G account - USDC tranfer",
			tx:      someTx,
			opIndex: 0,
			op: xdr.Operation{
				SourceAccount: xdr.MustMuxedAddressPtr(accountA),
				Body: xdr.OperationBody{
					Type: xdr.OperationTypePayment,
					PaymentOp: &xdr.PaymentOp{
						Destination: xdr.MustMuxedAddress(accountB),
						Asset:       usdcAsset,
						Amount:      hundredUnitsInt64,
					},
				},
			},
			expected: []*TokenTransferEvent{
				{
					Meta:  expectedEventMeta,
					Asset: usdcProtoAsset,
					Event: &TokenTransferEvent_Transfer{
						Transfer: &Transfer{
							From:   &addressProto.Address{AddressType: addressProto.AddressType_ADDRESS_TYPE_ACCOUNT, StrKey: accountA},
							To:     &addressProto.Address{AddressType: addressProto.AddressType_ADDRESS_TYPE_ACCOUNT, StrKey: accountB},
							Amount: hundredUnitsStr,
						},
					},
				},
			},
		},
		{
			name:    "G account to M Account - USDC transfer",
			tx:      someTx,
			opIndex: 0,
			op: xdr.Operation{
				SourceAccount: xdr.MustMuxedAddressPtr(accountA),
				Body: xdr.OperationBody{
					Type: xdr.OperationTypePayment,
					PaymentOp: &xdr.PaymentOp{
						Destination: muxedAccountFromGaddr(accountB, memoB),
						Asset:       usdcAsset,
						Amount:      hundredUnitsInt64,
					},
				},
			},
			expected: []*TokenTransferEvent{
				{
					Meta:  expectedEventMeta,
					Asset: usdcProtoAsset,
					Event: &TokenTransferEvent_Transfer{
						Transfer: &Transfer{
							From:   &addressProto.Address{AddressType: addressProto.AddressType_ADDRESS_TYPE_ACCOUNT, StrKey: accountA},
							To:     &addressProto.Address{AddressType: addressProto.AddressType_ADDRESS_TYPE_MUXED_ACCOUNT, StrKey: muxedAddrFromGaddr(accountB, memoB)}, // M-address format
							Amount: hundredUnitsStr,
						},
					},
				},
			},
		},
		{
			name:    "M account to G Account - USDC transfer",
			tx:      someTx,
			opIndex: 0,
			op: xdr.Operation{
				SourceAccount: muxedAccountPtrFromGaddr(accountA, memoA),
				Body: xdr.OperationBody{
					Type: xdr.OperationTypePayment,
					PaymentOp: &xdr.PaymentOp{
						Destination: xdr.MustMuxedAddress(accountB),
						Asset:       usdcAsset,
						Amount:      hundredUnitsInt64,
					},
				},
			},
			expected: []*TokenTransferEvent{
				{
					Meta:  expectedEventMeta,
					Asset: usdcProtoAsset,
					Event: &TokenTransferEvent_Transfer{
						Transfer: &Transfer{
							From:   &addressProto.Address{AddressType: addressProto.AddressType_ADDRESS_TYPE_MUXED_ACCOUNT, StrKey: muxedAddrFromGaddr(accountA, memoA)},
							To:     &addressProto.Address{AddressType: addressProto.AddressType_ADDRESS_TYPE_ACCOUNT, StrKey: accountB},
							Amount: hundredUnitsStr,
						},
					},
				},
			},
		},
		{
			name:    "G (issuer account) to G account - USDC mint",
			tx:      someTx,
			opIndex: 0,
			op: xdr.Operation{
				SourceAccount: xdr.MustMuxedAddressPtr(usdcIssuer),
				Body: xdr.OperationBody{
					Type: xdr.OperationTypePayment,
					PaymentOp: &xdr.PaymentOp{
						Destination: xdr.MustMuxedAddress(accountB),
						Asset:       usdcAsset,
						Amount:      hundredUnitsInt64,
					},
				},
			},
			expected: []*TokenTransferEvent{
				{
					Meta:  expectedEventMeta,
					Asset: usdcProtoAsset,
					Event: &TokenTransferEvent_Mint{
						Mint: &Mint{
							To:     &addressProto.Address{AddressType: addressProto.AddressType_ADDRESS_TYPE_ACCOUNT, StrKey: accountB},
							Amount: hundredUnitsStr,
						},
					},
				},
			},
		},
		{
			name:    "G (issuer account) to M account - USDC mint",
			tx:      someTx,
			opIndex: 0,
			op: xdr.Operation{
				SourceAccount: xdr.MustMuxedAddressPtr(usdcIssuer),
				Body: xdr.OperationBody{
					Type: xdr.OperationTypePayment,
					PaymentOp: &xdr.PaymentOp{
						Destination: muxedAccountFromGaddr(accountB, memoB),
						Asset:       usdcAsset,
						Amount:      hundredUnitsInt64,
					},
				},
			},
			expected: []*TokenTransferEvent{
				{
					Meta:  expectedEventMeta,
					Asset: usdcProtoAsset,
					Event: &TokenTransferEvent_Mint{
						Mint: &Mint{
							To:     &addressProto.Address{AddressType: addressProto.AddressType_ADDRESS_TYPE_MUXED_ACCOUNT, StrKey: muxedAddrFromGaddr(accountB, memoB)},
							Amount: hundredUnitsStr,
						},
					},
				},
			},
		},
		{
			name:    "G account to G (issuer account) - USDC burn",
			tx:      someTx,
			opIndex: 0,
			op: xdr.Operation{
				SourceAccount: xdr.MustMuxedAddressPtr(accountA),
				Body: xdr.OperationBody{
					Type: xdr.OperationTypePayment,
					PaymentOp: &xdr.PaymentOp{
						Destination: xdr.MustMuxedAddress(usdcIssuer),
						Asset:       usdcAsset,
						Amount:      hundredUnitsInt64,
					},
				},
			},
			expected: []*TokenTransferEvent{
				{
					Meta:  expectedEventMeta,
					Asset: usdcProtoAsset,
					Event: &TokenTransferEvent_Burn{
						Burn: &Burn{
							From:   &addressProto.Address{AddressType: addressProto.AddressType_ADDRESS_TYPE_ACCOUNT, StrKey: accountA}, // M-address format
							Amount: hundredUnitsStr,
						},
					},
				},
			},
		},
		{
			name:    "M account to G (issuer account) - USDC burn",
			tx:      someTx,
			opIndex: 0,
			op: xdr.Operation{
				SourceAccount: muxedAccountPtrFromGaddr(accountA, memoA),
				Body: xdr.OperationBody{
					Type: xdr.OperationTypePayment,
					PaymentOp: &xdr.PaymentOp{
						Destination: xdr.MustMuxedAddress(usdcIssuer),
						Asset:       usdcAsset,
						Amount:      hundredUnitsInt64,
					},
				},
			},
			expected: []*TokenTransferEvent{
				{
					Meta:  expectedEventMeta,
					Asset: usdcProtoAsset,
					Event: &TokenTransferEvent_Burn{
						Burn: &Burn{
							From:   &addressProto.Address{AddressType: addressProto.AddressType_ADDRESS_TYPE_MUXED_ACCOUNT, StrKey: muxedAddrFromGaddr(accountA, memoA)},
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
