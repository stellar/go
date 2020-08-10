package main

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	hProtocol "github.com/stellar/go/protocols/horizon"
)

// Volume stores volume of a various base pair over various time periods.
// DN represents the last N days. and HM represents the last M hours.
type Volume struct {
	D30 prometheus.Gauge
	D1  prometheus.Gauge
}

type xlmPrice struct {
	timestamp int64
	price     float64
}

type volumeHist struct {
	start     int64
	end       int64
	xlmVolume float64
	usdVolume float64
}

func trackVolumes(cfg Config, c trackerClient, watchedTPsPtr *[]prometheusWatchedTP) {
	watchedTPs := *watchedTPsPtr
	volumeMap := initVolumes(cfg, c, watchedTPs)

	go func() {
		updateVolume(cfg, c, watchedTPsPtr, volumeMap)
	}()
}

func initVolumes(cfg Config, c trackerClient, watchedTPs []prometheusWatchedTP) map[string][]volumeHist {
	req := mustCreateXlmPriceRequest()
	priceHist, err := getXlmPriceHistory(req)
	if err != nil {
		fmt.Printf("got error when getting xlm price history: %s\n", err)
	}

	volumeHistMap := make(map[string][]volumeHist)
	end := time.Now()
	start := end.Add(time.Duration(-30 * 24 * time.Hour))
	res := 15 * 60                                     // resolution length, in seconds
	cRes := time.Duration(res*1000) * time.Millisecond // horizon request must be in milliseconds
	for i, wtp := range watchedTPs {
		// TODO: Calculate volume for assets with non-native base.
		if wtp.TradePair.SellingAsset.Code != "XLM" && wtp.TradePair.SellingAsset.IssuerAddress != "native" {
			continue
		}

		taps, err := c.getAggTradesForTradePair(wtp.TradePair, start, end, cRes)
		if err != nil {
			fmt.Printf("got error getting agg trades for pair %s\n: %s", wtp.TradePair.String(), err)
		}

		records := getAggRecords(taps)
		volumeHist, err := constructVolumeHistory(records, priceHist, start, end, res)
		if err != nil {
			fmt.Printf("got error constructing volume history for pair %s\n: %s", wtp.TradePair.String(), err)
		}

		volumeHistMap[wtp.TradePair.String()] = volumeHist

		month := start.Unix()
		day := end.Add(time.Duration(-24 * time.Hour)).Unix()
		watchedTPs[i].XlmVolume.D1.Set(addXlmVolumeHistory(volumeHist, day))
		watchedTPs[i].XlmVolume.D30.Set(addXlmVolumeHistory(volumeHist, month))
		watchedTPs[i].UsdVolume.D1.Set(addUsdVolumeHistory(volumeHist, day))
		watchedTPs[i].UsdVolume.D30.Set(addUsdVolumeHistory(volumeHist, month))
	}
	return volumeHistMap
}

func updateVolume(cfg Config, c trackerClient, watchedTPsPtr *[]prometheusWatchedTP, volumeHistMap map[string][]volumeHist) {
	req := mustCreateXlmPriceRequest()
	historyUnit := time.Duration(15 * 60 * time.Second) // length of each individual unit of volume history
	cRes := time.Duration(60*1000) * time.Millisecond   // horizon client requests have a 1 minute resolution, in milliseconds
	day := time.Duration(24 * 60 * 60 * time.Second)    // number of seconds in a day
	watchedTPs := *watchedTPsPtr
	forLoopDuration := time.Duration(0)
	for {
		time.Sleep(historyUnit - forLoopDuration) // wait before starting the update

		price, err := getLatestXlmPrice(req)
		if err != nil {
			fmt.Printf("error while getting latest price: %s", err)
		}

		end := time.Now()
		start := end.Add(-1 * historyUnit)
		for i, wtp := range watchedTPs {
			// TODO: Calculate volume for assets with non-native base.
			if wtp.TradePair.SellingAsset.Code != "XLM" && wtp.TradePair.SellingAsset.IssuerAddress != "native" {
				continue
			}

			tps := wtp.TradePair.String()
			taps, err := c.getAggTradesForTradePair(wtp.TradePair, start, end, cRes)
			if err != nil {
				fmt.Printf("got error getting agg trades for pair %s\n: %s", tps, err)
			}

			records := getAggRecords(taps)
			sts := start.Unix()
			ets := end.Unix()
			xlmVolume, err := totalRecordsXlmVolume(records, start, end)
			if err != nil {
				fmt.Printf("got error aggregating xlm volume for pair %s\n: %s", tps, err)
			}

			latestVolume := volumeHist{
				start:     sts,
				end:       ets,
				xlmVolume: xlmVolume,
				usdVolume: price * xlmVolume,
			}

			// get the volumes of the oldest history unit, for both the day and month
			vh := volumeHistMap[tps]
			oldestVolumeMonth := vh[len(vh)-1]
			oldestVolumeDay := vh[int(day/historyUnit)]

			// remove the oldest volume, store the newest one
			vh = vh[:len(vh)-1]
			vh = append([]volumeHist{latestVolume}, vh...)
			volumeHistMap[tps] = vh

			// update the volume metrics using the difference between the latest and oldest
			// history units' volumes, as appropriate for that metric
			watchedTPs[i].XlmVolume.D30.Add(latestVolume.xlmVolume - oldestVolumeMonth.xlmVolume)
			watchedTPs[i].XlmVolume.D1.Add(latestVolume.xlmVolume - oldestVolumeDay.xlmVolume)
			watchedTPs[i].UsdVolume.D30.Add(latestVolume.usdVolume - oldestVolumeMonth.usdVolume)
			watchedTPs[i].UsdVolume.D1.Add(latestVolume.usdVolume - oldestVolumeDay.usdVolume)
		}

		forLoopDuration = time.Now().Sub(end)
	}
}

