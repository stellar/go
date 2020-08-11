package main

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Spread tracks the percent spread at various market depths.
type Spread struct {
	Top  prometheus.Gauge
	D100 prometheus.Gauge
	D1K  prometheus.Gauge
	D5K  prometheus.Gauge
	D25K prometheus.Gauge
	D50K prometheus.Gauge
}

// Slippage tracks the bid and ask slippages at various market depths.
type Slippage struct {
	BidD100 prometheus.Gauge
	AskD100 prometheus.Gauge
	BidD1K  prometheus.Gauge
	AskD1K  prometheus.Gauge
	BidD5K  prometheus.Gauge
	AskD5K  prometheus.Gauge
}

func trackSpreads(cfg Config, c trackerClient, watchedTPsPtr *[]prometheusWatchedTP) {
	req := mustCreateXlmPriceRequest()
	assetReq := mustCreateAssetPriceRequest()
	var assetMapStr string
	assetMapLastUpdated := time.Now().Add(-2 * time.Hour) // initialize so the first value won't be cached

	go func() {
		for {
			price, err := getLatestXlmPrice(req)
			if err != nil {
				fmt.Printf("error while getting latest price: %s", err)
			}

			watchedTPs := *watchedTPsPtr
			for i, wtp := range watchedTPs {
				obStats, err := c.getOrderBookForTradePair(wtp.TradePair)
				if err != nil {
					fmt.Printf("error while getting orderbook stats for asset pair %s: %s", wtp.TradePair, err)
				}

				spreadPct := calcSpreadPctForOrderBook(obStats)
				if err != nil {
					fmt.Printf("error while processing asset pair %s: %s", wtp.TradePair, err)
				}

				watchedTPs[i].Spread.Top.Set(spreadPct)

				// we only compute spreads at various depths for xlm-based pairs,
				// because our usd prices are in terms of xlm.
				if wtp.TradePair.SellingAsset.Code != "XLM" {
					continue
				}

				usdBids, err := getUsdBids(obStats.Bids, price)
				if err != nil {
					fmt.Printf("error while converting bids to USD: %s", err)
					continue
				}

				usdAsks, err := getUsdAsks(obStats.Asks, price)
				if err != nil {
					fmt.Printf("error while converting asks to USD: %s", err)
					continue
				}

				watchedTPs[i].Spread.D100.Set(calcSpreadPctAtDepth(usdBids, usdAsks, 100.))
				watchedTPs[i].Spread.D1K.Set(calcSpreadPctAtDepth(usdBids, usdAsks, 1000.))
				watchedTPs[i].Spread.D5K.Set(calcSpreadPctAtDepth(usdBids, usdAsks, 5000.))
				watchedTPs[i].Spread.D25K.Set(calcSpreadPctAtDepth(usdBids, usdAsks, 25000.))
				watchedTPs[i].Spread.D50K.Set(calcSpreadPctAtDepth(usdBids, usdAsks, 50000.))

				watchedTPs[i].Slippage.BidD100.Set(calcSlippageAtDepth(usdBids, usdAsks, 100., true))
				watchedTPs[i].Slippage.AskD100.Set(calcSlippageAtDepth(usdBids, usdAsks, 100., false))
				watchedTPs[i].Slippage.BidD1K.Set(calcSlippageAtDepth(usdBids, usdAsks, 1000., true))
				watchedTPs[i].Slippage.AskD1K.Set(calcSlippageAtDepth(usdBids, usdAsks, 1000., false))
				watchedTPs[i].Slippage.BidD5K.Set(calcSlippageAtDepth(usdBids, usdAsks, 5000., true))
				watchedTPs[i].Slippage.AskD5K.Set(calcSlippageAtDepth(usdBids, usdAsks, 5000., false))

				if assetMapLastUpdated.Before(time.Now().Add(-1 * time.Hour)) {
					assetMapStr, err = getPriceResponse(assetReq)
					if err != nil {
						fmt.Printf("error while getting price response: %s\n", err)
						return
					}
					assetMapLastUpdated = time.Now()
				}

				currency := watchedTPs[i].TradePair.BuyingAsset.Currency
				trueAssetUsdPrice, err := getAssetUSDPrice(assetMapStr, currency)
				if err != nil {
					fmt.Printf("error while getting asset price: %s\n", err)
					return
				}

				watchedTPs[i].FairValue.Set(calcFairValuePct(usdBids, usdAsks, trueAssetUsdPrice))
			}

			time.Sleep(time.Duration(cfg.CheckIntervalSeconds) * time.Second)
		}
	}()
}
