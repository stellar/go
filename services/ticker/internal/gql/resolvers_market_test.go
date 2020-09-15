package gql

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stellar/go/services/ticker/internal/tickerdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		CounterVolume:        200.0,
		TradeCount:           2,
		Open:                 10.0,
		Low:                  5.0,
		High:                 20.0,
		Close:                10.0,
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

	partialMkt := dbMarketToPartialMarket(dbMkt)
	reverseOs := reverseOrderbook(dbMkt)

	// Confirm that non-nil pair name and code results in an error.
	userCode := ""
	userPairName := ""
	processedMkt, err := postProcessPartialMarket(partialMkt, reverseOs, &userPairName, &userCode)
	require.EqualError(t, err, "cannot provide both pair name and code")
	assert.Nil(t, processedMkt)

	// Confirm that a nil pair name and code has no effect.
	processedMkt, err = postProcessPartialMarket(partialMkt, reverseOs, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, partialMkt, processedMkt)

	// Confirm that a matching pair name has no effect.
	userPairName = fmt.Sprintf("%s:%s / %s:%s", baseCode, baseIssuer, counterCode, counterIssuer)
	processedMkt, err = postProcessPartialMarket(partialMkt, reverseOs, &userPairName, nil)
	require.NoError(t, err)
	assert.Equal(t, partialMkt, processedMkt)

	// Confirm that a swapped pair name reverses counter and base assets and changes the orderbook.
	userPairName = fmt.Sprintf("%s:%s / %s:%s", counterCode, counterIssuer, baseCode, baseIssuer)
	processedMkt, err = postProcessPartialMarket(partialMkt, reverseOs, &userPairName, nil)
	require.NoError(t, err)
	assert.Equal(t, userPairName, processedMkt.TradePair)
	assert.Equal(t, counterCode, processedMkt.BaseAssetCode)
	assert.Equal(t, counterIssuer, processedMkt.BaseAssetIssuer)
	assert.Equal(t, baseCode, processedMkt.CounterAssetCode)
	assert.Equal(t, baseIssuer, processedMkt.CounterAssetIssuer)
	assert.Equal(t, 100.0, processedMkt.CounterVolume)
	assert.Equal(t, 200.0, processedMkt.BaseVolume)
	assert.Equal(t, 1/10.0, processedMkt.Open)
	assert.Equal(t, 1/20.0, processedMkt.Low)
	assert.Equal(t, 1/5.0, processedMkt.High)

	// There might be some floating point rounding issues, so this test
	// needs to be a bit more flexible. Since the change is 0.15, an error
	// around 0.0000000000001 is acceptable:
	changeDiff := math.Abs(0.15 - processedMkt.Change)
	assert.True(t, changeDiff < 0.0000000000001)

	assert.Equal(t, 1/10.0, processedMkt.Close)
	assert.Equal(t, reverseOs, processedMkt.OrderbookStats)

	userCode = "BTC"
	processedMkt, err = postProcessPartialMarket(partialMkt, reverseOs, nil, &userCode)
	require.NoError(t, err)
	assert.Equal(t, partialMkt, processedMkt)

	userCode = "ETH"
	processedMkt, err = postProcessPartialMarket(partialMkt, reverseOs, nil, &userCode)
	require.NoError(t, err)
	assert.Equal(t, userPairName, processedMkt.TradePair)
	assert.Equal(t, counterCode, processedMkt.BaseAssetCode)
	assert.Equal(t, counterIssuer, processedMkt.BaseAssetIssuer)
	assert.Equal(t, baseCode, processedMkt.CounterAssetCode)
	assert.Equal(t, baseIssuer, processedMkt.CounterAssetIssuer)
	assert.Equal(t, 100.0, processedMkt.CounterVolume)
	assert.Equal(t, 200.0, processedMkt.BaseVolume)
	assert.Equal(t, 1/10.0, processedMkt.Open)
	assert.Equal(t, 1/20.0, processedMkt.Low)
	assert.Equal(t, 1/5.0, processedMkt.High)

	// There might be some floating point rounding issues, so this test
	// needs to be a bit more flexible. Since the change is 0.15, an error
	// around 0.0000000000001 is acceptable:
	changeDiff = math.Abs(0.15 - processedMkt.Change)
	assert.True(t, changeDiff < 0.0000000000001)

	assert.Equal(t, 1/10.0, processedMkt.Close)
	assert.Equal(t, reverseOs, processedMkt.OrderbookStats)
}
