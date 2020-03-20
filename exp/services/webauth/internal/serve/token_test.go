package serve

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/exp/support/jwtkey"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	supportlog "github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToken_formInputSuccess(t *testing.T) {
	serverKey := keypair.MustRandom()
	t.Logf("Server signing key: %s", serverKey.Address())

	jwtPrivateKey, err := jwtkey.GenerateKey()
	require.NoError(t, err)

	account := keypair.MustRandom()
	t.Logf("Client account: %s", account.Address())

	chTx, err := txnbuild.BuildChallengeTx(
		serverKey.Seed(),
		account.Address(),
		"testserver",
		network.TestNetworkPassphrase,
		time.Minute,
	)
	require.NoError(t, err)
	t.Logf("Tx: %s", chTx)

	var tx xdr.TransactionEnvelope
	err = xdr.SafeUnmarshalBase64(chTx, &tx)
	require.NoError(t, err)
	hash, err := network.HashTransaction(&tx.V1.Tx, network.TestNetworkPassphrase)
	require.NoError(t, err)
	sigDec, err := account.SignDecorated(hash[:])
	require.NoError(t, err)
	tx.V1.Signatures = append(tx.V1.Signatures, sigDec)
	txSigned, err := xdr.MarshalBase64(tx)
	require.NoError(t, err)
	t.Logf("Signed: %s", txSigned)

	horizonClient := &horizonclient.MockClient{}
	horizonClient.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: account.Address()}).
		Return(
			horizon.Account{
				Thresholds: horizon.AccountThresholds{
					LowThreshold:  1,
					MedThreshold:  10,
					HighThreshold: 100,
				},
				Signers: []horizon.Signer{
					{
						Key:    account.Address(),
						Weight: 100,
					},
				}},
			nil,
		)

	h := tokenHandler{
		Logger:            supportlog.DefaultLogger,
		HorizonClient:     horizonClient,
		NetworkPassphrase: network.TestNetworkPassphrase,
		SigningAddress:    serverKey.FromAddress(),
		JWTPrivateKey:     jwtPrivateKey,
		JWTIssuer:         "https://example.com",
		JWTExpiresIn:      time.Minute,
	}

	body := url.Values{}
	body.Set("transaction", txSigned)
	r := httptest.NewRequest("POST", "/", strings.NewReader(body.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	res := struct {
		Token string `json:"token"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&res)
	require.NoError(t, err)

	t.Logf("JWT: %s", res.Token)

	token, err := jwt.Parse(res.Token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return &jwtPrivateKey.PublicKey, nil
	})
	require.NoError(t, err)

	claims := token.Claims.(jwt.MapClaims)
	assert.Equal(t, "https://example.com", claims["iss"])
	assert.Equal(t, account.Address(), claims["sub"])
	assert.Equal(t, account.Address(), claims["sub"])
	iat := time.Unix(int64(claims["iat"].(float64)), 0)
	exp := time.Unix(int64(claims["exp"].(float64)), 0)
	assert.True(t, iat.Before(time.Now()))
	assert.True(t, exp.After(time.Now()))
	assert.True(t, time.Now().Add(time.Minute).After(exp))
	assert.Equal(t, exp.Sub(iat), time.Minute)
}

func TestToken_jsonInputSuccess(t *testing.T) {
	serverKey := keypair.MustRandom()
	t.Logf("Server signing key: %s", serverKey.Address())

	jwtPrivateKey, err := jwtkey.GenerateKey()
	require.NoError(t, err)

	account := keypair.MustRandom()
	t.Logf("Client account: %s", account.Address())

	chTx, err := txnbuild.BuildChallengeTx(
		serverKey.Seed(),
		account.Address(),
		"testserver",
		network.TestNetworkPassphrase,
		time.Minute,
	)
	require.NoError(t, err)
	t.Logf("Tx: %s", chTx)

	var tx xdr.TransactionEnvelope
	err = xdr.SafeUnmarshalBase64(chTx, &tx)
	require.NoError(t, err)
	hash, err := network.HashTransaction(&tx.V1.Tx, network.TestNetworkPassphrase)
	require.NoError(t, err)
	sigDec, err := account.SignDecorated(hash[:])
	require.NoError(t, err)
	tx.V1.Signatures = append(tx.V1.Signatures, sigDec)
	txSigned, err := xdr.MarshalBase64(tx)
	require.NoError(t, err)
	t.Logf("Signed: %s", txSigned)

	horizonClient := &horizonclient.MockClient{}
	horizonClient.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: account.Address()}).
		Return(
			horizon.Account{
				Thresholds: horizon.AccountThresholds{
					LowThreshold:  1,
					MedThreshold:  10,
					HighThreshold: 100,
				},
				Signers: []horizon.Signer{
					{
						Key:    account.Address(),
						Weight: 100,
					},
				}},
			nil,
		)

	h := tokenHandler{
		Logger:            supportlog.DefaultLogger,
		HorizonClient:     horizonClient,
		NetworkPassphrase: network.TestNetworkPassphrase,
		SigningAddress:    serverKey.FromAddress(),
		JWTPrivateKey:     jwtPrivateKey,
		JWTIssuer:         "https://example.com",
		JWTExpiresIn:      time.Minute,
	}

	body := struct {
		Transaction string `json:"transaction"`
	}{
		Transaction: txSigned,
	}
	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)
	r := httptest.NewRequest("POST", "/", bytes.NewReader(bodyBytes))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	res := struct {
		Token string `json:"token"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&res)
	require.NoError(t, err)

	t.Logf("JWT: %s", res.Token)

	token, err := jwt.Parse(res.Token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return &jwtPrivateKey.PublicKey, nil
	})
	require.NoError(t, err)

	claims := token.Claims.(jwt.MapClaims)
	assert.Equal(t, "https://example.com", claims["iss"])
	assert.Equal(t, account.Address(), claims["sub"])
	assert.Equal(t, account.Address(), claims["sub"])
	iat := time.Unix(int64(claims["iat"].(float64)), 0)
	exp := time.Unix(int64(claims["exp"].(float64)), 0)
	assert.True(t, iat.Before(time.Now()))
	assert.True(t, exp.After(time.Now()))
	assert.True(t, time.Now().Add(time.Minute).After(exp))
	assert.Equal(t, exp.Sub(iat), time.Minute)
}

