package horizon

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/paths"
	horizonProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/simplepath"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
)

func pathFindingClient(tt *test.T, pathFinder paths.Finder) test.RequestHelper {
	router := chi.NewRouter()
	findPaths := FindPathsHandler{
		pathFinder: pathFinder,
		coreQ:      &core.Q{tt.CoreSession()},
	}
	findFixedPaths := FindFixedPathsHandler{
		pathFinder: pathFinder,
		coreQ:      &core.Q{tt.CoreSession()},
	}

	installPathFindingRoutes(findPaths, findFixedPaths, router)
	return test.NewRequestHelper(router)
}

func TestPathActions_Index(t *testing.T) {
	tt := test.Start(t).Scenario("paths")
	assertions := &Assertions{tt.Assert}
	defer tt.Finish()
	rh := pathFindingClient(
		tt,
		&simplepath.Finder{
			Q: &core.Q{tt.CoreSession()},
		},
	)

	// no query args
	w := rh.Get("/paths")
	assertions.Equal(400, w.Code)

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

	for _, uri := range []string{"/paths", "/paths/strict-receive"} {
		w = rh.Get(uri + "?" + q.Encode())
		assertions.Equal(200, w.Code)
		assertions.PageOf(3, w.Body)
	}
}

func TestPathActionsStillIngesting(t *testing.T) {
	tt := test.Start(t).Scenario("paths")
	defer tt.Finish()
	assertions := &Assertions{tt.Assert}
	rh := pathFindingClient(
		tt,
		simplepath.NewInMemoryFinder(orderbook.NewOrderBookGraph()),
	)

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

	for _, uri := range []string{"/paths", "/paths/strict-receive"} {
		w := rh.Get(uri + "?" + q.Encode())
		assertions.Equal(horizonProblem.StillIngesting.Status, w.Code)
		assertions.Problem(w.Body, horizonProblem.StillIngesting)
	}
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
			SellerId: xdr.MustAddress(offer.SellerID),
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
	tt := test.Start(t).Scenario("paths")
	defer tt.Finish()
	orderBookGraph := orderbook.NewOrderBookGraph()
	assertions := &Assertions{tt.Assert}
	inMemoryPathsClient := pathFindingClient(
		tt,
		simplepath.NewInMemoryFinder(orderBookGraph),
	)
	dbPathsClient := pathFindingClient(
		tt,
		&simplepath.Finder{
			Q: &core.Q{tt.CoreSession()},
		},
	)

	loadOffers(tt, orderBookGraph, "GA2NC4ZOXMXLVQAQQ5IQKJX47M3PKBQV2N5UV5Z4OXLQJ3CKMBA2O2YL")
	loadOffers(tt, orderBookGraph, "GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN")

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

	for _, uri := range []string{"/paths", "/paths/strict-receive"} {
		w := inMemoryPathsClient.Get(uri + "?" + q.Encode())
		assertions.Equal(http.StatusOK, w.Code)
		inMemoryResponse := []horizon.Path{}
		tt.UnmarshalPage(w.Body, &inMemoryResponse)

		w = dbPathsClient.Get(uri + "?" + q.Encode())
		assertions.Equal(http.StatusOK, w.Code)
		dbResponse := []horizon.Path{}
		tt.UnmarshalPage(w.Body, &dbResponse)

		assertions.Equal(inMemoryResponse, dbResponse)
	}
}

