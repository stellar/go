package actions

import (
	"context"
	"fmt"

	"github.com/stellar/go/amount"
	protocol "github.com/stellar/go/protocols/horizon"
	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest/processors"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/xdr"
)

// Populate fills out the details of a trade using a row from the history_trades
// table. This can be removed in release after P18 upgrade.
func PopulateTradeP17(
	ctx context.Context,
	dest *TradeP17,
	row history.Trade,
) {
	dest.ID = row.PagingToken()
	dest.PT = row.PagingToken()
	dest.BaseOfferID = ""
	if row.BaseOfferID.Valid {
		dest.BaseOfferID = fmt.Sprintf("%d", row.BaseOfferID.Int64)
	}
	dest.BaseAccount = row.BaseAccount.String // Always Valid in P17
	dest.BaseAssetType = row.BaseAssetType
	dest.BaseAssetCode = row.BaseAssetCode
	dest.BaseAssetIssuer = row.BaseAssetIssuer
	dest.BaseAmount = amount.String(xdr.Int64(row.BaseAmount))
	dest.CounterOfferID = ""
	if row.CounterOfferID.Valid {
		dest.CounterOfferID = fmt.Sprintf("%d", row.CounterOfferID.Int64)
	}
	dest.CounterAccount = row.CounterAccount.String // Always Valid in P17
	dest.CounterAssetType = row.CounterAssetType
	dest.CounterAssetCode = row.CounterAssetCode
	dest.CounterAssetIssuer = row.CounterAssetIssuer
	dest.CounterAmount = amount.String(xdr.Int64(row.CounterAmount))
	dest.LedgerCloseTime = row.LedgerCloseTime
	dest.BaseIsSeller = row.BaseIsSeller

	_, counterOfferIDType := processors.DecodeOfferID(row.CounterOfferID.Int64)
	if counterOfferIDType == processors.CoreOfferIDType {
		dest.OfferID = fmt.Sprintf("%d", row.CounterOfferID.Int64)
	} else {
		dest.OfferID = fmt.Sprintf("%d", row.BaseOfferID.Int64)
	}

	if row.HasPrice() {
		dest.Price = &protocol.Price{
			N: int32(row.PriceN.Int64),
			D: int32(row.PriceD.Int64),
		}
	}

	populateTradeLinks(ctx, dest, row.HistoryOperationID)
}

func populateTradeLinks(
	ctx context.Context,
	dest *TradeP17,
	opid int64,
) {
	lb := hal.LinkBuilder{horizonContext.BaseURL(ctx)}
	dest.Links.Base = lb.Link("/accounts", dest.BaseAccount)
	dest.Links.Counter = lb.Link("/accounts", dest.CounterAccount)
	dest.Links.Operation = lb.Link(
		"/operations",
		fmt.Sprintf("%d", opid),
	)
}
