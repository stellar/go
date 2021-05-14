package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stellar/go/amount"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/base"
	"github.com/stellar/go/services/regulated-assets-approval-server/internal/db/dbtest"
	kycstatus "github.com/stellar/go/services/regulated-assets-approval-server/internal/serve/kyc-status"
	"github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTxApproveHandlerValidate(t *testing.T) {
	// empty asset issuer KP.
	h := txApproveHandler{}
	err := h.validate()
	require.EqualError(t, err, "issuer keypair cannot be nil")
	// empty asset code.
	issuerAccKeyPair := keypair.MustRandom()
	h = txApproveHandler{
		issuerKP: issuerAccKeyPair,
	}
	err = h.validate()
	require.EqualError(t, err, "asset code cannot be empty")
	// No Horizon client.
	h = txApproveHandler{
		issuerKP:  issuerAccKeyPair,
		assetCode: "FOOBAR",
	}
	err = h.validate()
	require.EqualError(t, err, "horizon client cannot be nil")
	// No network passphrase.
	horizonMock := horizonclient.MockClient{}
	h = txApproveHandler{
		issuerKP:      issuerAccKeyPair,
		assetCode:     "FOOBAR",
		horizonClient: &horizonMock,
	}
	err = h.validate()
	require.EqualError(t, err, "network passphrase cannot be empty")
	// No db.
	h = txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         "FOOBAR",
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
	}
	err = h.validate()
	require.EqualError(t, err, "db cannot be nil")
	// Empty kycThreshold.
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()
	h = txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         "FOOBAR",
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
		db:                conn,
	}
	err = h.validate()
	require.EqualError(t, err, "kyc threshold cannot be less than or equal to zero")
	// Negative kycThreshold.
	h = txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         "FOOBAR",
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
		db:                conn,
		kycThreshold:      -1,
	}
	err = h.validate()
	require.EqualError(t, err, "kyc threshold cannot be less than or equal to zero")
	// no baseURL.
	h = txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         "FOOBAR",
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
		db:                conn,
		kycThreshold:      1,
	}
	err = h.validate()
	require.EqualError(t, err, "base url cannot be empty")
	// Success.
	h = txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         "FOOBAR",
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
		db:                conn,
		kycThreshold:      1,
		baseURL:           "https://sep8-server.test",
	}
	err = h.validate()
	require.NoError(t, err)
}

func TestTxApproveHandlerValidateKYC(t *testing.T) {
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()
	issuerAccKeyPair := keypair.MustRandom()
	horizonMock := horizonclient.MockClient{}
	kycThresholdAmount, err := amount.ParseInt64("500")
	assetGOAT := txnbuild.CreditAsset{
		Code:   "GOAT",
		Issuer: issuerAccKeyPair.Address(),
	}
	h := txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         assetGOAT.GetCode(),
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
		db:                conn,
		kycThreshold:      kycThresholdAmount,
		baseURL:           "https://sep8-server.test",
	}
	err = h.validate()
	require.NoError(t, err)
	// Test no KYC needed flow.
	destinationKP := keypair.MustRandom()
	paymentOP := txnbuild.Payment{
		Destination: destinationKP.Address(),
		Amount:      "100",
		Asset:       assetGOAT,
	}
	actionRequiredMessage, err := h.validateKYC(&paymentOP)
	require.NoError(t, err)
	require.Empty(t, actionRequiredMessage)
	// Test failing malformed payment amount.
	paymentOP = txnbuild.Payment{
		Destination: destinationKP.Address(),
		Amount:      "ten",
		Asset:       assetGOAT,
	}
	_, err = h.validateKYC(&paymentOP)
	assert.Contains(t,
		err.Error(),
		`parsing account payment amount from string to Int64: invalid amount format: ten`,
	)
	// Test Successful KYC validation response.
	paymentOP = txnbuild.Payment{
		Destination: destinationKP.Address(),
		Amount:      "501",
		Asset:       assetGOAT,
	}
	actionRequiredMessage, err = h.validateKYC(&paymentOP)
	require.NoError(t, err)
	assert.Equal(t, `Payments exceeding 500.0000000 GOAT requires KYC approval. Please provide an email address.`, actionRequiredMessage)
}

