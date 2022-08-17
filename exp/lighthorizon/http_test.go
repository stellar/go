package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stellar/go/exp/lighthorizon/services"
	"github.com/stellar/go/support/render/problem"
)

func TestUnknownUrl(t *testing.T) {
	recorder := httptest.NewRecorder()
	request, err := http.NewRequest("GET", "/unknown", nil)
	require.NoError(t, err)

	mockOperationService := &services.MockOperationService{}
	mockTransactionService := &services.MockTransactionService{}
	registry := prometheus.NewRegistry()

	lh := services.LightHorizon{
		Operations:   mockOperationService,
		Transactions: mockTransactionService,
	}

	handler := lightHorizonHTTPHandler(registry, lh)
	handler.ServeHTTP(recorder, request)

	resp := recorder.Result()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	raw, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var problem problem.P
	err = json.Unmarshal(raw, &problem)
	assert.NoError(t, err)
	assert.Equal(t, "Resource Missing", problem.Title)
	assert.Equal(t, "not_found", problem.Type)
}