func TestToken_jsonInputValidMultipleSigners(t *testing.T) {
	serverKey := keypair.MustRandom()
	t.Logf("Server signing key: %s", serverKey.Address())

	jwtPrivateKey, err := jwtkey.GenerateKey()
	require.NoError(t, err)

	account := keypair.MustRandom()
	t.Logf("Client account: %s", account.Address())

	accountSigner1 := keypair.MustRandom()
	t.Logf("Client account signer 1: %s", accountSigner1.Address())

	accountSigner2 := keypair.MustRandom()
	t.Logf("Client account signer 2: %s", accountSigner2.Address())

	chTx, err := txnbuild.BuildChallengeTx(
		serverKey.Seed(),
		account.Address(),
		"testserver",
		network.TestNetworkPassphrase,
		time.Minute,
	)
	require.NoError(t, err)
	t.Logf("Tx: %s", chTx)

	var tx xdr.TransactionEnvelope
	err = xdr.SafeUnmarshalBase64(chTx, &tx)
	require.NoError(t, err)
	hash, err := network.HashTransaction(&tx.V1.Tx, network.TestNetworkPassphrase)
	require.NoError(t, err)
	sigDec1, err := accountSigner1.SignDecorated(hash[:])
	require.NoError(t, err)
	tx.V1.Signatures = append(tx.V1.Signatures, sigDec1)
	sigDec2, err := accountSigner2.SignDecorated(hash[:])
	require.NoError(t, err)
	tx.V1.Signatures = append(tx.V1.Signatures, sigDec2)
	txSigned, err := xdr.MarshalBase64(tx)
	require.NoError(t, err)
	t.Logf("Signed: %s", txSigned)

	horizonClient := &horizonclient.MockClient{}
	horizonClient.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: account.Address()}).
		Return(
			horizon.Account{
				Thresholds: horizon.AccountThresholds{
					LowThreshold:  1,
					MedThreshold:  10,
					HighThreshold: 100,
				},
				Signers: []horizon.Signer{
					{
						Key:    accountSigner1.Address(),
						Weight: 40,
					},
					{
						Key:    accountSigner2.Address(),
						Weight: 60,
					},
				}},
			nil,
		)

	h := tokenHandler{
		Logger:            supportlog.DefaultLogger,
		HorizonClient:     horizonClient,
		NetworkPassphrase: network.TestNetworkPassphrase,
		SigningAddress:    serverKey.FromAddress(),
		JWTPrivateKey:     jwtPrivateKey,
		JWTIssuer:         "https://example.com",
		JWTExpiresIn:      time.Minute,
	}

	body := struct {
		Transaction string `json:"transaction"`
	}{
		Transaction: txSigned,
	}
	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)
	r := httptest.NewRequest("POST", "/", bytes.NewReader(bodyBytes))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	res := struct {
		Token string `json:"token"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&res)
	require.NoError(t, err)

	t.Logf("JWT: %s", res.Token)

	token, err := jwt.Parse(res.Token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return &jwtPrivateKey.PublicKey, nil
	})
	require.NoError(t, err)

	claims := token.Claims.(jwt.MapClaims)
	assert.Equal(t, "https://example.com", claims["iss"])
	assert.Equal(t, account.Address(), claims["sub"])
	iat := time.Unix(int64(claims["iat"].(float64)), 0)
	exp := time.Unix(int64(claims["exp"].(float64)), 0)
	assert.True(t, iat.Before(time.Now()))
	assert.True(t, exp.After(time.Now()))
	assert.True(t, time.Now().Add(time.Minute).After(exp))
	assert.Equal(t, exp.Sub(iat), time.Minute)
}

func TestToken_jsonInputNotEnoughWeight(t *testing.T) {
	serverKey := keypair.MustRandom()
	t.Logf("Server signing key: %s", serverKey.Address())

	jwtPrivateKey, err := jwtkey.GenerateKey()
	require.NoError(t, err)

	account := keypair.MustRandom()
	t.Logf("Client account: %s", account.Address())

	chTx, err := txnbuild.BuildChallengeTx(
		serverKey.Seed(),
		account.Address(),
		"testserver",
		network.TestNetworkPassphrase,
		time.Minute,
	)
	require.NoError(t, err)
	t.Logf("Tx: %s", chTx)

	var tx xdr.TransactionEnvelope
	err = xdr.SafeUnmarshalBase64(chTx, &tx)
	require.NoError(t, err)
	hash, err := network.HashTransaction(&tx.V1.Tx, network.TestNetworkPassphrase)
	require.NoError(t, err)
	sigDec, err := account.SignDecorated(hash[:])
	require.NoError(t, err)
	tx.V1.Signatures = append(tx.V1.Signatures, sigDec)
	txSigned, err := xdr.MarshalBase64(tx)
	require.NoError(t, err)
	t.Logf("Signed: %s", txSigned)

	horizonClient := &horizonclient.MockClient{}
	horizonClient.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: account.Address()}).
		Return(
			horizon.Account{
				Thresholds: horizon.AccountThresholds{
					LowThreshold:  1,
					MedThreshold:  10,
					HighThreshold: 100,
				},
				Signers: []horizon.Signer{
					{
						Key:    account.Address(),
						Weight: 10,
					},
				}},
			nil,
		)

	h := tokenHandler{
		Logger:            supportlog.DefaultLogger,
		HorizonClient:     horizonClient,
		NetworkPassphrase: network.TestNetworkPassphrase,
		SigningAddress:    serverKey.FromAddress(),
		JWTPrivateKey:     jwtPrivateKey,
		JWTIssuer:         "https://example.com",
		JWTExpiresIn:      time.Minute,
	}

	body := struct {
		Transaction string `json:"transaction"`
	}{
		Transaction: txSigned,
	}
	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)
	r := httptest.NewRequest("POST", "/", bytes.NewReader(bodyBytes))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	respBodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.JSONEq(t, `{"error":"The request could not be authenticated."}`, string(respBodyBytes))
}

func TestToken_jsonInputUnrecognizedSigner(t *testing.T) {
	serverKey := keypair.MustRandom()
	t.Logf("Server signing key: %s", serverKey.Address())

	jwtPrivateKey, err := jwtkey.GenerateKey()
	require.NoError(t, err)

	account := keypair.MustRandom()
	t.Logf("Client account: %s", account.Address())

	chTx, err := txnbuild.BuildChallengeTx(
		serverKey.Seed(),
		account.Address(),
		"testserver",
		network.TestNetworkPassphrase,
		time.Minute,
	)
	require.NoError(t, err)
	t.Logf("Tx: %s", chTx)

	var tx xdr.TransactionEnvelope
	err = xdr.SafeUnmarshalBase64(chTx, &tx)
	require.NoError(t, err)
	hash, err := network.HashTransaction(&tx.V1.Tx, network.TestNetworkPassphrase)
	require.NoError(t, err)
	sigDec, err := account.SignDecorated(hash[:])
	require.NoError(t, err)
	tx.V1.Signatures = append(tx.V1.Signatures, sigDec)
	txSigned, err := xdr.MarshalBase64(tx)
	require.NoError(t, err)
	t.Logf("Signed: %s", txSigned)

	horizonClient := &horizonclient.MockClient{}
	horizonClient.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: account.Address()}).
		Return(
			horizon.Account{
				Thresholds: horizon.AccountThresholds{
					LowThreshold:  1,
					MedThreshold:  10,
					HighThreshold: 100,
				},
				Signers: []horizon.Signer{
					{
						Key:    keypair.MustRandom().Address(),
						Weight: 100,
					},
				}},
			nil,
		)

	h := tokenHandler{
		Logger:            supportlog.DefaultLogger,
		HorizonClient:     horizonClient,
		NetworkPassphrase: network.TestNetworkPassphrase,
		SigningAddress:    serverKey.FromAddress(),
		JWTPrivateKey:     jwtPrivateKey,
		JWTIssuer:         "https://example.com",
		JWTExpiresIn:      time.Minute,
	}

	body := struct {
		Transaction string `json:"transaction"`
	}{
		Transaction: txSigned,
	}
	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)
	r := httptest.NewRequest("POST", "/", bytes.NewReader(bodyBytes))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	respBodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.JSONEq(t, `{"error":"The request could not be authenticated."}`, string(respBodyBytes))
}

func TestToken_jsonInputAccountNotExistSuccess(t *testing.T) {
	serverKey := keypair.MustRandom()
	t.Logf("Server signing key: %s", serverKey.Address())

	jwtPrivateKey, err := jwtkey.GenerateKey()
	require.NoError(t, err)

	account := keypair.MustRandom()
	t.Logf("Client account: %s", account.Address())

	chTx, err := txnbuild.BuildChallengeTx(
		serverKey.Seed(),
		account.Address(),
		"testserver",
		network.TestNetworkPassphrase,
		time.Minute,
	)
	require.NoError(t, err)
	t.Logf("Tx: %s", chTx)

	var tx xdr.TransactionEnvelope
	err = xdr.SafeUnmarshalBase64(chTx, &tx)
	require.NoError(t, err)
	hash, err := network.HashTransaction(&tx.V1.Tx, network.TestNetworkPassphrase)
	require.NoError(t, err)
	sigDec, err := account.SignDecorated(hash[:])
	require.NoError(t, err)
	tx.V1.Signatures = append(tx.V1.Signatures, sigDec)
	txSigned, err := xdr.MarshalBase64(tx)
	require.NoError(t, err)
	t.Logf("Signed: %s", txSigned)

	horizonClient := &horizonclient.MockClient{}
	horizonClient.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: account.Address()}).
		Return(
			horizon.Account{},
			&horizonclient.Error{
				Problem: problem.P{
					Type:   "https://stellar.org/horizon-errors/not_found",
					Title:  "Resource Missing",
					Status: 404,
				},
			},
		)

	h := tokenHandler{
		Logger:                      supportlog.DefaultLogger,
		HorizonClient:               horizonClient,
		NetworkPassphrase:           network.TestNetworkPassphrase,
		SigningAddress:              serverKey.FromAddress(),
		JWTPrivateKey:               jwtPrivateKey,
		JWTIssuer:                   "https://example.com",
		JWTExpiresIn:                time.Minute,
		AllowAccountsThatDoNotExist: true,
	}

	body := struct {
		Transaction string `json:"transaction"`
	}{
		Transaction: txSigned,
	}
	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)
	r := httptest.NewRequest("POST", "/", bytes.NewReader(bodyBytes))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	res := struct {
		Token string `json:"token"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&res)
	require.NoError(t, err)

	t.Logf("JWT: %s", res.Token)

	token, err := jwt.Parse(res.Token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return &jwtPrivateKey.PublicKey, nil
	})
	require.NoError(t, err)

	claims := token.Claims.(jwt.MapClaims)
	assert.Equal(t, "https://example.com", claims["iss"])
	assert.Equal(t, account.Address(), claims["sub"])
	assert.Equal(t, account.Address(), claims["sub"])
	iat := time.Unix(int64(claims["iat"].(float64)), 0)
	exp := time.Unix(int64(claims["exp"].(float64)), 0)
	assert.True(t, iat.Before(time.Now()))
	assert.True(t, exp.After(time.Now()))
	assert.True(t, time.Now().Add(time.Minute).After(exp))
	assert.Equal(t, exp.Sub(iat), time.Minute)
}

