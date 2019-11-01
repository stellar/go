package resourceadapter

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/services/horizon/internal/ledger"
)

func TestPopulateRoot(t *testing.T) {
	res := &horizon.Root{}
	PopulateRoot(context.Background(),
		res,
		ledger.State{CoreLatest: 1, HistoryLatest: 3, HistoryElder: 2},
		"hVersion",
		"cVersion",
		"passphrase",
		100,
		101,
		urlMustParse(t, "https://friendbot.example.com"),
		false,
	)

	assert.Equal(t, int32(1), res.CoreSequence)
	assert.Equal(t, int32(2), res.HistoryElderSequence)
	assert.Equal(t, int32(3), res.HorizonSequence)
	assert.Equal(t, "hVersion", res.HorizonVersion)
	assert.Equal(t, "cVersion", res.StellarCoreVersion)
	assert.Equal(t, "passphrase", res.NetworkPassphrase)
	assert.Equal(t, "https://friendbot.example.com/{?addr}", res.Links.Friendbot.Href)
	assert.Empty(t, res.Links.Accounts)
	assert.Empty(t, res.Links.Offer)
	assert.Empty(t, res.Links.Offers)

	// Without testbot
	res = &horizon.Root{}
	PopulateRoot(context.Background(),
		res,
		ledger.State{CoreLatest: 1, HistoryLatest: 3, HistoryElder: 2},
		"hVersion",
		"cVersion",
		"passphrase",
		100,
		101,
		nil,
		false,
	)

	assert.Equal(t, int32(1), res.CoreSequence)
	assert.Equal(t, int32(2), res.HistoryElderSequence)
	assert.Equal(t, int32(3), res.HorizonSequence)
	assert.Equal(t, "hVersion", res.HorizonVersion)
	assert.Equal(t, "cVersion", res.StellarCoreVersion)
	assert.Equal(t, "passphrase", res.NetworkPassphrase)
	assert.Empty(t, res.Links.Friendbot)

	// With experimental ingestion
	res = &horizon.Root{}
	PopulateRoot(context.Background(),
		res,
		ledger.State{CoreLatest: 1, HistoryLatest: 3, HistoryElder: 2},
		"hVersion",
		"cVersion",
		"passphrase",
		100,
		101,
		urlMustParse(t, "https://friendbot.example.com"),
		true,
	)

	assert.Equal(t, "/accounts?{signer}", res.Links.Accounts.Href)
	assert.Equal(t, "/offers/{offer_id}", res.Links.Offer.Href)
	assert.Equal(t, "/offers{?seller,selling_asset_type,selling_asset_code,selling_asset_issuer,buying_asset_type,buying_asset_code,buying_asset_issuer,cursor,limit,order}", res.Links.Offers.Href)
}

func urlMustParse(t *testing.T, s string) *url.URL {
	if u, err := url.Parse(s); err != nil {
		t.Fatalf("Unable to parse URL: %s/%v", s, err)
		return nil
	} else {
		return u
	}
}
