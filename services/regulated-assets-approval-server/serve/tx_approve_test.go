package serve

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
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
	handler := txApproveHandler{
		issuerKP:  issuerAccKeyPair,
		assetCode: assetGOAT.GetCode(),
	}
	rejectedResponse, err := handler.isRejected(ctx, req)
	require.NoError(t, err)
	wantRejectedResponse := txApproveResponse{
		Status: Sep8StatusRejected,
		Error:  missingParamErr,
	}
	assert.Equal(t, &wantRejectedResponse, rejectedResponse)

	// Test if can't parse XDR.
	req = txApproveRequest{
		Transaction: "BADXDRTRANSACTIONENVELOPE",
	}
	rejectedResponse, err = handler.isRejected(ctx, req)
	require.NoError(t, err)
	wantRejectedResponse = txApproveResponse{
		Status: Sep8StatusRejected,
		Error:  invalidParamErr,
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
	rejectedResponse, err = handler.isRejected(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, &wantRejectedResponse, rejectedResponse) // wantRejectedResponse is identical to "if can't parse XDR".

	// Test if the transaction sourceAccount the same as the server issuer account.
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
	rejectedResponse, err = handler.isRejected(ctx, req)
	require.NoError(t, err)
	wantRejectedResponse = txApproveResponse{
		Status: Sep8StatusRejected,
		Error:  invalidSrcAccErr,
	}
	assert.Equal(t, &wantRejectedResponse, rejectedResponse)

	// Test if the transaction's operation if operation is a payment.
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: kp01.Address()},
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.CreateClaimableBalance{
					SourceAccount: issuerAccKeyPair.Address(),
					Asset:         assetGOAT,
					Amount:        "1",
					Destinations: []txnbuild.Claimant{
						txnbuild.NewClaimant(kp02.Address(), nil),
					},
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
	rejectedResponse, err = handler.isRejected(ctx, req)
	require.NoError(t, err)
	wantRejectedResponse = txApproveResponse{
		Status: Sep8StatusRejected,
		Error:  unauthorizedOpErr,
	}
	assert.Equal(t, &wantRejectedResponse, rejectedResponse)

	// Test if the transaction's has more than one operation.
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
				&txnbuild.CreateClaimableBalance{
					SourceAccount: issuerAccKeyPair.Address(),
					Asset:         assetGOAT,
					Amount:        "1",
					Destinations: []txnbuild.Claimant{
						txnbuild.NewClaimant(kp02.Address(), nil),
					},
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
	rejectedResponse, err = handler.isRejected(ctx, req)
	require.NoError(t, err)
	wantRejectedResponse = txApproveResponse{
		Status: Sep8StatusRejected,
		Error:  exceededOpsLenErr,
	}
	assert.Equal(t, &wantRejectedResponse, rejectedResponse)

	// Test if the transaction's operation sourceAccount the same as the server issuer account
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
	rejectedResponse, err = handler.isRejected(ctx, req)
	require.NoError(t, err)
	wantRejectedResponse = txApproveResponse{
		Status: Sep8StatusRejected,
		Error:  unauthorizedOpErr,
	}
	assert.Equal(t, &wantRejectedResponse, rejectedResponse)

	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: kp01.Address()},
			IncrementSequenceNum: false,
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
	rejectedResponse, err = handler.isRejected(ctx, req)
	require.NoError(t, err)
	wantRejectedResponse = txApproveResponse{
		Status: Sep8StatusRejected,
		Error:  unauthorizedOpErr,
	}
	assert.Equal(t, &wantRejectedResponse, rejectedResponse)

	// Test revisable transaction
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
	rejectedResponse, err = handler.isRejected(ctx, req)
	require.NoError(t, err)
	assert.Nil(t, rejectedResponse)
}