func TestToken_jsonInputAccountNotExistFail(t *testing.T) {
	serverKey := keypair.MustRandom()
	t.Logf("Server signing key: %s", serverKey.Address())

	jwtPrivateKey, err := jwtkey.GenerateKey()
	require.NoError(t, err)

	account := keypair.MustRandom()
	t.Logf("Client account: %s", account.Address())

	otherSigner := keypair.MustRandom()
	t.Logf("Other signer: %s", otherSigner.Address())

	chTx, err := txnbuild.BuildChallengeTx(
		serverKey.Seed(),
		account.Address(),
		"testserver",
		network.TestNetworkPassphrase,
		time.Minute,
	)
	require.NoError(t, err)
	t.Logf("Tx: %s", chTx)

	var tx xdr.TransactionEnvelope
	err = xdr.SafeUnmarshalBase64(chTx, &tx)
	require.NoError(t, err)
	hash, err := network.HashTransaction(&tx.V1.Tx, network.TestNetworkPassphrase)
	require.NoError(t, err)
	sigDec, err := otherSigner.SignDecorated(hash[:])
	require.NoError(t, err)
	tx.V1.Signatures = append(tx.V1.Signatures, sigDec)
	txSigned, err := xdr.MarshalBase64(tx)
	require.NoError(t, err)
	t.Logf("Signed: %s", txSigned)

	horizonClient := &horizonclient.MockClient{}
	horizonClient.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: account.Address()}).
		Return(
			horizon.Account{},
			&horizonclient.Error{
				Problem: problem.P{
					Type:   "https://stellar.org/horizon-errors/not_found",
					Title:  "Resource Missing",
					Status: 404,
				},
			},
		)

	h := tokenHandler{
		Logger:                      supportlog.DefaultLogger,
		HorizonClient:               horizonClient,
		NetworkPassphrase:           network.TestNetworkPassphrase,
		SigningAddress:              serverKey.FromAddress(),
		JWTPrivateKey:               jwtPrivateKey,
		JWTIssuer:                   "https://example.com",
		JWTExpiresIn:                time.Minute,
		AllowAccountsThatDoNotExist: true,
	}

	body := struct {
		Transaction string `json:"transaction"`
	}{
		Transaction: txSigned,
	}
	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)
	r := httptest.NewRequest("POST", "/", bytes.NewReader(bodyBytes))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	respBodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.JSONEq(t, `{"error":"The request could not be authenticated."}`, string(respBodyBytes))
}

