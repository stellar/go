package token_transfer

import (
	"github.com/stellar/go/ingest"
	addressProto "github.com/stellar/go/ingest/address"
	assetProto "github.com/stellar/go/ingest/asset"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"testing"
)

var (
	someTxAccount = xdr.MustMuxedAddress("GBF3XFXGBGNQDN3HOSZ7NVRF6TJ2JOD5U6ELIWJOOEI6T5WKMQT2YSXQ")
	someTxHash    = xdr.Hash{1, 1, 1, 1}

	accountA = xdr.MustMuxedAddress("GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON")
	accountB = xdr.MustMuxedAddress("GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z")

	memoA = uint64(123)
	memoB = uint64(234)

	muxedAccountA, _ = xdr.MuxedAccountFromAccountId(accountA.Address(), memoA)
	muxedAccountB, _ = xdr.MuxedAccountFromAccountId(accountB.Address(), memoB)

	tenMillion = int64(1e7)

	oneUnit    = xdr.Int64(1 * tenMillion)
	twoUnits   = xdr.Int64(2 * tenMillion)
	threeUnits = xdr.Int64(3 * tenMillion)
	// fourUnits := xdr.Int64(4 * tenMillion)
	fiveUnits = xdr.Int64(5 * tenMillion)
	sixUnits  = xdr.Int64(6 * tenMillion)
	tenUnits  = xdr.Int64(10 * tenMillion)

	hundredUnits    = xdr.Int64(100 * tenMillion)
	hundredUnitsStr = "100.0000000"

	xlmAsset = xdr.MustNewNativeAsset()

	usdcIssuer     = "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN"
	usdcAccount    = xdr.MustMuxedAddress(usdcIssuer)
	usdcAsset      = xdr.MustNewCreditAsset("USDC", usdcIssuer)
	usdcProtoAsset = assetProto.NewIssuedAsset(usdcAsset)

	ethIssuer     = "GCEODJVUUVYVFD5KT4TOEDTMXQ76OPFOQC2EMYYMLPXQCUVPOB6XRWPQ"
	ethAccount    = xdr.MustMuxedAddress(ethIssuer)
	ethAsset      = xdr.MustNewCreditAsset("ETH", ethIssuer)
	ethProtoAsset = assetProto.NewIssuedAsset(ethAsset)

	btcIsuer      = "GBT4YAEGJQ5YSFUMNKX6BPBUOCPNAIOFAVZOF6MIME2CECBMEIUXFZZN"
	btcAccount    = xdr.MustMuxedAddress(btcIsuer)
	btcAsset      = xdr.MustNewCreditAsset("BTC", btcIsuer)
	btcProtoAsset = assetProto.NewIssuedAsset(btcAsset)

	lpBtcEthId, _  = xdr.NewPoolId(btcAsset, ethAsset, xdr.LiquidityPoolFeeV18)
	lpEthUsdcId, _ = xdr.NewPoolId(ethAsset, usdcAsset, xdr.LiquidityPoolFeeV18)

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

	// Some global anonymous functions.
	mintEvent = func(to *addressProto.Address, amt string, asset *assetProto.Asset) *TokenTransferEvent {
		return &TokenTransferEvent{
			Meta:  expectedEventMeta,
			Asset: asset,
			Event: &TokenTransferEvent_Mint{
				Mint: &Mint{
					To:     to,
					Amount: amt,
				},
			},
		}

	}

	burnEvent = func(from *addressProto.Address, amt string, asset *assetProto.Asset) *TokenTransferEvent {
		return &TokenTransferEvent{
			Meta:  expectedEventMeta,
			Asset: asset,
			Event: &TokenTransferEvent_Burn{
				Burn: &Burn{
					From:   from,
					Amount: amt,
				},
			},
		}

	}

	transferEvent = func(from *addressProto.Address, to *addressProto.Address, amt string, asset *assetProto.Asset) *TokenTransferEvent {
		return &TokenTransferEvent{
			Meta:  expectedEventMeta,
			Asset: asset,
			Event: &TokenTransferEvent_Transfer{
				Transfer: &Transfer{
					From:   from,
					To:     to,
					Amount: amt,
				},
			},
		}
	}
)

type testFixture struct {
	name     string
	tx       ingest.LedgerTransaction
	opIndex  uint32
	op       xdr.Operation
	opResult xdr.OperationResult
	expected []*TokenTransferEvent
	wantErr  bool
}

