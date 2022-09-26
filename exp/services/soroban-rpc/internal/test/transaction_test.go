package test

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/code"
	"github.com/creachadair/jrpc2/jhttp"
	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/exp/services/soroban-rpc/internal/methods"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
)

func TestSendTransactionSucceeds(t *testing.T) {
	test := NewTest(t)

	ch := jhttp.NewChannel(test.server.URL, nil)
	cli := jrpc2.NewClient(ch, nil)

	kp := keypair.Root(StandaloneNetworkPassphrase)
	address := kp.Address()
	account := txnbuild.NewSimpleAccount(address, 0)

	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount:        &account,
		IncrementSequenceNum: true,
		Operations: []txnbuild.Operation{
			&txnbuild.SetOptions{HomeDomain: txnbuild.NewHomeDomain("soroban.com")},
		},
		BaseFee: txnbuild.MinBaseFee,
		Preconditions: txnbuild.Preconditions{
			TimeBounds: txnbuild.NewInfiniteTimeout(),
		},
	})
	assert.NoError(t, err)
	tx, err = tx.Sign(StandaloneNetworkPassphrase, kp)
	assert.NoError(t, err)
	b64, err := tx.Base64()
	assert.NoError(t, err)

	request := methods.SendTransactionRequest{Transaction: b64}
	var result methods.SendTransactionResponse
	err = cli.CallResult(context.Background(), "sendTransaction", request, &result)
	assert.NoError(t, err)

	expectedHash, err := tx.HashHex(StandaloneNetworkPassphrase)
	assert.NoError(t, err)

	assert.Equal(t, methods.SendTransactionResponse{ID: expectedHash}, result)

	getTransactionStatus(t, cli, expectedHash, func(t *testing.T, response methods.TransactionStatusResponse) {
		assert.Equal(t, methods.TransactionComplete, response.Status)
		assert.Equal(t, expectedHash, response.ID)
		assert.Equal(t, true, response.Result.Successful)

		accountInfoRequest := methods.AccountRequest{
			Address: address,
		}
		var accountInfoResponse methods.AccountInfo
		err = cli.CallResult(context.Background(), "getAccount", accountInfoRequest, &accountInfoResponse)
		assert.NoError(t, err)
		assert.Equal(t, methods.AccountInfo{ID: address, Sequence: 1}, accountInfoResponse)
	})
}

func getTransactionStatus(t *testing.T, cli *jrpc2.Client, hash string, f func(*testing.T, methods.TransactionStatusResponse)) {
	for i := 0; i < 60; i++ {
		var result methods.TransactionStatusResponse
		request := methods.GetTransactionStatusRequest{Hash: hash}
		err := cli.CallResult(context.Background(), "getTransactionStatus", request, &result)
		assert.NoError(t, err)

		if result.Status == methods.TransactionPending {
			time.Sleep(time.Second)
			continue
		}

		f(t, result)
		return
	}
	t.Fatal("getTransactionStatus timed out")
}

func assertTransactionStatusError(t *testing.T, cli *jrpc2.Client, hash string, f func(*testing.T, error)) {
	for i := 0; i < 60; i++ {
		var result methods.TransactionStatusResponse
		request := methods.GetTransactionStatusRequest{Hash: hash}
		err := cli.CallResult(context.Background(), "getTransactionStatus", request, &result)

		if err == nil && result.Status == methods.TransactionPending {
			time.Sleep(time.Second)
			continue
		} else if err == nil {
			t.Fatalf("expected transaction to fail but got %v", result)
		}

		f(t, err)
		return
	}
	t.Fatal("getTransactionStatus timed out")
}

