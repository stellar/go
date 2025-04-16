package token_transfer

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/ingest"
	assetProto "github.com/stellar/go/ingest/asset"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"testing"
)

var (
	someNetworkPassphrase = "some network passphrase"
	someTxAccount         = xdr.MustMuxedAddress("GBF3XFXGBGNQDN3HOSZ7NVRF6TJ2JOD5U6ELIWJOOEI6T5WKMQT2YSXQ")
	someTxHash            = xdr.Hash{1, 1, 1, 1}

	accountA         = xdr.MustMuxedAddress("GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON")
	accountB         = xdr.MustMuxedAddress("GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z")
	memoA            = uint64(123)
	memoB            = uint64(234)
	muxedAccountA, _ = xdr.MuxedAccountFromAccountId(accountA.Address(), memoA)
	muxedAccountB, _ = xdr.MuxedAccountFromAccountId(accountB.Address(), memoB)

	oneUnit = xdr.Int64(1e7)

	unitsToStr = func(v xdr.Int64) string {
		return amount.String64Raw(v)
	}

	usdcIssuer     = "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN"
	usdcAccount    = xdr.MustMuxedAddress(usdcIssuer)
	usdcAsset      = xdr.MustNewCreditAsset("USDC", usdcIssuer)
	usdcProtoAsset = assetProto.NewProtoAsset(usdcAsset)

	ethIssuer     = "GCEODJVUUVYVFD5KT4TOEDTMXQ76OPFOQC2EMYYMLPXQCUVPOB6XRWPQ"
	ethAccount    = xdr.MustMuxedAddress(ethIssuer)
	ethAsset      = xdr.MustNewCreditAsset("ETH", ethIssuer)
	ethProtoAsset = assetProto.NewProtoAsset(ethAsset)

	btcIsuer      = "GBT4YAEGJQ5YSFUMNKX6BPBUOCPNAIOFAVZOF6MIME2CECBMEIUXFZZN"
	btcAccount    = xdr.MustMuxedAddress(btcIsuer)
	btcAsset      = xdr.MustNewCreditAsset("BTC", btcIsuer)
	btcProtoAsset = assetProto.NewProtoAsset(btcAsset)

	lpBtcEthId, _  = xdr.NewPoolId(btcAsset, ethAsset, xdr.LiquidityPoolFeeV18)
	lpEthUsdcId, _ = xdr.NewPoolId(ethAsset, usdcAsset, xdr.LiquidityPoolFeeV18)

	someBalanceId = xdr.ClaimableBalanceId{
		Type: xdr.ClaimableBalanceIdTypeClaimableBalanceIdTypeV0,
		V0:   &xdr.Hash{1, 2, 3, 4, 5},
	}

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
					SeqNum:        xdr.SequenceNumber(54321), // need this for generating CbIds from LPIds for revokeTrustline tests
				},
			},
		},
		Result: xdr.TransactionResultPair{},
		UnsafeMeta: xdr.TransactionMeta{
			V: 3,
			V3: &xdr.TransactionMetaV3{
				Operations: []xdr.OperationMeta{{}},
			},
		},
		LedgerVersion: 1234,
		Ledger:        someLcm,
		Hash:          someTxHash,
	}

	someTxWithOperationChanges = func(changes xdr.LedgerEntryChanges) ingest.LedgerTransaction {
		resp := someTx
		resp.UnsafeMeta.V = 3
		resp.UnsafeMeta.V3 = &xdr.TransactionMetaV3{
			Operations: []xdr.OperationMeta{
				{
					Changes: changes,
				},
			},
		}
		return resp
	}

	someOldTxWithOperationChanges = func(changes xdr.LedgerEntryChanges) ingest.LedgerTransaction {
		resp := someTxWithOperationChanges(changes)
		someOldLcm := someLcm
		someOldLcm.V0 =
			&xdr.LedgerCloseMetaV0{
				LedgerHeader: xdr.LedgerHeaderHistoryEntry{
					Header: xdr.LedgerHeader{
						LedgerVersion: 5, // This is to trigger the XLM reconciliation check
						LedgerSeq:     xdr.Uint32(12345),
						ScpValue:      xdr.StellarValue{CloseTime: xdr.TimePoint(12345 * 100)},
					},
				},
				TxSet:              xdr.TransactionSet{},
				TxProcessing:       nil,
				UpgradesProcessing: nil,
				ScpInfo:            nil,
			}
		resp.Ledger = someOldLcm
		return resp
	}

	someOperationIndex = uint32(0)

	contractIdStrFromAsset = func(asset xdr.Asset) string {
		contractId, _ := asset.ContractID(someNetworkPassphrase)
		return strkey.MustEncode(strkey.VersionByteContract, contractId[:])
	}

	// Some global anonymous functions.
	mintEvent = func(to string, amt string, asset *assetProto.Asset) *TokenTransferEvent {
		return &TokenTransferEvent{
			Meta: NewEventMetaFromTx(someTx, &someOperationIndex, contractIdStrFromAsset(asset.ToXdrAsset())),
			Event: &TokenTransferEvent_Mint{
				Mint: &Mint{
					To:     to,
					Asset:  asset,
					Amount: amt,
				},
			},
		}

	}

	burnEvent = func(from string, amt string, asset *assetProto.Asset) *TokenTransferEvent {
		return &TokenTransferEvent{
			Meta: NewEventMetaFromTx(someTx, &someOperationIndex, contractIdStrFromAsset(asset.ToXdrAsset())),
			Event: &TokenTransferEvent_Burn{
				Burn: &Burn{
					From:   from,
					Asset:  asset,
					Amount: amt,
				},
			},
		}

	}

	transferEvent = func(from string, to string, amt string, asset *assetProto.Asset) *TokenTransferEvent {
		return &TokenTransferEvent{
			Meta: NewEventMetaFromTx(someTx, &someOperationIndex, contractIdStrFromAsset(asset.ToXdrAsset())),
			Event: &TokenTransferEvent_Transfer{
				Transfer: &Transfer{
					From:   from,
					To:     to,
					Asset:  asset,
					Amount: amt,
				},
			},
		}
	}

	clawbackEvent = func(from string, amt string, asset *assetProto.Asset) *TokenTransferEvent {
		return &TokenTransferEvent{
			Meta: NewEventMetaFromTx(someTx, &someOperationIndex, contractIdStrFromAsset(asset.ToXdrAsset())),
			Event: &TokenTransferEvent_Clawback{
				Clawback: &Clawback{
					From:   from,
					Asset:  asset,
					Amount: amt,
				},
			},
		}
	}

	// Helpers to generate LedgerEntryChanges for Claimable Balances, to be fed to the operationMeta when creating sample transaction
	cbLedgerEntry = func(cbId xdr.ClaimableBalanceId, asset xdr.Asset, amount xdr.Int64) *xdr.ClaimableBalanceEntry {
		return &xdr.ClaimableBalanceEntry{
			BalanceId: cbId,
			Asset:     asset,
			Amount:    amount,
		}
	}

	generateCbEntryChangeState = func(cbLedgerEntry *xdr.ClaimableBalanceEntry) xdr.LedgerEntryChange {
		return xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
			State: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type:             xdr.LedgerEntryTypeClaimableBalance,
					ClaimableBalance: cbLedgerEntry,
				},
			},
		}
	}

	generateCbEntryCreatedChange = func(cbLedgerEntry *xdr.ClaimableBalanceEntry) xdr.LedgerEntryChange {
		return xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryCreated,
			Created: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type:             xdr.LedgerEntryTypeClaimableBalance,
					ClaimableBalance: cbLedgerEntry,
				},
			},
		}
	}

	generateCbEntryRemovedChange = func(cbId xdr.ClaimableBalanceId) xdr.LedgerEntryChange {
		return xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
			Removed: &xdr.LedgerKey{
				ClaimableBalance: &xdr.LedgerKeyClaimableBalance{
					BalanceId: cbId,
				},
			},
		}
	}

	// Helpers to generate LedgerEntryChanges for Liquidity Pools, to be fed to the operationMeta when creating sample transaction
	lpLedgerEntry = func(poolId xdr.PoolId, assetA xdr.Asset, assetB xdr.Asset, reserveA xdr.Int64, reserveB xdr.Int64) *xdr.LiquidityPoolEntry {
		return &xdr.LiquidityPoolEntry{
			LiquidityPoolId: poolId,
			Body: xdr.LiquidityPoolEntryBody{
				Type: xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
				ConstantProduct: &xdr.LiquidityPoolEntryConstantProduct{
					Params: xdr.LiquidityPoolConstantProductParameters{
						AssetA: assetA,
						AssetB: assetB,
					},
					ReserveA: reserveA,
					ReserveB: reserveB,
				},
			},
		}
	}

	generateLpEntryChangeState = func(lpLedgerEntry *xdr.LiquidityPoolEntry) xdr.LedgerEntryChange {
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

	generateLpEntryCreatedChange = func(lpLedgerEntry *xdr.LiquidityPoolEntry) xdr.LedgerEntryChange {
		return xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryCreated,
			Created: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type:          xdr.LedgerEntryTypeLiquidityPool,
					LiquidityPool: lpLedgerEntry,
				},
			},
		}
	}

	generateLpEntryRemovedChange = func(lpId xdr.PoolId) xdr.LedgerEntryChange {
		return xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
			Removed: &xdr.LedgerKey{
				LiquidityPool: &xdr.LedgerKeyLiquidityPool{
					LiquidityPoolId: lpId,
				},
			},
		}
	}

	generateLpEntryUpdatedChange = func(entry *xdr.LiquidityPoolEntry, newReserveA xdr.Int64, newReserveB xdr.Int64) xdr.LedgerEntryChange {
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

	paymentOp = func(src *xdr.MuxedAccount, dst xdr.MuxedAccount, asset xdr.Asset, amount xdr.Int64) xdr.Operation {
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
		ttp := NewEventsProcessor(someNetworkPassphrase)
		t.Run(fixture.name, func(t *testing.T) {
			events, err := ttp.EventsFromOperation(
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
						"index: %d\nExpected: %+v\nFound: %+v", i, fixture.expected[i], events[i])
				}
			}

		})
	}
}

