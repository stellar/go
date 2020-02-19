package serve

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stellar/go/exp/services/recoverysigner/internal/account"
	"github.com/stellar/go/exp/services/recoverysigner/internal/serve/auth"
	"github.com/stellar/go/keypair"
	supportlog "github.com/stellar/go/support/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountPost_new(t *testing.T) {
	s := account.NewMemoryStore()
	h := accountPostHandler{
		Logger:         supportlog.DefaultLogger,
		AccountStore:   s,
		SigningAddress: keypair.MustParseAddress("GCAPXRXSU7P6D353YGXMP6ROJIC744HO5OZCIWTXZQK2X757YU5KCHUE"),
	}

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{Address: "GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N"})
	req := `{
	"type": "share",
	"owner_identities": {
		"account": "GBF3XFXGBGNQDN3HOSZ7NVRF6TJ2JOD5U6ELIWJOOEI6T5WKMQT2YSXQ",
		"phone_number": "+10000000000",
		"email": "user1@example.com"
	},
	"other_identities": {
		"account": "GB5VOTKJ3IPGIYQBJ6GVJMUVEAIYGXZUJE4WYLPBJSHOTKLZTETBYOBI",
		"phone_number": "+20000000000",
		"email": "user2@example.com"
	}
}`
	r := httptest.NewRequest("POST", "/GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N", strings.NewReader(req))
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{
	"address": "GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N",
	"type": "share",
	"identity": "account",
	"signer": "GCAPXRXSU7P6D353YGXMP6ROJIC744HO5OZCIWTXZQK2X757YU5KCHUE"
}`
	assert.JSONEq(t, wantBody, string(body))

	acc, err := s.Get("GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N")
	require.NoError(t, err)
	wantAcc := account.Account{
		Address: "GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N",
		Type:    "share",
		OwnerIdentities: account.Identities{
			Address:     "GBF3XFXGBGNQDN3HOSZ7NVRF6TJ2JOD5U6ELIWJOOEI6T5WKMQT2YSXQ",
			PhoneNumber: "+10000000000",
			Email:       "user1@example.com",
		},
		OtherIdentities: account.Identities{
			Address:     "GB5VOTKJ3IPGIYQBJ6GVJMUVEAIYGXZUJE4WYLPBJSHOTKLZTETBYOBI",
			PhoneNumber: "+20000000000",
			Email:       "user2@example.com",
		},
	}
	assert.Equal(t, wantAcc, acc)
}

func TestAccountPost_accountAlreadyExists(t *testing.T) {
	s := account.NewMemoryStore()
	s.Add(account.Account{
		Address: "GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N",
		Type:    "personal",
	})
	h := accountPostHandler{
		Logger:         supportlog.DefaultLogger,
		AccountStore:   s,
		SigningAddress: keypair.MustParseAddress("GCAPXRXSU7P6D353YGXMP6ROJIC744HO5OZCIWTXZQK2X757YU5KCHUE"),
	}

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{Address: "GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N"})
	req := `{
	"type": "share"
}`
	r := httptest.NewRequest("POST", "/GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N", strings.NewReader(req))
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{
	"error": "The request could not be completed because the resource already exists."
}`
	assert.JSONEq(t, wantBody, string(body))

	acc, err := s.Get("GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N")
	require.NoError(t, err)
	wantAcc := account.Account{
		Address: "GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N",
		Type:    "personal",
	}
	assert.Equal(t, wantAcc, acc)
}

func TestAccountPost_notAuthenticatedForAccount(t *testing.T) {
	s := account.NewMemoryStore()
	h := accountPostHandler{
		Logger:         supportlog.DefaultLogger,
		AccountStore:   s,
		SigningAddress: keypair.MustParseAddress("GCAPXRXSU7P6D353YGXMP6ROJIC744HO5OZCIWTXZQK2X757YU5KCHUE"),
	}

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{Address: "GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N"})
	req := `{
	"type": "personal"
}`
	r := httptest.NewRequest("POST", "/GAIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N", strings.NewReader(req))
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{
	"error": "The request could not be authenticated."
}`
	assert.JSONEq(t, wantBody, string(body))

	_, err = s.Get("GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N")
	assert.Equal(t, account.ErrNotFound, err)
}
