package horizon

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/simplepath"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
)

func TestPathActions_Index(t *testing.T) {
	ht := StartHTTPTest(t, "paths")
	defer ht.Finish()

	// no query args
	w := ht.Get("/paths")
	ht.Assert.Equal(400, w.Code)

	// happy path
	var q = make(url.Values)

	q.Add(
		"destination_account",
		"GAEDTJ4PPEFVW5XV2S7LUXBEHNQMX5Q2GM562RJGOQG7GVCE5H3HIB4V",
	)
	q.Add(
		"source_account",
		"GARSFJNXJIHO6ULUBK3DBYKVSIZE7SC72S5DYBCHU7DKL22UXKVD7MXP",
	)
	q.Add(
		"destination_asset_issuer",
		"GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN",
	)
	q.Add("destination_asset_type", "credit_alphanum4")
	q.Add("destination_asset_code", "EUR")
	q.Add("destination_amount", "10")

	w = ht.Get("/paths?" + q.Encode())
	ht.Assert.Equal(200, w.Code)
	ht.Assert.PageOf(3, w.Body)

}

func TestPathActionsStillIngesting(t *testing.T) {
	ht := StartHTTPTest(t, "paths")
	defer ht.Finish()

	orderBookGraph := orderbook.NewOrderBookGraph()
	ht.App.paths = simplepath.NewInMemoryFinder(orderBookGraph)

	var q = make(url.Values)

	q.Add(
		"destination_account",
		"GAEDTJ4PPEFVW5XV2S7LUXBEHNQMX5Q2GM562RJGOQG7GVCE5H3HIB4V",
	)
	q.Add(
		"source_account",
		"GARSFJNXJIHO6ULUBK3DBYKVSIZE7SC72S5DYBCHU7DKL22UXKVD7MXP",
	)
	q.Add(
		"destination_asset_issuer",
		"GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN",
	)
	q.Add("destination_asset_type", "credit_alphanum4")
	q.Add("destination_asset_code", "EUR")
	q.Add("destination_amount", "10")

	w := ht.Get("/paths?" + q.Encode())
	ht.Assert.Equal(problem.StillIngesting.Status, w.Code)
}

func stringToAccountID(tt *test.T, address string) xdr.AccountId {
	raw, err := strkey.Decode(strkey.VersionByteAccountID, address)
	tt.Assert.NoError(err)

	var key xdr.Uint256
	copy(key[:], raw)

	accountID, err := xdr.NewAccountId(xdr.PublicKeyTypePublicKeyTypeEd25519, key)
	tt.Assert.NoError(err)

	return accountID
}

func loadOffers(tt *test.T, orderBookGraph *orderbook.OrderBookGraph, fromAddress string) {
	coreQ := &core.Q{tt.CoreSession()}
	offers := []core.Offer{}
	pageQuery := db2.PageQuery{
		Order: db2.OrderAscending,
		Limit: 100,
	}
	err := coreQ.OffersByAddress(&offers, fromAddress, pageQuery)
	tt.Assert.NoError(err)
	for _, offer := range offers {

		orderBookGraph.AddOffer(xdr.OfferEntry{
			SellerId: stringToAccountID(tt, offer.SellerID),
			OfferId:  xdr.Int64(offer.OfferID),
			Selling:  offer.SellingAsset,
			Buying:   offer.BuyingAsset,
			Amount:   offer.Amount,
			Price:    xdr.Price{N: xdr.Int32(offer.Price * 100), D: 100},
		})
	}
	tt.Assert.NoError(orderBookGraph.Apply())
}

func TestPathActionsInMemoryFinder(t *testing.T) {
	ht := StartHTTPTest(t, "paths")
	defer ht.Finish()

	orderBookGraph := orderbook.NewOrderBookGraph()
	ht.App.paths = simplepath.NewInMemoryFinder(orderBookGraph)

	loadOffers(ht.T, orderBookGraph, "GA2NC4ZOXMXLVQAQQ5IQKJX47M3PKBQV2N5UV5Z4OXLQJ3CKMBA2O2YL")
	loadOffers(ht.T, orderBookGraph, "GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN")

	var q = make(url.Values)

	q.Add(
		"destination_account",
		"GAEDTJ4PPEFVW5XV2S7LUXBEHNQMX5Q2GM562RJGOQG7GVCE5H3HIB4V",
	)
	q.Add(
		"source_account",
		"GARSFJNXJIHO6ULUBK3DBYKVSIZE7SC72S5DYBCHU7DKL22UXKVD7MXP",
	)
	q.Add(
		"destination_asset_issuer",
		"GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN",
	)
	q.Add("destination_asset_type", "credit_alphanum4")
	q.Add("destination_asset_code", "EUR")
	q.Add("destination_amount", "10")

	w := ht.Get("/paths?" + q.Encode())
	ht.Assert.Equal(http.StatusOK, w.Code)
	inMemoryResponse := []horizon.Path{}
	ht.UnmarshalPage(w.Body, &inMemoryResponse)

	ht.App.paths = &simplepath.Finder{ht.App.CoreQ()}
	w = ht.Get("/paths?" + q.Encode())
	ht.Assert.Equal(http.StatusOK, w.Code)
	dbResponse := []horizon.Path{}
	ht.UnmarshalPage(w.Body, &dbResponse)

	ht.Assert.Equal(inMemoryResponse, dbResponse)
}
