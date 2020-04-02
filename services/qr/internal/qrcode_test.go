package internal

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	supportlog "github.com/stellar/go/support/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQRCodeHandler(t *testing.T) {
	h := qrCodeHandler{
		Logger: supportlog.DefaultLogger,
	}

	r := httptest.NewRequest("POST", "/GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4.svg", nil)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}.svg", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "image/svg+xml", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody, err := ioutil.ReadFile("testdata/GA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4.svg")
	require.NoError(t, err)
	assert.Equal(t, wantBody, body)
}

func TestQRCodeHandler_invalidAddress(t *testing.T) {
	h := qrCodeHandler{
		Logger: supportlog.DefaultLogger,
	}

	r := httptest.NewRequest("POST", "/XA6HNE7O2N2IXIOBZNZ4IPTS2P6DSAJJF5GD5PDLH5GYOZ6WMPSKCXD4.svg", nil)

	w := httptest.NewRecorder()
	m := chi.NewMux()
	m.Post("/{address}.svg", h.ServeHTTP)
	m.ServeHTTP(w, r)
	resp := w.Result()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	wantBody := `{
	"error": "The request was invalid in some way."
}`
	assert.JSONEq(t, wantBody, string(body))
}
