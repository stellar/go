package actions

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
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

func TestGetOfferByIDHandler(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &history.Q{tt.HorizonSession()}
	handler := GetOfferByID{}
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
	_, err = q.InsertOffer(usdOffer, 4)
	tt.Assert.NoError(err)

	for _, testCase := range []struct {
		name          string
		request       *http.Request
		expectedError func(error)
		expectedOffer func(hal.Pageable)
	}{
		{
			"offer id is invalid",
			makeRequest(
				t, map[string]string{}, map[string]string{"id": "invalid"}, q.Session,
			),
			func(err error) {
				tt.Assert.Error(err)
				p := err.(*problem.P)
				tt.Assert.Equal("bad_request", p.Type)
				tt.Assert.Equal("id", p.Extras["invalid_field"])
			},
			func(response hal.Pageable) {
				tt.Assert.Nil(response)
			},
		},
		{
			"offer does not exist",
			makeRequest(
				t, map[string]string{}, map[string]string{"id": "1234567"}, q.Session,
			),
			func(err error) {
				tt.Assert.Equal(err, sql.ErrNoRows)
			},
			func(response hal.Pageable) {
				tt.Assert.Nil(response)
			},
		},
		{
			"offer with ledger close time",
			makeRequest(
				t, map[string]string{}, map[string]string{"id": "4"}, q.Session,
			),
			func(err error) {
				tt.Assert.NoError(err)
			},
			func(response hal.Pageable) {
				offer := response.(horizon.Offer)
				tt.Assert.Equal(int64(eurOffer.OfferId), offer.ID)
				tt.Assert.Equal("native", offer.Selling.Type)
				tt.Assert.Equal("credit_alphanum4", offer.Buying.Type)
				tt.Assert.Equal("EUR", offer.Buying.Code)
				tt.Assert.Equal(issuer.Address(), offer.Seller)
				tt.Assert.Equal(issuer.Address(), offer.Buying.Issuer)
				tt.Assert.Equal(int32(3), offer.LastModifiedLedger)
				tt.Assert.Equal(ledgerCloseTime, offer.LastModifiedTime.Unix())
			},
		},
		{
			"offer without ledger close time",
			makeRequest(
				t, map[string]string{}, map[string]string{"id": "6"}, q.Session,
			),
			func(err error) {
				tt.Assert.NoError(err)
			},
			func(response hal.Pageable) {
				offer := response.(horizon.Offer)
				tt.Assert.Equal(int64(usdOffer.OfferId), offer.ID)
				tt.Assert.Equal("credit_alphanum4", offer.Selling.Type)
				tt.Assert.Equal("EUR", offer.Selling.Code)
				tt.Assert.Equal("credit_alphanum4", offer.Buying.Type)
				tt.Assert.Equal("USD", offer.Buying.Code)
				tt.Assert.Equal(issuer.Address(), offer.Seller)
				tt.Assert.Equal(issuer.Address(), offer.Selling.Issuer)
				tt.Assert.Equal(issuer.Address(), offer.Buying.Issuer)
				tt.Assert.Equal(int32(4), offer.LastModifiedLedger)
				tt.Assert.Nil(offer.LastModifiedTime)
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			offer, err := handler.GetResource(httptest.NewRecorder(), testCase.request)
			testCase.expectedError(err)
			testCase.expectedOffer(offer)
		})
	}
}

