package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/protocols/stellarcore"
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

	txEnv.V1.Tx.Operations[0].Body.GetInvokeHostFunctionOp()
}

func TestContractInvokeHostFunctionCreateContractBySourceAccount(t *testing.T) {
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
	opXDR, err := createContractOp.BuildXDR()
	require.NoError(t, err)

	invokeHostFunctionOp := opXDR.Body.MustInvokeHostFunctionOp()
	expectedFootPrint, err := xdr.MarshalBase64(createContractOp.Ext.SorobanData.Resources.Footprint)
	require.NoError(t, err)

	response, err := itest.CoreClient().Preflight(
		context.Background(),
		createContractOp.SourceAccount,
		invokeHostFunctionOp,
	)
	require.NoError(t, err)
	err = xdr.SafeUnmarshalBase64(response.Footprint, createContractOp.Ext.SorobanData.Resources.Footprint)
	require.NoError(t, err)
	require.Equal(t, stellarcore.PreflightStatusOk, response.Status)
	require.Equal(t, expectedFootPrint, response.Footprint)
	require.Greater(t, response.CPUInstructions, uint64(0))
	require.Greater(t, response.MemoryBytes, uint64(0))
	require.Empty(t, response.Detail)

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
}

func TestContractInvokeHostFunctionInvokeStatelessContractFn(t *testing.T) {
	os.Setenv("HORIZON_INTEGRATION_TESTS_ENABLED", "true")
	os.Setenv("HORIZON_INTEGRATION_TESTS_CORE_MAX_SUPPORTED_PROTOCOL", "20")
	os.Setenv("HORIZON_INTEGRATION_TESTS_DOCKER_IMG", "sreuland/stellar-core:19.9.1-1270.04f2a6d5c.focal-soroban")

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

	invokeHostFunctionOp := &txnbuild.InvokeHostFunctions{
		Functions: []xdr.HostFunction{
			{
				Args: xdr.HostFunctionArgs{
					Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
					InvokeContract: &xdr.ScVec{
						contractIdParameter,
						contractFnParameter,
						xdr.ScVal{
							Type: xdr.ScValTypeScvU64,
							U64:  &firstParamValue,
						},
						xdr.ScVal{
							Type: xdr.ScValTypeScvU64,
							U64:  &secondParamValue,
						},
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
}

func TestContractInvokeHostFunctionInvokeStatefulContractFn(t *testing.T) {
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
