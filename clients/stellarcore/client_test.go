package stellarcore

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/jarcoal/httpmock"
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
	mockResp := proto.GetLedgerEntryRawResponse{
		Ledger: 1215, // checkpoint align on expected request
		Entries: []proto.RawLedgerEntryResponse{
			{
				Entry: "pretend this is XDR lol",
			},
			{
				Entry: "pretend this is another XDR lol",
			},
		},
	}

	var key xdr.LedgerKey
	acc, err := xdr.AddressToAccountId(keypair.MustRandom().Address())
	require.NoError(t, err)
	key.SetAccount(acc)

	// happy path - fetch an entry
	ce := hmock.On("POST", "http://localhost:11626/getledgerentryraw")
	hmock.RegisterResponder(
		"POST",
		"http://localhost:11626/getledgerentryraw",
		func(r *http.Request) (*http.Response, error) {
			// Ensure the request has the correct POST body
			requestData, ierr := io.ReadAll(r.Body)
			require.NoError(t, ierr)

			keyB64, ierr := key.MarshalBinaryBase64()
			require.NoError(t, ierr)
			expected := fmt.Sprintf("key=%s&ledgerSeq=1234", url.QueryEscape(keyB64))
			require.Equal(t, expected, string(requestData))

			resp, ierr := httpmock.NewJsonResponse(http.StatusOK, &mockResp)
			require.NoError(t, ierr)
			ce.Return(httpmock.ResponderFromResponse(resp))
			return resp, nil
		})

	resp, err := c.GetLedgerEntryRaw(context.Background(), 1234, key)
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.EqualValues(t, 1215, resp.Ledger)
	require.Len(t, resp.Entries, 2)
	require.Equal(t, "pretend this is XDR lol", resp.Entries[0].Entry)
	require.Equal(t, "pretend this is another XDR lol", resp.Entries[1].Entry)
}
