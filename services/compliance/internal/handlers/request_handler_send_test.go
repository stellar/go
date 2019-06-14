package handlers

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stellar/go/clients/stellartoml"
	"github.com/stellar/go/protocols/compliance"
	"github.com/stellar/go/protocols/federation"
	"github.com/stellar/go/support/http/httptest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/stellar/go/services/compliance/internal/config"
	"github.com/stellar/go/services/compliance/internal/db"
	"github.com/stellar/go/services/compliance/internal/mocks"
	"github.com/stellar/go/services/compliance/internal/test"
	"github.com/stellar/go/txnbuild"
)

func TestRequestHandlerSend(t *testing.T) {
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

	requestHandler := RequestHandler{
		Config:                  rhconfig,
		Client:                  mockHTTPClient,
		Database:                mockDatabase,
		FederationResolver:      mockFederationResolver,
		SignatureSignerVerifier: mockSignerVerifier,
		StellarTomlResolver:     mockStellartomlResolver,
	}

	testServer := httptest.NewServer(t, http.HandlerFunc(requestHandler.HandlerSend))
	defer testServer.Close()

	// When id parameter is missing, return error
	params := url.Values{
		// "id":           {"id"},
		"source":      {"bad"},
		"sender":      {"alice*stellar.org"}, // GAW77Z6GPWXSODJOMF5L5BMX6VMYGEJRKUNBC2CZ725JTQZORK74HQQD
		"destination": {"bob*stellar.org"},   // GAMVF7G4GJC4A7JMFJWLUAEIBFQD5RT3DCB5DC5TJDEKQBBACQ4JZVEE
		"amount":      {"20"},
	}

	statusCode, response := mocks.GetResponse(testServer, params)
	responseString := strings.TrimSpace(string(response))
	assert.Equal(t, 400, statusCode)
	expected := test.StringToJSONMap(`{
			  "code": "missing_parameter",
			  "message": "Required parameter is missing.",
			  "data": {
			    "name": "ID"
			  }
			}`)
	assert.Equal(t, expected, test.StringToJSONMap(responseString))

	// when source parameter is missing
	params = url.Values{
		"id": {"id"},
		// "source":      {"bad"},
		"sender":      {"alice*stellar.org"}, // GAW77Z6GPWXSODJOMF5L5BMX6VMYGEJRKUNBC2CZ725JTQZORK74HQQD
		"destination": {"bob*stellar.org"},   // GAMVF7G4GJC4A7JMFJWLUAEIBFQD5RT3DCB5DC5TJDEKQBBACQ4JZVEE
		"amount":      {"20"},
	}

	statusCode, response = mocks.GetResponse(testServer, params)
	responseString = strings.TrimSpace(string(response))
	assert.Equal(t, 400, statusCode)
	expected = test.StringToJSONMap(`{
			  "code": "missing_parameter",
			  "message": "Required parameter is missing.",
			  "data": {
			    "name": "Source"
			  }
			}`)
	assert.Equal(t, expected, test.StringToJSONMap(responseString))

	// When source param is invalid, return error
	params = url.Values{
		"id":           {"id"},
		"source":       {"bad"},
		"sender":       {"alice*stellar.org"}, // GAW77Z6GPWXSODJOMF5L5BMX6VMYGEJRKUNBC2CZ725JTQZORK74HQQD
		"destination":  {"bob*stellar.org"},   // GAMVF7G4GJC4A7JMFJWLUAEIBFQD5RT3DCB5DC5TJDEKQBBACQ4JZVEE
		"amount":       {"20"},
		"asset_code":   {"USD"},
		"asset_issuer": {"GAMVF7G4GJC4A7JMFJWLUAEIBFQD5RT3DCB5DC5TJDEKQBBACQ4JZVEE"},
		"extra_memo":   {"hello world"},
	}

	statusCode, response = mocks.GetResponse(testServer, params)
	responseString = strings.TrimSpace(string(response))
	assert.Equal(t, 400, statusCode)
	expected = test.StringToJSONMap(`{
			  "code": "invalid_parameter",
			  "message": "Invalid parameter.",
			  "data": {
			    "name": "Source"
			  }
			}`)
	assert.Equal(t, expected, test.StringToJSONMap(responseString, "more_info"))

	// When params are valid, it returns SendResponse when success (payment)
	params = url.Values{
		"id":           {"id"},
		"source":       {"GAW77Z6GPWXSODJOMF5L5BMX6VMYGEJRKUNBC2CZ725JTQZORK74HQQD"},
		"sender":       {"alice*stellar.org"}, // GAW77Z6GPWXSODJOMF5L5BMX6VMYGEJRKUNBC2CZ725JTQZORK74HQQD
		"destination":  {"bob*stellar.org"},   // GAMVF7G4GJC4A7JMFJWLUAEIBFQD5RT3DCB5DC5TJDEKQBBACQ4JZVEE
		"amount":       {"20"},
		"asset_code":   {"USD"},
		"asset_issuer": {"GAMVF7G4GJC4A7JMFJWLUAEIBFQD5RT3DCB5DC5TJDEKQBBACQ4JZVEE"},
		"extra_memo":   {"hello world"},
	}

	mockDatabase.Mock.On("GetAuthData", "id").Return(nil, nil).Once()
	mockDatabase.On("InsertAuthData", mock.AnythingOfType("*db.AuthData")).Run(func(args mock.Arguments) {
		entity, ok := args.Get(0).(*db.AuthData)
		assert.True(t, ok, "Invalid conversion")
		assert.Equal(t, "id", entity.RequestID)
		assert.Equal(t, "stellar.org", entity.Domain)
	}).Return(nil).Once()

	senderInfo := compliance.SenderInfo{FirstName: "John", LastName: "Doe"}
	senderInfoMap, err := senderInfo.Map()
	require.NoError(t, err)

	authServer := "https://acme.com/auth"

	mockFederationResolver.On(
		"LookupByAddress",
		"bob*stellar.org",
	).Return(&federation.NameResponse{
		AccountID: "GAMVF7G4GJC4A7JMFJWLUAEIBFQD5RT3DCB5DC5TJDEKQBBACQ4JZVEE",
		MemoType:  "text",
		Memo:      federation.Memo{"bob"},
	}, nil).Once()

	mockStellartomlResolver.On(
		"GetStellarToml",
		"stellar.org",
	).Return(&stellartoml.Response{AuthServer: authServer}, nil).Once()

	attachment := compliance.Attachment{
		Nonce: "nonce",
		Transaction: compliance.Transaction{
			Route:      "bob",
			Note:       "",
			SenderInfo: senderInfoMap,
			Extra:      "hello world",
		},
	}

	attachmentJSON, err := attachment.Marshal()
	require.NoError(t, err)
	attachHash, err := attachment.Hash()
	require.NoError(t, err)

	txnOp := &txnbuild.Payment{
		Destination: "GAMVF7G4GJC4A7JMFJWLUAEIBFQD5RT3DCB5DC5TJDEKQBBACQ4JZVEE",
		Amount:      "20",
		Asset:       txnbuild.CreditAsset{Code: "USD", Issuer: "GAMVF7G4GJC4A7JMFJWLUAEIBFQD5RT3DCB5DC5TJDEKQBBACQ4JZVEE"},
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

	txe, err := tx.Base64()
	require.NoError(t, err)

	authData := compliance.AuthData{
		Sender:         "alice*stellar.org",
		NeedInfo:       false,
		Tx:             txe,
		AttachmentJSON: string(attachmentJSON),
	}

	authDataJSON, err := authData.Marshal()
	require.NoError(t, err)

	authRequest := compliance.AuthRequest{
		DataJSON:  string(authDataJSON),
		Signature: "YeMlOYWNysyGBfsAe40z9dGgpRsKSQrqFIGAEsyJQ8osnXlLPynvJ2WQDGcBq2n5AA96YZdABhQz5ymqvxfQDw==",
	}

	authResponse := compliance.AuthResponse{
		InfoStatus: compliance.AuthStatusOk,
		TxStatus:   compliance.AuthStatusOk,
	}

	authResponseJSON, err := authResponse.Marshal()
	require.NoError(t, err)

	mockHTTPClient.On(
		"PostForm",
		rhconfig.Callbacks.FetchInfo,
		url.Values{"address": {"alice*stellar.org"}},
	).Return(
		mocks.BuildHTTPResponse(200, "{\"first_name\": \"John\", \"last_name\": \"Doe\"}"),
		nil,
	).Once()

	mockHTTPClient.On(
		"PostForm",
		authServer,
		authRequest.ToURLValues(),
	).Return(
		mocks.BuildHTTPResponse(200, string(authResponseJSON)),
		nil,
	).Once()

	mockSignerVerifier.On(
		"Sign",
		rhconfig.Keys.SigningSeed,
		[]byte(authRequest.DataJSON),
	).Return(authRequest.Signature, nil).Once()

	statusCode, response = mocks.GetResponse(testServer, params)
	responseString = strings.TrimSpace(string(response))
	assert.Equal(t, 200, statusCode)
	expected = test.StringToJSONMap(`{
				  "auth_response": {
				    "info_status": "ok",
				    "tx_status": "ok"
				  },
				  "transaction_xdr": "` + txe + `"
				}`)
	assert.Equal(t, expected, test.StringToJSONMap(responseString))

}
