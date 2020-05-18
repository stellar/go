package ticker

import (
	"time"

	horizonclient "github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/services/ticker/internal/scraper"
	"github.com/stellar/go/services/ticker/internal/tickerdb"
	"github.com/stellar/go/support/errors"
	hlog "github.com/stellar/go/support/log"
)

// RefreshOrderbookEntries updates the orderbook entries for the relevant markets that were active
// in the past 7-day interval
func RefreshOrderbookEntries(s *tickerdb.TickerSession, c *horizonclient.Client, l *hlog.Entry) error {
	sc := scraper.ScraperConfig{
		Client: c,
		Logger: l,
	}

	// Retrieve relevant markets for the past 7 days (168 hours):
	mkts, err := s.Retrieve7DRelevantMarkets()
	if err != nil {
		return errors.Wrap(err, "could not retrieve partial markets")
	}

	for _, mkt := range mkts {
		ob, err := sc.FetchOrderbookForAssets(
			mkt.BaseAssetType,
			mkt.BaseAssetCode,
			mkt.BaseAssetIssuer,
			mkt.CounterAssetType,
			mkt.CounterAssetCode,
			mkt.CounterAssetIssuer,
		)
		if err != nil {
			l.Error(errors.Wrap(err, "could not fetch orderbook for assets"))
			continue
		}

		dbOS := orderbookStatsToDBOrderbookStats(ob, mkt.BaseAssetID, mkt.CounterAssetID)
		err = s.InsertOrUpdateOrderbookStats(&dbOS, []string{"base_asset_id", "counter_asset_id"})
		if err != nil {
			l.Error(errors.Wrap(err, "could not insert orderbook stats into db"))
			continue
		}

		// Compute the orderbook stats for the reverse market.
		iob, err := sc.FetchOrderbookForAssets(
			mkt.CounterAssetType,
			mkt.CounterAssetCode,
			mkt.CounterAssetIssuer,
			mkt.BaseAssetType,
			mkt.BaseAssetCode,
			mkt.BaseAssetIssuer,
		)
		if err != nil {
			l.Error(errors.Wrap(err, "could not fetch reverse orderbook for assets"))
			continue
		}

		dbIOS := orderbookStatsToDBOrderbookStats(iob, mkt.CounterAssetID, mkt.BaseAssetID)
		err = s.InsertOrUpdateOrderbookStats(&dbIOS, []string{"base_asset_id", "counter_asset_id"})
		if err != nil {
			l.Error(errors.Wrap(err, "could not insert reverse orderbook stats into db"))
		}
	}

	return nil
}

func orderbookStatsToDBOrderbookStats(os scraper.OrderbookStats, bID, cID int32) tickerdb.OrderbookStats {
	return tickerdb.OrderbookStats{
		BaseAssetID:    bID,
		CounterAssetID: cID,
		NumBids:        os.NumBids,
		BidVolume:      os.BidVolume,
		HighestBid:     os.HighestBid,
		NumAsks:        os.NumAsks,
		AskVolume:      os.AskVolume,
		LowestAsk:      os.LowestAsk,
		Spread:         os.Spread,
		SpreadMidPoint: os.SpreadMidPoint,
		UpdatedAt:      time.Now(),
	}
}
