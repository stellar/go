package main

import (
	"sort"
	"strconv"

	hProtocol "github.com/stellar/go/protocols/horizon"
)

// usdOrder holds the USD representation of an XLM-based order on the DEX.
// This contains the amount of the asset in the order; its price in USD; and that amount in USD.
type usdOrder struct {
	xlmAmount float64
	usdPrice  float64
	usdAmount float64
}

// getUsdBids converts a list of bids to dollar amounts and sorts them in decreasing price order.
func getUsdBids(bids []hProtocol.PriceLevel, price float64) ([]usdOrder, error) {
	usdBids, err := getUsdOrders(bids, price)
	if err != nil {
		return []usdOrder{}, err
	}

	sort.Slice(usdBids, func(i, j int) bool {
		return usdBids[i].usdPrice >= usdBids[j].usdPrice
	})

	return usdBids, nil
}

// getUsdAsks converts a list of asks to dollar amounts and sorts them in increasing price order.
func getUsdAsks(asks []hProtocol.PriceLevel, price float64) ([]usdOrder, error) {
	usdAsks, err := getUsdOrders(asks, price)
	if err != nil {
		return []usdOrder{}, err
	}
	sort.Slice(usdAsks, func(i, j int) bool {
		return usdAsks[i].usdPrice <= usdAsks[j].usdPrice
	})
	return usdAsks, nil
}

// getUsdOrders converts XLM-based orders to dollar amounts.
func getUsdOrders(orders []hProtocol.PriceLevel, usdPerXlm float64) ([]usdOrder, error) {
	usdOrders := []usdOrder{}
	for _, order := range orders {
		xlmAmt, err := strconv.ParseFloat(order.Amount, 64)
		if err != nil {
			return []usdOrder{}, err
		}

		usdAmt := xlmAmt * usdPerXlm
		usdPrice := float64(order.PriceR.N) / float64(order.PriceR.D) / usdPerXlm
		currBid := usdOrder{
			xlmAmount: xlmAmt,
			usdAmount: usdAmt,
			usdPrice:  usdPrice,
		}
		usdOrders = append(usdOrders, currBid)
	}
	return usdOrders, nil
}
