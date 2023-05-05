package integration

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const add_u64_contract = "soroban_add_u64.wasm"
const increment_contract = "soroban_increment_contract.wasm"

// Tests use precompiled wasm bin files that are added to the testdata directory.
// Refer to ./services/horizon/internal/integration/contracts/README.md on how to recompile
// contract code if needed to new wasm.

func TestContractInvokeHostFunctionInstallContract(t *testing.T) {
	t.Skip("sac contract tests disabled until footprint/fees are set correctly")
	if integration.GetCoreMaxSupportedProtocol() < 20 {
		t.Skip("This test run does not support less than Protocol 20")
	}

	itest := integration.NewTest(t, integration.Config{
		ProtocolVersion: 20,
	})

	// establish which account will be contract owner, and load it's current seq
	sourceAccount, err := itest.Client().AccountDetail(horizonclient.AccountRequest{
		AccountID: itest.Master().Address(),
	})
	require.NoError(t, err)

	installContractOp := assembleInstallContractCodeOp(t, itest.Master().Address(), add_u64_contract)
	tx, err := itest.SubmitOperations(&sourceAccount, itest.Master(), installContractOp)
	require.NoError(t, err)
	clientTx, err := itest.Client().TransactionDetail(tx.Hash)
	require.NoError(t, err)

	assert.Equal(t, tx.Hash, clientTx.Hash)
	var txResult xdr.TransactionResult
	err = xdr.SafeUnmarshalBase64(clientTx.ResultXdr, &txResult)
	require.NoError(t, err)

	var txEnv xdr.TransactionEnvelope
	err = xdr.SafeUnmarshalBase64(clientTx.EnvelopeXdr, &txEnv)
	require.NoError(t, err)

	opResults, ok := txResult.OperationResults()
	assert.True(t, ok)
	assert.Equal(t, len(opResults), 1)
	invokeHostFunctionResult, ok := opResults[0].MustTr().GetInvokeHostFunctionResult()
	assert.True(t, ok)
	assert.Equal(t, invokeHostFunctionResult.Code, xdr.InvokeHostFunctionResultCodeInvokeHostFunctionSuccess)

	clientInvokeOp, err := itest.Client().Operations(horizonclient.OperationRequest{
		ForTransaction: tx.Hash,
	})
	require.NoError(t, err)

	invokeHostFunctionOpJson, ok := clientInvokeOp.Embedded.Records[0].(operations.InvokeHostFunction)
	assert.True(t, ok)
	assert.Len(t, invokeHostFunctionOpJson.HostFunctions, 1)
	assert.Len(t, invokeHostFunctionOpJson.HostFunctions[0].Parameters, 0)
	assert.Equal(t, invokeHostFunctionOpJson.HostFunctions[0].Type, "upload_wasm")

}

