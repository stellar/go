package horizonclient

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/support/http/httptest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperationRequestBuildUrl(t *testing.T) {
	op := OperationRequest{endpoint: "operations"}
	endpoint, err := op.BuildURL()

	// It should return valid all operations endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "operations", endpoint)

	op = OperationRequest{ForAccount: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU", endpoint: "operations"}
	endpoint, err = op.BuildURL()

	// It should return valid account operations endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/operations", endpoint)

	op = OperationRequest{ForLedger: 123, endpoint: "operations"}
	endpoint, err = op.BuildURL()

	// It should return valid ledger operations endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "ledgers/123/operations", endpoint)

	op = OperationRequest{forOperationID: "123", endpoint: "operations"}
	endpoint, err = op.BuildURL()

	// It should return valid operation operations endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "operations/123", endpoint)

	op = OperationRequest{ForTransaction: "123", endpoint: "payments"}
	endpoint, err = op.BuildURL()

	// It should return valid transaction payments endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "transactions/123/payments", endpoint)

	op = OperationRequest{ForLedger: 123, forOperationID: "789", endpoint: "operations"}
	endpoint, err = op.BuildURL()

	// error case: too many parameters for building any operation endpoint
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "invalid request: too many parameters")
	}

	op = OperationRequest{Cursor: "123456", Limit: 30, Order: OrderAsc, endpoint: "operations"}
	endpoint, err = op.BuildURL()
	// It should return valid all operations endpoint with query params and no errors
	require.NoError(t, err)
	assert.Equal(t, "operations?cursor=123456&limit=30&order=asc", endpoint)

	op = OperationRequest{Cursor: "123456", Limit: 30, Order: OrderAsc, endpoint: "payments"}
	endpoint, err = op.BuildURL()
	// It should return valid all operations endpoint with query params and no errors
	require.NoError(t, err)
	assert.Equal(t, "payments?cursor=123456&limit=30&order=asc", endpoint)
}

func ExampleClient_StreamOperations() {
	client := DefaultTestNetClient
	// operations for an account
	opRequest := OperationRequest{ForAccount: "GAIH3ULLFQ4DGSECF2AR555KZ4KNDGEKN4AFI4SU2M7B43MGK3QJZNSR", Cursor: "760209215489"}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Stop streaming after 60 seconds.
		time.Sleep(60 * time.Second)
		cancel()
	}()

	printHandler := func(op operations.Operation) {
		fmt.Println(op)
	}
	err := client.StreamOperations(ctx, opRequest, printHandler)
	if err != nil {
		fmt.Println(err)
	}
}

func ExampleClient_StreamPayments() {
	client := DefaultTestNetClient
	// all payments
	opRequest := OperationRequest{Cursor: "760209215489"}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Stop streaming after 60 seconds.
		time.Sleep(60 * time.Second)
		cancel()
	}()

	printHandler := func(op operations.Operation) {
		fmt.Println(op)
	}
	err := client.StreamPayments(ctx, opRequest, printHandler)
	if err != nil {
		fmt.Println(err)
	}
}

func TestOperationRequestStreamOperations(t *testing.T) {
	hmock := httptest.NewClient()
	client := &Client{
		HorizonURL: "https://localhost/",
		HTTP:       hmock,
	}

	// All operations
	operationRequest := OperationRequest{}
	ctx, cancel := context.WithCancel(context.Background())

	hmock.On(
		"GET",
		"https://localhost/operations?cursor=now",
	).ReturnString(200, operationStreamResponse)

	operationStream := make([]operations.Operation, 1)
	err := client.StreamOperations(ctx, operationRequest, func(op operations.Operation) {
		operationStream[0] = op
		cancel()
	})

	if assert.NoError(t, err) {
		assert.Equal(t, operationStream[0].GetType(), "create_account")
	}

	// Account payments
	operationRequest = OperationRequest{ForAccount: "GAIH3ULLFQ4DGSECF2AR555KZ4KNDGEKN4AFI4SU2M7B43MGK3QJZNSR"}
	ctx, cancel = context.WithCancel(context.Background())

	hmock.On(
		"GET",
		"https://localhost/accounts/GAIH3ULLFQ4DGSECF2AR555KZ4KNDGEKN4AFI4SU2M7B43MGK3QJZNSR/payments?cursor=now",
	).ReturnString(200, operationStreamResponse)

	err = client.StreamPayments(ctx, operationRequest, func(op operations.Operation) {
		operationStream[0] = op
		cancel()
	})

	if assert.NoError(t, err) {
		payment, ok := operationStream[0].(operations.CreateAccount)
		assert.Equal(t, ok, true)
		assert.Equal(t, payment.Funder, "GAIH3ULLFQ4DGSECF2AR555KZ4KNDGEKN4AFI4SU2M7B43MGK3QJZNSR")
	}

	// test connection error
	operationRequest = OperationRequest{}
	ctx, cancel = context.WithCancel(context.Background())

	hmock.On(
		"GET",
		"https://localhost/operations?cursor=now",
	).ReturnString(500, operationStreamResponse)

	err = client.StreamOperations(ctx, operationRequest, func(op operations.Operation) {
		operationStream[0] = op
		cancel()
	})

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "got bad HTTP status code 500")
	}
}

var operationStreamResponse = `data: {"_links":{"self":{"href":"https://horizon-testnet.stellar.org/operations/4934917427201"},"transaction":{"href":"https://horizon-testnet.stellar.org/transactions/1c1449106a54cccd8a2ec2094815ad9db30ae54c69c3309dd08d13fdb8c749de"},"effects":{"href":"https://horizon-testnet.stellar.org/operations/4934917427201/effects"},"succeeds":{"href":"https://horizon-testnet.stellar.org/effects?order=desc\u0026cursor=4934917427201"},"precedes":{"href":"https://horizon-testnet.stellar.org/effects?order=asc\u0026cursor=4934917427201"}},"id":"4934917427201","paging_token":"4934917427201","transaction_successful":true,"source_account":"GAIH3ULLFQ4DGSECF2AR555KZ4KNDGEKN4AFI4SU2M7B43MGK3QJZNSR","type":"create_account","type_i":0,"created_at":"2019-02-27T11:32:39Z","transaction_hash":"1c1449106a54cccd8a2ec2094815ad9db30ae54c69c3309dd08d13fdb8c749de","starting_balance":"10000.0000000","funder":"GAIH3ULLFQ4DGSECF2AR555KZ4KNDGEKN4AFI4SU2M7B43MGK3QJZNSR","account":"GDBLBBDIUULY3HGIKXNK6WVBISY7DCNCDA45EL7NTXWX5R4UZ26HGMGS"}
`
