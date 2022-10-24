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

// Tests refer to precompiled wasm bin files:
//
// `test_add_u64.wasm` has interface of one func called 'add':
/*
	{
		"type": "function",
		"name": "add",
		"inputs": [
			{
			"name": "a",
			"value": {
				"type": "u64"
			}
			},
			{
			"name": "b",
			"value": {
				"type": "u64"
			}
			}
		],
		"outputs": [
			{
			"type": "u64"
			}
		]
	}

*/
// compiled from the contract's rust source code:
// https://github.com/stellar/rs-soroban-sdk/blob/main/tests/add_u64/src/lib.rs

// `soroban_increment_contract.wasm` has interface of one func called 'increment':
/*
	{
		"type": "function",
		"name": "increment",
		"inputs": [],
		"outputs": [
			{
			"type": "u32"
			}
		]
	}
*/
// compiled from the contract's rust source code:
// https://github.com/stellar/soroban-examples/blob/main/increment/src/lib.rs

func TestInvokeHostFunctionCreateContractBySourceAccount(t *testing.T) {
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

	createContractOp := assembleCreateContractOp(t, itest.Master().Address(), "test_add_u64.wasm", "a1")
	opXDR, err := createContractOp.BuildXDR()
	require.NoError(t, err)

	invokeHostFunctionOp := opXDR.Body.MustInvokeHostFunctionOp()
	expectedFootPrint, err := xdr.MarshalBase64(invokeHostFunctionOp.Footprint)
	require.NoError(t, err)

	// clear footprint so we can verify preflight response
	invokeHostFunctionOp.Footprint = xdr.LedgerFootprint{}
	response, err := itest.CoreClient().Preflight(
		context.Background(),
		createContractOp.SourceAccount,
		invokeHostFunctionOp,
	)
	require.NoError(t, err)
	err = xdr.SafeUnmarshalBase64(response.Footprint, &invokeHostFunctionOp.Footprint)
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

func TestInvokeHostFunctionInvokeStatelessContractFn(t *testing.T) {

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

	createContractOp := assembleCreateContractOp(t, itest.Master().Address(), "test_add_u64.wasm", "a1")
	tx, err := itest.SubmitOperations(&sourceAccount, itest.Master(), createContractOp)
	require.NoError(t, err)

	// contract has been deployed, now invoke a simple 'add' fn on the contract
	contractID := createContractOp.Footprint.ReadWrite[0].MustContractData().ContractId

	contractIdBytes := contractID[:]
	contractIdParameterObj := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoBytes,
		Bin:  &contractIdBytes,
	}
	contractIdParameter := xdr.ScVal{
		Type: xdr.ScValTypeScvObject,
		Obj:  &contractIdParameterObj,
	}

	contractFnParameterSym := xdr.ScSymbol("add")
	contractFnParameter := xdr.ScVal{
		Type: xdr.ScValTypeScvSymbol,
		Sym:  &contractFnParameterSym,
	}

	firstParamValue := xdr.Uint64(4)
	firstParamValueObj := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoU64,
		U64:  &firstParamValue,
	}

	secondParamValue := xdr.Uint64(5)
	secondParamValueObj := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoU64,
		U64:  &secondParamValue,
	}

	contractCodeLedgerkeyAddr := xdr.ScStaticScsLedgerKeyContractCode

	invokeHostFunctionOp := &txnbuild.InvokeHostFunction{
		Function: xdr.HostFunctionHostFnInvokeContract,
		Parameters: xdr.ScVec{
			contractIdParameter,
			contractFnParameter,
			xdr.ScVal{
				Type: xdr.ScValTypeScvObject,
				Obj:  &firstParamValueObj,
			},
			xdr.ScVal{
				Type: xdr.ScValTypeScvObject,
				Obj:  &secondParamValueObj,
			},
		},
		Footprint: xdr.LedgerFootprint{
			ReadOnly: []xdr.LedgerKey{
				{
					Type: xdr.LedgerEntryTypeContractData,
					ContractData: &xdr.LedgerKeyContractData{
						ContractId: contractID,
						Key: xdr.ScVal{
							Type: xdr.ScValTypeScvStatic,
							Ic:   &contractCodeLedgerkeyAddr,
						},
					},
				},
			},
			ReadWrite: []xdr.LedgerKey{},
		},
		SourceAccount: sourceAccount.AccountID,
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
	scval := invokeHostFunctionResult.MustSuccess()
	assert.Equal(t, xdr.Uint64(9), scval.MustObj().MustU64())
}

