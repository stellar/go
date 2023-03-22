//lint:file-ignore U1001 Ignore all unused code, staticcheck doesn't understand testify/suite

package processors

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/guregu/null"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/support/contractevents"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type OperationsProcessorTestSuiteLedger struct {
	suite.Suite
	ctx                    context.Context
	processor              *OperationProcessor
	mockQ                  *history.MockQOperations
	mockBatchInsertBuilder *history.MockOperationsBatchInsertBuilder
}

func TestOperationProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(OperationsProcessorTestSuiteLedger))
}

func (s *OperationsProcessorTestSuiteLedger) SetupTest() {
	s.ctx = context.Background()
	s.mockQ = &history.MockQOperations{}
	s.mockBatchInsertBuilder = &history.MockOperationsBatchInsertBuilder{}
	s.mockQ.
		On("NewOperationBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()

	s.processor = NewOperationProcessor(
		s.mockQ,
		56,
		"test network",
	)
}

func (s *OperationsProcessorTestSuiteLedger) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockBatchInsertBuilder.AssertExpectations(s.T())
}

func (s *OperationsProcessorTestSuiteLedger) mockBatchInsertAdds(txs []ingest.LedgerTransaction, sequence uint32) error {
	for _, t := range txs {
		for i, op := range t.Envelope.Operations() {
			expected := transactionOperationWrapper{
				index:          uint32(i),
				transaction:    t,
				operation:      op,
				ledgerSequence: sequence,
			}
			details, err := expected.Details()
			if err != nil {
				return err
			}
			detailsJSON, err := json.Marshal(details)
			if err != nil {
				return err
			}

			source := expected.SourceAccount()
			acID := source.ToAccountId()
			var muxedAccount null.String
			if source.Type == xdr.CryptoKeyTypeKeyTypeMuxedEd25519 {
				muxedAccount = null.StringFrom(source.Address())
			}
			s.mockBatchInsertBuilder.On(
				"Add",
				s.ctx,
				expected.ID(),
				expected.TransactionID(),
				expected.Order(),
				expected.OperationType(),
				detailsJSON,
				acID.Address(),
				muxedAccount,
				mock.Anything,
			).Return(nil).Once()
		}
	}

	return nil
}

