package horizonclient

import (
	"context"
	"fmt"
	"testing"
	"time"

	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/support/http/httptest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOfferRequestBuildUrl(t *testing.T) {

	er := OfferRequest{ForAccount: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU"}
	endpoint, err := er.BuildUrl()

	// It should return valid offers endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/offers", endpoint)

	er = OfferRequest{ForAccount: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU", Cursor: "now", Order: OrderDesc}
	endpoint, err = er.BuildUrl()

	// It should return valid offers endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/offers?cursor=now&order=desc", endpoint)
}

func ExampleClient_StreamOffers() {
	client := DefaultTestNetClient
	// offers for account
	offerRequest := OfferRequest{ForAccount: "GAQHWQYBBW272OOXNQMMLCA5WY2XAZPODGB7Q3S5OKKIXVESKO55ZQ7C", Cursor: "1"}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Stop streaming after 60 seconds.
		time.Sleep(60 * time.Second)
		cancel()
	}()

	printHandler := func(offer hProtocol.Offer) {
		fmt.Println(offer)
	}
	err := client.StreamOffers(ctx, offerRequest, printHandler)
	if err != nil {
		fmt.Println(err)
	}
}

func TestOfferRequestStreamOffers(t *testing.T) {

	hmock := httptest.NewClient()
	client := &Client{
		HorizonURL: "https://localhost/",
		HTTP:       hmock,
	}

	// offers for account
	orRequest := OfferRequest{ForAccount: "GAQHWQYBBW272OOXNQMMLCA5WY2XAZPODGB7Q3S5OKKIXVESKO55ZQ7C"}
	ctx, cancel := context.WithCancel(context.Background())

	hmock.On(
		"GET",
		"https://localhost/accounts/GAQHWQYBBW272OOXNQMMLCA5WY2XAZPODGB7Q3S5OKKIXVESKO55ZQ7C/offers?cursor=now",
	).ReturnString(200, offerStreamResponse)

	offers := make([]hProtocol.Offer, 1)
	err := client.StreamOffers(ctx, orRequest, func(offer hProtocol.Offer) {
		offers[0] = offer
		cancel()
	})

	if assert.NoError(t, err) {
		assert.Equal(t, offers[0].Amount, "20.4266087")
		assert.Equal(t, offers[0].Seller, "GAQHWQYBBW272OOXNQMMLCA5WY2XAZPODGB7Q3S5OKKIXVESKO55ZQ7C")
	}

	// test error
	orRequest = OfferRequest{ForAccount: "GAQHWQYBBW272OOXNQMMLCA5WY2XAZPODGB7Q3S5OKKIXVESKO55ZQ7C"}
	ctx, cancel = context.WithCancel(context.Background())

	hmock.On(
		"GET",
		"https://localhost/accounts/GAQHWQYBBW272OOXNQMMLCA5WY2XAZPODGB7Q3S5OKKIXVESKO55ZQ7C/offers?cursor=now",
	).ReturnString(500, offerStreamResponse)

	offers = make([]hProtocol.Offer, 1)
	err = client.StreamOffers(ctx, orRequest, func(offer hProtocol.Offer) {
		cancel()
	})

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Got bad HTTP status code 500")
	}
}

var offerStreamResponse = `data: {"_links":{"self":{"href":"https://horizon-testnet.stellar.org/offers/5269100"},"offer_maker":{"href":"https://horizon-testnet.stellar.org/accounts/GAQHWQYBBW272OOXNQMMLCA5WY2XAZPODGB7Q3S5OKKIXVESKO55ZQ7C"}},"id":5269100,"paging_token":"5269100","seller":"GAQHWQYBBW272OOXNQMMLCA5WY2XAZPODGB7Q3S5OKKIXVESKO55ZQ7C","selling":{"asset_type":"credit_alphanum4","asset_code":"DSQ","asset_issuer":"GBDQPTQJDATT7Z7EO4COS4IMYXH44RDLLI6N6WIL5BZABGMUOVMLWMQF"},"buying":{"asset_type":"credit_alphanum4","asset_code":"XCS6","asset_issuer":"GBH2V47NOZRC56QAYCPV5JUBG5NVFJQF5AQTUNFNWNDHSWWTKH2MWR2L"},"amount":"20.4266087","price_r":{"n":24819,"d":10000000},"price":"0.0024819","last_modified_ledger":674449,"last_modified_time":"2019-04-08T11:56:41Z"}
`
