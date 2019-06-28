package mocks

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/stellar/go/support/http/httptest"
)

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
	if err != nil {
		panic(err)
	}
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

// PredefinedTime is a time.Time object that will be returned by Now() function
var PredefinedTime time.Time

// Now is a mocking a method
func Now() time.Time {
	return PredefinedTime
}

type Operation interface {
	PagingToken() string
	GetType() string
	GetID() string
	GetTransactionHash() string
	IsTransactionSuccessful() bool
}

type MockOperationResponse struct {
	PT                    string
	Type                  string
	ID                    string
	TransactionHash       string
	TransactionSuccessful bool
}

func (m MockOperationResponse) PagingToken() string {
	return m.PT
}

func (m MockOperationResponse) GetType() string {
	return m.Type
}

func (m MockOperationResponse) GetID() string {
	return m.ID
}

func (m MockOperationResponse) GetTransactionHash() string {
	return m.TransactionHash
}

func (m MockOperationResponse) IsTransactionSuccessful() bool {
	return m.TransactionSuccessful
}