func TestFeeEvent(t *testing.T) {
	failedTx := func(envelopeType xdr.EnvelopeType, txFee xdr.Int64) ingest.LedgerTransaction {
		tx := ingest.LedgerTransaction{
			Index:    1,
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
		return NewFeeEvent(NewEventMetaFromTx(someTx, nil, contractIdStrFromAsset(xlmAsset)),
			protoAddressFromAccount(someTxAccount), feeAmt, xlmProtoAsset)
	}

	tests := []testFixture{
		{
			name: "Fee Event only - Failed Fee Bump Transaction",
			tx:   failedTx(xdr.EnvelopeTypeEnvelopeTypeTxFeeBump, oneUnit/1e2), // Fee  = 0.01 XLM
			expected: []*TokenTransferEvent{
				expectedFeeEvent(unitsToStr(oneUnit / 1e2)),
			},
		},
		{
			name: "Fee Event only - Failed V1 Transaction",
			tx:   failedTx(xdr.EnvelopeTypeEnvelopeTypeTx, oneUnit/1e4), // Fee  = 0.0001 XLM ,
			expected: []*TokenTransferEvent{
				expectedFeeEvent(unitsToStr(oneUnit / 1e4)),
			},
		},
	}

	for _, fixture := range tests {
		ttp := NewEventsProcessor(someNetworkPassphrase)
		t.Run(fixture.name, func(t *testing.T) {
			events, err := ttp.EventsFromTransaction(fixture.tx)
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
			op:       createAccountOp(accountA, accountB.ToAccountId(), 100*oneUnit),
			opResult: xdr.OperationResult{},
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(accountA), protoAddressFromAccount(accountB), unitsToStr(100*oneUnit), xlmProtoAsset),
			},
		},
	}
	runTokenTransferEventTests(t, tests)
}

