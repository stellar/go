package apiclient

import (
	"net/url"
	"testing"

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
