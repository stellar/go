package actions

import (
	"fmt"
	"net/http"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
)

// TradesQuery query struct for trades end-points
type TradesQuery struct {
	AccountID          string `schema:"account_id" valid:"accountID,optional"`
	OfferID            uint64 `schema:"offer_id" valid:"-"`
	BaseAssetType      string `schema:"base_asset_type" valid:"assetType,optional"`
	BaseAssetIssuer    string `schema:"base_asset_issuer" valid:"accountID,optional"`
	BaseAssetCode      string `schema:"base_asset_code" valid:"-"`
	CounterAssetType   string `schema:"counter_asset_type" valid:"assetType,optional"`
	CounterAssetIssuer string `schema:"counter_asset_issuer" valid:"accountID,optional"`
	CounterAssetCode   string `schema:"counter_asset_code" valid:"-"`
}

// Base returns an xdr.Asset representing the base side of the trade.
func (q TradesQuery) Base() (*xdr.Asset, error) {
	if len(q.BaseAssetType) == 0 {
		return nil, nil
	}

	base, err := xdr.BuildAsset(
		q.BaseAssetType,
		q.BaseAssetIssuer,
		q.BaseAssetCode,
	)

	if err != nil {
		return nil, problem.MakeInvalidFieldProblem(
			"base_asset",
			errors.New(fmt.Sprintf("invalid base_asset: %s", err.Error())),
		)
	}

	return &base, nil
}

// Counter returns an *xdr.Asset representing the counter asset side of the trade.
func (q TradesQuery) Counter() (*xdr.Asset, error) {
	if len(q.CounterAssetType) == 0 {
		return nil, nil
	}

	counter, err := xdr.BuildAsset(
		q.CounterAssetType,
		q.CounterAssetIssuer,
		q.CounterAssetCode,
	)

	if err != nil {
		return nil, problem.MakeInvalidFieldProblem(
			"counter_asset",
			errors.New(fmt.Sprintf("invalid counter_asset: %s", err.Error())),
		)
	}

	return &counter, nil
}

// Validate runs custom validations base and counter
func (q TradesQuery) Validate() error {
	err := ValidateAssetParams(q.BaseAssetType, q.BaseAssetCode, q.BaseAssetIssuer, "base_")
	if err != nil {
		return err
	}
	base, err := q.Base()
	if err != nil {
		return err
	}

	err = ValidateAssetParams(q.CounterAssetType, q.CounterAssetCode, q.CounterAssetIssuer, "counter_")
	if err != nil {
		return err
	}
	counter, err := q.Counter()
	if err != nil {
		return err
	}

	if (base != nil && counter == nil) || (base == nil && counter != nil) {
		return problem.MakeInvalidFieldProblem(
			"base_asset_type,counter_asset_type",
			errors.New("this endpoint supports asset pairs but only one asset supplied"),
		)
	}

	return nil
}

// GetTradesHandler is the action handler for all end-points returning a list of trades.
type GetTradesHandler struct {
}

// GetResourcePage returns a page of trades.
func (handler GetTradesHandler) GetResourcePage(w HeaderWriter, r *http.Request) ([]hal.Pageable, error) {
	ctx := r.Context()

	pq, err := GetPageQuery(r)
	if err != nil {
		return nil, err
	}

	err = ValidateCursorWithinHistory(pq)
	if err != nil {
		return nil, err
	}

	qp := TradesQuery{}
	err = GetParams(&qp, r)
	if err != nil {
		return nil, err
	}

	historyQ, err := HistoryQFromRequest(r)
	if err != nil {
		return nil, err
	}

	trades := historyQ.Trades()

	if qp.AccountID != "" {
		trades.ForAccount(qp.AccountID)
	}

	baseAsset, err := qp.Base()
	if err != nil {
		return nil, err
	}

	if baseAsset != nil {
		baseAssetID, err2 := historyQ.GetAssetID(*baseAsset)
		if err2 != nil {
			return nil, err2
		}

		counterAsset, err2 := qp.Counter()
		if err2 != nil {
			return nil, err2
		}

		counterAssetID, err2 := historyQ.GetAssetID(*counterAsset)
		if err2 != nil {
			return nil, err2
		}
		trades = historyQ.TradesForAssetPair(baseAssetID, counterAssetID)
	}

	if qp.OfferID != 0 {
		trades = trades.ForOffer(int64(qp.OfferID))
	}

	var records []history.Trade
	err = trades.Page(pq).Select(&records)
	if err != nil {
		return nil, err
	}
	var response []hal.Pageable

	for _, record := range records {
		var res horizon.Trade
		resourceadapter.PopulateTrade(ctx, &res, record)
		response = append(response, res)
	}

	return response, nil
}