func TestReconciliation(t *testing.T) {
	accountEntry := func(acc xdr.MuxedAccount, balance xdr.Int64) *xdr.AccountEntry {
		return &xdr.AccountEntry{
			AccountId: acc.ToAccountId(),
			Balance:   1000 * oneUnit,
			SeqNum:    xdr.SequenceNumber(12345),
		}
	}

	generateAccountEntryChangState := func(accountEntry *xdr.AccountEntry) xdr.LedgerEntryChange {
		return xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
			State: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type:    xdr.LedgerEntryTypeAccount,
					Account: accountEntry,
				},
			},
		}
	}

	generateAccountEntryUpdatedChange := func(accountEntry *xdr.AccountEntry, newBalance xdr.Int64) xdr.LedgerEntryChange {
		accountEntry.Balance = newBalance
		return xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
			Updated: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type:    xdr.LedgerEntryTypeAccount,
					Account: accountEntry,
				},
			},
		}
	}

	tests := []testFixture{
		{
			name: "Source Account mints money - part 1",
			op:   paymentOp(&accountA, accountB, xlmAsset, 70*oneUnit),
			tx: someOldTxWithOperationChanges(
				xdr.LedgerEntryChanges{
					// Src initial state
					generateAccountEntryChangState(accountEntry(accountA, 1000*oneUnit)),
					// Src went up from 1000 to 1090 XLM instead of down by 70 XLM. Src minted money
					generateAccountEntryUpdatedChange(accountEntry(accountA, 1000*oneUnit), 1090*oneUnit),
				}),

			expected: []*TokenTransferEvent{
				// diff is between changesDiff and eventsDiff
				// diff = (1090 - 1000) -  (amount in payment that went from src to dst, i.e -70) --> 90 - (-70) = 160
				// +ve value of the diff means it's a mint. mint will appear before any events in the operation
				mintEvent(accountA.Address(), unitsToStr(160*oneUnit), xlmProtoAsset),
				transferEvent(accountA.Address(), accountB.Address(), unitsToStr(70*oneUnit), xlmProtoAsset),
			},
		},
		{
			name: "Source Account burns money  - part 1",
			op:   paymentOp(&accountA, accountB, xlmAsset, 70*oneUnit),
			tx: someOldTxWithOperationChanges(
				xdr.LedgerEntryChanges{
					// Src initial state
					generateAccountEntryChangState(accountEntry(accountA, 1000*oneUnit)),
					// Src went down from 1000 to 800 XLM instead of down by 70 XLM. Src burned money
					generateAccountEntryUpdatedChange(accountEntry(accountA, 1000*oneUnit), 800*oneUnit),
				}),

			expected: []*TokenTransferEvent{
				// diff is between changesDiff and eventsDiff
				// diff = (800 - 1000) -  (amount in payment that went from src to dst, i.e -70) --> -200 - (-70) = -130
				// -ve value of the diff means it's a burn. burn will appear after all events in the operation
				transferEvent(accountA.Address(), accountB.Address(), unitsToStr(70*oneUnit), xlmProtoAsset),
				burnEvent(accountA.Address(), unitsToStr(130*oneUnit), xlmProtoAsset),
			},
		},
		{
			name: "Source Account mints money - part 2",
			op:   paymentOp(&accountA, accountB, xlmAsset, 70*oneUnit),
			tx: someOldTxWithOperationChanges(
				xdr.LedgerEntryChanges{
					// Src initial state
					generateAccountEntryChangState(accountEntry(accountA, 1000*oneUnit)),
					// Src was supposed to go down by 70, instead it went down only by 20. Src minted money
					generateAccountEntryUpdatedChange(accountEntry(accountA, 1000*oneUnit), 980*oneUnit),
				}),

			expected: []*TokenTransferEvent{
				// diff is between changesDiff and eventsDiff
				// diff = (980 - 1000) -  (amount in payment that went from src to dst, i.e -70) --> -20 - (-70) = 50
				// +ve value of the diff means it's a mint. mint will appear before any events in the operation
				mintEvent(accountA.Address(), unitsToStr(50*oneUnit), xlmProtoAsset),
				transferEvent(accountA.Address(), accountB.Address(), unitsToStr(70*oneUnit), xlmProtoAsset),
			},
		},
		{
			name: "Source Account mints money - part 3",
			op:   paymentOp(&accountA, accountB, xlmAsset, 70*oneUnit),
			tx: someOldTxWithOperationChanges(
				xdr.LedgerEntryChanges{
					// no entries for src (imagine it was merged in a previous operation). This actually happened on pubnet
					// No changes implies that src balance didnt change (semantically), when it shud have gone done by 70. Src minted money
				}),

			expected: []*TokenTransferEvent{
				// diff is between changesDiff and eventsDiff
				// diff = 0 -  (amount in payment that went from src to dst, i.e -70) --> 0 - (-70) = 70
				// +ve value of the diff means it's a mint. mint will appear before any events in the operation
				mintEvent(accountA.Address(), unitsToStr(70*oneUnit), xlmProtoAsset),
				transferEvent(accountA.Address(), accountB.Address(), unitsToStr(70*oneUnit), xlmProtoAsset),
			},
		},
		{
			name: "Tx Account mints money by sending a 0 value payment",
			op:   paymentOp(nil, accountB, xlmAsset, 0*oneUnit),
			tx: someOldTxWithOperationChanges(
				xdr.LedgerEntryChanges{
					// Src initial state
					generateAccountEntryChangState(accountEntry(someTxAccount, 1000*oneUnit)),
					// diff = 1020 - 1000  = 20
					generateAccountEntryUpdatedChange(accountEntry(someTxAccount, 1000*oneUnit), 1020*oneUnit),
				}),

			expected: []*TokenTransferEvent{
				// +ve value of the diff means it's a mint. mint will appear before any events in the operation
				mintEvent(someTxAccount.Address(), unitsToStr(20*oneUnit), xlmProtoAsset),
				transferEvent(someTxAccount.Address(), accountB.Address(), unitsToStr(0*oneUnit), xlmProtoAsset),
			},
		},
		{
			name: "Tx Account burns money by sending a 0 value payment",
			op:   paymentOp(nil, accountB, xlmAsset, 0*oneUnit),
			tx: someOldTxWithOperationChanges(
				xdr.LedgerEntryChanges{
					// Src initial state
					generateAccountEntryChangState(accountEntry(someTxAccount, 1000*oneUnit)),
					// diff = 1000 - 980  = -20
					generateAccountEntryUpdatedChange(accountEntry(someTxAccount, 1000*oneUnit), 980*oneUnit),
				}),

			expected: []*TokenTransferEvent{
				// -ve value of the diff means it's a burn. burn` will appear before any events in the operation
				transferEvent(someTxAccount.Address(), accountB.Address(), unitsToStr(0*oneUnit), xlmProtoAsset),
				burnEvent(someTxAccount.Address(), unitsToStr(20*oneUnit), xlmProtoAsset),
			},
		},
	}

	runTokenTransferEventTests(t, tests)
}

func TestInflationEvents(t *testing.T) {
	inflationOp := xdr.Operation{
		Body: xdr.OperationBody{
			Type: xdr.OperationTypeInflation,
		},
	}
	inflationResults := func(payouts []xdr.InflationPayout) xdr.OperationResult {
		return xdr.OperationResult{
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeInflation,
				InflationResult: &xdr.InflationResult{
					Code:    xdr.InflationResultCodeInflationSuccess,
					Payouts: &payouts,
				},
			},
		}
	}

	tests := []testFixture{
		{
			name: "inflation payout to multiple recipients - multiple mints",
			tx:   someTx,
			op:   inflationOp,
			opResult: inflationResults([]xdr.InflationPayout{
				{Destination: accountA.ToAccountId(), Amount: 111 * oneUnit},
				{Destination: accountB.ToAccountId(), Amount: 123 * oneUnit},
			}),
			expected: []*TokenTransferEvent{
				mintEvent(accountA.Address(), unitsToStr(111*oneUnit), xlmProtoAsset),
				mintEvent(accountB.Address(), unitsToStr(123*oneUnit), xlmProtoAsset),
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
	hundredUnits := 100 * oneUnit
	tests := []testFixture{
		{
			name:     "successful account merge",
			tx:       someTx,
			op:       mergeAccountOp,
			opResult: mergedAccountResultWithBalance(&hundredUnits),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(accountA), protoAddressFromAccount(accountB), unitsToStr(hundredUnits), xlmProtoAsset),
			},
		},
		{
			name:     "empty account merge - no events",
			tx:       someTx,
			op:       mergeAccountOp,
			opResult: mergedAccountResultWithBalance(nil),
			expected: nil,
		},
	}
	runTokenTransferEventTests(t, tests)
}

