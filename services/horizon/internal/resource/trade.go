package resource

import (
	"fmt"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/httpx"
	"github.com/stellar/go/services/horizon/internal/render/hal"
	"golang.org/x/net/context"
	"github.com/stellar/go/price"
)

// Populate fills out the details of a trade using a row from the history_trades
// table.
func (res *Trade) Populate(
	ctx context.Context,
	row history.Trade,
) (err error) {
	res.ID = row.PagingToken()
	res.PT = row.PagingToken()
	res.OfferID = fmt.Sprintf("%d", row.OfferID)
	res.BaseAccount = row.BaseAccount
	res.BaseAssetType = row.BaseAssetType
	res.BaseAssetCode = row.BaseAssetCode
	res.BaseAssetIssuer = row.BaseAssetIssuer
	res.BaseAmount = amount.String(row.BaseAmount)
	res.CounterAccount = row.CounterAccount
	res.CounterAssetType = row.CounterAssetType
	res.CounterAssetCode = row.CounterAssetCode
	res.CounterAssetIssuer = row.CounterAssetIssuer
	res.CounterAmount = amount.String(row.CounterAmount)
	res.LedgerCloseTime = row.LedgerCloseTime
	res.BaseIsSeller = row.BaseIsSeller
	res.populateLinks(ctx, row.HistoryOperationID)
	return
}

// PagingToken implementation for hal.Pageable
func (res Trade) PagingToken() string {
	return res.PT
}

func (res *Trade) populateLinks(
	ctx context.Context,
	opid int64,
) {
	lb := hal.LinkBuilder{httpx.BaseURL(ctx)}
	res.Links.Base = lb.Link("/accounts", res.BaseAccount)
	res.Links.Counter = lb.Link("/accounts", res.CounterAccount)
	res.Links.Operation = lb.Link(
		"/operations",
		fmt.Sprintf("%d", opid),
	)
}

// Populate fills out the details of a trade using a row from the history_trades
// table.
func (res *TradeAggregation) Populate(
	ctx context.Context,
	row history.TradeAggregation,
) (err error) {
	res.Timestamp = row.Timestamp
	res.TradeCount = row.TradeCount
	res.BaseVolume = amount.StringFromInt64(row.BaseVolume)
	res.CounterVolume = amount.StringFromInt64(row.CounterVolume)
	res.Average = price.StringFromFloat64(row.Average)
	res.High = price.StringFromFloat64(row.High)
	res.Low = price.StringFromFloat64(row.Low)
	res.Open = price.StringFromFloat64(row.Open)
	res.Close = price.StringFromFloat64(row.Close)
	return
}

// PagingToken implementation for hal.Pageable. Not actually used
func (res TradeAggregation) PagingToken() string {
	return string(res.Timestamp)
}
