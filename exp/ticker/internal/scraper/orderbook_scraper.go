package scraper

import (
	"math"
	"strconv"

	"github.com/pkg/errors"
	horizonclient "github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/exp/ticker/internal/utils"
	hProtocol "github.com/stellar/go/protocols/horizon"
)

// fetchOrderbook fetches the orderbook stats for the base and counter assets provided in the parameters
func (c *ScraperConfig) fetchOrderbook(bType, bCode, bIssuer, cType, cCode, cIssuer string) (OrderbookStats, error) {
	obStats := OrderbookStats{
		BaseAssetCode:      bType,
		BaseAssetType:      bCode,
		BaseAssetIssuer:    bIssuer,
		CounterAssetCode:   cType,
		CounterAssetType:   cCode,
		CounterAssetIssuer: cIssuer,
		HighestBid:         math.Inf(-1), // start with -Inf to make sure we catch the correct max bid
		LowestAsk:          math.Inf(1),  // start with +Inf to make sure we catch the correct min ask
	}
	r := createOrderbookRequest(bType, bCode, bIssuer, cType, cCode, cIssuer)
	summary, err := c.Client.OrderBook(r)
	if err != nil {
		return obStats, errors.Wrap(err, "could not fetch orderbook summary")
	}

	err = calcOrderbookStats(&obStats, summary)
	return obStats, errors.Wrap(err, "could not calculate orderbook stats")
}

// calcOrderbookStats calculates the NumBids, BidVolume, BidMax, NumAsks, AskVolume and AskMin
// statistics for a given OrdebookStats instance
func calcOrderbookStats(obStats *OrderbookStats, summary hProtocol.OrderBookSummary) error {
	// Calculate Bid Data:
	obStats.NumBids = len(summary.Bids)
	if obStats.NumBids == 0 {
		obStats.HighestBid = 0
	}
	for _, bid := range summary.Bids {
		pricef, err := strconv.ParseFloat(bid.Price, 64)
		if err != nil {
			return errors.Wrap(err, "invalid bid price")
		}
		obStats.BidVolume += pricef
		if pricef > obStats.HighestBid {
			obStats.HighestBid = pricef
		}
	}

	// Calculate Ask Data:
	obStats.NumAsks = len(summary.Asks)
	if obStats.NumAsks == 0 {
		obStats.LowestAsk = 0
	}
	for _, ask := range summary.Asks {
		pricef, err := strconv.ParseFloat(ask.Price, 64)
		if err != nil {
			return errors.Wrap(err, "invalid ask price")
		}
		obStats.AskVolume += pricef
		if pricef < obStats.LowestAsk {
			obStats.LowestAsk = pricef
		}
	}

	obStats.Spread, obStats.SpreadMidPoint = utils.CalcSpread(obStats.HighestBid, obStats.LowestAsk)

	// Clean up remaining infinity values:
	if math.IsInf(obStats.LowestAsk, 0) {
		obStats.LowestAsk = 0
	}

	if math.IsInf(obStats.HighestBid, 0) {
		obStats.HighestBid = 0
	}

	return nil
}

// createOrderbookRequest generates a horizonclient.OrderBookRequest based on the base
// and counter asset parameters provided
func createOrderbookRequest(bType, bCode, bIssuer, cType, cCode, cIssuer string) horizonclient.OrderBookRequest {
	r := horizonclient.OrderBookRequest{
		SellingAssetType: horizonclient.AssetType(bType),
		BuyingAssetType:  horizonclient.AssetType(cType),
	}

	// The Horizon API requires *AssetCode and *AssetIssuer fields to be empty
	// when an Asset is native. As we store "XLM" as the asset code for native,
	// we should only add Code and Issuer info in case we're dealing with
	// non-native assets.
	// See: https://www.stellar.org/developers/horizon/reference/endpoints/orderbook-details.html
	if bType != string(horizonclient.AssetTypeNative) {
		r.SellingAssetCode = bCode
		r.SellingAssetIssuer = bIssuer
	}
	if cType != string(horizonclient.AssetTypeNative) {
		r.BuyingAssetCode = cCode
		r.BuyingAssetIssuer = cIssuer
	}

	return r
}
