package ticker

// MarketSummary represents a summary of statistics of all valid markets
// within a given period of time.
type MarketSummary struct {
	GeneratedAt int64         `json:"generated_at"`
	Pairs       []MarketStats `json:"pairs"`
}

// Market stats represents the statistics of a specific market (identified by
// a trade pair).
type MarketStats struct {
	TradePairName      string  `json:"name"`
	BaseVolume24h      float64 `json:"base_volume"`
	CounterVolume24h   float64 `json:"counter_volume"`
	TradeCount24h      int64   `json:"trade_count"`
	BaseVolume7d       float64 `json:"base_volume_7d"`
	CounterVolume7d    float64 `json:"counter_volume_7d"`
	TradeCount7d       int64   `json:"trade_count_7d"`
	LastPrice          float64 `json:"price"`
	LastPriceCloseTime int64   `json:"last_price_close_time"`
	PriceChange24h     float64 `json:"price_change_24h"`
	PriceChange7d      float64 `json:"price_change_7d"`
}