func TestContractInvokeHostFunctionCreateContractBySourceAccount(t *testing.T) {
	t.Skip("sac contract tests disabled until footprint/fees are set correctly")
	if integration.GetCoreMaxSupportedProtocol() < 20 {
		t.Skip("This test run does not support less than Protocol 20")
	}

	itest := integration.NewTest(t, integration.Config{
		ProtocolVersion: 20,
	})

	// establish which account will be contract owner, and load it's current seq
	sourceAccount, err := itest.Client().AccountDetail(horizonclient.AccountRequest{
		AccountID: itest.Master().Address(),
	})

	// Install the contract

	installContractOp := assembleInstallContractCodeOp(t, itest.Master().Address(), add_u64_contract)
	itest.MustSubmitOperations(&sourceAccount, itest.Master(), installContractOp)

	// Create the contract

	require.NoError(t, err)
	createContractOp := assembleCreateContractOp(t, itest.Master().Address(), add_u64_contract, "a1", itest.GetPassPhrase())
	tx, err := itest.SubmitOperations(&sourceAccount, itest.Master(), createContractOp)
	require.NoError(t, err)

	clientTx, err := itest.Client().TransactionDetail(tx.Hash)
	require.NoError(t, err)

	assert.Equal(t, tx.Hash, clientTx.Hash)
	var txResult xdr.TransactionResult
	err = xdr.SafeUnmarshalBase64(clientTx.ResultXdr, &txResult)
	require.NoError(t, err)

	opResults, ok := txResult.OperationResults()
	assert.True(t, ok)
	assert.Equal(t, len(opResults), 1)
	invokeHostFunctionResult, ok := opResults[0].MustTr().GetInvokeHostFunctionResult()
	assert.True(t, ok)
	assert.Equal(t, invokeHostFunctionResult.Code, xdr.InvokeHostFunctionResultCodeInvokeHostFunctionSuccess)

	clientInvokeOp, err := itest.Client().Operations(horizonclient.OperationRequest{
		ForTransaction: tx.Hash,
	})
	require.NoError(t, err)

	invokeHostFunctionOpJson, ok := clientInvokeOp.Embedded.Records[0].(operations.InvokeHostFunction)
	assert.True(t, ok)
	assert.Len(t, invokeHostFunctionOpJson.HostFunctions, 1)
	assert.Len(t, invokeHostFunctionOpJson.HostFunctions[0].Parameters, 2)
	assert.Equal(t, invokeHostFunctionOpJson.HostFunctions[0].Type, "create_contract")
	assert.Equal(t, invokeHostFunctionOpJson.HostFunctions[0].Parameters[0]["from"], "source_account")
	assert.Equal(t, invokeHostFunctionOpJson.HostFunctions[0].Parameters[0]["type"], "string")
	assert.Equal(t, invokeHostFunctionOpJson.HostFunctions[0].Parameters[1]["salt"], "110986164698320180327942133831752629430491002266485370052238869825166557303060")
	assert.Equal(t, invokeHostFunctionOpJson.HostFunctions[0].Parameters[1]["type"], "string")
}

