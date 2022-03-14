package integration

import (
	"strconv"
	"testing"
	"time"

	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/ingest/filters"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/assert"
)

func TestIngestionFiltering(t *testing.T) {
	// Set up test

	tt := assert.New(t)
	const adminPort uint16 = 6000
	itest := integration.NewTest(t, integration.Config{
		HorizonParameters: map[string]string{
			"admin-port":                 strconv.Itoa(int(adminPort)),
			"enable-ingestion-filtering": "true",
		},
	})
	itest.Client().AdminPort = adminPort

	fullKeys, accounts := itest.CreateAccounts(2, "10000")
	whitelistedAccount := accounts[0]
	whitelistedAccountKey := fullKeys[0]
	nonWhitelistedAccount := accounts[1]
	nonWhitelistedAccountKey := fullKeys[1]

	whitelistedAsset := txnbuild.CreditAsset{Code: "PTS", Issuer: itest.Master().Address()}
	itest.MustEstablishTrustline(whitelistedAccountKey, whitelistedAccount, whitelistedAsset)
	itest.MustEstablishTrustline(nonWhitelistedAccountKey, nonWhitelistedAccount, whitelistedAsset)

	nonWhitelistedAsset := txnbuild.CreditAsset{Code: "SEK", Issuer: itest.Master().Address()}
	itest.MustEstablishTrustline(whitelistedAccountKey, whitelistedAccount, nonWhitelistedAsset)
	itest.MustEstablishTrustline(nonWhitelistedAccountKey, nonWhitelistedAccount, nonWhitelistedAsset)
	enabled := true

	// Initialize filters

	// Force refresh of filters to be quick
	filters.FilterConfigCheckIntervalSeconds = 1

	expectedAccountFilter := hProtocol.AccountFilterConfig{
		Whitelist: []string{whitelistedAccount.GetAccountID()},
		Enabled:   &enabled,
	}
	err := itest.Client().AdminSetIngestionAccountFilter(expectedAccountFilter)
	tt.NoError(err)

	accountFilter, err := itest.Client().AdminGetIngestionAccountFilter()
	tt.NoError(err)

	tt.ElementsMatch(expectedAccountFilter.Whitelist, accountFilter.Whitelist)
	tt.Equal(expectedAccountFilter.Enabled, accountFilter.Enabled)

	asset, err := whitelistedAsset.ToXDR()
	tt.NoError(err)
	expectedAssetFilter := hProtocol.AssetFilterConfig{
		Whitelist: []string{asset.StringCanonical()},
		Enabled:   &enabled,
	}
	err = itest.Client().AdminSetIngestionAssetFilter(expectedAssetFilter)
	tt.NoError(err)

	assetFilter, err := itest.Client().AdminGetIngestionAssetFilter()
	tt.NoError(err)

	tt.ElementsMatch(expectedAssetFilter.Whitelist, assetFilter.Whitelist)
	tt.Equal(expectedAssetFilter.Enabled, assetFilter.Enabled)

	// Ensure filters are refreshed and ready to go
	time.Sleep(time.Duration(filters.FilterConfigCheckIntervalSeconds) * time.Second)

	// Make sure that when using a non-whitelisted account and a non-whitelisted asset,
	// the transaction is not stored
	itest.MustSubmitOperations(itest.MasterAccount(), itest.Master(),
		&txnbuild.Payment{
			Destination: nonWhitelistedAccount.GetAccountID(),
			Amount:      "10",
			Asset:       nonWhitelistedAsset,
		},
	)

	// TODO: this should give a 404
	// resp, err := itest.Client().TransactionDetail(txResp.ID)
	//

}
