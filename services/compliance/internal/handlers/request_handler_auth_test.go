package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stellar/go/clients/stellartoml"
	"github.com/stellar/go/protocols/compliance"
	"github.com/stellar/go/support/http/httptest"
	"github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/stellar/go/services/compliance/internal/config"
	"github.com/stellar/go/services/compliance/internal/mocks"
	"github.com/stellar/go/services/compliance/internal/test"
)

func TestRequestHandlerAuthInvalidParams(t *testing.T) {
	var rhconfig = &config.Config{
		NetworkPassphrase: "Test SDF Network ; September 2015",
		Keys: config.Keys{
			// GBYJZW5XFAI6XV73H5SAIUYK6XZI4CGGVBUBO3ANA2SV7KKDAXTV6AEB
			SigningSeed: "SDWTLFPALQSP225BSMX7HPZ7ZEAYSUYNDLJ5QI3YGVBNRUIIELWH3XUV",
		},
		Callbacks: config.Callbacks{
			FetchInfo: "http://fetch_info",
		},
	}

	var mockHTTPClient = new(mocks.MockHTTPClient)
	var mockDatabase = new(mocks.MockDatabase)
	var mockFederationResolver = new(mocks.MockFederationResolver)
	var mockSignerVerifier = new(mocks.MockSignerVerifier)
	var mockStellartomlResolver = new(mocks.MockStellartomlResolver)
	var mockNonceGenerator = new(mocks.MockNonceGenerator)

	requestHandler := RequestHandler{
		Config:                  rhconfig,
		Client:                  mockHTTPClient,
		Database:                mockDatabase,
		FederationResolver:      mockFederationResolver,
		SignatureSignerVerifier: mockSignerVerifier,
		StellarTomlResolver:     mockStellartomlResolver,
		NonceGenerator:          mockNonceGenerator,
	}

	testServer := httptest.NewServer(t, http.HandlerFunc(requestHandler.HandlerAuth))
	defer testServer.Close()

	// When signature is invalid
	mockStellartomlResolver.On(
		"GetStellarTomlByAddress",
		"alice*stellar.org",
	).Return(&stellartoml.Response{
		SigningKey: "GBYJZW5XFAI6XV73H5SAIUYK6XZI4CGGVBUBO3ANA2SV7KKDAXTV6AEB",
	}, nil).Once()

	attachment := compliance.Attachment{}
	attachHash, err := attachment.Hash()
	require.NoError(t, err)
	attachmentJSON, err := attachment.Marshal()
	require.NoError(t, err)

	txnOp := &txnbuild.Payment{
		Destination:   "GAMVF7G4GJC4A7JMFJWLUAEIBFQD5RT3DCB5DC5TJDEKQBBACQ4JZVEE",
		Amount:        "20",
		Asset:         txnbuild.CreditAsset{Code: "USD", Issuer: "GAMVF7G4GJC4A7JMFJWLUAEIBFQD5RT3DCB5DC5TJDEKQBBACQ4JZVEE"},
		SourceAccount: &txnbuild.SimpleAccount{AccountID: "GAW77Z6GPWXSODJOMF5L5BMX6VMYGEJRKUNBC2CZ725JTQZORK74HQQD"},
	}

	tx := txnbuild.Transaction{
		SourceAccount: &txnbuild.SimpleAccount{AccountID: "GAW77Z6GPWXSODJOMF5L5BMX6VMYGEJRKUNBC2CZ725JTQZORK74HQQD", Sequence: int64(-1)},
		Operations:    []txnbuild.Operation{txnOp},
		Timebounds:    txnbuild.NewInfiniteTimeout(),
		Network:       rhconfig.NetworkPassphrase,
		Memo:          txnbuild.MemoHash(attachHash),
	}

	err = tx.Build()
	require.NoError(t, err)

	err = tx.Sign()
	require.NoError(t, err)

	txeB64, err := tx.Base64()
	require.NoError(t, err)

	authData := compliance.AuthData{
		Sender:         "alice*stellar.org",
		NeedInfo:       false,
		Tx:             txeB64,
		AttachmentJSON: string(attachmentJSON),
	}

	authDataJSON, err := authData.Marshal()
	require.NoError(t, err)

	params := url.Values{
		"data": {string(authDataJSON)},
		"sig":  {"ACamNqa0dF8gf97URhFVKWSD7fmvZKc5At+8dCLM5ySR0HsHySF3G2WuwYP2nKjeqjKmu3U9Z3+u1P10w1KBCA=="},
	}

	mockSignerVerifier.On(
		"Verify",
		"GBYJZW5XFAI6XV73H5SAIUYK6XZI4CGGVBUBO3ANA2SV7KKDAXTV6AEB",
		mock.AnythingOfType("[]uint8"),
		mock.AnythingOfType("[]uint8"),
	).Return(errors.New("Verify error")).Once()

	statusCode, response := mocks.GetResponse(testServer, params)
	responseString := strings.TrimSpace(string(response))
	assert.Equal(t, 400, statusCode)
	expected := test.StringToJSONMap(`{
  "code": "invalid_parameter",
  "message": "Invalid parameter.",
  "data": {
    "name": "sig"
  }
}`)
	assert.Equal(t, expected, test.StringToJSONMap(responseString))

}
