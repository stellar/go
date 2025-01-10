package ingest

import (
	"testing"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestOperation(t *testing.T) {
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
						{},
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
	result map[string]interface{}
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
			result: map[string]interface{}{
				"account":          "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"funder":           "GAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABCAK",
				"funder_muxed":     "MAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPMJ2I",
				"funder_muxed_id":  uint64(123),
				"starting_balance": 2.5},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"amount":        float64(35),
				"asset_code":    "USDT",
				"asset_id":      int64(-8205667356306085451),
				"asset_issuer":  "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"asset_type":    "credit_alphanum4",
				"from":          "GAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABCAK",
				"from_muxed":    "MAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPMJ2I",
				"from_muxed_id": uint64(123),
				"to":            "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA"},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"amount":        float64(35),
				"asset_id":      int64(-5706705804583548011),
				"asset_type":    "native",
				"from":          "GAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABCAK",
				"from_muxed":    "MAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPMJ2I",
				"from_muxed_id": uint64(123),
				"to":            "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA"},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"amount":     895.14959,
				"asset_id":   int64(-5706705804583548011),
				"asset_type": "native",
				"from":       "GBT4YAEGJQ5YSFUMNKX6BPBUOCPNAIOFAVZOF6MIME2CECBMEIUXFZZN",
				"path": []Path{Path{AssetCode: "USDT",
					AssetIssuer: "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
					AssetType:   "credit_alphanum4"}},
				"source_amount":     float64(0),
				"source_asset_id":   int64(-5706705804583548011),
				"source_asset_type": "native",
				"source_max":        895.14959,
				"to":                "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA"},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"amount":            float64(76.586),
				"buying_asset_id":   int64(-5706705804583548011),
				"buying_asset_type": "native",
				"offer_id":          int64(0),
				"price":             0.514092,
				"price_r": Price{Numerator: 128523,
					Denominator: 250000},
				"selling_asset_code":   "USDT",
				"selling_asset_id":     int64(-8205667356306085451),
				"selling_asset_issuer": "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"selling_asset_type":   "credit_alphanum4"},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"amount":              float64(63.1595),
				"buying_asset_code":   "USDT",
				"buying_asset_id":     int64(-8205667356306085451),
				"buying_asset_issuer": "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"buying_asset_type":   "credit_alphanum4",
				"price":               0.0791606,
				"price_r": Price{Numerator: 99583200,
					Denominator: 1257990000},
				"selling_asset_id":   int64(-5706705804583548011),
				"selling_asset_type": "native"},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"clear_flags": []int32{1,
					2},
				"clear_flags_s": []string{"auth_required",
					"auth_revocable"},
				"high_threshold":    uint32(5),
				"home_domain":       "2019=DRA;n-test",
				"inflation_dest":    "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"low_threshold":     uint32(1),
				"master_key_weight": uint32(3),
				"med_threshold":     uint32(3),
				"set_flags":         []int32{4},
				"set_flags_s":       []string{"auth_immutable"},
				"signer_key":        "GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF",
				"signer_weight":     uint32(1)},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"asset_code":       "USSD",
				"asset_id":         int64(6690054458235693884),
				"asset_issuer":     "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"asset_type":       "credit_alphanum4",
				"limit":            5e+10,
				"trustee":          "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"trustor":          "GAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABCAK",
				"trustor_muxed":    "MAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPMJ2I",
				"trustor_muxed_id": uint64(123)},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"asset_type":        "liquidity_pool_shares",
				"limit":             5e+10,
				"liquidity_pool_id": "1c261d6c75930204a73b480c3020ab525e9be48ce93de6194cf69fb06f07452d",
				"trustor":           "GAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABCAK",
				"trustor_muxed":     "MAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPMJ2I",
				"trustor_muxed_id":  uint64(123)},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"asset_code":       "USDT",
				"asset_id":         int64(8181787832768848499),
				"asset_issuer":     "GAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABCAK",
				"asset_type":       "credit_alphanum4",
				"authorize":        true,
				"trustee":          "GAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABCAK",
				"trustee_muxed":    "MAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPMJ2I",
				"trustee_muxed_id": uint64(123),
				"trustor":          "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA"},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"account":          "GAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABCAK",
				"account_muxed":    "MAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPMJ2I",
				"account_muxed_id": uint64(123),
				"into":             "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA"},
		},
		{
			err:    nil,
			result: map[string]interface{}{},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"name":  "test",
				"value": "dmFsdWU=",
			},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"bump_to": "100",
			},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"amount":            float64(765.4501001),
				"buying_asset_id":   int64(-5706705804583548011),
				"buying_asset_type": "native",
				"offer_id":          int64(100),
				"price":             0.3496823,
				"price_r": Price{Numerator: 635863285,
					Denominator: 1818402817},
				"selling_asset_code":   "USDT",
				"selling_asset_id":     int64(-8205667356306085451),
				"selling_asset_issuer": "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"selling_asset_type":   "credit_alphanum4"},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"amount":          float64(64),
				"asset_id":        int64(-5706705804583548011),
				"asset_type":      "native",
				"destination_min": "428.0460538",
				"from":            "GAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABCAK",
				"from_muxed":      "MAISEMYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPMJ2I",
				"from_muxed_id":   uint64(123),
				"path": []Path{{AssetCode: "USDT",
					AssetIssuer: "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
					AssetType:   "credit_alphanum4"}},
				"source_amount":     0.1598182,
				"source_asset_id":   int64(-5706705804583548011),
				"source_asset_type": "native",
				"to":                "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA"},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"amount": 123456.789,
				"asset":  "USDT:GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"claimants": []Claimant{{Destination: "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
					Predicate: xdr.ClaimPredicate{Type: 0,
						AndPredicates: (*[]xdr.ClaimPredicate)(nil),
						OrPredicates:  (*[]xdr.ClaimPredicate)(nil),
						NotPredicate:  (**xdr.ClaimPredicate)(nil),
						AbsBefore:     (*xdr.Int64)(nil),
						RelBefore:     (*xdr.Int64)(nil)}}}},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"balance_id": "000000000102030405060708090000000000000000000000000000000000000000000000",
				"claimant":   "GBT4YAEGJQ5YSFUMNKX6BPBUOCPNAIOFAVZOF6MIME2CECBMEIUXFZZN"},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"sponsored_id": "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
			},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"signer_account_id": "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"signer_key":        "GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF"},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"account_id": "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA"},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"claimable_balance_id": "000000000102030405060708090000000000000000000000000000000000000000000000"},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"data_account_id": "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"data_name":       "test"},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"offer_id": int64(100)},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"trustline_account_id": "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"trustline_asset":      "USTT:GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA"},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"liquidity_pool_id": "0102030405060708090000000000000000000000000000000000000000000000"},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"amount":       0.1598182,
				"asset_code":   "USDT",
				"asset_id":     int64(-8205667356306085451),
				"asset_issuer": "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"asset_type":   "credit_alphanum4",
				"from":         "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA"},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"balance_id": "000000000102030405060708090000000000000000000000000000000000000000000000"},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"asset_code":    "USDT",
				"asset_id":      int64(-8205667356306085451),
				"asset_issuer":  "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"asset_type":    "credit_alphanum4",
				"clear_flags":   []int32{1, 2},
				"clear_flags_s": []string{"authorized", "authorized_to_maintain_liabilities"},
				"set_flags":     []int32{4},
				"set_flags_s":   []string{"clawback_enabled"},
				"trustor":       "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA"}},
		{
			err: nil,
			result: map[string]interface{}{
				"liquidity_pool_id":        "0102030405060708090000000000000000000000000000000000000000000000",
				"max_price":                1e+06,
				"max_price_r":              Price{Numerator: 1000000, Denominator: 1},
				"min_price":                1e-06,
				"min_price_r":              Price{Numerator: 1, Denominator: 1000000},
				"reserve_a_asset_code":     "USDT",
				"reserve_a_asset_id":       int64(-8205667356306085451),
				"reserve_a_asset_issuer":   "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"reserve_a_asset_type":     "credit_alphanum4",
				"reserve_a_deposit_amount": 1e-07,
				"reserve_a_max_amount":     0.0001,
				"reserve_b_asset_code":     "USDT",
				"reserve_b_asset_id":       int64(-8205667356306085451),
				"reserve_b_asset_issuer":   "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"reserve_b_asset_type":     "credit_alphanum4",
				"reserve_b_deposit_amount": 1e-07,
				"reserve_b_max_amount":     1e-05,
				"shares_received":          1e-07,
			},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"liquidity_pool_id":         "0102030405060708090000000000000000000000000000000000000000000000",
				"reserve_a_asset_code":      "USDT",
				"reserve_a_asset_id":        int64(-8205667356306085451),
				"reserve_a_asset_issuer":    "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"reserve_a_asset_type":      "credit_alphanum4",
				"reserve_a_min_amount":      1e-07,
				"reserve_a_withdraw_amount": -1e-07,
				"reserve_b_asset_code":      "USDT",
				"reserve_b_asset_id":        int64(-8205667356306085451),
				"reserve_b_asset_issuer":    "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"reserve_b_asset_type":      "credit_alphanum4",
				"reserve_b_min_amount":      1e-07,
				"reserve_b_withdraw_amount": -1e-07,
				"shares":                    4e-07,
			},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"asset_balance_changes": []map[string]interface{}{},
				"contract_id":           "CAJDIVTYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABR37",
				"function":              "HostFunctionTypeHostFunctionTypeInvokeContract",
				"ledger_key_hash":       []string{"AAAABgAAAAESNAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAA=="},
				"parameters": []map[string]string{
					{
						"type":  "Address",
						"value": "AAAAEgAAAAESNFZ4AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
					},
					{
						"type":  "Sym",
						"value": "AAAADwAAAAR0ZXN0",
					},
				},
				"parameters_decoded": []map[string]string{
					{
						"type":  "Address",
						"value": "CAJDIVTYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABR37",
					},
					{
						"type":  "Sym",
						"value": "test",
					},
				},
				"type": "invoke_contract",
			},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"address":         "CAJDIVTYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABR37",
				"contract_id":     "CAJDIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABT4W",
				"from":            "address",
				"function":        "HostFunctionTypeHostFunctionTypeCreateContract",
				"ledger_key_hash": []string{"AAAABgAAAAESNAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAA=="},
				"type":            "create_contract",
			},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"asset":           "USDT:GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"contract_id":     "CAJDIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABT4W",
				"from":            "asset",
				"function":        "HostFunctionTypeHostFunctionTypeCreateContract",
				"ledger_key_hash": []string{"AAAABgAAAAESNAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAA=="},
				"type":            "create_contract",
			},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"asset":           "USDT:GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"contract_id":     "CAJDIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABT4W",
				"from":            "asset",
				"function":        "HostFunctionTypeHostFunctionTypeCreateContractV2",
				"ledger_key_hash": []string{"AAAABgAAAAESNAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAA=="},
				"parameters": []map[string]string{
					{
						"type":  "B",
						"value": "AAAAAAAAAAE=",
					},
				},
				"parameters_decoded": []map[string]string{
					{
						"type":  "B",
						"value": "true",
					},
				},
				"type": "create_contract_v2",
			},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"function":        "HostFunctionTypeHostFunctionTypeUploadContractWasm",
				"ledger_key_hash": []string{"AAAABgAAAAESNAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAA=="},
				"type":            "upload_wasm",
			},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"extend_to":       int32(1234),
				"ledger_key_hash": []string{"AAAABgAAAAESNAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAA=="},
				"contract_id":     "CAJDIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABT4W",
				"type":            "extend_footprint_ttl",
			},
		},
		{
			err: nil,
			result: map[string]interface{}{
				"ledger_key_hash": []string{"AAAABgAAAAESNAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAA=="},
				"contract_id":     "CAJDIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABT4W",
				"type":            "restore_footprint",
			},
		},
	}

	return output
}
