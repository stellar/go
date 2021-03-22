package serve

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTxApproveHandler_isRejected(t *testing.T) {
	ctx := context.Background()
	issuerAccKeyPair := keypair.MustRandom()
	assetGOAT := txnbuild.CreditAsset{
		Code:   "GOAT",
		Issuer: issuerAccKeyPair.Address(),
	}
	// Test if no transaction is submitted.
	req := txApproveRequest{
		Transaction: "",
	}
	rejectedResponse, err := txApproveHandler{
		issuerAccountSecret: issuerAccKeyPair.Seed(),
		assetCode:           assetGOAT.GetCode(),
	}.isRejected(ctx, req)
	require.NoError(t, err)
	wantRejectedResponse := txApproveResponse{
		Status:  rejectedStatus,
		Message: missingParamMsg,
	}
	assert.Equal(t, &wantRejectedResponse, rejectedResponse)

	// Test if can't parse XDR.
	req = txApproveRequest{
		Transaction: "BADXDRTRANSACTIONENVELOPE",
	}
	rejectedResponse, err = txApproveHandler{
		issuerAccountSecret: issuerAccKeyPair.Seed(),
		assetCode:           assetGOAT.GetCode(),
	}.isRejected(ctx, req)
	require.NoError(t, err)
	wantRejectedResponse = txApproveResponse{
		Status:  rejectedStatus,
		Message: invalidParamMsg,
	}
	assert.Equal(t, &wantRejectedResponse, rejectedResponse)

	// Test if a non generic transaction fails, same result as malformed XDR.
	kp01 := keypair.MustRandom()
	kp02 := keypair.MustRandom()
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: kp01.Address()},
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
		Transaction: feeBumpTxEnc,
	}
	rejectedResponse, err = txApproveHandler{
		issuerAccountSecret: issuerAccKeyPair.Seed(),
		assetCode:           assetGOAT.GetCode(),
	}.isRejected(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, &wantRejectedResponse, rejectedResponse) // wantRejectedResponse is identical to "if can't parse XDR".

	// Test if the transaction sourceAccount the same as the server issuer account
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: issuerAccKeyPair.Address()},
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
	req = txApproveRequest{
		Transaction: txEnc,
	}
	rejectedResponse, err = txApproveHandler{
		issuerAccountSecret: issuerAccKeyPair.Seed(),
		assetCode:           assetGOAT.GetCode(),
	}.isRejected(ctx, req)
	require.NoError(t, err)
	wantRejectedResponse = txApproveResponse{
		Status:  rejectedStatus,
		Message: invalidSrcAccMsg,
	}
	assert.Equal(t, &wantRejectedResponse, rejectedResponse)

	// Test if the transaction's operation sourceaccount the same as the server issuer account
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: kp01.Address()},
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
	req = txApproveRequest{
		Transaction: txEnc,
	}
	rejectedResponse, err = txApproveHandler{
		issuerAccountSecret: issuerAccKeyPair.Seed(),
		assetCode:           assetGOAT.GetCode(),
	}.isRejected(ctx, req)
	require.NoError(t, err)
	wantRejectedResponse = txApproveResponse{
		Status:  rejectedStatus,
		Message: unauthorizedOpMsg,
	}
	assert.Equal(t, &wantRejectedResponse, rejectedResponse)

	// Test "not implemented"
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: kp01.Address()},
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
	txEnc, err = tx.Base64()
	req = txApproveRequest{
		Transaction: txEnc,
	}
	rejectedResponse, err = txApproveHandler{
		issuerAccountSecret: issuerAccKeyPair.Seed(),
		assetCode:           assetGOAT.GetCode(),
	}.isRejected(ctx, req)
	require.NoError(t, err)
	wantRejectedResponse = txApproveResponse{
		Status:  rejectedStatus,
		Message: notImplementedMsg,
	}
	assert.Equal(t, &wantRejectedResponse, rejectedResponse)
}

