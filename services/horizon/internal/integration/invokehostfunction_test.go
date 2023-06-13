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
	// Set a very generous fee (10 XLM) which would satisfy any contract invocation
	tx, err := itest.SubmitOperationsWithFee(&sourceAccount, itest.Master(), 10*stroopsIn1XLM, installContractOp)
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
	assert.Equal(t, invokeHostFunctionOpJson.Type, "upload_wasm")

}

func TestContractInvokeHostFunctionCreateContractByAddress(t *testing.T) {
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
	// Set a very generous fee (10 XLM) which would satisfy any contract invocation
	itest.MustSubmitOperationsWithFee(&sourceAccount, itest.Master(), 10*stroopsIn1XLM, installContractOp)

	// Create the contract

	require.NoError(t, err)
	createContractOp := assembleCreateContractOp(t, itest.Master().Address(), add_u64_contract, "a1", itest.GetPassPhrase())
	tx, err := itest.SubmitOperationsWithFee(&sourceAccount, itest.Master(), 10*stroopsIn1XLM, createContractOp)
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
	assert.Equal(t, invokeHostFunctionOpJson.Function, "HostFunctionTypeHostFunctionTypeCreateContract")
	assert.Equal(t, invokeHostFunctionOpJson.Type, "create_contract")
	assert.Equal(t, invokeHostFunctionOpJson.Address, sourceAccount.AccountID)
	assert.Equal(t, invokeHostFunctionOpJson.Salt, "110986164698320180327942133831752629430491002266485370052238869825166557303060")
}

