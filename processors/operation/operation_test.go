package operation

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/processors/utils"
	"github.com/stellar/go/xdr"
)

var testClaimantDetails = utils.Claimant{
	Destination: testAccount1Address,
	Predicate:   xdr.ClaimPredicate{},
}

var usdtAssetPath = utils.Path{
	AssetType:   "credit_alphanum4",
	AssetCode:   "USDT",
	AssetIssuer: testAccount4Address,
}

var genericSourceAccount, _ = xdr.NewMuxedAccount(xdr.CryptoKeyTypeKeyTypeEd25519, xdr.Uint256([32]byte{}))

var genericBumpOperation = xdr.Operation{
	SourceAccount: &genericSourceAccount,
	Body: xdr.OperationBody{
		Type:           xdr.OperationTypeBumpSequence,
		BumpSequenceOp: &xdr.BumpSequenceOp{},
	},
}

// a selection of hardcoded accounts with their IDs and addresses
var testAccount1Address = "GCEODJVUUVYVFD5KT4TOEDTMXQ76OPFOQC2EMYYMLPXQCUVPOB6XRWPQ"
var testAccount1ID, _ = xdr.AddressToAccountId(testAccount1Address)

var testAccount3Address = "GBT4YAEGJQ5YSFUMNKX6BPBUOCPNAIOFAVZOF6MIME2CECBMEIUXFZZN"
var testAccount3ID, _ = xdr.AddressToAccountId(testAccount3Address)
var testAccount3 = testAccount3ID.ToMuxedAccount()

var testAccount4Address = "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA"
var testAccount4ID, _ = xdr.AddressToAccountId(testAccount4Address)
var testAccount4 = testAccount4ID.ToMuxedAccount()

var lpAssetA = xdr.Asset{
	Type: xdr.AssetTypeAssetTypeNative,
}

var lpAssetB = xdr.Asset{
	Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
	AlphaNum4: &xdr.AlphaNum4{
		AssetCode: xdr.AssetCode4([4]byte{0x55, 0x53, 0x53, 0x44}),
		Issuer:    testAccount4ID,
	},
}

var genericTxMeta = utils.CreateSampleTxMeta(29, lpAssetA, lpAssetB)

var genericBumpOperationEnvelope = xdr.TransactionV1Envelope{
	Tx: xdr.Transaction{
		SourceAccount: genericSourceAccount,
		Memo:          xdr.Memo{},
		Operations: []xdr.Operation{
			genericBumpOperation,
		},
		Ext: xdr.TransactionExt{
			V: 0,
			SorobanData: &xdr.SorobanTransactionData{
				Ext: xdr.ExtensionPoint{
					V: 0,
				},
				Resources: xdr.SorobanResources{
					Footprint: xdr.LedgerFootprint{
						ReadOnly:  []xdr.LedgerKey{},
						ReadWrite: []xdr.LedgerKey{},
					},
				},
				ResourceFee: 100,
			},
		},
	},
}

var genericLedgerTransaction = ingest.LedgerTransaction{
	Index: 1,
	Envelope: xdr.TransactionEnvelope{
		Type: xdr.EnvelopeTypeEnvelopeTypeTx,
		V1:   &genericBumpOperationEnvelope,
	},
	Result: utils.CreateSampleResultMeta(true, 10).Result,
	UnsafeMeta: xdr.TransactionMeta{
		V:  1,
		V1: genericTxMeta,
	},
}

var usdtAsset = xdr.Asset{
	Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
	AlphaNum4: &xdr.AlphaNum4{
		AssetCode: xdr.AssetCode4([4]byte{0x55, 0x53, 0x44, 0x54}),
		Issuer:    testAccount4ID,
	},
}

var genericClaimableBalance = xdr.ClaimableBalanceId{
	Type: xdr.ClaimableBalanceIdTypeClaimableBalanceIdTypeV0,
	V0:   &xdr.Hash{1, 2, 3, 4, 5, 6, 7, 8, 9},
}

var testClaimant = xdr.Claimant{
	Type: xdr.ClaimantTypeClaimantTypeV0,
	V0: &xdr.ClaimantV0{
		Destination: testAccount1ID,
		Predicate: xdr.ClaimPredicate{
			Type: xdr.ClaimPredicateTypeClaimPredicateUnconditional,
		},
	},
}

var nativeAsset = xdr.MustNewNativeAsset()

var usdtChangeTrustAsset = xdr.ChangeTrustAsset{
	Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
	AlphaNum4: &xdr.AlphaNum4{
		AssetCode: xdr.AssetCode4([4]byte{0x55, 0x53, 0x53, 0x44}),
		Issuer:    testAccount4ID,
	},
}

var usdtTrustLineAsset = xdr.TrustLineAsset{
	Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
	AlphaNum4: &xdr.AlphaNum4{
		AssetCode: xdr.AssetCode4([4]byte{0x55, 0x53, 0x54, 0x54}),
		Issuer:    testAccount3ID,
	},
}

var usdtLiquidityPoolShare = xdr.ChangeTrustAsset{
	Type: xdr.AssetTypeAssetTypePoolShare,
	LiquidityPool: &xdr.LiquidityPoolParameters{
		Type: xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
		ConstantProduct: &xdr.LiquidityPoolConstantProductParameters{
			AssetA: lpAssetA,
			AssetB: lpAssetB,
			Fee:    30,
		},
	},
}

var genericCloseTime = time.Unix(0, 0)

func TestTransformOperation(t *testing.T) {
	type operationInput struct {
		operation        xdr.Operation
		index            int32
		transaction      ingest.LedgerTransaction
		ledgerClosedMeta xdr.LedgerCloseMeta
	}
	type transformTest struct {
		input      operationInput
		wantOutput OperationOutput
		wantErr    error
	}
	genericInput := operationInput{
		operation:   genericBumpOperation,
		index:       1,
		transaction: genericLedgerTransaction,
	}

	negativeOpTypeInput := genericInput
	negativeOpTypeEnvelope := genericBumpOperationEnvelope
	negativeOpTypeEnvelope.Tx.Operations[0].Body.Type = xdr.OperationType(-1)
	negativeOpTypeInput.operation.Body.Type = xdr.OperationType(-1)
	negativeOpTypeInput.transaction.Envelope.V1 = &negativeOpTypeEnvelope

	unknownOpTypeInput := genericInput
	unknownOpTypeEnvelope := genericBumpOperationEnvelope
	unknownOpTypeEnvelope.Tx.Operations[0].Body.Type = xdr.OperationType(99)
	unknownOpTypeInput.operation.Body.Type = xdr.OperationType(99)
	unknownOpTypeInput.transaction.Envelope.V1 = &unknownOpTypeEnvelope

	tests := []transformTest{
		{
			negativeOpTypeInput,
			OperationOutput{},
			fmt.Errorf("the operation type (-1) is negative for  operation 1 (operation id=4098)"),
		},
		{
			unknownOpTypeInput,
			OperationOutput{},
			fmt.Errorf("unknown operation type: "),
		},
	}
	hardCodedInputTransaction, err := makeOperationTestInput()
	assert.NoError(t, err)
	hardCodedOutputArray := makeOperationTestOutputs()
	hardCodedInputLedgerCloseMeta := makeLedgerCloseMeta()

	for i, op := range hardCodedInputTransaction.Envelope.Operations() {
		tests = append(tests, transformTest{
			input:      operationInput{op, int32(i), hardCodedInputTransaction, hardCodedInputLedgerCloseMeta},
			wantOutput: hardCodedOutputArray[i],
			wantErr:    nil,
		})
	}

	for _, test := range tests {
		actualOutput, actualError := TransformOperation(test.input.operation, test.input.index, test.input.transaction, 0, test.input.ledgerClosedMeta, "")
		assert.Equal(t, test.wantErr, actualError)
		assert.Equal(t, test.wantOutput, actualOutput)
	}
}