func TestPathActionsEmptySourceAcount(t *testing.T) {
	tt := test.Start(t).Scenario("paths")
	defer tt.Finish()
	orderBookGraph := orderbook.NewOrderBookGraph()
	assertions := &Assertions{tt.Assert}
	inMemoryPathsClient := pathFindingClient(
		tt,
		simplepath.NewInMemoryFinder(orderBookGraph),
	)
	dbPathsClient := pathFindingClient(
		tt,
		&simplepath.Finder{
			Q: &core.Q{tt.CoreSession()},
		},
	)

	loadOffers(tt, orderBookGraph, "GA2NC4ZOXMXLVQAQQ5IQKJX47M3PKBQV2N5UV5Z4OXLQJ3CKMBA2O2YL")
	loadOffers(tt, orderBookGraph, "GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN")

	var q = make(url.Values)

	q.Add(
		"destination_account",
		"GAEDTJ4PPEFVW5XV2S7LUXBEHNQMX5Q2GM562RJGOQG7GVCE5H3HIB4V",
	)
	q.Add(
		"source_account",
		// there is no account associated with this address
		"GD5PM5X7Q5MM54ERO2P5PXW3HD6HVZI5IRZGEDWS4OPFBGHNTF6XOWQO",
	)
	q.Add(
		"destination_asset_issuer",
		"GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN",
	)
	q.Add("destination_asset_type", "credit_alphanum4")
	q.Add("destination_asset_code", "EUR")
	q.Add("destination_amount", "10")

	for _, uri := range []string{"/paths", "/paths/strict-receive"} {
		w := inMemoryPathsClient.Get(uri + "?" + q.Encode())
		assertions.Equal(http.StatusOK, w.Code)
		inMemoryResponse := []horizon.Path{}
		tt.UnmarshalPage(w.Body, &inMemoryResponse)
		assertions.Empty(inMemoryResponse)

		w = dbPathsClient.Get(uri + "?" + q.Encode())
		assertions.Equal(http.StatusOK, w.Code)
		dbResponse := []horizon.Path{}
		tt.UnmarshalPage(w.Body, &dbResponse)
		assertions.Empty(dbResponse)
	}
}

func TestPathActionsStrictSendDestinationAssetValidation(t *testing.T) {
	tt := test.Start(t).Scenario("paths")
	defer tt.Finish()
	assertions := &Assertions{tt.Assert}
	orderBookGraph := orderbook.NewOrderBookGraph()
	rh := pathFindingClient(
		tt,
		simplepath.NewInMemoryFinder(orderBookGraph),
	)

	missingDestinationAccountAndAsset := make(url.Values)
	missingDestinationAccountAndAsset.Add(
		"source_asset_issuer",
		"GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN",
	)
	missingDestinationAccountAndAsset.Add(
		"source_account",
		"GARSFJNXJIHO6ULUBK3DBYKVSIZE7SC72S5DYBCHU7DKL22UXKVD7MXP",
	)
	missingDestinationAccountAndAsset.Add("source_asset_type", "credit_alphanum4")
	missingDestinationAccountAndAsset.Add("source_asset_code", "USD")
	missingDestinationAccountAndAsset.Add("source_amount", "10")

	destinationAccountAndAsset, err := url.ParseQuery(
		missingDestinationAccountAndAsset.Encode(),
	)
	tt.Assert.NoError(err)
	destinationAccountAndAsset.Add(
		"destination_asset_issuer",
		"GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN",
	)
	destinationAccountAndAsset.Add("destination_asset_type", "credit_alphanum4")
	destinationAccountAndAsset.Add("destination_asset_code", "EUR")
	destinationAccountAndAsset.Add(
		"destination_account",
		"GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN",
	)

	destinationAccountMatchesSourceAccount, err := url.ParseQuery(
		missingDestinationAccountAndAsset.Encode(),
	)
	tt.Assert.NoError(err)
	destinationAccountMatchesSourceAccount.Add(
		"destination_account",
		"GARSFJNXJIHO6ULUBK3DBYKVSIZE7SC72S5DYBCHU7DKL22UXKVD7MXP",
	)

	for _, testCase := range []struct {
		name            string
		q               url.Values
		expectedProblem problem.P
	}{
		{
			"both destination asset and destination account are missing",
			missingDestinationAccountAndAsset,
			destinationAssetOrDestinationAccount,
		},
		{
			"both destination asset and destination account are present",
			destinationAccountAndAsset,
			destinationAssetOrDestinationAccount,
		},
		{
			"destination account is the same as source account",
			destinationAccountMatchesSourceAccount,
			sourceAccountIsDestinationAccount,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			w := rh.Get("/paths/strict-send?" + testCase.q.Encode())
			assertions.Equal(testCase.expectedProblem.Status, w.Code)
			assertions.Problem(w.Body, testCase.expectedProblem)
		})
	}
}