func TestContractInvokeHostFunctionInvokeStatelessContractFn(t *testing.T) {
	t.Skip("sac contract tests disabled until footprint/fees are set correctly")
	if integration.GetCoreMaxSupportedProtocol() < 20 {
		t.Skip("This test run does not support less than Protocol 20")
	}

	itest := integration.NewTest(t, integration.Config{
		ProtocolVersion: 20,
	})

	// establish which account will be contract owner
	sourceAccount, err := itest.Client().AccountDetail(horizonclient.AccountRequest{
		AccountID: itest.Master().Address(),
	})
	require.NoError(t, err)

	// Install the contract

	installContractOp := assembleInstallContractCodeOp(t, itest.Master().Address(), add_u64_contract)
	itest.MustSubmitOperations(&sourceAccount, itest.Master(), installContractOp)

	// Create the contract

	createContractOp := assembleCreateContractOp(t, itest.Master().Address(), add_u64_contract, "a1", itest.GetPassPhrase())
	tx, err := itest.SubmitOperations(&sourceAccount, itest.Master(), createContractOp)
	require.NoError(t, err)

	// contract has been deployed, now invoke a simple 'add' fn on the contract
	contractID := createContractOp.Ext.SorobanData.Resources.Footprint.ReadWrite[0].MustContractData().ContractId
	contractCodeLedgerKey := createContractOp.Ext.SorobanData.Resources.Footprint.ReadOnly[0]

	contractIdBytes := contractID[:]
	contractIdParameter := xdr.ScVal{
		Type:  xdr.ScValTypeScvBytes,
		Bytes: (*xdr.ScBytes)(&contractIdBytes),
	}

	contractFnParameterSym := xdr.ScSymbol("add")
	contractFnParameter := xdr.ScVal{
		Type: xdr.ScValTypeScvSymbol,
		Sym:  &contractFnParameterSym,
	}

	firstParamValue := xdr.Uint64(4)
	secondParamValue := xdr.Uint64(5)

	firstParamScVal := xdr.ScVal{
		Type: xdr.ScValTypeScvU64,
		U64:  &firstParamValue,
	}
	secondParamScVal := xdr.ScVal{
		Type: xdr.ScValTypeScvU64,
		U64:  &secondParamValue,
	}

	invokeHostFunctionOp := &txnbuild.InvokeHostFunctions{
		Functions: []xdr.HostFunction{
			{
				Args: xdr.HostFunctionArgs{
					Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
					InvokeContract: &xdr.ScVec{
						contractIdParameter,
						contractFnParameter,
						firstParamScVal,
						secondParamScVal,
					},
				},
			},
		},
		SourceAccount: sourceAccount.AccountID,
		Ext: xdr.TransactionExt{
			V: 1,
			SorobanData: &xdr.SorobanTransactionData{
				Resources: xdr.SorobanResources{
					Footprint: xdr.LedgerFootprint{
						ReadOnly: []xdr.LedgerKey{
							{
								Type: xdr.LedgerEntryTypeContractData,
								ContractData: &xdr.LedgerKeyContractData{
									ContractId: contractID,
									Key: xdr.ScVal{
										Type: xdr.ScValTypeScvLedgerKeyContractExecutable,
										// symbolic: no value
									},
								},
							},
							contractCodeLedgerKey,
						},
						ReadWrite: []xdr.LedgerKey{},
					},
					Instructions:              0,
					ReadBytes:                 0,
					WriteBytes:                0,
					ExtendedMetaDataSizeBytes: 0,
				},
				RefundableFee: 1,
				Ext: xdr.ExtensionPoint{
					V: 0,
				},
			},
		},
	}

	tx, err = itest.SubmitOperations(&sourceAccount, itest.Master(), invokeHostFunctionOp)
	require.NoError(t, err)

	clientTx, err := itest.Client().TransactionDetail(tx.Hash)
	require.NoError(t, err)

	assert.Equal(t, tx.Hash, clientTx.Hash)
	var txResult xdr.TransactionResult
	err = xdr.SafeUnmarshalBase64(clientTx.ResultXdr, &txResult)
	require.NoError(t, err)

	opResults, ok := txResult.OperationResults()
	assert.True(t, ok)
	assert.Equal(t, len(opResults), 1)
	invokeHostFunctionResult, ok := opResults[0].MustTr().GetInvokeHostFunctionResult()
	assert.True(t, ok)
	assert.Equal(t, invokeHostFunctionResult.Code, xdr.InvokeHostFunctionResultCodeInvokeHostFunctionSuccess)

	// check the function response, should have summed the two input numbers
	scvals := invokeHostFunctionResult.MustSuccess()
	for _, scval := range scvals {
		assert.Equal(t, xdr.Uint64(9), scval.MustU64())
	}

	clientInvokeOp, err := itest.Client().Operations(horizonclient.OperationRequest{
		ForTransaction: tx.Hash,
	})
	require.NoError(t, err)

	invokeHostFunctionOpJson, ok := clientInvokeOp.Embedded.Records[0].(operations.InvokeHostFunction)
	assert.True(t, ok)
	assert.Len(t, invokeHostFunctionOpJson.HostFunctions, 1)
	assert.Len(t, invokeHostFunctionOpJson.HostFunctions[0].Parameters, 4)
	assert.Equal(t, invokeHostFunctionOpJson.HostFunctions[0].Type, "invoke_contract")
	assert.Equal(t, invokeHostFunctionOpJson.HostFunctions[0].Parameters[0]["value"], "AAAADQAAACDhq+vRxjISTR62JpK1SAnzz1cZKpSpkRlwLJH6Zrzssg==")
	assert.Equal(t, invokeHostFunctionOpJson.HostFunctions[0].Parameters[0]["type"], "Bytes")
	assert.Equal(t, invokeHostFunctionOpJson.HostFunctions[0].Parameters[1]["value"], "AAAADwAAAANhZGQA")
	assert.Equal(t, invokeHostFunctionOpJson.HostFunctions[0].Parameters[1]["type"], "Sym")
	assert.Equal(t, invokeHostFunctionOpJson.HostFunctions[0].Parameters[2]["value"], "AAAABQAAAAAAAAAE")
	assert.Equal(t, invokeHostFunctionOpJson.HostFunctions[0].Parameters[2]["type"], "U64")
	assert.Equal(t, invokeHostFunctionOpJson.HostFunctions[0].Parameters[3]["value"], "AAAABQAAAAAAAAAF")
	assert.Equal(t, invokeHostFunctionOpJson.HostFunctions[0].Parameters[3]["type"], "U64")
}

