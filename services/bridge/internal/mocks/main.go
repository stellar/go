package mocks

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/stellar/go/support/http/httptest"

	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

// MockTransactionSubmitter mocks TransactionSubmitter
type MockTransactionSubmitter struct {
	mock.Mock
}

// SubmitTransaction is a mocking a method
func (ts *MockTransactionSubmitter) SubmitTransaction(paymentID *string, seed string, operation []txnbuild.Operation, memo txnbuild.Memo) (hProtocol.TransactionSuccess, error) {
	a := ts.Called(paymentID, seed, operation, memo)
	return a.Get(0).(hProtocol.TransactionSuccess), a.Error(1)
}

// SignAndSubmitRawTransaction is a mocking a method
func (ts *MockTransactionSubmitter) SignAndSubmitRawTransaction(paymentID *string, seed string, tx *xdr.Transaction) (hProtocol.TransactionSuccess, error) {
	a := ts.Called(paymentID, seed, tx)
	return a.Get(0).(hProtocol.TransactionSuccess), a.Error(1)
}

// HTTPClientInterface helps mocking http.Client in tests
type HTTPClientInterface interface {
	PostForm(url string, data url.Values) (resp *http.Response, err error)
	Get(url string) (resp *http.Response, err error)
}

// BuildHTTPResponse is used in tests
func BuildHTTPResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
	}
}

// GetResponse is used in tests
func GetResponse(testServer *httptest.Server, values url.Values) (int, []byte) {
	res, err := http.PostForm(testServer.URL, values)
	if err != nil {
		panic(err)
	}
	response, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		panic(err)
	}
	return res.StatusCode, response
}

// JSONGetResponse is used in tests
func JSONGetResponse(testServer *httptest.Server, data map[string]interface{}) (int, []byte) {
	j, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	req, err := http.NewRequest("POST", testServer.URL, bytes.NewBuffer(j))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	response, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		panic(err)
	}
	return res.StatusCode, response
}
