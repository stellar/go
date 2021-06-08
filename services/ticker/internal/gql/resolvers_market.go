package gql

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/graph-gophers/graphql-go"
	"github.com/stellar/go/services/ticker/internal/tickerdb"
	"github.com/stellar/go/services/ticker/internal/utils"
)

// Markets resolves the markets() GraphQL query.
func (r *resolver) Markets(ctx context.Context, args struct {
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

	dbMarkets, err := r.db.RetrievePartialMarkets(ctx,
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
		processedMkt := dbMarketToPartialMarket(dbMkt)
		if pairName != "" {
			processedMkt, err = postProcessPartialMarket(dbMarketToPartialMarket(dbMkt), reverseOrderbook(dbMkt), &pairName, nil)
			if err != nil {
				return
			}
		}

		partialMarkets = append(partialMarkets, processedMkt)
	}
	return
}

// Ticker resolves the ticker() GraphQL query (TODO)
func (r *resolver) Ticker(ctx context.Context,
	args struct {
		Code        *string
		PairNames   *[]*string
		NumHoursAgo *int32
	},
) (partialMarkets []*partialMarket, err error) {
	if args.Code != nil && args.PairNames != nil {
		err = errors.New("Code and PairNames cannot both be provided")
		return
	}

	numHours, err := validateNumHoursAgo(args.NumHoursAgo)
	if err != nil {
		return
	}

	dbMarkets, err := r.db.RetrievePartialAggMarkets(ctx, args.Code, args.PairNames, numHours)
	if err != nil {
		// obfuscating sql errors to avoid exposing underlying
		// implementation
		err = errors.New("could not retrieve the requested data")
		return
	}

	for i, dbMkt := range dbMarkets {
		var processedMkt *partialMarket
		processedMkt, err = postProcessPartialMarket(
			dbMarketToPartialMarket(dbMkt),
			reverseOrderbook(dbMkt),
			getPairName(args.PairNames, i),
			args.Code,
		)
		if err != nil {
			return
		}

		partialMarkets = append(partialMarkets, processedMkt)
	}
	return
}

func getPairName(pairNames *[]*string, index int) *string {
	if pairNames == nil {
		return nil
	}
	return (*pairNames)[index]
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
	userPairName *string,
	userCode *string,
) (*partialMarket, error) {
	// Pair name and code cannot both be provided.
	if userPairName != nil && userCode != nil {
		return nil, errors.New("cannot provide both pair name and code")
	}

	// Return the passed-in partial market if the partial market
	// is nil or the user provided neither pair nor code.
	if dbMkt == nil {
		return dbMkt, nil
	}

	if userPairName == nil && userCode == nil {
		return dbMkt, nil
	}

	// If the user-requested trade pair matches the name
	// of the generated partial market, the market is as requested.
	var userPairNameStr string
	if userPairName != nil {
		userPairNameStr = *userPairName
	}

	if userPairNameStr == dbMkt.TradePair {
		return dbMkt, nil
	}

	// If the user-requested code is already the base pair of the market,
	// return the market as is.
	codesMatched, err := userCodeMatchesMarket(userCode, dbMkt)
	if err != nil {
		return nil, err
	}
	if codesMatched {
		return dbMkt, nil
	}

	// If the above conditions are not met, then we must swap base and counter
	// to match the user-requested market.
	// We swap base code/issuer/volume with counter.
	processedDbMkt := *dbMkt
	reversedPair, err := reversePairName(dbMkt.TradePair)
	if err != nil {
		return nil, err
	}
	processedDbMkt.TradePair = reversedPair
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
	return &processedDbMkt, nil
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

func userCodeMatchesMarket(userCodePtr *string, dbMkt *partialMarket) (bool, error) {
	if userCodePtr == nil {
		return false, nil
	}
	userCode := *userCodePtr

	// Depending on the trade pair format, get the correct
	// separator and split the trade pair.
	sep, err := getPairSep(dbMkt.TradePair)
	if err != nil {
		return false, err
	}

	assetsArr := strings.Split(dbMkt.TradePair, sep)
	if len(assetsArr) != 2 {
		return false, errors.New("invalid trade pair in market")
	}
	baseAsset := assetsArr[0]

	// If the asset is of format Code:Issuer, we do an extra split.
	// Else, we've already found the base code.
	baseCode := baseAsset
	if strings.Contains(baseAsset, ":") {
		baseAssetArr := strings.Split(baseAsset, ":")
		if len(baseAssetArr) != 2 {
			return false, errors.New("invalid base asset format in market")
		}
		baseCode = baseAssetArr[0]
	}

	return baseCode == userCode, nil
}

func reversePairName(pairName string) (string, error) {
	sep, err := getPairSep(pairName)
	if err != nil {
		return "", err
	}

	assetsArr := strings.Split(pairName, sep)
	if len(assetsArr) != 2 {
		return "", errors.New("invalid trade pair in market")
	}

	reversedAssetsArr := []string{assetsArr[1], assetsArr[0]}
	reversedPairName := strings.Join(reversedAssetsArr, sep)
	return reversedPairName, nil
}

// Get the appropriate string separator for the pair name, based
// on the ticker endpoint it's from.
func getPairSep(pairName string) (string, error) {
	// " / " indicates a trade pair from the Markets() endpoint.
	// "_" indicates a trade pair from the Ticker() endpoint.
	var sep string
	if strings.Contains(pairName, "/") {
		sep = " / "
	} else if strings.Contains(pairName, "_") {
		sep = "_"
	} else {
		return "", errors.New("could not get sep from trade pair")
	}
	return sep, nil
}