func TestContractInvokeHostFunctionInvokeStatefulContractFn(t *testing.T) {
	t.Skip("sac contract tests disabled until footprint/fees are set correctly")
	if integration.GetCoreMaxSupportedProtocol() < 20 {
		t.Skip("This test run does not support less than Protocol 20")
	}

	itest := integration.NewTest(t, integration.Config{
		ProtocolVersion: 20,
	})

	// establish which account will be contract owner
	sourceAccount, err := itest.Client().AccountDetail(horizonclient.AccountRequest{
		AccountID: itest.Master().Address(),
	})
	require.NoError(t, err)

	// Install the contract

	installContractOp := assembleInstallContractCodeOp(t, itest.Master().Address(), increment_contract)
	itest.MustSubmitOperations(&sourceAccount, itest.Master(), installContractOp)

	// Create the contract

	createContractOp := assembleCreateContractOp(t, itest.Master().Address(), increment_contract, "a1", itest.GetPassPhrase())
	tx, err := itest.SubmitOperations(&sourceAccount, itest.Master(), createContractOp)
	require.NoError(t, err)

	// contract has been deployed, now invoke a simple 'add' fn on the contract
	contractID := createContractOp.Ext.SorobanData.Resources.Footprint.ReadWrite[0].MustContractData().ContractId
	contractCodeLedgerKey := createContractOp.Ext.SorobanData.Resources.Footprint.ReadOnly[0]

	contractIdBytes := contractID[:]
	contractIdParameter := xdr.ScVal{
		Type:  xdr.ScValTypeScvBytes,
		Bytes: (*xdr.ScBytes)(&contractIdBytes),
	}

	contractFnParameterSym := xdr.ScSymbol("increment")
	contractFnParameter := xdr.ScVal{
		Type: xdr.ScValTypeScvSymbol,
		Sym:  &contractFnParameterSym,
	}

	contractStateFootprintSym := xdr.ScSymbol("COUNTER")
	invokeHostFunctionOp := &txnbuild.InvokeHostFunctions{
		Functions: []xdr.HostFunction{
			{
				Args: xdr.HostFunctionArgs{
					Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
					InvokeContract: &xdr.ScVec{
						contractIdParameter,
						contractFnParameter,
					},
				},
			},
		},
		SourceAccount: sourceAccount.AccountID,
		Ext: xdr.TransactionExt{
			V: 1,
			SorobanData: &xdr.SorobanTransactionData{
				Resources: xdr.SorobanResources{
					Footprint: xdr.LedgerFootprint{
						ReadOnly: []xdr.LedgerKey{
							{
								Type: xdr.LedgerEntryTypeContractData,
								ContractData: &xdr.LedgerKeyContractData{
									ContractId: contractID,
									Key: xdr.ScVal{
										Type: xdr.ScValTypeScvLedgerKeyContractExecutable,
										// symbolic: no value
									},
								},
							},
							contractCodeLedgerKey,
						},
						ReadWrite: []xdr.LedgerKey{
							{
								Type: xdr.LedgerEntryTypeContractData,
								ContractData: &xdr.LedgerKeyContractData{
									ContractId: contractID,
									Key: xdr.ScVal{
										Type: xdr.ScValTypeScvSymbol,
										Sym:  &contractStateFootprintSym,
									},
								},
							},
						},
					},
					Instructions:              0,
					ReadBytes:                 0,
					WriteBytes:                0,
					ExtendedMetaDataSizeBytes: 0,
				},
				RefundableFee: 1,
				Ext: xdr.ExtensionPoint{
					V: 0,
				},
			},
		},
	}

	tx, err = itest.SubmitOperations(&sourceAccount, itest.Master(), invokeHostFunctionOp)
	require.NoError(t, err)

	clientTx, err := itest.Client().TransactionDetail(tx.Hash)
	require.NoError(t, err)

	assert.Equal(t, tx.Hash, clientTx.Hash)
	var txResult xdr.TransactionResult
	err = xdr.SafeUnmarshalBase64(clientTx.ResultXdr, &txResult)
	require.NoError(t, err)

	opResults, ok := txResult.OperationResults()
	assert.True(t, ok)
	assert.Equal(t, len(opResults), 1)
	invokeHostFunctionResult, ok := opResults[0].MustTr().GetInvokeHostFunctionResult()
	assert.True(t, ok)
	assert.Equal(t, invokeHostFunctionResult.Code, xdr.InvokeHostFunctionResultCodeInvokeHostFunctionSuccess)

	// check the function response, should have incremented state from 0 to 1
	scvals := invokeHostFunctionResult.MustSuccess()
	for _, scval := range scvals {
		assert.Equal(t, xdr.Uint32(1), scval.MustU32())
	}

	clientInvokeOp, err := itest.Client().Operations(horizonclient.OperationRequest{
		ForTransaction: tx.Hash,
	})
	require.NoError(t, err)

	invokeHostFunctionOpJson, ok := clientInvokeOp.Embedded.Records[0].(operations.InvokeHostFunction)
	assert.True(t, ok)
	assert.Len(t, invokeHostFunctionOpJson.HostFunctions, 1)
	assert.Len(t, invokeHostFunctionOpJson.HostFunctions[0].Parameters, 2)
	assert.Equal(t, invokeHostFunctionOpJson.HostFunctions[0].Type, "invoke_contract")
	assert.Equal(t, invokeHostFunctionOpJson.HostFunctions[0].Parameters[0]["value"], "AAAADQAAACDhq+vRxjISTR62JpK1SAnzz1cZKpSpkRlwLJH6Zrzssg==")
	assert.Equal(t, invokeHostFunctionOpJson.HostFunctions[0].Parameters[0]["type"], "Bytes")
	assert.Equal(t, invokeHostFunctionOpJson.HostFunctions[0].Parameters[1]["value"], "AAAADwAAAAlpbmNyZW1lbnQAAAA=")
	assert.Equal(t, invokeHostFunctionOpJson.HostFunctions[0].Parameters[1]["type"], "Sym")
}

