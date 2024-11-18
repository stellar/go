package apiclient

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stellar/go/support/http/httptest"
	"github.com/stretchr/testify/assert"
)

func Test_GetURL(t *testing.T) {
	c := &APIClient{
		BaseURL: "https://stellar.org",
	}

	qstr := url.Values{}
	qstr.Add("type", "forward")
	qstr.Add("federation_type", "bank_account")
	qstr.Add("swift", "BOPBPHMM")
	qstr.Add("acct", "2382376")
	furl := c.GetURL("federation", qstr)
	assert.Equal(t, "https://stellar.org/federation?acct=2382376&federation_type=bank_account&swift=BOPBPHMM&type=forward", furl)
}

func Test_CallAPI(t *testing.T) {
	testCases := []struct {
		name          string
		mockResponses []httptest.ResponseData
		expected      interface{}
		expectedError string
		retries       bool
	}{
		{
			name: "status 200 - Success",
			mockResponses: []httptest.ResponseData{
				{Status: http.StatusOK, Body: `{"data": "Okay Response"}`, Header: nil},
			},
			expected:      map[string]interface{}{"data": "Okay Response"},
			expectedError: "",
			retries:       false,
		},
		{
			name: "success with retries - status 429 and 503 then 200",
			mockResponses: []httptest.ResponseData{
				{Status: http.StatusTooManyRequests, Body: `{"data": "First Response"}`, Header: nil},
				{Status: http.StatusServiceUnavailable, Body: `{"data": "Second Response"}`, Header: nil},
				{Status: http.StatusOK, Body: `{"data": "Third Response"}`, Header: nil},
				{Status: http.StatusOK, Body: `{"data": "Fourth Response"}`, Header: nil},
			},
			expected:      map[string]interface{}{"data": "Third Response"},
			expectedError: "",
			retries:       true,
		},
		{
			name: "failure - status 500",
			mockResponses: []httptest.ResponseData{
				{Status: http.StatusInternalServerError, Body: `{"error": "Internal Server Error"}`, Header: nil},
			},
			expected:      nil,
			expectedError: "API request failed with status 500",
			retries:       false,
		},
		{
			name: "failure - status 401",
			mockResponses: []httptest.ResponseData{
				{Status: http.StatusUnauthorized, Body: `{"error": "Bad authorization"}`, Header: nil},
			},
			expected:      nil,
			expectedError: "API request failed with status 401",
			retries:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hmock := httptest.NewClient()
			hmock.On("GET", "https://stellar.org/federation?acct=2382376").
				ReturnMultipleResults(tc.mockResponses)

			c := &APIClient{
				BaseURL: "https://stellar.org",
				HTTP:    hmock,
			}

			qstr := url.Values{}
			qstr.Add("acct", "2382376")

			reqParams := RequestParams{
				RequestType: "GET",
				Endpoint:    "federation",
				QueryParams: qstr,
			}

			result, err := c.CallAPI(reqParams)

			if tc.expectedError != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tc.expectedError)
				}
				if err.Error() != tc.expectedError {
					t.Fatalf("expected error %q, got %q", tc.expectedError, err.Error())
				}
			} else if err != nil {
				t.Fatal(err)
			}

			if tc.expected != nil {
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}
