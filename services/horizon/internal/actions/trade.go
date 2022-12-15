package actions

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	gTime "time"

	"github.com/stellar/go/protocols/horizon"
	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/support/time"
	"github.com/stellar/go/xdr"
)

// TradeAssetsQueryParams represents the base and counter assets on trade related end-points.
type TradeAssetsQueryParams struct {
	BaseAssetType      string `schema:"base_asset_type" valid:"assetType,optional"`
	BaseAssetIssuer    string `schema:"base_asset_issuer" valid:"accountID,optional"`
	BaseAssetCode      string `schema:"base_asset_code" valid:"-"`
	CounterAssetType   string `schema:"counter_asset_type" valid:"assetType,optional"`
	CounterAssetIssuer string `schema:"counter_asset_issuer" valid:"accountID,optional"`
	CounterAssetCode   string `schema:"counter_asset_code" valid:"-"`
}

// Base returns an xdr.Asset representing the base side of the trade.
func (q TradeAssetsQueryParams) Base() (*xdr.Asset, error) {
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
func (q TradeAssetsQueryParams) Counter() (*xdr.Asset, error) {
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

// TradesQuery query struct for trades end-points
type TradesQuery struct {
	AccountID              string `schema:"account_id" valid:"accountID,optional"`
	OfferID                uint64 `schema:"offer_id" valid:"-"`
	PoolID                 string `schema:"liquidity_pool_id" valid:"sha256,optional"`
	TradeType              string `schema:"trade_type" valid:"tradeType,optional"`
	TradeAssetsQueryParams `valid:"optional"`
}

// Validate runs custom validations base and counter
func (q TradesQuery) Validate() error {
	base, err := q.Base()
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

	if base != nil && q.OfferID != 0 {
		return problem.MakeInvalidFieldProblem(
			"base_asset_type,counter_asset_type,offer_id",
			errors.New("this endpoint does not support filtering by both asset pair and offer id "),
		)
	}

	if base != nil && q.PoolID != "" {
		return problem.MakeInvalidFieldProblem(
			"base_asset_type,counter_asset_type,liquidity_pool_id",
			errors.New("this endpoint does not support filtering by both asset pair and liquidity pool id "),
		)
	}

	count, err := countNonEmpty(
		q.AccountID,
		q.OfferID,
		q.PoolID,
	)
	if err != nil {
		return problem.BadRequest
	}

	if count > 1 {
		return problem.MakeInvalidFieldProblem(
			"account_id,liquidity_pool_id,offer_id",
			errors.New(
				"Use a single filter for trades, you can only use one of account_id, liquidity_pool_id, offer_id",
			),
		)
	}

	if q.OfferID != 0 && q.TradeType == history.LiquidityPoolTrades {
		return problem.MakeInvalidFieldProblem(
			"trade_type",
			errors.Errorf("trade_type %s cannot be used with the offer_id filter", q.TradeType),
		)
	}

	if q.PoolID != "" && q.TradeType == history.OrderbookTrades {
		return problem.MakeInvalidFieldProblem(
			"trade_type",
			errors.Errorf("trade_type %s cannot be used with the liquidity_pool_id filter", q.TradeType),
		)
	}
	return nil
}

// GetTradesHandler is the action handler for all end-points returning a list of trades.
type GetTradesHandler struct {
	LedgerState *ledger.State
	CoreStateGetter
}

// GetResourcePage returns a page of trades.
func (handler GetTradesHandler) GetResourcePage(w HeaderWriter, r *http.Request) ([]hal.Pageable, error) {
	ctx := r.Context()

	pq, err := GetPageQuery(handler.LedgerState, r)
	if err != nil {
		return nil, err
	}

	err = validateCursorWithinHistory(handler.LedgerState, pq)
	if err != nil {
		return nil, err
	}

	qp := TradesQuery{}
	if err = getParams(&qp, r); err != nil {
		return nil, err
	}
	if qp.TradeType == "" {
		qp.TradeType = history.AllTrades
	}

	historyQ, err := horizonContext.HistoryQFromRequest(r)
	if err != nil {
		return nil, err
	}

	var records []history.Trade
	var baseAsset, counterAsset *xdr.Asset
	baseAsset, err = qp.Base()
	if err != nil {
		return nil, err
	}

	if baseAsset != nil {
		counterAsset, err = qp.Counter()
		if err != nil {
			return nil, err
		}

		records, err = historyQ.GetTradesForAssets(ctx, pq, qp.AccountID, qp.TradeType, *baseAsset, *counterAsset)
	} else if qp.OfferID != 0 {
		records, err = historyQ.GetTradesForOffer(ctx, pq, int64(qp.OfferID))
	} else if qp.PoolID != "" {
		records, err = historyQ.GetTradesForLiquidityPool(ctx, pq, qp.PoolID)
	} else {
		records, err = historyQ.GetTrades(ctx, pq, qp.AccountID, qp.TradeType)
	}
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

// TradeAggregationsQuery query struct for trade_aggregations end-point
type TradeAggregationsQuery struct {
	OffsetFilter           uint64      `schema:"offset" valid:"-"`
	StartTimeFilter        time.Millis `schema:"start_time" valid:"-"`
	EndTimeFilter          time.Millis `schema:"end_time" valid:"-"`
	ResolutionFilter       uint64      `schema:"resolution" valid:"-"`
	TradeAssetsQueryParams `valid:"optional"`
}

// Validate runs validations on tradeAggregationsQuery
func (q TradeAggregationsQuery) Validate() error {
	base, err := q.Base()
	if err != nil {
		return err
	}
	if base == nil {
		return problem.MakeInvalidFieldProblem(
			"base_asset_type",
			errors.New("Missing required field"),
		)
	}
	counter, err := q.Counter()
	if err != nil {
		return err
	}
	if counter == nil {
		return problem.MakeInvalidFieldProblem(
			"counter_asset_type",
			errors.New("Missing required field"),
		)
	}

	//check if resolution is legal
	resolutionDuration := gTime.Duration(q.ResolutionFilter) * gTime.Millisecond
	if history.StrictResolutionFiltering {
		if _, ok := history.AllowedResolutions[resolutionDuration]; !ok {
			return problem.MakeInvalidFieldProblem(
				"resolution",
				errors.New("illegal or missing resolution. "+
					"allowed resolutions are: 1 minute (60000), 5 minutes (300000), 15 minutes (900000), 1 hour (3600000), "+
					"1 day (86400000) and 1 week (604800000)"),
			)
		}
	}
	// check if offset is legal
	offsetDuration := gTime.Duration(q.OffsetFilter) * gTime.Millisecond
	if offsetDuration%gTime.Hour != 0 || offsetDuration >= gTime.Hour*24 || offsetDuration > resolutionDuration {
		return problem.MakeInvalidFieldProblem(
			"offset",
			errors.New("illegal or missing offset. offset must be a multiple of an"+
				" hour, less than or equal to the resolution, and less than 24 hours"),
		)
	}

	return nil
}

// GetTradeAggregationsHandler is the action handler for trade_aggregations
type GetTradeAggregationsHandler struct {
	LedgerState *ledger.State
	CoreStateGetter
}

// GetResource returns a page of trade aggregations
func (handler GetTradeAggregationsHandler) GetResource(w HeaderWriter, r *http.Request) (interface{}, error) {
	ctx := r.Context()
	pq, err := GetPageQuery(handler.LedgerState, r)
	if err != nil {
		return nil, err
	}
	err = validateCursorWithinHistory(handler.LedgerState, pq)
	if err != nil {
		return nil, err
	}
	qp := TradeAggregationsQuery{}
	if err = getParams(&qp, r); err != nil {
		return nil, err
	}

	historyQ, err := horizonContext.HistoryQFromRequest(r)
	if err != nil {
		return nil, err
	}

	records, err := handler.fetchRecords(ctx, historyQ, qp, pq)
	if err != nil {
		return nil, err
	}
	var aggregations []hal.Pageable
	for _, record := range records {
		var res horizon.TradeAggregation
		err = resourceadapter.PopulateTradeAggregation(ctx, &res, record)
		if err != nil {
			return nil, err
		}
		aggregations = append(aggregations, res)
	}

	return handler.buildPage(r, aggregations)
}

func (handler GetTradeAggregationsHandler) fetchRecords(ctx context.Context, historyQ *history.Q, qp TradeAggregationsQuery, pq db2.PageQuery) ([]history.TradeAggregation, error) {
	baseAsset, err := qp.Base()
	if err != nil {
		return nil, err
	}

	baseAssetID, err := historyQ.GetAssetID(ctx, *baseAsset)
	if err != nil {
		p := problem.BadRequest
		if historyQ.NoRows(err) {
			p = problem.NotFound
			err = errors.New("not found")
		}
		return nil, problem.NewProblemWithInvalidField(
			p,
			"base_asset",
			err,
		)
	}

	counterAsset, err := qp.Counter()
	if err != nil {
		return nil, err
	}

	counterAssetID, err := historyQ.GetAssetID(ctx, *counterAsset)
	if err != nil {
		p := problem.BadRequest
		if historyQ.NoRows(err) {
			p = problem.NotFound
			err = errors.New("not found")
		}

		return nil, problem.NewProblemWithInvalidField(
			p,
			"counter_asset",
			err,
		)
	}

	//initialize the query builder with required params
	tradeAggregationsQ, err := historyQ.GetTradeAggregationsQ(
		baseAssetID,
		counterAssetID,
		int64(qp.ResolutionFilter),
		int64(qp.OffsetFilter),
		pq,
	)
	if err != nil {
		return nil, err
	}

	//set time range if supplied
	if !qp.StartTimeFilter.IsNil() {
		tradeAggregationsQ, err = tradeAggregationsQ.WithStartTime(qp.StartTimeFilter)
		if err != nil {
			return nil, problem.MakeInvalidFieldProblem(
				"start_time",
				errors.New(
					"illegal start time. adjusted start time must "+
						"be less than the provided end time if the end time is greater than 0",
				),
			)
		}
	}
	if !qp.EndTimeFilter.IsNil() {
		tradeAggregationsQ, err = tradeAggregationsQ.WithEndTime(qp.EndTimeFilter)
		if err != nil {
			return nil, problem.MakeInvalidFieldProblem(
				"end_time",
				errors.New(
					"illegal end time. adjusted end time "+
						"must be greater than the offset and greater than the provided start time",
				),
			)
		}
	}

	var records []history.TradeAggregation
	err = historyQ.Select(ctx, &records, tradeAggregationsQ.GetSql())
	if err != nil {
		return nil, err
	}
	return records, err
}

// BuildPage builds a custom hal page for this handler
func (handler GetTradeAggregationsHandler) buildPage(r *http.Request, records []hal.Pageable) (hal.Page, error) {
	ctx := r.Context()
	pageQuery, err := GetPageQuery(handler.LedgerState, r, DisableCursorValidation)
	if err != nil {
		return hal.Page{}, err
	}
	qp := TradeAggregationsQuery{}
	if err = getParams(&qp, r); err != nil {
		return hal.Page{}, err
	}

	page := hal.Page{
		Cursor: pageQuery.Cursor,
		Order:  pageQuery.Order,
		Limit:  pageQuery.Limit,
	}
	page.Init()

	for _, record := range records {
		page.Add(record)
	}

	newURL := FullURL(ctx)
	q := newURL.Query()

	page.Links.Self = hal.NewLink(newURL.String())

	//adjust time range for next page
	if uint64(len(records)) == 0 {
		page.Links.Next = page.Links.Self
	} else {
		lastRecord := records[len(records)-1]

		lastRecordTA, ok := lastRecord.(horizon.TradeAggregation)
		if !ok {
			panic(fmt.Sprintf("Unknown type: %T", lastRecord))
		}
		timestamp := lastRecordTA.Timestamp

		if page.Order == "asc" {
			newStartTime := timestamp + int64(qp.ResolutionFilter)
			if newStartTime >= qp.EndTimeFilter.ToInt64() {
				newStartTime = qp.EndTimeFilter.ToInt64()
			}
			q.Set("start_time", strconv.FormatInt(newStartTime, 10))
			newURL.RawQuery = q.Encode()
			page.Links.Next = hal.NewLink(newURL.String())
		} else { //desc
			newEndTime := timestamp
			if newEndTime <= qp.StartTimeFilter.ToInt64() {
				newEndTime = qp.StartTimeFilter.ToInt64()
			}
			q.Set("end_time", strconv.FormatInt(newEndTime, 10))
			newURL.RawQuery = q.Encode()
			page.Links.Next = hal.NewLink(newURL.String())
		}
	}

	return page, nil
}
