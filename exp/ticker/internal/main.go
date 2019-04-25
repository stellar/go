package ticker

import (
	"github.com/stellar/go/exp/ticker/internal/scraper"
)

// MarketSummary represents a summary of statistics of all valid markets
// within a given period of time.
type MarketSummary struct {
	GeneratedAt int64         `json:"generated_at"`
	Pairs       []MarketStats `json:"pairs"`
}

// Market stats represents the statistics of a specific market (identified by
// a trade pair).
type MarketStats struct {
	TradePairName    string  `json:"name"`
	BaseVolume24h    float64 `json:"base_volume"`
	CounterVolume24h float64 `json:"counter_volume"`
	TradeCount24h    int64   `json:"trade_count"`
	Open24h          float64 `json:"open"`
	Low24h           float64 `json:"low"`
	High24h          float64 `json:"high"`
	Change24h        float64 `json:"change"`
	BaseVolume7d     float64 `json:"base_volume_7d"`
	CounterVolume7d  float64 `json:"counter_volume_7d"`
	TradeCount7d     int64   `json:"trade_count_7d"`
	Open7d           float64 `json:"open_7d"`
	Low7d            float64 `json:"low_7d"`
	High7d           float64 `json:"high_7d"`
	Change7d         float64 `json:"change_7d"`
	Price            float64 `json:"price"`
	Close            float64 `json:"close"`
	CloseTime        int64   `json:"close_time"`
	NumBids          int     `json:"num_bids"`
	BidVolume        float64 `json:"bid_volume"`
	HighestBid       float64 `json:"highest_bid"`
	NumAsks          int     `json:"num_asks"`
	AskVolume        float64 `json:"ask_volume"`
	LowestAsk        float64 `json:"lowest_ask"`
	Spread           float64 `json:"spread"`
	SpreadMidPoint   float64 `json:"spread_mid_point"`
}

// Asset Sumary represents the collection of valid assets.
type AssetSummary struct {
	GeneratedAt int64   `json:"generated_at"`
	Assets      []Asset `json:"assets"`
}

// Asset represent the aggregated data for a given asset.
type Asset struct {
	scraper.FinalAsset

	LastValidTimestamp int64 `json:"last_valid"`
}
