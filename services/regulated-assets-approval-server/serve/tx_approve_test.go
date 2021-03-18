package serve

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/network"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/base"
	"github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

/*func TestFriendbotHandler_serveHTTP_missingAddress(t *testing.T) {
	ctx := context.Background()

	handler := txApproveHandler{
		accountIssuerSecret: "SB6SFUY6ZJ2ETQHTY456GDAQ547R6NDAU74DTI2CKVVI4JERTUXKB2R4",
		assetCode:           "FOO",
		horizonClient:       horizonclient.DefaultTestNetClient,
		horizonURL:          "https://horizon-testnet.stellar.org/",
		networkPassphrase:   network.TestNetworkPassphrase,
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/friendbot", nil)
	r = r.WithContext(ctx)
	m := chi.NewMux()
	m.Get("/friendbot", handler.ServeHTTP)
	m.ServeHTTP(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody := `{
		"error":"Missing query paramater \"addr\"."
	}`
	require.JSONEq(t, wantBody, string(body))
}

func TestFriendbotHandler_serveHTTP_invalidAddress(t *testing.T) {
	ctx := context.Background()

	handler := txApproveHandler{
		accountIssuerSecret: "SB6SFUY6ZJ2ETQHTY456GDAQ547R6NDAU74DTI2CKVVI4JERTUXKB2R4",
		assetCode:           "FOO",
		horizonClient:       horizonclient.DefaultTestNetClient,
		horizonURL:          "https://horizon-testnet.stellar.org/",
		networkPassphrase:   network.TestNetworkPassphrase,
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/friendbot?addr=123", nil)
	r = r.WithContext(ctx)
	m := chi.NewMux()
	m.Get("/friendbot", handler.ServeHTTP)
	m.ServeHTTP(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody := `{
		"error":"\"addr\" is not a valid Stellar address."
	}`
	require.JSONEq(t, wantBody, string(body))
}

func TestFriendbotHandler_serveHTTP_accountDoesntExist(t *testing.T) {
	ctx := context.Background()

	horizonMock := horizonclient.MockClient{}
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: "GA2ILZPZAQ4R5PRKZ2X2AFAZK3ND6AGA4VFBQGR66BH36PV3VKMWLLZP"}).
		Return(horizon.Account{}, errors.New("something went wrong")) // account doesn't exist on ledger

	handler := txApproveHandler{
		accountIssuerSecret: "SB6SFUY6ZJ2ETQHTY456GDAQ547R6NDAU74DTI2CKVVI4JERTUXKB2R4",
		assetCode:           "FOO",
		horizonClient:       &horizonMock,
		horizonURL:          "https://horizon-testnet.stellar.org/",
		networkPassphrase:   network.TestNetworkPassphrase,
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/friendbot?addr=GA2ILZPZAQ4R5PRKZ2X2AFAZK3ND6AGA4VFBQGR66BH36PV3VKMWLLZP", nil)
	r = r.WithContext(ctx)
	m := chi.NewMux()
	m.Get("/friendbot", handler.ServeHTTP)
	m.ServeHTTP(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody := `{
		"error":"Please make sure the provided account address already exists in the network."
	}`
	require.JSONEq(t, wantBody, string(body))
}

func TestFriendbotHandler_serveHTTP_missingTrustline(t *testing.T) {
	ctx := context.Background()

	horizonMock := horizonclient.MockClient{}
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: "GA2ILZPZAQ4R5PRKZ2X2AFAZK3ND6AGA4VFBQGR66BH36PV3VKMWLLZP"}).
		Return(horizon.Account{}, nil)

	handler := txApproveHandler{
		accountIssuerSecret: "SB6SFUY6ZJ2ETQHTY456GDAQ547R6NDAU74DTI2CKVVI4JERTUXKB2R4",
		assetCode:           "FOO",
		horizonClient:       &horizonMock,
		horizonURL:          "https://horizon-testnet.stellar.org/",
		networkPassphrase:   network.TestNetworkPassphrase,
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/friendbot?addr=GA2ILZPZAQ4R5PRKZ2X2AFAZK3ND6AGA4VFBQGR66BH36PV3VKMWLLZP", nil)
	r = r.WithContext(ctx)
	m := chi.NewMux()
	m.Get("/friendbot", handler.ServeHTTP)
	m.ServeHTTP(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody := `{
		"error":"Account with address GA2ILZPZAQ4R5PRKZ2X2AFAZK3ND6AGA4VFBQGR66BH36PV3VKMWLLZP doesn't have a trustline for FOO:GDCRZMSHZGQYSRXPWDMIUNUQW36SV2NIC3C7R6WWT6XDO267WCI2TTBR"
	}`
	require.JSONEq(t, wantBody, string(body))
}

func TestFriendbotHandler_serveHTTP_issuerAccountDoesntExist(t *testing.T) {
	ctx := context.Background()

	// declare a logging buffer to validate output logs
	buf := new(strings.Builder)
	log.DefaultLogger.Logger.SetOutput(buf)
	log.DefaultLogger.Logger.SetLevel(log.InfoLevel)

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
		Return(horizon.Account{}, errors.New("account doesn't exist")) // issuer account doesn't exist on ledger

	handler := txApproveHandler{
		accountIssuerSecret: "SDVFEIZ3WH5F6GHGK56QITTC5IO6QJ2UIQDWCHE72DAFZFSXYPIHQ6EV", // GDDIO6SFRD4SJEQFJOSKPIDYTDM7LM4METFBKN4NFGVR5DTGB7H75N5S
		assetCode:           "FOO",
		horizonClient:       &horizonMock,
		horizonURL:          "https://horizon-testnet.stellar.org/",
		networkPassphrase:   network.TestNetworkPassphrase,
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/friendbot?addr=GA2ILZPZAQ4R5PRKZ2X2AFAZK3ND6AGA4VFBQGR66BH36PV3VKMWLLZP", nil)
	r = r.WithContext(ctx)
	m := chi.NewMux()
	m.Get("/friendbot", handler.ServeHTTP)
	m.ServeHTTP(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody := `{
		"error":"An error occurred while processing this request."
	}`
	require.JSONEq(t, wantBody, string(body))
	require.Contains(t, buf.String(), "getting detail for issuer account")
}
*/
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

	handler := txApproveHandler{
		accountIssuerSecret: "SDVFEIZ3WH5F6GHGK56QITTC5IO6QJ2UIQDWCHE72DAFZFSXYPIHQ6EV", // GDDIO6SFRD4SJEQFJOSKPIDYTDM7LM4METFBKN4NFGVR5DTGB7H75N5S
		assetCode:           "FOO",
		horizonClient:       &horizonMock,
		horizonURL:          "https://horizon-testnet.stellar.org/",
		networkPassphrase:   network.TestNetworkPassphrase,
	}
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

	// w := httptest.NewRecorder()
	// r := httptest.NewRequest("POST", "/tx_approve", strings.NewReader(req))
	// r = r.WithContext(ctx)
	// m := chi.NewMux()
	// m.Post("/tx_approve", handler.ServeHTTP)
	// m.ServeHTTP(w, r)

	// resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody := `{
		"message":"ok"
	}`
	require.JSONEq(t, wantBody, string(body))
}
