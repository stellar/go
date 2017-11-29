package horizon

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/render/hal"
	"github.com/stellar/go/services/horizon/internal/resource"
	"github.com/stellar/go/services/horizon/internal/resource/base"
)

func TestAssetsActions(t *testing.T) {
	testDomain := struct {
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
		Links: testDomain,
		Asset: base.Asset{
			Type:   "credit_alphanum4",
			Code:   "BTC",
			Issuer: "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
		},
		PT:          "3",
		Amount:      "100.9876000",
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
		Amount:      "1000.0000000",
		NumAccounts: 1,
		Flags: resource.AccountFlags{
			AuthRequired:  false,
			AuthRevocable: true,
		},
	}
	USDGateway := resource.AssetStat{
		Links: testDomain,
		Asset: base.Asset{
			Type:   "credit_alphanum4",
			Code:   "USD",
			Issuer: "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
		},
		PT:          "1",
		Amount:      "300001.0434000",
		NumAccounts: 2,
		Flags: resource.AccountFlags{
			AuthRequired:  true,
			AuthRevocable: false,
		},
	}

	testCases := []struct {
		path         string
		wantSelf     string
		wantPrevious string
		wantNext     string
		wantItems    []resource.AssetStat
	}{
		{
			"/assets",
			"/assets?order=asc&limit=10&cursor=",
			"/assets?order=desc&limit=10&cursor=1",
			"/assets?order=asc&limit=10&cursor=3",
			[]resource.AssetStat{USDGateway, SCOTScott, BTCGateway},
		},
		// limit
		{
			"/assets?limit=4",
			"/assets?order=asc&limit=4&cursor=",
			"/assets?order=desc&limit=4&cursor=1",
			"/assets?order=asc&limit=4&cursor=3",
			[]resource.AssetStat{USDGateway, SCOTScott, BTCGateway},
		}, {
			"/assets?limit=3",
			"/assets?order=asc&limit=3&cursor=",
			"/assets?order=desc&limit=3&cursor=1",
			"/assets?order=asc&limit=3&cursor=3",
			[]resource.AssetStat{USDGateway, SCOTScott, BTCGateway},
		}, {
			"/assets?limit=2",
			"/assets?order=asc&limit=2&cursor=",
			"/assets?order=desc&limit=2&cursor=1",
			"/assets?order=asc&limit=2&cursor=2",
			[]resource.AssetStat{USDGateway, SCOTScott},
		}, {
			"/assets?limit=1",
			"/assets?order=asc&limit=1&cursor=",
			"/assets?order=desc&limit=1&cursor=1",
			"/assets?order=asc&limit=1&cursor=1",
			[]resource.AssetStat{USDGateway},
		},
		// cursor
		{
			"/assets?cursor=0",
			"/assets?order=asc&limit=10&cursor=0",
			"/assets?order=desc&limit=10&cursor=1",
			"/assets?order=asc&limit=10&cursor=3",
			[]resource.AssetStat{USDGateway, SCOTScott, BTCGateway},
		}, {
			"/assets?cursor=1",
			"/assets?order=asc&limit=10&cursor=1",
			"/assets?order=desc&limit=10&cursor=2",
			"/assets?order=asc&limit=10&cursor=3",
			[]resource.AssetStat{SCOTScott, BTCGateway},
		}, {
			"/assets?cursor=2",
			"/assets?order=asc&limit=10&cursor=2",
			"/assets?order=desc&limit=10&cursor=3",
			"/assets?order=asc&limit=10&cursor=3",
			[]resource.AssetStat{BTCGateway},
		}, {
			"/assets?cursor=3",
			"/assets?order=asc&limit=10&cursor=3",
			"/assets?order=desc&limit=10&cursor=3", // TODO NNS 2 - I think this should be cursor=4 but it returns cursor=3, is that a bug?
			"/assets?order=asc&limit=10&cursor=3",
			[]resource.AssetStat{},
		},
		// order
		{
			"/assets?order=asc",
			"/assets?order=asc&limit=10&cursor=",
			"/assets?order=desc&limit=10&cursor=1",
			"/assets?order=asc&limit=10&cursor=3",
			[]resource.AssetStat{USDGateway, SCOTScott, BTCGateway},
		}, {
			"/assets?order=desc",
			"/assets?order=desc&limit=10&cursor=2147483647",
			"/assets?order=asc&limit=10&cursor=3",
			"/assets?order=desc&limit=10&cursor=1",
			[]resource.AssetStat{BTCGateway, SCOTScott, USDGateway},
		}, {
			"/assets?order=desc&cursor=3",
			"/assets?order=desc&limit=10&cursor=3",
			"/assets?order=asc&limit=10&cursor=2",
			"/assets?order=desc&limit=10&cursor=1",
			[]resource.AssetStat{SCOTScott, USDGateway},
		}, {
			"/assets?order=desc&cursor=1",
			"/assets?order=desc&limit=10&cursor=1",
			"/assets?order=asc&limit=10&cursor=1", // TODO NNS 2 - I think this should be cursor=0
			"/assets?order=desc&limit=10&cursor=1",
			[]resource.AssetStat{},
		},
		// asset_code
		{
			"/assets?asset_code=noexist",
			"/assets?order=asc&limit=10&cursor=&asset_code=noexist",
			"/assets?order=desc&limit=10&cursor=&asset_code=noexist", // TODO NNS 2 - imo, should be cursor=1
			"/assets?order=asc&limit=10&cursor=&asset_code=noexist",
			[]resource.AssetStat{},
		}, {
			"/assets?asset_code=USD",
			"/assets?order=asc&limit=10&cursor=&asset_code=USD",
			"/assets?order=desc&limit=10&cursor=1&asset_code=USD",
			"/assets?order=asc&limit=10&cursor=1&asset_code=USD",
			[]resource.AssetStat{USDGateway},
		}, {
			"/assets?asset_code=BTC",
			"/assets?order=asc&limit=10&cursor=&asset_code=BTC",
			"/assets?order=desc&limit=10&cursor=3&asset_code=BTC",
			"/assets?order=asc&limit=10&cursor=3&asset_code=BTC",
			[]resource.AssetStat{BTCGateway},
		}, {
			"/assets?asset_code=SCOT",
			"/assets?order=asc&limit=10&cursor=&asset_code=SCOT",
			"/assets?order=desc&limit=10&cursor=2&asset_code=SCOT",
			"/assets?order=asc&limit=10&cursor=2&asset_code=SCOT",
			[]resource.AssetStat{SCOTScott},
		},
		// asset_issuer
		{
			"/assets?asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
			"/assets?order=asc&limit=10&cursor=&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
			"/assets?order=desc&limit=10&cursor=1&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
			"/assets?order=asc&limit=10&cursor=3&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
			[]resource.AssetStat{USDGateway, BTCGateway},
		}, {
			"/assets?asset_issuer=GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
			"/assets?order=asc&limit=10&cursor=&asset_issuer=GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
			"/assets?order=desc&limit=10&cursor=2&asset_issuer=GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
			"/assets?order=asc&limit=10&cursor=2&asset_issuer=GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
			[]resource.AssetStat{SCOTScott},
		}, {
			"/assets?asset_issuer=GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
			"/assets?order=asc&limit=10&cursor=&asset_issuer=GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
			"/assets?order=desc&limit=10&cursor=&asset_issuer=GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", // TODO NNS 2 - imo should be cursor=1
			"/assets?order=asc&limit=10&cursor=&asset_issuer=GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
			[]resource.AssetStat{},
		},
		// combined
		{
			"/assets?asset_code=USD&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4&cursor=0",
			"/assets?order=asc&limit=10&cursor=0&asset_code=USD&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
			"/assets?order=desc&limit=10&cursor=1&asset_code=USD&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
			"/assets?order=asc&limit=10&cursor=1&asset_code=USD&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
			[]resource.AssetStat{USDGateway},
		}, {
			"/assets?asset_code=USD&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4&cursor=1",
			"/assets?order=asc&limit=10&cursor=1&asset_code=USD&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
			"/assets?order=desc&limit=10&cursor=1&asset_code=USD&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", // TODO NNS 2 - imo, should be cursor=2
			"/assets?order=asc&limit=10&cursor=1&asset_code=USD&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
			[]resource.AssetStat{},
		}, {
			"/assets?asset_code=BTC&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4&cursor=1",
			"/assets?order=asc&limit=10&cursor=1&asset_code=BTC&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
			"/assets?order=desc&limit=10&cursor=3&asset_code=BTC&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
			"/assets?order=asc&limit=10&cursor=3&asset_code=BTC&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
			[]resource.AssetStat{BTCGateway},
		}, {
			"/assets?asset_code=BTC&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
			"/assets?order=asc&limit=10&cursor=&asset_code=BTC&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
			"/assets?order=desc&limit=10&cursor=3&asset_code=BTC&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
			"/assets?order=asc&limit=10&cursor=3&asset_code=BTC&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
			[]resource.AssetStat{BTCGateway},
		}, {
			"/assets?asset_code=BTC&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4&limit=1&order=desc",
			"/assets?order=desc&limit=1&cursor=2147483647&asset_code=BTC&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
			"/assets?order=asc&limit=1&cursor=3&asset_code=BTC&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
			"/assets?order=desc&limit=1&cursor=3&asset_code=BTC&asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
			[]resource.AssetStat{BTCGateway},
		},
	}

	for _, kase := range testCases {
		t.Run(kase.path, func(t *testing.T) {
			ht := StartHTTPTest(t, "ingest_asset_stats")
			defer ht.Finish()

			w := ht.Get(kase.path)
			ht.Assert.Equal(200, w.Code)
			ht.Assert.PageOf(len(kase.wantItems), w.Body)

			records := []resource.AssetStat{}
			links := ht.UnmarshalPage(w.Body, &records)
			for i := range kase.wantItems {
				ht.Assert.Equal(kase.wantItems[i], records[i])
			}

			ht.Assert.Equal(("http://" + kase.wantSelf), links.Self.Href)
			ht.Assert.Equal(("http://" + kase.wantPrevious), links.Prev.Href)
			ht.Assert.Equal(("http://" + kase.wantNext), links.Next.Href)
		})
	}
}

func TestInvalidAssetCode(t *testing.T) {
	ht := StartHTTPTest(t, "ingest_asset_stats")
	defer ht.Finish()

	w := ht.Get("/assets?asset_code=ABCDEFGHIJKL")
	ht.Assert.Equal(200, w.Code)

	w = ht.Get("/assets?asset_code=ABCDEFGHIJKLM")
	ht.Assert.Equal(400, w.Code)
}

func TestInvalidAssetIssuer(t *testing.T) {
	ht := StartHTTPTest(t, "ingest_asset_stats")
	defer ht.Finish()

	w := ht.Get("/assets?asset_issuer=GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4")
	ht.Assert.Equal(200, w.Code)

	w = ht.Get("/assets?asset_issuer=invalid")
	ht.Assert.Equal(400, w.Code)
}