func assembleInstallContractCodeOp(t *testing.T, sourceAccount string, wasmFileName string) *txnbuild.InvokeHostFunctions {
	// Assemble the InvokeHostFunction CreateContract operation:
	// CAP-0047 - https://github.com/stellar/stellar-protocol/blob/master/core/cap-0047.md#creating-a-contract-using-invokehostfunctionop

	contract, err := os.ReadFile(filepath.Join("testdata", wasmFileName))
	require.NoError(t, err)
	t.Logf("Contract File Contents: %v", hex.EncodeToString(contract))

	installContractCodeArgs, err := xdr.UploadContractWasmArgs{Code: contract}.MarshalBinary()
	assert.NoError(t, err)
	contractHash := sha256.Sum256(installContractCodeArgs)

	return &txnbuild.InvokeHostFunctions{
		Functions: []xdr.HostFunction{
			{
				Args: xdr.HostFunctionArgs{
					Type: xdr.HostFunctionTypeHostFunctionTypeUploadContractWasm,
					UploadContractWasm: &xdr.UploadContractWasmArgs{
						Code: contract,
					},
				},
			},
		},
		SourceAccount: sourceAccount,
		Ext: xdr.TransactionExt{
			V: 1,
			SorobanData: &xdr.SorobanTransactionData{
				Resources: xdr.SorobanResources{
					Footprint: xdr.LedgerFootprint{
						ReadWrite: []xdr.LedgerKey{
							{
								Type: xdr.LedgerEntryTypeContractCode,
								ContractCode: &xdr.LedgerKeyContractCode{
									Hash: contractHash,
								},
							},
						},
					},
					Instructions:              0,
					ReadBytes:                 0,
					WriteBytes:                0,
					ExtendedMetaDataSizeBytes: 0,
				},
				RefundableFee: 1,
				Ext: xdr.ExtensionPoint{
					V: 0,
				},
			},
		},
	}
}