func TestContractInvokeHostFunctionInvokeStatelessContractFn(t *testing.T) {
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

	accountId := xdr.MustAddress(sourceAccount.AccountID)

	// Install the contract

	installContractOp := assembleInstallContractCodeOp(t, itest.Master().Address(), add_u64_contract)
	itest.MustSubmitOperationsWithFee(&sourceAccount, itest.Master(), 10*stroopsIn1XLM, installContractOp)

	// Create the contract

	createContractOp := assembleCreateContractOp(t, itest.Master().Address(), add_u64_contract, "a1", itest.GetPassPhrase())
	tx, err := itest.SubmitOperationsWithFee(&sourceAccount, itest.Master(), 10*stroopsIn1XLM, createContractOp)
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

	invokeHostFunctionOp := &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeContract: &xdr.ScVec{
				contractIdParameter,
				contractFnParameter,
				firstParamScVal,
				secondParamScVal,
			},
		},
		Auth: []xdr.SorobanAuthorizationEntry{
			{
				Credentials: xdr.SorobanCredentials{
					Type: xdr.SorobanCredentialsTypeSorobanCredentialsAddress,
					Address: &xdr.SorobanAddressCredentials{
						Address: xdr.ScAddress{
							Type:      xdr.ScAddressTypeScAddressTypeAccount,
							AccountId: &accountId,
						},
						Nonce:         0,
						SignatureArgs: nil,
					},
				},
				RootInvocation: xdr.SorobanAuthorizedInvocation{
					Function: xdr.SorobanAuthorizedFunction{
						Type: xdr.SorobanAuthorizedFunctionTypeSorobanAuthorizedFunctionTypeContractFn,
						ContractFn: &xdr.SorobanAuthorizedContractFunction{
							ContractAddress: xdr.ScAddress{
								Type:      xdr.ScAddressTypeScAddressTypeAccount,
								AccountId: &accountId,
							},
							FunctionName: "foo",
							Args:         nil,
						},
					},
					SubInvocations: nil,
				},
			},
		},
		SourceAccount: sourceAccount.AccountID,
		Ext: xdr.TransactionExt{
			V: 1,
			SorobanData: getMaxSorobanTransactionData(xdr.LedgerFootprint{
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
			}),
		},
	}

	tx, err = itest.SubmitOperationsWithFee(&sourceAccount, itest.Master(), 10*stroopsIn1XLM, invokeHostFunctionOp)
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
	hash := invokeHostFunctionResult.MustSuccess()
	assert.Equal(t, xdr.Uint64(9), hash)

	clientInvokeOp, err := itest.Client().Operations(horizonclient.OperationRequest{
		ForTransaction: tx.Hash,
	})
	require.NoError(t, err)

	invokeHostFunctionOpJson, ok := clientInvokeOp.Embedded.Records[0].(operations.InvokeHostFunction)
	assert.True(t, ok)
	assert.Len(t, invokeHostFunctionOpJson.Parameters, 2)
	assert.Equal(t, invokeHostFunctionOpJson.Function, "HostFunctionTypeHostFunctionTypeInvokeContract")
	assert.Equal(t, invokeHostFunctionOpJson.Type, "invoke_contract")
	assert.Equal(t, invokeHostFunctionOpJson.Parameters[0].Value, "AAAADQAAACDhq+vRxjISTR62JpK1SAnzz1cZKpSpkRlwLJH6Zrzssg==")
	assert.Equal(t, invokeHostFunctionOpJson.Parameters[0].Type, "Bytes")
	assert.Equal(t, invokeHostFunctionOpJson.Parameters[1].Value, "AAAADwAAAANhZGQA")
	assert.Equal(t, invokeHostFunctionOpJson.Parameters[1].Type, "Sym")
	assert.Equal(t, invokeHostFunctionOpJson.Parameters[2].Value, "AAAABQAAAAAAAAAE")
	assert.Equal(t, invokeHostFunctionOpJson.Parameters[2].Type, "U64")
	assert.Equal(t, invokeHostFunctionOpJson.Parameters[3].Value, "AAAABQAAAAAAAAAF")
	assert.Equal(t, invokeHostFunctionOpJson.Parameters[3].Type, "U64")
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
	itest.MustSubmitOperationsWithFee(&sourceAccount, itest.Master(), 10*stroopsIn1XLM, installContractOp)

	// Create the contract

	createContractOp := assembleCreateContractOp(t, itest.Master().Address(), increment_contract, "a1", itest.GetPassPhrase())
	tx, err := itest.SubmitOperationsWithFee(&sourceAccount, itest.Master(), 10*stroopsIn1XLM, createContractOp)
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
	invokeHostFunctionOp := &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeContract: &xdr.ScVec{
				contractIdParameter,
				contractFnParameter,
			},
		},
		SourceAccount: sourceAccount.AccountID,
		Ext: xdr.TransactionExt{
			V: 1,
			SorobanData: getMaxSorobanTransactionData(xdr.LedgerFootprint{
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
			}),
		},
	}

	tx, err = itest.SubmitOperationsWithFee(&sourceAccount, itest.Master(), 10*stroopsIn1XLM, invokeHostFunctionOp)
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
	hash := invokeHostFunctionResult.MustSuccess()
	assert.Equal(t, xdr.Uint32(1), hash)

	clientInvokeOp, err := itest.Client().Operations(horizonclient.OperationRequest{
		ForTransaction: tx.Hash,
	})
	require.NoError(t, err)

	invokeHostFunctionOpJson, ok := clientInvokeOp.Embedded.Records[0].(operations.InvokeHostFunction)
	assert.True(t, ok)
	assert.Len(t, invokeHostFunctionOpJson.Parameters, 2)
	assert.Equal(t, invokeHostFunctionOpJson.Function, "HostFunctionTypeHostFunctionTypeInvokeContract")
	assert.Equal(t, invokeHostFunctionOpJson.Type, "invoke_contract")
	assert.Equal(t, invokeHostFunctionOpJson.Parameters[0].Value, "AAAADQAAACDhq+vRxjISTR62JpK1SAnzz1cZKpSpkRlwLJH6Zrzssg==")
	assert.Equal(t, invokeHostFunctionOpJson.Parameters[0].Type, "Bytes")
	assert.Equal(t, invokeHostFunctionOpJson.Parameters[1].Value, "AAAADwAAAAlpbmNyZW1lbnQAAAA=")
	assert.Equal(t, invokeHostFunctionOpJson.Parameters[1].Type, "Sym")
}

const stroopsIn1XLM = int64(10_000_000)

func getMaxSorobanTransactionData(fp xdr.LedgerFootprint) *xdr.SorobanTransactionData {
	// From https://soroban.stellar.org/docs/learn/fees-and-metering#resource-limits
	return &xdr.SorobanTransactionData{
		Resources: xdr.SorobanResources{
			Footprint:                 fp,
			Instructions:              40_000_000,
			ReadBytes:                 200 * 1024,
			WriteBytes:                100 * 1024,
			ExtendedMetaDataSizeBytes: 200 * 1024,
		},
		// 1 XML should be future-proof
		RefundableFee: 1 * xdr.Int64(stroopsIn1XLM),
		Ext: xdr.ExtensionPoint{
			V: 0,
		},
	}
}