func TestTxApproveHandlerHandleKYCRequiredOperationIfNeeded(t *testing.T) {
	ctx := context.Background()
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()
	issuerAccKeyPair := keypair.MustRandom()
	horizonMock := horizonclient.MockClient{}
	kycThresholdAmount, err := amount.ParseInt64("500")
	assetGOAT := txnbuild.CreditAsset{
		Code:   "GOAT",
		Issuer: issuerAccKeyPair.Address(),
	}
	h := txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         assetGOAT.GetCode(),
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
		db:                conn,
		kycThreshold:      kycThresholdAmount,
		baseURL:           "https://sep8-server.test",
	}
	err = h.validate()
	require.NoError(t, err)
	// Test successful action_required response.
	sourceKP := keypair.MustRandom()
	destinationKP := keypair.MustRandom()
	paymentOP := txnbuild.Payment{
		SourceAccount: sourceKP.Address(),
		Destination:   destinationKP.Address(),
		Amount:        "501",
		Asset:         assetGOAT,
	}
	actionRequiredTxApprovalResponse, err := h.handleKYCRequiredOperationIfNeeded(ctx, sourceKP.Address(), &paymentOP)
	require.NoError(t, err)
	wantTXApprovalResponse := txApprovalResponse{
		Status:       sep8Status("action_required"),
		Message:      `Payments exceeding ` + amount.StringFromInt64(h.kycThreshold) + ` GOAT requires KYC approval. Please provide an email address.`,
		StatusCode:   http.StatusOK,
		ActionURL:    actionRequiredTxApprovalResponse.ActionURL,
		ActionMethod: "POST",
		ActionFields: []string{"email_address"},
	}
	assert.Equal(t, &wantTXApprovalResponse, actionRequiredTxApprovalResponse)
	// Test if the kyc attempt was logged in db's accounts_kyc_status table.
	const q = `
	SELECT stellar_address
	FROM accounts_kyc_status
	WHERE stellar_address = $1
	`
	var stellarAddress string
	err = h.db.QueryRowContext(ctx, q, sourceKP.Address()).Scan(&stellarAddress)
	require.NoError(t, err)
	assert.Equal(t, sourceKP.Address(), stellarAddress)
}