func TestPathActionsStrictSend(t *testing.T) {
	tt := test.Start(t).Scenario("paths")
	defer tt.Finish()
	assertions := &Assertions{tt.Assert}
	orderBookGraph := orderbook.NewOrderBookGraph()
	rh := pathFindingClient(
		tt,
		simplepath.NewInMemoryFinder(orderBookGraph),
	)

	loadOffers(tt, orderBookGraph, "GA2NC4ZOXMXLVQAQQ5IQKJX47M3PKBQV2N5UV5Z4OXLQJ3CKMBA2O2YL")
	loadOffers(tt, orderBookGraph, "GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN")

	var q = make(url.Values)

	q.Add(
		"source_asset_issuer",
		"GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN",
	)
	q.Add(
		"source_account",
		"GARSFJNXJIHO6ULUBK3DBYKVSIZE7SC72S5DYBCHU7DKL22UXKVD7MXP",
	)
	q.Add(
		"destination_asset_issuer",
		"GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN",
	)
	q.Add("source_asset_type", "credit_alphanum4")
	q.Add("source_asset_code", "USD")
	q.Add("source_amount", "10")
	q.Add("destination_asset_type", "credit_alphanum4")
	q.Add("destination_asset_code", "EUR")

	w := rh.Get("/paths/strict-send?" + q.Encode())
	assertions.Equal(http.StatusOK, w.Code)
	inMemoryResponse := []horizon.Path{}
	tt.UnmarshalPage(w.Body, &inMemoryResponse)
	assertions.Len(inMemoryResponse, 4)
	for i, path := range inMemoryResponse {
		assertions.Equal(path.SourceAssetCode, "USD")
		assertions.Equal(path.SourceAssetType, "credit_alphanum4")
		assertions.Equal(path.SourceAssetIssuer, "GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN")
		assertions.Equal(path.SourceAmount, "10.0000000")

		assertions.Equal(path.DestinationAssetCode, "EUR")
		assertions.Equal(path.DestinationAssetType, "credit_alphanum4")
		assertions.Equal(path.DestinationAssetIssuer, "GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN")
		if i > 1 {
			previous, err := strconv.ParseFloat(inMemoryResponse[i-1].DestinationAmount, 64)
			assertions.NoError(err)

			current, err := strconv.ParseFloat(path.DestinationAmount, 64)
			assertions.NoError(err)

			assertions.True(previous >= current)
		}
	}
}

func TestPathActionsStrictSendAccount(t *testing.T) {
	tt := test.Start(t).Scenario("paths")
	defer tt.Finish()
	assertions := &Assertions{tt.Assert}
	orderBookGraph := orderbook.NewOrderBookGraph()
	rh := pathFindingClient(
		tt,
		simplepath.NewInMemoryFinder(orderBookGraph),
	)

	loadOffers(tt, orderBookGraph, "GA2NC4ZOXMXLVQAQQ5IQKJX47M3PKBQV2N5UV5Z4OXLQJ3CKMBA2O2YL")
	loadOffers(tt, orderBookGraph, "GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN")

	var q = make(url.Values)

	q.Add(
		"source_asset_issuer",
		"GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN",
	)
	q.Add("source_asset_type", "credit_alphanum4")
	q.Add("source_asset_code", "USD")
	q.Add("source_amount", "10")
	q.Add(
		"destination_account",
		"GA2NC4ZOXMXLVQAQQ5IQKJX47M3PKBQV2N5UV5Z4OXLQJ3CKMBA2O2YL",
	)

	w := rh.Get("/paths/strict-send?" + q.Encode())
	assertions.Equal(http.StatusOK, w.Code)
	inMemoryResponse := []horizon.Path{}
	tt.UnmarshalPage(w.Body, &inMemoryResponse)
	assertions.Len(inMemoryResponse, 12)

	for i, path := range inMemoryResponse {
		assertions.Equal(path.SourceAssetCode, "USD")
		assertions.Equal(path.SourceAssetType, "credit_alphanum4")
		assertions.Equal(path.SourceAssetIssuer, "GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN")
		assertions.Equal(path.SourceAmount, "10.0000000")

		if path.DestinationAssetType == "credit_alphanum4" && path.DestinationAssetCode == "USD" {
			assertions.Equal(path.DestinationAssetIssuer, "GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN")
			assertions.Equal(path.DestinationAmount, "10.0000000")
			assertions.Len(path.Path, 0)
		}

		if i > 1 &&
			inMemoryResponse[i-1].DestinationAssetType == path.DestinationAssetType &&
			inMemoryResponse[i-1].DestinationAssetCode == path.DestinationAssetCode &&
			inMemoryResponse[i-1].DestinationAssetIssuer == path.DestinationAssetIssuer {
			previous, err := strconv.ParseFloat(inMemoryResponse[i-1].DestinationAmount, 64)
			assertions.NoError(err)

			current, err := strconv.ParseFloat(path.DestinationAmount, 64)
			assertions.NoError(err)

			assertions.True(previous >= current)
		}
	}
}
