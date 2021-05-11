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
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/base"
	"github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTxApproveHandlerValidate(t *testing.T) {
	// empty asset issuer KP
	h := txApproveHandler{}
	err := h.validate()
	require.EqualError(t, err, "issuer keypair cannot be nil")

	// empty asset code
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

	// Success
	h = txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         "FOOBAR",
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
	}
	err = h.validate()
	require.NoError(t, err)
}

func TestTxApproveHandlerTxApprove(t *testing.T) {
	ctx := context.Background()
	issuerAccKeyPair := keypair.MustRandom()
	kp01 := keypair.MustRandom()
	kp02 := keypair.MustRandom()
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
		On("AccountDetail", horizonclient.AccountRequest{AccountID: kp01.Address()}).
		Return(horizon.Account{
			AccountID: kp01.Address(),
			Sequence:  "2",
		}, nil)
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: kp02.Address()}).
		Return(horizon.Account{
			AccountID: kp02.Address(),
			Sequence:  "3",
		}, nil)
	handler := txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         assetGOAT.GetCode(),
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
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
	acc, err := handler.horizonClient.AccountDetail(horizonclient.AccountRequest{AccountID: kp01.Address()})
	require.NoError(t, err)
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &acc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					Destination: kp02.Address(),
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
			FeeAccount: kp02.Address(),
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

	// Test if the transaction sourceAccount the same as the server issuer account
	issuerAcc, err := handler.horizonClient.AccountDetail(horizonclient.AccountRequest{AccountID: issuerAccKeyPair.Address()})
	require.NoError(t, err)
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &issuerAcc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					Destination: kp01.Address(),
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

	// Test if the transaction's operation sourceAccount the same as the server issuer account
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &acc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					SourceAccount: issuerAccKeyPair.Address(),
					Destination:   kp01.Address(),
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

	// Test if operation is not a payment (in this case allowing trust for a random account)
	kp03 := keypair.MustRandom()
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: kp03.Address()}).
		Return(horizon.Account{
			AccountID: kp02.Address(),
			Sequence:  "5",
		}, nil)
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &acc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.AllowTrust{
					Trustor:   kp03.Address(),
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

	// Test if multiple operations in transaction
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &acc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					SourceAccount: kp01.Address(),
					Destination:   kp02.Address(),
					Amount:        "1",
					Asset:         assetGOAT,
				},
				&txnbuild.Payment{
					SourceAccount: kp01.Address(),
					Destination:   kp02.Address(),
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
}

func TestAPI_RejectedIntegration(t *testing.T) {
	ctx := context.Background()
	issuerAccKeyPair := keypair.MustRandom()
	assetGOAT := txnbuild.CreditAsset{
		Code:   "GOAT",
		Issuer: issuerAccKeyPair.Address(),
	}
	horizonMock := horizonclient.MockClient{}
	kp01 := keypair.MustRandom()
	kp02 := keypair.MustRandom()
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
		On("AccountDetail", horizonclient.AccountRequest{AccountID: kp01.Address()}).
		Return(horizon.Account{
			AccountID: kp01.Address(),
			Sequence:  "2",
		}, nil)
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: kp02.Address()}).
		Return(horizon.Account{
			AccountID: kp02.Address(),
			Sequence:  "3",
		}, nil)
	handler := txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         assetGOAT.GetCode(),
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
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
	acc, err := handler.horizonClient.AccountDetail(horizonclient.AccountRequest{AccountID: kp01.Address()})
	require.NoError(t, err)
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &acc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					Destination: kp02.Address(),
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
			FeeAccount: kp02.Address(),
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

	// Test if the transaction sourceAccount the same as the server issuer account
	issuerAcc, err := handler.horizonClient.AccountDetail(horizonclient.AccountRequest{AccountID: issuerAccKeyPair.Address()})
	require.NoError(t, err)
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &issuerAcc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					Destination: kp01.Address(),
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

	// Test if the transaction's operation sourceAccount the same as the server issuer account
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &acc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					SourceAccount: issuerAccKeyPair.Address(),
					Destination:   kp01.Address(),
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
	kp03 := keypair.MustRandom()
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: kp03.Address()}).
		Return(horizon.Account{
			AccountID: kp02.Address(),
			Sequence:  "6",
		}, nil)
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &acc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.AllowTrust{
					Trustor:   kp03.Address(),
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

	// Test if more than one operation in transaction
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &acc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					SourceAccount: kp01.Address(),
					Destination:   kp02.Address(),
					Amount:        "1",
					Asset:         assetGOAT,
				},
				&txnbuild.Payment{
					SourceAccount: kp01.Address(),
					Destination:   kp03.Address(),
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
}