// RunTokenTransferEventTests runs a standard set of tests for token transfer event processing
func runTokenTransferEventTests(t *testing.T, tests []testFixture) {
	for _, fixture := range tests {
		t.Run(fixture.name, func(t *testing.T) {
			events, err := ProcessTokenTransferEventsFromOperation(
				fixture.tx,
				fixture.opIndex,
				fixture.op,
				fixture.opResult,
			)

			if fixture.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, len(events), len(fixture.expected),
				"length mismatch: got %d events, expected %d",
				len(events), len(fixture.expected))

			// Create a map to track which events have been matched
			matched := make([]bool, len(events))

			// For each expected event, try to find a matching actual event
			for i, expectedEvent := range fixture.expected {
				found := false
				for j, actualEvent := range events {
					if !matched[j] && proto.Equal(expectedEvent, actualEvent) {
						matched[j] = true
						found = true
						break
					}
				}

				if !found {
					assert.Fail(t, "Expected event not found",
						"Expected event %d: %+v\nAvailable events: %+v",
						i, expectedEvent, events)
				}
			}

		})
	}
}

func TestAccountCreateEvents(t *testing.T) {
	createAccountOp := func(sourceAccount xdr.MuxedAccount, newAccount xdr.AccountId, startingBalance xdr.Int64) xdr.Operation {
		return xdr.Operation{
			SourceAccount: &sourceAccount,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeCreateAccount,
				CreateAccountOp: &xdr.CreateAccountOp{
					Destination:     newAccount,
					StartingBalance: startingBalance,
				},
			},
		}
	}
	tests := []testFixture{
		{
			name:     "successful account creation",
			tx:       someTx,
			opIndex:  0,
			op:       createAccountOp(accountA, accountB.ToAccountId(), hundredUnits),
			opResult: xdr.OperationResult{},
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(accountA), protoAddressFromAccount(accountB), hundredUnitsStr, xlmProtoAsset),
			},
		},
	}

	runTokenTransferEventTests(t, tests)
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

	mergedAccountResultWithBalance := func(balance *xdr.Int64) xdr.OperationResult {
		return xdr.OperationResult{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeAccountMerge,
				AccountMergeResult: &xdr.AccountMergeResult{
					Code:                 xdr.AccountMergeResultCodeAccountMergeSuccess,
					SourceAccountBalance: balance,
				},
			},
		}
	}

	tests := []testFixture{
		{
			name:     "successful account merge",
			tx:       someTx,
			opIndex:  0,
			op:       mergeAccountOp,
			opResult: mergedAccountResultWithBalance(&hundredUnits),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(accountA), protoAddressFromAccount(accountB), hundredUnitsStr, xlmProtoAsset),
			},
		},
		{
			name:     "empty account merge - no events",
			tx:       someTx,
			opIndex:  0,
			op:       mergeAccountOp,
			opResult: mergedAccountResultWithBalance(nil),
			expected: nil,
		},
	}

	runTokenTransferEventTests(t, tests)
}

