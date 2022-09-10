package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stellar/go/exp/lighthorizon/actions"
	"github.com/stellar/go/exp/lighthorizon/services"
	"github.com/stellar/go/support/render/problem"
)

func TestUnknownUrl(t *testing.T) {
	recorder := httptest.NewRecorder()
	request, err := http.NewRequest("GET", "/unknown", nil)
	require.NoError(t, err)

	prepareTestHttpHandler().ServeHTTP(recorder, request)

	resp := recorder.Result()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	raw, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	var problem problem.P
	err = json.Unmarshal(raw, &problem)
	assert.NoError(t, err)
	assert.Equal(t, "Resource Missing", problem.Title)
	assert.Equal(t, "not_found", problem.Type)
}

func TestRootResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	request, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)

	prepareTestHttpHandler().ServeHTTP(recorder, request)

	var root actions.RootResponse
	raw, err := io.ReadAll(recorder.Result().Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &root))
	require.Equal(t, HorizonLiteVersion, root.Version)
}

func prepareTestHttpHandler() http.Handler {
	mockOperationService := &services.MockOperationService{}
	mockTransactionService := &services.MockTransactionService{}
	registry := prometheus.NewRegistry()

	lh := services.LightHorizon{
		Operations:   mockOperationService,
		Transactions: mockTransactionService,
	}

	return lightHorizonHTTPHandler(registry, lh)
}
