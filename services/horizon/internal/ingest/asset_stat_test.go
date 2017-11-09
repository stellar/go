package ingest

import (
	"fmt"
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/xdr"
)

func TestStatTrustlinesInfo(t *testing.T) {
	type AssetState struct {
		assetType       xdr.AssetType
		assetCode       string
		assetIssuer     string
		wantNumAccounts int
		wantAmount      int64
	}

	testCases := []struct {
		scenario   string
		assetState []AssetState
	}{
		{
			"asset_stat_trustlines_1",
			[]AssetState{{
				xdr.AssetTypeAssetTypeCreditAlphanum4,
				"USD",
				"GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
				1,
				0,
			}},
		}, {
			"asset_stat_trustlines_2",
			[]AssetState{{
				xdr.AssetTypeAssetTypeCreditAlphanum4,
				"USD",
				"GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
				1,
				0,
			}},
		}, {
			"asset_stat_trustlines_3",
			[]AssetState{{
				xdr.AssetTypeAssetTypeCreditAlphanum4,
				"USD1",
				"GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
				1,
				0,
			}, {
				xdr.AssetTypeAssetTypeCreditAlphanum4,
				"USD2",
				"GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
				1,
				0,
			}},
		}, {
			"asset_stat_trustlines_4",
			[]AssetState{{
				xdr.AssetTypeAssetTypeCreditAlphanum4,
				"USD",
				"GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
				1,
				0,
			}, {
				xdr.AssetTypeAssetTypeCreditAlphanum4,
				"USD",
				"GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
				1,
				0,
			}},
		}, {
			"asset_stat_trustlines_5",
			[]AssetState{{
				xdr.AssetTypeAssetTypeCreditAlphanum4,
				"USD",
				"GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
				1,
				0,
			}},
		}, {
			"asset_stat_trustlines_6",
			[]AssetState{{
				xdr.AssetTypeAssetTypeCreditAlphanum4,
				"USD",
				"GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
				1,
				1012345000,
			}},
		}, {
			"asset_stat_trustlines_7",
			[]AssetState{{
				xdr.AssetTypeAssetTypeCreditAlphanum4,
				"USD",
				"GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
				2,
				1012345000,
			}},
		},
	}

	for _, kase := range testCases {
		t.Run(kase.scenario, func(t *testing.T) {
			tt := test.Start(t).ScenarioWithoutHorizon(kase.scenario)
			defer tt.Finish()

			session := &db.Session{DB: tt.CoreDB}
			coreQ := &core.Q{Session: session}

			for i, asset := range kase.assetState {
				numAccounts, amount, err := statTrustlinesInfo(
					coreQ,
					int32(asset.assetType),
					asset.assetCode,
					asset.assetIssuer,
				)

				tt.Require.NoError(err)
				tt.Assert.Equal(int32(asset.wantNumAccounts), numAccounts, fmt.Sprintf("asset index: %d", i))
				tt.Assert.Equal(int64(asset.wantAmount), amount, fmt.Sprintf("asset index: %d", i))
			}
		})
	}
}

func TestStatAccountInfo(t *testing.T) {
	testCases := []struct {
		account   string
		wantFlags int8
		wantToml  string
	}{
		{
			"GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
			0,
			"https://example.com/.well-known/stellar.toml",
		}, {
			"GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
			1,
			"https://abc.com/.well-known/stellar.toml",
		}, {
			"GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON",
			2,
			"",
		}, {
			"GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
			3,
			"",
		},
	}

	for _, kase := range testCases {
		t.Run(kase.account, func(t *testing.T) {
			tt := test.Start(t).ScenarioWithoutHorizon("asset_stat_account")
			defer tt.Finish()

			session := &db.Session{DB: tt.CoreDB}
			coreQ := &core.Q{Session: session}

			flags, toml, err := statAccountInfo(coreQ, kase.account)
			tt.Require.NoError(err)
			tt.Assert.Equal(kase.wantFlags, flags)
			tt.Assert.Equal(kase.wantToml, toml)
		})
	}
}
