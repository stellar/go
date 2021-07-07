package main

import (
	"sort"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	hProtocol "github.com/xdbfoundation/go/protocols/frontier"
)

// Orderbook tracks top-level orderbook statistics.
// Note that volume is denominated in USD for easiest viewing.
type Orderbook struct {
	NumBids       prometheus.Gauge
	NumAsks       prometheus.Gauge
	BidBaseVolume prometheus.Gauge
	BidUsdVolume  prometheus.Gauge
	AskBaseVolume prometheus.Gauge
	AskUsdVolume  prometheus.Gauge
}

// usdOrder holds the USD representation of an XDB-based order on the DEX.
// This contains the amount of the asset in the order; its price in USD; and that amount in USD.
type usdOrder struct {
	xdbAmount  float64
	usdPrice   float64
	usdAmount  float64
	baseAmount float64
}

// convertBids converts a list of bids into dollar and base asset amounts and sorts them in decreasing price order.
func convertBids(bids []hProtocol.PriceLevel, xdbUsdPrice, baseUsdPrice float64) ([]usdOrder, error) {
	convertedBids, err := convertOrders(bids, xdbUsdPrice, baseUsdPrice)
	if err != nil {
		return []usdOrder{}, err
	}

	// sort in decreasing order
	sort.Slice(convertedBids, func(i, j int) bool {
		return convertedBids[i].usdPrice >= convertedBids[j].usdPrice
	})

	return convertedBids, nil
}

func convertAsks(asks []hProtocol.PriceLevel, xdbUsdPrice, baseUsdPrice float64) ([]usdOrder, error) {
	convertedAsks, err := convertOrders(asks, xdbUsdPrice, baseUsdPrice)
	if err != nil {
		return []usdOrder{}, err
	}

	// sort in increasing order
	sort.Slice(convertedAsks, func(i, j int) bool {
		return convertedAsks[i].usdPrice <= convertedAsks[j].usdPrice
	})
	return convertedAsks, nil
}

func convertOrders(orders []hProtocol.PriceLevel, xdbUsdPrice, baseUsdPrice float64) ([]usdOrder, error) {
	convertedOrders := []usdOrder{}
	for _, order := range orders {
		xdbAmt, err := strconv.ParseFloat(order.Amount, 64)
		if err != nil {
			return []usdOrder{}, err
		}

		usdAmt := xdbAmt * xdbUsdPrice
		usdPrice := float64(order.PriceR.N) / float64(order.PriceR.D) * xdbUsdPrice
		baseAmt := usdAmt * baseUsdPrice
		cOrder := usdOrder{
			xdbAmount:  xdbAmt,
			usdPrice:   usdPrice,
			usdAmount:  usdAmt,
			baseAmount: baseAmt,
		}

		convertedOrders = append(convertedOrders, cOrder)
	}

	return convertedOrders, nil
}

func getOrdersUsdVolume(orders []usdOrder) (v float64) {
	for _, o := range orders {
		v += o.usdAmount
	}
	return
}

func getOrdersBaseVolume(orders []usdOrder) (v float64) {
	for _, o := range orders {
		v += o.baseAmount
	}
	return
}
