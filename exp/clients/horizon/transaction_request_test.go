package horizonclient

import (
	"context"
	"fmt"
	"testing"
	"time"

	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/support/http/httptest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionRequestBuildUrl(t *testing.T) {
	tr := TransactionRequest{}
	endpoint, err := tr.BuildURL()

	// It should return valid all transactions endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "transactions", endpoint)

	tr = TransactionRequest{ForAccount: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU"}
	endpoint, err = tr.BuildURL()

	// It should return valid account transactions endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/transactions", endpoint)

	tr = TransactionRequest{ForLedger: 123}
	endpoint, err = tr.BuildURL()

	// It should return valid ledger transactions endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "ledgers/123/transactions", endpoint)

	tr = TransactionRequest{forTransactionHash: "123"}
	endpoint, err = tr.BuildURL()

	// It should return valid operation transactions endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "transactions/123", endpoint)

	tr = TransactionRequest{ForLedger: 123, forTransactionHash: "789"}
	endpoint, err = tr.BuildURL()

	// error case: too many parameters for building any operation endpoint
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "invalid request: too many parameters")
	}

	tr = TransactionRequest{Cursor: "123456", Limit: 30, Order: OrderAsc, IncludeFailed: true}
	endpoint, err = tr.BuildURL()
	// It should return valid all transactions endpoint with query params and no errors
	require.NoError(t, err)
	assert.Equal(t, "transactions?cursor=123456&include_failed=true&limit=30&order=asc", endpoint)

}

func ExampleClient_StreamTransactions() {
	client := DefaultTestNetClient
	// all transactions
	transactionRequest := TransactionRequest{Cursor: "760209215489"}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Stop streaming after 60 seconds.
		time.Sleep(60 * time.Second)
		cancel()
	}()

	printHandler := func(tr hProtocol.Transaction) {
		fmt.Println(tr)
	}
	err := client.StreamTransactions(ctx, transactionRequest, printHandler)
	if err != nil {
		fmt.Println(err)
	}
}

func TestTransactionRequestStreamTransactions(t *testing.T) {
	hmock := httptest.NewClient()
	client := &Client{
		HorizonURL: "https://localhost/",
		HTTP:       hmock,
	}

	// all transactions
	trRequest := TransactionRequest{}
	ctx, cancel := context.WithCancel(context.Background())

	hmock.On(
		"GET",
		"https://localhost/transactions?cursor=now",
	).ReturnString(200, txStreamResponse)

	transactions := make([]hProtocol.Transaction, 1)
	err := client.StreamTransactions(ctx, trRequest, func(tr hProtocol.Transaction) {
		transactions[0] = tr
		cancel()
	})

	if assert.NoError(t, err) {
		assert.Equal(t, transactions[0].Hash, "1534f6507420c6871b557cc2fc800c29fb1ed1e012e694993ffe7a39c824056e")
		assert.Equal(t, transactions[0].Account, "GAIH3ULLFQ4DGSECF2AR555KZ4KNDGEKN4AFI4SU2M7B43MGK3QJZNSR")
	}

	// transactions for accounts
	trRequest = TransactionRequest{ForAccount: "GAIH3ULLFQ4DGSECF2AR555KZ4KNDGEKN4AFI4SU2M7B43MGK3QJZNSR"}
	ctx, cancel = context.WithCancel(context.Background())

	hmock.On(
		"GET",
		"https://localhost/accounts/GAIH3ULLFQ4DGSECF2AR555KZ4KNDGEKN4AFI4SU2M7B43MGK3QJZNSR/transactions?cursor=now",
	).ReturnString(200, txStreamResponse)

	transactions = make([]hProtocol.Transaction, 1)
	err = client.StreamTransactions(ctx, trRequest, func(tr hProtocol.Transaction) {
		transactions[0] = tr
		cancel()
	})

	if assert.NoError(t, err) {
		assert.Equal(t, transactions[0].Hash, "1534f6507420c6871b557cc2fc800c29fb1ed1e012e694993ffe7a39c824056e")
		assert.Equal(t, transactions[0].Account, "GAIH3ULLFQ4DGSECF2AR555KZ4KNDGEKN4AFI4SU2M7B43MGK3QJZNSR")
	}

	// test error
	trRequest = TransactionRequest{}
	ctx, cancel = context.WithCancel(context.Background())

	hmock.On(
		"GET",
		"https://localhost/transactions?cursor=now",
	).ReturnString(500, txStreamResponse)

	transactions = make([]hProtocol.Transaction, 1)
	err = client.StreamTransactions(ctx, trRequest, func(tr hProtocol.Transaction) {
		cancel()
	})

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "got bad HTTP status code 500")
	}
}