func TestSendTransactionBadSequence(t *testing.T) {
	test := NewTest(t)

	ch := jhttp.NewChannel(test.server.URL, nil)
	cli := jrpc2.NewClient(ch, nil)

	kp := keypair.Root(StandaloneNetworkPassphrase)
	address := kp.Address()
	account := txnbuild.NewSimpleAccount(address, 0)

	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount: &account,
		Operations: []txnbuild.Operation{
			&txnbuild.SetOptions{HomeDomain: txnbuild.NewHomeDomain("soroban.com")},
		},
		BaseFee: txnbuild.MinBaseFee,
		Preconditions: txnbuild.Preconditions{
			TimeBounds: txnbuild.NewInfiniteTimeout(),
		},
	})
	assert.NoError(t, err)
	tx, err = tx.Sign(StandaloneNetworkPassphrase, kp)
	assert.NoError(t, err)
	b64, err := tx.Base64()
	assert.NoError(t, err)

	request := methods.SendTransactionRequest{Transaction: b64}
	var result methods.SendTransactionResponse
	err = cli.CallResult(context.Background(), "sendTransaction", request, &result)
	assert.NoError(t, err)

	expectedHash, err := tx.HashHex(StandaloneNetworkPassphrase)
	assert.NoError(t, err)

	assert.Equal(t, methods.SendTransactionResponse{ID: expectedHash}, result)

	assertTransactionStatusError(t, cli, expectedHash, func(t *testing.T, err error) {
		rpcErr := err.(*jrpc2.Error)
		assert.Equal(t, "Transaction Failed", rpcErr.Message)
		assert.Equal(t, code.InvalidRequest, rpcErr.Code)
		assert.Equal(
			t,
			"{\"envelope_xdr\":\"AAAAAgAAAABzdv3ojkzWHMD7KUoXhrPx0GH18vHKV0ZfqpMiEblG1gAAAGQAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAABQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAC3Nvcm9iYW4uY29tAAAAAAAAAAAAAAAAARG5RtYAAABAvSifLEf7tP1tZ5sN/GYzqNmZnGV2BnMHHSaaRLSC7tzKu6vedJrdFX/u8iJRQZICF4T7FQQGl2BFEMmdF+8uCg==\",\"result_codes\":{\"transaction\":\"tx_bad_seq\"},\"result_xdr\":\"AAAAAAAAAAD////7AAAAAA==\"}",
			string(rpcErr.Data),
		)
	})

	// assert that the transaction was not included in any ledger
	accountInfoRequest := methods.AccountRequest{
		Address: address,
	}
	var accountInfoResponse methods.AccountInfo
	err = cli.CallResult(context.Background(), "getAccount", accountInfoRequest, &accountInfoResponse)
	assert.NoError(t, err)
	assert.Equal(t, methods.AccountInfo{ID: address, Sequence: 0}, accountInfoResponse)
}

func TestSendTransactionFailed(t *testing.T) {
	test := NewTest(t)

	ch := jhttp.NewChannel(test.server.URL, nil)
	cli := jrpc2.NewClient(ch, nil)

	kp := keypair.Root(StandaloneNetworkPassphrase)
	address := kp.Address()
	account := txnbuild.NewSimpleAccount(address, 0)

	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount:        &account,
		IncrementSequenceNum: true,
		Operations: []txnbuild.Operation{
			&txnbuild.Payment{
				Destination: "GD5KD2KEZJIGTC63IGW6UMUSMVUVG5IHG64HUTFWCHVZH2N2IBOQN7PS",
				Amount:      amount.StringFromInt64(math.MaxInt64),
				Asset:       txnbuild.NativeAsset{},
			},
		},
		BaseFee: txnbuild.MinBaseFee,
		Preconditions: txnbuild.Preconditions{
			TimeBounds: txnbuild.NewInfiniteTimeout(),
		},
	})
	assert.NoError(t, err)
	tx, err = tx.Sign(StandaloneNetworkPassphrase, kp)
	assert.NoError(t, err)
	b64, err := tx.Base64()
	assert.NoError(t, err)

	request := methods.SendTransactionRequest{Transaction: b64}
	var result methods.SendTransactionResponse
	err = cli.CallResult(context.Background(), "sendTransaction", request, &result)
	assert.NoError(t, err)

	expectedHash, err := tx.HashHex(StandaloneNetworkPassphrase)
	assert.NoError(t, err)

	assert.Equal(t, methods.SendTransactionResponse{ID: expectedHash}, result)

	getTransactionStatus(t, cli, expectedHash, func(t *testing.T, response methods.TransactionStatusResponse) {
		assert.Equal(t, methods.TransactionComplete, response.Status)
		assert.Equal(t, expectedHash, response.ID)
		assert.Equal(t, false, response.Result.Successful)

		// assert that the transaction was not included in any ledger
		accountInfoRequest := methods.AccountRequest{
			Address: address,
		}
		var accountInfoResponse methods.AccountInfo
		err = cli.CallResult(context.Background(), "getAccount", accountInfoRequest, &accountInfoResponse)
		assert.NoError(t, err)
		assert.Equal(t, methods.AccountInfo{ID: address, Sequence: 1}, accountInfoResponse)
	})
}