func TestPaymentEvents(t *testing.T) {
	paymentOp := func(src *xdr.MuxedAccount, dst xdr.MuxedAccount, asset xdr.Asset, amount xdr.Int64) xdr.Operation {
		return xdr.Operation{
			SourceAccount: src,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypePayment,
				PaymentOp: &xdr.PaymentOp{
					Destination: dst,
					Amount:      amount,
					Asset:       asset,
				},
			},
		}
	}

	tests := []testFixture{
		{
			name:    "G account to G account - XLM transfer",
			tx:      someTx,
			opIndex: 0,
			op:      paymentOp(&accountA, accountB, xlmAsset, hundredUnits),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(accountA), protoAddressFromAccount(accountB), hundredUnitsStr, xlmProtoAsset),
			},
		},
		{
			name:    "G account to G account - USDC transfer",
			tx:      someTx,
			opIndex: 0,
			op:      paymentOp(&accountA, accountB, usdcAsset, hundredUnits),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(accountA), protoAddressFromAccount(accountB), hundredUnitsStr, usdcProtoAsset),
			},
		},
		{
			name:    "G account to M Account - USDC transfer",
			tx:      someTx,
			opIndex: 0,
			op:      paymentOp(&accountA, muxedAccountB, usdcAsset, hundredUnits),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(accountA), protoAddressFromAccount(muxedAccountB), hundredUnitsStr, usdcProtoAsset),
			},
		},
		{
			name:    "M account to G Account - USDC transfer",
			tx:      someTx,
			opIndex: 0,
			op:      paymentOp(&muxedAccountA, accountB, usdcAsset, hundredUnits),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(muxedAccountA), protoAddressFromAccount(accountB), hundredUnitsStr, usdcProtoAsset),
			},
		},
		{
			name:    "G (issuer account) to G account - USDC mint",
			tx:      someTx,
			opIndex: 0,
			op:      paymentOp(&usdcAccount, accountB, usdcAsset, hundredUnits),
			expected: []*TokenTransferEvent{
				mintEvent(protoAddressFromAccount(accountB), hundredUnitsStr, usdcProtoAsset),
			},
		},
		{
			name:    "G (issuer account) to M account - USDC mint",
			tx:      someTx,
			opIndex: 0,
			op:      paymentOp(&usdcAccount, muxedAccountB, usdcAsset, hundredUnits),
			expected: []*TokenTransferEvent{
				mintEvent(protoAddressFromAccount(muxedAccountB), hundredUnitsStr, usdcProtoAsset),
			},
		},
		{
			name:    "G account to G (issuer account) - USDC burn",
			tx:      someTx,
			opIndex: 0,
			op:      paymentOp(&accountA, usdcAccount, usdcAsset, hundredUnits),
			expected: []*TokenTransferEvent{
				burnEvent(protoAddressFromAccount(accountA), hundredUnitsStr, usdcProtoAsset),
			},
		},
		{
			name:    "M account to G (issuer account) - USDC burn",
			tx:      someTx,
			opIndex: 0,
			op:      paymentOp(&muxedAccountA, usdcAccount, usdcAsset, hundredUnits),
			expected: []*TokenTransferEvent{
				burnEvent(protoAddressFromAccount(muxedAccountA), hundredUnitsStr, usdcProtoAsset),
			},
		},
	}

	runTokenTransferEventTests(t, tests)
}

