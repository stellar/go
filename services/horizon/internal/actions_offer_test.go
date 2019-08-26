package horizon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/render/sse"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

func TestOfferActions_Show(t *testing.T) {
	var (
		issuer      = xdr.MustAddress("GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H")
		nativeAsset = xdr.Asset{
			Type: xdr.AssetTypeAssetTypeNative,
		}

		eurAsset = xdr.Asset{
			Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
			AlphaNum4: &xdr.AssetAlphaNum4{
				AssetCode: [4]byte{'e', 'u', 'r', 0},
				Issuer:    issuer,
			},
		}
		eurOffer = xdr.OfferEntry{
			SellerId: issuer,
			OfferId:  xdr.Int64(4),
			Buying:   eurAsset,
			Selling:  nativeAsset,
			Price: xdr.Price{
				N: 1,
				D: 1,
			},
			Flags:  1,
			Amount: xdr.Int64(500),
		}
		twoEurOffer = xdr.OfferEntry{
			SellerId: issuer,
			OfferId:  xdr.Int64(5),
			Buying:   eurAsset,
			Selling:  nativeAsset,
			Price: xdr.Price{
				N: 2,
				D: 1,
			},
			Flags:  2,
			Amount: xdr.Int64(500),
		}
	)

	ht := StartHTTPTest(t, "base")
	ht.App.config.EnableExperimentalIngestion = true
	defer ht.Finish()
	q := &history.Q{ht.HorizonSession()}

	ht.Assert.NoError(q.UpdateLastLedgerExpIngest(3))
	ht.Assert.NoError(q.UpsertOffer(eurOffer, 3))
	ht.Assert.NoError(q.UpsertOffer(twoEurOffer, 20))

	w := ht.Get(fmt.Sprintf("/offers/%v", eurOffer.OfferId))

	if ht.Assert.Equal(200, w.Code) {
		var result horizon.Offer
		err := json.Unmarshal(w.Body.Bytes(), &result)
		ht.Require.NoError(err)
		ht.Assert.Equal(int64(eurOffer.OfferId), result.ID)
		ht.Assert.Equal("native", result.Selling.Type)
		ht.Assert.Equal("credit_alphanum4", result.Buying.Type)
		ht.Assert.Equal(issuer.Address(), result.Seller)
		ht.Assert.Equal(issuer.Address(), result.Buying.Issuer)
		ht.Assert.Equal(int32(3), result.LastModifiedLedger)

		lastModifiedTime, err := time.Parse("2006-01-02 15:04:05", "2019-06-03 16:34:02")
		ht.Require.NoError(err)
		ht.Assert.Equal(lastModifiedTime, *result.LastModifiedTime)
	}

	w = ht.Get(fmt.Sprintf("/offers/%v", twoEurOffer.OfferId))

	if ht.Assert.Equal(200, w.Code) {
		var result horizon.Offer
		err := json.Unmarshal(w.Body.Bytes(), &result)
		ht.Require.NoError(err)
		ht.Assert.Equal(int32(20), result.LastModifiedLedger)
		ht.Assert.Nil(result.LastModifiedTime)
	}
}

func TestOfferActions_OfferDoesNotExist(t *testing.T) {
	ht := StartHTTPTest(t, "base")
	ht.App.config.EnableExperimentalIngestion = true
	defer ht.Finish()
	q := &history.Q{ht.HorizonSession()}
	ht.Assert.NoError(q.UpdateLastLedgerExpIngest(3))

	w := ht.Get("/offers/123456")

	ht.Assert.Equal(404, w.Code)
}

func TestOfferActionsStillIngesting_Show(t *testing.T) {
	ht := StartHTTPTest(t, "base")
	ht.App.config.EnableExperimentalIngestion = true
	defer ht.Finish()
	q := &history.Q{ht.HorizonSession()}
	ht.Assert.NoError(q.UpdateLastLedgerExpIngest(0))

	w := ht.Get("/offers/123456")
	ht.Assert.Equal(problem.StillIngesting.Status, w.Code)
}

