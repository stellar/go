package horizon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/actions"
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
)

func TestOfferActions_Show(t *testing.T) {
	ht := StartHTTPTest(t, "base")
	ht.App.config.EnableExperimentalIngestion = true
	defer ht.Finish()
	q := &history.Q{ht.HorizonSession()}

	ht.Assert.NoError(q.UpdateLastLedgerExpIngest(3))

	_, err := q.InsertOffer(eurOffer, 3)
	ht.Assert.NoError(err)
	_, err = q.InsertOffer(twoEurOffer, 20)
	ht.Assert.NoError(err)

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
	ht := StartHTTPTestWithoutScenario(t)
	ht.App.config.EnableExperimentalIngestion = true
	defer ht.Finish()
	q := &history.Q{ht.HorizonSession()}
	ht.Assert.NoError(q.UpdateLastLedgerExpIngest(3))

	w := ht.Get("/offers/123456")

	ht.Assert.Equal(404, w.Code)
}

func TestOfferActionsStillIngesting_Show(t *testing.T) {
	ht := StartHTTPTestWithoutScenario(t)
	ht.App.config.EnableExperimentalIngestion = true
	defer ht.Finish()
	q := &history.Q{ht.HorizonSession()}
	ht.Assert.NoError(q.UpdateLastLedgerExpIngest(0))

	w := ht.Get("/offers/123456")
	ht.Assert.Equal(problem.StillIngesting.Status, w.Code)
}

func TestOfferActionsRequiresExperimentalIngestion(t *testing.T) {
	ht := StartHTTPTestWithoutScenario(t)
	ht.App.config.EnableExperimentalIngestion = true
	defer ht.Finish()
	q := &history.Q{ht.HorizonSession()}
	ht.Assert.NoError(q.UpdateLastLedgerExpIngest(0))

	w := ht.Get("/offers")
	ht.Assert.Equal(problem.StillIngesting.Status, w.Code)

	ht.Assert.NoError(q.UpdateLastLedgerExpIngest(2))
	w = ht.Get("/offers")
	ht.Assert.Equal(http.StatusOK, w.Code)

	ht.App.config.EnableExperimentalIngestion = false
	w = ht.Get("/offers")
	ht.Assert.Equal(http.StatusNotFound, w.Code)
}

func TestOfferActionsExperimentalIngestion(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()

	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &history.Q{tt.HorizonSession()}
	app := &App{
		historyQ: q,
	}
	app.config.EnableExperimentalIngestion = true

	handler := actions.GetAccountOffersHandler{HistoryQ: q}
	client := accountOffersClient(tt, app, handler)

	tt.Assert.NoError(q.UpdateLastLedgerExpIngest(0))
	w := client.Get(fmt.Sprintf("/accounts/%s/offers", issuer.Address()))
	tt.Assert.Equal(problem.StillIngesting.Status, w.Code)

	tt.Assert.NoError(q.UpdateLastLedgerExpIngest(3))
	w = client.Get(fmt.Sprintf("/accounts/%s/offers", issuer.Address()))
	tt.Assert.Equal(http.StatusOK, w.Code)

	app.config.EnableExperimentalIngestion = false
	w = client.Get(fmt.Sprintf("/accounts/%s/offers", issuer.Address()))
	tt.Assert.Equal(http.StatusNotFound, w.Code)
}

func accountOffersClient(
	tt *test.T,
	app *App,
	handler actions.GetAccountOffersHandler,
) test.RequestHelper {
	router := chi.NewRouter()
	router.Use(appContextMiddleware(app))

	installAccountOfferRoute(handler, sse.StreamHandler{}, true, router)
	return test.NewRequestHelper(router)
}
