package horizon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

var (
	issuer = xdr.MustAddress("GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H")
	seller = xdr.MustAddress("GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2")

	nativeAsset = xdr.MustNewNativeAsset()
	usdAsset    = xdr.MustNewCreditAsset("USD", issuer.Address())
	eurAsset    = xdr.MustNewCreditAsset("EUR", issuer.Address())

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

func TestOfferActions_Show(t *testing.T) {
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

		ledger := new(history.Ledger)
		err = q.LedgerBySequence(ledger, 3)

		ht.Assert.NoError(err)
		ht.Assert.True(ledger.ClosedAt.Equal(*result.LastModifiedTime))
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

		if ht.Assert.Equal(http.StatusOK, w.Code) {
			ht.Assert.PageOf(3, w.Body)

			var records []horizon.Offer
			ht.UnmarshalPage(w.Body, &records)

			ht.Assert.Equal(int64(eurOffer.OfferId), records[0].ID)
			ht.Assert.Equal("native", records[0].Selling.Type)
			ht.Assert.Equal("credit_alphanum4", records[0].Buying.Type)
			ht.Assert.Equal(issuer.Address(), records[0].Seller)
			ht.Assert.Equal(issuer.Address(), records[0].Buying.Issuer)
			ht.Assert.Equal(int32(3), records[0].LastModifiedLedger)

			ledger := new(history.Ledger)
			err := q.LedgerBySequence(ledger, 3)

			ht.Assert.NoError(err)
			ht.Assert.True(ledger.ClosedAt.Equal(*records[0].LastModifiedTime))
		}
	})

	t.Run("Filter by seller", func(t *testing.T) {
		w := ht.Get(fmt.Sprintf("/offers?seller=%s", issuer.Address()))

		if ht.Assert.Equal(http.StatusOK, w.Code) {
			ht.Assert.PageOf(2, w.Body)
			var records []horizon.Offer
			ht.UnmarshalPage(w.Body, &records)

			for _, record := range records {
				ht.Assert.Equal(issuer.Address(), record.Seller)
			}
		}
	})

	t.Run("Filter by selling asset", func(t *testing.T) {
		asset := horizon.Asset{}
		nativeAsset.Extract(&asset.Type, &asset.Code, &asset.Issuer)
		w := ht.Get(fmt.Sprintf("/offers?selling_asset_type=%s", asset.Type))

		if ht.Assert.Equal(http.StatusOK, w.Code) {
			ht.Assert.PageOf(2, w.Body)
			var records []horizon.Offer
			ht.UnmarshalPage(w.Body, &records)

			for _, record := range records {
				ht.Assert.Equal(asset, record.Selling)
			}
		}

		asset = horizon.Asset{}
		eurAsset.Extract(&asset.Type, &asset.Code, &asset.Issuer)

		url := fmt.Sprintf(
			"/offers?selling_asset_type=%s&selling_asset_code=%s&selling_asset_issuer=%s",
			asset.Type,
			asset.Code,
			asset.Issuer,
		)
		w = ht.Get(url)

		if ht.Assert.Equal(http.StatusOK, w.Code) {
			ht.Assert.PageOf(1, w.Body)
			var records []horizon.Offer
			ht.UnmarshalPage(w.Body, &records)

			for _, record := range records {
				ht.Assert.Equal(asset, record.Selling)
			}
		}
	})

	t.Run("Filter by buying asset", func(t *testing.T) {
		asset := horizon.Asset{}
		eurAsset.Extract(&asset.Type, &asset.Code, &asset.Issuer)

		url := fmt.Sprintf(
			"/offers?buying_asset_type=%s&buying_asset_code=%s&buying_asset_issuer=%s",
			asset.Type,
			asset.Code,
			asset.Issuer,
		)

		w := ht.Get(url)

		if ht.Assert.Equal(http.StatusOK, w.Code) {
			ht.Assert.PageOf(2, w.Body)
			var records []horizon.Offer
			ht.UnmarshalPage(w.Body, &records)

			for _, record := range records {
				ht.Assert.Equal(asset, record.Buying)
			}
		}

		asset = horizon.Asset{}
		usdAsset.Extract(&asset.Type, &asset.Code, &asset.Issuer)

		url = fmt.Sprintf(
			"/offers?buying_asset_type=%s&buying_asset_code=%s&buying_asset_issuer=%s",
			asset.Type,
			asset.Code,
			asset.Issuer,
		)

		w = ht.Get(url)

		if ht.Assert.Equal(http.StatusOK, w.Code) {
			ht.Assert.PageOf(1, w.Body)
			var records []horizon.Offer
			ht.UnmarshalPage(w.Body, &records)

			for _, record := range records {
				ht.Assert.Equal(asset, record.Buying)
			}
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
func TestOfferActions_AccountIndexExperimentalIngestion(t *testing.T) {
	ht := StartHTTPTest(t, "base")
	ht.App.config.EnableExperimentalIngestion = true
	defer ht.Finish()
	q := &history.Q{ht.HorizonSession()}

	ht.Assert.NoError(q.UpdateLastLedgerExpIngest(3))
	ht.Assert.NoError(q.UpsertOffer(eurOffer, 3))
	ht.Assert.NoError(q.UpsertOffer(twoEurOffer, 3))
	ht.Assert.NoError(q.UpsertOffer(usdOffer, 3))

	w := ht.Get(fmt.Sprintf("/accounts/%s/offers", issuer.Address()))

	if ht.Assert.Equal(http.StatusOK, w.Code) {
		ht.Assert.PageOf(2, w.Body)
		var records []horizon.Offer
		ht.UnmarshalPage(w.Body, &records)
		for _, record := range records {
			ht.Assert.Equal(issuer.Address(), record.Seller)
		}
	}
}
func TestOfferActions_AccountIndex(t *testing.T) {
	ht := StartHTTPTest(t, "trades")
	defer ht.Finish()
	q := &history.Q{ht.HorizonSession()}

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

		ht.Assert.EqualValues(8, records[2]["last_modified_ledger"])

		ledger := new(history.Ledger)
		err := q.LedgerBySequence(ledger, 8)
		ht.Assert.NoError(err)

		recordTime, err := time.Parse("2006-01-02T15:04:05Z", records[2]["last_modified_time"].(string))
		ht.Assert.NoError(err)
		ht.Assert.True(ledger.ClosedAt.Equal(recordTime))
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