func TestTxApproveHandlerTxApprove(t *testing.T) {
	ctx := context.Background()
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()
	issuerAccKeyPair := keypair.MustRandom()
	senderAccKP := keypair.MustRandom()
	receiverAccKP := keypair.MustRandom()
	assetGOAT := txnbuild.CreditAsset{
		Code:   "GOAT",
		Issuer: issuerAccKeyPair.Address(),
	}
	// Test if no transaction is submitted.
	req := txApproveRequest{
		Tx: "",
	}
	horizonMock := horizonclient.MockClient{}
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: issuerAccKeyPair.Address()}).
		Return(horizon.Account{
			AccountID: issuerAccKeyPair.Address(),
			Sequence:  "1",
			Balances: []horizon.Balance{
				{
					Asset:   base.Asset{Code: "ASSET", Issuer: issuerAccKeyPair.Address()},
					Balance: "0",
				},
			},
		}, nil)
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: senderAccKP.Address()}).
		Return(horizon.Account{
			AccountID: senderAccKP.Address(),
			Sequence:  "2",
		}, nil)
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: receiverAccKP.Address()}).
		Return(horizon.Account{
			AccountID: receiverAccKP.Address(),
			Sequence:  "3",
		}, nil)
	kycThresholdAmount, err := amount.ParseInt64("500")
	require.NoError(t, err)
	handler := txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         assetGOAT.GetCode(),
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
		db:                conn,
		kycThreshold:      kycThresholdAmount,
		baseURL:           "https://sep8-server.test",
	}
	rejectedResponse, err := handler.txApprove(ctx, req)
	require.NoError(t, err)
	wantRejectedResponse := txApprovalResponse{
		Status:     "rejected",
		Error:      `Missing parameter "tx".`,
		StatusCode: http.StatusBadRequest,
	}
	assert.Equal(t, &wantRejectedResponse, rejectedResponse)
	// Test if can't parse XDR.
	req = txApproveRequest{
		Tx: "BADXDRTRANSACTIONENVELOPE",
	}
	rejectedResponse, err = handler.txApprove(ctx, req)
	require.NoError(t, err)
	wantRejectedResponse = txApprovalResponse{
		Status:     "rejected",
		Error:      `Invalid parameter "tx".`,
		StatusCode: http.StatusBadRequest,
	}
	assert.Equal(t, &wantRejectedResponse, rejectedResponse)
	// Test if a non generic transaction fails, same result as malformed XDR.
	senderAcc, err := handler.horizonClient.AccountDetail(horizonclient.AccountRequest{AccountID: senderAccKP.Address()})
	require.NoError(t, err)
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &senderAcc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					Destination: receiverAccKP.Address(),
					Amount:      "1",
					Asset:       assetGOAT,
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	require.NoError(t, err)
	feeBumpTx, err := txnbuild.NewFeeBumpTransaction(
		txnbuild.FeeBumpTransactionParams{
			Inner:      tx,
			FeeAccount: receiverAccKP.Address(),
			BaseFee:    2 * txnbuild.MinBaseFee,
		},
	)
	require.NoError(t, err)
	feeBumpTxEnc, err := feeBumpTx.Base64()
	require.NoError(t, err)
	req = txApproveRequest{
		Tx: feeBumpTxEnc,
	}
	rejectedResponse, err = handler.txApprove(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, &wantRejectedResponse, rejectedResponse) // wantRejectedResponse is identical to "if can't parse XDR".
	// Test if the transaction sourceAccount the same as the server issuer account.
	issuerAcc, err := handler.horizonClient.AccountDetail(horizonclient.AccountRequest{AccountID: issuerAccKeyPair.Address()})
	require.NoError(t, err)
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &issuerAcc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					Destination: senderAccKP.Address(),
					Amount:      "1",
					Asset:       assetGOAT,
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	require.NoError(t, err)
	txEnc, err := tx.Base64()
	require.NoError(t, err)
	req = txApproveRequest{
		Tx: txEnc,
	}
	rejectedResponse, err = handler.txApprove(ctx, req)
	require.NoError(t, err)
	wantRejectedResponse = txApprovalResponse{
		Status:     "rejected",
		Error:      "The source account is invalid.",
		StatusCode: http.StatusBadRequest,
	}
	assert.Equal(t, &wantRejectedResponse, rejectedResponse)
	// Test if the transaction's operation sourceAccount the same as the server issuer account.
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &senderAcc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					SourceAccount: issuerAccKeyPair.Address(),
					Destination:   senderAccKP.Address(),
					Amount:        "1",
					Asset:         assetGOAT,
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	require.NoError(t, err)
	txEnc, err = tx.Base64()
	require.NoError(t, err)
	req = txApproveRequest{
		Tx: txEnc,
	}
	rejectedResponse, err = handler.txApprove(ctx, req)
	require.NoError(t, err)
	wantRejectedResponse = txApprovalResponse{
		Status:     "rejected",
		Error:      "There is one or more unauthorized operations in the provided transaction.",
		StatusCode: http.StatusBadRequest,
	}
	assert.Equal(t, &wantRejectedResponse, rejectedResponse)
	// Test if operation is not a payment (in this case allowing trust for receiverAccKP).
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &senderAcc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.AllowTrust{
					Trustor:   receiverAccKP.Address(),
					Type:      assetGOAT,
					Authorize: true,
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	require.NoError(t, err)
	txEnc, err = tx.Base64()
	req = txApproveRequest{
		Tx: txEnc,
	}
	rejectedResponse, err = handler.txApprove(ctx, req)
	require.NoError(t, err)
	wantRejectedResponse = txApprovalResponse{
		Status:     "rejected",
		Error:      "There is one or more unauthorized operations in the provided transaction.",
		StatusCode: http.StatusBadRequest,
	}
	assert.Equal(t, &wantRejectedResponse, rejectedResponse)
	// Test if multiple operations in transaction.
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &senderAcc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					SourceAccount: senderAccKP.Address(),
					Destination:   receiverAccKP.Address(),
					Amount:        "1",
					Asset:         assetGOAT,
				},
				&txnbuild.Payment{
					SourceAccount: senderAccKP.Address(),
					Destination:   receiverAccKP.Address(),
					Amount:        "2",
					Asset:         assetGOAT,
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	require.NoError(t, err)
	txEnc, err = tx.Base64()
	require.NoError(t, err)
	req = txApproveRequest{
		Tx: txEnc,
	}
	rejectedResponse, err = handler.txApprove(ctx, req)
	require.NoError(t, err)
	wantRejectedResponse = txApprovalResponse{
		Status:     "rejected",
		Error:      "Please submit a transaction with exactly one operation of type payment.",
		StatusCode: http.StatusBadRequest,
	}
	assert.Equal(t, &wantRejectedResponse, rejectedResponse)
	// Test if transaction source account seq num is equal to account sequence+1.
	// This tests the scenario where sequence numbers are too far in the future.
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount: &horizon.Account{
				AccountID: senderAccKP.Address(),
				Sequence:  "50",
			},
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					SourceAccount: senderAccKP.Address(),
					Destination:   receiverAccKP.Address(),
					Amount:        "1",
					Asset:         assetGOAT,
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	require.NoError(t, err)
	txEnc, err = tx.Base64()
	require.NoError(t, err)
	req = txApproveRequest{
		Tx: txEnc,
	}
	rejectedResponse, err = handler.txApprove(ctx, req)
	require.NoError(t, err)
	wantRejectedResponse = txApprovalResponse{
		Status:     "rejected",
		Error:      "Invalid transaction sequence number.",
		StatusCode: http.StatusBadRequest,
	}
	assert.Equal(t, &wantRejectedResponse, rejectedResponse)
}

