package test

import (
	"context"
	"crypto/sha256"
	"net/http"
	"testing"
	"time"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/jhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stellar/go/exp/services/soroban-rpc/internal/methods"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

var (
	testContract = []byte("a contract")
	testSalt     = sha256.Sum256([]byte("a1"))
)

// createInvokeHostOperation creates a dummy InvokeHostFunctionOp. In this case by installing a contract code.
func createInvokeHostOperation(t *testing.T, sourceAccount string, includeFootprint bool) *txnbuild.InvokeHostFunction {
	return createInstallContractCodeOperation(t, sourceAccount, testContract, includeFootprint)
}

func createInstallContractCodeOperation(t *testing.T, sourceAccount string, contractCode []byte, includeFootprint bool) *txnbuild.InvokeHostFunction {
	var footprint xdr.LedgerFootprint
	if includeFootprint {
		// TODO: Check this is still right
		ledgerKey := xdr.LedgerKeyContractData{
			ContractId: xdr.Hash(getContractID(t, sourceAccount, testSalt)),
			Key:        getContractCodeLedgerKey(),
		}
		footprint = xdr.LedgerFootprint{
			ReadWrite: []xdr.LedgerKey{
				{
					Type:         xdr.LedgerEntryTypeContractData,
					ContractData: &ledgerKey,
				},
			},
		}
	}

	return &txnbuild.InvokeHostFunction{
		Footprint: footprint,
		Function: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInstallContractCode,
			InstallContractCodeArgs: &xdr.InstallContractCodeArgs{
				Code: contractCode,
			},
		},
		SourceAccount: sourceAccount,
	}
}

func createCreateContractOperation(t *testing.T, sourceAccount string, contractCode []byte, includeFootprint bool) *txnbuild.InvokeHostFunction {
	saltParam := xdr.Uint256(testSalt)

	var footprint xdr.LedgerFootprint
	if includeFootprint {
		// TODO: Check this is still right
		ledgerKey := xdr.LedgerKeyContractData{
			ContractId: xdr.Hash(getContractID(t, sourceAccount, testSalt)),
			Key:        getContractCodeLedgerKey(),
		}
		footprint = xdr.LedgerFootprint{
			ReadWrite: []xdr.LedgerKey{
				{
					Type:         xdr.LedgerEntryTypeContractData,
					ContractData: &ledgerKey,
				},
			},
		}
	}

	installContractCodeArgs, err := xdr.InstallContractCodeArgs{Code: contractCode}.MarshalBinary()
	assert.NoError(t, err)
	contractHash := xdr.Hash(sha256.Sum256(installContractCodeArgs))

	// two operations, install, then create.
	return &txnbuild.InvokeHostFunction{
		Footprint: footprint,
		Function: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeCreateContract,
			CreateContractArgs: &xdr.CreateContractArgs{
				ContractId: xdr.ContractId{
					Type: xdr.ContractIdTypeContractIdFromSourceAccount,
					Salt: &saltParam,
				},
				Source: xdr.ScContractCode{
					Type:   xdr.ScContractCodeTypeSccontractCodeWasmRef,
					WasmId: &contractHash,
				},
			},
		},
		SourceAccount: sourceAccount,
	}
}

func getContractCodeLedgerKey() xdr.ScVal {
	ledgerKeyContractCodeAddr := xdr.ScStaticScsLedgerKeyContractCode
	contractCodeLedgerKey := xdr.ScVal{
		Type: xdr.ScValTypeScvStatic,
		Ic:   &ledgerKeyContractCodeAddr,
	}
	return contractCodeLedgerKey
}

func getContractID(t *testing.T, sourceAccount string, salt [32]byte) [32]byte {
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
	return hashedContractID
}

