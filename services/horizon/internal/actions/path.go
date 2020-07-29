package actions

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/protocols/horizon"
	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/paths"
	horizonProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/services/horizon/internal/simplepath"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
)

// FindPathsHandler is the http handler for the find payment paths endpoint
type FindPathsHandler struct {
	StaleThreshold       uint
	MaxPathLength        uint
	SetLastLedgerHeader  bool
	MaxAssetsParamLength int
	PathFinder           paths.Finder
}

// StrictReceivePathsQuery query struct for paths/strict-send end-point
type StrictReceivePathsQuery struct {
	SourceAssets           string `schema:"source_assets" valid:"-"`
	SourceAccount          string `schema:"source_account" valid:"accountID,optional"`
	DestinationAccount     string `schema:"destination_account" valid:"accountID,optional"`
	DestinationAssetType   string `schema:"destination_asset_type" valid:"assetType"`
	DestinationAssetIssuer string `schema:"destination_asset_issuer" valid:"accountID,optional"`
	DestinationAssetCode   string `schema:"destination_asset_code" valid:"-"`
	DestinationAmount      string `schema:"destination_amount" valid:"amount"`
}

// Assets returns a list of xdr.Asset
func (q StrictReceivePathsQuery) Assets() ([]xdr.Asset, error) {
	return xdr.BuildAssets(q.SourceAssets)
}

// Amount returns source amount
func (q StrictReceivePathsQuery) Amount() xdr.Int64 {
	parsed, err := amount.Parse(q.DestinationAmount)
	if err != nil {
		panic(err)
	}
	return parsed
}

// DestinationAsset returns an xdr.Asset
func (q StrictReceivePathsQuery) DestinationAsset() xdr.Asset {
	asset, err := xdr.BuildAsset(
		q.DestinationAssetType,
		q.DestinationAssetIssuer,
		q.DestinationAssetCode,
	)

	if err != nil {
		panic(err)
	}

	return asset
}

// URITemplate returns a rfc6570 URI template for the query struct
func (q StrictReceivePathsQuery) URITemplate() string {
	return "/paths/strict-receive{?" + strings.Join(getURIParams(&q, false), ",") + "}"
}

// Validate runs custom validations.
func (q StrictReceivePathsQuery) Validate() error {
	if (len(q.SourceAccount) > 0) == (len(q.SourceAssets) > 0) {
		return SourceAssetsOrSourceAccountProblem
	}

	err := validateAssetParams(
		q.DestinationAssetType,
		q.DestinationAssetCode,
		q.DestinationAssetIssuer,
		"destination_",
	)

	if err != nil {
		return err
	}

	_, err = q.Assets()

	if err != nil {
		return problem.MakeInvalidFieldProblem(
			"source_assets",
			err,
		)
	}

	return nil
}

// SourceAssetsOrSourceAccountProblem custom error where source assets or account is required
var SourceAssetsOrSourceAccountProblem = problem.P{
	Type:   "bad_request",
	Title:  "Bad Request",
	Status: http.StatusBadRequest,
	Detail: "The request requires either a list of source assets or a source account. " +
		"Both fields cannot be present.",
}

// GetResource finds a list of strict receive paths
func (handler FindPathsHandler) GetResource(w HeaderWriter, r *http.Request) (interface{}, error) {
	var err error
	ctx := r.Context()
	qp := StrictReceivePathsQuery{}

	if err = getParams(&qp, r); err != nil {
		return nil, err
	}

	query := paths.Query{}
	query.DestinationAmount = qp.Amount()
	sourceAccount := qp.SourceAccount
	query.SourceAssets, _ = qp.Assets()

	if len(query.SourceAssets) > handler.MaxAssetsParamLength {
		return nil, problem.MakeInvalidFieldProblem(
			"source_assets",
			fmt.Errorf("list of assets exceeds maximum length of %d", handler.MaxPathLength),
		)
	}
	query.DestinationAsset = qp.DestinationAsset()
	if sourceAccount != "" {
		sourceAccount := xdr.MustAddress(sourceAccount)
		query.SourceAccount = &sourceAccount
		query.ValidateSourceBalance = true
		query.SourceAssets, query.SourceAssetBalances, err = assetsForAddress(r, query.SourceAccount.Address())
		if err != nil {
			return nil, err
		}
	} else {
		for range query.SourceAssets {
			query.SourceAssetBalances = append(query.SourceAssetBalances, 0)
		}
	}

	records := []paths.Path{}
	if len(query.SourceAssets) > 0 {
		var lastIngestedLedger uint32
		records, lastIngestedLedger, err = handler.PathFinder.Find(query, handler.MaxPathLength)
		if err == simplepath.ErrEmptyInMemoryOrderBook {
			err = horizonProblem.StillIngesting
		}
		if err != nil {
			return nil, err
		}

		if handler.SetLastLedgerHeader {
			// To make the Last-Ledger header consistent with the response content,
			// we need to extract it from the ledger and not the DB.
			// Thus, we overwrite the header if it was previously set.
			SetLastLedgerHeader(w, lastIngestedLedger)
		}
	}

	return renderPaths(ctx, records)
}

