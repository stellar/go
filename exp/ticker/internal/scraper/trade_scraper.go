package scraper

import (
	"context"
	"fmt"
	"time"

	horizonclient "github.com/stellar/go/exp/clients/horizon"
	hProtocol "github.com/stellar/go/protocols/horizon"
)

// checkRecords check if a list of records contains entries older than minTime. If it does,
// it will return a filtered page with only the passing records and lastPage = true.
func checkRecords(trades []hProtocol.Trade, minTime time.Time) (lastPage bool, cleanTrades []hProtocol.Trade) {
	lastPage = false
	for _, t := range trades {
		if t.LedgerCloseTime.After(minTime) {
			cleanTrades = append(cleanTrades, t)
		} else {
			fmt.Println("Reached entries older than the acceptable time range:", t.LedgerCloseTime)
			lastPage = true
			return
		}
	}
	return
}

// retrieveTrades retrieves trades from the Horizon API for the last timeDelta period.
// If limit = 0, will fetch all trades within that period.
func retrieveTrades(c *horizonclient.Client, since time.Time, limit int) (trades []hProtocol.Trade, err error) {
	r := horizonclient.TradeRequest{Limit: 200, Order: horizonclient.OrderDesc}

	tradesPage, err := c.Trades(r)
	if err != nil {
		return
	}

	fmt.Println("Fetching trades from Horizon")

	for tradesPage.Links.Next.Href != tradesPage.Links.Self.Href {
		// Enforcing time boundaries:
		last, cleanTrades := checkRecords(tradesPage.Embedded.Records, since)
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
		fmt.Println("Cursor currently at:", n)
		r.Cursor = n
		tradesPage, err = c.Trades(r)
		if err != nil {
			return trades, err
		}
	}

	fmt.Println("Last close time ingested:", trades[len(trades)-1].LedgerCloseTime)
	fmt.Printf("Fetched: %d trades\n", len(trades))
	return
}

// streamTrades streams trades directly from horizon and calls the handler function
// whenever a new trade appears.
func streamTrades(ctx context.Context, c *horizonclient.Client, h horizonclient.TradeHandler) error {
	r := horizonclient.TradeRequest{
		Limit:  200,
		Cursor: "now",
	}

	return r.StreamTrades(ctx, c, h)
}
