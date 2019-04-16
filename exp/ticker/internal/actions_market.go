package ticker

import (
	"encoding/json"
	"time"

	"github.com/stellar/go/exp/ticker/internal/tickerdb"
	"github.com/stellar/go/exp/ticker/internal/utils"
	hlog "github.com/stellar/go/support/log"
)

// GenerateMarketSummaryFile generates a MarketSummary with the statistics for all
// valid markets within the database and outputs it to <filename>.
func GenerateMarketSummaryFile(s *tickerdb.TickerSession, l *hlog.Entry, filename string) error {
	l.Infoln("Generating market data...")
	marketSummary, err := GenerateMarketSummary(s)
	if err != nil {
		return err
	}
	l.Infoln("Market data successfully generated!")

	jsonMkt, err := json.MarshalIndent(marketSummary, "", "    ")
	if err != nil {
		return err
	}

	l.Infoln("Writing market data to: ", filename)
	numBytes, err := utils.WriteJSONToFile(jsonMkt, filename)
	if err != nil {
		return err
	}
	l.Infof("Wrote %d bytes to %s\n", numBytes, filename)
	return nil
}

// GenerateMarketSummary outputs a MarketSummary with the statistics for all
// valid markets within the database.
func GenerateMarketSummary(s *tickerdb.TickerSession) (ms MarketSummary, err error) {
	var marketStatsSlice []MarketStats
	now := time.Now()
	nowMillis := utils.TimeToUnixEpoch(now)

	dbMarkets, err := s.RetrieveMarketData()
	if err != nil {
		return
	}

	for _, dbMarket := range dbMarkets {
		marketStats := dbMarketToJSON(dbMarket)
		marketStatsSlice = append(marketStatsSlice, marketStats)
	}

	ms = MarketSummary{
		GeneratedAt: nowMillis,
		Pairs:       marketStatsSlice,
	}
	return
}

func dbMarketToJSON(m tickerdb.Market) MarketStats {
	closeTime := utils.TimeToUnixEpoch(m.LastPriceCloseTime)
	return MarketStats{
		TradePairName:      m.TradePair,
		BaseVolume24h:      m.BaseVolume24h,
		CounterVolume24h:   m.CounterVolume24h,
		TradeCount24h:      m.TradeCount24h,
		BaseVolume7d:       m.BaseVolume7d,
		CounterVolume7d:    m.CounterVolume7d,
		TradeCount7d:       m.TradeCount7d,
		LastPrice:          m.LastPrice,
		LastPriceCloseTime: closeTime,
		PriceChange24h:     m.PriceChange24h,
		PriceChange7d:      m.PriceChange7d,
	}
}