func TestOfferActions_Index(t *testing.T) {
	var (
		issuer = xdr.MustAddress("GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H")
		seller = xdr.MustAddress("GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2")

		nativeAsset = xdr.Asset{
			Type: xdr.AssetTypeAssetTypeNative,
		}
		usdAsset = xdr.Asset{
			Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
			AlphaNum4: &xdr.AssetAlphaNum4{
				AssetCode: [4]byte{'u', 's', 'd', 0},
				Issuer:    issuer,
			},
		}
		eurAsset = xdr.Asset{
			Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
			AlphaNum4: &xdr.AssetAlphaNum4{
				AssetCode: [4]byte{'e', 'u', 'r', 0},
				Issuer:    issuer,
			},
		}
		eurOffer = xdr.OfferEntry{
			SellerId: issuer,
			OfferId:  xdr.Int64(4),
			Buying:   eurAsset,
			Selling:  nativeAsset,
			Price: xdr.Price{
				N: 1,
				D: 1,
			},
			Flags:  1,
			Amount: xdr.Int64(500),
		}
		twoEurOffer = xdr.OfferEntry{
			SellerId: seller,
			OfferId:  xdr.Int64(5),
			Buying:   eurAsset,
			Selling:  nativeAsset,
			Price: xdr.Price{
				N: 2,
				D: 1,
			},
			Flags:  2,
			Amount: xdr.Int64(500),
		}
		usdOffer = xdr.OfferEntry{
			SellerId: issuer,
			OfferId:  xdr.Int64(6),
			Buying:   usdAsset,
			Selling:  eurAsset,
			Price: xdr.Price{
				N: 1,
				D: 1,
			},
			Flags:  1,
			Amount: xdr.Int64(500),
		}
	)

	ht := StartHTTPTest(t, "base")
	ht.App.config.EnableExperimentalIngestion = true
	defer ht.Finish()
	q := &history.Q{ht.HorizonSession()}

	ht.Assert.NoError(q.UpdateLastLedgerExpIngest(3))
	ht.Assert.NoError(q.UpsertOffer(eurOffer, 3))
	ht.Assert.NoError(q.UpsertOffer(twoEurOffer, 3))
	ht.Assert.NoError(q.UpsertOffer(usdOffer, 3))

	t.Run("No filter", func(t *testing.T) {
		w := ht.Get("/offers")

		if ht.Assert.Equal(200, w.Code) {
			ht.Assert.PageOf(3, w.Body)

			var records []horizon.Offer
			ht.UnmarshalPage(w.Body, &records)

			ht.Assert.Equal(int64(eurOffer.OfferId), records[0].ID)
			ht.Assert.Equal("native", records[0].Selling.Type)
			ht.Assert.Equal("credit_alphanum4", records[0].Buying.Type)
			ht.Assert.Equal(issuer.Address(), records[0].Seller)
			ht.Assert.Equal(issuer.Address(), records[0].Buying.Issuer)
			ht.Assert.Equal(int32(3), records[0].LastModifiedLedger)

			lastModifiedTime, err := time.Parse("2006-01-02 15:04:05", "2019-06-03 16:34:02")
			ht.Require.NoError(err)
			ht.Assert.Equal(lastModifiedTime, *records[0].LastModifiedTime)
		}
	})

	t.Run("Filter by seller", func(t *testing.T) {
		w := ht.Get("/offers?seller=GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H")

		if ht.Assert.Equal(200, w.Code) {
			ht.Assert.PageOf(2, w.Body)
		}
	})

	t.Run("Filter by selling asset", func(t *testing.T) {
		w := ht.Get("/offers?selling_asset_type=native")

		if ht.Assert.Equal(200, w.Code) {
			ht.Assert.PageOf(2, w.Body)
		}

		url := fmt.Sprintf(
			"/offers?selling_asset_type=%s&selling_asset_code=%s&selling_asset_issuer=%s",
			"credit_alphanum4",
			"eur",
			"GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
		)

		w = ht.Get(url)

		if ht.Assert.Equal(200, w.Code) {
			ht.Assert.PageOf(1, w.Body)
		}
	})
}