func TestPaymentEvents(t *testing.T) {

	tests := []testFixture{
		{
			name: "G account to G account - XLM transfer",
			tx:   someTx,
			op:   paymentOp(&accountA, accountB, xlmAsset, 100*oneUnit),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(accountA), protoAddressFromAccount(accountB), unitsToStr(100*oneUnit), xlmProtoAsset),
			},
		},
		{
			name: "G account to G account - USDC transfer",
			tx:   someTx,
			op:   paymentOp(&accountA, accountB, usdcAsset, 100*oneUnit),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(accountA), protoAddressFromAccount(accountB), unitsToStr(100*oneUnit), usdcProtoAsset),
			},
		},
		{
			name: "G account to M Account - USDC transfer",
			tx:   someTx,
			op:   paymentOp(&accountA, muxedAccountB, usdcAsset, 100*oneUnit),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(accountA), protoAddressFromAccount(muxedAccountB), unitsToStr(100*oneUnit), usdcProtoAsset),
			},
		},
		{
			name: "M account to G Account - USDC transfer",
			tx:   someTx,
			op:   paymentOp(&muxedAccountA, accountB, usdcAsset, 100*oneUnit),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(muxedAccountA), protoAddressFromAccount(accountB), unitsToStr(100*oneUnit), usdcProtoAsset),
			},
		},
		{
			name: "G (issuer account) to G account - USDC mint",
			tx:   someTx,
			op:   paymentOp(&usdcAccount, accountB, usdcAsset, 100*oneUnit),
			expected: []*TokenTransferEvent{
				mintEvent(protoAddressFromAccount(accountB), unitsToStr(100*oneUnit), usdcProtoAsset),
			},
		},
		{
			name: "G (issuer account) to M account - USDC mint",
			tx:   someTx,
			op:   paymentOp(&usdcAccount, muxedAccountB, usdcAsset, 100*oneUnit),
			expected: []*TokenTransferEvent{
				mintEvent(protoAddressFromAccount(muxedAccountB), unitsToStr(100*oneUnit), usdcProtoAsset),
			},
		},
		{
			name: "G account to G (issuer account) - USDC burn",
			tx:   someTx,
			op:   paymentOp(&accountA, usdcAccount, usdcAsset, 100*oneUnit),
			expected: []*TokenTransferEvent{
				burnEvent(protoAddressFromAccount(accountA), unitsToStr(100*oneUnit), usdcProtoAsset),
			},
		},
		{
			name: "M account to G (issuer account) - USDC burn",
			tx:   someTx,
			op:   paymentOp(&muxedAccountA, usdcAccount, usdcAsset, 100*oneUnit),
			expected: []*TokenTransferEvent{
				burnEvent(protoAddressFromAccount(muxedAccountA), unitsToStr(100*oneUnit), usdcProtoAsset),
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

	tests := []testFixture{
		{
			name: "ManageBuyOffer - Buy USDC for XLM (2 claim atoms, Transfer events)",
			tx:   someTx,
			op:   manageBuyOfferOp(nil), // don't care for anything in xdr.Operation other than source account
			opResult: manageBuyOfferResult(
				[]xdr.ClaimAtom{
					// 1 USDC == 5 XLM
					generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeOrderBook, &accountA, nil, usdcAsset, oneUnit, xlmAsset, 5*oneUnit),
					generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeOrderBook, &accountB, nil, usdcAsset, 2*oneUnit, xlmAsset, 10*oneUnit),
				},
			),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(accountA), protoAddressFromAccount(someTxAccount), unitsToStr(oneUnit), usdcProtoAsset),
				transferEvent(protoAddressFromAccount(someTxAccount), protoAddressFromAccount(accountA), unitsToStr(5*oneUnit), xlmProtoAsset),

				transferEvent(protoAddressFromAccount(accountB), protoAddressFromAccount(someTxAccount), unitsToStr(2*oneUnit), usdcProtoAsset),
				transferEvent(protoAddressFromAccount(someTxAccount), protoAddressFromAccount(accountB), unitsToStr(10*oneUnit), xlmProtoAsset),
			},
		},

		{
			name: "ManageSellOffer - Sell USDC for XLM (2 claim atoms, Transfer events)",
			tx:   someTx,
			op:   manageSellOfferOp(nil), // don't care for anything in xdr.Operation other than source account
			opResult: manageSellOfferResult(
				[]xdr.ClaimAtom{
					// 1 USDC = 3 XLM
					generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeOrderBook, &accountA, nil, xlmAsset, 3*oneUnit, usdcAsset, oneUnit),
					generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeOrderBook, &accountB, nil, xlmAsset, 6*oneUnit, usdcAsset, 2*oneUnit),
				}),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(accountA), protoAddressFromAccount(someTxAccount), unitsToStr(3*oneUnit), xlmProtoAsset),
				transferEvent(protoAddressFromAccount(someTxAccount), protoAddressFromAccount(accountA), unitsToStr(oneUnit), usdcProtoAsset),

				transferEvent(protoAddressFromAccount(accountB), protoAddressFromAccount(someTxAccount), unitsToStr(6*oneUnit), xlmProtoAsset),
				transferEvent(protoAddressFromAccount(someTxAccount), protoAddressFromAccount(accountB), unitsToStr(2*oneUnit), usdcProtoAsset),
			},
		},

		{
			name: "ManageBuyOffer - Buy USDC for XLM (Source is USDC issuer, 2 claim atoms, BURN events)",
			tx:   someTx,
			op:   manageBuyOfferOp(&usdcAccount), // don't care for anything in xdr.Operation other than source account
			opResult: manageBuyOfferResult([]xdr.ClaimAtom{
				// 1 USDC == 5 XLM
				generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeOrderBook, &accountA, nil, usdcAsset, oneUnit, xlmAsset, 5*oneUnit),
				generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeOrderBook, &accountB, nil, usdcAsset, 2*oneUnit, xlmAsset, 10*oneUnit),
			}),
			expected: []*TokenTransferEvent{
				burnEvent(protoAddressFromAccount(accountA), unitsToStr(oneUnit), usdcProtoAsset),
				transferEvent(protoAddressFromAccount(usdcAccount), protoAddressFromAccount(accountA), unitsToStr(5*oneUnit), xlmProtoAsset),

				burnEvent(protoAddressFromAccount(accountB), unitsToStr(2*oneUnit), usdcProtoAsset),
				transferEvent(protoAddressFromAccount(usdcAccount), protoAddressFromAccount(accountB), unitsToStr(10*oneUnit), xlmProtoAsset),
			},
		},

		{
			name: "ManageSellOffer - Sell USDC for XLM (Source is USDC issuer, 2 claim atoms, MINT events)",
			tx:   someTx,
			op:   manageSellOfferOp(&usdcAccount), // don't care for anything in xdr.Operation other than source account
			opResult: manageSellOfferResult([]xdr.ClaimAtom{
				// 1 USDC = 3 XLM
				generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeOrderBook, &accountA, nil, xlmAsset, 3*oneUnit, usdcAsset, oneUnit),
				generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeOrderBook, &accountB, nil, xlmAsset, 6*oneUnit, usdcAsset, 2*oneUnit),

				//{SellerId: accountA.ToAccountId(), AssetSold: xlmAsset, AssetBought: usdcAsset, AmountSold: 3 * oneUnit, AmountBought: oneUnit},
				//{SellerId: accountB.ToAccountId(), AssetSold: xlmAsset, AssetBought: usdcAsset, AmountSold: 6 * oneUnit, AmountBought: 2 * oneUnit},
			}),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(accountA), protoAddressFromAccount(usdcAccount), unitsToStr(3*oneUnit), xlmProtoAsset),
				mintEvent(protoAddressFromAccount(accountA), unitsToStr(oneUnit), usdcProtoAsset),

				transferEvent(protoAddressFromAccount(accountB), protoAddressFromAccount(usdcAccount), unitsToStr(6*oneUnit), xlmProtoAsset),
				mintEvent(protoAddressFromAccount(accountB), unitsToStr(2*oneUnit), usdcProtoAsset),
			},
		},
	}
	runTokenTransferEventTests(t, tests)
}