func TestSimulateTransactionSucceeds(t *testing.T) {
	test := NewTest(t)

	ch := jhttp.NewChannel(test.server.URL, nil)
	client := jrpc2.NewClient(ch, nil)

	sourceAccount := keypair.Root(StandaloneNetworkPassphrase).Address()
	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount: &txnbuild.SimpleAccount{
			AccountID: sourceAccount,
			Sequence:  0,
		},
		IncrementSequenceNum: false,
		Operations: []txnbuild.Operation{
			createInvokeHostOperation(t, sourceAccount, false),
		},
		BaseFee: txnbuild.MinBaseFee,
		Memo:    nil,
		Preconditions: txnbuild.Preconditions{
			TimeBounds: txnbuild.NewInfiniteTimeout(),
		},
	})
	require.NoError(t, err)
	txB64, err := tx.Base64()
	require.NoError(t, err)
	request := methods.SimulateTransactionRequest{Transaction: txB64}

	var result methods.SimulateTransactionResponse
	err = client.CallResult(context.Background(), "simulateTransaction", request, &result)
	assert.NoError(t, err)
	assert.Greater(t, result.LatestLedger, int64(0))
	assert.Greater(t, result.Cost.CPUInstructions, uint64(0))
	assert.Greater(t, result.Cost.MemoryBytes, uint64(0))
	assert.Equal(
		t,
		methods.SimulateTransactionResponse{
			Footprint: "AAAAAAAAAAEAAAAGkvS4fCJA01o8HRusdDVaD5Z7F2lkyM3UfhQOjETmlDMAAAADAAAAAw==",
			Cost: methods.SimulateTransactionCost{
				CPUInstructions: result.Cost.CPUInstructions,
				MemoryBytes:     result.Cost.MemoryBytes,
			},
			Results: []methods.InvokeHostFunctionResult{
				{XDR: "AAAABAAAAAEAAAAEAAAAIJL0uHwiQNNaPB0brHQ1Wg+WexdpZMjN1H4UDoxE5pQz"},
			},
			LatestLedger: result.LatestLedger,
		},
		result,
	)

	// test operation which does not have a source account
	withoutSourceAccountOp := createInvokeHostOperation(t, "", false)
	tx, err = txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount: &txnbuild.SimpleAccount{
			AccountID: sourceAccount,
			Sequence:  0,
		},
		IncrementSequenceNum: false,
		Operations:           []txnbuild.Operation{withoutSourceAccountOp},
		BaseFee:              txnbuild.MinBaseFee,
		Memo:                 nil,
		Preconditions: txnbuild.Preconditions{
			TimeBounds: txnbuild.NewInfiniteTimeout(),
		},
	})
	require.NoError(t, err)
	txB64, err = tx.Base64()
	require.NoError(t, err)
	request = methods.SimulateTransactionRequest{Transaction: txB64}

	var resultForRequestWithoutOpSource methods.SimulateTransactionResponse
	err = client.CallResult(context.Background(), "simulateTransaction", request, &resultForRequestWithoutOpSource)
	assert.NoError(t, err)
	assert.Equal(t, result, resultForRequestWithoutOpSource)

	// test that operation source account takes precedence over tx source account
	tx, err = txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount: &txnbuild.SimpleAccount{
			AccountID: keypair.Root("test passphrase").Address(),
			Sequence:  0,
		},
		IncrementSequenceNum: false,
		Operations:           []txnbuild.Operation{createInvokeHostOperation(t, sourceAccount, false)},
		BaseFee:              txnbuild.MinBaseFee,
		Memo:                 nil,
		Preconditions: txnbuild.Preconditions{
			TimeBounds: txnbuild.NewInfiniteTimeout(),
		},
	})
	require.NoError(t, err)
	txB64, err = tx.Base64()
	require.NoError(t, err)
	request = methods.SimulateTransactionRequest{Transaction: txB64}

	var resultForRequestWithDifferentTxSource methods.SimulateTransactionResponse
	err = client.CallResult(context.Background(), "simulateTransaction", request, &resultForRequestWithDifferentTxSource)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, resultForRequestWithDifferentTxSource.LatestLedger, result.LatestLedger)
	// apart from latest ledger the response should be the same
	resultForRequestWithDifferentTxSource.LatestLedger = result.LatestLedger
	assert.Equal(t, result, resultForRequestWithDifferentTxSource)
}

func TestSimulateTransactionError(t *testing.T) {
	test := NewTest(t)

	ch := jhttp.NewChannel(test.server.URL, nil)
	client := jrpc2.NewClient(ch, nil)

	sourceAccount := keypair.Root(StandaloneNetworkPassphrase).Address()
	invokeHostOp := createInvokeHostOperation(t, sourceAccount, false)
	invokeHostOp.Function.InstallContractCodeArgs = nil
	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount: &txnbuild.SimpleAccount{
			AccountID: keypair.Root(StandaloneNetworkPassphrase).Address(),
			Sequence:  0,
		},
		IncrementSequenceNum: false,
		Operations:           []txnbuild.Operation{invokeHostOp},
		BaseFee:              txnbuild.MinBaseFee,
		Memo:                 nil,
		Preconditions: txnbuild.Preconditions{
			TimeBounds: txnbuild.NewInfiniteTimeout(),
		},
	})
	require.NoError(t, err)
	txB64, err := tx.Base64()
	require.NoError(t, err)
	request := methods.SimulateTransactionRequest{Transaction: txB64}

	var result methods.SimulateTransactionResponse
	err = client.CallResult(context.Background(), "simulateTransaction", request, &result)
	assert.NoError(t, err)
	assert.Empty(t, result.Results)
	assert.Greater(t, result.LatestLedger, int64(0))
	assert.Contains(t, result.Error, "InputArgsWrongLength")
}

