package serve

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stellar/go/amount"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/regulated-assets-approval-server/internal/db/dbtest"
	kycstatus "github.com/stellar/go/services/regulated-assets-approval-server/internal/serve/kyc-status"
	"github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI_txApprove_rejected(t *testing.T) {
	ctx := context.Background()
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()

	issuerKP := keypair.MustRandom()
	horizonMock := horizonclient.MockClient{}
	kycThresholdAmount, err := amount.ParseInt64("500")
	require.NoError(t, err)

	handler := txApproveHandler{
		issuerKP:          issuerKP,
		assetCode:         "FOO",
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
		db:                conn,
		kycThreshold:      kycThresholdAmount,
		baseURL:           "https://example.com",
	}

	// rejected if no transaction "tx"is submitted
	m := chi.NewMux()
	m.Post("/tx-approve", handler.ServeHTTP)
	r := httptest.NewRequest("POST", "/tx-approve", nil)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{
		"status": "rejected",
		"error": "Missing parameter \"tx\"."
	}`
	require.JSONEq(t, wantBody, string(body))
}

func TestAPI_txApprove_revised(t *testing.T) {
	ctx := context.Background()
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()

	// Perpare accounts on mock horizon.
	senderKP := keypair.MustRandom()
	receiverKP := keypair.MustRandom()
	issuerKP := keypair.MustRandom()
	assetGOAT := txnbuild.CreditAsset{
		Code:   "GOAT",
		Issuer: issuerKP.Address(),
	}
	horizonMock := horizonclient.MockClient{}
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: senderKP.Address()}).
		Return(horizon.Account{
			AccountID: senderKP.Address(),
			Sequence:  "5",
		}, nil)

	kycThresholdAmount, err := amount.ParseInt64("500")
	require.NoError(t, err)
	handler := txApproveHandler{
		issuerKP:          issuerKP,
		assetCode:         assetGOAT.GetCode(),
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
		db:                conn,
		kycThreshold:      kycThresholdAmount,
		baseURL:           "https://example.com",
	}

	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount: &horizon.Account{
				AccountID: senderKP.Address(),
				Sequence:  "5",
			},
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					SourceAccount: senderKP.Address(),
					Destination:   receiverKP.Address(),
					Amount:        "1",
					Asset:         assetGOAT,
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	require.NoError(t, err)
	txe, err := tx.Base64()
	require.NoError(t, err)

	m := chi.NewMux()
	m.Post("/tx-approve", handler.ServeHTTP)
	req := `{
		"tx": "` + txe + `"
	}`
	r := httptest.NewRequest("POST", "/tx-approve", strings.NewReader(req))
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var gotResponse txApprovalResponse
	err = json.Unmarshal(body, &gotResponse)
	require.NoError(t, err)
	require.Equal(t, sep8StatusRevised, gotResponse.Status)
	require.Equal(t, "Authorization and deauthorization operations were added.", gotResponse.Message)

	gotGenericTx, err := txnbuild.TransactionFromXDR(gotResponse.Tx)
	require.NoError(t, err)
	gotTx, ok := gotGenericTx.Transaction()
	require.True(t, ok)

	require.Len(t, gotTx.Operations(), 5)
	// AllowTrust op where issuer fully authorizes account A, asset X
	op0, ok := gotTx.Operations()[0].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op0.Trustor, senderKP.Address())
	assert.Equal(t, op0.Type.GetCode(), assetGOAT.GetCode())
	require.True(t, op0.Authorize)
	// AllowTrust op where issuer fully authorizes account B, asset X
	op1, ok := gotTx.Operations()[1].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op1.Trustor, receiverKP.Address())
	assert.Equal(t, op1.Type.GetCode(), assetGOAT.GetCode())
	require.True(t, op1.Authorize)
	// Payment from A to B
	op2, ok := gotTx.Operations()[2].(*txnbuild.Payment)
	require.True(t, ok)
	assert.Equal(t, op2.Destination, receiverKP.Address())
	assert.Equal(t, op2.Asset, assetGOAT)
	// AllowTrust op where issuer fully deauthorizes account B, asset X
	op3, ok := gotTx.Operations()[3].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op3.Trustor, receiverKP.Address())
	assert.Equal(t, op3.Type.GetCode(), assetGOAT.GetCode())
	require.False(t, op3.Authorize)
	// AllowTrust op where issuer fully deauthorizes account A, asset X
	op4, ok := gotTx.Operations()[4].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op4.Trustor, senderKP.Address())
	assert.Equal(t, op4.Type.GetCode(), assetGOAT.GetCode())
	require.False(t, op4.Authorize)
}

func TestAPI_txAprove_actionRequired(t *testing.T) {
	ctx := context.Background()
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()

	// prepare handler dependencies
	senderKP := keypair.MustRandom()
	receiverKP := keypair.MustRandom()
	issuerKP := keypair.MustRandom()
	assetGOAT := txnbuild.CreditAsset{
		Code:   "GOAT",
		Issuer: issuerKP.Address(),
	}
	kycThresholdAmount, err := amount.ParseInt64("500")
	require.NoError(t, err)
	horizonMock := horizonclient.MockClient{}
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: senderKP.Address()}).
		Return(horizon.Account{
			AccountID: senderKP.Address(),
			Sequence:  "1",
		}, nil)

	handler := txApproveHandler{
		issuerKP:          issuerKP,
		assetCode:         assetGOAT.GetCode(),
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
		db:                conn,
		kycThreshold:      kycThresholdAmount,
		baseURL:           "https://example.com",
	}

	// setup route handlers
	m := chi.NewMux()
	m.Post("/tx-approve", handler.ServeHTTP)
	m.Post("/kyc-status/{callback_id}", kycstatus.PostHandler{DB: conn}.ServeHTTP)

	// Step 1: Client sends payment with 500+ GOAT
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount: &horizon.Account{
				AccountID: senderKP.Address(),
				Sequence:  "1",
			},
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					SourceAccount: senderKP.Address(),
					Destination:   receiverKP.Address(),
					Amount:        "501",
					Asset:         assetGOAT,
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	require.NoError(t, err)
	txe, err := tx.Base64()
	require.NoError(t, err)

	r := httptest.NewRequest("POST", "/tx-approve", strings.NewReader(`{"tx": "`+txe+`"}`))
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var callbackID string
	q := `SELECT callback_id FROM accounts_kyc_status WHERE stellar_address = $1`
	err = conn.QueryRowContext(ctx, q, senderKP.Address()).Scan(&callbackID)
	require.NoError(t, err)

	var txApprovePOSTResponse txApprovalResponse
	err = json.Unmarshal(body, &txApprovePOSTResponse)
	require.NoError(t, err)
	wantTXApprovalResponse := txApprovalResponse{
		Status:       sep8Status("action_required"),
		Message:      "Payments exceeding 500.00 GOAT require KYC approval. Please provide an email address.",
		ActionURL:    "https://example.com/kyc-status/" + callbackID,
		ActionMethod: "POST",
		ActionFields: []string{"email_address"},
	}
	assert.Equal(t, wantTXApprovalResponse, txApprovePOSTResponse)
}

func TestAPI_txAprove_actionRequiredFlow(t *testing.T) {
	ctx := context.Background()
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()

	// prepare handler dependencies
	senderKP := keypair.MustRandom()
	receiverKP := keypair.MustRandom()
	issuerKP := keypair.MustRandom()
	assetGOAT := txnbuild.CreditAsset{
		Code:   "GOAT",
		Issuer: issuerKP.Address(),
	}
	kycThresholdAmount, err := amount.ParseInt64("500")
	require.NoError(t, err)
	horizonMock := horizonclient.MockClient{}
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: senderKP.Address()}).
		Return(horizon.Account{
			AccountID: senderKP.Address(),
			Sequence:  "1",
		}, nil)

	handler := txApproveHandler{
		issuerKP:          issuerKP,
		assetCode:         assetGOAT.GetCode(),
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
		db:                conn,
		kycThreshold:      kycThresholdAmount,
		baseURL:           "https://example.com",
	}

	// setup route handlers
	m := chi.NewMux()
	m.Post("/tx-approve", handler.ServeHTTP)
	m.Post("/kyc-status/{callback_id}", kycstatus.PostHandler{DB: conn}.ServeHTTP)

	// Step 1: client sends payment with 500+ GOAT
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount: &horizon.Account{
				AccountID: senderKP.Address(),
				Sequence:  "1",
			},
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					SourceAccount: senderKP.Address(),
					Destination:   receiverKP.Address(),
					Amount:        "501",
					Asset:         assetGOAT,
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	require.NoError(t, err)
	txe, err := tx.Base64()
	require.NoError(t, err)

	r := httptest.NewRequest("POST", "/tx-approve", strings.NewReader(`{"tx": "`+txe+`"}`))
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var callbackID string
	q := `SELECT callback_id FROM accounts_kyc_status WHERE stellar_address = $1`
	err = conn.QueryRowContext(ctx, q, senderKP.Address()).Scan(&callbackID)
	require.NoError(t, err)

	var gotResponse txApprovalResponse
	err = json.Unmarshal(body, &gotResponse)
	require.NoError(t, err)
	wantTXApprovalResponse := txApprovalResponse{
		Status:       sep8Status("action_required"),
		Message:      "Payments exceeding 500.00 GOAT require KYC approval. Please provide an email address.",
		ActionURL:    "https://example.com/kyc-status/" + callbackID,
		ActionMethod: "POST",
		ActionFields: []string{"email_address"},
	}
	assert.Equal(t, wantTXApprovalResponse, gotResponse)

	// Step 2: client follows up with action required. KYC should get approved for emails not starting with "x"
	actionMethod := gotResponse.ActionMethod
	actionURL := gotResponse.ActionURL
	actionFields := strings.NewReader(`{"email_address": "test@email.com"}`)
	r = httptest.NewRequest(actionMethod, actionURL, actionFields)
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody := `{"result": "no_further_action_required"}`
	require.JSONEq(t, wantBody, string(body))

	// Step 3: verify transactions with 500+ GOAT can now be revised
	r = httptest.NewRequest("POST", "/tx-approve", strings.NewReader(`{"tx": "`+txe+`"}`))
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	gotResponse = txApprovalResponse{}
	err = json.Unmarshal(body, &gotResponse)
	require.NoError(t, err)
	assert.Equal(t, sep8StatusRevised, gotResponse.Status)
	assert.Equal(t, "Authorization and deauthorization operations were added.", gotResponse.Message)
	require.NotEmpty(t, gotResponse.Tx)

	// Step 4: client follows up with action required. KYC should get rejected for emails starting with "x"
	actionFields = strings.NewReader(`{"email_address": "xtest@email.com"}`)
	r = httptest.NewRequest(actionMethod, actionURL, actionFields)
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody = `{"result": "no_further_action_required"}`
	require.JSONEq(t, wantBody, string(body))

	// Step 5: verify transactions with 500+ GOAT are rejected
	r = httptest.NewRequest("POST", "/tx-approve", strings.NewReader(`{"tx": "`+txe+`"}`))
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody = `{
		"status": "rejected",
		"error": "Your KYC was rejected and you're not authorized for operations above 500.00 GOAT."
	}`
	require.JSONEq(t, wantBody, string(body))
}