func TestManageOfferEvents(t *testing.T) {
	manageBuyOfferOp := func(sourceAccount xdr.MuxedAccount) xdr.Operation {
		return xdr.Operation{
			SourceAccount: &sourceAccount,
			Body: xdr.OperationBody{
				Type:             xdr.OperationTypeManageBuyOffer,
				ManageBuyOfferOp: &xdr.ManageBuyOfferOp{},
			},
		}
	}

	manageBuyOfferResult := func(claims []xdr.ClaimOfferAtom) xdr.OperationResult {
		var offersClaimed []xdr.ClaimAtom
		for _, c := range claims {
			offersClaimed = append(offersClaimed, xdr.ClaimAtom{
				Type:      xdr.ClaimAtomTypeClaimAtomTypeOrderBook,
				OrderBook: &c,
			})
		}

		return xdr.OperationResult{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeManageBuyOffer,
				ManageBuyOfferResult: &xdr.ManageBuyOfferResult{
					Success: &xdr.ManageOfferSuccessResult{
						OffersClaimed: offersClaimed,
					},
				},
			},
		}
	}

	manageSellOfferOp := func(sourceAccount xdr.MuxedAccount) xdr.Operation {
		return xdr.Operation{
			SourceAccount: &sourceAccount,
			Body: xdr.OperationBody{
				Type:              xdr.OperationTypeManageSellOffer,
				ManageSellOfferOp: &xdr.ManageSellOfferOp{},
			},
		}
	}

	manageSellOfferResult := func(claims []xdr.ClaimOfferAtom) xdr.OperationResult {
		var offersClaimed []xdr.ClaimAtom
		for _, c := range claims {
			offersClaimed = append(offersClaimed, xdr.ClaimAtom{
				Type:      xdr.ClaimAtomTypeClaimAtomTypeOrderBook,
				OrderBook: &c,
			})
		}

		return xdr.OperationResult{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeManageSellOffer,
				ManageSellOfferResult: &xdr.ManageSellOfferResult{
					Success: &xdr.ManageOfferSuccessResult{
						OffersClaimed: offersClaimed,
					},
				},
			},
		}
	}

	// Fixture
	tests := []testFixture{
		{
			name:    "ManageBuyOffer - Buy USDC for XLM (2 claim atoms, Transfer events)",
			tx:      someTx,
			opIndex: 0,
			op:      manageBuyOfferOp(someTxAccount), // don't care for anything in xdr.Operation other than source account
			opResult: manageBuyOfferResult(
				[]xdr.ClaimOfferAtom{
					// 1 USDC == 5 XLM
					{SellerId: accountA.ToAccountId(), AssetSold: usdcAsset, AssetBought: xlmAsset, AmountSold: oneUnit, AmountBought: fiveUnits},
					{SellerId: accountB.ToAccountId(), AssetSold: usdcAsset, AssetBought: xlmAsset, AmountSold: twoUnits, AmountBought: tenUnits},
				},
			),

			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(accountA), protoAddressFromAccount(someTxAccount), "1.0000000", usdcProtoAsset),
				transferEvent(protoAddressFromAccount(someTxAccount), protoAddressFromAccount(accountA), "5.0000000", xlmProtoAsset),

				transferEvent(protoAddressFromAccount(accountB), protoAddressFromAccount(someTxAccount), "2.0000000", usdcProtoAsset),
				transferEvent(protoAddressFromAccount(someTxAccount), protoAddressFromAccount(accountB), "10.0000000", xlmProtoAsset),
			},
		},

		{
			name:    "ManageSellOffer - Sell USDC for XLM (2 claim atoms, Transfer events)",
			tx:      someTx,
			opIndex: 0,
			op:      manageSellOfferOp(someTxAccount), // don't care for anything in xdr.Operation other than source account
			opResult: manageSellOfferResult([]xdr.ClaimOfferAtom{
				// 1 USDC = 3 XLM
				{SellerId: accountA.ToAccountId(), AssetSold: xlmAsset, AssetBought: usdcAsset, AmountSold: threeUnits, AmountBought: oneUnit},
				{SellerId: accountB.ToAccountId(), AssetSold: xlmAsset, AssetBought: usdcAsset, AmountSold: sixUnits, AmountBought: twoUnits},
			}),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(accountA), protoAddressFromAccount(someTxAccount), "3.0000000", xlmProtoAsset),
				transferEvent(protoAddressFromAccount(someTxAccount), protoAddressFromAccount(accountA), "1.0000000", usdcProtoAsset),

				transferEvent(protoAddressFromAccount(accountB), protoAddressFromAccount(someTxAccount), "6.0000000", xlmProtoAsset),
				transferEvent(protoAddressFromAccount(someTxAccount), protoAddressFromAccount(accountB), "2.0000000", usdcProtoAsset),
			},
		},

		{
			name:    "ManageBuyOffer - Buy USDC for XLM (Source is USDC issuer, 2 claim atoms, BURN events)",
			tx:      someTx,
			opIndex: 0,
			op:      manageBuyOfferOp(usdcAccount), // don't care for anything in xdr.Operation other than source account
			opResult: manageBuyOfferResult([]xdr.ClaimOfferAtom{
				// 1 USDC == 5 XLM
				{SellerId: accountA.ToAccountId(), AssetSold: usdcAsset, AssetBought: xlmAsset, AmountSold: oneUnit, AmountBought: fiveUnits},
				{SellerId: accountB.ToAccountId(), AssetSold: usdcAsset, AssetBought: xlmAsset, AmountSold: twoUnits, AmountBought: tenUnits},
			}),
			expected: []*TokenTransferEvent{
				burnEvent(protoAddressFromAccount(accountA), "1.0000000", usdcProtoAsset),
				transferEvent(protoAddressFromAccount(usdcAccount), protoAddressFromAccount(accountA), "5.0000000", xlmProtoAsset),

				burnEvent(protoAddressFromAccount(accountB), "2.0000000", usdcProtoAsset),
				transferEvent(protoAddressFromAccount(usdcAccount), protoAddressFromAccount(accountB), "10.0000000", xlmProtoAsset),
			},
		},

		{
			name:    "ManageSellOffer - Sell USDC for XLM (Source is USDC issuer, 2 claim atoms, MINT events)",
			tx:      someTx,
			opIndex: 0,
			op:      manageSellOfferOp(usdcAccount), // don't care for anything in xdr.Operation other than source account
			opResult: manageSellOfferResult([]xdr.ClaimOfferAtom{
				// 1 USDC = 3 XLM
				{SellerId: accountA.ToAccountId(), AssetSold: xlmAsset, AssetBought: usdcAsset, AmountSold: threeUnits, AmountBought: oneUnit},
				{SellerId: accountB.ToAccountId(), AssetSold: xlmAsset, AssetBought: usdcAsset, AmountSold: sixUnits, AmountBought: twoUnits},
			}),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(accountA), protoAddressFromAccount(usdcAccount), "3.0000000", xlmProtoAsset),
				mintEvent(protoAddressFromAccount(accountA), "1.0000000", usdcProtoAsset),

				transferEvent(protoAddressFromAccount(accountB), protoAddressFromAccount(usdcAccount), "6.0000000", xlmProtoAsset),
				mintEvent(protoAddressFromAccount(accountB), "2.0000000", usdcProtoAsset),
			},
		},
	}

	runTokenTransferEventTests(t, tests)
}

