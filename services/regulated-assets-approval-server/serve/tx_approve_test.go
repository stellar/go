package serve

import (
	"context"
	"testing"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTxApproveHandler_isRejected(t *testing.T) {
	ctx := context.Background()
	distAccKeyPair := keypair.MustRandom()
	issuerAccKeyPair := keypair.MustRandom()
	assetGOAT := txnbuild.CreditAsset{
		Code:   "GOAT",
		Issuer: issuerAccKeyPair.Address(),
	}

	req := txApproveRequest{
		Transaction: "",
	}
	rejectedResponse, err := txApproveHandler{DistributionAccount: distAccKeyPair.Address()}.isRejected(ctx, req)
	require.NoError(t, err)
	wantRejectedResponse := txApproveResponse{
		Status:  Rejected,
		Message: "Missing parameter \"tx\"",
	}
	assert.Equal(t, &wantRejectedResponse, rejectedResponse)
	wantRejectedResponse = txApproveResponse{
		Status:  Rejected,
		Message: "Missing parameter \"tx\"",
	}
	req = txApproveRequest{
		Transaction: "BADXDRTRANSACTIONENVELOPE",
	}
	rejectedResponse, err = txApproveHandler{}.isRejected(ctx, req)
	require.EqualError(t, err, "Parsing transaction failed.")
	wantRejectedResponse = txApproveResponse{
		Status:  Rejected,
		Message: "Invalid parameter \"tx\"",
	}
	assert.Equal(t, &wantRejectedResponse, rejectedResponse)
	// Test if a "fee bump" transaction fails
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
	t.Log("Tx:", feeBumpTxEnc)
	req = txApproveRequest{
		Transaction: feeBumpTxEnc,
	}
	rejectedResponse, err = txApproveHandler{DistributionAccount: distAccKeyPair.Address()}.isRejected(ctx, req)
	require.EqualError(t, err, "Transaction is not a simple transaction.")
	assert.Equal(t, &wantRejectedResponse, rejectedResponse)
	tx, err = txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: distAccKeyPair.Address()},
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
}

//! Mute until TestTxApproveHandler_isRejected is complete
/*
func TestTxApproveHandler_serveHTTP(t *testing.T) {
	ctx := context.Background()

	horizonMock := horizonclient.MockClient{}
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: "GA2ILZPZAQ4R5PRKZ2X2AFAZK3ND6AGA4VFBQGR66BH36PV3VKMWLLZP"}).
		Return(horizon.Account{
			Balances: []horizon.Balance{
				{
					Asset:   base.Asset{Code: "FOO", Issuer: "GDDIO6SFRD4SJEQFJOSKPIDYTDM7LM4METFBKN4NFGVR5DTGB7H75N5S"},
					Balance: "0",
				},
			},
		}, nil)
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: "GDDIO6SFRD4SJEQFJOSKPIDYTDM7LM4METFBKN4NFGVR5DTGB7H75N5S"}).
		Return(horizon.Account{
			AccountID: "GDDIO6SFRD4SJEQFJOSKPIDYTDM7LM4METFBKN4NFGVR5DTGB7H75N5S",
			Sequence:  "1",
		}, nil)
	horizonMock.
		On("SubmitTransaction", mock.AnythingOfType("*txnbuild.Transaction")).
		Return(horizon.Transaction{}, nil)

	handler := txApproveHandler{}
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4"},
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.SetOptions{
					Signer: &txnbuild.Signer{
						Address: "GD7CGJSJ5OBOU5KOP2UQDH3MPY75UTEY27HVV5XPSL2X6DJ2VGTOSXEU",
						Weight:  20,
					},
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewTimebounds(0, 1),
		},
	)
	require.NoError(t, err)
	txEnc, err := tx.Base64()
	require.NoError(t, err)
	t.Log("Tx:", txEnc)
	req := `{
	"tx": "` + txEnc + `"
}`
	r := httptest.NewRequest("POST", "/tx_approve", strings.NewReader(req))
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/tx_approve", handler.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	req = `{
		"tx": "brokenXDR"
	}`
	r = httptest.NewRequest("POST", "/tx_approve", strings.NewReader(req))
	r = r.WithContext(ctx)

	w = httptest.NewRecorder()
	m = chi.NewMux()
	m.Post("/tx_approve", handler.ServeHTTP)
	m.ServeHTTP(w, r)
	resp = w.Result()

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody := `{
		{"status":"rejected", "error":"Invalid parameter \"tx\""}
	}`
	require.JSONEq(t, wantBody, string(body))
}
*/
