package apiclient

import (
	"net/url"
	"testing"

	"github.com/stellar/go/support/http/httptest"
	"github.com/stretchr/testify/assert"
)

func Test_url(t *testing.T) {
	c := &APIClient{
		BaseURL: "https://stellar.org",
	}

	qstr := url.Values{}
	qstr.Add("type", "forward")
	qstr.Add("federation_type", "bank_account")
	qstr.Add("swift", "BOPBPHMM")
	qstr.Add("acct", "2382376")
	furl := c.url("federation", qstr)
	assert.Equal(t, "https://stellar.org/federation?acct=2382376&federation_type=bank_account&swift=BOPBPHMM&type=forward", furl)
}

func Test_getRequest(t *testing.T) {
	friendbotFundResponse := `{"key": "value"}`

	hmock := httptest.NewClient()
	c := &APIClient{
		BaseURL: "https://stellar.org",
		HTTP:    hmock,
	}
	hmock.On(
		"GET",
		"https://stellar.org/federation?acct=2382376&federation_type=bank_account&swift=BOPBPHMM&type=forward",
	).ReturnString(200, friendbotFundResponse)
	qstr := url.Values{}

	qstr.Add("type", "forward")
	qstr.Add("federation_type", "bank_account")
	qstr.Add("swift", "BOPBPHMM")
	qstr.Add("acct", "2382376")

	result, err := c.getRequest("federation", qstr)
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]interface{}{"key": "value"}
	assert.Equal(t, expected, result)
}
