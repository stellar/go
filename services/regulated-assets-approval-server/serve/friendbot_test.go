package serve

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/network"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/base"
	"github.com/stellar/go/support/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFriendbotHandler_validate(t *testing.T) {
	// missing secret seed
	fh := friendbotHandler{}
	err := fh.validate()
	require.EqualError(t, err, "issuer secret cannot be empty")

	// invalid secret seed
	fh = friendbotHandler{
		accountIssuerSecret: "foo bar",
	}
	err = fh.validate()
	require.EqualError(t, err, "the provided string \"foo bar\" is not a valid Stellar account seed")

	// missing asset code
	fh = friendbotHandler{
		accountIssuerSecret: "SB6SFUY6ZJ2ETQHTY456GDAQ547R6NDAU74DTI2CKVVI4JERTUXKB2R4",
	}
	err = fh.validate()
	require.EqualError(t, err, "asset code cannot be empty")

	// missing horizon client
	fh = friendbotHandler{
		accountIssuerSecret: "SB6SFUY6ZJ2ETQHTY456GDAQ547R6NDAU74DTI2CKVVI4JERTUXKB2R4",
		assetCode:           "FOO",
	}
	err = fh.validate()
	require.EqualError(t, err, "horizon client cannot be nil")

	// missing horizon URL
	fh = friendbotHandler{
		accountIssuerSecret: "SB6SFUY6ZJ2ETQHTY456GDAQ547R6NDAU74DTI2CKVVI4JERTUXKB2R4",
		assetCode:           "FOO",
		horizonClient:       horizonclient.DefaultTestNetClient,
	}
	err = fh.validate()
	require.EqualError(t, err, "horizon url cannot be empty")

	// missing network passphrase
	fh = friendbotHandler{
		accountIssuerSecret: "SB6SFUY6ZJ2ETQHTY456GDAQ547R6NDAU74DTI2CKVVI4JERTUXKB2R4",
		assetCode:           "FOO",
		horizonClient:       horizonclient.DefaultTestNetClient,
		horizonURL:          "https://horizon-testnet.stellar.org/",
	}
	err = fh.validate()
	require.EqualError(t, err, "network passphrase cannot be empty")

	// success!
	fh = friendbotHandler{
		accountIssuerSecret: "SB6SFUY6ZJ2ETQHTY456GDAQ547R6NDAU74DTI2CKVVI4JERTUXKB2R4",
		assetCode:           "FOO",
		horizonClient:       horizonclient.DefaultTestNetClient,
		horizonURL:          "https://horizon-testnet.stellar.org/",
		networkPassphrase:   network.TestNetworkPassphrase,
	}
	err = fh.validate()
	require.NoError(t, err)
}

func TestFriendbotHandler_serveHTTP_missingAddress(t *testing.T) {
	ctx := context.Background()

	handler := friendbotHandler{
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

	handler := friendbotHandler{
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

	// mock account that doesn't  exist on ledger
	horizonMock := horizonclient.MockClient{}
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: "GA2ILZPZAQ4R5PRKZ2X2AFAZK3ND6AGA4VFBQGR66BH36PV3VKMWLLZP"}).
		Return(horizon.Account{}, errors.New("something went wrong"))

	handler := friendbotHandler{
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

	// mock account that doesn't  exist on ledger
	horizonMock := horizonclient.MockClient{}
	horizonMock.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: "GA2ILZPZAQ4R5PRKZ2X2AFAZK3ND6AGA4VFBQGR66BH36PV3VKMWLLZP"}).
		Return(horizon.Account{}, nil)

	handler := friendbotHandler{
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
		"error":"Address GA2ILZPZAQ4R5PRKZ2X2AFAZK3ND6AGA4VFBQGR66BH36PV3VKMWLLZP doesn't have a trustline for FOO:GDCRZMSHZGQYSRXPWDMIUNUQW36SV2NIC3C7R6WWT6XDO267WCI2TTBR"
	}`
	require.JSONEq(t, wantBody, string(body))
}

func TestFriendbotHandler_serveHTTP_issuerAccountDoesntExist(t *testing.T) {
	ctx := context.Background()

	// mock account that doesn't  exist on ledger
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
		Return(horizon.Account{}, errors.New("account doesn't exist"))

	handler := friendbotHandler{
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
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody := `{
		"error":"Please make sure the issuer account address already exists in the network."
	}`
	require.JSONEq(t, wantBody, string(body))
}

func TestFriendbotHandler_serveHTTP(t *testing.T) {
	ctx := context.Background()

	// mock account that doesn't  exist on ledger
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

	handler := friendbotHandler{
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
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	wantBody := `{
		"message":"ok"
	}`
	require.JSONEq(t, wantBody, string(body))
}
