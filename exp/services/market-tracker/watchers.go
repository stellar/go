package main

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func configPrometheusWatchers(tps []TradePair) (watchedTPs []prometheusWatchedTP) {
	for _, tp := range tps {
		labels := prometheus.Labels{
			"tradePair":    fmt.Sprintf("%s", tp),
			"buyingAsset":  fmt.Sprintf("%s", tp.BuyingAsset),
			"sellingAsset": fmt.Sprintf("%s", tp.SellingAsset),
		}

		pwtp := prometheusWatchedTP{
			TradePair: tp,
			Spread:    createSpread(labels),
			XlmVolume: createVolume("xlm", labels),
			UsdVolume: createVolume("usd", labels),
			Slippage:  createSlippage(labels),
			FairValue: createFairMarket(labels),
		}
		watchedTPs = append(watchedTPs, pwtp)
	}
	return
}

func createSpread(labels prometheus.Labels) Spread {
	return Spread{
		Top:  createSpreadGauge("", "", labels),
		D100: createSpreadGauge("_100", "at depth $100", labels),
		D1K:  createSpreadGauge("_1K", "at depth $1000", labels),
		D5K:  createSpreadGauge("_5K", "at depth $5000", labels),
		D25K: createSpreadGauge("_25K", "at depth $25,000", labels),
		D50K: createSpreadGauge("_50K", "at depth $50,000", labels),
	}
}

func createSpreadGauge(depthShort, depthLong string, labels prometheus.Labels) prometheus.Gauge {
	return promauto.NewGauge(prometheus.GaugeOpts{
		Name:        fmt.Sprintf("stellar_market_tracker_spread%s", depthShort),
		ConstLabels: labels,
		Help:        fmt.Sprintf("Percentage market spread %s", depthLong),
	})
}

func createVolume(currency string, labels prometheus.Labels) Volume {
	return Volume{
		D30: createVolumeGauge(currency, "30d", "the past 30 days", labels),
		D1:  createVolumeGauge(currency, "1d", "the past 1 day", labels),
	}
}

func createVolumeGauge(currency, timeShort, timeLong string, labels prometheus.Labels) prometheus.Gauge {
	return promauto.NewGauge(prometheus.GaugeOpts{
		Name:        fmt.Sprintf("stellar_market_tracker_%s_volume_%s", currency, timeShort),
		ConstLabels: labels,
		Help:        fmt.Sprintf("Trading volume in %s over %s", currency, timeLong),
	})
}

func createSlippage(labels prometheus.Labels) Slippage {
	return Slippage{
		BidD100: createSlippageGauge("bid", "100", labels),
		AskD100: createSlippageGauge("ask", "100", labels),
		BidD1K:  createSlippageGauge("bid", "1K", labels),
		AskD1K:  createSlippageGauge("ask", "1K", labels),
		BidD5K:  createSlippageGauge("bid", "5K", labels),
		AskD5K:  createSlippageGauge("ask", "5K", labels),
	}
}

func createSlippageGauge(orderType, depth string, labels prometheus.Labels) prometheus.Gauge {
	return promauto.NewGauge(prometheus.GaugeOpts{
		Name:        fmt.Sprintf("stellar_market_tracker_%s_slippage_%s", orderType, depth),
		ConstLabels: labels,
		Help:        fmt.Sprintf("Slippage of %s at depth %s", orderType, depth),
	})
}

func createFairMarket(labels prometheus.Labels) prometheus.Gauge {
	return promauto.NewGauge(prometheus.GaugeOpts{
		Name:        fmt.Sprintf("stellar_market_tracker_fmp"),
		ConstLabels: labels,
		Help:        fmt.Sprintf("Pct difference of DEX value from fair market value"),
	})
}
