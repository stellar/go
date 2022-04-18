package horizonclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClaimableBalanceBuildUrl(t *testing.T) {

	cbr := ClaimableBalanceRequest{
		ID: "1235",
	}
	url, err := cbr.BuildURL()
	assert.Equal(t, "claimable_balances/1235", url)
	assert.NoError(t, err)

	//if the ID is included, you cannot include another parameter
	cbr = ClaimableBalanceRequest{
		ID:       "1235",
		Claimant: "CLAIMANTADDRESS",
	}
	_, err = cbr.BuildURL()
	assert.EqualError(t, err, "invalid request: too many parameters")

	//if you have two parameters, and neither of them are ID, it must use both in the URL
	cbr = ClaimableBalanceRequest{
		Claimant: "CLAIMANTADDRESS",
		Asset:    "TEST:ISSUERADDRESS",
	}
	url, err = cbr.BuildURL()
	assert.NoError(t, err)
	assert.Equal(t, "claimable_balances?asset=TEST%3AISSUERADDRESS&claimant=CLAIMANTADDRESS", url)

	//check limit
	cbr = ClaimableBalanceRequest{
		Claimant: "CLAIMANTADDRESS",
		Asset:    "TEST:ISSUERADDRESS",
		Limit:    200,
	}
	url, err = cbr.BuildURL()
	assert.NoError(t, err)
	assert.Equal(t, "claimable_balances?asset=TEST%3AISSUERADDRESS&claimant=CLAIMANTADDRESS&limit=200", url)

	cbr = ClaimableBalanceRequest{
		Claimant: "CLAIMANTADDRESS",
		Asset:    "TEST:ISSUERADDRESS",
		Limit:    201,
	}
	_, err = cbr.BuildURL()
	assert.EqualError(t, err, "invalid request: limit 201 is greater than limit max of 200")

}