func TestPathPaymentEvents(t *testing.T) {
	strictSendOp := func(sourceAccount *xdr.MuxedAccount, destAccount xdr.MuxedAccount, sendAsset xdr.Asset, destAsset xdr.Asset) xdr.Operation {
		return xdr.Operation{
			SourceAccount: sourceAccount,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypePathPaymentStrictSend,
				// dont really need the operation Paths for calculating events. Only need the claimAtoms in the result
				PathPaymentStrictSendOp: &xdr.PathPaymentStrictSendOp{
					SendAsset:   sendAsset,
					Destination: destAccount,
					DestAsset:   destAsset,
				},
			},
		}
	}

	strictSendResult := func(claims []xdr.ClaimAtom, destAmount xdr.Int64) xdr.OperationResult {
		return xdr.OperationResult{
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypePathPaymentStrictSend,
				PathPaymentStrictSendResult: &xdr.PathPaymentStrictSendResult{
					Success: &xdr.PathPaymentStrictSendResultSuccess{
						Offers: claims,
						Last: xdr.SimplePaymentResult{
							Amount: destAmount,
						},
					},
				},
			},
		}
	}

	strictReceiveOp := func(sourceAccount *xdr.MuxedAccount, destAccount xdr.MuxedAccount, sendAsset xdr.Asset, destAsset xdr.Asset, destAmount xdr.Int64) xdr.Operation {
		return xdr.Operation{
			SourceAccount: sourceAccount,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypePathPaymentStrictReceive,
				// dont really need the operation Paths for calculating events. Only need the claimAtoms in the result
				PathPaymentStrictReceiveOp: &xdr.PathPaymentStrictReceiveOp{
					SendAsset:   sendAsset,
					DestAsset:   destAsset,
					Destination: destAccount,
					DestAmount:  destAmount,
				},
			},
		}
	}

	strictReceiveResult := func(claims []xdr.ClaimAtom) xdr.OperationResult {
		return xdr.OperationResult{
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypePathPaymentStrictReceive,
				PathPaymentStrictReceiveResult: &xdr.PathPaymentStrictReceiveResult{
					Success: &xdr.PathPaymentStrictReceiveResultSuccess{Offers: claims},
				},
			},
		}
	}

	someXlmSellerAccount := xdr.MustMuxedAddress("GC7ERFCD7QLDFRSEPLYB3GYSWX6GYMCHLDL45N4S5Q2N5EJDOMOJ63V4")
	someEthSellerAccount := xdr.MustMuxedAddress("GAUJETIZVEP2NRYLUESJ3LS66NVCEGMON4UDCBCSBEVPIID773P2W6AY")

	tests := []testFixture{
		{
			name: "Strict Send - A (BTC Issuer) sends BTC to B as ETH - 2 Offers (BTC/XLM, XLM/ETH) - Mint and Transfer events",
			tx:   someTx,
			op:   strictSendOp(&btcAccount, accountB, btcAsset, ethAsset),
			opResult: strictSendResult(
				[]xdr.ClaimAtom{
					// source Account sold(minted) some of its BTC and bought XLM from some XLM holder
					generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeOrderBook, &someXlmSellerAccount, nil, xlmAsset, 5*oneUnit, btcAsset, oneUnit),
					// source Account then sold XLM and bought ETH from some ETH holder
					generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeOrderBook, &someEthSellerAccount, nil, ethAsset, 2*oneUnit, xlmAsset, 10*oneUnit),
				},
				// source Account then sent that ETH to destination
				100*oneUnit,
			),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(someXlmSellerAccount), protoAddressFromAccount(btcAccount), unitsToStr(5*oneUnit), xlmProtoAsset),
				mintEvent(protoAddressFromAccount(someXlmSellerAccount), unitsToStr(oneUnit), btcProtoAsset),

				transferEvent(protoAddressFromAccount(someEthSellerAccount), protoAddressFromAccount(btcAccount), unitsToStr(2*oneUnit), ethProtoAsset),
				transferEvent(protoAddressFromAccount(btcAccount), protoAddressFromAccount(someEthSellerAccount), unitsToStr(10*oneUnit), xlmProtoAsset),

				// Final transfer from source to dest of ETH
				transferEvent(protoAddressFromAccount(btcAccount), protoAddressFromAccount(accountB), unitsToStr(100*oneUnit), ethProtoAsset),
			},
		},
		{
			name: "Strict Receive - A (BTC Issuer) sends BTC to B (ETH Issuer) as ETH - 2 Offers (BTC/XLM, XLM/ETH) - Mint, Transfer and Burn events",
			tx:   someTx,
			op:   strictReceiveOp(&btcAccount, ethAccount, btcAsset, ethAsset, 6*oneUnit),
			opResult: strictReceiveResult(
				[]xdr.ClaimAtom{
					// source Account sold(minted) some of its BTC and bought XLM from some XLM holder
					generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeOrderBook, &someXlmSellerAccount, nil, xlmAsset, 5*oneUnit, btcAsset, oneUnit),
					// source Account then sold XLM and bought ETH from some ETH holder
					generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeOrderBook, &someEthSellerAccount, nil, ethAsset, 2*oneUnit, xlmAsset, 10*oneUnit),
				},
			),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(someXlmSellerAccount), protoAddressFromAccount(btcAccount), unitsToStr(5*oneUnit), xlmProtoAsset),
				mintEvent(protoAddressFromAccount(someXlmSellerAccount), unitsToStr(oneUnit), btcProtoAsset),

				transferEvent(protoAddressFromAccount(someEthSellerAccount), protoAddressFromAccount(btcAccount), unitsToStr(2*oneUnit), ethProtoAsset),
				transferEvent(protoAddressFromAccount(btcAccount), protoAddressFromAccount(someEthSellerAccount), unitsToStr(10*oneUnit), xlmProtoAsset),

				// Final burn from source to dest of ETH
				burnEvent(protoAddressFromAccount(btcAccount), unitsToStr(6*oneUnit), ethProtoAsset),
			},
		},
		{
			name: "Strict Send - A (BTC Issuer) sends BTC to B as USDC - 2 LP sweeps (BTC/ETH, ETH/USDC) - Mint and Transfer events",
			tx:   someTx,
			op:   strictSendOp(&btcAccount, accountB, btcAsset, usdcAsset),
			opResult: strictSendResult(
				[]xdr.ClaimAtom{
					// source Account traded against the BtcEth Liquidity pool and acquired Eth and sold BTC(minted)
					generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeLiquidityPool, nil, &lpBtcEthId, ethAsset, 5*oneUnit, btcAsset, oneUnit),
					// source Account then traded against the EthUsdc pool and acquired USDC
					generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeLiquidityPool, nil, &lpEthUsdcId, usdcAsset, 10*oneUnit, ethAsset, 3*oneUnit),
				},
				// source Account then sent that USDC to destination
				100*oneUnit,
			),
			expected: []*TokenTransferEvent{
				transferEvent(lpIdToStrkey(lpBtcEthId), protoAddressFromAccount(btcAccount), unitsToStr(5*oneUnit), ethProtoAsset),
				mintEvent(lpIdToStrkey(lpBtcEthId), unitsToStr(oneUnit), btcProtoAsset),

				transferEvent(lpIdToStrkey(lpEthUsdcId), protoAddressFromAccount(btcAccount), unitsToStr(10*oneUnit), usdcProtoAsset),
				transferEvent(protoAddressFromAccount(btcAccount), lpIdToStrkey(lpEthUsdcId), unitsToStr(3*oneUnit), ethProtoAsset),

				// Final transfer from source to dest of ETH
				transferEvent(protoAddressFromAccount(btcAccount), protoAddressFromAccount(accountB), unitsToStr(100*oneUnit), usdcProtoAsset),
			},
		},
		{
			name: "Strict Receive - A (BTC Issuer) sends BTC to B (USDC Issuer) as USDC - 2 LP sweeps (BTC/ETH, ETH/USDC) - Mint, Transfer and Burn events",
			tx:   someTx,
			op:   strictReceiveOp(&btcAccount, usdcAccount, btcAsset, usdcAsset, 6*oneUnit),
			opResult: strictReceiveResult(
				[]xdr.ClaimAtom{
					// source Account traded against the BtcEth Liquidity pool and acquired Eth and sold BTC(minted)
					generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeLiquidityPool, nil, &lpBtcEthId, ethAsset, 5*oneUnit, btcAsset, oneUnit),
					// source Account then traded against the EthUsdc pool and acquired USDC
					generateClaimAtom(xdr.ClaimAtomTypeClaimAtomTypeLiquidityPool, nil, &lpEthUsdcId, usdcAsset, 10*oneUnit, ethAsset, 3*oneUnit),
				},
			),
			expected: []*TokenTransferEvent{
				transferEvent(lpIdToStrkey(lpBtcEthId), protoAddressFromAccount(btcAccount), unitsToStr(5*oneUnit), ethProtoAsset),
				mintEvent(lpIdToStrkey(lpBtcEthId), unitsToStr(oneUnit), btcProtoAsset),

				transferEvent(lpIdToStrkey(lpEthUsdcId), protoAddressFromAccount(btcAccount), unitsToStr(10*oneUnit), usdcProtoAsset),
				transferEvent(protoAddressFromAccount(btcAccount), lpIdToStrkey(lpEthUsdcId), unitsToStr(3*oneUnit), ethProtoAsset),

				// Final burn from source to dest of USDC
				burnEvent(protoAddressFromAccount(btcAccount), unitsToStr(6*oneUnit), usdcProtoAsset),
			},
		},
	}
	runTokenTransferEventTests(t, tests)
}