func TestTxApproveHandler_Approve(t *testing.T) {
	ctx := context.Background()
	issuerAccKeyPair := keypair.MustRandom()
	assetGOAT := txnbuild.CreditAsset{
		Code:   "GOAT",
		Issuer: issuerAccKeyPair.Address(),
	}
	kp01 := keypair.MustRandom()
	kp02 := keypair.MustRandom()
	tx, err := txnbuild.NewTransaction(
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
			Timebounds: txnbuild.NewTimeout(400),
		},
	)
	require.NoError(t, err)
	txEnc, err := tx.Base64()
	req := txApproveRequest{
		Transaction: txEnc,
	}

	approvedResponse, err := txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         assetGOAT.GetCode(),
		networkPassphrase: network.TestNetworkPassphrase,
	}.Approve(ctx, req)
	require.NoError(t, err)

	// Decode the request transaction.
	parsed, err := txnbuild.TransactionFromXDR(approvedResponse.Transaction)
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
}

func TestTxApproveHandler_serveHTTPJson(t *testing.T) {
	ctx := context.Background()
	issuerAccKeyPair := keypair.MustRandom()
	assetGOAT := txnbuild.CreditAsset{
		Code:   "GOAT",
		Issuer: issuerAccKeyPair.Address(),
	}
	handler := txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         assetGOAT.GetCode(),
		networkPassphrase: network.TestNetworkPassphrase,
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
		"status":"rejected", "error":"Missing parameter \"tx\"."
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
		"status":"rejected", "error":"Invalid parameter \"tx\"."
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
		"status":"rejected", "error":"Invalid parameter \"tx\"."
	}`
	require.JSONEq(t, wantBody, string(body))

	// Test if the transaction sourceAccount the same as the server issuer account.
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
		"status":"rejected", "error":"The source account is invalid."
	}`
	require.JSONEq(t, wantBody, string(body))

	// Test if the transaction's operation sourceAccount the same as the server issuer account.
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
		"status":"rejected", "error":"There is one or more unauthorized operations in the provided transaction."
	}`
	require.JSONEq(t, wantBody, string(body))

	// Test revisable transaction.
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

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	var response map[string]string
	err = json.Unmarshal([]byte(string(body)), &response)

	_, exists := response["status"]
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, response["status"], "revised")
	_, exists = response["status"]
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, response["message"], "Authorization and deauthorization operations were added.")
	_, exists = response["tx"]
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestTxApproveHandler_serveHTTPForm(t *testing.T) {
	ctx := context.Background()
	issuerAccKeyPair := keypair.MustRandom()
	assetGOAT := txnbuild.CreditAsset{
		Code:   "GOAT",
		Issuer: issuerAccKeyPair.Address(),
	}
	handler := txApproveHandler{
		issuerKP:          issuerAccKeyPair,
		assetCode:         assetGOAT.GetCode(),
		networkPassphrase: network.TestNetworkPassphrase,
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
		"status":"rejected", "error":"Missing parameter \"tx\"."
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
		"status":"rejected", "error":"Invalid parameter \"tx\"."
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
			"status":"rejected", "error":"Invalid parameter \"tx\"."
			}`
	require.JSONEq(t, wantBody, string(body))

	// Test if the transaction sourceAccount the same as the server issuer account.
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
				"status":"rejected", "error":"The source account is invalid."
				}`
	require.JSONEq(t, wantBody, string(body))

	// Test if the transaction's operation sourceaccount the same as the server issuer account.
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
			"status":"rejected", "error":"There is one or more unauthorized operations in the provided transaction."
			}`
	require.JSONEq(t, wantBody, string(body))

	// Test revisable transaction.
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

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	var response map[string]string
	err = json.Unmarshal([]byte(string(body)), &response)

	_, exists := response["status"]
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, response["status"], "revised")
	_, exists = response["status"]
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, response["message"], "Authorization and deauthorization operations were added.")
	// Cant test the tx result due to the the hash being nondeterministic.
	_, exists = response["tx"]
	require.NoError(t, err)
	assert.True(t, exists)
}
