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
	preFlightOp, minFee := itest.PreflightHostFunctions(&sourceAccount, *installContractOp)
	tx := itest.MustSubmitOperationsWithFee(&sourceAccount, itest.Master(), minFee, &preFlightOp)

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
	assert.Equal(t, invokeHostFunctionOpJson.Function, "HostFunctionTypeHostFunctionTypeUploadContractWasm")

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
	require.NoError(t, err)

	// Install the contract
	installContractOp := assembleInstallContractCodeOp(t, itest.Master().Address(), add_u64_contract)
	preFlightOp, minFee := itest.PreflightHostFunctions(&sourceAccount, *installContractOp)
	itest.MustSubmitOperationsWithFee(&sourceAccount, itest.Master(), minFee, &preFlightOp)

	// Create the contract
	createContractOp := assembleCreateContractOp(t, itest.Master().Address(), add_u64_contract, "a1", itest.GetPassPhrase())
	preFlightOp, minFee = itest.PreflightHostFunctions(&sourceAccount, *createContractOp)
	tx, err := itest.SubmitOperationsWithFee(&sourceAccount, itest.Master(), minFee, &preFlightOp)
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

	// Install the contract
	installContractOp := assembleInstallContractCodeOp(t, itest.Master().Address(), add_u64_contract)
	preFlightOp, minFee := itest.PreflightHostFunctions(&sourceAccount, *installContractOp)
	itest.MustSubmitOperationsWithFee(&sourceAccount, itest.Master(), minFee, &preFlightOp)

	// Create the contract
	createContractOp := assembleCreateContractOp(t, itest.Master().Address(), add_u64_contract, "a1", itest.GetPassPhrase())
	preFlightOp, minFee = itest.PreflightHostFunctions(&sourceAccount, *createContractOp)
	tx, err := itest.SubmitOperationsWithFee(&sourceAccount, itest.Master(), minFee, &preFlightOp)
	require.NoError(t, err)

	// contract has been deployed, now invoke a simple 'add' fn on the contract
	contractID := preFlightOp.Ext.SorobanData.Resources.Footprint.ReadWrite[0].MustContractData().Contract.ContractId
	require.NotNil(t, contractID)
	contractIdParameter := xdr.ScVal{
		Type: xdr.ScValTypeScvAddress,
		Address: &xdr.ScAddress{
			Type:       xdr.ScAddressTypeScAddressTypeContract,
			ContractId: contractID,
		},
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
		SourceAccount: sourceAccount.AccountID,
	}

	preFlightOp, minFee = itest.PreflightHostFunctions(&sourceAccount, *invokeHostFunctionOp)
	tx, err = itest.SubmitOperationsWithFee(&sourceAccount, itest.Master(), minFee, &preFlightOp)
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
	invokeResult := xdr.Uint64(9)
	expectedScVal := xdr.ScVal{Type: xdr.ScValTypeScvU64, U64: &invokeResult}
	var transactionMeta xdr.TransactionMeta
	assert.NoError(t, xdr.SafeUnmarshalBase64(tx.ResultMetaXdr, &transactionMeta))
	assert.True(t, expectedScVal.Equals(transactionMeta.V3.SorobanMeta.ReturnValue))

	clientInvokeOp, err := itest.Client().Operations(horizonclient.OperationRequest{
		ForTransaction: tx.Hash,
	})
	require.NoError(t, err)

	invokeHostFunctionOpJson, ok := clientInvokeOp.Embedded.Records[0].(operations.InvokeHostFunction)
	assert.True(t, ok)
	assert.Len(t, invokeHostFunctionOpJson.Parameters, 4)
	assert.Equal(t, invokeHostFunctionOpJson.Function, "HostFunctionTypeHostFunctionTypeInvokeContract")
	addressParam, err := xdr.MarshalBase64(contractIdParameter)
	require.NoError(t, err)
	assert.Equal(t, invokeHostFunctionOpJson.Parameters[0].Value, addressParam)
	assert.Equal(t, invokeHostFunctionOpJson.Parameters[0].Type, "Address")
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
	preFlightOp, minFee := itest.PreflightHostFunctions(&sourceAccount, *installContractOp)
	itest.MustSubmitOperationsWithFee(&sourceAccount, itest.Master(), minFee, &preFlightOp)

	// Create the contract

	createContractOp := assembleCreateContractOp(t, itest.Master().Address(), increment_contract, "a1", itest.GetPassPhrase())
	preFlightOp, minFee = itest.PreflightHostFunctions(&sourceAccount, *createContractOp)
	tx, err := itest.SubmitOperationsWithFee(&sourceAccount, itest.Master(), minFee, &preFlightOp)
	require.NoError(t, err)

	// contract has been deployed, now invoke a simple 'add' fn on the contract
	contractID := preFlightOp.Ext.SorobanData.Resources.Footprint.ReadWrite[0].MustContractData().Contract.ContractId
	require.NotNil(t, contractID)
	contractIdParameter := xdr.ScVal{
		Type: xdr.ScValTypeScvAddress,
		Address: &xdr.ScAddress{
			Type:       xdr.ScAddressTypeScAddressTypeContract,
			ContractId: contractID,
		},
	}

	contractFnParameterSym := xdr.ScSymbol("increment")
	contractFnParameter := xdr.ScVal{
		Type: xdr.ScValTypeScvSymbol,
		Sym:  &contractFnParameterSym,
	}

	invokeHostFunctionOp := &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeContract: &xdr.ScVec{
				contractIdParameter,
				contractFnParameter,
			},
		},
		SourceAccount: sourceAccount.AccountID,
	}

	preFlightOp, minFee = itest.PreflightHostFunctions(&sourceAccount, *invokeHostFunctionOp)
	tx, err = itest.SubmitOperationsWithFee(&sourceAccount, itest.Master(), minFee, &preFlightOp)
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
	invokeResult := xdr.Uint32(1)
	expectedScVal := xdr.ScVal{Type: xdr.ScValTypeScvU32, U32: &invokeResult}
	var transactionMeta xdr.TransactionMeta
	assert.NoError(t, xdr.SafeUnmarshalBase64(clientTx.ResultMetaXdr, &transactionMeta))
	assert.True(t, expectedScVal.Equals(transactionMeta.V3.SorobanMeta.ReturnValue))

	clientInvokeOp, err := itest.Client().Operations(horizonclient.OperationRequest{
		ForTransaction: tx.Hash,
	})
	require.NoError(t, err)

	invokeHostFunctionOpJson, ok := clientInvokeOp.Embedded.Records[0].(operations.InvokeHostFunction)
	assert.True(t, ok)
	assert.Len(t, invokeHostFunctionOpJson.Parameters, 2)
	assert.Equal(t, invokeHostFunctionOpJson.Function, "HostFunctionTypeHostFunctionTypeInvokeContract")
	addressParam, err := xdr.MarshalBase64(contractIdParameter)
	require.NoError(t, err)
	assert.Equal(t, invokeHostFunctionOpJson.Parameters[0].Value, addressParam)
	assert.Equal(t, invokeHostFunctionOpJson.Parameters[0].Type, "Address")
	assert.Equal(t, invokeHostFunctionOpJson.Parameters[1].Value, "AAAADwAAAAlpbmNyZW1lbnQAAAA=")
	assert.Equal(t, invokeHostFunctionOpJson.Parameters[1].Type, "Sym")
}

func assembleInstallContractCodeOp(t *testing.T, sourceAccount string, wasmFileName string) *txnbuild.InvokeHostFunction {
	// Assemble the InvokeHostFunction UploadContractWasm operation:
	// CAP-0047 - https://github.com/stellar/stellar-protocol/blob/master/core/cap-0047.md#creating-a-contract-using-invokehostfunctionop

	contract, err := os.ReadFile(filepath.Join("testdata", wasmFileName))
	require.NoError(t, err)
	t.Logf("Contract File Contents: %v", hex.EncodeToString(contract))

	return &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeUploadContractWasm,
			Wasm: &contract,
		},
		SourceAccount: sourceAccount,
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

	accountId := xdr.MustAddress(sourceAccount)
	require.NoError(t, err)
	contractHash := xdr.Hash(sha256.Sum256(contract))

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
				Executable: xdr.ContractExecutable{
					Type:     xdr.ContractExecutableTypeContractExecutableWasm,
					WasmHash: &contractHash,
				},
			},
		},
		SourceAccount: sourceAccount,
	}
}