func TestLiquidityPoolEvents(t *testing.T) {
	lpDepositOp := func(poolId xdr.PoolId, sourceAccount *xdr.MuxedAccount) xdr.Operation {
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

	lpWithdrawOp := func(poolId xdr.PoolId, sourceAccount *xdr.MuxedAccount) xdr.Operation {
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

	tests := []testFixture{
		{
			name: "Liquidity Pool Deposit Operation - New LP Creation - Transfer",
			op:   lpDepositOp(lpBtcEthId, nil), // source account = Tx account
			tx: someTxWithOperationChanges(
				xdr.LedgerEntryChanges{
					// pre = nil, post = new
					generateLpEntryCreatedChange(lpLedgerEntry(lpBtcEthId, btcAsset, ethAsset, oneUnit, 3*oneUnit)),
				}),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(someTxAccount), lpIdToStrkey(lpBtcEthId), unitsToStr(oneUnit), btcProtoAsset),
				transferEvent(protoAddressFromAccount(someTxAccount), lpIdToStrkey(lpBtcEthId), unitsToStr(3*oneUnit), ethProtoAsset),
			},
		},
		{
			name: "Liquidity Pool Deposit Operation - Existing LP Update - Transfer",
			op:   lpDepositOp(lpBtcEthId, nil), // source account = Tx account
			tx: someTxWithOperationChanges(
				xdr.LedgerEntryChanges{
					generateLpEntryChangeState(lpLedgerEntry(lpBtcEthId, btcAsset, ethAsset, oneUnit, 3*oneUnit)), // pre state
					// Increase BTC from 1 -> 5, ETH from 3 -> 10
					generateLpEntryUpdatedChange(lpLedgerEntry(lpBtcEthId, btcAsset, ethAsset, oneUnit, 3*oneUnit), 5*oneUnit, 10*oneUnit), // post state
				}),
			expected: []*TokenTransferEvent{
				transferEvent(protoAddressFromAccount(someTxAccount), lpIdToStrkey(lpBtcEthId), unitsToStr(4*oneUnit), btcProtoAsset),
				transferEvent(protoAddressFromAccount(someTxAccount), lpIdToStrkey(lpBtcEthId), unitsToStr(7*oneUnit), ethProtoAsset),
			},
		},
		{
			name: "Liquidity Pool Deposit Operation - New LP Creation - Mint",
			op:   lpDepositOp(lpBtcEthId, &btcAccount), //operation source account = BTC Asset Issuer
			tx: someTxWithOperationChanges(
				xdr.LedgerEntryChanges{
					// pre = nil, post = new
					generateLpEntryCreatedChange(lpLedgerEntry(lpBtcEthId, btcAsset, ethAsset, oneUnit, 3*oneUnit)),
				}),
			expected: []*TokenTransferEvent{
				mintEvent(lpIdToStrkey(lpBtcEthId), unitsToStr(oneUnit), btcProtoAsset),
				transferEvent(protoAddressFromAccount(btcAccount), lpIdToStrkey(lpBtcEthId), unitsToStr(3*oneUnit), ethProtoAsset),
			},
		},
		{
			name: "Liquidity Pool Withdraw Operation - Existing LP Update - Transfer",
			op:   lpWithdrawOp(lpBtcEthId, nil), // source account = Tx account
			tx: someTxWithOperationChanges(
				xdr.LedgerEntryChanges{
					generateLpEntryChangeState(lpLedgerEntry(lpBtcEthId, btcAsset, ethAsset, 5*oneUnit, 10*oneUnit)), // pre state
					// Decrease BTC from 5 -> 2, ETH from 10 -> 2
					generateLpEntryUpdatedChange(lpLedgerEntry(lpBtcEthId, btcAsset, ethAsset, 5*oneUnit, 10*oneUnit), 2*oneUnit, 2*oneUnit), // post state
				}),
			expected: []*TokenTransferEvent{
				transferEvent(lpIdToStrkey(lpBtcEthId), protoAddressFromAccount(someTxAccount), unitsToStr(3*oneUnit), btcProtoAsset),
				transferEvent(lpIdToStrkey(lpBtcEthId), protoAddressFromAccount(someTxAccount), unitsToStr(8*oneUnit), ethProtoAsset),
			},
		},
		{
			name: "Liquidity Pool Withdraw Operation - LP Removed - Transfer",
			op:   lpWithdrawOp(lpBtcEthId, nil), // source account = Tx account
			tx: someTxWithOperationChanges(
				xdr.LedgerEntryChanges{
					// pre != nil, post = nil
					generateLpEntryChangeState(lpLedgerEntry(lpBtcEthId, btcAsset, ethAsset, 5*oneUnit, 10*oneUnit)),
					generateLpEntryRemovedChange(lpBtcEthId),
				}),
			expected: []*TokenTransferEvent{
				transferEvent(lpIdToStrkey(lpBtcEthId), protoAddressFromAccount(someTxAccount), unitsToStr(5*oneUnit), btcProtoAsset),
				transferEvent(lpIdToStrkey(lpBtcEthId), protoAddressFromAccount(someTxAccount), unitsToStr(10*oneUnit), ethProtoAsset),
			},
		},
		{
			name: "Liquidity Pool Withdraw Operation - LP Removed - Burn",
			op:   lpWithdrawOp(lpBtcEthId, &ethAccount), // operation Source Account = EthIssuer
			tx: someTxWithOperationChanges(
				xdr.LedgerEntryChanges{
					// pre != nil, post = nil
					generateLpEntryChangeState(lpLedgerEntry(lpBtcEthId, btcAsset, ethAsset, 5*oneUnit, 10*oneUnit)),
					generateLpEntryRemovedChange(lpBtcEthId),
				}),
			expected: []*TokenTransferEvent{
				transferEvent(lpIdToStrkey(lpBtcEthId), protoAddressFromAccount(ethAccount), unitsToStr(5*oneUnit), btcProtoAsset),
				burnEvent(lpIdToStrkey(lpBtcEthId), unitsToStr(10*oneUnit), ethProtoAsset),
			},
		},
	}
	runTokenTransferEventTests(t, tests)
}