func TestSimulateTransactionMultipleOperations(t *testing.T) {
	test := NewTest(t)

	ch := jhttp.NewChannel(test.server.URL, nil)
	client := jrpc2.NewClient(ch, nil)

	sourceAccount := keypair.Root(StandaloneNetworkPassphrase).Address()
	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount: &txnbuild.SimpleAccount{
			AccountID: keypair.Root(StandaloneNetworkPassphrase).Address(),
			Sequence:  0,
		},
		IncrementSequenceNum: false,
		Operations: []txnbuild.Operation{
			createInstallContractCodeOperation(t, sourceAccount, testContract, false),
			createCreateContractOperation(t, sourceAccount, testContract, false),
		},
		BaseFee: txnbuild.MinBaseFee,
		Memo:    nil,
		Preconditions: txnbuild.Preconditions{
			TimeBounds: txnbuild.NewInfiniteTimeout(),
		},
	})
	require.NoError(t, err)
	txB64, err := tx.Base64()
	require.NoError(t, err)
	request := methods.SimulateTransactionRequest{Transaction: txB64}

	var result methods.SimulateTransactionResponse
	err = client.CallResult(context.Background(), "simulateTransaction", request, &result)
	assert.NoError(t, err)
	assert.Equal(
		t,
		methods.SimulateTransactionResponse{
			Error: "Transaction contains more than one operation",
		},
		result,
	)
}

func TestSimulateTransactionWithoutInvokeHostFunction(t *testing.T) {
	test := NewTest(t)

	ch := jhttp.NewChannel(test.server.URL, nil)
	client := jrpc2.NewClient(ch, nil)

	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount: &txnbuild.SimpleAccount{
			AccountID: keypair.Root(StandaloneNetworkPassphrase).Address(),
			Sequence:  0,
		},
		IncrementSequenceNum: false,
		Operations: []txnbuild.Operation{
			&txnbuild.BumpSequence{BumpTo: 1},
		},
		BaseFee: txnbuild.MinBaseFee,
		Memo:    nil,
		Preconditions: txnbuild.Preconditions{
			TimeBounds: txnbuild.NewInfiniteTimeout(),
		},
	})
	require.NoError(t, err)
	txB64, err := tx.Base64()
	require.NoError(t, err)
	request := methods.SimulateTransactionRequest{Transaction: txB64}

	var result methods.SimulateTransactionResponse
	err = client.CallResult(context.Background(), "simulateTransaction", request, &result)
	assert.NoError(t, err)
	assert.Equal(
		t,
		methods.SimulateTransactionResponse{
			Error: "Transaction does not contain invoke host function operation",
		},
		result,
	)
}

func TestSimulateTransactionUnmarshalError(t *testing.T) {
	test := NewTest(t)

	ch := jhttp.NewChannel(test.server.URL, nil)
	client := jrpc2.NewClient(ch, nil)

	request := methods.SimulateTransactionRequest{Transaction: "invalid"}
	var result methods.SimulateTransactionResponse
	err := client.CallResult(context.Background(), "simulateTransaction", request, &result)
	assert.NoError(t, err)
	assert.Equal(
		t,
		"Could not unmarshal transaction",
		result.Error,
	)
}

func TestSimulateTransactionDeadlineError(t *testing.T) {
	test := NewTest(t)
	test.coreClient.HTTP = &http.Client{
		Timeout: time.Microsecond,
	}

	ch := jhttp.NewChannel(test.server.URL, nil)
	client := jrpc2.NewClient(ch, nil)

	sourceAccount := keypair.Root(StandaloneNetworkPassphrase).Address()
	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount: &txnbuild.SimpleAccount{
			AccountID: sourceAccount,
			Sequence:  0,
		},
		IncrementSequenceNum: false,
		Operations:           []txnbuild.Operation{createInvokeHostOperation(t, sourceAccount, false)},
		BaseFee:              txnbuild.MinBaseFee,
		Memo:                 nil,
		Preconditions: txnbuild.Preconditions{
			TimeBounds: txnbuild.NewInfiniteTimeout(),
		},
	})
	require.NoError(t, err)
	txB64, err := tx.Base64()
	require.NoError(t, err)
	request := methods.SimulateTransactionRequest{Transaction: txB64}

	var result methods.SimulateTransactionResponse
	err = client.CallResult(context.Background(), "simulateTransaction", request, &result)
	assert.NoError(t, err)
	assert.Equal(
		t,
		"Could not submit request to core",
		result.Error,
	)
}