var txStreamResponse = `data: {"_links":{"self":{"href":"https://horizon-testnet.stellar.org/transactions/1534f6507420c6871b557cc2fc800c29fb1ed1e012e694993ffe7a39c824056e"},"account":{"href":"https://horizon-testnet.stellar.org/accounts/GAIH3ULLFQ4DGSECF2AR555KZ4KNDGEKN4AFI4SU2M7B43MGK3QJZNSR"},"ledger":{"href":"https://horizon-testnet.stellar.org/ledgers/607387"},"operations":{"href":"https://horizon-testnet.stellar.org/transactions/1534f6507420c6871b557cc2fc800c29fb1ed1e012e694993ffe7a39c824056e/operations{?cursor,limit,order}","templated":true},"effects":{"href":"https://horizon-testnet.stellar.org/transactions/1534f6507420c6871b557cc2fc800c29fb1ed1e012e694993ffe7a39c824056e/effects{?cursor,limit,order}","templated":true},"precedes":{"href":"https://horizon-testnet.stellar.org/transactions?order=asc\u0026cursor=2608707301036032"},"succeeds":{"href":"https://horizon-testnet.stellar.org/transactions?order=desc\u0026cursor=2608707301036032"}},"id":"1534f6507420c6871b557cc2fc800c29fb1ed1e012e694993ffe7a39c824056e","paging_token":"2608707301036032","successful":true,"hash":"1534f6507420c6871b557cc2fc800c29fb1ed1e012e694993ffe7a39c824056e","ledger":607387,"created_at":"2019-04-04T12:07:03Z","source_account":"GAIH3ULLFQ4DGSECF2AR555KZ4KNDGEKN4AFI4SU2M7B43MGK3QJZNSR","source_account_sequence":"4660039930473","fee_paid":100,"operation_count":1,"envelope_xdr":"AAAAABB90WssODNIgi6BHveqzxTRmIpvAFRyVNM+Hm2GVuCcAAAAZAAABD0ABlJpAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAmLuzasXDMqsqgFK4xkbLxJLzmQQzkiCF2SnKPD+b1TsAAAAXSHboAAAAAAAAAAABhlbgnAAAAECqxhXduvtzs65keKuTzMtk76cts2WeVB2pZKYdlxlOb1EIbOpFhYizDSXVfQlAvvg18qV6oNRr7ls4nnEm2YIK","result_xdr":"AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=","result_meta_xdr":"AAAAAQAAAAIAAAADAAlEmwAAAAAAAAAAEH3Rayw4M0iCLoEe96rPFNGYim8AVHJU0z4ebYZW4JwBT3aiixBA2AAABD0ABlJoAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAlEmwAAAAAAAAAAEH3Rayw4M0iCLoEe96rPFNGYim8AVHJU0z4ebYZW4JwBT3aiixBA2AAABD0ABlJpAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAAwAAAAMACUSbAAAAAAAAAAAQfdFrLDgzSIIugR73qs8U0ZiKbwBUclTTPh5thlbgnAFPdqKLEEDYAAAEPQAGUmkAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEACUSbAAAAAAAAAAAQfdFrLDgzSIIugR73qs8U0ZiKbwBUclTTPh5thlbgnAFPdotCmVjYAAAEPQAGUmkAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAACUSbAAAAAAAAAACYu7NqxcMyqyqAUrjGRsvEkvOZBDOSIIXZKco8P5vVOwAAABdIdugAAAlEmwAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==","fee_meta_xdr":"AAAAAgAAAAMACUSaAAAAAAAAAAAQfdFrLDgzSIIugR73qs8U0ZiKbwBUclTTPh5thlbgnAFPdqKLEEE8AAAEPQAGUmgAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEACUSbAAAAAAAAAAAQfdFrLDgzSIIugR73qs8U0ZiKbwBUclTTPh5thlbgnAFPdqKLEEDYAAAEPQAGUmgAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==","memo_type":"none","signatures":["qsYV3br7c7OuZHirk8zLZO+nLbNlnlQdqWSmHZcZTm9RCGzqRYWIsw0l1X0JQL74NfKleqDUa+5bOJ5xJtmCCg=="]}
`