func TestClawbackEvents(t *testing.T) {
	clawbackOp := func(from xdr.MuxedAccount, asset xdr.Asset, amount xdr.Int64) xdr.Operation {
		return xdr.Operation{
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeClawback,
				ClawbackOp: &xdr.ClawbackOp{
					Asset:  asset,
					From:   from,
					Amount: amount,
				},
			},
		}
	}

	tests := []testFixture{
		{
			name: "Clawback XLM from some account",
			op:   clawbackOp(accountA, xlmAsset, 100*oneUnit),
			tx:   someTx,
			expected: []*TokenTransferEvent{
				clawbackEvent(protoAddressFromAccount(accountA), unitsToStr(100*oneUnit), xlmProtoAsset),
			},
		},
		{
			name: "Clawback USDC from some account",
			op:   clawbackOp(accountB, usdcAsset, oneUnit/1e2),
			tx:   someTx,
			expected: []*TokenTransferEvent{
				clawbackEvent(protoAddressFromAccount(accountB), unitsToStr(oneUnit/1e2), usdcProtoAsset),
			},
		},
	}
	runTokenTransferEventTests(t, tests)
}

func TestClawbackClaimableBalanceEvents(t *testing.T) {
	clawbackCbOp := func(cbId xdr.ClaimableBalanceId) xdr.Operation {
		return xdr.Operation{
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeClawbackClaimableBalance,
				ClawbackClaimableBalanceOp: &xdr.ClawbackClaimableBalanceOp{
					BalanceId: someBalanceId,
				},
			},
		}
	}

	tests := []testFixture{
		{
			name: "Clawback XLM claimable balance - Claimable Balance removed",
			op:   clawbackCbOp(someBalanceId),
			tx: someTxWithOperationChanges(
				xdr.LedgerEntryChanges{
					// pre != nil, post = nil
					generateCbEntryChangeState(cbLedgerEntry(someBalanceId, xlmAsset, 100*oneUnit)),
					generateCbEntryRemovedChange(someBalanceId),
				},
			),
			expected: []*TokenTransferEvent{
				clawbackEvent(cbIdToStrkey(someBalanceId), unitsToStr(100*oneUnit), xlmProtoAsset),
			},
		},
		{
			name: "Clawback USDC claimable balance - Claimable Balance removed",
			op:   clawbackCbOp(someBalanceId),
			tx: someTxWithOperationChanges(
				xdr.LedgerEntryChanges{
					// pre != nil, post = nil
					generateCbEntryChangeState(cbLedgerEntry(someBalanceId, usdcAsset, oneUnit/1e3)),
					generateCbEntryRemovedChange(someBalanceId),
				},
			),
			expected: []*TokenTransferEvent{
				clawbackEvent(cbIdToStrkey(someBalanceId), unitsToStr(oneUnit/1e3), usdcProtoAsset),
			},
		},
	}
	runTokenTransferEventTests(t, tests)
}

