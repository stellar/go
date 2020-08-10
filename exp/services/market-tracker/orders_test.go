package main

import (
	"testing"

	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stretchr/testify/assert"
)

var badAmtOrders = []hProtocol.PriceLevel{hProtocol.PriceLevel{
	PriceR: hProtocol.Price{
		N: 4,
		D: 2,
	},
	Price:  "2.0",
	Amount: "amount",
}}

var lowUsdOrder = usdOrder{
	xlmAmount: 50.0,
	usdAmount: 5.0,
	usdPrice:  20.0,
}

var highUsdOrder = usdOrder{
	xlmAmount: 100.0,
	usdAmount: 10.0,
	usdPrice:  25.0,
}

func TestGetUsdBids(t *testing.T) {
	usdXlmPrice := 0.10
	bids, err := getUsdBids(badAmtOrders, usdXlmPrice)
	assert.Error(t, err)
	assert.Equal(t, 0, len(bids))

	bids, err = getUsdBids(hOrders, usdXlmPrice)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, bids[0].usdPrice, bids[1].usdPrice)
	assert.Equal(t, highUsdOrder, bids[0])
	assert.Equal(t, lowUsdOrder, bids[1])
}

func TestGetUsdAsks(t *testing.T) {
	usdXlmPrice := 0.10
	asks, err := getUsdAsks(badAmtOrders, usdXlmPrice)
	assert.Error(t, err)
	assert.Equal(t, 0, len(asks))

	orders := []hProtocol.PriceLevel{hHighOrder, hLowOrder}
	asks, err = getUsdAsks(orders, usdXlmPrice)
	assert.NoError(t, err)
	assert.LessOrEqual(t, asks[0].usdPrice, asks[1].usdPrice)
	assert.Equal(t, lowUsdOrder, asks[0])
	assert.Equal(t, highUsdOrder, asks[1])
}

func TestGetUsdOrders(t *testing.T) {
	usdXlmPrice := 0.10
	usdOrders, err := getUsdOrders(badAmtOrders, usdXlmPrice)
	assert.Error(t, err)

	usdOrders, err = getUsdOrders(hOrders, usdXlmPrice)
	assert.NoError(t, err)
	assert.Equal(t, lowUsdOrder, usdOrders[0])
	assert.Equal(t, highUsdOrder, usdOrders[1])
}
