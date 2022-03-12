package integration

import (
	"strconv"
	"testing"

	hProtocol "github.com/stellar/go/protocols/horizon"
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
	err := itest.Client().AdminSetIngestionAccountFilter(hProtocol.AccountFilterConfig{
		Whitelist: []string{whitelistedAccount.GetAccountID()},
		Enabled:   &enabled,
	})
	tt.NoError(err)

	filter, err := itest.Client().AdminGetIngestionAccountFilter()
	tt.NoError(err)
	expectedAccountRules := []string{whitelistedAccount.GetAccountID()}

	tt.ElementsMatch(filter.Whitelist, expectedAccountRules)
	tt.Equal(filter.Enabled, true)
}
