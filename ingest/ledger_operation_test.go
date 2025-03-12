package ingest

import (
	"encoding/json"
	"testing"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

var testScSymbol xdr.ScSymbol = "test"
var testScBool bool = true

func TestOperation(t *testing.T) {
	o := LedgerOperation{
		OperationIndex:    int32(0),
		Operation:         operationTestInput()[1],
		Transaction:       transactionTestInput(),
		NetworkPassphrase: "",
	}

	assert.Equal(t, "GAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABCAK", o.SourceAccount())
	assert.Equal(t, int32(1), o.Type())
	assert.Equal(t, "OperationTypePayment", o.TypeString())
	assert.Equal(t, int64(131335723340009473), o.ID())

	var ok bool
	var sourceAccountMuxed string
	sourceAccountMuxed, ok = o.SourceAccountMuxed()
	assert.Equal(t, true, ok)
	assert.Equal(t, "MAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPMJ2I", sourceAccountMuxed)

	assert.Equal(t, "OperationResultCodeOpInner", o.OperationResultCode())

	var err error
	var operationTraceCode string
	operationTraceCode, err = o.OperationTraceCode()
	assert.Equal(t, nil, err)
	assert.Equal(t, "PathPaymentStrictReceiveResultCodePathPaymentStrictReceiveSuccess", operationTraceCode)
}

func TestOperationDetails(t *testing.T) {
	testOutput := resultTestOutput()
	for i, op := range operationTestInput() {
		ledgerOperation := LedgerOperation{
			OperationIndex:    int32(i),
			Operation:         op,
			Transaction:       transactionTestInput(),
			NetworkPassphrase: "",
		}

		result, err := ledgerOperation.OperationDetails()
		assert.Equal(t, testOutput[i].err, err)
		assert.Equal(t, testOutput[i].result, result)
	}
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

func transactionTestInput() *LedgerTransaction {
	testAccountAddress := "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA"
	testAccountID, _ := xdr.AddressToAccountId(testAccountAddress)
	dummyBool := true

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

	transaction := &LedgerTransaction{
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
								Footprint: xdr.LedgerFootprint{
									ReadOnly: []xdr.LedgerKey{
										{
											Type: 6,
											ContractData: &xdr.LedgerKeyContractData{
												Contract: xdr.ScAddress{
													Type:       1,
													ContractId: &xdr.Hash{0x12, 0x34},
												},
												Key: xdr.ScVal{
													Type: 0,
													B:    &dummyBool,
												},
											},
										},
									},
								},
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
					Results: &[]xdr.OperationResult{
						{
							Code: 0,
							Tr: &xdr.OperationResultTr{
								Type: 2,
								PathPaymentStrictReceiveResult: &xdr.PathPaymentStrictReceiveResult{
									Code:     0,
									Success:  &xdr.PathPaymentStrictReceiveResultSuccess{},
									NoIssuer: &xdr.Asset{},
								},
							},
						},
						{},
						{},
						{
							Code: 0,
							Tr: &xdr.OperationResultTr{
								Type: 2,
								PathPaymentStrictReceiveResult: &xdr.PathPaymentStrictReceiveResult{
									Code:     0,
									Success:  &xdr.PathPaymentStrictReceiveResultSuccess{},
									NoIssuer: &xdr.Asset{},
								},
							},
						},
						{},
						{},
						{},
						{},
						{},
						{},
						{},
						{},
						{},
						{},
						{},
						{
							Code: 0,
							Tr: &xdr.OperationResultTr{
								Type: 13,
								PathPaymentStrictSendResult: &xdr.PathPaymentStrictSendResult{
									Code: 0,
									Success: &xdr.PathPaymentStrictSendResultSuccess{
										Last: xdr.SimplePaymentResult{
											Amount: 640000000,
										},
									},
									NoIssuer: &xdr.Asset{},
								},
							},
						},
						{},
						{},
						{},
						{},
						{},
						{},
						{},
						{},
						{},
						{},
						{},
						{},
						{},
						{
							Code: 0,
							Tr: &xdr.OperationResultTr{
								Type: 22,
								LiquidityPoolDepositResult: &xdr.LiquidityPoolDepositResult{
									Code: 0,
								},
							},
						},
						{
							Code: 0,
							Tr: &xdr.OperationResultTr{
								Type: 23,
								LiquidityPoolWithdrawResult: &xdr.LiquidityPoolWithdrawResult{
									Code: 0,
								},
							},
						},
						{},
					},
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
				Operations: []xdr.OperationMeta{
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{},
					{
						Changes: []xdr.LedgerEntryChange{
							{
								Type: 3,
								State: &xdr.LedgerEntry{
									Data: xdr.LedgerEntryData{
										Type: 5,
										LiquidityPool: &xdr.LiquidityPoolEntry{
											LiquidityPoolId: xdr.PoolId{1, 2, 3, 4, 5, 6, 7, 8, 9},
											Body: xdr.LiquidityPoolEntryBody{
												Type: 0,
												ConstantProduct: &xdr.LiquidityPoolEntryConstantProduct{
													Params: xdr.LiquidityPoolConstantProductParameters{
														AssetA: xdr.Asset{
															Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
															AlphaNum4: &xdr.AlphaNum4{
																AssetCode: xdr.AssetCode4([4]byte{0x55, 0x53, 0x44, 0x54}),
																Issuer:    testAccountID,
															},
														},
														AssetB: xdr.Asset{
															Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
															AlphaNum4: &xdr.AlphaNum4{
																AssetCode: xdr.AssetCode4([4]byte{0x55, 0x53, 0x44, 0x54}),
																Issuer:    testAccountID,
															},
														},
														Fee: 1,
													},
													ReserveA:                 1,
													ReserveB:                 1,
													TotalPoolShares:          1,
													PoolSharesTrustLineCount: 1,
												},
											},
										},
									},
								},
							},
							{
								Type: 1,
								Updated: &xdr.LedgerEntry{
									Data: xdr.LedgerEntryData{
										Type: 5,
										LiquidityPool: &xdr.LiquidityPoolEntry{
											LiquidityPoolId: xdr.PoolId{1, 2, 3, 4, 5, 6, 7, 8, 9},
											Body: xdr.LiquidityPoolEntryBody{
												Type: 0,
												ConstantProduct: &xdr.LiquidityPoolEntryConstantProduct{
													Params: xdr.LiquidityPoolConstantProductParameters{
														AssetA: xdr.Asset{
															Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
															AlphaNum4: &xdr.AlphaNum4{
																AssetCode: xdr.AssetCode4([4]byte{0x55, 0x53, 0x44, 0x54}),
																Issuer:    testAccountID,
															},
														},
														AssetB: xdr.Asset{
															Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
															AlphaNum4: &xdr.AlphaNum4{
																AssetCode: xdr.AssetCode4([4]byte{0x55, 0x53, 0x44, 0x54}),
																Issuer:    testAccountID,
															},
														},
														Fee: 1,
													},
													ReserveA:                 2,
													ReserveB:                 2,
													TotalPoolShares:          2,
													PoolSharesTrustLineCount: 1,
												},
											},
										},
									},
								},
							},
						},
					},
					{
						Changes: []xdr.LedgerEntryChange{
							{
								Type: 3,
								State: &xdr.LedgerEntry{
									Data: xdr.LedgerEntryData{
										Type: 5,
										LiquidityPool: &xdr.LiquidityPoolEntry{
											LiquidityPoolId: xdr.PoolId{1, 2, 3, 4, 5, 6, 7, 8, 9},
											Body: xdr.LiquidityPoolEntryBody{
												Type: 0,
												ConstantProduct: &xdr.LiquidityPoolEntryConstantProduct{
													Params: xdr.LiquidityPoolConstantProductParameters{
														AssetA: xdr.Asset{
															Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
															AlphaNum4: &xdr.AlphaNum4{
																AssetCode: xdr.AssetCode4([4]byte{0x55, 0x53, 0x44, 0x54}),
																Issuer:    testAccountID,
															},
														},
														AssetB: xdr.Asset{
															Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
															AlphaNum4: &xdr.AlphaNum4{
																AssetCode: xdr.AssetCode4([4]byte{0x55, 0x53, 0x44, 0x54}),
																Issuer:    testAccountID,
															},
														},
														Fee: 1,
													},
													ReserveA:                 1,
													ReserveB:                 1,
													TotalPoolShares:          1,
													PoolSharesTrustLineCount: 1,
												},
											},
										},
									},
								},
							},
							{
								Type: 1,
								Updated: &xdr.LedgerEntry{
									Data: xdr.LedgerEntryData{
										Type: 5,
										LiquidityPool: &xdr.LiquidityPoolEntry{
											LiquidityPoolId: xdr.PoolId{1, 2, 3, 4, 5, 6, 7, 8, 9},
											Body: xdr.LiquidityPoolEntryBody{
												Type: 0,
												ConstantProduct: &xdr.LiquidityPoolEntryConstantProduct{
													Params: xdr.LiquidityPoolConstantProductParameters{
														AssetA: xdr.Asset{
															Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
															AlphaNum4: &xdr.AlphaNum4{
																AssetCode: xdr.AssetCode4([4]byte{0x55, 0x53, 0x44, 0x54}),
																Issuer:    testAccountID,
															},
														},
														AssetB: xdr.Asset{
															Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
															AlphaNum4: &xdr.AlphaNum4{
																AssetCode: xdr.AssetCode4([4]byte{0x55, 0x53, 0x44, 0x54}),
																Issuer:    testAccountID,
															},
														},
														Fee: 1,
													},
													ReserveA:                 2,
													ReserveB:                 2,
													TotalPoolShares:          2,
													PoolSharesTrustLineCount: 1,
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
		},
		LedgerVersion: 22,
		Ledger:        ledgerTestInput(),
		Hash:          xdr.Hash{},
	}

	return transaction
}

type testOutput struct {
	err    error
	result interface{}
}

func operationTestInput() []xdr.Operation {
	testAccountAddress := "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA"
	testAccountID, _ := xdr.AddressToAccountId(testAccountAddress)
	testAccountMuxed := testAccountID.ToMuxedAccount()

	sourceAccountAddress := "GBT4YAEGJQ5YSFUMNKX6BPBUOCPNAIOFAVZOF6MIME2CECBMEIUXFZZN"
	sourceAccountID, _ := xdr.AddressToAccountId(sourceAccountAddress)
	sourceAccountMuxed := sourceAccountID.ToMuxedAccount()

	usdtAsset := xdr.Asset{
		Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
		AlphaNum4: &xdr.AlphaNum4{
			AssetCode: xdr.AssetCode4([4]byte{0x55, 0x53, 0x44, 0x54}),
			Issuer:    testAccountID,
		},
	}
	assetCode, _ := usdtAsset.ToAssetCode("USDT")
	nativeAsset := xdr.MustNewNativeAsset()

	clearFlags := xdr.Uint32(3)
	setFlags := xdr.Uint32(4)
	masterWeight := xdr.Uint32(3)
	lowThresh := xdr.Uint32(1)
	medThresh := xdr.Uint32(3)
	highThresh := xdr.Uint32(5)
	homeDomain := xdr.String32("2019=DRA;n-test")
	signerKey, _ := xdr.NewSignerKey(xdr.SignerKeyTypeSignerKeyTypeEd25519, xdr.Uint256([32]byte{}))
	signer := xdr.Signer{
		Key:    signerKey,
		Weight: xdr.Uint32(1),
	}

	usdtChangeTrustAsset := xdr.ChangeTrustAsset{
		Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
		AlphaNum4: &xdr.AlphaNum4{
			AssetCode: xdr.AssetCode4([4]byte{0x55, 0x53, 0x53, 0x44}),
			Issuer:    testAccountID,
		},
	}

	usdtLiquidityPoolShare := xdr.ChangeTrustAsset{
		Type: xdr.AssetTypeAssetTypePoolShare,
		LiquidityPool: &xdr.LiquidityPoolParameters{
			Type: xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
			ConstantProduct: &xdr.LiquidityPoolConstantProductParameters{
				AssetA: nativeAsset,
				AssetB: usdtAsset,
				Fee:    30,
			},
		},
	}

	dataValue := xdr.DataValue([]byte{0x76, 0x61, 0x6c, 0x75, 0x65})

	testClaimant := xdr.Claimant{
		Type: xdr.ClaimantTypeClaimantTypeV0,
		V0: &xdr.ClaimantV0{
			Destination: testAccountID,
			Predicate: xdr.ClaimPredicate{
				Type: xdr.ClaimPredicateTypeClaimPredicateUnconditional,
			},
		},
	}

	claimableBalance := xdr.ClaimableBalanceId{
		Type: xdr.ClaimableBalanceIdTypeClaimableBalanceIdTypeV0,
		V0:   &xdr.Hash{1, 2, 3, 4, 5, 6, 7, 8, 9},
	}

	contractHash := xdr.Hash{0x12, 0x34, 0x56, 0x78}
	salt := [32]byte{0x12, 0x34, 0x56}
	wasm := []byte{0x12, 0x34}
	dummyBool := true

	operation := []xdr.Operation{
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeCreateAccount,
				CreateAccountOp: &xdr.CreateAccountOp{
					StartingBalance: 25000000,
					Destination:     testAccountID,
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypePayment,
				PaymentOp: &xdr.PaymentOp{
					Destination: testAccountMuxed,
					Asset:       usdtAsset,
					Amount:      350000000,
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypePayment,
				PaymentOp: &xdr.PaymentOp{
					Destination: testAccountMuxed,
					Asset:       nativeAsset,
					Amount:      350000000,
				},
			},
		},
		{
			SourceAccount: &sourceAccountMuxed,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypePathPaymentStrictReceive,
				PathPaymentStrictReceiveOp: &xdr.PathPaymentStrictReceiveOp{
					SendAsset:   nativeAsset,
					SendMax:     8951495900,
					Destination: testAccountMuxed,
					DestAsset:   nativeAsset,
					DestAmount:  8951495900,
					Path:        []xdr.Asset{usdtAsset},
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeManageSellOffer,
				ManageSellOfferOp: &xdr.ManageSellOfferOp{
					Selling: usdtAsset,
					Buying:  nativeAsset,
					Amount:  765860000,
					Price: xdr.Price{
						N: 128523,
						D: 250000,
					},
					OfferId: 0,
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeCreatePassiveSellOffer,
				CreatePassiveSellOfferOp: &xdr.CreatePassiveSellOfferOp{
					Selling: nativeAsset,
					Buying:  usdtAsset,
					Amount:  631595000,
					Price: xdr.Price{
						N: 99583200,
						D: 1257990000,
					},
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeSetOptions,
				SetOptionsOp: &xdr.SetOptionsOp{
					InflationDest: &testAccountID,
					ClearFlags:    &clearFlags,
					SetFlags:      &setFlags,
					MasterWeight:  &masterWeight,
					LowThreshold:  &lowThresh,
					MedThreshold:  &medThresh,
					HighThreshold: &highThresh,
					HomeDomain:    &homeDomain,
					Signer:        &signer,
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeChangeTrust,
				ChangeTrustOp: &xdr.ChangeTrustOp{
					Line:  usdtChangeTrustAsset,
					Limit: xdr.Int64(500000000000000000),
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeChangeTrust,
				ChangeTrustOp: &xdr.ChangeTrustOp{
					Line:  usdtLiquidityPoolShare,
					Limit: xdr.Int64(500000000000000000),
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeAllowTrust,
				AllowTrustOp: &xdr.AllowTrustOp{
					Trustor:   testAccountID,
					Asset:     assetCode,
					Authorize: xdr.Uint32(1),
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type:        xdr.OperationTypeAccountMerge,
				Destination: &testAccountMuxed,
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeInflation,
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeManageData,
				ManageDataOp: &xdr.ManageDataOp{
					DataName:  "test",
					DataValue: &dataValue,
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeBumpSequence,
				BumpSequenceOp: &xdr.BumpSequenceOp{
					BumpTo: xdr.SequenceNumber(100),
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeManageBuyOffer,
				ManageBuyOfferOp: &xdr.ManageBuyOfferOp{
					Selling:   usdtAsset,
					Buying:    nativeAsset,
					BuyAmount: 7654501001,
					Price: xdr.Price{
						N: 635863285,
						D: 1818402817,
					},
					OfferId: 100,
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypePathPaymentStrictSend,
				PathPaymentStrictSendOp: &xdr.PathPaymentStrictSendOp{
					SendAsset:   nativeAsset,
					SendAmount:  1598182,
					Destination: testAccountMuxed,
					DestAsset:   nativeAsset,
					DestMin:     4280460538,
					Path:        []xdr.Asset{usdtAsset},
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeCreateClaimableBalance,
				CreateClaimableBalanceOp: &xdr.CreateClaimableBalanceOp{
					Asset:     usdtAsset,
					Amount:    1234567890000,
					Claimants: []xdr.Claimant{testClaimant},
				},
			},
		},
		{
			SourceAccount: &sourceAccountMuxed,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeClaimClaimableBalance,
				ClaimClaimableBalanceOp: &xdr.ClaimClaimableBalanceOp{
					BalanceId: claimableBalance,
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeBeginSponsoringFutureReserves,
				BeginSponsoringFutureReservesOp: &xdr.BeginSponsoringFutureReservesOp{
					SponsoredId: testAccountID,
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeRevokeSponsorship,
				RevokeSponsorshipOp: &xdr.RevokeSponsorshipOp{
					Type: xdr.RevokeSponsorshipTypeRevokeSponsorshipSigner,
					Signer: &xdr.RevokeSponsorshipOpSigner{
						AccountId: testAccountID,
						SignerKey: signer.Key,
					},
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeRevokeSponsorship,
				RevokeSponsorshipOp: &xdr.RevokeSponsorshipOp{
					Type: xdr.RevokeSponsorshipTypeRevokeSponsorshipLedgerEntry,
					LedgerKey: &xdr.LedgerKey{
						Type: xdr.LedgerEntryTypeAccount,
						Account: &xdr.LedgerKeyAccount{
							AccountId: testAccountID,
						},
					},
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeRevokeSponsorship,
				RevokeSponsorshipOp: &xdr.RevokeSponsorshipOp{
					Type: xdr.RevokeSponsorshipTypeRevokeSponsorshipLedgerEntry,
					LedgerKey: &xdr.LedgerKey{
						Type: xdr.LedgerEntryTypeClaimableBalance,
						ClaimableBalance: &xdr.LedgerKeyClaimableBalance{
							BalanceId: claimableBalance,
						},
					},
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeRevokeSponsorship,
				RevokeSponsorshipOp: &xdr.RevokeSponsorshipOp{
					Type: xdr.RevokeSponsorshipTypeRevokeSponsorshipLedgerEntry,
					LedgerKey: &xdr.LedgerKey{
						Type: xdr.LedgerEntryTypeData,
						Data: &xdr.LedgerKeyData{
							AccountId: testAccountID,
							DataName:  "test",
						},
					},
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeRevokeSponsorship,
				RevokeSponsorshipOp: &xdr.RevokeSponsorshipOp{
					Type: xdr.RevokeSponsorshipTypeRevokeSponsorshipLedgerEntry,
					LedgerKey: &xdr.LedgerKey{
						Type: xdr.LedgerEntryTypeOffer,
						Offer: &xdr.LedgerKeyOffer{
							SellerId: testAccountID,
							OfferId:  100,
						},
					},
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeRevokeSponsorship,
				RevokeSponsorshipOp: &xdr.RevokeSponsorshipOp{
					Type: xdr.RevokeSponsorshipTypeRevokeSponsorshipLedgerEntry,
					LedgerKey: &xdr.LedgerKey{
						Type: xdr.LedgerEntryTypeTrustline,
						TrustLine: &xdr.LedgerKeyTrustLine{
							AccountId: testAccountID,
							Asset: xdr.TrustLineAsset{
								Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
								AlphaNum4: &xdr.AlphaNum4{
									AssetCode: xdr.AssetCode4([4]byte{0x55, 0x53, 0x54, 0x54}),
									Issuer:    testAccountID,
								},
							},
						},
					},
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeRevokeSponsorship,
				RevokeSponsorshipOp: &xdr.RevokeSponsorshipOp{
					Type: xdr.RevokeSponsorshipTypeRevokeSponsorshipLedgerEntry,
					LedgerKey: &xdr.LedgerKey{
						Type: xdr.LedgerEntryTypeLiquidityPool,
						LiquidityPool: &xdr.LedgerKeyLiquidityPool{
							LiquidityPoolId: xdr.PoolId{1, 2, 3, 4, 5, 6, 7, 8, 9},
						},
					},
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeClawback,
				ClawbackOp: &xdr.ClawbackOp{
					Asset:  usdtAsset,
					From:   testAccountMuxed,
					Amount: 1598182,
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeClawbackClaimableBalance,
				ClawbackClaimableBalanceOp: &xdr.ClawbackClaimableBalanceOp{
					BalanceId: claimableBalance,
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeSetTrustLineFlags,
				SetTrustLineFlagsOp: &xdr.SetTrustLineFlagsOp{
					Trustor:    testAccountID,
					Asset:      usdtAsset,
					SetFlags:   setFlags,
					ClearFlags: clearFlags,
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeLiquidityPoolDeposit,
				LiquidityPoolDepositOp: &xdr.LiquidityPoolDepositOp{
					LiquidityPoolId: xdr.PoolId{1, 2, 3, 4, 5, 6, 7, 8, 9},
					MaxAmountA:      1000,
					MaxAmountB:      100,
					MinPrice: xdr.Price{
						N: 1,
						D: 1000000,
					},
					MaxPrice: xdr.Price{
						N: 1000000,
						D: 1,
					},
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeLiquidityPoolWithdraw,
				LiquidityPoolWithdrawOp: &xdr.LiquidityPoolWithdrawOp{
					LiquidityPoolId: xdr.PoolId{1, 2, 3, 4, 5, 6, 7, 8, 9},
					Amount:          4,
					MinAmountA:      1,
					MinAmountB:      1,
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeInvokeHostFunction,
				InvokeHostFunctionOp: &xdr.InvokeHostFunctionOp{
					HostFunction: xdr.HostFunction{
						Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
						InvokeContract: &xdr.InvokeContractArgs{
							ContractAddress: xdr.ScAddress{
								Type:       xdr.ScAddressTypeScAddressTypeContract,
								ContractId: &contractHash,
							},
							FunctionName: "test",
							Args:         []xdr.ScVal{},
						},
					},
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeInvokeHostFunction,
				InvokeHostFunctionOp: &xdr.InvokeHostFunctionOp{
					HostFunction: xdr.HostFunction{
						Type: xdr.HostFunctionTypeHostFunctionTypeCreateContract,
						CreateContract: &xdr.CreateContractArgs{
							ContractIdPreimage: xdr.ContractIdPreimage{
								Type: xdr.ContractIdPreimageTypeContractIdPreimageFromAddress,
								FromAddress: &xdr.ContractIdPreimageFromAddress{
									Address: xdr.ScAddress{
										Type:       xdr.ScAddressTypeScAddressTypeContract,
										ContractId: &contractHash,
									},
									Salt: salt,
								},
							},
							Executable: xdr.ContractExecutable{},
						},
					},
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeInvokeHostFunction,
				InvokeHostFunctionOp: &xdr.InvokeHostFunctionOp{
					HostFunction: xdr.HostFunction{
						Type: xdr.HostFunctionTypeHostFunctionTypeCreateContract,
						CreateContract: &xdr.CreateContractArgs{
							ContractIdPreimage: xdr.ContractIdPreimage{
								Type:      xdr.ContractIdPreimageTypeContractIdPreimageFromAsset,
								FromAsset: &usdtAsset,
							},
							Executable: xdr.ContractExecutable{},
						},
					},
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeInvokeHostFunction,
				InvokeHostFunctionOp: &xdr.InvokeHostFunctionOp{
					HostFunction: xdr.HostFunction{
						Type: xdr.HostFunctionTypeHostFunctionTypeCreateContractV2,
						CreateContractV2: &xdr.CreateContractArgsV2{
							ContractIdPreimage: xdr.ContractIdPreimage{
								Type:      xdr.ContractIdPreimageTypeContractIdPreimageFromAsset,
								FromAsset: &usdtAsset,
							},
							Executable: xdr.ContractExecutable{},
							ConstructorArgs: []xdr.ScVal{
								{
									Type: xdr.ScValTypeScvBool,
									B:    &dummyBool,
								},
							},
						},
					},
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeInvokeHostFunction,
				InvokeHostFunctionOp: &xdr.InvokeHostFunctionOp{
					HostFunction: xdr.HostFunction{
						Type: xdr.HostFunctionTypeHostFunctionTypeUploadContractWasm,
						Wasm: &wasm,
					},
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeExtendFootprintTtl,
				ExtendFootprintTtlOp: &xdr.ExtendFootprintTtlOp{
					Ext: xdr.ExtensionPoint{
						V: 0,
					},
					ExtendTo: 1234,
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeRestoreFootprint,
				RestoreFootprintOp: &xdr.RestoreFootprintOp{
					Ext: xdr.ExtensionPoint{
						V: 0,
					},
				},
			},
		},
	}

	return operation
}

func resultTestOutput() []testOutput {
	output := []testOutput{
		{
			err: nil,
			result: CreateAccountDetail{
				Account:         "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				Funder:          "GAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABCAK",
				FunderMuxed:     "MAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPMJ2I",
				FunderMuxedID:   uint64(123),
				StartingBalance: int64(25000000)},
		},
		{
			err: nil,
			result: PaymentDetail{
				Amount:      int64(350000000),
				AssetCode:   "USDT",
				AssetIssuer: "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				AssetType:   "credit_alphanum4",
				From:        "GAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABCAK",
				FromMuxed:   "MAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPMJ2I",
				FromMuxedID: uint64(123),
				To:          "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA"},
		},
		{
			err: nil,
			result: PaymentDetail{
				Amount:      int64(350000000),
				AssetType:   "native",
				From:        "GAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABCAK",
				FromMuxed:   "MAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPMJ2I",
				FromMuxedID: uint64(123),
				To:          "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA"},
		},
		{
			err: nil,
			result: PathPaymentStrictReceiveDetail{
				Amount:    int64(8951495900),
				AssetType: "native",
				From:      "GBT4YAEGJQ5YSFUMNKX6BPBUOCPNAIOFAVZOF6MIME2CECBMEIUXFZZN",
				Path: []Path{
					{
						AssetCode:   "USDT",
						AssetIssuer: "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
						AssetType:   "credit_alphanum4",
					},
				},
				SourceAmount:    int64(0),
				SourceAssetType: "native",
				SourceMax:       int64(8951495900),
				To:              "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA"},
		},
		{
			err: nil,
			result: ManageSellOffer{
				Amount:             int64(765860000),
				BuyingAssetType:    "native",
				OfferID:            int64(0),
				Price:              0.514092,
				PriceN:             128523,
				PriceD:             250000,
				SellingAssetCode:   "USDT",
				SellingAssetIssuer: "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				SellingAssetType:   "credit_alphanum4"},
		},
		{
			err: nil,
			result: CreatePassiveSellOfferDetail{
				Amount:            int64(631595000),
				BuyingAssetCode:   "USDT",
				BuyingAssetIssuer: "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				BuyingAssetType:   "credit_alphanum4",
				Price:             0.0791606,
				PriceN:            99583200,
				PriceD:            1257990000,
				SellingAssetType:  "native"},
		},
		{
			err: nil,
			result: SetOptionsDetails{
				ClearFlags:           []int32{1, 2},
				ClearFlagsString:     []string{"auth_required", "auth_revocable"},
				HighThreshold:        uint32(5),
				HomeDomain:           "2019=DRA;n-test",
				InflationDestination: "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				LowThreshold:         uint32(1),
				MasterKeyWeight:      uint32(3),
				MediumThreshold:      uint32(3),
				SetFlags:             []int32{4},
				SetFlagsString:       []string{"auth_immutable"},
				SignerKey:            "GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF",
				SignerWeight:         uint32(1)},
		},
		{
			err: nil,
			result: ChangeTrustDetail{
				AssetCode:      "USSD",
				AssetIssuer:    "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				AssetType:      "credit_alphanum4",
				Limit:          int64(500000000000000000),
				Trustee:        "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				Trustor:        "GAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABCAK",
				TrustorMuxed:   "MAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPMJ2I",
				TrustorMuxedID: uint64(123)},
		},
		{
			err: nil,
			result: ChangeTrustDetail{
				AssetType:       "liquidity_pool_shares",
				Limit:           int64(500000000000000000),
				LiquidityPoolID: "HCYdbHWTAgSnO0gMMCCrUl6b5IzpPeYZTPafsG8HRS0=",
				Trustor:         "GAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABCAK",
				TrustorMuxed:    "MAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPMJ2I",
				TrustorMuxedID:  uint64(123)},
		},
		{
			err: nil,
			result: AllowTrustDetail{
				AssetCode:      "USDT",
				AssetIssuer:    "GAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABCAK",
				AssetType:      "credit_alphanum4",
				Authorize:      true,
				Trustee:        "GAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABCAK",
				TrusteeMuxed:   "MAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPMJ2I",
				TrusteeMuxedID: uint64(123),
				Trustor:        "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA"},
		},
		{
			err: nil,
			result: AccountMergeDetail{
				Account:        "GAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABCAK",
				AccountMuxed:   "MAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPMJ2I",
				AccountMuxedID: uint64(123),
				Into:           "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA"},
		},
		{
			err:    nil,
			result: InflationDetail{},
		},
		{
			err: nil,
			result: ManageDataDetail{
				Name:  "test",
				Value: "dmFsdWU=",
			},
		},
		{
			err: nil,
			result: BumpSequenceDetails{
				BumpTo: int64(100),
			},
		},
		{
			err: nil,
			result: ManageBuyOffer{
				Amount:             int64(7654501001),
				BuyingAssetType:    "native",
				OfferID:            int64(100),
				Price:              0.3496823,
				PriceN:             635863285,
				PriceD:             1818402817,
				SellingAssetCode:   "USDT",
				SellingAssetIssuer: "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				SellingAssetType:   "credit_alphanum4"},
		},
		{
			err: nil,
			result: PathPaymentStrictSendDetail{
				Amount:         int64(640000000),
				AssetType:      "native",
				DestinationMin: 4280460538,
				From:           "GAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABCAK",
				FromMuxed:      "MAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPMJ2I",
				FromMuxedID:    uint64(123),
				Path: []Path{
					{
						AssetCode:   "USDT",
						AssetIssuer: "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
						AssetType:   "credit_alphanum4",
					},
				},
				SourceAmount:    int64(1598182),
				SourceAssetType: "native",
				To:              "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA"},
		},
		{
			err: nil,
			result: CreateClaimableBalanceDetail{
				Amount:      int64(1234567890000),
				AssetCode:   "USDT",
				AssetIssuer: "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				AssetType:   "credit_alphanum4",
				Claimants: []Claimant{
					{
						Destination: "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
						Predicate: xdr.ClaimPredicate{
							Type:          0,
							AndPredicates: (*[]xdr.ClaimPredicate)(nil),
							OrPredicates:  (*[]xdr.ClaimPredicate)(nil),
							NotPredicate:  (**xdr.ClaimPredicate)(nil),
							AbsBefore:     (*xdr.Int64)(nil),
							RelBefore:     (*xdr.Int64)(nil),
						},
					},
				},
			},
		},
		{
			err: nil,
			result: ClaimClaimableBalanceDetail{
				BalanceID: "AAAAAAECAwQFBgcICQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
				Claimant:  "GBT4YAEGJQ5YSFUMNKX6BPBUOCPNAIOFAVZOF6MIME2CECBMEIUXFZZN",
			},
		},
		{
			err: nil,
			result: BeginSponsoringFutureReservesDetail{
				SponsoredID: "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
			},
		},
		{
			err: nil,
			result: RevokeSponsorshipDetail{
				SignerAccountID: "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				SignerKey:       "GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF",
			},
		},
		{
			err: nil,
			result: RevokeSponsorshipDetail{
				LedgerKeyDetails: LedgerKeyDetail{
					AccountID: "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				},
			},
		},
		{
			err: nil,
			result: RevokeSponsorshipDetail{
				LedgerKeyDetails: LedgerKeyDetail{
					ClaimableBalanceID: "000000000102030405060708090000000000000000000000000000000000000000000000",
				},
			},
		},
		{
			err: nil,
			result: RevokeSponsorshipDetail{
				LedgerKeyDetails: LedgerKeyDetail{
					DataAccountID: "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
					DataName:      "test",
				},
			},
		},
		{
			err: nil,
			result: RevokeSponsorshipDetail{
				LedgerKeyDetails: LedgerKeyDetail{
					OfferID: int64(100),
				},
			},
		},
		{
			err: nil,
			result: RevokeSponsorshipDetail{
				LedgerKeyDetails: LedgerKeyDetail{
					TrustlineAccountID:   "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
					TrustlineAssetCode:   "USTT",
					TrustlineAssetIssuer: "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
					TrustlineAssetType:   "credit_alphanum4",
				},
			},
		},
		{
			err: nil,
			result: RevokeSponsorshipDetail{
				LedgerKeyDetails: LedgerKeyDetail{
					LiquidityPoolID: "AQIDBAUGBwgJAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
				},
			},
		},
		{
			err: nil,
			result: ClawbackDetail{
				Amount:      int64(1598182),
				AssetCode:   "USDT",
				AssetIssuer: "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				AssetType:   "credit_alphanum4",
				From:        "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
			},
		},
		{
			err: nil,
			result: ClawbackClaimableBalanceDetail{
				BalanceID: "AAAAAAECAwQFBgcICQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
			},
		},
		{
			err: nil,
			result: SetTrustlineFlagsDetail{
				AssetCode:        "USDT",
				AssetIssuer:      "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				AssetType:        "credit_alphanum4",
				ClearFlags:       []int32{1, 2},
				ClearFlagsString: []string{"authorized", "authorized_to_maintain_liabilities"},
				SetFlags:         []int32{4},
				SetFlagsString:   []string{"clawback_enabled"},
				Trustor:          "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
			},
		},
		{
			err: nil,
			result: LiquidityPoolDepositDetail{
				LiquidityPoolID: "AQIDBAUGBwgJAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
				MaxPrice:        1e+06,
				MaxPriceN:       1000000,
				MaxPriceD:       1,
				MinPrice:        1e-06,
				MinPriceN:       1,
				MinPriceD:       1000000,
				ReserveAssetA: ReserveAsset{
					AssetCode:     "USDT",
					AssetIssuer:   "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
					AssetType:     "credit_alphanum4",
					DepositAmount: int64(1),
					MaxAmount:     int64(1000),
				},
				ReserveAssetB: ReserveAsset{
					AssetCode:     "USDT",
					AssetIssuer:   "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
					AssetType:     "credit_alphanum4",
					DepositAmount: 1,
					MaxAmount:     int64(100),
				},
				SharesReceived: 1,
			},
		},
		{
			err: nil,
			result: LiquidityPoolWithdrawDetail{
				LiquidityPoolID: "AQIDBAUGBwgJAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
				ReserveAssetA: ReserveAsset{
					AssetCode:      "USDT",
					AssetIssuer:    "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
					AssetType:      "credit_alphanum4",
					MinAmount:      int64(1),
					WithdrawAmount: int64(-1),
				},

				ReserveAssetB: ReserveAsset{
					AssetCode:      "USDT",
					AssetIssuer:    "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
					AssetType:      "credit_alphanum4",
					MinAmount:      int64(1),
					WithdrawAmount: int64(-1),
				},
				Shares: int64(4),
			},
		},
		{
			err: nil,
			result: InvokeHostFunctionDetail{
				AssetBalanceChanges: []BalanceChangeDetail{},
				ContractID:          "CAJDIVTYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABR37",
				Function:            "HostFunctionTypeHostFunctionTypeInvokeContract",
				LedgerKeyHash:       []string{"AAAABgAAAAESNAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAA=="},
				Parameters: []interface{}{
					json.RawMessage{
						0x7b, 0x22, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x22, 0x3a, 0x22, 0x43, 0x41, 0x4a, 0x44,
						0x49, 0x56, 0x54, 0x59, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41,
						0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41,
						0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41,
						0x42, 0x52, 0x33, 0x37, 0x22, 0x7d,
					},
					json.RawMessage{0x7b, 0x22, 0x73, 0x79, 0x6d, 0x62, 0x6f, 0x6c, 0x22, 0x3a, 0x22, 0x74, 0x65, 0x73, 0x74, 0x22, 0x7d},
				},
				Type: "invoke_contract",
			},
		},
		{
			err: nil,
			result: InvokeHostFunctionDetail{
				Address:       "CAJDIVTYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABR37",
				ContractID:    "CAJDIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABT4W",
				From:          "address",
				Function:      "HostFunctionTypeHostFunctionTypeCreateContract",
				LedgerKeyHash: []string{"AAAABgAAAAESNAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAA=="},
				Type:          "create_contract",
			},
		},
		{
			err: nil,
			result: InvokeHostFunctionDetail{
				AssetCode:     "USDT",
				AssetIssuer:   "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				AssetType:     "credit_alphanum4",
				ContractID:    "CAJDIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABT4W",
				From:          "asset",
				Function:      "HostFunctionTypeHostFunctionTypeCreateContract",
				LedgerKeyHash: []string{"AAAABgAAAAESNAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAA=="},
				Type:          "create_contract",
			},
		},
		{
			err: nil,
			result: InvokeHostFunctionDetail{
				AssetCode:     "USDT",
				AssetIssuer:   "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				AssetType:     "credit_alphanum4",
				ContractID:    "CAJDIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABT4W",
				From:          "asset",
				Function:      "HostFunctionTypeHostFunctionTypeCreateContractV2",
				LedgerKeyHash: []string{"AAAABgAAAAESNAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAA=="},
				Parameters: []interface{}{
					json.RawMessage{0x7b, 0x22, 0x62, 0x6f, 0x6f, 0x6c, 0x22, 0x3a, 0x74, 0x72, 0x75, 0x65, 0x7d},
				},
				Type: "create_contract_v2",
			},
		},
		{
			err: nil,
			result: InvokeHostFunctionDetail{
				Function:      "HostFunctionTypeHostFunctionTypeUploadContractWasm",
				LedgerKeyHash: []string{"AAAABgAAAAESNAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAA=="},
				Type:          "upload_wasm",
			},
		},
		{
			err: nil,
			result: ExtendFootprintTtlDetail{
				ExtendTo:      uint32(1234),
				LedgerKeyHash: []string{"AAAABgAAAAESNAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAA=="},
				ContractID:    "CAJDIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABT4W",
				Type:          "extend_footprint_ttl",
			},
		},
		{
			err: nil,
			result: RestoreFootprintDetail{
				LedgerKeyHash: []string{"AAAABgAAAAESNAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAA=="},
				ContractID:    "CAJDIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABT4W",
				Type:          "restore_footprint",
			},
		},
	}

	return output
}
