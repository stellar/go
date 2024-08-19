package stellarcore

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stellar/go/keypair"
	proto "github.com/stellar/go/protocols/stellarcore"
	"github.com/stellar/go/support/http/httptest"
	"github.com/stellar/go/xdr"
)

func TestSubmitTransaction(t *testing.T) {
	hmock := httptest.NewClient()
	c := &Client{HTTP: hmock, URL: "http://localhost:11626"}

	// happy path - new transaction
	hmock.On("GET", "http://localhost:11626/tx?blob=foo").
		ReturnJSON(http.StatusOK, proto.TXResponse{
			Status: proto.TXStatusPending,
		})

	resp, err := c.SubmitTransaction(context.Background(), "foo")

	if assert.NoError(t, err) {
		assert.Equal(t, proto.TXStatusPending, resp.Status)
	}
}

func TestSubmitTransactionError(t *testing.T) {
	hmock := httptest.NewClient()
	c := &Client{HTTP: hmock, URL: "http://localhost:11626"}

	// happy path - new transaction
	hmock.On("GET", "http://localhost:11626/tx?blob=foo").
		ReturnString(
			200,
			`{"diagnostic_events":"AAAAAQAAAAAAAAAAAAAAAAAAAAIAAAAAAAAAAgAAAA8AAAAFZXJyb3IAAAAAAAACAAAAAwAAAAUAAAAQAAAAAQAAAAMAAAAOAAAAU3RyYW5zYWN0aW9uIGBzb3JvYmFuRGF0YS5yZXNvdXJjZUZlZWAgaXMgbG93ZXIgdGhhbiB0aGUgYWN0dWFsIFNvcm9iYW4gcmVzb3VyY2UgZmVlAAAAAAUAAAAAAAEJcwAAAAUAAAAAAAG6fA==","error":"AAAAAAABCdf////vAAAAAA==","status":"ERROR"}`,
		)

	resp, err := c.SubmitTransaction(context.Background(), "foo")

	if assert.NoError(t, err) {
		assert.Equal(t, "ERROR", resp.Status)
		assert.Equal(t, resp.Error, "AAAAAAABCdf////vAAAAAA==")
		assert.Equal(t, "AAAAAQAAAAAAAAAAAAAAAAAAAAIAAAAAAAAAAgAAAA8AAAAFZXJyb3IAAAAAAAACAAAAAwAAAAUAAAAQAAAAAQAAAAMAAAAOAAAAU3RyYW5zYWN0aW9uIGBzb3JvYmFuRGF0YS5yZXNvdXJjZUZlZWAgaXMgbG93ZXIgdGhhbiB0aGUgYWN0dWFsIFNvcm9iYW4gcmVzb3VyY2UgZmVlAAAAAAUAAAAAAAEJcwAAAAUAAAAAAAG6fA==", resp.DiagnosticEvents)
	}
}

func TestManualClose(t *testing.T) {
	hmock := httptest.NewClient()
	c := &Client{HTTP: hmock, URL: "http://localhost:11626"}

	// happy path - new transaction
	hmock.On("GET", "http://localhost:11626/manualclose").
		ReturnString(http.StatusOK, "Manually triggered a ledger close with sequence number 7")

	err := c.ManualClose(context.Background())

	assert.NoError(t, err)
}

func TestManualClose_NotAvailable(t *testing.T) {
	hmock := httptest.NewClient()
	c := &Client{HTTP: hmock, URL: "http://localhost:11626"}

	// happy path - new transaction
	hmock.On("GET", "http://localhost:11626/manualclose").
		ReturnString(http.StatusOK,
			`{"exception": "Set MANUAL_CLOSE=true"}`,
		)

	err := c.ManualClose(context.Background())

	assert.EqualError(t, err, "exception in response: Set MANUAL_CLOSE=true")
}

func TestGetLedgerEntries(t *testing.T) {
	hmock := httptest.NewClient()
	c := &Client{HTTP: hmock, URL: "http://localhost:11626"}

	// build a fake response body
	mockResp := proto.GetLedgerEntriesResponse{
		Ledger: 1215, // checkpoint align on expected request
		Entries: []proto.LedgerEntryResponse{
			{
				Entry: "pretend this is XDR lol",
				State: proto.DeadState,
			},
			{
				Entry: "pretend this is another XDR lol",
				State: proto.ArchivedStateNoProof,
			},
		},
	}

	// happy path - fetch an entry
	hmock.On("POST", "http://localhost:11626/getledgerentry").
		ReturnJSON(http.StatusOK, &mockResp)

	var key xdr.LedgerKey
	acc, err := xdr.AddressToAccountId(keypair.MustRandom().Address())
	require.NoError(t, err)
	key.SetAccount(acc)

	resp, err := c.GetLedgerEntries(context.Background(), 1234, key)
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.EqualValues(t, 1215, resp.Ledger)
	require.Len(t, resp.Entries, 2)
	require.Equal(t, resp.Entries[0].State, proto.DeadState)
	require.Equal(t, resp.Entries[1].State, proto.ArchivedStateNoProof)
}
