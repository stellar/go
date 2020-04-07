package serve

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stellar/go/exp/services/recoverysigner/internal/account"
	"github.com/stellar/go/exp/services/recoverysigner/internal/db/dbtest"
	"github.com/stellar/go/exp/services/recoverysigner/internal/serve/auth"
	"github.com/stellar/go/keypair"
	supportlog "github.com/stellar/go/support/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test that when authenticated with an account, but querying the wrong account,
// an error is returned.
func TestAccountGet_authenticatedNotAuthorized(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	s.Add(account.Account{
		Address: "GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N",
		Identities: []account.Identity{
			{
				Role: "sender",
				AuthMethods: []account.AuthMethod{
					{Type: account.AuthMethodTypeAddress, Value: "GCGZ3CNBE47IWAA5YIKDZL2XYYLA2UKJPS55P5EJ4VOMLK523PF3G7EM"},
				},
			},
		},
	})
	h := accountGetHandler{
		Logger:         supportlog.DefaultLogger,
		AccountStore:   s,
		SigningAddress: keypair.MustParseAddress("GCAPXRXSU7P6D353YGXMP6ROJIC744HO5OZCIWTXZQK2X757YU5KCHUE"),
	}

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{Address: "GCNPATZQVSFGGSAHR4T54WNELPHYEBTSKH4IIKUTC7CHPLG6EPPC4PJL"})
	r := httptest.NewRequest("GET", "/GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N", nil)
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Get("/{address}", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{
	"error": "The resource at the url requested was not found."
}`

	assert.JSONEq(t, wantBody, string(body))
}

func TestAccountGet_notAuthenticated(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	s.Add(account.Account{
		Address: "GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N",
		Identities: []account.Identity{
			{
				Role: "sender",
				AuthMethods: []account.AuthMethod{
					{Type: account.AuthMethodTypeAddress, Value: "GCGZ3CNBE47IWAA5YIKDZL2XYYLA2UKJPS55P5EJ4VOMLK523PF3G7EM"},
				},
			},
		},
	})
	h := accountGetHandler{
		Logger:         supportlog.DefaultLogger,
		AccountStore:   s,
		SigningAddress: keypair.MustParseAddress("GCAPXRXSU7P6D353YGXMP6ROJIC744HO5OZCIWTXZQK2X757YU5KCHUE"),
	}

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{})
	r := httptest.NewRequest("GET", "/GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N", nil)
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Get("/{address}", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{
	"error": "The request could not be authenticated."
}`

	assert.JSONEq(t, wantBody, string(body))
}

func TestAccountGet_notFound(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	s.Add(account.Account{
		Address: "GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N",
		Identities: []account.Identity{
			{
				Role: "sender",
				AuthMethods: []account.AuthMethod{
					{Type: account.AuthMethodTypeAddress, Value: "GCGZ3CNBE47IWAA5YIKDZL2XYYLA2UKJPS55P5EJ4VOMLK523PF3G7EM"},
				},
			},
		},
	})
	h := accountGetHandler{
		Logger:         supportlog.DefaultLogger,
		AccountStore:   s,
		SigningAddress: keypair.MustParseAddress("GCAPXRXSU7P6D353YGXMP6ROJIC744HO5OZCIWTXZQK2X757YU5KCHUE"),
	}

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{Address: "GCGZ3CNBE47IWAA5YIKDZL2XYYLA2UKJPS55P5EJ4VOMLK523PF3G7EM"})
	r := httptest.NewRequest("GET", "/GCGZ3CNBE47IWAA5YIKDZL2XYYLA2UKJPS55P5EJ4VOMLK523PF3G7EM", nil)
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Get("/{address}", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{
	"error": "The resource at the url requested was not found."
}`

	assert.JSONEq(t, wantBody, string(body))
}

func TestAccountGet_authenticatedByAddress(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	s.Add(account.Account{
		Address: "GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N",
		Identities: []account.Identity{
			{
				Role: "sender",
				AuthMethods: []account.AuthMethod{
					{Type: account.AuthMethodTypeAddress, Value: "GCGZ3CNBE47IWAA5YIKDZL2XYYLA2UKJPS55P5EJ4VOMLK523PF3G7EM"},
				},
			},
			{
				Role: "receiver",
				AuthMethods: []account.AuthMethod{
					{Type: account.AuthMethodTypePhoneNumber, Value: "+10000000000"},
				},
			},
		},
	})
	h := accountGetHandler{
		Logger:         supportlog.DefaultLogger,
		AccountStore:   s,
		SigningAddress: keypair.MustParseAddress("GCAPXRXSU7P6D353YGXMP6ROJIC744HO5OZCIWTXZQK2X757YU5KCHUE"),
	}

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{Address: "GCGZ3CNBE47IWAA5YIKDZL2XYYLA2UKJPS55P5EJ4VOMLK523PF3G7EM"})
	r := httptest.NewRequest("GET", "/GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N", nil)
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Get("/{address}", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{
	"address": "GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N",
	"identities": [
		{
			"role": "sender",
			"authenticated": true
		},
		{
			"role": "receiver"
		}
	],
	"signer": "GCAPXRXSU7P6D353YGXMP6ROJIC744HO5OZCIWTXZQK2X757YU5KCHUE"
}`

	assert.JSONEq(t, wantBody, string(body))
}

