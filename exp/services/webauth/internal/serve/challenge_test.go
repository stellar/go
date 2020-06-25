package serve

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	supportlog "github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChallenge(t *testing.T) {
	serverKey := keypair.MustRandom()
	account := keypair.MustRandom()

	h := challengeHandler{
		Logger:             supportlog.DefaultLogger,
		ServerName:         "testserver",
		NetworkPassphrase:  network.TestNetworkPassphrase,
		SigningKey:         serverKey,
		ChallengeExpiresIn: time.Minute,
	}

	r := httptest.NewRequest("GET", "/?account="+account.Address(), nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	res := struct {
		Transaction       string `json:"transaction"`
		NetworkPassphrase string `json:"network_passphrase"`
	}{}
	err := json.NewDecoder(resp.Body).Decode(&res)
	require.NoError(t, err)

	var tx xdr.TransactionEnvelope
	err = xdr.SafeUnmarshalBase64(res.Transaction, &tx)
	require.NoError(t, err)

	assert.Len(t, tx.Signatures(), 1)
	sourceAccount := tx.SourceAccount().ToAccountId()
	assert.Equal(t, serverKey.Address(), sourceAccount.Address())
	assert.Equal(t, tx.SeqNum(), int64(0))
	assert.Equal(t, time.Unix(int64(tx.TimeBounds().MaxTime), 0).Sub(time.Unix(int64(tx.TimeBounds().MinTime), 0)), time.Minute)
	assert.Len(t, tx.Operations(), 1)
	opSourceAccount := tx.Operations()[0].SourceAccount.ToAccountId()
	assert.Equal(t, account.Address(), opSourceAccount.Address())
	assert.Equal(t, xdr.OperationTypeManageData, tx.Operations()[0].Body.Type)
	assert.Regexp(t, "^testserver auth", tx.Operations()[0].Body.ManageDataOp.DataName)

	hash, err := network.HashTransactionInEnvelope(tx, res.NetworkPassphrase)
	require.NoError(t, err)
	assert.NoError(t, serverKey.FromAddress().Verify(hash[:], tx.V0.Signatures[0].Signature))

	assert.Equal(t, network.TestNetworkPassphrase, res.NetworkPassphrase)
}

func TestChallengeNoAccount(t *testing.T) {
	h := challengeHandler{}

	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"error":"The request was invalid in some way."}`, string(body))
}

func TestChallengeInvalidAccount(t *testing.T) {
	h := challengeHandler{}

	r := httptest.NewRequest("GET", "/?account=GREATACCOUNT", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"error":"The request was invalid in some way."}`, string(body))
}
