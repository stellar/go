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

	// Initialize filters
	err := itest.Client().AdminSetIngestionFilter(hProtocol.IngestionFilter{
		Rules: map[string]interface{}{
			// TODO: this rule structure shouldn't  be implicit for the the SDK client
			"account_whitelist": []string{whitelistedAccount.GetAccountID()},
		},
		Enabled: true,
		Name:    hProtocol.IngestionFilterAccountName,
	})
	tt.NoError(err)

	filter, err := itest.Client().AdminGetIngestionFilter(hProtocol.IngestionFilterAccountName)
	tt.NoError(err)
	// TODO: The client returns a []string{} when getting the filter by requires []interface{} when setting it
	expectedAccountRules := map[string]interface{}{
		// TODO: this rule structure shouldn't  be implicit for the the SDK client
		"account_whitelist": []interface{}{whitelistedAccount.GetAccountID()},
	}
	tt.Equal(expectedAccountRules, filter.Rules)
	tt.Equal(hProtocol.IngestionFilterAccountName, filter.Name)

	// TODO ....
}