func TestAccountGet_authenticatedByPhoneNumber(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	s.Add(account.Account{
		Address: "GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N",
		Identities: []account.Identity{
			{
				Role: "sender",
				AuthMethods: []account.AuthMethod{
					{Type: account.AuthMethodTypeAddress, Value: "GCGZ3CNBE47IWAA5YIKDZL2XYYLA2UKJPS55P5EJ4VOMLK523PF3G7EM"},
				},
			},
			{
				Role: "receiver",
				AuthMethods: []account.AuthMethod{
					{Type: account.AuthMethodTypePhoneNumber, Value: "+10000000000"},
				},
			},
		},
	})
	h := accountGetHandler{
		Logger:         supportlog.DefaultLogger,
		AccountStore:   s,
		SigningAddress: keypair.MustParseAddress("GCAPXRXSU7P6D353YGXMP6ROJIC744HO5OZCIWTXZQK2X757YU5KCHUE"),
	}

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{PhoneNumber: "+10000000000"})
	r := httptest.NewRequest("GET", "/GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N", nil)
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Get("/{address}", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{
	"address": "GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N",
	"identities": [
		{
			"role": "sender"
		},
		{
			"role": "receiver",
			"authenticated": true
		}
	],
	"signer": "GCAPXRXSU7P6D353YGXMP6ROJIC744HO5OZCIWTXZQK2X757YU5KCHUE"
}`

	assert.JSONEq(t, wantBody, string(body))
}

func TestAccountGet_authenticatedByEmail(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	s.Add(account.Account{
		Address: "GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N",
		Identities: []account.Identity{
			{
				Role: "sender",
				AuthMethods: []account.AuthMethod{
					{Type: account.AuthMethodTypeAddress, Value: "GCGZ3CNBE47IWAA5YIKDZL2XYYLA2UKJPS55P5EJ4VOMLK523PF3G7EM"},
				},
			},
			{
				Role: "receiver",
				AuthMethods: []account.AuthMethod{
					{Type: account.AuthMethodTypeEmail, Value: "user1@example.com"},
				},
			},
		},
	})
	h := accountGetHandler{
		Logger:         supportlog.DefaultLogger,
		AccountStore:   s,
		SigningAddress: keypair.MustParseAddress("GCAPXRXSU7P6D353YGXMP6ROJIC744HO5OZCIWTXZQK2X757YU5KCHUE"),
	}

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{Email: "user1@example.com"})
	r := httptest.NewRequest("GET", "/GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N", nil)
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Get("/{address}", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{
	"address": "GDIXCQJ2W2N6TAS6AYW4LW2EBV7XNRUCLNHQB37FARDEWBQXRWP47Q6N",
	"identities": [
		{
			"role": "sender"
		},
		{
			"role": "receiver",
			"authenticated": true
		}
	],
	"signer": "GCAPXRXSU7P6D353YGXMP6ROJIC744HO5OZCIWTXZQK2X757YU5KCHUE"
}`

	assert.JSONEq(t, wantBody, string(body))
}