func TestFeeEvent(t *testing.T) {
	failedTx := func(envelopeType xdr.EnvelopeType, txFee xdr.Int64) ingest.LedgerTransaction {
		tx := ingest.LedgerTransaction{
			Ledger:   someLcm,
			Hash:     someTxHash,
			Envelope: xdr.TransactionEnvelope{},
			Result: xdr.TransactionResultPair{
				TransactionHash: someTxHash,
				Result: xdr.TransactionResult{
					FeeCharged: txFee,
					Result:     xdr.TransactionResultResult{},
				},
			},
		}
		if envelopeType == xdr.EnvelopeTypeEnvelopeTypeTxFeeBump {
			tx.Envelope = xdr.TransactionEnvelope{
				Type: xdr.EnvelopeTypeEnvelopeTypeTxFeeBump,
				FeeBump: &xdr.FeeBumpTransactionEnvelope{
					Tx: xdr.FeeBumpTransaction{
						FeeSource: someTxAccount,
						InnerTx: xdr.FeeBumpTransactionInnerTx{
							Type: xdr.EnvelopeTypeEnvelopeTypeTx,
							V1:   &xdr.TransactionV1Envelope{},
						},
					},
				},
			}
			tx.Result.Result.Result.Code = xdr.TransactionResultCodeTxFeeBumpInnerFailed
		} else if envelopeType == xdr.EnvelopeTypeEnvelopeTypeTx {
			tx.Envelope = xdr.TransactionEnvelope{
				Type: xdr.EnvelopeTypeEnvelopeTypeTx,
				V1: &xdr.TransactionV1Envelope{
					Tx: xdr.Transaction{
						SourceAccount: someTxAccount,
					},
				},
			}
			tx.Result.Result.Result.Code = xdr.TransactionResultCodeTxFailed
		}
		return tx
	}

	expectedFeeEvent := func(feeAmt string) *TokenTransferEvent {
		return NewFeeEvent(
			someLcm.LedgerSequence(), someLcm.ClosedAt(), someTxHash.HexString(), protoAddressFromAccount(someTxAccount),
			feeAmt,
		)
	}

	// Fixture
	tests := []testFixture{
		{
			name: "Fee Event only - Failed Fee Bump Transaction",
			tx:   failedTx(xdr.EnvelopeTypeEnvelopeTypeTxFeeBump, xdr.Int64(1e7/1e2)), // Fee  = 0.01 XLM
			expected: []*TokenTransferEvent{
				expectedFeeEvent("0.0100000"),
			},
		},
		{
			name: "Fee Event only - Failed V1 Transaction",
			tx:   failedTx(xdr.EnvelopeTypeEnvelopeTypeTx, xdr.Int64(1e7/1e4)), // Fee  = 0.0001 XLM ,
			expected: []*TokenTransferEvent{
				expectedFeeEvent("0.0001000"),
			},
		},
	}

	for _, fixture := range tests {
		t.Run(fixture.name, func(t *testing.T) {
			events, err := ProcessTokenTransferEventsFromTransaction(fixture.tx)
			assert.NoError(t, err)
			assert.Equal(t, len(fixture.expected), len(events))
			for i := range events {
				assert.True(t, proto.Equal(events[i], fixture.expected[i]),
					"Expected event: %+v\nFound event: %+v", fixture.expected[i], events[i])
			}
		})
	}
}