func renderPaths(ctx context.Context, records []paths.Path) (hal.BasePage, error) {
	var page hal.BasePage
	page.Init()
	for _, p := range records {
		var res horizon.Path
		if err := resourceadapter.PopulatePath(ctx, &res, p); err != nil {
			return hal.BasePage{}, err
		}
		page.Add(res)
	}
	return page, nil
}

// FindFixedPathsHandler is the http handler for the find fixed payment paths endpoint
// Fixed payment paths are payment paths where both the source and destination asset are fixed
type FindFixedPathsHandler struct {
	MaxPathLength        uint
	MaxAssetsParamLength int
	SetLastLedgerHeader  bool
	PathFinder           paths.Finder
}

// DestinationAssetsOrDestinationAccountProblem custom error where destination asserts or accounts are required
var DestinationAssetsOrDestinationAccountProblem = problem.P{
	Type:   "bad_request",
	Title:  "Bad Request",
	Status: http.StatusBadRequest,
	Detail: "The request requires either a list of destination assets or a destination account. " +
		"Both fields cannot be present.",
}

// FindFixedPathsQuery query struct for paths/strict-send end-point
type FindFixedPathsQuery struct {
	DestinationAccount string `schema:"destination_account" valid:"accountID,optional"`
	DestinationAssets  string `schema:"destination_assets" valid:"-"`
	SourceAssetType    string `schema:"source_asset_type" valid:"assetType"`
	SourceAssetIssuer  string `schema:"source_asset_issuer" valid:"accountID,optional"`
	SourceAssetCode    string `schema:"source_asset_code" valid:"-"`
	SourceAmount       string `schema:"source_amount" valid:"amount"`
}

// URITemplate returns a rfc6570 URI template for the query struct
func (q FindFixedPathsQuery) URITemplate() string {
	return "/paths/strict-send{?" + strings.Join(getURIParams(&q, false), ",") + "}"
}

// Validate runs custom validations.
func (q FindFixedPathsQuery) Validate() error {
	if (len(q.DestinationAccount) > 0) == (len(q.DestinationAssets) > 0) {
		return DestinationAssetsOrDestinationAccountProblem
	}

	err := validateAssetParams(
		q.SourceAssetType,
		q.SourceAssetCode,
		q.SourceAssetIssuer,
		"source_",
	)

	if err != nil {
		return err
	}

	_, err = q.Assets()

	if err != nil {
		return problem.MakeInvalidFieldProblem(
			"destination_assets",
			err,
		)
	}

	return nil
}

// Assets returns a list of xdr.Asset
func (q FindFixedPathsQuery) Assets() ([]xdr.Asset, error) {
	return xdr.BuildAssets(q.DestinationAssets)
}

// Amount returns source amount
func (q FindFixedPathsQuery) Amount() xdr.Int64 {
	parsed, err := amount.Parse(q.SourceAmount)
	if err != nil {
		panic(err)
	}
	return parsed
}

// SourceAsset returns an xdr.Asset
func (q FindFixedPathsQuery) SourceAsset() xdr.Asset {
	asset, err := xdr.BuildAsset(
		q.SourceAssetType,
		q.SourceAssetIssuer,
		q.SourceAssetCode,
	)

	if err != nil {
		panic(err)
	}

	return asset
}

// GetResource returns a list of strict send paths
func (handler FindFixedPathsHandler) GetResource(w HeaderWriter, r *http.Request) (interface{}, error) {
	var err error
	ctx := r.Context()
	qp := FindFixedPathsQuery{}

	if err = getParams(&qp, r); err != nil {
		return nil, err
	}

	destinationAccount := qp.DestinationAccount
	destinationAssets, _ := qp.Assets()

	if len(destinationAssets) > handler.MaxAssetsParamLength {
		return nil, problem.MakeInvalidFieldProblem(
			"destination_assets",
			fmt.Errorf("list of assets exceeds maximum length of %d", handler.MaxPathLength),
		)
	}

	if destinationAccount != "" {
		destinationAssets, _, err = assetsForAddress(r, destinationAccount)
		if err != nil {
			return nil, err
		}
	}

	sourceAsset := qp.SourceAsset()
	amountToSpend := qp.Amount()

	records := []paths.Path{}
	if len(destinationAssets) > 0 {
		var lastIngestedLedger uint32
		records, lastIngestedLedger, err = handler.PathFinder.FindFixedPaths(
			sourceAsset,
			amountToSpend,
			destinationAssets,
			handler.MaxPathLength,
		)
		if err == simplepath.ErrEmptyInMemoryOrderBook {
			err = horizonProblem.StillIngesting
		}
		if err != nil {
			return nil, err
		}

		if handler.SetLastLedgerHeader {
			// To make the Last-Ledger header consistent with the response content,
			// we need to extract it from the ledger and not the DB.
			// Thus, we overwrite the header if it was previously set.
			SetLastLedgerHeader(w, lastIngestedLedger)
		}
	}

	return renderPaths(ctx, records)
}

func assetsForAddress(r *http.Request, addy string) ([]xdr.Asset, []xdr.Int64, error) {
	historyQ, err := horizonContext.HistoryQFromRequest(r)
	if err != nil {
		return nil, nil, err
	}
	return historyQ.AssetsForAddress(addy)
}