func getAggRecords(taps []hProtocol.TradeAggregationsPage) (records []hProtocol.TradeAggregation) {
	for _, tap := range taps {
		records = append(records, tap.Embedded.Records...)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Timestamp > records[j].Timestamp })
	return
}

func constructVolumeHistory(tas []hProtocol.TradeAggregation, prices []xlmPrice, start, end time.Time, res int) ([]volumeHist, error) {
	if len(prices) < 2 {
		return []volumeHist{}, fmt.Errorf("mis-formed xlm price history from stellar expert")
	}

	volumeHistory := []volumeHist{}
	priceIdx := -1
	recordIdx := 0
	currEnd := end
	for currEnd.After(start) {
		// find the weighted price for the current interval
		cets := currEnd.Unix()
		csts := cets - int64(res)
		priceIdx = findTimestampPriceIndex(csts, prices, priceIdx)

		cwp, err := calcWeightedPrice(csts, priceIdx, prices)
		if err != nil {
			return []volumeHist{}, err
		}

		// find total volume of records in this interval
		// TODO: This loop does not correctly include records before the start
		// time. however, that should not happen, given that we define start before
		// calling the horizon client.
		currXlmVolume := 0.
		for recordIdx < len(tas) {
			r := tas[recordIdx]
			rts := r.Timestamp / 1000
			if rts < csts {
				// if record is before timeframe, break
				break
			} else if rts > cets {
				// if record is after timeframe, continue to next
				// record, since this could be in range
				recordIdx++
				continue
			} else {
				recordXlmVolume, err := strconv.ParseFloat(r.CounterVolume, 64)
				if err != nil {
					return []volumeHist{}, err
				}
				currXlmVolume += recordXlmVolume
				recordIdx++
			}
		}

		currVolume := volumeHist{
			start:     csts,
			end:       cets,
			xlmVolume: currXlmVolume,
			usdVolume: cwp * currXlmVolume,
		}

		currEnd = currEnd.Add(time.Duration(-1*res) * time.Second)
		volumeHistory = append(volumeHistory, currVolume)
	}
	return volumeHistory, nil
}

func addXlmVolumeHistory(history []volumeHist, end int64) (xlmVolume float64) {
	for _, vh := range history {
		if vh.end < end {
			break
		}
		xlmVolume += vh.xlmVolume
	}
	return
}

func addUsdVolumeHistory(history []volumeHist, end int64) (usdVolume float64) {
	for _, vh := range history {
		if vh.end < end {
			break
		}
		usdVolume += vh.usdVolume
	}
	return
}

func totalRecordsXlmVolume(tas []hProtocol.TradeAggregation, start, end time.Time) (float64, error) {
	tv := 0.0
	for _, ta := range tas {
		ts := time.Unix(ta.Timestamp/1000, 0) // timestamps are milliseconds since epoch time
		if ts.Before(start) || ts.After(end) {
			return 0.0, fmt.Errorf("record at timestamp %v is out of time bounds %v to %v", ts, start, end)
		}

		cv, err := strconv.ParseFloat(ta.CounterVolume, 64)
		if err != nil {
			return 0.0, err
		}
		tv += cv
	}
	return tv, nil
}

// findTimestampPriceIndex iterates through an array of timestamps and prices, and returns the
// index of the oldest such pair that is more recent than the given timestamp.
// This assumes those pairs are sorted by decreasing timestamp, (i.e. most recent first).
func findTimestampPriceIndex(timestamp int64, prices []xlmPrice, startIndex int) int {
	index := startIndex
	if index < 0 {
		if timestamp > prices[0].timestamp {
			return index
		}
		index = 0
	}

	for index < len(prices)-1 {
		if prices[index].timestamp >= timestamp && timestamp >= prices[index+1].timestamp {
			break
		}
		index++
	}
	return index
}

func calcWeightedPrice(timestamp int64, startIndex int, prices []xlmPrice) (float64, error) {
	// we expect prices sorted in decreasing timestamp (i.e., most recent first)
	// TODO: Use resolution to weight prices.
	if startIndex < 0 {
		if timestamp < prices[0].timestamp {
			return 0.0, fmt.Errorf("update price index before calculating price")
		}
		return prices[0].price, nil
	} else if startIndex >= len(prices)-1 {
		if timestamp > prices[len(prices)-1].timestamp {
			return 0.0, fmt.Errorf("update price index before calculating price")
		}
		return prices[len(prices)-1].price, nil
	}

	if timestamp > prices[startIndex].timestamp || timestamp < prices[startIndex+1].timestamp {
		return 0.0, fmt.Errorf("update price index before calculating price")
	}

	avgPrice := (prices[startIndex].price + prices[startIndex+1].price) / 2
	return avgPrice, nil
}
