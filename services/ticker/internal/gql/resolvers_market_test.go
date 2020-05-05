package gql

import (
	"fmt"
	"testing"
	"time"

	"github.com/stellar/go/services/ticker/internal/tickerdb"
	"github.com/stretchr/testify/assert"
)

func TestPostProcessPartialMarket(t *testing.T) {
	baseCode := "BTC"
	baseIssuer := "GCF3TQXKZJNFJK7HCMNE2O2CUNKCJH2Y2ROISTBPLC7C5EIA5NNG2XZB"
	counterCode := "ETH"
	counterIssuer := "ABF3TQXKZJNFJK7HCMNE2O2CUNKCJH2Y2ROISTBPLC7C5EIA5NNG2XZB"
	tradePair := fmt.Sprintf("%s:%s / %s:%s", baseCode, baseIssuer, counterCode, counterIssuer)
	now := time.Now()

	dbMkt := tickerdb.PartialMarket{
		TradePairName:        tradePair,
		BaseAssetCode:        baseCode,
		BaseAssetIssuer:      baseIssuer,
		CounterAssetCode:     counterCode,
		CounterAssetIssuer:   counterIssuer,
		BaseVolume:           100.0,
		TradeCount:           2,
		Open:                 20.0,
		Low:                  10.0,
		High:                 25.0,
		Close:                20.0,
		NumBids:              10,
		BidVolume:            1000.0,
		HighestBid:           250.0,
		NumAsks:              15,
		AskVolume:            200.0,
		LowestAsk:            150.0,
		NumBidsReverse:       20,
		BidVolumeReverse:     400.0,
		HighestBidReverse:    100.0,
		NumAsksReverse:       10,
		AskVolumeReverse:     250.0,
		LowestAskReverse:     100.0,
		IntervalStart:        now,
		FirstLedgerCloseTime: now,
		LastLedgerCloseTime:  now,
	}

	partialMarket := dbMarketToPartialMarket(dbMkt)
	reverseOs := reverseOrderbook(dbMkt)

	processedMkt := postProcessPartialMarket(partialMarket, reverseOs, nil, nil)
	assert.Equal(t, partialMarket, processedMkt)

	isNewEndpoint := false
	processedMkt = postProcessPartialMarket(partialMarket, reverseOs, nil, &isNewEndpoint)
	assert.Equal(t, partialMarket, processedMkt)

	isNewEndpoint = true
	// Confirm that a nil pair name with the new endpoint inverts prices.
	processedMkt = postProcessPartialMarket(partialMarket, reverseOs, nil, &isNewEndpoint)
	assert.Equal(t, 1/partialMarket.Open, processedMkt.Open)
	assert.Equal(t, 1/partialMarket.High, processedMkt.Low)
	assert.Equal(t, 1/partialMarket.Low, processedMkt.High)
	assert.Equal(t, 1/partialMarket.Low-1/partialMarket.High, processedMkt.Change)
	assert.Equal(t, 1/partialMarket.Close, processedMkt.Close)

	// Confirm that an empty pair name with the new endpoint inverts prices.
	oldPairName := ""
	processedMkt = postProcessPartialMarket(partialMarket, reverseOs, &oldPairName, &isNewEndpoint)
	assert.Equal(t, 1/partialMarket.Open, processedMkt.Open)
	assert.Equal(t, 1/partialMarket.High, processedMkt.Low)
	assert.Equal(t, 1/partialMarket.Low, processedMkt.High)
	assert.Equal(t, 1/partialMarket.Low-1/partialMarket.High, processedMkt.Change)
	assert.Equal(t, 1/partialMarket.Close, processedMkt.Close)

	// Confirm that a matching pair name with the new endpoint inverts prices.
	oldPairName = fmt.Sprintf("%s:%s / %s:%s", baseCode, baseIssuer, counterCode, counterIssuer)
	processedMkt = postProcessPartialMarket(partialMarket, reverseOs, &oldPairName, &isNewEndpoint)
	assert.Equal(t, 1/partialMarket.Open, processedMkt.Open)
	assert.Equal(t, 1/partialMarket.High, processedMkt.Low)
	assert.Equal(t, 1/partialMarket.Low, processedMkt.High)
	assert.Equal(t, 1/partialMarket.Low-1/partialMarket.High, processedMkt.Change)
	assert.Equal(t, 1/partialMarket.Close, processedMkt.Close)

	// Confirm that a swapped pair name reverses counter and base assets and changes the orderbook.
	oldPairName = fmt.Sprintf("%s:%s / %s:%s", counterCode, counterIssuer, baseCode, baseIssuer)
	processedMkt = postProcessPartialMarket(partialMarket, reverseOs, &oldPairName, &isNewEndpoint)
	assert.Equal(t, oldPairName, processedMkt.TradePair)
	assert.Equal(t, partialMarket.CounterAssetCode, processedMkt.BaseAssetCode)
	assert.Equal(t, partialMarket.CounterAssetIssuer, processedMkt.BaseAssetIssuer)
	assert.Equal(t, partialMarket.BaseAssetCode, processedMkt.CounterAssetCode)
	assert.Equal(t, partialMarket.BaseAssetIssuer, processedMkt.CounterAssetIssuer)
	assert.Equal(t, partialMarket.BaseVolume, processedMkt.CounterVolume)
	assert.Equal(t, partialMarket.CounterVolume, processedMkt.BaseVolume)
	assert.Equal(t, reverseOs, processedMkt.OrderbookStats)
}