func assembleCreateContractOp(t *testing.T, sourceAccount string, wasmFileName string, contractSalt string, passPhrase string) *txnbuild.InvokeHostFunctions {
	// Assemble the InvokeHostFunction CreateContract operation:
	// CAP-0047 - https://github.com/stellar/stellar-protocol/blob/master/core/cap-0047.md#creating-a-contract-using-invokehostfunctionop

	contract, err := os.ReadFile(filepath.Join("testdata", wasmFileName))
	require.NoError(t, err)

	salt := sha256.Sum256([]byte(contractSalt))
	t.Logf("Salt hash: %v", hex.EncodeToString(salt[:]))

	networkId := xdr.Hash(sha256.Sum256([]byte(passPhrase)))
	preImage := xdr.HashIdPreimage{
		Type: xdr.EnvelopeTypeEnvelopeTypeContractIdFromSourceAccount,
		SourceAccountContractId: &xdr.HashIdPreimageSourceAccountContractId{
			NetworkId: networkId,
			Salt:      salt,
		},
	}
	preImage.SourceAccountContractId.SourceAccount.SetAddress(sourceAccount)
	xdrPreImageBytes, err := preImage.MarshalBinary()
	require.NoError(t, err)
	hashedContractID := sha256.Sum256(xdrPreImageBytes)

	saltParameter := xdr.Uint256(salt)

	installContractCodeArgs, err := xdr.UploadContractWasmArgs{Code: contract}.MarshalBinary()
	assert.NoError(t, err)
	contractHash := xdr.Hash(sha256.Sum256(installContractCodeArgs))

	ledgerKey := xdr.LedgerKeyContractData{
		ContractId: xdr.Hash(hashedContractID),
		Key: xdr.ScVal{
			Type: xdr.ScValTypeScvLedgerKeyContractExecutable,
			// symbolic: no value
		},
	}

	return &txnbuild.InvokeHostFunctions{
		Functions: []xdr.HostFunction{
			{
				Args: xdr.HostFunctionArgs{
					Type: xdr.HostFunctionTypeHostFunctionTypeCreateContract,
					CreateContract: &xdr.CreateContractArgs{
						ContractId: xdr.ContractId{
							Type: xdr.ContractIdTypeContractIdFromSourceAccount,
							Salt: &saltParameter,
						},
						Executable: xdr.ScContractExecutable{
							Type:   xdr.ScContractExecutableTypeSccontractExecutableWasmRef,
							WasmId: &contractHash,
						},
					},
				},
			},
		},
		SourceAccount: sourceAccount,
		Ext: xdr.TransactionExt{
			V: 1,
			SorobanData: &xdr.SorobanTransactionData{
				Resources: xdr.SorobanResources{
					Footprint: xdr.LedgerFootprint{
						ReadWrite: []xdr.LedgerKey{
							{
								Type:         xdr.LedgerEntryTypeContractData,
								ContractData: &ledgerKey,
							},
						},
						ReadOnly: []xdr.LedgerKey{
							{
								Type: xdr.LedgerEntryTypeContractCode,
								ContractCode: &xdr.LedgerKeyContractCode{
									Hash: contractHash,
								},
							},
						},
					},
					Instructions:              0,
					ReadBytes:                 0,
					WriteBytes:                0,
					ExtendedMetaDataSizeBytes: 0,
				},
				RefundableFee: 1,
				Ext: xdr.ExtensionPoint{
					V: 0,
				},
			},
		},
	}
}
