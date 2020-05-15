package gql

import (
	"errors"
	"fmt"

	"github.com/graph-gophers/graphql-go"
	"github.com/stellar/go/services/ticker/internal/tickerdb"
	"github.com/stellar/go/services/ticker/internal/utils"
)

// Markets resolves the markets() GraphQL query.
func (r *resolver) Markets(args struct {
	BaseAssetCode      *string
	BaseAssetIssuer    *string
	CounterAssetCode   *string
	CounterAssetIssuer *string
	NumHoursAgo        *int32
}) (partialMarkets []*partialMarket, err error) {
	numHours, err := validateNumHoursAgo(args.NumHoursAgo)
	if err != nil {
		return
	}

	var pairName string
	if args.BaseAssetCode != nil {
		pairName = fmt.Sprintf("%s:%s / %s:%s", *args.BaseAssetCode, *args.BaseAssetIssuer, *args.CounterAssetCode, *args.CounterAssetIssuer)
	}

	dbMarkets, err := r.db.RetrievePartialMarkets(
		args.BaseAssetCode,
		args.BaseAssetIssuer,
		args.CounterAssetCode,
		args.CounterAssetIssuer,
		numHours,
	)
	if err != nil {
		// obfuscating sql errors to avoid exposing underlying
		// implementation
		err = errors.New("could not retrieve the requested data")
		return
	}

	for _, dbMkt := range dbMarkets {
		processedMkt := postProcessPartialMarket(dbMarketToPartialMarket(dbMkt), reverseOrderbook(dbMkt), &pairName)
		partialMarkets = append(partialMarkets, processedMkt)
	}
	return
}

// Ticker resolves the ticker() GraphQL query (TODO)
func (r *resolver) Ticker(
	args struct {
		PairName    *string
		NumHoursAgo *int32
	},
) (partialMarkets []*partialMarket, err error) {
	numHours, err := validateNumHoursAgo(args.NumHoursAgo)
	if err != nil {
		return
	}

	dbMarkets, err := r.db.RetrievePartialAggMarkets(args.PairName, numHours)
	if err != nil {
		// obfuscating sql errors to avoid exposing underlying
		// implementation
		err = errors.New("could not retrieve the requested data")
		return
	}

	for _, dbMkt := range dbMarkets {
		processedMkt := postProcessPartialMarket(dbMarketToPartialMarket(dbMkt), reverseOrderbook(dbMkt), args.PairName)
		partialMarkets = append(partialMarkets, processedMkt)
	}
	return

}

// validateNumHoursAgo validates if the numHoursAgo parameter is within an acceptable
// time range (at most 168 hours ago = 7 days)
func validateNumHoursAgo(n *int32) (int, error) {
	if n == nil {
		return 24, nil // default numHours = 24
	}

	if *n <= 168 {
		return int(*n), nil
	}

	return 0, errors.New("numHoursAgo cannot be greater than 168 (7 days)")
}

// dbMarketToPartialMarket converts a tickerdb.PartialMarket to a *partialMarket
func dbMarketToPartialMarket(dbMarket tickerdb.PartialMarket) *partialMarket {
	spread, spreadMidPoint := utils.CalcSpread(dbMarket.HighestBid, dbMarket.LowestAsk)
	os := orderbookStats{
		BidCount:       BigInt(dbMarket.NumBids),
		BidVolume:      dbMarket.BidVolume,
		BidMax:         dbMarket.HighestBid,
		AskCount:       BigInt(dbMarket.NumAsks),
		AskVolume:      dbMarket.AskVolume,
		AskMin:         dbMarket.LowestAsk,
		Spread:         spread,
		SpreadMidPoint: spreadMidPoint,
	}

	return &partialMarket{
		TradePair:            dbMarket.TradePairName,
		BaseAssetCode:        dbMarket.BaseAssetCode,
		BaseAssetIssuer:      dbMarket.BaseAssetIssuer,
		CounterAssetCode:     dbMarket.CounterAssetCode,
		CounterAssetIssuer:   dbMarket.CounterAssetIssuer,
		BaseVolume:           dbMarket.BaseVolume,
		CounterVolume:        dbMarket.CounterVolume,
		TradeCount:           dbMarket.TradeCount,
		Open:                 dbMarket.Open,
		Low:                  dbMarket.Low,
		High:                 dbMarket.High,
		Change:               dbMarket.Change,
		Close:                dbMarket.Close,
		IntervalStart:        graphql.Time{Time: dbMarket.IntervalStart},
		FirstLedgerCloseTime: graphql.Time{Time: dbMarket.FirstLedgerCloseTime},
		LastLedgerCloseTime:  graphql.Time{Time: dbMarket.LastLedgerCloseTime},
		OrderbookStats:       os,
	}
}

func postProcessPartialMarket(
	dbMkt *partialMarket,
	reverseOS orderbookStats,
	oldPairName *string,
) *partialMarket {
	// If a nil partial market was passed, return it.
	if dbMkt == nil {
		return dbMkt
	}

	// Get the requested pair name from the user.
	// If none was provided, the user wants all markets.
	var oldPairNameStr string
	if oldPairName != nil {
		oldPairNameStr = *oldPairName
	}

	// If the user-requested trade pair matches the name
	// of the generated partial market, the market is as expected.
	processedDbMkt := *dbMkt
	if oldPairNameStr == dbMkt.TradePair || oldPairName == nil || oldPairNameStr == "" {
		return &processedDbMkt
	}

	// We swap base code/issuer/volume with counter.
	processedDbMkt.TradePair = oldPairNameStr
	processedDbMkt.BaseAssetCode, processedDbMkt.CounterAssetCode = processedDbMkt.CounterAssetCode, processedDbMkt.BaseAssetCode
	processedDbMkt.BaseAssetIssuer, processedDbMkt.CounterAssetIssuer = processedDbMkt.CounterAssetIssuer, processedDbMkt.BaseAssetIssuer
	processedDbMkt.BaseVolume, processedDbMkt.CounterVolume = processedDbMkt.CounterVolume, processedDbMkt.BaseVolume

	// Since prices are now denominated in counter, we invert the existing ones.
	processedDbMkt.Open = invertIfNonZero(dbMkt.Open)
	processedDbMkt.Low = invertIfNonZero(dbMkt.High)
	processedDbMkt.High = invertIfNonZero(dbMkt.Low)
	processedDbMkt.Change = processedDbMkt.High - processedDbMkt.Low
	processedDbMkt.Close = invertIfNonZero(dbMkt.Close)

	// We substitute the orderbook for the reversed pair.
	processedDbMkt.OrderbookStats = reverseOS
	return &processedDbMkt
}

func reverseOrderbook(dbMarket tickerdb.PartialMarket) orderbookStats {
	spread, spreadMidPoint := utils.CalcSpread(dbMarket.HighestBidReverse, dbMarket.LowestAskReverse)
	os := orderbookStats{
		BidCount:       BigInt(dbMarket.NumBidsReverse),
		BidVolume:      dbMarket.BidVolumeReverse,
		BidMax:         dbMarket.HighestBidReverse,
		AskCount:       BigInt(dbMarket.NumAsksReverse),
		AskVolume:      dbMarket.AskVolumeReverse,
		AskMin:         dbMarket.LowestAskReverse,
		Spread:         spread,
		SpreadMidPoint: spreadMidPoint,
	}
	return os
}

func invertIfNonZero(num float64) float64 {
	if num != 0 {
		return 1 / num
	}
	return num
}
