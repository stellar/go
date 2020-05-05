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
	IsNewEndpoint      *bool
}) (partialMarkets []*partialMarket, err error) {
	numHours, err := validateNumHoursAgo(args.NumHoursAgo)
	if err != nil {
		return
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
	var pairName string
	if args.BaseAssetCode != nil {
		pairName = fmt.Sprintf("%s:%s / %s:%s", *args.BaseAssetCode, *args.BaseAssetIssuer, *args.CounterAssetCode, *args.CounterAssetIssuer)
	}
	for _, dbMkt := range dbMarkets {
		partialMarkets = append(partialMarkets, postProcessPartialMarket(dbMarketToPartialMarket(dbMkt), reverseOrderbook(dbMkt), &pairName, args.IsNewEndpoint))
	}
	return
}

// Ticker resolves the ticker() GraphQL query (TODO)
func (r *resolver) Ticker(
	args struct {
		PairName      *string
		NumHoursAgo   *int32
		IsNewEndpoint *bool
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
		partialMarkets = append(partialMarkets, postProcessPartialMarket(dbMarketToPartialMarket(dbMkt), reverseOrderbook(dbMkt), args.PairName, args.IsNewEndpoint))
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
	reverseOs orderbookStats,
	oldPairName *string,
	isNewEndpoint *bool,
) (processedDbMkt *partialMarket) {
	// If the user does not provide the new endpoint flag,
	// we assume they want the pre-existing behavior. (This also
	// assures backwards compatibility.)
	processedDbMkt = dbMkt
	if isNewEndpoint == nil {
		return
	}

	// If the user specifies the original endpoint, then
	// we return the given market.
	if *isNewEndpoint == false {
		return
	}

	// If the user did not provide a trade pair name, then they
	// want all markets. We return the query result's market, following the
	// convention of the original endpoint.
	if oldPairName == nil {
		return
	}

	// If the user-requested trade pair already matches the name
	// of the generated partial market, no post-processing is required.
	if *oldPairName == dbMkt.TradePair || *oldPairName == "" {
		return
	}

	// We construct a partial market with the base and counter assets reversed.
	processedDbMkt = &partialMarket{
		TradePair:            *oldPairName,
		BaseAssetCode:        dbMkt.CounterAssetCode,
		BaseAssetIssuer:      dbMkt.CounterAssetIssuer,
		CounterAssetCode:     dbMkt.BaseAssetCode,
		CounterAssetIssuer:   dbMkt.BaseAssetIssuer,
		BaseVolume:           dbMkt.BaseVolumeReverse,
		CounterVolume:        dbMkt.CounterVolumeReverse,
		TradeCount:           dbMkt.TradeCount,
		Open:                 1 / dbMkt.Open,
		Low:                  1 / dbMkt.High,
		High:                 1 / dbMkt.Low,
		Change:               1/dbMkt.Low - 1/dbMkt.High,
		Close:                1 / dbMkt.Close,
		IntervalStart:        dbMkt.IntervalStart,
		FirstLedgerCloseTime: dbMkt.FirstLedgerCloseTime,
		LastLedgerCloseTime:  dbMkt.LastLedgerCloseTime,
		OrderbookStats:       reverseOs,
	}
	return
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
