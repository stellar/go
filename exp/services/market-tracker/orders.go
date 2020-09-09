package main

import (
	"sort"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	hProtocol "github.com/stellar/go/protocols/horizon"
)

// Orderbook tracks top-level orderbook statistics.
// Note that volume is denominated in USD for easiest viewing.
type Orderbook struct {
	NumBids   prometheus.Gauge
	NumAsks   prometheus.Gauge
	BidVolume prometheus.Gauge
	AskVolume prometheus.Gauge
}

// usdOrder holds the USD representation of an XLM-based order on the DEX.
// This contains the amount of the asset in the order; its price in USD; and that amount in USD.
type usdOrder struct {
	xlmAmount  float64
	usdPrice   float64
	usdAmount  float64
	baseAmount float64
}

// convertBids converts a list of bids into dollar and base asset amounts and sorts them in decreasing price order.
func convertBids(bids []hProtocol.PriceLevel, xlmUsdPrice, baseUsdPrice float64) ([]usdOrder, error) {
	convertedBids := []usdOrder{}
	for _, b := range bids {
		// for bids, both the price and the amount will be in XLM
		// see: https://github.com/stellar/go/issues/612
		xlmAmt, err := strconv.ParseFloat(b.Amount, 64)
		if err != nil {
			return []usdOrder{}, err
		}

		usdAmt := xlmAmt * xlmUsdPrice
		usdPrice := (float64(b.PriceR.N) / float64(b.PriceR.D)) * xlmUsdPrice
		baseAmt := usdAmt * baseUsdPrice

		cb := usdOrder{
			xlmAmount:  xlmAmt,
			usdAmount:  usdAmt,
			usdPrice:   usdPrice,
			baseAmount: baseAmt,
		}

		convertedBids = append(convertedBids, cb)
	}

	// sort in decreasing order by price
	sort.Slice(convertedBids, func(i, j int) bool {
		return convertedBids[i].usdPrice >= convertedBids[j].usdPrice
	})

	return convertedBids, nil
}

func convertAsks(asks []hProtocol.PriceLevel, xlmUsdPrice, baseUsdPrice float64) ([]usdOrder, error) {
	convertedAsks := []usdOrder{}
	for _, a := range asks {
		// for asks, the amount will be in base, but the price in XLM
		// see: https://github.com/stellar/go/issues/612
		baseAmt, err := strconv.ParseFloat(a.Amount, 64)
		if err != nil {
			return []usdOrder{}, err
		}

		askXlmPrice := float64(a.PriceR.N) / float64(a.PriceR.D)
		xlmAmt := baseAmt * askXlmPrice
		usdAmt := xlmAmt * xlmUsdPrice
		usdPrice := askXlmPrice * xlmUsdPrice

		ca := usdOrder{
			xlmAmount:  xlmAmt,
			usdPrice:   usdPrice,
			usdAmount:  usdAmt,
			baseAmount: baseAmt,
		}

		convertedAsks = append(convertedAsks, ca)
	}

	// sort in increasing order
	sort.Slice(convertedAsks, func(i, j int) bool {
		return convertedAsks[i].usdPrice <= convertedAsks[j].usdPrice
	})
	return convertedAsks, nil
}

func getOrdersVolume(orders []usdOrder) (v float64) {
	for _, o := range orders {
		v += o.usdAmount
	}
	return
}
