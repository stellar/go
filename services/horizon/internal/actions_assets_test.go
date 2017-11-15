package horizon

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/render/hal"
	"github.com/stellar/go/services/horizon/internal/resource"
	"github.com/stellar/go/services/horizon/internal/resource/base"
)

func TestAssetsActions(t *testing.T) {
	testcom := struct {
		Toml hal.Link `json:"toml"`
	}{
		Toml: hal.NewLink("https://test.com/.well-known/stellar.toml"),
	}
	empty := struct {
		Toml hal.Link `json:"toml"`
	}{
		Toml: hal.NewLink(""),
	}

	BTCGateway := resource.AssetStat{
		Links: testcom,
		Asset: base.Asset{
			Type:   "credit_alphanum4",
			Code:   "BTC",
			Issuer: "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
		},
		PT:          "3",
		Amount:      1009876000,
		NumAccounts: 1,
		Flags: resource.AccountFlags{
			AuthRequired:  true,
			AuthRevocable: false,
		},
	}
	SCOTScott := resource.AssetStat{
		Links: empty,
		Asset: base.Asset{
			Type:   "credit_alphanum4",
			Code:   "SCOT",
			Issuer: "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
		},
		PT:          "2",
		Amount:      10000000000,
		NumAccounts: 1,
		Flags: resource.AccountFlags{
			AuthRequired:  false,
			AuthRevocable: true,
		},
	}
	USDGateway := resource.AssetStat{
		Links: testcom,
		Asset: base.Asset{
			Type:   "credit_alphanum4",
			Code:   "USD",
			Issuer: "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
		},
		PT:          "1",
		Amount:      3000010434000,
		NumAccounts: 2,
		Flags: resource.AccountFlags{
			AuthRequired:  true,
			AuthRevocable: false,
		},
	}

	testCases := []struct {
		path      string
		wantItems []resource.AssetStat
	}{
		{"/assets", []resource.AssetStat{USDGateway, SCOTScott, BTCGateway}},
		// limit
		{"/assets?limit=4", []resource.AssetStat{USDGateway, SCOTScott, BTCGateway}},
		{"/assets?limit=3", []resource.AssetStat{USDGateway, SCOTScott, BTCGateway}},
		{"/assets?limit=2", []resource.AssetStat{USDGateway, SCOTScott}},
		{"/assets?limit=1", []resource.AssetStat{USDGateway}},
		// cursor
		{"/assets?cursor=0", []resource.AssetStat{USDGateway, SCOTScott, BTCGateway}},
		{"/assets?cursor=1", []resource.AssetStat{SCOTScott, BTCGateway}},
		{"/assets?cursor=2", []resource.AssetStat{BTCGateway}},
		{"/assets?cursor=3", []resource.AssetStat{}},
		// order
		{"/assets?order=asc", []resource.AssetStat{USDGateway, SCOTScott, BTCGateway}},
		{"/assets?order=desc", []resource.AssetStat{BTCGateway, SCOTScott, USDGateway}},
		// asset_code
		{"/assets?asset_code=noexist", []resource.AssetStat{}},
		{"/assets?asset_code=USD", []resource.AssetStat{USDGateway}},
		{"/assets?asset_code=BTC", []resource.AssetStat{BTCGateway}},
		{"/assets?asset_code=SCOT", []resource.AssetStat{SCOTScott}},
		// asset_issuer
		{"/assets?asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", []resource.AssetStat{USDGateway, BTCGateway}},
		{"/assets?asset_issuer=GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", []resource.AssetStat{SCOTScott}},
		{"/assets?asset_issuer=GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", []resource.AssetStat{}},
		// combined
		{"/assets?asset_code=USD&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4&cursor=0", []resource.AssetStat{USDGateway}},
		{"/assets?asset_code=USD&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4&cursor=1", []resource.AssetStat{}},
		{"/assets?asset_code=BTC&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4&cursor=1", []resource.AssetStat{BTCGateway}},
		{"/assets?asset_code=BTC&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", []resource.AssetStat{BTCGateway}},
		{"/assets?asset_code=BTC&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4&limit=1&order=desc", []resource.AssetStat{BTCGateway}},
	}

	for _, kase := range testCases {
		t.Run(kase.path, func(t *testing.T) {
			ht := StartHTTPTest(t, "ingest_asset_stats")
			defer ht.Finish()

			w := ht.Get(kase.path)
			ht.Assert.Equal(200, w.Code)
			ht.Assert.PageOf(len(kase.wantItems), w.Body)

			records := []resource.AssetStat{}
			ht.UnmarshalPage(w.Body, &records)
			for i := range kase.wantItems {
				ht.Assert.Equal(kase.wantItems[i], records[i])
			}
		})
	}
}
