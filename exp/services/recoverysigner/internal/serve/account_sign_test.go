package serve

import (
	"context"
	"github.com/stellar/go/xdr"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stellar/go/exp/services/recoverysigner/internal/account"
	"github.com/stellar/go/exp/services/recoverysigner/internal/db/dbtest"
	"github.com/stellar/go/exp/services/recoverysigner/internal/serve/auth"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	supportlog "github.com/stellar/go/support/log"
	"github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test that when the account does not exist it returns not found.
func TestAccountSign_authenticatedButNotFound(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	h := accountSignHandler{
		Logger:            supportlog.DefaultLogger,
		AccountStore:      s,
		SigningKey:        keypair.MustParseFull("SBIB72S6JMTGJRC6LMKLC5XMHZ2IOHZSZH4SASTN47LECEEJ7QEB6EYK"),
		NetworkPassphrase: network.TestNetworkPassphrase,
	}

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4"})
	req := `{
	"transaction": ""
}`
	r := httptest.NewRequest("POST", "/GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4/sign", strings.NewReader(req))
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}/sign", h.ServeHTTP)
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

// Test that when the account exists but the authenticated client does not have
// permission to access it returns not found.
func TestAccountSign_accountAuthenticatedButNotPermitted(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	s.Add(account.Account{
		Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4",
	})
	s.Add(account.Account{
		Address: "GBLOP46WEVXWO5N75TDX7GXLYFQE3XLDT5NQ2VYIBEWWEMSZWR3AUISZ",
	})
	h := accountSignHandler{
		Logger:            supportlog.DefaultLogger,
		AccountStore:      s,
		SigningKey:        keypair.MustParseFull("SBIB72S6JMTGJRC6LMKLC5XMHZ2IOHZSZH4SASTN47LECEEJ7QEB6EYK"),
		NetworkPassphrase: network.TestNetworkPassphrase,
	}

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{Address: "GBLOP46WEVXWO5N75TDX7GXLYFQE3XLDT5NQ2VYIBEWWEMSZWR3AUISZ"})
	req := `{
	"transaction": ""
}`
	r := httptest.NewRequest("POST", "/GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4/sign", strings.NewReader(req))
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}/sign", h.ServeHTTP)
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

func TestAccountSign_accountAuthenticatedButInvalidAddress(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	h := accountSignHandler{
		Logger:            supportlog.DefaultLogger,
		AccountStore:      s,
		SigningKey:        keypair.MustParseFull("SBIB72S6JMTGJRC6LMKLC5XMHZ2IOHZSZH4SASTN47LECEEJ7QEB6EYK"),
		NetworkPassphrase: network.TestNetworkPassphrase,
	}

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4"})
	r := httptest.NewRequest("POST", "/ZA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4/sign", nil)
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}/sign", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{"error": "The request was invalid in some way."}`
	assert.JSONEq(t, wantBody, string(body))
}

func TestAccountSign_accountAuthenticatedButEmptyAddress(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	h := accountSignHandler{
		Logger:            supportlog.DefaultLogger,
		AccountStore:      s,
		SigningKey:        keypair.MustParseFull("SBIB72S6JMTGJRC6LMKLC5XMHZ2IOHZSZH4SASTN47LECEEJ7QEB6EYK"),
		NetworkPassphrase: network.TestNetworkPassphrase,
	}

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4"})
	r := httptest.NewRequest("POST", "//sign", nil)
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}/sign", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{"error": "The request was invalid in some way."}`
	assert.JSONEq(t, wantBody, string(body))
}