func makeLedgerCloseMeta() (ledgerCloseMeta xdr.LedgerCloseMeta) {
	return xdr.LedgerCloseMeta{
		V: 0,
		V0: &xdr.LedgerCloseMetaV0{
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Header: xdr.LedgerHeader{
					ScpValue: xdr.StellarValue{
						CloseTime: 0,
					},
					LedgerSeq: 0,
				},
			},
		},
	}
}

// Creates a single transaction that contains one of every operation type
func makeOperationTestInput() (inputTransaction ingest.LedgerTransaction, err error) {
	inputTransaction = genericLedgerTransaction
	inputEnvelope := genericBumpOperationEnvelope

	inputEnvelope.Tx.SourceAccount = testAccount3
	hardCodedInflationDest := testAccount4ID

	hardCodedTrustAsset, err := usdtAsset.ToAssetCode("USDT")
	if err != nil {
		return
	}

	contractHash := xdr.Hash{}
	salt := [32]byte{}
	assetCode := [12]byte{}
	assetIssuer := xdr.Uint256{}
	wasm := []byte{}
	dummyBool := true

	hardCodedClearFlags := xdr.Uint32(3)
	hardCodedSetFlags := xdr.Uint32(4)
	hardCodedMasterWeight := xdr.Uint32(3)
	hardCodedLowThresh := xdr.Uint32(1)
	hardCodedMedThresh := xdr.Uint32(3)
	hardCodedHighThresh := xdr.Uint32(5)
	hardCodedHomeDomain := xdr.String32("2019=DRA;n-test")
	hardCodedSignerKey, err := xdr.NewSignerKey(xdr.SignerKeyTypeSignerKeyTypeEd25519, xdr.Uint256([32]byte{}))
	if err != nil {
		return
	}

	hardCodedSigner := xdr.Signer{
		Key:    hardCodedSignerKey,
		Weight: xdr.Uint32(1),
	}

	hardCodedClaimableBalance := genericClaimableBalance
	hardCodedClaimant := testClaimant
	hardCodedDataValue := xdr.DataValue([]byte{0x76, 0x61, 0x6c, 0x75, 0x65})
	hardCodedSequenceNumber := xdr.SequenceNumber(100)
	inputOperations := []xdr.Operation{
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeCreateAccount,
				CreateAccountOp: &xdr.CreateAccountOp{
					StartingBalance: 25000000,
					Destination:     testAccount4ID,
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypePayment,
				PaymentOp: &xdr.PaymentOp{
					Destination: testAccount4,
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
					Destination: testAccount4,
					Asset:       nativeAsset,
					Amount:      350000000,
				},
			},
		},
		{
			SourceAccount: &testAccount3,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypePathPaymentStrictReceive,
				PathPaymentStrictReceiveOp: &xdr.PathPaymentStrictReceiveOp{
					SendAsset:   nativeAsset,
					SendMax:     8951495900,
					Destination: testAccount4,
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
					InflationDest: &hardCodedInflationDest,
					ClearFlags:    &hardCodedClearFlags,
					SetFlags:      &hardCodedSetFlags,
					MasterWeight:  &hardCodedMasterWeight,
					LowThreshold:  &hardCodedLowThresh,
					MedThreshold:  &hardCodedMedThresh,
					HighThreshold: &hardCodedHighThresh,
					HomeDomain:    &hardCodedHomeDomain,
					Signer:        &hardCodedSigner,
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
					Trustor:   testAccount4ID,
					Asset:     hardCodedTrustAsset,
					Authorize: xdr.Uint32(1),
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type:        xdr.OperationTypeAccountMerge,
				Destination: &testAccount4,
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
					DataValue: &hardCodedDataValue,
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeBumpSequence,
				BumpSequenceOp: &xdr.BumpSequenceOp{
					BumpTo: hardCodedSequenceNumber,
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
					Destination: testAccount4,
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
					Claimants: []xdr.Claimant{hardCodedClaimant},
				},
			},
		},
		{
			SourceAccount: &testAccount3,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeClaimClaimableBalance,
				ClaimClaimableBalanceOp: &xdr.ClaimClaimableBalanceOp{
					BalanceId: hardCodedClaimableBalance,
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeBeginSponsoringFutureReserves,
				BeginSponsoringFutureReservesOp: &xdr.BeginSponsoringFutureReservesOp{
					SponsoredId: testAccount4ID,
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
						AccountId: testAccount4ID,
						SignerKey: hardCodedSigner.Key,
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
							AccountId: testAccount4ID,
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
							BalanceId: hardCodedClaimableBalance,
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
							AccountId: testAccount4ID,
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
							SellerId: testAccount3ID,
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
							AccountId: testAccount3ID,
							Asset:     usdtTrustLineAsset,
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
					From:   testAccount4,
					Amount: 1598182,
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeClawbackClaimableBalance,
				ClawbackClaimableBalanceOp: &xdr.ClawbackClaimableBalanceOp{
					BalanceId: hardCodedClaimableBalance,
				},
			},
		},
		{
			SourceAccount: nil,
			Body: xdr.OperationBody{
				Type: xdr.OperationTypeSetTrustLineFlags,
				SetTrustLineFlagsOp: &xdr.SetTrustLineFlagsOp{
					Trustor:    testAccount4ID,
					Asset:      usdtAsset,
					SetFlags:   hardCodedSetFlags,
					ClearFlags: hardCodedClearFlags,
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
								Type: xdr.ContractIdPreimageTypeContractIdPreimageFromAsset,
								FromAsset: &xdr.Asset{
									Type: xdr.AssetTypeAssetTypeCreditAlphanum12,
									AlphaNum12: &xdr.AlphaNum12{
										AssetCode: assetCode,
										Issuer: xdr.AccountId{
											Type:    xdr.PublicKeyTypePublicKeyTypeEd25519,
											Ed25519: &assetIssuer,
										},
									},
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
						Type: xdr.HostFunctionTypeHostFunctionTypeCreateContractV2,
						CreateContractV2: &xdr.CreateContractArgsV2{
							ContractIdPreimage: xdr.ContractIdPreimage{
								Type: xdr.ContractIdPreimageTypeContractIdPreimageFromAsset,
								FromAsset: &xdr.Asset{
									Type: xdr.AssetTypeAssetTypeCreditAlphanum12,
									AlphaNum12: &xdr.AlphaNum12{
										AssetCode: assetCode,
										Issuer: xdr.AccountId{
											Type:    xdr.PublicKeyTypePublicKeyTypeEd25519,
											Ed25519: &assetIssuer,
										},
									},
								},
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
	inputEnvelope.Tx.Operations = inputOperations
	results := []xdr.OperationResult{
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeCreateAccount,
				CreateAccountResult: &xdr.CreateAccountResult{
					Code: xdr.CreateAccountResultCodeCreateAccountSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypePayment,
				PaymentResult: &xdr.PaymentResult{
					Code: xdr.PaymentResultCodePaymentSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypePayment,
				PaymentResult: &xdr.PaymentResult{
					Code: xdr.PaymentResultCodePaymentSuccess,
				},
			},
		},
		// There needs to be a true result for path payment receive and send
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypePathPaymentStrictReceive,
				PathPaymentStrictReceiveResult: &xdr.PathPaymentStrictReceiveResult{
					Code: xdr.PathPaymentStrictReceiveResultCodePathPaymentStrictReceiveSuccess,
					Success: &xdr.PathPaymentStrictReceiveResultSuccess{
						Last: xdr.SimplePaymentResult{Amount: 8946764349},
					},
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeManageSellOffer,
				ManageSellOfferResult: &xdr.ManageSellOfferResult{
					Code: xdr.ManageSellOfferResultCodeManageSellOfferSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeManageSellOffer,
				ManageSellOfferResult: &xdr.ManageSellOfferResult{
					Code: xdr.ManageSellOfferResultCodeManageSellOfferSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeSetOptions,
				SetOptionsResult: &xdr.SetOptionsResult{
					Code: xdr.SetOptionsResultCodeSetOptionsSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeChangeTrust,
				ChangeTrustResult: &xdr.ChangeTrustResult{
					Code: xdr.ChangeTrustResultCodeChangeTrustSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeChangeTrust,
				ChangeTrustResult: &xdr.ChangeTrustResult{
					Code: xdr.ChangeTrustResultCodeChangeTrustSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeAllowTrust,
				AllowTrustResult: &xdr.AllowTrustResult{
					Code: xdr.AllowTrustResultCodeAllowTrustSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeAccountMerge,
				AccountMergeResult: &xdr.AccountMergeResult{
					Code: xdr.AccountMergeResultCodeAccountMergeSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeInflation,
				InflationResult: &xdr.InflationResult{
					Code: xdr.InflationResultCodeInflationSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeManageData,
				ManageDataResult: &xdr.ManageDataResult{
					Code: xdr.ManageDataResultCodeManageDataSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeBumpSequence,
				BumpSeqResult: &xdr.BumpSequenceResult{
					Code: xdr.BumpSequenceResultCodeBumpSequenceSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeManageBuyOffer,
				ManageBuyOfferResult: &xdr.ManageBuyOfferResult{
					Code: xdr.ManageBuyOfferResultCodeManageBuyOfferSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypePathPaymentStrictSend,
				PathPaymentStrictSendResult: &xdr.PathPaymentStrictSendResult{
					Code: xdr.PathPaymentStrictSendResultCodePathPaymentStrictSendSuccess,
					Success: &xdr.PathPaymentStrictSendResultSuccess{
						Last: xdr.SimplePaymentResult{Amount: 4334043858},
					},
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeCreateClaimableBalance,
				CreateClaimableBalanceResult: &xdr.CreateClaimableBalanceResult{
					Code: xdr.CreateClaimableBalanceResultCodeCreateClaimableBalanceSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeClaimClaimableBalance,
				ClaimClaimableBalanceResult: &xdr.ClaimClaimableBalanceResult{
					Code: xdr.ClaimClaimableBalanceResultCodeClaimClaimableBalanceSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeBeginSponsoringFutureReserves,
				BeginSponsoringFutureReservesResult: &xdr.BeginSponsoringFutureReservesResult{
					Code: xdr.BeginSponsoringFutureReservesResultCodeBeginSponsoringFutureReservesSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeRevokeSponsorship,
				RevokeSponsorshipResult: &xdr.RevokeSponsorshipResult{
					Code: xdr.RevokeSponsorshipResultCodeRevokeSponsorshipSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeRevokeSponsorship,
				RevokeSponsorshipResult: &xdr.RevokeSponsorshipResult{
					Code: xdr.RevokeSponsorshipResultCodeRevokeSponsorshipSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeRevokeSponsorship,
				RevokeSponsorshipResult: &xdr.RevokeSponsorshipResult{
					Code: xdr.RevokeSponsorshipResultCodeRevokeSponsorshipSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeRevokeSponsorship,
				RevokeSponsorshipResult: &xdr.RevokeSponsorshipResult{
					Code: xdr.RevokeSponsorshipResultCodeRevokeSponsorshipSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeRevokeSponsorship,
				RevokeSponsorshipResult: &xdr.RevokeSponsorshipResult{
					Code: xdr.RevokeSponsorshipResultCodeRevokeSponsorshipSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeRevokeSponsorship,
				RevokeSponsorshipResult: &xdr.RevokeSponsorshipResult{
					Code: xdr.RevokeSponsorshipResultCodeRevokeSponsorshipSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeRevokeSponsorship,
				RevokeSponsorshipResult: &xdr.RevokeSponsorshipResult{
					Code: xdr.RevokeSponsorshipResultCodeRevokeSponsorshipSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeClawback,
				ClawbackResult: &xdr.ClawbackResult{
					Code: xdr.ClawbackResultCodeClawbackSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeClawbackClaimableBalance,
				ClawbackClaimableBalanceResult: &xdr.ClawbackClaimableBalanceResult{
					Code: xdr.ClawbackClaimableBalanceResultCodeClawbackClaimableBalanceSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeSetTrustLineFlags,
				SetTrustLineFlagsResult: &xdr.SetTrustLineFlagsResult{
					Code: xdr.SetTrustLineFlagsResultCodeSetTrustLineFlagsSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeLiquidityPoolDeposit,
				LiquidityPoolDepositResult: &xdr.LiquidityPoolDepositResult{
					Code: xdr.LiquidityPoolDepositResultCodeLiquidityPoolDepositSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeLiquidityPoolWithdraw,
				LiquidityPoolWithdrawResult: &xdr.LiquidityPoolWithdrawResult{
					Code: xdr.LiquidityPoolWithdrawResultCodeLiquidityPoolWithdrawSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeInvokeHostFunction,
				InvokeHostFunctionResult: &xdr.InvokeHostFunctionResult{
					Code: xdr.InvokeHostFunctionResultCodeInvokeHostFunctionSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeInvokeHostFunction,
				InvokeHostFunctionResult: &xdr.InvokeHostFunctionResult{
					Code: xdr.InvokeHostFunctionResultCodeInvokeHostFunctionSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeInvokeHostFunction,
				InvokeHostFunctionResult: &xdr.InvokeHostFunctionResult{
					Code: xdr.InvokeHostFunctionResultCodeInvokeHostFunctionSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeInvokeHostFunction,
				InvokeHostFunctionResult: &xdr.InvokeHostFunctionResult{
					Code: xdr.InvokeHostFunctionResultCodeInvokeHostFunctionSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeInvokeHostFunction,
				InvokeHostFunctionResult: &xdr.InvokeHostFunctionResult{
					Code: xdr.InvokeHostFunctionResultCodeInvokeHostFunctionSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeInvokeHostFunction,
				InvokeHostFunctionResult: &xdr.InvokeHostFunctionResult{
					Code: xdr.InvokeHostFunctionResultCodeInvokeHostFunctionSuccess,
				},
			},
		},
		{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeInvokeHostFunction,
				InvokeHostFunctionResult: &xdr.InvokeHostFunctionResult{
					Code: xdr.InvokeHostFunctionResultCodeInvokeHostFunctionSuccess,
				},
			},
		},
	}
	inputTransaction.Result.Result.Result.Results = &results
	inputTransaction.Envelope.V1 = &inputEnvelope
	return
}

func makeOperationTestOutputs() (transformedOperations []OperationOutput) {
	hardCodedSourceAccountAddress := testAccount3Address
	hardCodedDestAccountAddress := testAccount4Address
	hardCodedLedgerClose := genericCloseTime.UTC()
	var nilStringArray []string

	transformedOperations = []OperationOutput{
		{
			SourceAccount: hardCodedSourceAccountAddress,
			Type:          0,
			TypeString:    "create_account",
			TransactionID: 4096,
			OperationID:   4097,
			OperationDetails: map[string]interface{}{
				"account":          hardCodedDestAccountAddress,
				"funder":           hardCodedSourceAccountAddress,
				"starting_balance": 2.5,
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "CreateAccountResultCodeCreateAccountSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"account":          hardCodedDestAccountAddress,
				"funder":           hardCodedSourceAccountAddress,
				"starting_balance": 2.5,
			},
		},
		{
			Type:          1,
			TypeString:    "payment",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4098,
			OperationDetails: map[string]interface{}{
				"from":         hardCodedSourceAccountAddress,
				"to":           hardCodedDestAccountAddress,
				"amount":       35.0,
				"asset_code":   "USDT",
				"asset_type":   "credit_alphanum4",
				"asset_issuer": hardCodedDestAccountAddress,
				"asset_id":     int64(-8205667356306085451),
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "PaymentResultCodePaymentSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"from":         hardCodedSourceAccountAddress,
				"to":           hardCodedDestAccountAddress,
				"amount":       35.0,
				"asset_code":   "USDT",
				"asset_type":   "credit_alphanum4",
				"asset_issuer": hardCodedDestAccountAddress,
				"asset_id":     int64(-8205667356306085451),
			},
		},
		{
			Type:          1,
			TypeString:    "payment",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4099,
			OperationDetails: map[string]interface{}{
				"from":       hardCodedSourceAccountAddress,
				"to":         hardCodedDestAccountAddress,
				"amount":     35.0,
				"asset_type": "native",
				"asset_id":   int64(-5706705804583548011),
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "PaymentResultCodePaymentSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"from":       hardCodedSourceAccountAddress,
				"to":         hardCodedDestAccountAddress,
				"amount":     35.0,
				"asset_type": "native",
				"asset_id":   int64(-5706705804583548011),
			},
		},
		{
			Type:          2,
			TypeString:    "path_payment_strict_receive",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4100,
			OperationDetails: map[string]interface{}{
				"from":              hardCodedSourceAccountAddress,
				"to":                hardCodedDestAccountAddress,
				"source_amount":     894.6764349,
				"source_max":        895.14959,
				"amount":            895.14959,
				"source_asset_type": "native",
				"source_asset_id":   int64(-5706705804583548011),
				"asset_type":        "native",
				"asset_id":          int64(-5706705804583548011),
				"path":              []utils.Path{usdtAssetPath},
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "PathPaymentStrictReceiveResultCodePathPaymentStrictReceiveSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"from":              hardCodedSourceAccountAddress,
				"to":                hardCodedDestAccountAddress,
				"source_amount":     894.6764349,
				"source_max":        895.14959,
				"amount":            895.14959,
				"source_asset_type": "native",
				"source_asset_id":   int64(-5706705804583548011),
				"asset_type":        "native",
				"asset_id":          int64(-5706705804583548011),
				"path":              []utils.Path{usdtAssetPath},
			},
		},
		{
			Type:          3,
			TypeString:    "manage_sell_offer",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4101,
			OperationDetails: map[string]interface{}{
				"price":    0.514092,
				"amount":   76.586,
				"offer_id": int64(0.0),
				"price_r": utils.Price{
					Numerator:   128523,
					Denominator: 250000,
				},
				"selling_asset_code":   "USDT",
				"selling_asset_type":   "credit_alphanum4",
				"selling_asset_issuer": hardCodedDestAccountAddress,
				"selling_asset_id":     int64(-8205667356306085451),
				"buying_asset_type":    "native",
				"buying_asset_id":      int64(-5706705804583548011),
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "ManageSellOfferResultCodeManageSellOfferSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"price":    0.514092,
				"amount":   76.586,
				"offer_id": int64(0.0),
				"price_r": utils.Price{
					Numerator:   128523,
					Denominator: 250000,
				},
				"selling_asset_code":   "USDT",
				"selling_asset_type":   "credit_alphanum4",
				"selling_asset_issuer": hardCodedDestAccountAddress,
				"selling_asset_id":     int64(-8205667356306085451),
				"buying_asset_type":    "native",
				"buying_asset_id":      int64(-5706705804583548011),
			},
		},
		{
			Type:          4,
			TypeString:    "create_passive_sell_offer",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4102,
			OperationDetails: map[string]interface{}{
				"amount": 63.1595,
				"price":  0.0791606,
				"price_r": utils.Price{
					Numerator:   99583200,
					Denominator: 1257990000,
				},
				"buying_asset_code":   "USDT",
				"buying_asset_type":   "credit_alphanum4",
				"buying_asset_issuer": hardCodedDestAccountAddress,
				"buying_asset_id":     int64(-8205667356306085451),
				"selling_asset_type":  "native",
				"selling_asset_id":    int64(-5706705804583548011),
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "ManageSellOfferResultCodeManageSellOfferSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"amount": 63.1595,
				"price":  0.0791606,
				"price_r": utils.Price{
					Numerator:   99583200,
					Denominator: 1257990000,
				},
				"buying_asset_code":   "USDT",
				"buying_asset_type":   "credit_alphanum4",
				"buying_asset_issuer": hardCodedDestAccountAddress,
				"buying_asset_id":     int64(-8205667356306085451),
				"selling_asset_type":  "native",
				"selling_asset_id":    int64(-5706705804583548011),
			},
		},
		{
			Type:          5,
			TypeString:    "set_options",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4103,
			OperationDetails: map[string]interface{}{
				"inflation_dest":    hardCodedDestAccountAddress,
				"clear_flags":       []int32{1, 2},
				"clear_flags_s":     []string{"auth_required", "auth_revocable"},
				"set_flags":         []int32{4},
				"set_flags_s":       []string{"auth_immutable"},
				"master_key_weight": uint32(3),
				"low_threshold":     uint32(1),
				"med_threshold":     uint32(3),
				"high_threshold":    uint32(5),
				"home_domain":       "2019=DRA;n-test",
				"signer_key":        "GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF",
				"signer_weight":     uint32(1),
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "SetOptionsResultCodeSetOptionsSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"inflation_dest":    hardCodedDestAccountAddress,
				"clear_flags":       []int32{1, 2},
				"clear_flags_s":     []string{"auth_required", "auth_revocable"},
				"set_flags":         []int32{4},
				"set_flags_s":       []string{"auth_immutable"},
				"master_key_weight": uint32(3),
				"low_threshold":     uint32(1),
				"med_threshold":     uint32(3),
				"high_threshold":    uint32(5),
				"home_domain":       "2019=DRA;n-test",
				"signer_key":        "GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF",
				"signer_weight":     uint32(1),
			},
		},
		{
			Type:          6,
			TypeString:    "change_trust",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4104,
			OperationDetails: map[string]interface{}{
				"trustor":      hardCodedSourceAccountAddress,
				"trustee":      hardCodedDestAccountAddress,
				"limit":        50000000000.0,
				"asset_code":   "USSD",
				"asset_type":   "credit_alphanum4",
				"asset_issuer": hardCodedDestAccountAddress,
				"asset_id":     int64(6690054458235693884),
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "ChangeTrustResultCodeChangeTrustSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"trustor":      hardCodedSourceAccountAddress,
				"trustee":      hardCodedDestAccountAddress,
				"limit":        50000000000.0,
				"asset_code":   "USSD",
				"asset_type":   "credit_alphanum4",
				"asset_issuer": hardCodedDestAccountAddress,
				"asset_id":     int64(6690054458235693884),
			},
		},
		{
			Type:          6,
			TypeString:    "change_trust",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4105,
			OperationDetails: map[string]interface{}{
				"trustor":           hardCodedSourceAccountAddress,
				"limit":             50000000000.0,
				"asset_type":        "liquidity_pool_shares",
				"liquidity_pool_id": "185a6b384c651552ba09b32851b79f5f6ab61e80883d303f52bea1406a4923f0",
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "ChangeTrustResultCodeChangeTrustSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"trustor":           hardCodedSourceAccountAddress,
				"limit":             50000000000.0,
				"asset_type":        "liquidity_pool_shares",
				"liquidity_pool_id": "185a6b384c651552ba09b32851b79f5f6ab61e80883d303f52bea1406a4923f0",
			},
		},
		{
			Type:          7,
			TypeString:    "allow_trust",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4106,
			OperationDetails: map[string]interface{}{
				"trustee":      hardCodedSourceAccountAddress,
				"trustor":      hardCodedDestAccountAddress,
				"authorize":    true,
				"asset_code":   "USDT",
				"asset_type":   "credit_alphanum4",
				"asset_issuer": hardCodedSourceAccountAddress,
				"asset_id":     int64(8485542065083974675),
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "AllowTrustResultCodeAllowTrustSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"trustee":      hardCodedSourceAccountAddress,
				"trustor":      hardCodedDestAccountAddress,
				"authorize":    true,
				"asset_code":   "USDT",
				"asset_type":   "credit_alphanum4",
				"asset_issuer": hardCodedSourceAccountAddress,
				"asset_id":     int64(8485542065083974675),
			},
		},
		{
			Type:          8,
			TypeString:    "account_merge",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4107,
			OperationDetails: map[string]interface{}{
				"account": hardCodedSourceAccountAddress,
				"into":    hardCodedDestAccountAddress,
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "AccountMergeResultCodeAccountMergeSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"account": hardCodedSourceAccountAddress,
				"into":    hardCodedDestAccountAddress,
			},
		},
		{
			Type:                 9,
			TypeString:           "inflation",
			SourceAccount:        hardCodedSourceAccountAddress,
			TransactionID:        4096,
			OperationID:          4108,
			OperationDetails:     map[string]interface{}{},
			ClosedAt:             hardCodedLedgerClose,
			OperationResultCode:  "OperationResultCodeOpInner",
			OperationTraceCode:   "InflationResultCodeInflationSuccess",
			LedgerSequence:       0,
			OperationDetailsJSON: map[string]interface{}{},
		},
		{
			Type:          10,
			TypeString:    "manage_data",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4109,
			OperationDetails: map[string]interface{}{
				"name":  "test",
				"value": base64.StdEncoding.EncodeToString([]byte{0x76, 0x61, 0x6c, 0x75, 0x65}),
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "ManageDataResultCodeManageDataSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"name":  "test",
				"value": base64.StdEncoding.EncodeToString([]byte{0x76, 0x61, 0x6c, 0x75, 0x65}),
			},
		},
		{
			Type:          11,
			TypeString:    "bump_sequence",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4110,
			OperationDetails: map[string]interface{}{
				"bump_to": "100",
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "BumpSequenceResultCodeBumpSequenceSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"bump_to": "100",
			},
		},
		{
			Type:          12,
			TypeString:    "manage_buy_offer",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4111,
			OperationDetails: map[string]interface{}{
				"price":  0.3496823,
				"amount": 765.4501001,
				"price_r": utils.Price{
					Numerator:   635863285,
					Denominator: 1818402817,
				},
				"selling_asset_code":   "USDT",
				"selling_asset_type":   "credit_alphanum4",
				"selling_asset_issuer": hardCodedDestAccountAddress,
				"selling_asset_id":     int64(-8205667356306085451),
				"buying_asset_type":    "native",
				"buying_asset_id":      int64(-5706705804583548011),
				"offer_id":             int64(100),
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "ManageBuyOfferResultCodeManageBuyOfferSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"price":  0.3496823,
				"amount": 765.4501001,
				"price_r": utils.Price{
					Numerator:   635863285,
					Denominator: 1818402817,
				},
				"selling_asset_code":   "USDT",
				"selling_asset_type":   "credit_alphanum4",
				"selling_asset_issuer": hardCodedDestAccountAddress,
				"selling_asset_id":     int64(-8205667356306085451),
				"buying_asset_type":    "native",
				"buying_asset_id":      int64(-5706705804583548011),
				"offer_id":             int64(100),
			},
		},
		{
			Type:          13,
			TypeString:    "path_payment_strict_send",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4112,
			OperationDetails: map[string]interface{}{
				"from":              hardCodedSourceAccountAddress,
				"to":                hardCodedDestAccountAddress,
				"source_amount":     0.1598182,
				"destination_min":   "428.0460538",
				"amount":            433.4043858,
				"path":              []utils.Path{usdtAssetPath},
				"source_asset_type": "native",
				"source_asset_id":   int64(-5706705804583548011),
				"asset_type":        "native",
				"asset_id":          int64(-5706705804583548011),
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "PathPaymentStrictSendResultCodePathPaymentStrictSendSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"from":              hardCodedSourceAccountAddress,
				"to":                hardCodedDestAccountAddress,
				"source_amount":     0.1598182,
				"destination_min":   "428.0460538",
				"amount":            433.4043858,
				"path":              []utils.Path{usdtAssetPath},
				"source_asset_type": "native",
				"source_asset_id":   int64(-5706705804583548011),
				"asset_type":        "native",
				"asset_id":          int64(-5706705804583548011),
			},
		},
		{
			Type:          14,
			TypeString:    "create_claimable_balance",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4113,
			OperationDetails: map[string]interface{}{
				"asset":     "USDT:GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"amount":    123456.789,
				"claimants": []utils.Claimant{testClaimantDetails},
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "CreateClaimableBalanceResultCodeCreateClaimableBalanceSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"asset":     "USDT:GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"amount":    123456.789,
				"claimants": []utils.Claimant{testClaimantDetails},
			},
		},
		{
			Type:          15,
			TypeString:    "claim_claimable_balance",
			SourceAccount: testAccount3Address,
			TransactionID: 4096,
			OperationID:   4114,
			OperationDetails: map[string]interface{}{
				"claimant":   hardCodedSourceAccountAddress,
				"balance_id": "000000000102030405060708090000000000000000000000000000000000000000000000",
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "ClaimClaimableBalanceResultCodeClaimClaimableBalanceSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"claimant":   hardCodedSourceAccountAddress,
				"balance_id": "000000000102030405060708090000000000000000000000000000000000000000000000",
			},
		},
		{
			Type:          16,
			TypeString:    "begin_sponsoring_future_reserves",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4115,
			OperationDetails: map[string]interface{}{
				"sponsored_id": hardCodedDestAccountAddress,
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "BeginSponsoringFutureReservesResultCodeBeginSponsoringFutureReservesSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"sponsored_id": hardCodedDestAccountAddress,
			},
		},
		{
			Type:          18,
			TypeString:    "revoke_sponsorship",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4116,
			OperationDetails: map[string]interface{}{
				"signer_account_id": hardCodedDestAccountAddress,
				"signer_key":        "GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF",
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "RevokeSponsorshipResultCodeRevokeSponsorshipSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"signer_account_id": hardCodedDestAccountAddress,
				"signer_key":        "GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF",
			},
		},
		{
			Type:          18,
			TypeString:    "revoke_sponsorship",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4117,
			OperationDetails: map[string]interface{}{
				"account_id": hardCodedDestAccountAddress,
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "RevokeSponsorshipResultCodeRevokeSponsorshipSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"account_id": hardCodedDestAccountAddress,
			},
		},
		{
			Type:          18,
			TypeString:    "revoke_sponsorship",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4118,
			OperationDetails: map[string]interface{}{
				"claimable_balance_id": "000000000102030405060708090000000000000000000000000000000000000000000000",
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "RevokeSponsorshipResultCodeRevokeSponsorshipSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"claimable_balance_id": "000000000102030405060708090000000000000000000000000000000000000000000000",
			},
		},
		{
			Type:          18,
			TypeString:    "revoke_sponsorship",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4119,
			OperationDetails: map[string]interface{}{
				"data_account_id": hardCodedDestAccountAddress,
				"data_name":       "test",
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "RevokeSponsorshipResultCodeRevokeSponsorshipSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"data_account_id": hardCodedDestAccountAddress,
				"data_name":       "test",
			},
		},
		{
			Type:          18,
			TypeString:    "revoke_sponsorship",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4120,
			OperationDetails: map[string]interface{}{
				"offer_id": int64(100),
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "RevokeSponsorshipResultCodeRevokeSponsorshipSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"offer_id": int64(100),
			},
		},
		{
			Type:          18,
			TypeString:    "revoke_sponsorship",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4121,
			OperationDetails: map[string]interface{}{
				"trustline_account_id": testAccount3Address,
				"trustline_asset":      "USTT:GBT4YAEGJQ5YSFUMNKX6BPBUOCPNAIOFAVZOF6MIME2CECBMEIUXFZZN",
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "RevokeSponsorshipResultCodeRevokeSponsorshipSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"trustline_account_id": testAccount3Address,
				"trustline_asset":      "USTT:GBT4YAEGJQ5YSFUMNKX6BPBUOCPNAIOFAVZOF6MIME2CECBMEIUXFZZN",
			},
		},
		{
			Type:          18,
			TypeString:    "revoke_sponsorship",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4122,
			OperationDetails: map[string]interface{}{
				"liquidity_pool_id": "0102030405060708090000000000000000000000000000000000000000000000",
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "RevokeSponsorshipResultCodeRevokeSponsorshipSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"liquidity_pool_id": "0102030405060708090000000000000000000000000000000000000000000000",
			},
		},
		{
			Type:          19,
			TypeString:    "clawback",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4123,
			OperationDetails: map[string]interface{}{
				"from":         hardCodedDestAccountAddress,
				"amount":       0.1598182,
				"asset_code":   "USDT",
				"asset_issuer": "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"asset_type":   "credit_alphanum4",
				"asset_id":     int64(-8205667356306085451),
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "ClawbackResultCodeClawbackSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"from":         hardCodedDestAccountAddress,
				"amount":       0.1598182,
				"asset_code":   "USDT",
				"asset_issuer": "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"asset_type":   "credit_alphanum4",
				"asset_id":     int64(-8205667356306085451),
			},
		},
		{
			Type:          20,
			TypeString:    "clawback_claimable_balance",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4124,
			OperationDetails: map[string]interface{}{
				"balance_id": "000000000102030405060708090000000000000000000000000000000000000000000000",
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "ClawbackClaimableBalanceResultCodeClawbackClaimableBalanceSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"balance_id": "000000000102030405060708090000000000000000000000000000000000000000000000",
			},
		},
		{
			Type:          21,
			TypeString:    "set_trust_line_flags",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4125,
			OperationDetails: map[string]interface{}{
				"asset_code":    "USDT",
				"asset_issuer":  "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"asset_type":    "credit_alphanum4",
				"asset_id":      int64(-8205667356306085451),
				"trustor":       testAccount4Address,
				"clear_flags":   []int32{1, 2},
				"clear_flags_s": []string{"authorized", "authorized_to_maintain_liabilities"},
				"set_flags":     []int32{4},
				"set_flags_s":   []string{"clawback_enabled"},
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "SetTrustLineFlagsResultCodeSetTrustLineFlagsSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"asset_code":    "USDT",
				"asset_issuer":  "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"asset_type":    "credit_alphanum4",
				"asset_id":      int64(-8205667356306085451),
				"trustor":       testAccount4Address,
				"clear_flags":   []int32{1, 2},
				"clear_flags_s": []string{"authorized", "authorized_to_maintain_liabilities"},
				"set_flags":     []int32{4},
				"set_flags_s":   []string{"clawback_enabled"},
			},
		},
		{
			Type:          22,
			TypeString:    "liquidity_pool_deposit",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4126,
			OperationDetails: map[string]interface{}{
				"liquidity_pool_id":        "0102030405060708090000000000000000000000000000000000000000000000",
				"reserve_a_asset_type":     "native",
				"reserve_a_asset_id":       int64(-5706705804583548011),
				"reserve_a_max_amount":     0.0001,
				"reserve_a_deposit_amount": 0.0001,
				"reserve_b_asset_type":     "credit_alphanum4",
				"reserve_b_asset_code":     "USSD",
				"reserve_b_asset_issuer":   "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"reserve_b_asset_id":       int64(6690054458235693884),
				"reserve_b_deposit_amount": 0.00001,
				"reserve_b_max_amount":     0.00001,
				"max_price":                1000000.0000000,
				"max_price_r": utils.Price{
					Numerator:   1000000,
					Denominator: 1,
				},
				"min_price": 0.0000010,
				"min_price_r": utils.Price{
					Numerator:   1,
					Denominator: 1000000,
				},
				"shares_received": 0.0000002,
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "LiquidityPoolDepositResultCodeLiquidityPoolDepositSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"liquidity_pool_id":        "0102030405060708090000000000000000000000000000000000000000000000",
				"reserve_a_asset_type":     "native",
				"reserve_a_asset_id":       int64(-5706705804583548011),
				"reserve_a_max_amount":     0.0001,
				"reserve_a_deposit_amount": 0.0001,
				"reserve_b_asset_type":     "credit_alphanum4",
				"reserve_b_asset_code":     "USSD",
				"reserve_b_asset_issuer":   "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"reserve_b_asset_id":       int64(6690054458235693884),
				"reserve_b_deposit_amount": 0.00001,
				"reserve_b_max_amount":     0.00001,
				"max_price":                1000000.0000000,
				"max_price_r": utils.Price{
					Numerator:   1000000,
					Denominator: 1,
				},
				"min_price": 0.0000010,
				"min_price_r": utils.Price{
					Numerator:   1,
					Denominator: 1000000,
				},
				"shares_received": 0.0000002,
			},
		},
		{
			Type:          23,
			TypeString:    "liquidity_pool_withdraw",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4127,
			OperationDetails: map[string]interface{}{
				"liquidity_pool_id":         "0102030405060708090000000000000000000000000000000000000000000000",
				"reserve_a_asset_type":      "native",
				"reserve_a_asset_id":        int64(-5706705804583548011),
				"reserve_a_min_amount":      0.0000001,
				"reserve_a_withdraw_amount": -0.0001,
				"reserve_b_asset_type":      "credit_alphanum4",
				"reserve_b_asset_code":      "USSD",
				"reserve_b_asset_issuer":    "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"reserve_b_asset_id":        int64(6690054458235693884),
				"reserve_b_withdraw_amount": -0.00001,
				"reserve_b_min_amount":      0.0000001,
				"shares":                    0.0000004,
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "LiquidityPoolWithdrawResultCodeLiquidityPoolWithdrawSuccess",
			LedgerSequence:      0,
			OperationDetailsJSON: map[string]interface{}{
				"liquidity_pool_id":         "0102030405060708090000000000000000000000000000000000000000000000",
				"reserve_a_asset_type":      "native",
				"reserve_a_asset_id":        int64(-5706705804583548011),
				"reserve_a_min_amount":      0.0000001,
				"reserve_a_withdraw_amount": -0.0001,
				"reserve_b_asset_type":      "credit_alphanum4",
				"reserve_b_asset_code":      "USSD",
				"reserve_b_asset_issuer":    "GBVVRXLMNCJQW3IDDXC3X6XCH35B5Q7QXNMMFPENSOGUPQO7WO7HGZPA",
				"reserve_b_asset_id":        int64(6690054458235693884),
				"reserve_b_withdraw_amount": -0.00001,
				"reserve_b_min_amount":      0.0000001,
				"shares":                    0.0000004,
			},
		},
		{
			Type:          24,
			TypeString:    "invoke_host_function",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4128,
			OperationDetails: map[string]interface{}{
				"function":              "HostFunctionTypeHostFunctionTypeInvokeContract",
				"type":                  "invoke_contract",
				"contract_id":           "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABSC4",
				"contract_code_hash":    "",
				"asset_balance_changes": []map[string]interface{}{},
				"ledger_key_hash":       nilStringArray,
				"parameters": []map[string]string{
					{
						"type":  "Address",
						"value": "AAAAEgAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
					},
					{
						"type":  "Sym",
						"value": "AAAADwAAAAR0ZXN0",
					},
				},
				"parameters_decoded": []map[string]string{
					{
						"type":  "Address",
						"value": "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABSC4",
					},
					{
						"type":  "Sym",
						"value": "test",
					},
				},
			},
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "InvokeHostFunctionResultCodeInvokeHostFunctionSuccess",
			ClosedAt:            hardCodedLedgerClose,
			OperationDetailsJSON: map[string]interface{}{
				"function":              "HostFunctionTypeHostFunctionTypeInvokeContract",
				"type":                  "invoke_contract",
				"contract_id":           "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABSC4",
				"contract_code_hash":    "",
				"asset_balance_changes": []map[string]interface{}{},
				"ledger_key_hash":       nilStringArray,
				"parameters": []map[string]string{
					{
						"type":  "Address",
						"value": "AAAAEgAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
					},
					{
						"type":  "Sym",
						"value": "AAAADwAAAAR0ZXN0",
					},
				},
				"parameters_decoded": []map[string]string{
					{
						"type":  "Address",
						"value": "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABSC4",
					},
					{
						"type":  "Sym",
						"value": "test",
					},
				},
			},
		},
		{
			Type:          24,
			TypeString:    "invoke_host_function",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4129,
			OperationDetails: map[string]interface{}{
				"function":           "HostFunctionTypeHostFunctionTypeCreateContract",
				"type":               "create_contract",
				"contract_id":        "",
				"contract_code_hash": "",
				"from":               "address",
				"address":            "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABSC4",
				"ledger_key_hash":    nilStringArray,
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "InvokeHostFunctionResultCodeInvokeHostFunctionSuccess",
			OperationDetailsJSON: map[string]interface{}{
				"function":           "HostFunctionTypeHostFunctionTypeCreateContract",
				"type":               "create_contract",
				"contract_id":        "",
				"contract_code_hash": "",
				"from":               "address",
				"address":            "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABSC4",
				"ledger_key_hash":    nilStringArray,
			},
		},
		{
			Type:          24,
			TypeString:    "invoke_host_function",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4130,
			OperationDetails: map[string]interface{}{
				"function":           "HostFunctionTypeHostFunctionTypeCreateContract",
				"type":               "create_contract",
				"contract_id":        "",
				"contract_code_hash": "",
				"from":               "asset",
				"asset":              ":GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF",
				"ledger_key_hash":    nilStringArray,
			},
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "InvokeHostFunctionResultCodeInvokeHostFunctionSuccess",
			ClosedAt:            hardCodedLedgerClose,
			OperationDetailsJSON: map[string]interface{}{
				"function":           "HostFunctionTypeHostFunctionTypeCreateContract",
				"type":               "create_contract",
				"contract_id":        "",
				"contract_code_hash": "",
				"from":               "asset",
				"asset":              ":GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF",
				"ledger_key_hash":    nilStringArray,
			},
		},
		{
			Type:          24,
			TypeString:    "invoke_host_function",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4131,
			OperationDetails: map[string]interface{}{
				"function":           "HostFunctionTypeHostFunctionTypeCreateContractV2",
				"type":               "create_contract_v2",
				"contract_id":        "",
				"contract_code_hash": "",
				"from":               "asset",
				"asset":              ":GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF",
				"ledger_key_hash":    nilStringArray,
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
			},
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "InvokeHostFunctionResultCodeInvokeHostFunctionSuccess",
			ClosedAt:            hardCodedLedgerClose,
			OperationDetailsJSON: map[string]interface{}{
				"function":           "HostFunctionTypeHostFunctionTypeCreateContractV2",
				"type":               "create_contract_v2",
				"contract_id":        "",
				"contract_code_hash": "",
				"from":               "asset",
				"asset":              ":GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF",
				"ledger_key_hash":    nilStringArray,
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
			},
		},
		{
			Type:          24,
			TypeString:    "invoke_host_function",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4132,
			OperationDetails: map[string]interface{}{
				"function":           "HostFunctionTypeHostFunctionTypeUploadContractWasm",
				"type":               "upload_wasm",
				"contract_code_hash": "",
				"ledger_key_hash":    nilStringArray,
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "InvokeHostFunctionResultCodeInvokeHostFunctionSuccess",
			OperationDetailsJSON: map[string]interface{}{
				"function":           "HostFunctionTypeHostFunctionTypeUploadContractWasm",
				"type":               "upload_wasm",
				"contract_code_hash": "",
				"ledger_key_hash":    nilStringArray,
			},
		},
		{
			Type:          25,
			TypeString:    "extend_footprint_ttl",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4133,
			OperationDetails: map[string]interface{}{
				"type":               "extend_footprint_ttl",
				"extend_to":          xdr.Uint32(1234),
				"contract_id":        "",
				"contract_code_hash": "",
				"ledger_key_hash":    nilStringArray,
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "InvokeHostFunctionResultCodeInvokeHostFunctionSuccess",
			OperationDetailsJSON: map[string]interface{}{
				"type":               "extend_footprint_ttl",
				"extend_to":          xdr.Uint32(1234),
				"contract_id":        "",
				"contract_code_hash": "",
				"ledger_key_hash":    nilStringArray,
			},
		},
		{
			Type:          26,
			TypeString:    "restore_footprint",
			SourceAccount: hardCodedSourceAccountAddress,
			TransactionID: 4096,
			OperationID:   4134,
			OperationDetails: map[string]interface{}{
				"type":               "restore_footprint",
				"contract_id":        "",
				"contract_code_hash": "",
				"ledger_key_hash":    nilStringArray,
			},
			ClosedAt:            hardCodedLedgerClose,
			OperationResultCode: "OperationResultCodeOpInner",
			OperationTraceCode:  "InvokeHostFunctionResultCodeInvokeHostFunctionSuccess",
			OperationDetailsJSON: map[string]interface{}{
				"type":               "restore_footprint",
				"contract_id":        "",
				"contract_code_hash": "",
				"ledger_key_hash":    nilStringArray,
			},
		},
	}
	return
}

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

func transactionTestInput() *ingest.LedgerTransaction {
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

	transaction := &ingest.LedgerTransaction{
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
				LiquidityPoolID: "1c261d6c75930204a73b480c3020ab525e9be48ce93de6194cf69fb06f07452d",
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
					LiquidityPoolID: "0102030405060708090000000000000000000000000000000000000000000000",
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
				LiquidityPoolID: "0102030405060708090000000000000000000000000000000000000000000000",
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
				LiquidityPoolID: "0102030405060708090000000000000000000000000000000000000000000000",
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
					[]uint8{0x0, 0x0, 0x0, 0x12, 0x0, 0x0, 0x0, 0x1, 0x12, 0x34, 0x56, 0x78, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
					[]uint8{0x0, 0x0, 0x0, 0xf, 0x0, 0x0, 0x0, 0x4, 0x74, 0x65, 0x73, 0x74},
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
					[]uint8{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
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
