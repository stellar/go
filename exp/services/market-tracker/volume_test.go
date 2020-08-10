package main

import (
	"testing"
	"time"

	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stretchr/testify/assert"
)

var pts int64 = 1594668800
var res = 15 * 60 // 15 minutes, in seconds

var xp1 = xlmPrice{
	timestamp: pts,
	price:     1.00,
}

var xp2 = xlmPrice{
	timestamp: pts - int64(res),
	price:     2.00,
}

var xp3 = xlmPrice{
	timestamp: pts - int64(2*res),
	price:     3.00,
}

var xp4 = xlmPrice{
	timestamp: pts - int64(3*res),
	price:     4.00,
}

var prices = []xlmPrice{xp1, xp2, xp3, xp4}

func TestTotalRecordsXlmVolume(t *testing.T) {
	res := 15 * 60
	ta1 := hProtocol.TradeAggregation{Timestamp: 1000 * (pts - int64(res/2))}
	ta1.CounterVolume = "100.0"
	ta2 := hProtocol.TradeAggregation{Timestamp: 1000 * (pts - int64(3*res/2))}
	ta2.CounterVolume = "200.0"
	ta3 := hProtocol.TradeAggregation{Timestamp: 1000 * (pts - int64(5*res/2))}
	ta3.CounterVolume = "300.0"
	tas := []hProtocol.TradeAggregation{ta1, ta2, ta3}

	pt := time.Unix(pts, 0)
	total, err := totalRecordsXlmVolume(tas, time.Unix(pts-int64(res*3), 0), pt)
	assert.NoError(t, err)
	assert.Equal(t, 600., total)

	total, err = totalRecordsXlmVolume(tas, time.Unix(pts-int64(res*2), 0), pt)
	assert.Error(t, err)
	assert.Equal(t, 0., total)
}

func TestAddVolumeHistory(t *testing.T) {
	// every 15 min over 30 days
	numIntervals := 4 * 24 * 30
	vh := []volumeHist{}
	i := 0
	for i < numIntervals {
		s := pts - int64(i*15*60)
		h := volumeHist{
			start:     s,
			end:       s - 15*60,
			xlmVolume: 100.0,
			usdVolume: 10.0,
		}
		vh = append(vh, h)
		i++
	}

	// one day, in seconds
	end := pts - 24*60*60
	usd24h := addUsdVolumeHistory(vh, end)
	assert.Equal(t, 960., usd24h)
	xlm24h := addXlmVolumeHistory(vh, end)
	assert.Equal(t, 9600., xlm24h)

	// 30d, in seconds
	end = pts - 30*24*60*60
	usd30d := addUsdVolumeHistory(vh, end)
	assert.Equal(t, 28800., usd30d)
	xlm30d := addXlmVolumeHistory(vh, end)
	assert.Equal(t, 288000., xlm30d)
}

func TestConstructVolumeHistory(t *testing.T) {
	res := 15 * 60
	ta1 := hProtocol.TradeAggregation{Timestamp: 1000 * (pts - int64(res/2))}
	ta1.CounterVolume = "100.0"
	ta2 := hProtocol.TradeAggregation{Timestamp: 1000 * (pts - int64(3*res/2))}
	ta2.CounterVolume = "200.0"
	ta3 := hProtocol.TradeAggregation{Timestamp: 1000 * (pts - int64(5*res/2))}
	ta3.CounterVolume = "300.0"
	tas := []hProtocol.TradeAggregation{ta1, ta2, ta3}
	start := time.Unix(pts-24*60*60, 0)
	end := time.Unix(pts, 0)

	errPrices := []xlmPrice{}
	volumeHist, err := constructVolumeHistory(tas, errPrices, start, end, res)
	assert.Error(t, err)
	assert.Equal(t, 0, len(volumeHist))

	volumeHist, err = constructVolumeHistory(tas, prices, start, end, res)
	assert.NoError(t, err)
	assert.Equal(t, 24*4, len(volumeHist))

	assert.Equal(t, pts-int64(res), volumeHist[0].start)
	assert.Equal(t, pts, volumeHist[0].end)
	assert.Equal(t, float64(100), volumeHist[0].xlmVolume)
	assert.Equal(t, float64(150), volumeHist[0].usdVolume)

	assert.Equal(t, pts-int64(2*res), volumeHist[1].start)
	assert.Equal(t, pts-int64(res), volumeHist[1].end)
	assert.Equal(t, float64(200), volumeHist[1].xlmVolume)
	assert.Equal(t, float64(500), volumeHist[1].usdVolume)

	assert.Equal(t, pts-int64(3*res), volumeHist[2].start)
	assert.Equal(t, pts-int64(2*res), volumeHist[2].end)
	assert.Equal(t, float64(300), volumeHist[2].xlmVolume)
	assert.Equal(t, float64(1050), volumeHist[2].usdVolume)
}

func TestFindTimestampPriceIndex(t *testing.T) {
	idx := -1
	ts := pts + int64(res/2)
	newIdx := findTimestampPriceIndex(ts, prices, idx)
	assert.Equal(t, -1, newIdx)

	ts = pts - int64(res/2)
	newIdx = findTimestampPriceIndex(ts, prices, idx)
	assert.Equal(t, 0, newIdx)

	ts = pts - (int64(res) + 1)
	newIdx = findTimestampPriceIndex(ts, prices, idx)
	assert.Equal(t, 1, newIdx)

	ts = pts - (int64(2*res) + 1)
	newIdx = findTimestampPriceIndex(ts, prices, idx)
	assert.Equal(t, 2, newIdx)
}

func TestCalcWeightedPrice(t *testing.T) {
	idx := -1
	ts := pts - int64(res/2)
	wp, err := calcWeightedPrice(ts, idx, prices)
	assert.Error(t, err)

	ts = pts + int64(res/2)
	wp, err = calcWeightedPrice(ts, idx, prices)
	assert.NoError(t, err)
	assert.Equal(t, 1., wp)

	idx = 0
	wp, err = calcWeightedPrice(ts, idx, prices)
	assert.Error(t, err)

	ts = pts - int64(res/2)
	wp, err = calcWeightedPrice(ts, idx, prices)
	assert.NoError(t, err)
	assert.Equal(t, 1.5, wp)

	idx = 4
	ts = pts - int64(res/2)
	wp, err = calcWeightedPrice(ts, idx, prices)
	assert.Error(t, err)

	ts = pts - int64(res*5)
	wp, err = calcWeightedPrice(ts, idx, prices)
}