func TestClaimClaimableBalanceEvents(t *testing.T) {
	claimCbop := func(cbId xdr.ClaimableBalanceId, sourceAccount *xdr.MuxedAccount) xdr.Operation {
		return xdr.Operation{
			SourceAccount: sourceAccount,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeClaimClaimableBalance,
				ClaimClaimableBalanceOp: &xdr.ClaimClaimableBalanceOp{
					BalanceId: cbId,
				},
			},
		}
	}

	tests := []testFixture{
		{
			name: "Claim XLM claimable balance - Claimable Balance removed - Transfer Event",
			op:   claimCbop(someBalanceId, &accountA), // money moves from CB to the source account of operation
			tx: someTxWithOperationChanges(
				xdr.LedgerEntryChanges{
					// pre != nil, post = nil
					generateCbEntryChangeState(cbLedgerEntry(someBalanceId, xlmAsset, oneUnit)),
					generateCbEntryRemovedChange(someBalanceId),
				},
			),
			expected: []*TokenTransferEvent{
				transferEvent(cbIdToStrkey(someBalanceId), protoAddressFromAccount(accountA), unitsToStr(oneUnit), xlmProtoAsset),
			},
		},
		{
			name: "Claim USDC claimable balance back by issuer - Claimable Balance removed - Burn Event",
			op:   claimCbop(someBalanceId, &usdcAccount), // money moves from CB to the source account of operation
			tx: someTxWithOperationChanges(
				xdr.LedgerEntryChanges{
					// pre != nil, post = nil
					generateCbEntryChangeState(cbLedgerEntry(someBalanceId, usdcAsset, oneUnit/1e3)),
					generateCbEntryRemovedChange(someBalanceId),
				},
			),
			expected: []*TokenTransferEvent{
				burnEvent(cbIdToStrkey(someBalanceId), unitsToStr(oneUnit/1e3), usdcProtoAsset),
			},
		},
	}
	runTokenTransferEventTests(t, tests)
}

func TestAllowTrustAndSetTrustlineFlagsRevokeTrustlineTest(t *testing.T) {
	// For these 2 operations, it really doest matter what the operation or operation result is.
	// We are strictly going by the liquidity pool and trustlines that might be created
	// I am simply creating one here for completeness, because it needs to be passed to the EventsFromOperation() function
	trustlineRevokeOp := xdr.Operation{
		Body: xdr.OperationBody{
			Type: xdr.OperationTypeSetTrustLineFlags,
			SetTrustLineFlagsOp: &xdr.SetTrustLineFlagsOp{
				ClearFlags: xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
			},
		},
	}

	generatedCbIdForBtc, _ := ClaimableBalanceIdFromRevocation(lpBtcEthId, btcAsset, xdr.SequenceNumber(someTx.Envelope.SeqNum()), someTx.Envelope.SourceAccount().ToAccountId(), 0)
	generatedCbIdForEth, _ := ClaimableBalanceIdFromRevocation(lpBtcEthId, ethAsset, xdr.SequenceNumber(someTx.Envelope.SeqNum()), someTx.Envelope.SourceAccount().ToAccountId(), 0)

	tests := []testFixture{
		{
			name:     "Trustline Revoked - No Liquidity pool ledger entry changes - no events",
			op:       trustlineRevokeOp,
			tx:       someTxWithOperationChanges([]xdr.LedgerEntryChange{}),
			expected: nil,
		},
		{
			name: "Trustline Revoked - LP entries present, but no CB entries - no events",
			op:   trustlineRevokeOp,
			tx: someTxWithOperationChanges([]xdr.LedgerEntryChange{
				// a realistic case where the reserveA and reserveB were 0, so no CBs were created
				generateLpEntryChangeState(lpLedgerEntry(lpBtcEthId, btcAsset, ethAsset, 0, 0)),
				generateLpEntryRemovedChange(lpBtcEthId),
			}),
			expected: nil,
		},
		{
			name: "Trustline Revoked - LP entries present, CB entries for the both assets in LP - 2 transfer events",
			op:   trustlineRevokeOp,
			tx: someTxWithOperationChanges([]xdr.LedgerEntryChange{
				// 1 unit of BTC and 2 units of ETH need to go from LPId to a claimable balance
				generateLpEntryChangeState(lpLedgerEntry(lpBtcEthId, btcAsset, ethAsset, oneUnit, 2*oneUnit)),
				generateLpEntryRemovedChange(lpBtcEthId),

				// New Cb for BTC from lpBtcEthId
				generateCbEntryCreatedChange(cbLedgerEntry(generatedCbIdForBtc, btcAsset, oneUnit)),
				// New CB for ETH from lpBtcEthId
				generateCbEntryCreatedChange(cbLedgerEntry(generatedCbIdForEth, ethAsset, 2*oneUnit)),
			}),
			expected: []*TokenTransferEvent{
				transferEvent(lpIdToStrkey(lpBtcEthId), cbIdToStrkey(generatedCbIdForBtc), unitsToStr(oneUnit), btcProtoAsset),
				transferEvent(lpIdToStrkey(lpBtcEthId), cbIdToStrkey(generatedCbIdForEth), unitsToStr(2*oneUnit), ethProtoAsset),
			},
		},
		{
			name: "Trustline Revoked - LP entries present, only 1 CB entry created -1 transfer event, 1 mint event",
			op:   trustlineRevokeOp,
			tx: someTxWithOperationChanges([]xdr.LedgerEntryChange{
				// 1 unit of BTC and 2 units of ETH need to go from LPId to a claimable balance
				generateLpEntryChangeState(lpLedgerEntry(lpBtcEthId, btcAsset, ethAsset, oneUnit, 2*oneUnit)),
				generateLpEntryRemovedChange(lpBtcEthId),

				// New Cb for BTC from lpBtcEthId
				generateCbEntryCreatedChange(cbLedgerEntry(generatedCbIdForBtc, btcAsset, oneUnit)),
				// No CB created for ETH. i.e one burn event
			}),
			expected: []*TokenTransferEvent{
				transferEvent(lpIdToStrkey(lpBtcEthId), cbIdToStrkey(generatedCbIdForBtc), unitsToStr(oneUnit), btcProtoAsset),
				burnEvent(lpIdToStrkey(lpBtcEthId), unitsToStr(2*oneUnit), ethProtoAsset),
			},
		},
	}
	runTokenTransferEventTests(t, tests)
}