// Test that when the account exists but the authenticated client does not have
// permission to access it returns not found.
func TestAccountSign_phoneNumberAuthenticatedButNotPermitted(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	s.Add(account.Account{
		Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4",
		Identities: []account.Identity{
			{
				Role: "sender",
				AuthMethods: []account.AuthMethod{
					{Type: account.AuthMethodTypePhoneNumber, Value: "+10000000000"},
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
	s.Add(account.Account{
		Address: "GBLOP46WEVXWO5N75TDX7GXLYFQE3XLDT5NQ2VYIBEWWEMSZWR3AUISZ",
		Identities: []account.Identity{
			{
				Role: "sender",
				AuthMethods: []account.AuthMethod{
					{Type: account.AuthMethodTypePhoneNumber, Value: "+20000000000"},
				},
			},
			{
				Role: "receiver",
				AuthMethods: []account.AuthMethod{
					{Type: account.AuthMethodTypePhoneNumber, Value: "+20000000000"},
				},
			},
		},
	})
	h := accountSignHandler{
		Logger:            supportlog.DefaultLogger,
		AccountStore:      s,
		SigningKey:        keypair.MustParseFull("SBIB72S6JMTGJRC6LMKLC5XMHZ2IOHZSZH4SASTN47LECEEJ7QEB6EYK"),
		NetworkPassphrase: network.TestNetworkPassphrase,
	}

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{PhoneNumber: "+20000000000"})
	req := `{
	"transaction": ""
}`
	r := httptest.NewRequest("POST", "/GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4/sign", strings.NewReader(req))
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}/sign", h.ServeHTTP)
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

// Test that when the account exists but the authenticated client does not have
// permission to access it returns not found.
func TestAccountSign_emailAuthenticatedButNotPermitted(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	s.Add(account.Account{
		Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4",
		Identities: []account.Identity{
			{
				Role: "sender",
				AuthMethods: []account.AuthMethod{
					{Type: account.AuthMethodTypeEmail, Value: "user1@example.com"},
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
	s.Add(account.Account{
		Address: "GBLOP46WEVXWO5N75TDX7GXLYFQE3XLDT5NQ2VYIBEWWEMSZWR3AUISZ",
		Identities: []account.Identity{
			{
				Role: "sender",
				AuthMethods: []account.AuthMethod{
					{Type: account.AuthMethodTypeEmail, Value: "user2@example.com"},
				},
			},
			{
				Role: "receiver",
				AuthMethods: []account.AuthMethod{
					{Type: account.AuthMethodTypeEmail, Value: "user2@example.com"},
				},
			},
		},
	})
	h := accountSignHandler{
		Logger:            supportlog.DefaultLogger,
		AccountStore:      s,
		SigningKey:        keypair.MustParseFull("SBIB72S6JMTGJRC6LMKLC5XMHZ2IOHZSZH4SASTN47LECEEJ7QEB6EYK"),
		NetworkPassphrase: network.TestNetworkPassphrase,
	}

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{PhoneNumber: "user2@example.com"})
	req := `{
	"transaction": ""
}`
	r := httptest.NewRequest("POST", "/GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4/sign", strings.NewReader(req))
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}/sign", h.ServeHTTP)
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

// Test that when the source account of the transaction matches the account the
// request is for, that the transaction is signed and a signature is returned.
// The operation source account does not need to be set.
func TestAccountSign_accountAuthenticatedTxSourceAccountValid(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	s.Add(account.Account{
		Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4",
	})
	h := accountSignHandler{
		Logger:            supportlog.DefaultLogger,
		AccountStore:      s,
		SigningKey:        keypair.MustParseFull("SBIB72S6JMTGJRC6LMKLC5XMHZ2IOHZSZH4SASTN47LECEEJ7QEB6EYK"),
		NetworkPassphrase: network.TestNetworkPassphrase,
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

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4"})
	req := `{
	"transaction": "` + txEnc + `"
}`
	r := httptest.NewRequest("POST", "/GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4/sign", strings.NewReader(req))
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}/sign", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{
	"signer": "GBOG4KF66M4AFRBUHOTJQJRO7BGGFCSGIICTI5BHXHKXCWV2C67QRN5H",
	"signature": "okp0ISR/hjU6ItsfXie6ArlQ3YWkBBqEAM5TJrthALdawV5DzcpuwBKi0QE/iBgoU7eY0hY3RPdxm8mXGNYfCQ==",
	"network_passphrase": "Test SDF Network ; September 2015"
}`
	assert.JSONEq(t, wantBody, string(body))
}

// Test that when the source account of the transaction and operation are both
// set to values that match the account the request is for, that the
// transaction is signed and a signature is returned.
func TestAccountSign_accountAuthenticatedTxAndOpSourceAccountValid(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	s.Add(account.Account{
		Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4",
	})
	h := accountSignHandler{
		Logger:            supportlog.DefaultLogger,
		AccountStore:      s,
		SigningKey:        keypair.MustParseFull("SBIB72S6JMTGJRC6LMKLC5XMHZ2IOHZSZH4SASTN47LECEEJ7QEB6EYK"),
		NetworkPassphrase: network.TestNetworkPassphrase,
	}

	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4"},
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.SetOptions{
					SourceAccount: &txnbuild.SimpleAccount{AccountID: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4"},
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

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4"})
	req := `{
	"transaction": "` + txEnc + `"
}`
	r := httptest.NewRequest("POST", "/GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4/sign", strings.NewReader(req))
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}/sign", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{
	"signer": "GBOG4KF66M4AFRBUHOTJQJRO7BGGFCSGIICTI5BHXHKXCWV2C67QRN5H",
	"signature": "MKAkl+R3VT5DJw6Qed8jO8ERD4RcQ4dJlN+UR2n7nT6AVBXnKBk0zqBZnDuB153zfTYmuA5kmsRiNr5terHVBg==",
	"network_passphrase": "Test SDF Network ; September 2015"
}`
	assert.JSONEq(t, wantBody, string(body))
}

// Test that when the source account of the transaction is not the account sign
// the request is calling sign on a bad request response is returned.
func TestAccountSign_accountAuthenticatedTxSourceAccountInvalid(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	s.Add(account.Account{
		Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4",
	})
	h := accountSignHandler{
		Logger:            supportlog.DefaultLogger,
		AccountStore:      s,
		SigningKey:        keypair.MustParseFull("SBIB72S6JMTGJRC6LMKLC5XMHZ2IOHZSZH4SASTN47LECEEJ7QEB6EYK"),
		NetworkPassphrase: network.TestNetworkPassphrase,
	}

	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: "GA47G3ZQBUR5NF2ZECG774O3QGKFMAW75XLXSCDICFDDV5GKGRFGFSOI"},
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.SetOptions{
					SourceAccount: &txnbuild.SimpleAccount{AccountID: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4"},
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

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4"})
	req := `{
	"transaction": "` + txEnc + `"
}`
	r := httptest.NewRequest("POST", "/GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4/sign", strings.NewReader(req))
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}/sign", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{"error": "The request was invalid in some way."}`
	assert.JSONEq(t, wantBody, string(body))
}

// Test that when the source account of the operation is not the account sign
// the request is calling sign on a bad request response is returned.
func TestAccountSign_accountAuthenticatedOpSourceAccountInvalid(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	s.Add(account.Account{
		Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4",
	})
	h := accountSignHandler{
		Logger:            supportlog.DefaultLogger,
		AccountStore:      s,
		SigningKey:        keypair.MustParseFull("SBIB72S6JMTGJRC6LMKLC5XMHZ2IOHZSZH4SASTN47LECEEJ7QEB6EYK"),
		NetworkPassphrase: network.TestNetworkPassphrase,
	}

	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4"},
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.SetOptions{
					SourceAccount: &txnbuild.SimpleAccount{AccountID: "GA47G3ZQBUR5NF2ZECG774O3QGKFMAW75XLXSCDICFDDV5GKGRFGFSOI"},
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

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4"})
	req := `{
	"transaction": "` + txEnc + `"
}`
	r := httptest.NewRequest("POST", "/GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4/sign", strings.NewReader(req))
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}/sign", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{"error": "The request was invalid in some way."}`
	assert.JSONEq(t, wantBody, string(body))
}

// Test that when the source account of the operation and transaction is not
// the account sign the request is calling sign on a bad request response is
// returned.
func TestAccountSign_accountAuthenticatedTxAndOpSourceAccountInvalid(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	s.Add(account.Account{
		Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4",
	})
	h := accountSignHandler{
		Logger:            supportlog.DefaultLogger,
		AccountStore:      s,
		SigningKey:        keypair.MustParseFull("SBIB72S6JMTGJRC6LMKLC5XMHZ2IOHZSZH4SASTN47LECEEJ7QEB6EYK"),
		NetworkPassphrase: network.TestNetworkPassphrase,
	}

	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: "GA47G3ZQBUR5NF2ZECG774O3QGKFMAW75XLXSCDICFDDV5GKGRFGFSOI"},
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.SetOptions{
					SourceAccount: &txnbuild.SimpleAccount{AccountID: "GA47G3ZQBUR5NF2ZECG774O3QGKFMAW75XLXSCDICFDDV5GKGRFGFSOI"},
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

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4"})
	req := `{
	"transaction": "` + txEnc + `"
}`
	r := httptest.NewRequest("POST", "/GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4/sign", strings.NewReader(req))
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}/sign", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{"error": "The request was invalid in some way."}`
	assert.JSONEq(t, wantBody, string(body))
}

// Test that when authenticated with a phone number signing is possible.
func TestAccountSign_phoneNumberOwnerAuthenticated(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	s.Add(account.Account{
		Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4",
		Identities: []account.Identity{
			{
				Role: "sender",
				AuthMethods: []account.AuthMethod{
					{Type: account.AuthMethodTypePhoneNumber, Value: "+10000000000"},
				},
			},
		},
	})
	h := accountSignHandler{
		Logger:            supportlog.DefaultLogger,
		AccountStore:      s,
		SigningKey:        keypair.MustParseFull("SBIB72S6JMTGJRC6LMKLC5XMHZ2IOHZSZH4SASTN47LECEEJ7QEB6EYK"),
		NetworkPassphrase: network.TestNetworkPassphrase,
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

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{PhoneNumber: "+10000000000"})
	req := `{
	"transaction": "` + txEnc + `"
}`
	r := httptest.NewRequest("POST", "/GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4/sign", strings.NewReader(req))
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}/sign", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{
	"signer": "GBOG4KF66M4AFRBUHOTJQJRO7BGGFCSGIICTI5BHXHKXCWV2C67QRN5H",
	"signature": "okp0ISR/hjU6ItsfXie6ArlQ3YWkBBqEAM5TJrthALdawV5DzcpuwBKi0QE/iBgoU7eY0hY3RPdxm8mXGNYfCQ==",
	"network_passphrase": "Test SDF Network ; September 2015"
}`
	assert.JSONEq(t, wantBody, string(body))
}

// Test that when authenticated with a phone number signing is possible.
func TestAccountSign_phoneNumberOtherAuthenticated(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	s.Add(account.Account{
		Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4",
		Identities: []account.Identity{
			{
				Role: "receiver",
				AuthMethods: []account.AuthMethod{
					{Type: account.AuthMethodTypePhoneNumber, Value: "+10000000000"},
				},
			},
		},
	})
	h := accountSignHandler{
		Logger:            supportlog.DefaultLogger,
		AccountStore:      s,
		SigningKey:        keypair.MustParseFull("SBIB72S6JMTGJRC6LMKLC5XMHZ2IOHZSZH4SASTN47LECEEJ7QEB6EYK"),
		NetworkPassphrase: network.TestNetworkPassphrase,
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

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{PhoneNumber: "+10000000000"})
	req := `{
	"transaction": "` + txEnc + `"
}`
	r := httptest.NewRequest("POST", "/GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4/sign", strings.NewReader(req))
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}/sign", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{
	"signer": "GBOG4KF66M4AFRBUHOTJQJRO7BGGFCSGIICTI5BHXHKXCWV2C67QRN5H",
	"signature": "okp0ISR/hjU6ItsfXie6ArlQ3YWkBBqEAM5TJrthALdawV5DzcpuwBKi0QE/iBgoU7eY0hY3RPdxm8mXGNYfCQ==",
	"network_passphrase": "Test SDF Network ; September 2015"
}`
	assert.JSONEq(t, wantBody, string(body))
}

// Test that when authenticated with a email signing is possible.
func TestAccountSign_emailOwnerAuthenticated(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	s.Add(account.Account{
		Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4",
		Identities: []account.Identity{
			{
				Role: "sender",
				AuthMethods: []account.AuthMethod{
					{Type: account.AuthMethodTypeEmail, Value: "user1@example.com"},
				},
			},
		},
	})
	h := accountSignHandler{
		Logger:            supportlog.DefaultLogger,
		AccountStore:      s,
		SigningKey:        keypair.MustParseFull("SBIB72S6JMTGJRC6LMKLC5XMHZ2IOHZSZH4SASTN47LECEEJ7QEB6EYK"),
		NetworkPassphrase: network.TestNetworkPassphrase,
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

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{Email: "user1@example.com"})
	req := `{
	"transaction": "` + txEnc + `"
}`
	r := httptest.NewRequest("POST", "/GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4/sign", strings.NewReader(req))
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}/sign", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{
	"signer": "GBOG4KF66M4AFRBUHOTJQJRO7BGGFCSGIICTI5BHXHKXCWV2C67QRN5H",
	"signature": "okp0ISR/hjU6ItsfXie6ArlQ3YWkBBqEAM5TJrthALdawV5DzcpuwBKi0QE/iBgoU7eY0hY3RPdxm8mXGNYfCQ==",
	"network_passphrase": "Test SDF Network ; September 2015"
}`
	assert.JSONEq(t, wantBody, string(body))
}

// Test that when authenticated with a email signing is possible.
func TestAccountSign_emailOtherAuthenticated(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	s.Add(account.Account{
		Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4",
		Identities: []account.Identity{
			{
				Role: "receiver",
				AuthMethods: []account.AuthMethod{
					{Type: account.AuthMethodTypeEmail, Value: "user1@example.com"},
				},
			},
		},
	})
	h := accountSignHandler{
		Logger:            supportlog.DefaultLogger,
		AccountStore:      s,
		SigningKey:        keypair.MustParseFull("SBIB72S6JMTGJRC6LMKLC5XMHZ2IOHZSZH4SASTN47LECEEJ7QEB6EYK"),
		NetworkPassphrase: network.TestNetworkPassphrase,
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

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{Email: "user1@example.com"})
	req := `{
	"transaction": "` + txEnc + `"
}`
	r := httptest.NewRequest("POST", "/GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4/sign", strings.NewReader(req))
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}/sign", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{
	"signer": "GBOG4KF66M4AFRBUHOTJQJRO7BGGFCSGIICTI5BHXHKXCWV2C67QRN5H",
	"signature": "okp0ISR/hjU6ItsfXie6ArlQ3YWkBBqEAM5TJrthALdawV5DzcpuwBKi0QE/iBgoU7eY0hY3RPdxm8mXGNYfCQ==",
	"network_passphrase": "Test SDF Network ; September 2015"
}`
	assert.JSONEq(t, wantBody, string(body))
}

// Test that when the transaction cannot be parsed it errors.
func TestAccountSign_cannotParseTransaction(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	s.Add(account.Account{
		Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4",
	})
	h := accountSignHandler{
		Logger:            supportlog.DefaultLogger,
		AccountStore:      s,
		SigningKey:        keypair.MustParseFull("SBIB72S6JMTGJRC6LMKLC5XMHZ2IOHZSZH4SASTN47LECEEJ7QEB6EYK"),
		NetworkPassphrase: network.TestNetworkPassphrase,
	}

	txEnc := "AAAAADx2k+7TdIuhwctzxD5y0/w5ASkvTD68az9Nh2fWY+ShAAAAZAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAPHaT7tN0i6HBy3PEPnLT/DkBKS9MPrxrP02HZ9Zj5KEAAAAAAJiWgAAAAAAAAAA"

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4"})
	req := `{
	"transaction": "` + txEnc + `"
}`
	r := httptest.NewRequest("POST", "/GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4/sign", strings.NewReader(req))
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}/sign", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{"error": "The request was invalid in some way."}`
	assert.JSONEq(t, wantBody, string(body))
}

// Test that the sign endpoint responds with an error when given a fee bump transaction.
func TestAccountSign_rejectsFeeBumpTx(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	s.Add(account.Account{
		Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4",
	})
	h := accountSignHandler{
		Logger:            supportlog.DefaultLogger,
		AccountStore:      s,
		SigningKey:        keypair.MustParseFull("SBIB72S6JMTGJRC6LMKLC5XMHZ2IOHZSZH4SASTN47LECEEJ7QEB6EYK"),
		NetworkPassphrase: network.TestNetworkPassphrase,
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

	// Action needed in release: horizonclient-v3.1.0
	// remove manual envelope type configuration because
	// once protocol 13 is enabled txnbuild will generate
	// v1 transaction envelopes by default
	innerTxEnvelope, err := tx.TxEnvelope()
	require.NoError(t, err)
	innerTxEnvelope.V1 = &xdr.TransactionV1Envelope{
		Tx: xdr.Transaction{
			SourceAccount: innerTxEnvelope.SourceAccount(),
			Fee:           xdr.Uint32(innerTxEnvelope.Fee()),
			SeqNum:        xdr.SequenceNumber(innerTxEnvelope.SeqNum()),
			TimeBounds:    innerTxEnvelope.V0.Tx.TimeBounds,
			Memo:          innerTxEnvelope.Memo(),
			Operations:    innerTxEnvelope.Operations(),
		},
	}
	innerTxEnvelope.Type = xdr.EnvelopeTypeEnvelopeTypeTx
	innerTxEnvelope.V0 = nil
	innerTxEnvelopeB64, err := xdr.MarshalBase64(innerTxEnvelope)
	require.NoError(t, err)
	parsed, err := txnbuild.TransactionFromXDR(innerTxEnvelopeB64)
	tx, _ = parsed.Transaction()

	feeBumpTx, err := txnbuild.NewFeeBumpTransaction(
		txnbuild.FeeBumpTransactionParams{
			Inner:      tx,
			FeeAccount: "GD7CGJSJ5OBOU5KOP2UQDH3MPY75UTEY27HVV5XPSL2X6DJ2VGTOSXEU",
			BaseFee:    2 * txnbuild.MinBaseFee,
		},
	)

	feeBumpTxB64, err := feeBumpTx.Base64()
	require.NoError(t, err)
	t.Log("Tx:", feeBumpTxB64)

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4"})
	req := `{
	"transaction": "` + feeBumpTxB64 + `"
}`
	r := httptest.NewRequest("POST", "/GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4/sign", strings.NewReader(req))
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}/sign", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{"error": "The request was invalid in some way."}`
	assert.JSONEq(t, wantBody, string(body))
}

// Test that the request can be made as content-text form instead of JSON.
func TestAccountSign_validContentTypeForm(t *testing.T) {
	s := &account.DBStore{DB: dbtest.Open(t).Open()}
	s.Add(account.Account{
		Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4",
	})
	h := accountSignHandler{
		Logger:            supportlog.DefaultLogger,
		AccountStore:      s,
		SigningKey:        keypair.MustParseFull("SBIB72S6JMTGJRC6LMKLC5XMHZ2IOHZSZH4SASTN47LECEEJ7QEB6EYK"),
		NetworkPassphrase: network.TestNetworkPassphrase,
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

	ctx := context.Background()
	ctx = auth.NewContext(ctx, auth.Auth{Address: "GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4"})
	reqValues := url.Values{}
	reqValues.Set("transaction", txEnc)
	req := reqValues.Encode()
	t.Log("Request Body:", req)
	r := httptest.NewRequest("POST", "/GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4/sign", strings.NewReader(req))
	r = r.WithContext(ctx)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}/sign", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{
	"signer": "GBOG4KF66M4AFRBUHOTJQJRO7BGGFCSGIICTI5BHXHKXCWV2C67QRN5H",
	"signature": "okp0ISR/hjU6ItsfXie6ArlQ3YWkBBqEAM5TJrthALdawV5DzcpuwBKi0QE/iBgoU7eY0hY3RPdxm8mXGNYfCQ==",
	"network_passphrase": "Test SDF Network ; September 2015"
}`
	assert.JSONEq(t, wantBody, string(body))
}
