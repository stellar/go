package actions

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/support/render/hal"
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

func makeOffersRequest(t *testing.T, queryParams map[string]string) *http.Request {
	request, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	query := url.Values{}
	for key, value := range queryParams {
		query.Set(key, value)
	}
	request.URL.RawQuery = query.Encode()

	ctx := context.WithValue(context.Background(), chi.RouteCtxKey, chi.NewRouteContext())
	return request.WithContext(ctx)
}

func makeAccountOffersRequest(
	t *testing.T,
	accountID string,
	queryParams map[string]string,
) *http.Request {
	request, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	query := url.Values{}
	for key, value := range queryParams {
		query.Set(key, value)
	}
	request.URL.RawQuery = query.Encode()

	chiRouteContext := chi.NewRouteContext()
	chiRouteContext.URLParams.Add("account_id", accountID)
	ctx := context.WithValue(context.Background(), chi.RouteCtxKey, chiRouteContext)
	return request.WithContext(ctx)
}

func pageableToOffers(t *testing.T, page []hal.Pageable) []horizon.Offer {
	var offers []horizon.Offer
	for _, entry := range page {
		offers = append(offers, entry.(horizon.Offer))
	}
	return offers
}

func TestGetOffersHandler(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &history.Q{tt.HorizonSession()}
	handler := GetOffersHandler{HistoryQ: q}
	ingestion := ingest.Ingestion{DB: tt.HorizonSession()}

	ledgerCloseTime := time.Now().Unix()
	tt.Assert.NoError(ingestion.Start())
	ingestion.Ledger(
		1,
		&core.LedgerHeader{Sequence: 3, CloseTime: ledgerCloseTime},
		0,
		0,
		0,
	)
	tt.Assert.NoError(ingestion.Flush())
	tt.Assert.NoError(ingestion.Close())

	_, err := q.InsertOffer(eurOffer, 3)
	tt.Assert.NoError(err)
	_, err = q.InsertOffer(twoEurOffer, 3)
	tt.Assert.NoError(err)
	_, err = q.InsertOffer(usdOffer, 3)
	tt.Assert.NoError(err)

	t.Run("No filter", func(t *testing.T) {
		records, err := handler.GetResourcePage(makeOffersRequest(t, map[string]string{}))
		tt.Assert.NoError(err)
		tt.Assert.Len(records, 3)

		offers := pageableToOffers(t, records)

		tt.Assert.Equal(int64(eurOffer.OfferId), offers[0].ID)
		tt.Assert.Equal("native", offers[0].Selling.Type)
		tt.Assert.Equal("credit_alphanum4", offers[0].Buying.Type)
		tt.Assert.Equal(issuer.Address(), offers[0].Seller)
		tt.Assert.Equal(issuer.Address(), offers[0].Buying.Issuer)
		tt.Assert.Equal(int32(3), offers[0].LastModifiedLedger)
		tt.Assert.Equal(ledgerCloseTime, offers[0].LastModifiedTime.Unix())
	})

	t.Run("Filter by seller", func(t *testing.T) {
		records, err := handler.GetResourcePage(makeOffersRequest(
			t,
			map[string]string{
				"seller": issuer.Address(),
			},
		))
		tt.Assert.NoError(err)
		tt.Assert.Len(records, 2)

		offers := pageableToOffers(t, records)
		for _, offer := range offers {
			tt.Assert.Equal(issuer.Address(), offer.Seller)
		}
	})

	t.Run("Filter by selling asset", func(t *testing.T) {
		asset := horizon.Asset{}
		nativeAsset.Extract(&asset.Type, &asset.Code, &asset.Issuer)

		records, err := handler.GetResourcePage(makeOffersRequest(
			t,
			map[string]string{
				"selling_asset_type": asset.Type,
			},
		))
		tt.Assert.NoError(err)
		tt.Assert.Len(records, 2)

		offers := pageableToOffers(t, records)
		for _, offer := range offers {
			tt.Assert.Equal(asset, offer.Selling)
		}

		asset = horizon.Asset{}
		eurAsset.Extract(&asset.Type, &asset.Code, &asset.Issuer)

		records, err = handler.GetResourcePage(makeOffersRequest(
			t,
			map[string]string{
				"selling_asset_type":   asset.Type,
				"selling_asset_code":   asset.Code,
				"selling_asset_issuer": asset.Issuer,
			},
		))
		tt.Assert.NoError(err)
		tt.Assert.Len(records, 1)

		offers = pageableToOffers(t, records)
		tt.Assert.Equal(asset, offers[0].Selling)
	})

	t.Run("Filter by buying asset", func(t *testing.T) {
		asset := horizon.Asset{}
		eurAsset.Extract(&asset.Type, &asset.Code, &asset.Issuer)

		records, err := handler.GetResourcePage(makeOffersRequest(
			t,
			map[string]string{
				"buying_asset_type":   asset.Type,
				"buying_asset_code":   asset.Code,
				"buying_asset_issuer": asset.Issuer,
			},
		))
		tt.Assert.NoError(err)
		tt.Assert.Len(records, 2)

		offers := pageableToOffers(t, records)
		for _, offer := range offers {
			tt.Assert.Equal(asset, offer.Buying)
		}

		asset = horizon.Asset{}
		usdAsset.Extract(&asset.Type, &asset.Code, &asset.Issuer)

		records, err = handler.GetResourcePage(makeOffersRequest(
			t,
			map[string]string{
				"buying_asset_type":   asset.Type,
				"buying_asset_code":   asset.Code,
				"buying_asset_issuer": asset.Issuer,
			},
		))
		tt.Assert.NoError(err)
		tt.Assert.Len(records, 1)

		offers = pageableToOffers(t, records)
		for _, offer := range offers {
			tt.Assert.Equal(asset, offer.Buying)
		}
	})
}

func TestGetAccountOffersHandler(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()

	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &history.Q{tt.HorizonSession()}
	handler := GetAccountOffersHandler{
		HistoryQ: q,
	}

	_, err := q.InsertOffer(eurOffer, 3)
	tt.Assert.NoError(err)
	_, err = q.InsertOffer(twoEurOffer, 3)
	tt.Assert.NoError(err)
	_, err = q.InsertOffer(usdOffer, 3)
	tt.Assert.NoError(err)

	records, err := handler.GetResourcePage(
		makeAccountOffersRequest(t, issuer.Address(), map[string]string{}),
	)
	tt.Assert.NoError(err)
	tt.Assert.Len(records, 2)

	offers := pageableToOffers(t, records)

	for _, offer := range offers {
		tt.Assert.Equal(issuer.Address(), offer.Seller)
	}
}
