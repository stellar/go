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

	lpDepositOp = func(poolId xdr.PoolId, sourceAccount *xdr.MuxedAccount) xdr.Operation {
		return xdr.Operation{
			SourceAccount: sourceAccount,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeLiquidityPoolDeposit,
				LiquidityPoolDepositOp: &xdr.LiquidityPoolDepositOp{
					LiquidityPoolId: poolId,
				},
			},
		}
	}

	lpWithdrawOp = func(poolId xdr.PoolId, sourceAccount *xdr.MuxedAccount) xdr.Operation {
		return xdr.Operation{
			SourceAccount: sourceAccount,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeLiquidityPoolWithdraw,
				LiquidityPoolWithdrawOp: &xdr.LiquidityPoolWithdrawOp{
					LiquidityPoolId: poolId,
				},
			},
		}
	}

	lpLedgerEntry = func(poolId xdr.PoolId, assetA xdr.Asset, assetB xdr.Asset, reserveA xdr.Int64, reserveB xdr.Int64) *xdr.LiquidityPoolEntry {
		return &xdr.LiquidityPoolEntry{
			LiquidityPoolId: poolId,
			Body: xdr.LiquidityPoolEntryBody{
				Type: xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
				ConstantProduct: &xdr.LiquidityPoolEntryConstantProduct{
					Params: xdr.LiquidityPoolConstantProductParameters{
						AssetA: assetA,
						AssetB: assetB,
						Fee:    20,
					},
					ReserveA: reserveA,
					ReserveB: reserveB,
				},
			},
		}
	}

	lpEntryBaseState = func(lpLedgerEntry *xdr.LiquidityPoolEntry) xdr.LedgerEntryChange {
		return xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
			State: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type:          xdr.LedgerEntryTypeLiquidityPool,
					LiquidityPool: lpLedgerEntry,
				},
			},
		}
	}

	lpEntryCreatedChange = func(lpEntry *xdr.LiquidityPoolEntry) xdr.LedgerEntryChange {
		return xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryCreated,
			Created: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type:          xdr.LedgerEntryTypeLiquidityPool,
					LiquidityPool: lpEntry,
				},
			},
		}
	}

	lpEntryRemovedChange = func(lpId xdr.PoolId) xdr.LedgerEntryChange {
		return xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
			Removed: &xdr.LedgerKey{
				LiquidityPool: &xdr.LedgerKeyLiquidityPool{
					LiquidityPoolId: lpId,
				},
			},
		}
	}

	lpEntryUpdatedChange = func(entry *xdr.LiquidityPoolEntry, newReserveA xdr.Int64, newReserveB xdr.Int64) xdr.LedgerEntryChange {
		entry.Body.ConstantProduct.ReserveA = newReserveA
		entry.Body.ConstantProduct.ReserveB = newReserveB
		return xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
			Updated: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type:          xdr.LedgerEntryTypeLiquidityPool,
					LiquidityPool: entry,
				},
			},
		}
	}

	generateClaimAtom = func(claimAtomType xdr.ClaimAtomType, sellerId *xdr.MuxedAccount, lpId *xdr.PoolId, assetSold xdr.Asset, amountSold xdr.Int64, assetBought xdr.Asset, amountBought xdr.Int64) xdr.ClaimAtom {
		claimAtom := xdr.ClaimAtom{
			Type: claimAtomType,
		}
		if claimAtomType == xdr.ClaimAtomTypeClaimAtomTypeLiquidityPool {
			claimAtom.LiquidityPool = &xdr.ClaimLiquidityAtom{
				LiquidityPoolId: *lpId,
				AssetBought:     assetBought,
				AmountBought:    amountBought,
				AssetSold:       assetSold,
				AmountSold:      amountSold,
			}
		} else if claimAtomType == xdr.ClaimAtomTypeClaimAtomTypeOrderBook {
			claimAtom.OrderBook = &xdr.ClaimOfferAtom{
				SellerId:     sellerId.ToAccountId(),
				AssetBought:  assetBought,
				AmountBought: amountBought,
				AssetSold:    assetSold,
				AmountSold:   amountSold,
			}
		}
		return claimAtom
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

func someTxWithOperationChanges(changes xdr.LedgerEntryChanges) ingest.LedgerTransaction {
	return ingest.LedgerTransaction{
		Index: 1,
		Envelope: xdr.TransactionEnvelope{
			Type: xdr.EnvelopeTypeEnvelopeTypeTx,
			V1: &xdr.TransactionV1Envelope{
				Tx: xdr.Transaction{
					SourceAccount: someTxAccount,
				},
			},
		},
		Result: xdr.TransactionResultPair{},
		UnsafeMeta: xdr.TransactionMeta{
			V: 3,
			V3: &xdr.TransactionMetaV3{
				Operations: []xdr.OperationMeta{
					{
						Changes: changes,
					},
				},
			},
		},
		LedgerVersion: 1234,
		Ledger:        someLcm,
		Hash:          someTxHash,
	}
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

			// For each expected event, try to find a matching actual event
			for i := range fixture.expected {
				if !proto.Equal(events[i], fixture.expected[i]) {
					assert.Fail(t, "result does not match expected event",
						"Expected: %+v\nFound: %+v", fixture.expected[i], events[i])
				}
			}

		})
	}
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
	manageBuyOfferOp := func(sourceAccount *xdr.MuxedAccount) xdr.Operation {
		return xdr.Operation{
			SourceAccount: sourceAccount,
			Body: xdr.OperationBody{
				Type:             xdr.OperationTypeManageBuyOffer,
				ManageBuyOfferOp: &xdr.ManageBuyOfferOp{},
			},
		}
	}

	manageBuyOfferResult := func(claims []xdr.ClaimAtom) xdr.OperationResult {
		return xdr.OperationResult{
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeManageBuyOffer,
				ManageBuyOfferResult: &xdr.ManageBuyOfferResult{
					Success: &xdr.ManageOfferSuccessResult{
						OffersClaimed: claims,
					},
				},
			},
		}
	}

	manageSellOfferOp := func(sourceAccount *xdr.MuxedAccount) xdr.Operation {
		return xdr.Operation{
			SourceAccount: sourceAccount,
			Body: xdr.OperationBody{
				Type:              xdr.OperationTypeManageSellOffer,
				ManageSellOfferOp: &xdr.ManageSellOfferOp{},
			},
		}
	}

	manageSellOfferResult := func(claims []xdr.ClaimAtom) xdr.OperationResult {
		return xdr.OperationResult{
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeManageSellOffer,
				ManageSellOfferResult: &xdr.ManageSellOfferResult{
					Success: &xdr.ManageOfferSuccessResult{
						OffersClaimed: claims,
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
			op:      manageBuyOfferOp(nil), // don't care for anything in xdr.Operation other than source account
			opResult: manageBuyOfferResult(
				[]xdr.ClaimAtom{
					// 1 USDC == 5 XLM
					generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeOrderBook, &accountA, nil, usdcAsset, oneUnit, xlmAsset, fiveUnits),
					generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeOrderBook, &accountB, nil, usdcAsset, twoUnits, xlmAsset, tenUnits),
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
			op:      manageSellOfferOp(nil), // don't care for anything in xdr.Operation other than source account
			opResult: manageSellOfferResult(
				[]xdr.ClaimAtom{
					// 1 USDC = 3 XLM
					generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeOrderBook, &accountA, nil, xlmAsset, threeUnits, usdcAsset, oneUnit),
					generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeOrderBook, &accountB, nil, xlmAsset, sixUnits, usdcAsset, twoUnits),
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
			op:      manageBuyOfferOp(&usdcAccount), // don't care for anything in xdr.Operation other than source account
			opResult: manageBuyOfferResult([]xdr.ClaimAtom{
				// 1 USDC == 5 XLM
				generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeOrderBook, &accountA, nil, usdcAsset, oneUnit, xlmAsset, fiveUnits),
				generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeOrderBook, &accountB, nil, usdcAsset, twoUnits, xlmAsset, tenUnits),
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
			op:      manageSellOfferOp(&usdcAccount), // don't care for anything in xdr.Operation other than source account
			opResult: manageSellOfferResult([]xdr.ClaimAtom{
				// 1 USDC = 3 XLM
				generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeOrderBook, &accountA, nil, xlmAsset, threeUnits, usdcAsset, oneUnit),
				generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeOrderBook, &accountB, nil, xlmAsset, sixUnits, usdcAsset, twoUnits),

				//{SellerId: accountA.ToAccountId(), AssetSold: xlmAsset, AssetBought: usdcAsset, AmountSold: threeUnits, AmountBought: oneUnit},
				//{SellerId: accountB.ToAccountId(), AssetSold: xlmAsset, AssetBought: usdcAsset, AmountSold: sixUnits, AmountBought: twoUnits},
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

func TestLiquidityPoolEvents(t *testing.T) {
	tests := []testFixture{
		{
			name:    "Liquidity Pool Deposit Operation - New LP Creation - Transfer",
			opIndex: 0,
			op:      lpDepositOp(lpBtcEthId, nil), // source account = Tx account
			tx: someTxWithOperationChanges(
				xdr.LedgerEntryChanges{
					lpEntryCreatedChange(lpLedgerEntry(lpBtcEthId, btcAsset, ethAsset, oneUnit, threeUnits)),
				}),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(someTxAccount), protoAddressFromLpHash(lpBtcEthId), "1.0000000", btcProtoAsset),
				transferEvent(protoAddressFromAccount(someTxAccount), protoAddressFromLpHash(lpBtcEthId), "3.0000000", ethProtoAsset),
			},
		},
		{
			name:    "Liquidity Pool Deposit Operation - Existing LP Update - Transfer",
			opIndex: 0,
			op:      lpDepositOp(lpBtcEthId, nil), // source account = Tx account
			tx: someTxWithOperationChanges(
				xdr.LedgerEntryChanges{
					lpEntryBaseState(lpLedgerEntry(lpBtcEthId, btcAsset, ethAsset, oneUnit, threeUnits)),
					// Increase BTC from 1 -> 5, ETH from 3 -> 10
					lpEntryUpdatedChange(lpLedgerEntry(lpBtcEthId, btcAsset, ethAsset, oneUnit, threeUnits), fiveUnits, tenUnits),
				}),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(someTxAccount), protoAddressFromLpHash(lpBtcEthId), "4.0000000", btcProtoAsset),
				transferEvent(protoAddressFromAccount(someTxAccount), protoAddressFromLpHash(lpBtcEthId), "7.0000000", ethProtoAsset),
			},
		},
		{
			name:    "Liquidity Pool Deposit Operation - New LP Creation - Mint",
			opIndex: 0,
			op:      lpDepositOp(lpBtcEthId, &btcAccount), //operation source account = BTC Asset Issuer
			tx: someTxWithOperationChanges(
				xdr.LedgerEntryChanges{
					lpEntryCreatedChange(lpLedgerEntry(lpBtcEthId, btcAsset, ethAsset, oneUnit, threeUnits)),
				}),
			expected: []*TokenTransferEvent{
				mintEvent(protoAddressFromLpHash(lpBtcEthId), "1.0000000", btcProtoAsset),
				transferEvent(protoAddressFromAccount(btcAccount), protoAddressFromLpHash(lpBtcEthId), "3.0000000", ethProtoAsset),
			},
		},
		{
			name:    "Liquidity Pool Withdraw Operation - Existing LP Update - Transfer",
			opIndex: 0,
			op:      lpWithdrawOp(lpBtcEthId, nil), // source account = Tx account
			tx: someTxWithOperationChanges(
				xdr.LedgerEntryChanges{
					lpEntryBaseState(lpLedgerEntry(lpBtcEthId, btcAsset, ethAsset, fiveUnits, tenUnits)),
					// Decrease BTC from 5 -> 2, ETH from 10 -> 2
					lpEntryUpdatedChange(lpLedgerEntry(lpBtcEthId, btcAsset, ethAsset, fiveUnits, tenUnits), twoUnits, twoUnits),
				}),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromLpHash(lpBtcEthId), protoAddressFromAccount(someTxAccount), "3.0000000", btcProtoAsset),
				transferEvent(protoAddressFromLpHash(lpBtcEthId), protoAddressFromAccount(someTxAccount), "8.0000000", ethProtoAsset),
			},
		},
		{
			name:    "Liquidity Pool Withdraw Operation - LP Removed - Transfer",
			opIndex: 0,
			op:      lpWithdrawOp(lpBtcEthId, nil), // source account = Tx account
			tx: someTxWithOperationChanges(
				xdr.LedgerEntryChanges{
					lpEntryBaseState(lpLedgerEntry(lpBtcEthId, btcAsset, ethAsset, fiveUnits, tenUnits)),
					lpEntryRemovedChange(lpBtcEthId),
				}),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromLpHash(lpBtcEthId), protoAddressFromAccount(someTxAccount), "5.0000000", btcProtoAsset),
				transferEvent(protoAddressFromLpHash(lpBtcEthId), protoAddressFromAccount(someTxAccount), "10.0000000", ethProtoAsset),
			},
		},
		{
			name:    "Liquidity Pool Withdraw Operation - LP Removed - Burn",
			opIndex: 0,
			op:      lpWithdrawOp(lpBtcEthId, &ethAccount), // operation Source Account = EthIssuer
			tx: someTxWithOperationChanges(
				xdr.LedgerEntryChanges{
					lpEntryBaseState(lpLedgerEntry(lpBtcEthId, btcAsset, ethAsset, fiveUnits, tenUnits)),
					lpEntryRemovedChange(lpBtcEthId),
				}),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromLpHash(lpBtcEthId), protoAddressFromAccount(ethAccount), "5.0000000", btcProtoAsset),
				burnEvent(protoAddressFromLpHash(lpBtcEthId), "10.0000000", ethProtoAsset),
			},
		},
	}

	runTokenTransferEventTests(t, tests)
}
