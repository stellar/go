package horizon

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

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

func pathFindingClient(tt *test.T, pathFinder paths.Finder, maxAssetsParamLength int) test.RequestHelper {
	router := chi.NewRouter()
	findPaths := FindPathsHandler{
		pathFinder:           pathFinder,
		maxAssetsParamLength: maxAssetsParamLength,
		coreQ:                &core.Q{tt.CoreSession()},
	}
	findFixedPaths := FindFixedPathsHandler{
		pathFinder:           pathFinder,
		maxAssetsParamLength: maxAssetsParamLength,
		coreQ:                &core.Q{tt.CoreSession()},
	}

	installPathFindingRoutes(findPaths, findFixedPaths, router, false)
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
		3,
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
		3,
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

func TestPathActionsStateInvalid(t *testing.T) {
	rh := StartHTTPTest(t, "paths")
	defer rh.Finish()

	rh.App.config.EnableExperimentalIngestion = true
	rh.App.web.router = chi.NewRouter()
	orderBookGraph := orderbook.NewOrderBookGraph()
	rh.App.web.mustInstallMiddlewares(rh.App, time.Minute)
	rh.App.web.mustInstallActions(
		rh.App.config,
		simplepath.NewInMemoryFinder(orderBookGraph),
	)
	rh.RH = test.NewRequestHelper(rh.App.web.router)

	w := rh.Get("/paths")
	// Still ingesting
	rh.Assert.Equal(503, w.Code)

	err := rh.App.historyQ.UpdateLastLedgerExpIngest(10)
	rh.Assert.NoError(err)

	err = rh.App.historyQ.UpdateExpStateInvalid(true)
	rh.Assert.NoError(err)

	w = rh.Get("/paths")
	// State invalid
	rh.Assert.Equal(500, w.Code)
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

	coreQ := &core.Q{tt.CoreSession()}
	sourceAccount := "GARSFJNXJIHO6ULUBK3DBYKVSIZE7SC72S5DYBCHU7DKL22UXKVD7MXP"
	sourceAssets, _, err := coreQ.AssetsForAddress(sourceAccount)
	tt.Assert.NoError(err)

	inMemoryPathsClient := pathFindingClient(
		tt,
		simplepath.NewInMemoryFinder(orderBookGraph),
		len(sourceAssets),
	)
	dbPathsClient := pathFindingClient(
		tt,
		&simplepath.Finder{
			Q: &core.Q{tt.CoreSession()},
		},
		len(sourceAssets),
	)

	loadOffers(tt, orderBookGraph, "GA2NC4ZOXMXLVQAQQ5IQKJX47M3PKBQV2N5UV5Z4OXLQJ3CKMBA2O2YL")
	loadOffers(tt, orderBookGraph, "GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN")

	var withSourceAccount = make(url.Values)
	withSourceAccount.Add(
		"destination_account",
		"GAEDTJ4PPEFVW5XV2S7LUXBEHNQMX5Q2GM562RJGOQG7GVCE5H3HIB4V",
	)
	withSourceAccount.Add(
		"source_account",
		sourceAccount,
	)
	withSourceAccount.Add(
		"destination_asset_issuer",
		"GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN",
	)
	withSourceAccount.Add("destination_asset_type", "credit_alphanum4")
	withSourceAccount.Add("destination_asset_code", "EUR")
	withSourceAccount.Add("destination_amount", "10")

	withSourceAssets, err := url.ParseQuery(
		withSourceAccount.Encode(),
	)
	tt.Assert.NoError(err)
	withSourceAssets.Del("source_account")
	withSourceAssets.Add("source_assets", assetsToURLParam(sourceAssets))

	for _, uri := range []string{"/paths", "/paths/strict-receive"} {
		w := inMemoryPathsClient.Get(uri + "?" + withSourceAccount.Encode())
		tt.Assert.Equal(http.StatusOK, w.Code)
		inMemorySourceAccountResponse := []horizon.Path{}
		tt.UnmarshalPage(w.Body, &inMemorySourceAccountResponse)

		w = dbPathsClient.Get(uri + "?" + withSourceAccount.Encode())
		tt.Assert.Equal(http.StatusOK, w.Code)
		dbSourceAccountResponse := []horizon.Path{}
		tt.UnmarshalPage(w.Body, &dbSourceAccountResponse)

		tt.Assert.True(len(inMemorySourceAccountResponse) > 0)
		tt.Assert.Equal(inMemorySourceAccountResponse, dbSourceAccountResponse)

		w = inMemoryPathsClient.Get(uri + "?" + withSourceAssets.Encode())
		tt.Assert.Equal(http.StatusOK, w.Code)
		inMemorySourceAssetsResponse := []horizon.Path{}
		tt.UnmarshalPage(w.Body, &inMemorySourceAssetsResponse)

		w = dbPathsClient.Get(uri + "?" + withSourceAccount.Encode())
		tt.Assert.Equal(http.StatusOK, w.Code)
		dbSourceAssetsResponse := []horizon.Path{}
		tt.UnmarshalPage(w.Body, &dbSourceAssetsResponse)

		tt.Assert.Equal(inMemorySourceAssetsResponse, dbSourceAssetsResponse)
		tt.Assert.Equal(inMemorySourceAssetsResponse, inMemorySourceAccountResponse)
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
		3,
	)
	dbPathsClient := pathFindingClient(
		tt,
		&simplepath.Finder{
			Q: &core.Q{tt.CoreSession()},
		},
		3,
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

func TestPathActionsSourceAssetsValidation(t *testing.T) {
	tt := test.Start(t).Scenario("paths")
	defer tt.Finish()
	assertions := &Assertions{tt.Assert}
	orderBookGraph := orderbook.NewOrderBookGraph()
	rh := pathFindingClient(
		tt,
		simplepath.NewInMemoryFinder(orderBookGraph),
		3,
	)

	missingSourceAccountAndAssets := make(url.Values)
	missingSourceAccountAndAssets.Add(
		"destination_asset_issuer",
		"GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN",
	)
	missingSourceAccountAndAssets.Add("destination_asset_type", "credit_alphanum4")
	missingSourceAccountAndAssets.Add("destination_asset_code", "USD")
	missingSourceAccountAndAssets.Add("destination_amount", "10")

	sourceAccountAndAssets, err := url.ParseQuery(
		missingSourceAccountAndAssets.Encode(),
	)
	tt.Assert.NoError(err)
	sourceAccountAndAssets.Add(
		"source_assets",
		"EUR:GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN",
	)
	sourceAccountAndAssets.Add(
		"source_account",
		"GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN",
	)

	tooManySourceAssets, err := url.ParseQuery(
		missingSourceAccountAndAssets.Encode(),
	)
	tt.Assert.NoError(err)
	tooManySourceAssets.Add(
		"source_assets",
		"EUR:GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN,"+
			"GBP:GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN,"+
			"USD:GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN,"+
			"SEK:GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN",
	)

	for _, testCase := range []struct {
		name            string
		q               url.Values
		expectedProblem problem.P
	}{
		{
			"both destination asset and destination account are missing",
			missingSourceAccountAndAssets,
			sourceAssetsOrSourceAccount,
		},
		{
			"both destination asset and destination account are present",
			sourceAccountAndAssets,
			sourceAssetsOrSourceAccount,
		},
		{
			"too many assets in destination_assets",
			tooManySourceAssets,
			*problem.MakeInvalidFieldProblem(
				"source_assets",
				fmt.Errorf("list of assets exceeds maximum length of 3"),
			),
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			w := rh.Get("/paths/strict-receive?" + testCase.q.Encode())
			assertions.Equal(testCase.expectedProblem.Status, w.Code)
			assertions.Problem(w.Body, testCase.expectedProblem)
		})
	}
}

func TestPathActionsDestinationAssetsValidation(t *testing.T) {
	tt := test.Start(t).Scenario("paths")
	defer tt.Finish()
	assertions := &Assertions{tt.Assert}
	orderBookGraph := orderbook.NewOrderBookGraph()
	rh := pathFindingClient(
		tt,
		simplepath.NewInMemoryFinder(orderBookGraph),
		3,
	)

	missingDestinationAccountAndAssets := make(url.Values)
	missingDestinationAccountAndAssets.Add(
		"source_asset_issuer",
		"GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN",
	)
	missingDestinationAccountAndAssets.Add(
		"source_account",
		"GARSFJNXJIHO6ULUBK3DBYKVSIZE7SC72S5DYBCHU7DKL22UXKVD7MXP",
	)
	missingDestinationAccountAndAssets.Add("source_asset_type", "credit_alphanum4")
	missingDestinationAccountAndAssets.Add("source_asset_code", "USD")
	missingDestinationAccountAndAssets.Add("source_amount", "10")

	destinationAccountAndAssets, err := url.ParseQuery(
		missingDestinationAccountAndAssets.Encode(),
	)
	tt.Assert.NoError(err)
	destinationAccountAndAssets.Add(
		"destination_assets",
		"EUR:GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN",
	)
	destinationAccountAndAssets.Add(
		"destination_account",
		"GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN",
	)

	tooManyDestinationAssets, err := url.ParseQuery(
		missingDestinationAccountAndAssets.Encode(),
	)
	tt.Assert.NoError(err)
	tooManyDestinationAssets.Add(
		"destination_assets",
		"EUR:GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN,"+
			"GBP:GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN,"+
			"USD:GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN,"+
			"SEK:GDSBCQO34HWPGUGQSP3QBFEXVTSR2PW46UIGTHVWGWJGQKH3AFNHXHXN",
	)

	for _, testCase := range []struct {
		name            string
		q               url.Values
		expectedProblem problem.P
	}{
		{
			"both destination asset and destination account are missing",
			missingDestinationAccountAndAssets,
			destinationAssetsOrDestinationAccount,
		},
		{
			"both destination asset and destination account are present",
			destinationAccountAndAssets,
			destinationAssetsOrDestinationAccount,
		},
		{
			"too many assets in destination_assets",
			tooManyDestinationAssets,
			*problem.MakeInvalidFieldProblem(
				"destination_assets",
				fmt.Errorf("list of assets exceeds maximum length of 3"),
			),
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

	coreQ := &core.Q{tt.CoreSession()}
	destinationAccount := "GA2NC4ZOXMXLVQAQQ5IQKJX47M3PKBQV2N5UV5Z4OXLQJ3CKMBA2O2YL"
	destinationAssets, _, err := coreQ.AssetsForAddress(destinationAccount)
	tt.Assert.NoError(err)

	rh := pathFindingClient(
		tt,
		simplepath.NewInMemoryFinder(orderBookGraph),
		len(destinationAssets),
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
		destinationAccount,
	)

	w := rh.Get("/paths/strict-send?" + q.Encode())
	assertions.Equal(http.StatusOK, w.Code)
	accountResponse := []horizon.Path{}
	tt.UnmarshalPage(w.Body, &accountResponse)
	assertions.Len(accountResponse, 12)

	for i, path := range accountResponse {
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
			accountResponse[i-1].DestinationAssetType == path.DestinationAssetType &&
			accountResponse[i-1].DestinationAssetCode == path.DestinationAssetCode &&
			accountResponse[i-1].DestinationAssetIssuer == path.DestinationAssetIssuer {
			previous, err := strconv.ParseFloat(accountResponse[i-1].DestinationAmount, 64)
			assertions.NoError(err)

			current, err := strconv.ParseFloat(path.DestinationAmount, 64)
			assertions.NoError(err)

			assertions.True(previous >= current)
		}
	}

	q.Del("destination_account")
	q.Add("destination_assets", assetsToURLParam(destinationAssets))
	w = rh.Get("/paths/strict-send?" + q.Encode())
	assertions.Equal(http.StatusOK, w.Code)
	assetListResponse := []horizon.Path{}
	tt.UnmarshalPage(w.Body, &assetListResponse)
	assertions.Len(assetListResponse, 12)
	tt.Assert.Equal(accountResponse, assetListResponse)
}

func assetsToURLParam(xdrAssets []xdr.Asset) string {
	var assets []string
	for _, xdrAsset := range xdrAssets {
		var assetType, code, issuer string
		xdrAsset.MustExtract(&assetType, &code, &issuer)
		if assetType == "native" {
			assets = append(assets, "native")
		} else {
			assets = append(assets, fmt.Sprintf("%s:%s", code, issuer))
		}
	}

	return strings.Join(assets, ",")
}