func TestAPI_RevisedIntegration(t *testing.T) {
	ctx := context.Background()
	issuerAccKeyPair := keypair.MustRandom()
	kp01 := keypair.MustRandom()
	kp02 := keypair.MustRandom()
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
		On("AccountDetail", horizonclient.AccountRequest{AccountID: kp01.Address()}).
		Return(horizon.Account{
			AccountID: kp01.Address(),
			Sequence:  "5",
		}, nil)
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: kp02.Address()}).
		Return(horizon.Account{
			AccountID: kp02.Address(),
			Sequence:  "0",
		}, nil)
	handler := txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         assetGOAT.GetCode(),
		horizonClient:     &horizonMock,
		networkPassphrase: network.TestNetworkPassphrase,
	}

	// Test Successful request, transaction source account set == the payment source account
	acc, err := handler.horizonClient.AccountDetail(horizonclient.AccountRequest{AccountID: kp01.Address()})
	require.NoError(t, err)
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &acc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					SourceAccount: kp01.Address(),
					Destination:   kp02.Address(),
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
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	var response map[string]string
	err = json.Unmarshal(body, &response)

	_, exists := response["status"]
	assert.True(t, exists)
	assert.Equal(t, response["status"], "revised")
	_, exists = response["message"]
	assert.True(t, exists)
	assert.Equal(t, response["message"], "Authorization and deauthorization operations were added.")
	_, exists = response["tx"]
	assert.True(t, exists)

	// Decode the request's transaction.
	parsed, err := txnbuild.TransactionFromXDR(response["tx"])
	require.NoError(t, err)
	tx, ok := parsed.Transaction()
	require.True(t, ok)

	// Check if revised transaction only has 5 operations.
	require.Len(t, tx.Operations(), 5)

	// Check Operation 1: AllowTrust op where issuer fully authorizes account A, asset X.
	op1, ok := tx.Operations()[0].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op1.Trustor, kp01.Address())
	assert.Equal(t, op1.Type.GetCode(), assetGOAT.GetCode())
	require.True(t, op1.Authorize)

	// Check  Operation 2: AllowTrust op where issuer fully authorizes account B, asset X.
	op2, ok := tx.Operations()[1].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op2.Trustor, kp02.Address())
	assert.Equal(t, op2.Type.GetCode(), assetGOAT.GetCode())
	require.True(t, op2.Authorize)

	// Check Operation 3: Payment from A to B.
	op3, ok := tx.Operations()[2].(*txnbuild.Payment)
	require.True(t, ok)
	assert.Equal(t, op3.SourceAccount, kp01.Address())
	assert.Equal(t, op3.Destination, kp02.Address())
	assert.Equal(t, op3.Asset, assetGOAT)

	// Check Operation 4: AllowTrust op where issuer fully deauthorizes account B, asset X.
	op4, ok := tx.Operations()[3].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op4.Trustor, kp02.Address())
	assert.Equal(t, op4.Type.GetCode(), assetGOAT.GetCode())
	require.False(t, op4.Authorize)

	// Check Operation 5: AllowTrust op where issuer fully deauthorizes account A, asset X.
	op5, ok := tx.Operations()[4].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op5.Trustor, kp01.Address())
	assert.Equal(t, op5.Type.GetCode(), assetGOAT.GetCode())
	require.False(t, op5.Authorize)

	// Test Successful request, transaction source account set and the no payment source account
	acc, err = handler.horizonClient.AccountDetail(horizonclient.AccountRequest{AccountID: kp01.Address()})
	require.NoError(t, err)
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &acc,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					Destination: kp02.Address(),
					Amount:      "2",
					Asset:       assetGOAT,
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

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	err = json.Unmarshal(body, &response)

	_, exists = response["status"]
	assert.True(t, exists)
	assert.Equal(t, response["status"], "revised")
	_, exists = response["message"]
	assert.True(t, exists)
	assert.Equal(t, response["message"], "Authorization and deauthorization operations were added.")
	_, exists = response["tx"]
	assert.True(t, exists)

	// Decode the request's transaction.
	parsed, err = txnbuild.TransactionFromXDR(response["tx"])
	require.NoError(t, err)
	tx, ok = parsed.Transaction()
	require.True(t, ok)

	// Check if revised transaction only has 5 operations.
	require.Len(t, tx.Operations(), 5)

	// Check Operation 1: AllowTrust op where issuer fully authorizes account A, asset X.
	op1, ok = tx.Operations()[0].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op1.Trustor, kp01.Address())
	assert.Equal(t, op1.Type.GetCode(), assetGOAT.GetCode())
	require.True(t, op1.Authorize)

	// Check Operation 2: AllowTrust op where issuer fully authorizes account B, asset X.
	op2, ok = tx.Operations()[1].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op2.Trustor, kp02.Address())
	assert.Equal(t, op2.Type.GetCode(), assetGOAT.GetCode())
	require.True(t, op2.Authorize)

	// Check Operation 3: Payment to B.
	op3, ok = tx.Operations()[2].(*txnbuild.Payment)
	require.True(t, ok)
	assert.Equal(t, op3.SourceAccount, "")
	assert.Equal(t, op3.Destination, kp02.Address())
	assert.Equal(t, op3.Asset, assetGOAT)

	// Check Operation 4: AllowTrust op where issuer fully deauthorizes account B, asset X.
	op4, ok = tx.Operations()[3].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op4.Trustor, kp02.Address())
	assert.Equal(t, op4.Type.GetCode(), assetGOAT.GetCode())
	require.False(t, op4.Authorize)

	// Check Operation 5: AllowTrust op where issuer fully deauthorizes account A, asset X.
	op5, ok = tx.Operations()[4].(*txnbuild.AllowTrust)
	require.True(t, ok)
	assert.Equal(t, op5.Trustor, kp01.Address())
	assert.Equal(t, op5.Type.GetCode(), assetGOAT.GetCode())
	require.False(t, op5.Authorize)
}