func TestOfferActionsStillIngesting_Index(t *testing.T) {
	ht := StartHTTPTest(t, "base")
	ht.App.config.EnableExperimentalIngestion = true
	defer ht.Finish()
	q := &history.Q{ht.HorizonSession()}
	ht.Assert.NoError(q.UpdateLastLedgerExpIngest(0))

	w := ht.Get("/offers")
	ht.Assert.Equal(problem.StillIngesting.Status, w.Code)
}

func TestOfferActions_AccountIndex(t *testing.T) {
	ht := StartHTTPTest(t, "trades")
	defer ht.Finish()

	w := ht.Get(
		"/accounts/GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2/offers",
	)

	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.PageOf(3, w.Body)

		//test last modified timestamp
		var records []map[string]interface{}
		ht.UnmarshalPage(w.Body, &records)

		// Test asset fields population
		ht.Assert.Equal("credit_alphanum4", records[2]["selling"].(map[string]interface{})["asset_type"])
		ht.Assert.Equal("EUR", records[2]["selling"].(map[string]interface{})["asset_code"])
		ht.Assert.Equal("GCQPYGH4K57XBDENKKX55KDTWOTK5WDWRQOH2LHEDX3EKVIQRLMESGBG", records[2]["selling"].(map[string]interface{})["asset_issuer"])

		ht.Assert.Equal("credit_alphanum4", records[2]["buying"].(map[string]interface{})["asset_type"])
		ht.Assert.Equal("USD", records[2]["buying"].(map[string]interface{})["asset_code"])
		ht.Assert.Equal("GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", records[2]["buying"].(map[string]interface{})["asset_issuer"])

		t2018, err := time.Parse("2006-01-02", "2018-01-01")
		ht.Assert.NoError(err)
		recordTime, err := time.Parse("2006-01-02T15:04:05Z", records[2]["last_modified_time"].(string))
		ht.Assert.True(recordTime.After(t2018))
		ht.Assert.EqualValues(8, records[2]["last_modified_ledger"])
	}
}

func TestOfferActions_IndexNoLedgerData(t *testing.T) {
	ht := StartHTTPTest(t, "trades")
	defer ht.Finish()

	// Remove ledger data
	_, err := ht.App.HistoryQ().ExecRaw("DELETE FROM history_ledgers WHERE sequence=?", 8)
	ht.Assert.NoError(err)

	w := ht.Get(
		"/accounts/GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2/offers",
	)

	// Since 0.15.0 Horizon returns empty data instead of 500 error
	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.PageOf(3, w.Body)

		//test last modified timestamp
		var records []map[string]interface{}
		ht.UnmarshalPage(w.Body, &records)
		ht.Assert.NotEmpty(records[2]["last_modified_ledger"])
		ht.Assert.Nil(records[2]["last_modified_time"])
	}
}

func TestOfferActions_SSE(t *testing.T) {
	tt := test.Start(t).Scenario("trades")
	defer tt.Finish()

	ctx := context.Background()
	stream := sse.NewStream(ctx, httptest.NewRecorder())
	oa := OffersByAccountAction{Action: *NewTestAction(ctx, "/foo/bar?account_id=GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2")}

	oa.SSE(stream)
	tt.Require.NoError(oa.Err)

	_, err := tt.CoreSession().ExecRaw(
		`DELETE FROM offers WHERE sellerid = ?`,
		"GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
	)
	tt.Require.NoError(err)

	oa.SSE(stream)
	tt.Require.NoError(oa.Err)
}
