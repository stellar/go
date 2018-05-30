package assets

import (
	"strconv"
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/test"
)

func TestAssetsStatsQExec(t *testing.T) {
	item0 := AssetStatsR{
		SortKey:     "BTC_GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4_credit_alphanum4",
		Type:        "credit_alphanum4",
		Code:        "BTC",
		Issuer:      "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
		Amount:      "1009876000",
		NumAccounts: 1,
		Flags:       1,
		Toml:        "https://test.com/.well-known/stellar.toml",
	}

	item1 := AssetStatsR{
		SortKey:     "SCOT_GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU_credit_alphanum4",
		Type:        "credit_alphanum4",
		Code:        "SCOT",
		Issuer:      "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
		Amount:      "10000000000",
		NumAccounts: 1,
		Flags:       2,
		Toml:        "",
	}

	item2 := AssetStatsR{
		SortKey:     "USD_GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4_credit_alphanum4",
		Type:        "credit_alphanum4",
		Code:        "USD",
		Issuer:      "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
		Amount:      "3000010434000",
		NumAccounts: 2,
		Flags:       1,
		Toml:        "https://test.com/.well-known/stellar.toml",
	}

	testCases := []struct {
		query AssetStatsQ
		want  []AssetStatsR
	}{
		{
			AssetStatsQ{},
			[]AssetStatsR{item0, item1, item2},
		}, {
			AssetStatsQ{
				PageQuery: &db2.PageQuery{
					Order: "asc",
					Limit: 10,
				},
			},
			[]AssetStatsR{item0, item1, item2},
		}, {
			AssetStatsQ{
				PageQuery: &db2.PageQuery{
					Order: "desc",
					Limit: 10,
				},
			},
			[]AssetStatsR{item2, item1, item0},
		},
	}

	for i, kase := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			tt := test.Start(t).Scenario("ingest_asset_stats")
			defer tt.Finish()

			sql, err := kase.query.GetSQL()
			tt.Require.NoError(err)

			var results []AssetStatsR
			err = history.Q{Session: tt.HorizonSession()}.Select(&results, sql)
			tt.Require.NoError(err)
			if !tt.Assert.Equal(3, len(results)) {
				return
			}

			tt.Assert.Equal(len(kase.want), len(results))
			for i := range kase.want {
				tt.Assert.Equal(kase.want[i], results[i])
			}
		})
	}
}
