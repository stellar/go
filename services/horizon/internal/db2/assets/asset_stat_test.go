package assets

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/test"
)

func TestAssetsStatsQExec(t *testing.T) {
	tt := test.Start(t).Scenario("ingest_asset_stats")
	defer tt.Finish()

	sql, err := AssetStatsQ{}.GetSQL()
	tt.Require.NoError(err)

	var result []AssetStatsR
	err = history.Q{Session: tt.HorizonSession()}.Select(&result, sql)
	tt.Require.NoError(err)
	if !tt.Assert.Equal(3, len(result)) {
		return
	}

	tt.Assert.Equal(AssetStatsR{
		ID:          3,
		Type:        "credit_alphanum4",
		Code:        "BTC",
		Issuer:      "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
		Amount:      1009876000,
		NumAccounts: 1,
		Flags:       1,
		Toml:        "https://test.com/.well-known/stellar.toml",
	}, result[0])

	tt.Assert.Equal(AssetStatsR{
		ID:          2,
		Type:        "credit_alphanum4",
		Code:        "SCOT",
		Issuer:      "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
		Amount:      10000000000,
		NumAccounts: 1,
		Flags:       2,
		Toml:        "",
	}, result[1])

	tt.Assert.Equal(AssetStatsR{
		ID:          1,
		Type:        "credit_alphanum4",
		Code:        "USD",
		Issuer:      "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
		Amount:      3000010434000,
		NumAccounts: 2,
		Flags:       1,
		Toml:        "https://test.com/.well-known/stellar.toml",
	}, result[2])
}