func TestTxApproveHandler_serveHTTPJson(t *testing.T) {
	ctx := context.Background()
	issuerAccKeyPair := keypair.MustRandom()
	assetGOAT := txnbuild.CreditAsset{
		Code:   "GOAT",
		Issuer: issuerAccKeyPair.Address(),
	}
	handler := txApproveHandler{
		issuerAccountSecret: issuerAccKeyPair.Seed(),
		assetCode:           assetGOAT.GetCode(),
	}

	// Test if no transaction is submitted.
	req := `{
		"tx": ""
	}`
	r := httptest.NewRequest("POST", "/tx_approve", strings.NewReader(req))
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/tx_approve", handler.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody := `{
		"status":"rejected", "message":"Missing parameter \"tx\"."
	}`
	require.JSONEq(t, wantBody, string(body))

	// Test if can't parse XDR.
	req = `{
		"tx": "BADXDRTRANSACTIONENVELOPE"
	}`
	r = httptest.NewRequest("POST", "/tx_approve", strings.NewReader(req))
	r = r.WithContext(ctx)

	w = httptest.NewRecorder()
	m = chi.NewMux()
	m.Post("/tx_approve", handler.ServeHTTP)
	m.ServeHTTP(w, r)
	resp = w.Result()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody = `{
		"status":"rejected", "message":"Invalid parameter \"tx\"."
	}`
	require.JSONEq(t, wantBody, string(body))

	// Test if a non generic transaction fails, same result as malformed XDR.
	kp01 := keypair.MustRandom()
	kp02 := keypair.MustRandom()
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: kp01.Address()},
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
	r = httptest.NewRequest("POST", "/tx_approve", strings.NewReader(req))
	r = r.WithContext(ctx)

	w = httptest.NewRecorder()
	m = chi.NewMux()
	m.Post("/tx_approve", handler.ServeHTTP)
	m.ServeHTTP(w, r)
	resp = w.Result()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody = `{
		"status":"rejected", "message":"Invalid parameter \"tx\"."
	}`
	require.JSONEq(t, wantBody, string(body))

	// Test if the transaction sourceAccount the same as the server issuer account
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: issuerAccKeyPair.Address()},
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
	req = `{
		"tx": "` + txEnc + `"
	}`
	r = httptest.NewRequest("POST", "/tx_approve", strings.NewReader(req))
	r = r.WithContext(ctx)

	w = httptest.NewRecorder()
	m = chi.NewMux()
	m.Post("/tx_approve", handler.ServeHTTP)
	m.ServeHTTP(w, r)
	resp = w.Result()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody = `{
		"status":"rejected", "message":"The source account is invalid."
	}`
	require.JSONEq(t, wantBody, string(body))

	// Test if the transaction's operation sourceaccount the same as the server issuer account
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: kp01.Address()},
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
	req = `{
		"tx": "` + txEnc + `"
	}`
	r = httptest.NewRequest("POST", "/tx_approve", strings.NewReader(req))
	r = r.WithContext(ctx)

	w = httptest.NewRecorder()
	m = chi.NewMux()
	m.Post("/tx_approve", handler.ServeHTTP)
	m.ServeHTTP(w, r)
	resp = w.Result()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody = `{
		"status":"rejected", "message":"There is one or more unauthorized operations in the provided transaction."
	}`
	require.JSONEq(t, wantBody, string(body))

	// Test "not implemented"
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: kp01.Address()},
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
	txEnc, err = tx.Base64()
	req = `{
		"tx": "` + txEnc + `"
	}`
	r = httptest.NewRequest("POST", "/tx_approve", strings.NewReader(req))
	r = r.WithContext(ctx)

	w = httptest.NewRecorder()
	m = chi.NewMux()
	m.Post("/tx_approve", handler.ServeHTTP)
	m.ServeHTTP(w, r)
	resp = w.Result()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody = `{
		"status":"rejected", "message":"Not implemented."
	}`
	require.JSONEq(t, wantBody, string(body))
}

func TestTxApproveHandler_serveHTTPForm(t *testing.T) {
	ctx := context.Background()
	issuerAccKeyPair := keypair.MustRandom()
	assetGOAT := txnbuild.CreditAsset{
		Code:   "GOAT",
		Issuer: issuerAccKeyPair.Address(),
	}
	handler := txApproveHandler{
		issuerAccountSecret: issuerAccKeyPair.Seed(),
		assetCode:           assetGOAT.GetCode(),
	}

	// Test if no transaction is submitted.
	req := url.Values{}
	req.Set("tx", "")

	r := httptest.NewRequest("POST", "/tx_approve", strings.NewReader(req.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/tx_approve", handler.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody := `{
		"status":"rejected", "message":"Missing parameter \"tx\"."
	}`
	require.JSONEq(t, wantBody, string(body))

	// Test if can't parse XDR.
	req.Set("tx", "BADXDRTRANSACTIONENVELOPE")
	r = httptest.NewRequest("POST", "/tx_approve", strings.NewReader(req.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r = r.WithContext(ctx)

	w = httptest.NewRecorder()
	m = chi.NewMux()
	m.Post("/tx_approve", handler.ServeHTTP)
	m.ServeHTTP(w, r)
	resp = w.Result()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody = `{
		"status":"rejected", "message":"Invalid parameter \"tx\"."
		}`
	require.JSONEq(t, wantBody, string(body))

	// Test if a non generic transaction fails, same result as malformed XDR.
	kp01 := keypair.MustRandom()
	kp02 := keypair.MustRandom()
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: kp01.Address()},
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
	req.Set("tx", feeBumpTxEnc)
	r = httptest.NewRequest("POST", "/tx_approve", strings.NewReader(req.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r = r.WithContext(ctx)

	w = httptest.NewRecorder()
	m = chi.NewMux()
	m.Post("/tx_approve", handler.ServeHTTP)
	m.ServeHTTP(w, r)
	resp = w.Result()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody = `{
			"status":"rejected", "message":"Invalid parameter \"tx\"."
			}`
	require.JSONEq(t, wantBody, string(body))

	// Test if the transaction sourceAccount the same as the server issuer account
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: issuerAccKeyPair.Address()},
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
	req.Set("tx", txEnc)
	r = httptest.NewRequest("POST", "/tx_approve", strings.NewReader(req.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r = r.WithContext(ctx)

	w = httptest.NewRecorder()
	m = chi.NewMux()
	m.Post("/tx_approve", handler.ServeHTTP)
	m.ServeHTTP(w, r)
	resp = w.Result()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody = `{
				"status":"rejected", "message":"The source account is invalid."
				}`
	require.JSONEq(t, wantBody, string(body))

	// Test if the transaction's operation sourceaccount the same as the server issuer account
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: kp01.Address()},
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
	req.Set("tx", txEnc)
	r = httptest.NewRequest("POST", "/tx_approve", strings.NewReader(req.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r = r.WithContext(ctx)

	w = httptest.NewRecorder()
	m = chi.NewMux()
	m.Post("/tx_approve", handler.ServeHTTP)
	m.ServeHTTP(w, r)
	resp = w.Result()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody = `{
			"status":"rejected", "message":"There is one or more unauthorized operations in the provided transaction."
			}`
	require.JSONEq(t, wantBody, string(body))

	// Test "not implemented"
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: kp01.Address()},
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
	txEnc, err = tx.Base64()
	req.Set("tx", txEnc)
	r = httptest.NewRequest("POST", "/tx_approve", strings.NewReader(req.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r = r.WithContext(ctx)

	w = httptest.NewRecorder()
	m = chi.NewMux()
	m.Post("/tx_approve", handler.ServeHTTP)
	m.ServeHTTP(w, r)
	resp = w.Result()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody = `{
			"status":"rejected", "message":"Not implemented."
		}`
	require.JSONEq(t, wantBody, string(body))
}