func TestToken_jsonInputAccountNotExistNotAllowed(t *testing.T) {
	serverKey := keypair.MustRandom()
	t.Logf("Server signing key: %s", serverKey.Address())

	jwtPrivateKey, err := jwtkey.GenerateKey()
	require.NoError(t, err)

	account := keypair.MustRandom()
	t.Logf("Client account: %s", account.Address())

	chTx, err := txnbuild.BuildChallengeTx(
		serverKey.Seed(),
		account.Address(),
		"testserver",
		network.TestNetworkPassphrase,
		time.Minute,
	)
	require.NoError(t, err)
	t.Logf("Tx: %s", chTx)

	var tx xdr.TransactionEnvelope
	err = xdr.SafeUnmarshalBase64(chTx, &tx)
	require.NoError(t, err)
	hash, err := network.HashTransaction(&tx.V1.Tx, network.TestNetworkPassphrase)
	require.NoError(t, err)
	sigDec, err := account.SignDecorated(hash[:])
	require.NoError(t, err)
	tx.V1.Signatures = append(tx.V1.Signatures, sigDec)
	txSigned, err := xdr.MarshalBase64(tx)
	require.NoError(t, err)
	t.Logf("Signed: %s", txSigned)

	horizonClient := &horizonclient.MockClient{}
	horizonClient.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: account.Address()}).
		Return(
			horizon.Account{},
			&horizonclient.Error{
				Problem: problem.P{
					Type:   "https://stellar.org/horizon-errors/not_found",
					Title:  "Resource Missing",
					Status: 404,
				},
			},
		)

	h := tokenHandler{
		Logger:                      supportlog.DefaultLogger,
		HorizonClient:               horizonClient,
		NetworkPassphrase:           network.TestNetworkPassphrase,
		SigningAddress:              serverKey.FromAddress(),
		JWTPrivateKey:               jwtPrivateKey,
		JWTIssuer:                   "https://example.com",
		JWTExpiresIn:                time.Minute,
		AllowAccountsThatDoNotExist: false,
	}

	body := struct {
		Transaction string `json:"transaction"`
	}{
		Transaction: txSigned,
	}
	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)
	r := httptest.NewRequest("POST", "/", bytes.NewReader(bodyBytes))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	respBodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.JSONEq(t, `{"error":"The request could not be authenticated."}`, string(respBodyBytes))
}
