package mocks

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

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