func (s *OperationsProcessorTestSuiteLedger) TestInvokeFunctionDetails() {
	sourceAddress := "GAUJETIZVEP2NRYLUESJ3LS66NVCEGMON4UDCBCSBEVPIID773P2W6AY"
	source := xdr.MustMuxedAddress(sourceAddress)
	sourceAccountId := source.ToAccountId()
	sourcePublicKey := sourceAccountId.Ed25519

	contractParamVal1 := xdr.ScSymbol("func1")
	contractParamVal2 := xdr.Int32(-5)
	contractParamVal3 := xdr.Uint32(6)
	contractParamVal4 := xdr.Uint64(3)
	scoObjectBytes := []byte{0, 1, 2}
	contractParamVal5 := xdr.ScObject{
		Type: xdr.ScObjectTypeScoBytes,
		Bin:  &scoObjectBytes,
	}
	contractParamVal5Addr := &contractParamVal5
	contractParamVal6 := xdr.ScStaticScsTrue

	ledgerKeyAccount := xdr.LedgerKeyAccount{
		AccountId: source.ToAccountId(),
	}

	tx := ingest.LedgerTransaction{
		UnsafeMeta: xdr.TransactionMeta{
			V:  2,
			V2: &xdr.TransactionMetaV2{},
		},
	}

	s.T().Run("InvokeContract", func(t *testing.T) {
		wrapper := transactionOperationWrapper{
			transaction: tx,
			operation: xdr.Operation{
				SourceAccount: &source,
				Body: xdr.OperationBody{
					Type: xdr.OperationTypeInvokeHostFunction,
					InvokeHostFunctionOp: &xdr.InvokeHostFunctionOp{
						Function: xdr.HostFunction{
							Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
							InvokeArgs: &xdr.ScVec{
								{
									Type: xdr.ScValTypeScvSymbol,
									Sym:  &contractParamVal1,
								},
								{
									Type: xdr.ScValTypeScvI32,
									I32:  &contractParamVal2,
								},
								{
									Type: xdr.ScValTypeScvU32,
									U32:  &contractParamVal3,
								},
								{
									Type: xdr.ScValTypeScvBitset,
									Bits: &contractParamVal4,
								},
								{
									Type: xdr.ScValTypeScvObject,
									Obj:  &contractParamVal5Addr,
								},
								{
									Type: xdr.ScValTypeScvStatic,
									Ic:   &contractParamVal6,
								},
								{
									// invalid ScVal
									Type: 5555,
								},
							},
						},
						Footprint: xdr.LedgerFootprint{
							ReadOnly: []xdr.LedgerKey{
								{
									Type:    xdr.LedgerEntryTypeAccount,
									Account: &ledgerKeyAccount,
								},
							},
						},
					},
				},
			},
		}

		details, err := wrapper.Details()
		s.Assert().NoError(err)
		s.Assert().Equal(details["function"].(string), "HostFunctionTypeHostFunctionTypeInvokeContract")

		raw, err := wrapper.operation.Body.InvokeHostFunctionOp.Footprint.MarshalBinary()
		s.Assert().NoError(err)
		s.Assert().Equal(details["footprint"].(string), base64.StdEncoding.EncodeToString(raw))

		serializedParams := details["parameters"].([]map[string]string)
		var args []xdr.ScVal = *(wrapper.operation.Body.InvokeHostFunctionOp.Function.InvokeArgs)
		s.assertInvokeHostFunctionParameter(serializedParams, 0, "Sym", args[0])
		s.assertInvokeHostFunctionParameter(serializedParams, 1, "I32", args[1])
		s.assertInvokeHostFunctionParameter(serializedParams, 2, "U32", args[2])
		s.assertInvokeHostFunctionParameter(serializedParams, 3, "Bits", args[3])
		s.assertInvokeHostFunctionParameter(serializedParams, 4, "Obj", args[4])
		s.assertInvokeHostFunctionParameter(serializedParams, 5, "Ic", args[5])
		s.assertInvokeHostFunctionParameter(serializedParams, 6, "n/a", args[6])
	})

	s.T().Run("InvokeContractWithSACEventsInDetails", func(t *testing.T) {

		randomIssuer := keypair.MustRandom()
		randomAsset := xdr.MustNewCreditAsset("TESTING", randomIssuer.Address())
		passphrase := "passphrase"
		randomAccount := keypair.MustRandom().Address()
		contractId := [32]byte{}
		zeroContractStrKey, err := strkey.Encode(strkey.VersionByteContract, contractId[:])
		s.Assert().NoError(err)

		xferContractEvent := contractevents.GenerateEvent(contractevents.EventTypeTransfer, randomAccount, zeroContractStrKey, "", randomAsset, big.NewInt(10000000), passphrase)
		burnContractEvent := contractevents.GenerateEvent(contractevents.EventTypeBurn, zeroContractStrKey, "", "", randomAsset, big.NewInt(10000000), passphrase)
		mintContractEvent := contractevents.GenerateEvent(contractevents.EventTypeMint, "", zeroContractStrKey, randomAccount, randomAsset, big.NewInt(10000000), passphrase)
		clawbackContractEvent := contractevents.GenerateEvent(contractevents.EventTypeClawback, zeroContractStrKey, "", randomAccount, randomAsset, big.NewInt(10000000), passphrase)

		tx = ingest.LedgerTransaction{
			UnsafeMeta: xdr.TransactionMeta{
				V: 3,
				V3: &xdr.TransactionMetaV3{
					Events: []xdr.OperationEvents{
						{
							Events: []xdr.ContractEvent{
								xferContractEvent,
								burnContractEvent,
								mintContractEvent,
								clawbackContractEvent,
							},
						},
					},
				},
			},
		}
		wrapper := transactionOperationWrapper{
			transaction: tx,
			operation: xdr.Operation{
				SourceAccount: &source,
				Body: xdr.OperationBody{
					Type: xdr.OperationTypeInvokeHostFunction,
					InvokeHostFunctionOp: &xdr.InvokeHostFunctionOp{
						Function: xdr.HostFunction{
							Type:       xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
							InvokeArgs: &xdr.ScVec{},
						},
						Footprint: xdr.LedgerFootprint{
							ReadOnly: []xdr.LedgerKey{
								{
									Type:    xdr.LedgerEntryTypeAccount,
									Account: &ledgerKeyAccount,
								},
							},
						},
					},
				},
			},
			network: passphrase,
		}

		details, err := wrapper.Details()
		s.Assert().NoError(err)
		s.Assert().Len(details["asset_balance_changes"], 4)

		found := 0
		for _, assetBalanceChanged := range details["asset_balance_changes"].([]map[string]interface{}) {
			if assetBalanceChanged["type"] == "transfer" {
				s.Assert().Equal(assetBalanceChanged["from"], randomAccount)
				s.Assert().Equal(assetBalanceChanged["to"], zeroContractStrKey)
				s.Assert().Equal(assetBalanceChanged["amount"], "1.0000000")
				found++
			}

			if assetBalanceChanged["type"] == "burn" {
				s.Assert().Equal(assetBalanceChanged["from"], zeroContractStrKey)
				s.Assert().NotContains(assetBalanceChanged, "to")
				s.Assert().Equal(assetBalanceChanged["amount"], "1.0000000")
				found++
			}

			if assetBalanceChanged["type"] == "mint" {
				s.Assert().Equal(assetBalanceChanged["from"], randomAccount)
				s.Assert().Equal(assetBalanceChanged["to"], zeroContractStrKey)
				s.Assert().Equal(assetBalanceChanged["amount"], "1.0000000")
				found++
			}

			if assetBalanceChanged["type"] == "clawback" {
				s.Assert().Equal(assetBalanceChanged["from"], zeroContractStrKey)
				s.Assert().Equal(assetBalanceChanged["to"], randomAccount)
				s.Assert().Equal(assetBalanceChanged["amount"], "1.0000000")
				found++
			}
		}
		s.Assert().Equal(found, 4, "should have one balance changed record for each of mint, burn, clawback, transfer")
	})

	s.T().Run("CreateContract", func(t *testing.T) {
		signature := xdr.Signature{
			0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 5, 6, 7, 8,
		}
		signatureXdr, err := xdr.MarshalBase64(signature)
		s.Assert().NoError(err)
		salt := xdr.Uint256{
			0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 1, 2, 3, 4,
		}
		wrapper := transactionOperationWrapper{
			transaction: tx,
			operation: xdr.Operation{
				SourceAccount: &source,
				Body: xdr.OperationBody{
					Type: xdr.OperationTypeInvokeHostFunction,
					InvokeHostFunctionOp: &xdr.InvokeHostFunctionOp{
						Function: xdr.HostFunction{
							Type: xdr.HostFunctionTypeHostFunctionTypeCreateContract,
							CreateContractArgs: &xdr.CreateContractArgs{
								ContractId: xdr.ContractId{
									Type: xdr.ContractIdTypeContractIdFromEd25519PublicKey,
									FromEd25519PublicKey: &xdr.ContractIdFromEd25519PublicKey{
										Key:       *sourcePublicKey,
										Signature: signature,
										Salt:      salt,
									},
								},
								Source: xdr.ScContractCode{
									Type: xdr.ScContractCodeTypeSccontractCodeToken,
								},
							},
						},
						Footprint: xdr.LedgerFootprint{
							ReadOnly: []xdr.LedgerKey{
								{
									Type:    xdr.LedgerEntryTypeAccount,
									Account: &ledgerKeyAccount,
								},
							},
						},
					},
				},
			},
		}

		details, err := wrapper.Details()
		s.Assert().NoError(err)
		s.Assert().Equal(details["function"].(string), "HostFunctionTypeHostFunctionTypeCreateContract")

		raw, err := wrapper.operation.Body.InvokeHostFunctionOp.Footprint.MarshalBinary()
		s.Assert().NoError(err)
		s.Assert().Equal(details["footprint"].(string), base64.StdEncoding.EncodeToString(raw))

		s.Assert().Equal(details["type"].(string), "ContractIdTypeContractIdFromEd25519PublicKey")
		s.Assert().Equal(details["key"].(string), sourceAddress)
		s.Assert().Equal(details["signature"].(string), signatureXdr)
		s.Assert().Equal(details["salt"].(string), "16909060")
	})

	s.T().Run("InstallContractCode", func(t *testing.T) {
		code := []byte("Some contract code")
		wrapper := transactionOperationWrapper{
			transaction: tx,
			operation: xdr.Operation{
				SourceAccount: &source,
				Body: xdr.OperationBody{
					Type: xdr.OperationTypeInvokeHostFunction,
					InvokeHostFunctionOp: &xdr.InvokeHostFunctionOp{
						Function: xdr.HostFunction{
							Type: xdr.HostFunctionTypeHostFunctionTypeInstallContractCode,
							InstallContractCodeArgs: &xdr.InstallContractCodeArgs{
								Code: code,
							},
						},
						Footprint: xdr.LedgerFootprint{
							ReadOnly: []xdr.LedgerKey{
								{
									Type:    xdr.LedgerEntryTypeAccount,
									Account: &ledgerKeyAccount,
								},
							},
						},
					},
				},
			},
		}

		details, err := wrapper.Details()
		s.Assert().NoError(err)
		s.Assert().Equal(details["function"].(string), "HostFunctionTypeHostFunctionTypeInstallContractCode")

		raw, err := wrapper.operation.Body.InvokeHostFunctionOp.Footprint.MarshalBinary()
		s.Assert().NoError(err)
		s.Assert().Equal(details["footprint"].(string), base64.StdEncoding.EncodeToString(raw))

		s.Assert().Equal(details["code"].(string), base64.StdEncoding.EncodeToString(code))
	})
}