func TestAPI_RejectedIntegration(t *testing.T) {
	ctx := context.Background()
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()
	issuerAccKeyPair := keypair.MustRandom()
	assetGOAT := txnbuild.CreditAsset{
		Code:   "GOAT",
		Issuer: issuerAccKeyPair.Address(),
	}
	horizonMock := horizonclient.MockClient{}
	senderAccKP := keypair.MustRandom()
	receiverAccKP := keypair.MustRandom()
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: issuerAccKeyPair.Address()}).
		Return(horizon.Account{
			AccountID: issuerAccKeyPair.Address(),
			Sequence:  "1",
			Balances: []horizon.Balance{
				{
					Asset:   base.Asset{Code: "ASSET", Issuer: issuerAccKeyPair.Address()},
					Balance: "1",
				},
			},
		}, nil)
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: senderAccKP.Address()}).
		Return(horizon.Account{
			AccountID: senderAccKP.Address(),
			Sequence:  "2",
		}, nil)
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: receiverAccKP.Address()}).
		Return(horizon.Account{
			AccountID: receiverAccKP.Address(),
			Sequence:  "3",
		}, nil)
	kycThresholdAmount, err := amount.ParseInt64("500")
	require.NoError(t, err)
	handler := txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         assetGOAT.GetCode(),
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
		db:                conn,
		kycThreshold:      kycThresholdAmount,
		baseURL:           "https://sep8-server.test",
	}
	// Test if no transaction is submitted.
	m := chi.NewMux()
	m.Post("/tx-approve", handler.ServeHTTP)
	req := `{
		"tx": ""
		}`
	r := httptest.NewRequest("POST", "/tx-approve", strings.NewReader(req))
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody := `{
		"status":"rejected", "error":"Missing parameter \"tx\"."
	}`
	require.JSONEq(t, wantBody, string(body))
	// Test if can't parse XDR.
	req = `{
		"tx": "BADXDRTRANSACTIONENVELOPE"
	}`
	r = httptest.NewRequest("POST", "/tx-approve", strings.NewReader(req))
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody = `{
		"status":"rejected", "error":"Invalid parameter \"tx\"."
	}`
	require.JSONEq(t, wantBody, string(body))
	// Test if a non generic transaction fails, same result as malformed XDR.
	senderAcc, err := handler.horizonClient.AccountDetail(horizonclient.AccountRequest{AccountID: senderAccKP.Address()})
	require.NoError(t, err)
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &senderAcc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					Destination: receiverAccKP.Address(),
					Amount:      "1",
					Asset:       assetGOAT,
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	require.NoError(t, err)
	feeBumpTx, err := txnbuild.NewFeeBumpTransaction(
		txnbuild.FeeBumpTransactionParams{
			Inner:      tx,
			FeeAccount: receiverAccKP.Address(),
			BaseFee:    2 * txnbuild.MinBaseFee,
		},
	)
	require.NoError(t, err)
	feeBumpTxEnc, err := feeBumpTx.Base64()
	require.NoError(t, err)
	req = `{
		"tx": "` + feeBumpTxEnc + `"
	}`
	r = httptest.NewRequest("POST", "/tx-approve", strings.NewReader(req))
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody = `{
		"status":"rejected", "error":"Invalid parameter \"tx\"."
	}`
	require.JSONEq(t, wantBody, string(body))
	// Test if the transaction sourceAccount the same as the server issuer account.
	issuerAcc, err := handler.horizonClient.AccountDetail(horizonclient.AccountRequest{AccountID: issuerAccKeyPair.Address()})
	require.NoError(t, err)
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &issuerAcc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					Destination: senderAccKP.Address(),
					Amount:      "1",
					Asset:       assetGOAT,
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	require.NoError(t, err)
	txEnc, err := tx.Base64()
	require.NoError(t, err)
	req = `{
		"tx": "` + txEnc + `"
	}`
	r = httptest.NewRequest("POST", "/tx-approve", strings.NewReader(req))
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody = `{
		"status":"rejected", "error":"The source account is invalid."
	}`
	require.JSONEq(t, wantBody, string(body))
	// Test if the transaction's operation sourceAccount the same as the server issuer account.
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &senderAcc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					SourceAccount: issuerAccKeyPair.Address(),
					Destination:   senderAccKP.Address(),
					Amount:        "1",
					Asset:         assetGOAT,
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	require.NoError(t, err)
	txEnc, err = tx.Base64()
	require.NoError(t, err)
	req = `{
		"tx": "` + txEnc + `"
	}`
	r = httptest.NewRequest("POST", "/tx-approve", strings.NewReader(req))
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody = `{
		"status":"rejected", "error":"There is one or more unauthorized operations in the provided transaction."
	}`
	require.JSONEq(t, wantBody, string(body))
	// Test if the transaction's operation is not a payment.
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &senderAcc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.AllowTrust{
					Trustor:   receiverAccKP.Address(),
					Type:      assetGOAT,
					Authorize: true,
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	require.NoError(t, err)
	txEnc, err = tx.Base64()
	require.NoError(t, err)
	req = `{
		"tx": "` + txEnc + `"
	}`
	r = httptest.NewRequest("POST", "/tx-approve", strings.NewReader(req))
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody = `{
		"status":"rejected", "error":"There is one or more unauthorized operations in the provided transaction."
	}`
	require.JSONEq(t, wantBody, string(body))
	// Test if more than one operation in transaction.
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &senderAcc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					SourceAccount: senderAccKP.Address(),
					Destination:   receiverAccKP.Address(),
					Amount:        "1",
					Asset:         assetGOAT,
				},
				&txnbuild.Payment{
					SourceAccount: senderAccKP.Address(),
					Destination:   receiverAccKP.Address(),
					Amount:        "2",
					Asset:         assetGOAT,
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	require.NoError(t, err)
	txEnc, err = tx.Base64()
	require.NoError(t, err)
	req = `{
		"tx": "` + txEnc + `"
	}`
	r = httptest.NewRequest("POST", "/tx-approve", strings.NewReader(req))
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody = `{
		"status":"rejected", "error":"Please submit a transaction with exactly one operation of type payment."
	}`
	require.JSONEq(t, wantBody, string(body))
	// Test when transaction source account seq num is not equal to account sequence+1.
	// This tests the scenario where sequence numbers are too far in the future.
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount: &horizon.Account{
				AccountID: senderAccKP.Address(),
				Sequence:  "50",
			},
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					SourceAccount: senderAccKP.Address(),
					Destination:   receiverAccKP.Address(),
					Amount:        "1",
					Asset:         assetGOAT,
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	require.NoError(t, err)
	txEnc, err = tx.Base64()
	require.NoError(t, err)
	req = `{
		"tx": "` + txEnc + `"
	}`
	r = httptest.NewRequest("POST", "/tx-approve", strings.NewReader(req))
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody = `{
		"status":"rejected", "error":"Invalid transaction sequence number."
	}`
	require.JSONEq(t, wantBody, string(body))
}