func TestGetOffersHandler(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &history.Q{tt.HorizonSession()}
	handler := GetOffersHandler{}
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
		records, err := handler.GetResourcePage(
			httptest.NewRecorder(),
			makeRequest(
				t, map[string]string{}, map[string]string{}, q.Session,
			),
		)
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
		records, err := handler.GetResourcePage(
			httptest.NewRecorder(),
			makeRequest(
				t,
				map[string]string{
					"seller": issuer.Address(),
				},
				map[string]string{},
				q.Session,
			),
		)
		tt.Assert.NoError(err)
		tt.Assert.Len(records, 2)

		offers := pageableToOffers(t, records)
		for _, offer := range offers {
			tt.Assert.Equal(issuer.Address(), offer.Seller)
		}

		_, err = handler.GetResourcePage(
			httptest.NewRecorder(),
			makeRequest(
				t,
				map[string]string{
					"seller": "GCXEWJ6U4KPGTNTBY5HX4WQ2EEVPWV2QKXEYIQ32IDYIX",
				},
				map[string]string{},
				q.Session,
			),
		)
		tt.Assert.Error(err)
		tt.Assert.IsType(&problem.P{}, err)
		p := err.(*problem.P)
		tt.Assert.Equal("bad_request", p.Type)
		tt.Assert.Equal("seller", p.Extras["invalid_field"])
		tt.Assert.Equal(
			"Account ID must start with `G` and contain 56 alphanum characters",
			p.Extras["reason"],
		)
	})

	t.Run("Filter by selling asset", func(t *testing.T) {
		asset := horizon.Asset{}
		nativeAsset.Extract(&asset.Type, &asset.Code, &asset.Issuer)
		records, err := handler.GetResourcePage(
			httptest.NewRecorder(),
			makeRequest(
				t,
				map[string]string{
					"selling_asset_type": asset.Type,
				},
				map[string]string{},
				q.Session,
			),
		)
		tt.Assert.NoError(err)
		tt.Assert.Len(records, 2)

		offers := pageableToOffers(t, records)
		for _, offer := range offers {
			tt.Assert.Equal(asset, offer.Selling)
		}

		asset = horizon.Asset{}
		eurAsset.Extract(&asset.Type, &asset.Code, &asset.Issuer)

		records, err = handler.GetResourcePage(
			httptest.NewRecorder(),
			makeRequest(
				t,
				map[string]string{
					"selling_asset_type":   asset.Type,
					"selling_asset_code":   asset.Code,
					"selling_asset_issuer": asset.Issuer,
				},
				map[string]string{},
				q.Session,
			),
		)
		tt.Assert.NoError(err)
		tt.Assert.Len(records, 1)

		offers = pageableToOffers(t, records)
		tt.Assert.Equal(asset, offers[0].Selling)
	})

	t.Run("Filter by buying asset", func(t *testing.T) {
		asset := horizon.Asset{}
		eurAsset.Extract(&asset.Type, &asset.Code, &asset.Issuer)

		records, err := handler.GetResourcePage(
			httptest.NewRecorder(),
			makeRequest(
				t,
				map[string]string{
					"buying_asset_type":   asset.Type,
					"buying_asset_code":   asset.Code,
					"buying_asset_issuer": asset.Issuer,
				},
				map[string]string{},
				q.Session,
			),
		)
		tt.Assert.NoError(err)
		tt.Assert.Len(records, 2)

		offers := pageableToOffers(t, records)
		for _, offer := range offers {
			tt.Assert.Equal(asset, offer.Buying)
		}

		asset = horizon.Asset{}
		usdAsset.Extract(&asset.Type, &asset.Code, &asset.Issuer)

		records, err = handler.GetResourcePage(
			httptest.NewRecorder(),
			makeRequest(
				t,
				map[string]string{
					"buying_asset_type":   asset.Type,
					"buying_asset_code":   asset.Code,
					"buying_asset_issuer": asset.Issuer,
				},
				map[string]string{},
				q.Session,
			),
		)
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
	handler := GetAccountOffersHandler{}

	_, err := q.InsertOffer(eurOffer, 3)
	tt.Assert.NoError(err)
	_, err = q.InsertOffer(twoEurOffer, 3)
	tt.Assert.NoError(err)
	_, err = q.InsertOffer(usdOffer, 3)
	tt.Assert.NoError(err)

	records, err := handler.GetResourcePage(
		httptest.NewRecorder(),
		makeRequest(
			t,
			map[string]string{},
			map[string]string{"account_id": issuer.Address()},
			q.Session,
		),
	)
	tt.Assert.NoError(err)
	tt.Assert.Len(records, 2)

	offers := pageableToOffers(t, records)

	for _, offer := range offers {
		tt.Assert.Equal(issuer.Address(), offer.Seller)
	}
}

func pageableToOffers(t *testing.T, page []hal.Pageable) []horizon.Offer {
	var offers []horizon.Offer
	for _, entry := range page {
		offers = append(offers, entry.(horizon.Offer))
	}
	return offers
}

func TestOffersQueryURLTemplate(t *testing.T) {
	tt := assert.New(t)
	expected := "/offers{?selling_asset_type,selling_asset_issuer,selling_asset_code,buying_asset_type,buying_asset_issuer,buying_asset_code,seller,cursor,limit,order}"
	offersQuery := OffersQuery{}
	tt.Equal(expected, offersQuery.URITemplate())
}
