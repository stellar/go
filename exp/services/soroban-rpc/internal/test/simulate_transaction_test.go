package test

import (
	"context"
	"crypto/sha256"
	"testing"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/jhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stellar/go/exp/services/soroban-rpc/internal/methods"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

func createInvokeHostOperation(t *testing.T) xdr.Operation {
	accountKp := keypair.Root(StandaloneNetworkPassphrase)
	sha256Hash := sha256.New()
	contract := []byte("a contract")
	salt := sha256.Sum256([]byte("a1"))
	separator := []byte("create_contract_from_ed25519(contract: Vec<u8>, salt: u256, key: u256, sig: Vec<u8>)")

	sha256Hash.Write(separator)
	sha256Hash.Write(salt[:])
	sha256Hash.Write(contract)

	contractHash := sha256Hash.Sum([]byte{})
	contractSig, err := accountKp.Sign(contractHash)
	require.NoError(t, err)

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

	publicKeySlice := []byte(accountKp.PublicKey())
	publicKeyParameterAddr := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoBytes,
		Bin:  &publicKeySlice,
	}
	publicKeyParameter := xdr.ScVal{
		Type: xdr.ScValTypeScvObject,
		Obj:  &publicKeyParameterAddr,
	}

	contractSignatureParaeterAddr := &xdr.ScObject{
		Type: xdr.ScObjectTypeScoBytes,
		Bin:  &contractSig,
	}
	contractSignatureParameter := xdr.ScVal{
		Type: xdr.ScValTypeScvObject,
		Obj:  &contractSignatureParaeterAddr,
	}

	op, err := (&txnbuild.InvokeHostFunction{
		Function: xdr.HostFunctionHostFnCreateContract,
		Parameters: xdr.ScVec{
			contractNameParameter,
			saltParameter,
			publicKeyParameter,
			contractSignatureParameter,
		},
	}).BuildXDR()
	assert.NoError(t, err)
	return op
}

func TestSimulateTransactionSucceeds(t *testing.T) {
	test := NewTest(t)

	ch := jhttp.NewChannel(test.server.URL, nil)
	cli := jrpc2.NewClient(ch, nil)

	invokeHostOpB64, err := xdr.MarshalBase64(createInvokeHostOperation(t).Body.MustInvokeHostFunctionOp())
	require.NoError(t, err)

	request := methods.SimulateTransactionRequest{InvokeHostFunctionOp: invokeHostOpB64}
	var result methods.SimulateTransactionResponse
	err = cli.CallResult(context.Background(), "simulateTransaction", request, &result)
	assert.NoError(t, err)
	assert.Equal(
		t,
		methods.SimulateTransactionResponse{
			Result:    "AAAABAAAAAEAAAAEAAAAIGbba7aVJHQuYwtVAaQdNhRSu6PxDWrUPGsKBRiUum0g",
			Footprint: "AAAAAAAAAAEAAAAGZttrtpUkdC5jC1UBpB02FFK7o/ENatQ8awoFGJS6bSAAAAADAAAAAw==",
			Cost: methods.SimulateTransactionCost{
				CPUInstructions: 606,
				MemoryBytes:     66,
			},
		},
		result,
	)
}

func TestSimulateTransactionError(t *testing.T) {
	test := NewTest(t)

	ch := jhttp.NewChannel(test.server.URL, nil)
	cli := jrpc2.NewClient(ch, nil)

	invokeHostOp := createInvokeHostOperation(t).Body.MustInvokeHostFunctionOp()
	// remove signature parameter
	invokeHostOp.Parameters = invokeHostOp.Parameters[:len(invokeHostOp.Parameters)-1]
	invokeHostOpB64, err := xdr.MarshalBase64(invokeHostOp)
	require.NoError(t, err)

	request := methods.SimulateTransactionRequest{InvokeHostFunctionOp: invokeHostOpB64}
	var result methods.SimulateTransactionResponse
	err = cli.CallResult(context.Background(), "simulateTransaction", request, &result)
	assert.NoError(t, err)
	assert.Equal(
		t,
		methods.SimulateTransactionResponse{
			Error: "HostError\nValue: Status(HostFunctionError(InputArgsWrongLength))\n\nDebug events (newest first):\n   0: \"unexpected arguments to 'CreateContract' host function\"\n\nBacktrace (newest first):\n   0: <unknown>\n   1: <unknown>\n   2: <unknown>\n   3: <unknown>\n   4: <unknown>\n   5: <unknown>\n   6: <unknown>\n   7: <unknown>\n   8: <unknown>\n   9: <unknown>\n  10: <unknown>\n  11: <unknown>\n  12: <unknown>\n  13: <unknown>\n  14: <unknown>\n  15: <unknown>\n  16: <unknown>\n  17: <unknown>\n  18: <unknown>\n  19: <unknown>\n  20: <unknown>\n  21: <unknown>\n  22: <unknown>\n  23: __libc_start_main\n  24: <unknown>\n\n",
		},
		result,
	)
}

func TestSimulateTransactionUnmarshalError(t *testing.T) {
	test := NewTest(t)

	ch := jhttp.NewChannel(test.server.URL, nil)
	cli := jrpc2.NewClient(ch, nil)

	request := methods.SimulateTransactionRequest{InvokeHostFunctionOp: "invalid"}
	var result methods.SimulateTransactionResponse
	err := cli.CallResult(context.Background(), "simulateTransaction", request, &result)
	assert.NoError(t, err)
	assert.Equal(
		t,
		"Could unmarshal invoke host function op: decoding HostFunction: decoding HostFunction: xdr:DecodeInt: unexpected EOF while decoding 4 bytes - read: '[138 123 218]'",
		result.Error,
	)
}
