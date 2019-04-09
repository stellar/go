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

func TestLedgerRequestBuildUrl(t *testing.T) {
	lr := LedgerRequest{}
	endpoint, err := lr.BuildUrl()

	// It should return valid all ledgers endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "ledgers", endpoint)

	lr = LedgerRequest{forSequence: 123}
	endpoint, err = lr.BuildUrl()

	// It should return valid ledger detail endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "ledgers/123", endpoint)

	lr = LedgerRequest{forSequence: 123, Cursor: "now", Order: OrderDesc}
	endpoint, err = lr.BuildUrl()

	// It should return valid ledger detail endpoint, with no cursor or order
	require.NoError(t, err)
	assert.Equal(t, "ledgers/123", endpoint)

	lr = LedgerRequest{Cursor: "now", Order: OrderDesc}
	endpoint, err = lr.BuildUrl()

	// It should return valid ledgers endpoint, with cursor and order
	require.NoError(t, err)
	assert.Equal(t, "ledgers?cursor=now&order=desc", endpoint)
}

func TestLedgerDetail(t *testing.T) {
	hmock := httptest.NewClient()
	client := &Client{
		HorizonURL: "https://localhost/",
		HTTP:       hmock,
	}

	// invalid parameters
	var sequence uint32 = 0
	hmock.On(
		"GET",
		"https://localhost/ledgers/",
	).ReturnString(200, ledgerResponse)

	_, err := client.LedgerDetail(sequence)
	// error case: invalid sequence
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Invalid sequence number provided")
	}

	// happy path
	hmock.On(
		"GET",
		"https://localhost/ledgers/69859",
	).ReturnString(200, ledgerResponse)

	sequence = 69859
	ledger, err := client.LedgerDetail(sequence)
	ftc := int32(1)

	if assert.NoError(t, err) {
		assert.Equal(t, ledger.ID, "71a40c0581d8d7c1158e1d9368024c5f9fd70de17a8d277cdd96781590cc10fb")
		assert.Equal(t, ledger.PT, "300042120331264")
		assert.Equal(t, ledger.Sequence, int32(69859))
		assert.Equal(t, ledger.FailedTransactionCount, &ftc)
	}

	// failure response
	hmock.On(
		"GET",
		"https://localhost/ledgers/69859",
	).ReturnString(404, notFoundResponse)

	_, err = client.LedgerDetail(sequence)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Horizon error")
		horizonError, ok := err.(*Error)
		assert.Equal(t, ok, true)
		assert.Equal(t, horizonError.Problem.Title, "Resource Missing")
	}

	// connection error
	hmock.On(
		"GET",
		"https://localhost/ledgers/69859",
	).ReturnError("http.Client error")

	_, err = client.LedgerDetail(sequence)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "http.Client error")
		_, ok := err.(*Error)
		assert.Equal(t, ok, false)
	}
}

func ExampleClient_StreamLedgers() {
	client := DefaultTestNetClient
	// all ledgers from now
	ledgerRequest := LedgerRequest{}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Stop streaming after 60 seconds.
		time.Sleep(60 * time.Second)
		cancel()
	}()

	printHandler := func(ledger hProtocol.Ledger) {
		fmt.Println(ledger)
	}
	err := client.StreamLedgers(ctx, ledgerRequest, printHandler)
	if err != nil {
		fmt.Println(err)
	}
}
func TestLedgerRequestStreamLedgers(t *testing.T) {
	hmock := httptest.NewClient()
	client := &Client{
		HorizonURL: "https://localhost/",
		HTTP:       hmock,
	}
	ledgerRequest := LedgerRequest{Cursor: "1"}
	ctx, cancel := context.WithCancel(context.Background())

	hmock.On(
		"GET",
		"https://localhost/ledgers?cursor=1",
	).ReturnString(200, ledgerStreamResponse)

	ledgers := make([]hProtocol.Ledger, 1)
	err := client.StreamLedgers(ctx, ledgerRequest, func(ledger hProtocol.Ledger) {
		ledgers[0] = ledger
		cancel()

	})

	if assert.NoError(t, err) {
		assert.Equal(t, ledgers[0].Sequence, int32(560339))
	}

	// test error
	ledgerRequest = LedgerRequest{}
	ctx, cancel = context.WithCancel(context.Background())

	hmock.On(
		"GET",
		"https://localhost/ledgers?cursor=now",
	).ReturnString(500, ledgerStreamResponse)

	err = client.StreamLedgers(ctx, ledgerRequest, func(ledger hProtocol.Ledger) {
		ledgers[0] = ledger
		cancel()

	})

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Got bad HTTP status code 500")

	}

}

var ledgerStreamResponse = `data: {"_links":{"self":{"href":"https://horizon-testnet.stellar.org/ledgers/560339"},"transactions":{"href":"https://horizon-testnet.stellar.org/ledgers/560339/transactions{?cursor,limit,order}","templated":true},"operations":{"href":"https://horizon-testnet.stellar.org/ledgers/560339/operations{?cursor,limit,order}","templated":true},"payments":{"href":"https://horizon-testnet.stellar.org/ledgers/560339/payments{?cursor,limit,order}","templated":true},"effects":{"href":"https://horizon-testnet.stellar.org/ledgers/560339/effects{?cursor,limit,order}","templated":true}},"id":"66f4d95dab22dbc422585cc4b011716014e81df3599cee8db9c776cfc3a31e93","paging_token":"2406637679673344","hash":"66f4d95dab22dbc422585cc4b011716014e81df3599cee8db9c776cfc3a31e93","prev_hash":"6071f1e52a6bf37aba3f7437081577eafe69f78593c465fc5028c46a4746dda3","sequence":560339,"successful_transaction_count":5,"failed_transaction_count":1,"operation_count":44,"closed_at":"2019-04-01T16:47:05Z","total_coins":"100057227213.0436903","fee_pool":"57227816.6766542","base_fee_in_stroops":100,"base_reserve_in_stroops":5000000,"max_tx_set_size":100,"protocol_version":10,"header_xdr":"AAAACmBx8eUqa/N6uj90NwgVd+r+afeFk8Rl/FAoxGpHRt2jdIn+3X+/O3PFUUZ8Tgy4rfD1oNamR+9NMOCM2V6ndksAAAAAXKJAiQAAAAAAAAAAPyIIYU6Y37lve/MwZls1vmbgxgFdx93hdzOn6g8kHhQ1BS9aAKuXtApQoE3gKpjQ5ze0H9qUruyOUsbM776zXQAIjNMN4r8uJHCvJwACCHvk18POAAAAAwAAAAAAQZnVAAAAZABMS0AAAABkkiIcXkjaTtc9zTQBn0o72CUBe3u+2Mz7W6dgkvkYcJJle8JCNmXx5HcRlDSHJzzBShc8C3rQUIsIuJ93eoBMgHeYAzfholE8hjvrHrqoHq8jfPowxj1FGD6HaUPD1PHTcBXmf0U0cs2Ki0NBDDKNcwKC84nUPdumCkdAxSuEzn4AAAAA"}
`
