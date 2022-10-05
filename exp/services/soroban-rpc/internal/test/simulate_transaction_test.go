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

func createInvokeHostOperation(sourceAccount string) *txnbuild.InvokeHostFunction {
	contract := []byte("a contract")
	salt := sha256.Sum256([]byte("a1"))

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

	return &txnbuild.InvokeHostFunction{
		Function: xdr.HostFunctionHostFnCreateContractWithSourceAccount,
		Parameters: xdr.ScVec{
			contractNameParameter,
			saltParameter,
		},
		SourceAccount: sourceAccount,
	}
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
		Operations:           []txnbuild.Operation{createInvokeHostOperation(sourceAccount)},
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
	withoutSourceAccountOp := createInvokeHostOperation("")
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
		Operations:           []txnbuild.Operation{createInvokeHostOperation(sourceAccount)},
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
	assert.Equal(t, result, resultForRequestWithDifferentTxSource)
}

func TestSimulateTransactionError(t *testing.T) {
	test := NewTest(t)

	ch := jhttp.NewChannel(test.server.URL, nil)
	client := jrpc2.NewClient(ch, nil)

	sourceAccount := keypair.Root(StandaloneNetworkPassphrase).Address()
	invokeHostOp := createInvokeHostOperation(sourceAccount)
	invokeHostOp.Parameters = invokeHostOp.Parameters[:len(invokeHostOp.Parameters)-1]
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
	assert.Equal(
		t,
		methods.SimulateTransactionResponse{
			Error: "HostError\nValue: Status(HostFunctionError(InputArgsWrongLength))\n\nDebug events (newest first):\n   0: \"unexpected arguments to 'CreateContractWithSourceAccount' host function\"\n\nBacktrace (newest first):\n   0: <unknown>\n   1: <unknown>\n   2: <unknown>\n   3: <unknown>\n   4: <unknown>\n   5: <unknown>\n   6: <unknown>\n   7: <unknown>\n   8: <unknown>\n   9: <unknown>\n  10: <unknown>\n  11: <unknown>\n  12: <unknown>\n  13: <unknown>\n  14: <unknown>\n  15: <unknown>\n  16: <unknown>\n  17: <unknown>\n  18: <unknown>\n  19: <unknown>\n  20: <unknown>\n  21: <unknown>\n  22: <unknown>\n  23: <unknown>\n  24: <unknown>\n  25: __libc_start_main\n  26: <unknown>\n\n",
		},
		result,
	)
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
			createInvokeHostOperation(sourceAccount),

			createInvokeHostOperation(sourceAccount),
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
		Operations:           []txnbuild.Operation{createInvokeHostOperation(sourceAccount)},
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
