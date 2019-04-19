package scraper

import (
	"time"

	"github.com/stellar/go/exp/ticker/internal/utils"

	horizonclient "github.com/stellar/go/exp/clients/horizon"
	hProtocol "github.com/stellar/go/protocols/horizon"
)

// checkRecords check if a list of records contains entries older than minTime. If it does,
// it will return a filtered page with only the passing records and lastPage = true.
func (c *ScraperConfig) checkRecords(trades []hProtocol.Trade, minTime time.Time) (lastPage bool, cleanTrades []hProtocol.Trade) {
	lastPage = false
	for _, t := range trades {
		if t.LedgerCloseTime.After(minTime) {
			normalizeTradeAssets(&t)
			cleanTrades = append(cleanTrades, t)
		} else {
			c.Logger.Debugln("Reached entries older than the acceptable time range:", t.LedgerCloseTime)
			lastPage = true
			return
		}
	}
	return
}

// retrieveTrades retrieves trades from the Horizon API for the last timeDelta period.
// If limit = 0, will fetch all trades within that period.
func (c *ScraperConfig) retrieveTrades(since time.Time, limit int) (trades []hProtocol.Trade, err error) {
	r := horizonclient.TradeRequest{Limit: 200, Order: horizonclient.OrderDesc}

	tradesPage, err := c.Client.Trades(r)
	if err != nil {
		return
	}

	for tradesPage.Links.Next.Href != tradesPage.Links.Self.Href {
		// Enforcing time boundaries:
		last, cleanTrades := c.checkRecords(tradesPage.Embedded.Records, since)
		if last {
			trades = append(trades, cleanTrades...)
			break
		} else {
			trades = append(trades, tradesPage.Embedded.Records...)
		}

		// Enforcing limit of results:
		if limit != 0 {
			numTrades := len(trades)
			if numTrades >= limit {
				diff := numTrades - limit
				trades = trades[0 : numTrades-diff]
				break
			}
		}

		// Finding next page's params:
		nextURL := tradesPage.Links.Next.Href
		n, err := nextCursor(nextURL)
		if err != nil {
			return trades, err
		}
		c.Logger.Debugln("Cursor currently at:", n)
		r.Cursor = n
		tradesPage, err = c.Client.Trades(r)
		if err != nil {
			return trades, err
		}
	}

	return
}

// streamTrades streams trades directly from horizon and calls the handler function
// whenever a new trade appears.
func (c *ScraperConfig) streamTrades(h horizonclient.TradeHandler, cursor string) error {
	if cursor == "" {
		cursor = "now"
	}

	r := horizonclient.TradeRequest{
		Limit:  200,
		Cursor: cursor,
	}

	return r.StreamTrades(*c.Ctx, c.Client, h)
}

// normalizeTradeAssets enforces the following rules:
// 1. native asset type refers to a "XLM" code and a "native" issuer
// 2. native is always the base asset (and if not, base and counter are swapped)
// 3. when trades are between two non-native, the base is the asset whose string
// comes first alphabetically.
func normalizeTradeAssets(t *hProtocol.Trade) {
	addNativeData(t)
	if t.BaseAssetType == "native" {
		return
	}
	if t.CounterAssetType == "native" {
		reverseAssets(t)
		return
	}
	bAssetString := utils.GetAssetString(t.BaseAssetType, t.BaseAssetCode, t.BaseAssetIssuer)
	cAssetString := utils.GetAssetString(t.CounterAssetType, t.CounterAssetCode, t.CounterAssetIssuer)
	if bAssetString > cAssetString {
		reverseAssets(t)
	}
}

// addNativeData adds additional fields when one of the assets is native.
func addNativeData(trade *hProtocol.Trade) {
	if trade.BaseAssetType == "native" {
		trade.BaseAssetCode = "XLM"
		trade.BaseAssetIssuer = "native"
	}

	if trade.CounterAssetType == "native" {
		trade.CounterAssetCode = "XLM"
		trade.CounterAssetIssuer = "native"
	}
}

// reverseAssets swaps out the base and counter assets of a trade.
func reverseAssets(trade *hProtocol.Trade) {
	trade.BaseAmount, trade.CounterAmount = trade.CounterAmount, trade.BaseAmount
	trade.BaseAccount, trade.CounterAccount = trade.CounterAccount, trade.BaseAccount
	trade.BaseAssetCode, trade.CounterAssetCode = trade.CounterAssetCode, trade.BaseAssetCode
	trade.BaseAssetType, trade.CounterAssetType = trade.CounterAssetType, trade.BaseAssetType
	trade.BaseAssetIssuer, trade.CounterAssetIssuer = trade.CounterAssetIssuer, trade.BaseAssetIssuer

	trade.BaseIsSeller = !trade.BaseIsSeller
	trade.Price.N, trade.Price.D = trade.Price.D, trade.Price.N
}
