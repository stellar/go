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

func TestInvokeHostFunctionCreateContractByKey(t *testing.T) {
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

	createContractOp := assembleCreateContractOp(t, itest.Master().Address())
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

	paramsBin, err := createContractOp.Parameters.MarshalBinary()
	require.NoError(t, err)
	t.Log("XDR create contract args to Submit:", hex.EncodeToString(paramsBin))

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

func assembleCreateContractOp(t *testing.T, sourceAccount string) *txnbuild.InvokeHostFunction {
	// Assemble the InvokeHostFunction CreateContract operation, this is supposed to follow the
	// specs in CAP-0047 - https://github.com/stellar/stellar-protocol/blob/master/core/cap-0047.md#creating-a-contract-using-invokehostfunctionop

	// this defines a simple contract with interface of one func

	/*
		    {
				"type": "function",
				"name": "add",
				"inputs": [
					{
					"name": "a",
					"value": {
						"type": "i32"
					}
					},
					{
					"name": "b",
					"value": {
						"type": "i32"
					}
					}
				],
				"outputs": [
					{
					"type": "i32"
					}
				]
			}
	*/

	contract, err := os.ReadFile(filepath.Join("testdata", "example_add_i32.wasm"))
	require.NoError(t, err)
	t.Logf("Contract File Contents: %v", hex.EncodeToString(contract))
	salt := sha256.Sum256([]byte("a1"))
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
