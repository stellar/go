package ticker

import (
	"time"

	horizonclient "github.com/stellar/go/exp/clients/horizon"
	"github.com/stellar/go/exp/ticker/internal/scraper"
)

// BackfillTrades ingest the most recent trades (since "since") directly from Horizon
// into the database.
func BackfillTrades(numDays int, limit int) (err error) {
	c := horizonclient.DefaultPublicNetClient
	now := time.Now()
	since := now.AddDate(0, 0, -numDays)

	err = scraper.FetchAllTrades(c, since, limit)
	return
}