func assembleInstallContractCodeOp(t *testing.T, sourceAccount string, wasmFileName string) *txnbuild.InvokeHostFunction {
	// Assemble the InvokeHostFunction UploadContractWasm operation:
	// CAP-0047 - https://github.com/stellar/stellar-protocol/blob/master/core/cap-0047.md#creating-a-contract-using-invokehostfunctionop

	contract, err := os.ReadFile(filepath.Join("testdata", wasmFileName))
	require.NoError(t, err)
	t.Logf("Contract File Contents: %v", hex.EncodeToString(contract))

	contractHash := sha256.Sum256(contract)

	return &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeUploadContractWasm,
			Wasm: &contract,
		},
		SourceAccount: sourceAccount,
		Ext: xdr.TransactionExt{
			V: 1,
			SorobanData: getMaxSorobanTransactionData(xdr.LedgerFootprint{
				ReadWrite: []xdr.LedgerKey{
					{
						Type: xdr.LedgerEntryTypeContractCode,
						ContractCode: &xdr.LedgerKeyContractCode{
							Hash: contractHash,
						},
					},
				},
			}),
		},
	}
}

func assembleCreateContractOp(t *testing.T, sourceAccount string, wasmFileName string, contractSalt string, passPhrase string) *txnbuild.InvokeHostFunction {
	// Assemble the InvokeHostFunction CreateContract operation:
	// CAP-0047 - https://github.com/stellar/stellar-protocol/blob/master/core/cap-0047.md#creating-a-contract-using-invokehostfunctionop

	contract, err := os.ReadFile(filepath.Join("testdata", wasmFileName))
	require.NoError(t, err)

	salt := sha256.Sum256([]byte(contractSalt))
	t.Logf("Salt hash: %v", hex.EncodeToString(salt[:]))
	saltParameter := xdr.Uint256(salt)

	networkId := xdr.Hash(sha256.Sum256([]byte(passPhrase)))
	accountId := xdr.MustAddress(sourceAccount)
	preImage := xdr.HashIdPreimage{
		Type: xdr.EnvelopeTypeEnvelopeTypeContractId,
		ContractId: &xdr.HashIdPreimageContractId{
			NetworkId: networkId,
			ContractIdPreimage: xdr.ContractIdPreimage{
				Type: xdr.ContractIdPreimageTypeContractIdPreimageFromAddress,
				FromAddress: &xdr.ContractIdPreimageFromAddress{
					Address: xdr.ScAddress{
						Type:      xdr.ScAddressTypeScAddressTypeAccount,
						AccountId: &accountId,
					},
					Salt: salt,
				},
			},
		},
	}
	xdrPreImageBytes, err := preImage.MarshalBinary()
	require.NoError(t, err)
	hashedContractID := sha256.Sum256(xdrPreImageBytes)

	contractHash := xdr.Hash(sha256.Sum256(contract))

	ledgerKey := xdr.LedgerKeyContractData{
		ContractId: xdr.Hash(hashedContractID),
		Key: xdr.ScVal{
			Type: xdr.ScValTypeScvLedgerKeyContractExecutable,
			// symbolic: no value
		},
	}

	return &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeCreateContract,
			CreateContract: &xdr.CreateContractArgs{
				ContractIdPreimage: xdr.ContractIdPreimage{
					Type: xdr.ContractIdPreimageTypeContractIdPreimageFromAddress,
					FromAddress: &xdr.ContractIdPreimageFromAddress{
						Address: xdr.ScAddress{
							Type:      xdr.ScAddressTypeScAddressTypeAccount,
							AccountId: &accountId,
						},
						Salt: saltParameter,
					},
				},
				Executable: xdr.ScContractExecutable{
					Type:   xdr.ScContractExecutableTypeSccontractExecutableWasmRef,
					WasmId: &contractHash,
				},
			},
		},
		SourceAccount: sourceAccount,
		Ext: xdr.TransactionExt{
			V: 1,
			SorobanData: getMaxSorobanTransactionData(xdr.LedgerFootprint{
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
			}),
		},
	}
}