func TestInvokeHostFunctionInvokeStatefulContractFn(t *testing.T) {
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

	createContractOp := assembleCreateContractOp(t, itest.Master().Address(), "soroban_increment_contract.wasm", "a1")
	tx, err := itest.SubmitOperations(&sourceAccount, itest.Master(), createContractOp)
	require.NoError(t, err)

	// contract has been deployed, now invoke a simple 'add' fn on the contract
	contractID := createContractOp.Footprint.ReadWrite[0].MustContractData().ContractId

	contractIdBytes := contractID[:]
	contractIdParameterObj := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoBytes,
		Bin:  &contractIdBytes,
	}
	contractIdParameter := xdr.ScVal{
		Type: xdr.ScValTypeScvObject,
		Obj:  &contractIdParameterObj,
	}

	contractFnParameterSym := xdr.ScSymbol("increment")
	contractFnParameter := xdr.ScVal{
		Type: xdr.ScValTypeScvSymbol,
		Sym:  &contractFnParameterSym,
	}

	contractStateFootprintSym := xdr.ScSymbol("COUNTER")
	contractCodeLedgerkeyAddr := xdr.ScStaticScsLedgerKeyContractCode

	invokeHostFunctionOp := &txnbuild.InvokeHostFunction{
		Function: xdr.HostFunctionHostFnInvokeContract,
		Parameters: xdr.ScVec{
			contractIdParameter,
			contractFnParameter,
		},
		Footprint: xdr.LedgerFootprint{
			ReadOnly: []xdr.LedgerKey{
				{
					Type: xdr.LedgerEntryTypeContractData,
					ContractData: &xdr.LedgerKeyContractData{
						ContractId: contractID,
						Key: xdr.ScVal{
							Type: xdr.ScValTypeScvStatic,
							Ic:   &contractCodeLedgerkeyAddr,
						},
					},
				},
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
		SourceAccount: sourceAccount.AccountID,
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
	scval := invokeHostFunctionResult.MustSuccess()
	assert.Equal(t, xdr.Uint32(1), scval.MustU32())
}

func assembleCreateContractOp(t *testing.T, sourceAccount string, wasmFileName string, contractSalt string) *txnbuild.InvokeHostFunction {
	// Assemble the InvokeHostFunction CreateContract operation:
	// CAP-0047 - https://github.com/stellar/stellar-protocol/blob/master/core/cap-0047.md#creating-a-contract-using-invokehostfunctionop

	contract, err := os.ReadFile(filepath.Join("testdata", wasmFileName))
	require.NoError(t, err)

	t.Logf("Contract File Contents: %v", hex.EncodeToString(contract))
	salt := sha256.Sum256([]byte(contractSalt))
	t.Logf("Salt hash: %v", hex.EncodeToString(salt[:]))

	preImage := xdr.HashIdPreimage{
		Type: xdr.EnvelopeTypeEnvelopeTypeContractIdFromSourceAccount,
		SourceAccountContractId: &xdr.HashIdPreimageSourceAccountContractId{
			Salt: salt,
		},
	}
	preImage.SourceAccountContractId.SourceAccount.SetAddress(sourceAccount)
	xdrPreImageBytes, err := preImage.MarshalBinary()
	require.NoError(t, err)
	hashedContractID := sha256.Sum256(xdrPreImageBytes)

	contractNameParameterAddr := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoBytes,
		Bin:  &contract,
	}
	contractNameParameter := xdr.ScVal{
		Type: xdr.ScValTypeScvObject,
		Obj:  &contractNameParameterAddr,
	}

	saltySlice := salt[:]
	saltParameterAddr := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoBytes,
		Bin:  &saltySlice,
	}
	saltParameter := xdr.ScVal{
		Type: xdr.ScValTypeScvObject,
		Obj:  &saltParameterAddr,
	}

	ledgerKeyContractCodeAddr := xdr.ScStaticScsLedgerKeyContractCode
	ledgerKey := xdr.LedgerKeyContractData{
		ContractId: xdr.Hash(hashedContractID),
		Key: xdr.ScVal{
			Type: xdr.ScValTypeScvStatic,
			Ic:   &ledgerKeyContractCodeAddr,
		},
	}

	return &txnbuild.InvokeHostFunction{
		Function: xdr.HostFunctionHostFnCreateContractWithSourceAccount,
		Footprint: xdr.LedgerFootprint{
			ReadWrite: []xdr.LedgerKey{
				{
					Type:         xdr.LedgerEntryTypeContractData,
					ContractData: &ledgerKey,
				},
			},
		},
		Parameters: xdr.ScVec{
			contractNameParameter,
			saltParameter,
		},
		SourceAccount: sourceAccount,
	}
}
