package integration

import (
	"strconv"
	"testing"
	"time"

	"github.com/stellar/go/clients/horizonclient"
	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/ingest/filters"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/assert"
)

func TestFilteringAccountWhiteList(t *testing.T) {
	tt := assert.New(t)
	const adminPort uint16 = 6000
	itest := integration.NewTest(t, integration.Config{
		HorizonParameters: map[string]string{
			"admin-port":                     strconv.Itoa(int(adminPort)),
			"exp-enable-ingestion-filtering": "true",
		},
	})

	fullKeys, accounts := itest.CreateAccounts(2, "10000")
	whitelistedAccount := accounts[0]
	whitelistedAccountKey := fullKeys[0]
	nonWhitelistedAccount := accounts[1]
	nonWhitelistedAccountKey := fullKeys[1]
	enabled := true

	// all assets are allowed by default because the asset filter config is empty.
	defaultAllowedAsset := txnbuild.CreditAsset{Code: "PTS", Issuer: itest.Master().Address()}
	itest.MustEstablishTrustline(whitelistedAccountKey, whitelistedAccount, defaultAllowedAsset)
	itest.MustEstablishTrustline(nonWhitelistedAccountKey, nonWhitelistedAccount, defaultAllowedAsset)

	// assert that by system default, filters with no rules yet, allow all first
	txResp := itest.MustSubmitOperations(itest.MasterAccount(), itest.Master(),
		&txnbuild.Payment{
			Destination: nonWhitelistedAccount.GetAccountID(),
			Amount:      "10",
			Asset:       defaultAllowedAsset,
		},
	)

	txResp, err := itest.Client().TransactionDetail(txResp.Hash)
	tt.NoError(err)

	// Setup a whitelisted account rule, force refresh of filter configs to be quick
	filters.FilterConfigCheckIntervalSeconds = 1

	expectedAccountFilter := hProtocol.AccountFilterConfig{
		Whitelist: []string{whitelistedAccount.GetAccountID()},
		Enabled:   &enabled,
	}
	err = itest.AdminClient().SetIngestionAccountFilter(expectedAccountFilter)
	tt.NoError(err)

	accountFilter, err := itest.AdminClient().GetIngestionAccountFilter()
	tt.NoError(err)

	tt.ElementsMatch(expectedAccountFilter.Whitelist, accountFilter.Whitelist)
	tt.Equal(expectedAccountFilter.Enabled, accountFilter.Enabled)

	// Ensure the latest filter configs are reloaded by the ingestion state machine processor
	time.Sleep(time.Duration(filters.FilterConfigCheckIntervalSeconds) * time.Second)

	// Make sure that when using a non-whitelisted account, the transaction is not stored
	txResp = itest.MustSubmitOperations(itest.MasterAccount(), itest.Master(),
		&txnbuild.Payment{
			Destination: nonWhitelistedAccount.GetAccountID(),
			Amount:      "10",
			Asset:       defaultAllowedAsset,
		},
	)
	_, err = itest.Client().TransactionDetail(txResp.Hash)
	tt.True(horizonclient.IsNotFoundError(err))

	// Make sure that when using a whitelisted account, the transaction is stored
	txResp = itest.MustSubmitOperations(itest.MasterAccount(), itest.Master(),
		&txnbuild.Payment{
			Destination: whitelistedAccount.GetAccountID(),
			Amount:      "10",
			Asset:       defaultAllowedAsset,
		},
	)
	_, err = itest.Client().TransactionDetail(txResp.Hash)
	tt.NoError(err)
}

func TestFilteringAssetWhiteList(t *testing.T) {
	tt := assert.New(t)
	const adminPort uint16 = 6000
	itest := integration.NewTest(t, integration.Config{
		HorizonParameters: map[string]string{
			"admin-port":                     strconv.Itoa(int(adminPort)),
			"exp-enable-ingestion-filtering": "true",
		},
	})

	fullKeys, accounts := itest.CreateAccounts(1, "10000")
	defaultAllowedAccount := accounts[0]
	defaultAllowedAccountKey := fullKeys[0]

	whitelistedAsset := txnbuild.CreditAsset{Code: "PTS", Issuer: itest.Master().Address()}
	itest.MustEstablishTrustline(defaultAllowedAccountKey, defaultAllowedAccount, whitelistedAsset)

	nonWhitelistedAsset := txnbuild.CreditAsset{Code: "SEK", Issuer: itest.Master().Address()}
	itest.MustEstablishTrustline(defaultAllowedAccountKey, defaultAllowedAccount, nonWhitelistedAsset)
	enabled := true

	// assert that by system default, filters with no rules yet, allow all first
	txResp := itest.MustSubmitOperations(itest.MasterAccount(), itest.Master(),
		&txnbuild.Payment{
			Destination: defaultAllowedAccount.GetAccountID(),
			Amount:      "10",
			Asset:       nonWhitelistedAsset,
		},
	)

	_, err := itest.Client().TransactionDetail(txResp.Hash)
	tt.NoError(err)

	// Setup a whitelisted asset rule, force refresh of filters to be quick
	filters.FilterConfigCheckIntervalSeconds = 1

	asset, err := whitelistedAsset.ToXDR()
	tt.NoError(err)
	expectedAssetFilter := hProtocol.AssetFilterConfig{
		Whitelist: []string{asset.StringCanonical()},
		Enabled:   &enabled,
	}
	err = itest.AdminClient().SetIngestionAssetFilter(expectedAssetFilter)
	tt.NoError(err)

	assetFilter, err := itest.AdminClient().GetIngestionAssetFilter()
	tt.NoError(err)

	tt.ElementsMatch(expectedAssetFilter.Whitelist, assetFilter.Whitelist)
	tt.Equal(expectedAssetFilter.Enabled, assetFilter.Enabled)

	// Ensure the latest filter configs are reloaded by the ingestion state machine processor
	time.Sleep(time.Duration(filters.FilterConfigCheckIntervalSeconds) * time.Second)

	// Make sure that when using a non-whitelisted asset, the transaction is not stored
	txResp = itest.MustSubmitOperations(itest.MasterAccount(), itest.Master(),
		&txnbuild.Payment{
			Destination: defaultAllowedAccount.GetAccountID(),
			Amount:      "10",
			Asset:       nonWhitelistedAsset,
		},
	)
	_, err = itest.Client().TransactionDetail(txResp.Hash)
	tt.True(horizonclient.IsNotFoundError(err))

	// Make sure that when using a whitelisted asset, the transaction is stored
	txResp = itest.MustSubmitOperations(itest.MasterAccount(), itest.Master(),
		&txnbuild.Payment{
			Destination: defaultAllowedAccount.GetAccountID(),
			Amount:      "10",
			Asset:       whitelistedAsset,
		},
	)
	_, err = itest.Client().TransactionDetail(txResp.Hash)
	tt.NoError(err)
}