func (s *OperationsProcessorTestSuiteLedger) assertInvokeHostFunctionParameter(parameters []map[string]string, paramPosition int, expectedType string, expectedVal xdr.ScVal) {
	serializedParam := parameters[paramPosition]
	s.Assert().Equal(serializedParam["type"], expectedType)
	if expectedSerializedXdr, err := expectedVal.MarshalBinary(); err == nil {
		s.Assert().Equal(serializedParam["value"], base64.StdEncoding.EncodeToString(expectedSerializedXdr))
	} else {
		s.Assert().Equal(serializedParam["value"], "n/a")
	}
}

func (s *OperationsProcessorTestSuiteLedger) TestAddOperationSucceeds() {
	unmuxed := xdr.MustAddress("GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2")
	muxed := xdr.MuxedAccount{
		Type: xdr.CryptoKeyTypeKeyTypeMuxedEd25519,
		Med25519: &xdr.MuxedAccountMed25519{
			Id:      0xdeadbeefdeadbeef,
			Ed25519: *unmuxed.Ed25519,
		},
	}
	firstTx := createTransaction(true, 1)
	firstTx.Index = 1
	firstTx.Envelope.Operations()[0].Body = xdr.OperationBody{
		Type: xdr.OperationTypePayment,
		PaymentOp: &xdr.PaymentOp{
			Destination: muxed,
			Asset:       xdr.Asset{Type: xdr.AssetTypeAssetTypeNative},
			Amount:      100,
		},
	}
	firstTx.Envelope.V1.Tx.SourceAccount = muxed
	secondTx := createTransaction(false, 3)
	thirdTx := createTransaction(true, 4)

	txs := []ingest.LedgerTransaction{
		firstTx,
		secondTx,
		thirdTx,
	}

	var err error

	err = s.mockBatchInsertAdds(txs, uint32(56))
	s.Assert().NoError(err)
	s.mockBatchInsertBuilder.On("Exec", s.ctx).Return(nil).Once()
	s.Assert().NoError(s.processor.Commit(s.ctx))

	for _, tx := range txs {
		err = s.processor.ProcessTransaction(s.ctx, tx)
		s.Assert().NoError(err)
	}
}

func (s *OperationsProcessorTestSuiteLedger) TestAddOperationFails() {
	tx := createTransaction(true, 1)

	s.mockBatchInsertBuilder.
		On(
			"Add", s.ctx,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
		).Return(errors.New("transient error")).Once()

	err := s.processor.ProcessTransaction(s.ctx, tx)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error batch inserting operation rows: transient error")
}

func (s *OperationsProcessorTestSuiteLedger) TestExecFails() {
	s.mockBatchInsertBuilder.On("Exec", s.ctx).Return(errors.New("transient error")).Once()
	err := s.processor.Commit(s.ctx)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "transient error")
}