func TestAPI_RevisedIntegration(t *testing.T) {
	ctx := context.Background()
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()
	issuerAccKeyPair := keypair.MustRandom()
	senderAccKP := keypair.MustRandom()
	receiverAccKP := keypair.MustRandom()
	assetGOAT := txnbuild.CreditAsset{
		Code:   "GOAT",
		Issuer: issuerAccKeyPair.Address(),
	}
	horizonMock := horizonclient.MockClient{}
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: issuerAccKeyPair.Address()}).
		Return(horizon.Account{
			Balances: []horizon.Balance{
				{
					Asset:   base.Asset{Code: "ASSET", Issuer: issuerAccKeyPair.Address()},
					Balance: "0",
				},
			},
		}, nil)
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: senderAccKP.Address()}).
		Return(horizon.Account{
			AccountID: senderAccKP.Address(),
			Sequence:  "5",
		}, nil)
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: receiverAccKP.Address()}).
		Return(horizon.Account{
			AccountID: receiverAccKP.Address(),
			Sequence:  "0",
		}, nil)
	kycThresholdAmount, err := amount.ParseInt64("500")
	require.NoError(t, err)
	handler := txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         assetGOAT.GetCode(),
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
		db:                conn,
		kycThreshold:      kycThresholdAmount,
		baseURL:           "https://sep8-server.test",
	}
	// Test Successful request.
	senderAcc, err := handler.horizonClient.AccountDetail(horizonclient.AccountRequest{AccountID: senderAccKP.Address()})
	require.NoError(t, err)
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &senderAcc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					SourceAccount: senderAccKP.Address(),
					Destination:   receiverAccKP.Address(),
					Amount:        "1",
					Asset:         assetGOAT,
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	require.NoError(t, err)
	txEnc, err := tx.Base64()
	require.NoError(t, err)
	m := chi.NewMux()
	m.Post("/tx-approve", handler.ServeHTTP)
	req := `{
		"tx": "` + txEnc + `"
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
	var txApprovePOSTResponse txApprovalResponse
	err = json.Unmarshal(body, &txApprovePOSTResponse)
	require.NoError(t, err)
	wantTXApprovalResponse := txApprovalResponse{
		Status:  sep8Status("revised"),
		Tx:      txApprovePOSTResponse.Tx,
		Message: `Authorization and deauthorization operations were added.`,
	}
	assert.Equal(t, wantTXApprovalResponse, txApprovePOSTResponse)
	// Decode the request's transaction.
	parsed, err := txnbuild.TransactionFromXDR(txApprovePOSTResponse.Tx)
	require.NoError(t, err)
	tx, ok := parsed.Transaction()
	require.True(t, ok)
	// Check if revised transaction only has 5 operations.
	require.Len(t, tx.Operations(), 5)
	// Check Operation 1: AllowTrust op where issuer fully authorizes account A, asset X.
	op1, ok := tx.Operations()[0].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op1.Trustor, senderAccKP.Address())
	assert.Equal(t, op1.Type.GetCode(), assetGOAT.GetCode())
	require.True(t, op1.Authorize)
	// Check  Operation 2: AllowTrust op where issuer fully authorizes account B, asset X.
	op2, ok := tx.Operations()[1].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op2.Trustor, receiverAccKP.Address())
	assert.Equal(t, op2.Type.GetCode(), assetGOAT.GetCode())
	require.True(t, op2.Authorize)
	// Check Operation 3: Payment from A to B.
	op3, ok := tx.Operations()[2].(*txnbuild.Payment)
	require.True(t, ok)
	assert.Equal(t, op3.SourceAccount, senderAccKP.Address())
	assert.Equal(t, op3.Destination, receiverAccKP.Address())
	assert.Equal(t, op3.Asset, assetGOAT)
	// Check Operation 4: AllowTrust op where issuer fully deauthorizes account B, asset X.
	op4, ok := tx.Operations()[3].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op4.Trustor, receiverAccKP.Address())
	assert.Equal(t, op4.Type.GetCode(), assetGOAT.GetCode())
	require.False(t, op4.Authorize)
	// Check Operation 5: AllowTrust op where issuer fully deauthorizes account A, asset X.
	op5, ok := tx.Operations()[4].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op5.Trustor, senderAccKP.Address())
	assert.Equal(t, op5.Type.GetCode(), assetGOAT.GetCode())
	require.False(t, op5.Authorize)
}

func TestAPI_KYCIntegration(t *testing.T) {
	ctx := context.Background()
	db := dbtest.Open(t)
	defer db.Close()
	conn := db.Open()
	defer conn.Close()
	issuerAccKeyPair := keypair.MustRandom()
	senderAccKP := keypair.MustRandom()
	receiverAccKP := keypair.MustRandom()
	assetGOAT := txnbuild.CreditAsset{
		Code:   "GOAT",
		Issuer: issuerAccKeyPair.Address(),
	}
	horizonMock := horizonclient.MockClient{}
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: issuerAccKeyPair.Address()}).
		Return(horizon.Account{
			Balances: []horizon.Balance{
				{
					Asset:   base.Asset{Code: "ASSET", Issuer: issuerAccKeyPair.Address()},
					Balance: "0",
				},
			},
		}, nil)
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: senderAccKP.Address()}).
		Return(horizon.Account{
			AccountID: senderAccKP.Address(),
			Sequence:  "5",
		}, nil)
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: receiverAccKP.Address()}).
		Return(horizon.Account{
			AccountID: receiverAccKP.Address(),
			Sequence:  "0",
		}, nil)
	kycThresholdAmount, err := amount.ParseInt64("500")
	require.NoError(t, err)
	handler := txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         assetGOAT.GetCode(),
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
		db:                conn,
		kycThreshold:      kycThresholdAmount,
		baseURL:           "https://sep8-server.test",
	}
	// Submit tx with payment of 501 GOATs to POST /tx_approve.
	// Server's KYC threshold is <=500 GOATs.
	// Currently action required response has placeholder messages until kyc-status/ is implemented.
	senderAcc, err := handler.horizonClient.AccountDetail(horizonclient.AccountRequest{AccountID: senderAccKP.Address()})
	require.NoError(t, err)
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &senderAcc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					SourceAccount: senderAccKP.Address(),
					Destination:   receiverAccKP.Address(),
					Amount:        "501",
					Asset:         assetGOAT,
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	require.NoError(t, err)
	txEnc, err := tx.Base64()
	require.NoError(t, err)
	m := chi.NewMux()
	m.Post("/tx-approve", handler.ServeHTTP)
	req := `{
		"tx": "` + txEnc + `"
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
	var txApprovePOSTResponse txApprovalResponse
	err = json.Unmarshal(body, &txApprovePOSTResponse)
	require.NoError(t, err)
	wantTXApprovalResponse := txApprovalResponse{
		Status:       sep8Status("action_required"),
		Message:      `Payments exceeding ` + amount.StringFromInt64(handler.kycThreshold) + ` GOAT requires KYC approval. Please provide an email address.`,
		ActionURL:    txApprovePOSTResponse.ActionURL,
		ActionMethod: "POST",
		ActionFields: []string{"email_address"},
	}
	assert.Equal(t, wantTXApprovalResponse, txApprovePOSTResponse)
	// Setup /kyc-status route for subsequent integration steps
	m.Route("/kyc-status", func(mux chi.Router) {
		mux.Post("/{callback_id}", kycstatus.PostHandler{
			DB: conn,
		}.ServeHTTP)
	})
	// RxUUID is a regex used to validate correct UUIDs, https://w.wiki/39fK
	var RxUUID = regexp.MustCompile(
		`[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}`)
	// Grab callbackID
	callbackID := RxUUID.FindAllString(txApprovePOSTResponse.ActionURL, 1)[0]
	// Verify the KYC entree was inserted in db
	const q = `
		SELECT callback_id
		FROM accounts_kyc_status
		WHERE stellar_address = $1
	`
	var (
		returnedCallbackID string
	)
	err = handler.db.QueryRowContext(ctx, q, senderAccKP.Address()).Scan(&returnedCallbackID)
	require.NoError(t, err)
	assert.Equal(t, callbackID, returnedCallbackID)
	// Submit a request to action_url using the action_method and sending an email_address that doesn't start with "xx"
	req = `{
		"email_address": "TestEmail@email.com"
	}`
	r = httptest.NewRequest("POST", fmt.Sprintf("/kyc-status/%s", callbackID), strings.NewReader(req))
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody := `{
		"result": "no_further_action_required",
		"message": "Your KYC has been approved!"
	  }`
	require.JSONEq(t, wantBody, string(body))
	// Revise tx via a new tx-approve/ POST
	req = `{
		"tx": "` + txEnc + `"
	}`
	r = httptest.NewRequest("POST", "/tx-approve", strings.NewReader(req))
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	txApprovePOSTResponse = txApprovalResponse{}
	assert.Empty(t, txApprovePOSTResponse)
	err = json.Unmarshal(body, &txApprovePOSTResponse)
	require.NoError(t, err)
	wantTXApprovalResponse = txApprovalResponse{
		Status:  sep8Status("revised"),
		Tx:      txApprovePOSTResponse.Tx,
		Message: `Authorization and deauthorization operations were added.`,
	}
	assert.Equal(t, wantTXApprovalResponse, txApprovePOSTResponse)
	// Decode the request's transaction.
	parsed, err := txnbuild.TransactionFromXDR(txApprovePOSTResponse.Tx)
	require.NoError(t, err)
	tx, ok := parsed.Transaction()
	require.True(t, ok)
	// Check if revised transaction only has 5 operations.
	require.Len(t, tx.Operations(), 5)
	// Check Operation 1: AllowTrust op where issuer fully authorizes account A, asset X.
	op1, ok := tx.Operations()[0].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op1.Trustor, senderAccKP.Address())
	assert.Equal(t, op1.Type.GetCode(), assetGOAT.GetCode())
	require.True(t, op1.Authorize)
	// Check  Operation 2: AllowTrust op where issuer fully authorizes account B, asset X.
	op2, ok := tx.Operations()[1].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op2.Trustor, receiverAccKP.Address())
	assert.Equal(t, op2.Type.GetCode(), assetGOAT.GetCode())
	require.True(t, op2.Authorize)
	// Check Operation 3: Payment from A to B.
	op3, ok := tx.Operations()[2].(*txnbuild.Payment)
	require.True(t, ok)
	assert.Equal(t, op3.SourceAccount, senderAccKP.Address())
	assert.Equal(t, op3.Destination, receiverAccKP.Address())
	assert.Equal(t, op3.Asset, assetGOAT)
	// Check Operation 4: AllowTrust op where issuer fully deauthorizes account B, asset X.
	op4, ok := tx.Operations()[3].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op4.Trustor, receiverAccKP.Address())
	assert.Equal(t, op4.Type.GetCode(), assetGOAT.GetCode())
	require.False(t, op4.Authorize)
	// Check Operation 5: AllowTrust op where issuer fully deauthorizes account A, asset X.
	op5, ok := tx.Operations()[4].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op5.Trustor, senderAccKP.Address())
	assert.Equal(t, op5.Type.GetCode(), assetGOAT.GetCode())
	require.False(t, op5.Authorize)
	// Test rejected KYC response
	req = `{
		"email_address": "xxTestEmail@email.com"
	}`
	r = httptest.NewRequest("POST", fmt.Sprintf("/kyc-status/%s", callbackID), strings.NewReader(req))
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody = `{
		"result": "no_further_action_required",
		"message": "Your KYC has been rejected!"
	  }`
	require.JSONEq(t, wantBody, string(body))
	// Attempt to revise tx via a new tx-approve/ POST after rejection
	req = `{
		"tx": "` + txEnc + `"
	}`
	r = httptest.NewRequest("POST", "/tx-approve", strings.NewReader(req))
	r = r.WithContext(ctx)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	resp = w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody = `{
		"status":"rejected", "error":"Your KYC was rejected and you're not authorized for operations above 500.0000000 GOAT."
	}`
	require.JSONEq(t, wantBody, string(body))
}
