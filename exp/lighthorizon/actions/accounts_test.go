package actions

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/exp/lighthorizon/services"
	"github.com/stellar/go/support/render/problem"
)

func setupTest() {
	problem.RegisterHost("")
}

func TestTxByAccountMissingParamError(t *testing.T) {
	setupTest()
	recorder := httptest.NewRecorder()
	request := buildHttpRequest(
		t,
		map[string]string{},
		map[string]string{},
	)

	mockOperationService := &services.MockOperationService{}
	mockTransactionService := &services.MockTransactionService{}

	lh := services.LightHorizon{
		Operations:   mockOperationService,
		Transactions: mockTransactionService,
	}

	handler := NewTXByAccountHandler(lh)
	handler(recorder, request)

	resp := recorder.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	raw, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var problem problem.P
	err = json.Unmarshal(raw, &problem)
	assert.NoError(t, err)
	assert.Equal(t, "Bad Request", problem.Title)
	assert.Equal(t, "bad_request", problem.Type)
	assert.Equal(t, "account_id", problem.Extras["invalid_field"])
	assert.Equal(t, "The request you sent was invalid in some way.", problem.Detail)
	assert.Equal(t, "unable to find account_id in url path", problem.Extras["reason"])
}

func TestTxByAccountServerError(t *testing.T) {
	setupTest()
	recorder := httptest.NewRecorder()
	pathParams := make(map[string]string)
	pathParams["account_id"] = "G1234"
	request := buildHttpRequest(
		t,
		map[string]string{},
		pathParams,
	)

	mockOperationService := &services.MockOperationService{}
	mockTransactionService := &services.MockTransactionService{}
	mockTransactionService.On("GetTransactionsByAccount", mock.Anything, mock.Anything, mock.Anything, "G1234").Return([]common.Transaction{}, errors.New("not good"))

	lh := services.LightHorizon{
		Operations:   mockOperationService,
		Transactions: mockTransactionService,
	}

	handler := NewTXByAccountHandler(lh)
	handler(recorder, request)

	resp := recorder.Result()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	raw, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var problem problem.P
	err = json.Unmarshal(raw, &problem)
	assert.NoError(t, err)
	assert.Equal(t, "Internal Server Error", problem.Title)
	assert.Equal(t, "server_error", problem.Type)
}

func TestOpsByAccountMissingParamError(t *testing.T) {
	setupTest()
	recorder := httptest.NewRecorder()
	request := buildHttpRequest(
		t,
		map[string]string{},
		map[string]string{},
	)

	mockOperationService := &services.MockOperationService{}
	mockTransactionService := &services.MockTransactionService{}

	lh := services.LightHorizon{
		Operations:   mockOperationService,
		Transactions: mockTransactionService,
	}

	handler := NewOpsByAccountHandler(lh)
	handler(recorder, request)

	resp := recorder.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	raw, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var problem problem.P
	err = json.Unmarshal(raw, &problem)
	assert.NoError(t, err)
	assert.Equal(t, "Bad Request", problem.Title)
	assert.Equal(t, "bad_request", problem.Type)
	assert.Equal(t, "account_id", problem.Extras["invalid_field"])
	assert.Equal(t, "The request you sent was invalid in some way.", problem.Detail)
	assert.Equal(t, "unable to find account_id in url path", problem.Extras["reason"])
}

func TestOpsByAccountServerError(t *testing.T) {
	setupTest()
	recorder := httptest.NewRecorder()
	pathParams := make(map[string]string)
	pathParams["account_id"] = "G1234"
	request := buildHttpRequest(
		t,
		map[string]string{},
		pathParams,
	)

	mockOperationService := &services.MockOperationService{}
	mockTransactionService := &services.MockTransactionService{}
	mockOperationService.On("GetOperationsByAccount", mock.Anything, mock.Anything, mock.Anything, "G1234").Return([]common.Operation{}, errors.New("not good"))

	lh := services.LightHorizon{
		Operations:   mockOperationService,
		Transactions: mockTransactionService,
	}

	handler := NewOpsByAccountHandler(lh)
	handler(recorder, request)

	resp := recorder.Result()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	raw, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var problem problem.P
	err = json.Unmarshal(raw, &problem)
	assert.NoError(t, err)
	assert.Equal(t, "Internal Server Error", problem.Title)
	assert.Equal(t, "server_error", problem.Type)
}

func buildHttpRequest(
	t *testing.T,
	queryParams map[string]string,
	routeParams map[string]string,
) *http.Request {
	request, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)

	query := url.Values{}
	for key, value := range queryParams {
		query.Set(key, value)
	}
	request.URL.RawQuery = query.Encode()

	chiRouteContext := chi.NewRouteContext()
	for key, value := range routeParams {
		chiRouteContext.URLParams.Add(key, value)
	}
	ctx := context.WithValue(context.Background(), chi.RouteCtxKey, chiRouteContext)
	return request.WithContext(ctx)
}
