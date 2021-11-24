package orderbook

import (
	"context"

	"github.com/stellar/go/price"
	"github.com/stellar/go/xdr"
)

// Path represents a payment path from a source asset to some destination asset
type Path struct {
	SourceAsset       xdr.Asset
	SourceAmount      xdr.Int64
	DestinationAsset  xdr.Asset
	DestinationAmount xdr.Int64

	// sourceAssetString and destinationAssetString are included as an
	// optimization to improve the performance of sorting paths by avoiding
	// serializing assets to strings repeatedly
	sourceAssetString      string
	destinationAssetString string

	InteriorNodes []xdr.Asset
}

// SourceAssetString returns the string representation of the path's source asset
func (p *Path) SourceAssetString() string {
	if p.sourceAssetString == "" {
		p.sourceAssetString = p.SourceAsset.String()
	}
	return p.sourceAssetString
}

// DestinationAssetString returns the string representation of the path's destination asset
func (p *Path) DestinationAssetString() string {
	if p.destinationAssetString == "" {
		p.destinationAssetString = p.DestinationAsset.String()
	}
	return p.destinationAssetString
}

type Venues struct {
	offers []xdr.OfferEntry
	pool   xdr.LiquidityPoolEntry // can be empty, check body pointer
}

type searchState interface {
	// considerPools returns true if we will consider liquidity pools in our path
	// finding search.
	considerPools() bool

	// isTerminalNode returns true if the current asset is a terminal node in our
	// path finding search.
	isTerminalNode(asset string) bool

	// includePath returns true if the current path which ends at the given asset
	// and produces the given amount satisfies our search criteria.
	includePath(
		currentAsset string,
		currentAssetAmount xdr.Int64,
	) bool

	// betterPathAmount returns true if alternativeAmount is better than currentAmount
	// Given two paths (current path and alternative path) which lead to the same asset
	// but possibly have different amounts of that asset, betterPathAmount will return
	// true if the alternative path is better than the current path.
	betterPathAmount(currentAmount, alternativeAmount xdr.Int64) bool

	// appendToPaths appends the current path to our result list.
	appendToPaths(
		path []xdr.Asset,
		currentAsset string,
		currentAssetAmount xdr.Int64,
	)

	// venues returns all possible trading opportunities for a particular asset.
	//
	// The result is grouped by the next asset hop, mapping to a sorted list of
	// offers (by price) and a liquidity pool (if one exists for that trading
	// pair).
	venues(currentAsset string) edgeSet

	// consumeOffers will consume the given set of offers to trade our
	// current asset for a different asset.
	consumeOffers(
		currentAssetAmount xdr.Int64,
		currentBestAmount xdr.Int64,
		offers []xdr.OfferEntry,
	) (xdr.Asset, xdr.Int64, error)

	// consumePool will consume the given liquidity pool to trade our
	// current asset for a different asset.
	consumePool(
		pool xdr.LiquidityPoolEntry,
		currentAsset xdr.Asset,
		currentAssetAmount xdr.Int64,
	) (xdr.Int64, error)
}

type pathNode struct {
	assetString string
	asset       xdr.Asset
	prev        *pathNode
}

func (p *pathNode) contains(src, dst string) bool {
	for cur := p; cur != nil && cur.prev != nil; cur = cur.prev {
		if cur.assetString == dst && cur.prev.assetString == src {
			return true
		}
	}
	return false
}

func reversePath(path []xdr.Asset) {
	for i := len(path)/2 - 1; i >= 0; i-- {
		opp := len(path) - 1 - i
		path[i], path[opp] = path[opp], path[i]
	}
}

func (e *pathNode) path() []xdr.Asset {
	var result []xdr.Asset
	for cur := e; cur != nil; cur = cur.prev {
		result = append(result, cur.asset)
	}

	reversePath(result)
	return result
}

func search(
	ctx context.Context,
	state searchState,
	maxPathLength int,
	sourceAssetString string,
	sourceAsset xdr.Asset,
	sourceAssetAmount xdr.Int64,
) error {
	bestAmount := map[string]xdr.Int64{sourceAssetString: sourceAssetAmount}
	updateAmount := map[string]xdr.Int64{sourceAssetString: sourceAssetAmount}
	bestPath := map[string]*pathNode{sourceAssetString: {
		assetString: sourceAssetString,
		asset:       sourceAsset,
		prev:        nil,
	}}

	for i := 0; i < maxPathLength; i++ {
		updatePath := map[string]*pathNode{}

		for currentAssetString, currentAmount := range bestAmount {
			pathToCurrentAsset := bestPath[currentAssetString]
			currentAsset := pathToCurrentAsset.asset
			edges := state.venues(currentAssetString)
			for j := 0; j < len(edges); j++ {
				// Exit early if the context was cancelled.
				if err := ctx.Err(); err != nil {
					return err
				}
				nextAssetString, venues := edges[j].key, edges[j].value

				// If we're on our last step ignore any edges which don't lead to
				// our desired destination. This optimization will save us from
				// doing wasted computation.
				if i == maxPathLength-1 && !state.isTerminalNode(nextAssetString) {
					continue
				}

				// Make sure we don't use an edge more than once.
				if pathToCurrentAsset.contains(currentAssetString, nextAssetString) {
					continue
				}

				nextAsset, nextAssetAmount, err := processVenues(state, currentAsset, currentAmount, venues)
				if err != nil {
					return err
				}
				if nextAssetAmount <= 0 {
					continue
				}

				if bestNextAssetAmount, ok := updateAmount[nextAssetString]; !ok || state.betterPathAmount(bestNextAssetAmount, nextAssetAmount) {
					updateAmount[nextAssetString] = nextAssetAmount

					if pathToNext, ok := updatePath[nextAssetString]; ok {
						pathToNext.prev = pathToCurrentAsset
					} else {
						updatePath[nextAssetString] = &pathNode{
							assetString: nextAssetString,
							asset:       nextAsset,
							prev:        pathToCurrentAsset,
						}
					}

					// We could avoid this step until the last iteration, but we would
					// like to include multiple paths in the response to give the user
					// other options in case the best path is already consumed.
					if state.includePath(nextAssetString, nextAssetAmount) {
						state.appendToPaths(
							append(bestPath[currentAssetString].path(), nextAsset),
							nextAssetString,
							nextAssetAmount,
						)
					}
				}
			}
		}

		// Only update bestPath and bestAmount if we have more iterations left in
		// the algorithm. This optimization will save us from doing wasted
		// computation.
		if i < maxPathLength-1 {
			for assetString, betterPath := range updatePath {
				bestPath[assetString] = betterPath
				bestAmount[assetString] = updateAmount[assetString]
			}
		}
	}

	return nil
}

// sellingGraphSearchState configures a DFS on the orderbook graph where only
// edges in `graph.edgesForSellingAsset` are traversed.
//
// The DFS maintains the following invariants:
//  - no node is repeated
//  - no offers are consumed from the `ignoreOffersFrom` account
//  - each payment path must begin with an asset in `targetAssets`
//  - also, the required source asset amount cannot exceed the balance in
//    `targetAssets`
type sellingGraphSearchState struct {
	graph                  *OrderBookGraph
	destinationAsset       xdr.Asset
	destinationAssetAmount xdr.Int64
	ignoreOffersFrom       *xdr.AccountId
	targetAssets           map[string]xdr.Int64
	validateSourceBalance  bool
	paths                  []Path
	includePools           bool
}

func (state *sellingGraphSearchState) isTerminalNode(currentAsset string) bool {
	_, ok := state.targetAssets[currentAsset]
	return ok
}

func (state *sellingGraphSearchState) includePath(currentAsset string, currentAssetAmount xdr.Int64) bool {
	targetAssetBalance, ok := state.targetAssets[currentAsset]
	return ok && (!state.validateSourceBalance || targetAssetBalance >= currentAssetAmount)
}

func (state *sellingGraphSearchState) betterPathAmount(currentAmount, alternativeAmount xdr.Int64) bool {
	return alternativeAmount < currentAmount
}

func (state *sellingGraphSearchState) appendToPaths(
	path []xdr.Asset,
	currentAssetString string,
	currentAssetAmount xdr.Int64,
) {
	currentAsset := path[len(path)-1]
	if len(path) > 2 {
		path = path[1 : len(path)-1]
		reversePath(path)
	} else {
		path = []xdr.Asset{}
	}

	state.paths = append(state.paths, Path{
		sourceAssetString: currentAssetString,
		SourceAmount:      currentAssetAmount,
		SourceAsset:       currentAsset,
		InteriorNodes:     path,
		DestinationAsset:  state.destinationAsset,
		DestinationAmount: state.destinationAssetAmount,
	})
}

func (state *sellingGraphSearchState) venues(currentAsset string) edgeSet {
	return state.graph.venuesForSellingAsset[currentAsset]
}

func (state *sellingGraphSearchState) consumeOffers(
	currentAssetAmount xdr.Int64,
	currentBestAmount xdr.Int64,
	offers []xdr.OfferEntry,
) (xdr.Asset, xdr.Int64, error) {
	nextAmount, err := consumeOffersForSellingAsset(
		offers, state.ignoreOffersFrom, currentAssetAmount, currentBestAmount)

	var nextAsset xdr.Asset
	if len(offers) > 0 {
		nextAsset = offers[0].Buying
	}

	return nextAsset, positiveMin(currentBestAmount, nextAmount), err
}

func (state *sellingGraphSearchState) considerPools() bool {
	return state.includePools
}

func (state *sellingGraphSearchState) consumePool(
	pool xdr.LiquidityPoolEntry,
	currentAsset xdr.Asset,
	currentAssetAmount xdr.Int64,
) (xdr.Int64, error) {
	// How many of the previous hop do we need to get this amount?
	return makeTrade(pool, getOtherAsset(currentAsset, pool),
		tradeTypeExpectation, currentAssetAmount)
}

// buyingGraphSearchState configures a DFS on the orderbook graph where only
// edges in `graph.edgesForBuyingAsset` are traversed.
//
// The DFS maintains the following invariants:
//  - no node is repeated
//  - no offers are consumed from the `ignoreOffersFrom` account
//  - each payment path must terminate with an asset in `targetAssets`
//  - each payment path must begin with `sourceAsset`
type buyingGraphSearchState struct {
	graph             *OrderBookGraph
	sourceAsset       xdr.Asset
	sourceAssetAmount xdr.Int64
	targetAssets      map[string]bool
	paths             []Path
	includePools      bool
}

func (state *buyingGraphSearchState) isTerminalNode(currentAsset string) bool {
	return state.targetAssets[currentAsset]
}

func (state *buyingGraphSearchState) includePath(currentAsset string, currentAssetAmount xdr.Int64) bool {
	return state.targetAssets[currentAsset]
}

func (state *buyingGraphSearchState) betterPathAmount(currentAmount, alternativeAmount xdr.Int64) bool {
	return alternativeAmount > currentAmount
}

func (state *buyingGraphSearchState) appendToPaths(
	path []xdr.Asset,
	currentAssetString string,
	currentAssetAmount xdr.Int64,
) {
	currentAsset := path[len(path)-1]
	if len(path) > 2 {
		path = path[1 : len(path)-1]
	} else {
		path = []xdr.Asset{}
	}

	state.paths = append(state.paths, Path{
		SourceAmount:           state.sourceAssetAmount,
		SourceAsset:            state.sourceAsset,
		InteriorNodes:          path,
		DestinationAsset:       currentAsset,
		DestinationAmount:      currentAssetAmount,
		destinationAssetString: currentAssetString,
	})
}

func (state *buyingGraphSearchState) venues(currentAsset string) edgeSet {
	return state.graph.venuesForBuyingAsset[currentAsset]
}

func (state *buyingGraphSearchState) consumeOffers(
	currentAssetAmount xdr.Int64,
	currentBestAmount xdr.Int64,
	offers []xdr.OfferEntry,
) (xdr.Asset, xdr.Int64, error) {
	nextAmount, err := consumeOffersForBuyingAsset(offers, currentAssetAmount)

	var nextAsset xdr.Asset
	if len(offers) > 0 {
		nextAsset = offers[0].Selling
	}

	return nextAsset, max(nextAmount, currentBestAmount), err
}

func (state *buyingGraphSearchState) considerPools() bool {
	return state.includePools
}

func (state *buyingGraphSearchState) consumePool(
	pool xdr.LiquidityPoolEntry,
	currentAsset xdr.Asset,
	currentAssetAmount xdr.Int64,
) (xdr.Int64, error) {
	return makeTrade(pool, currentAsset, tradeTypeDeposit, currentAssetAmount)
}

func consumeOffersForSellingAsset(
	offers []xdr.OfferEntry,
	ignoreOffersFrom *xdr.AccountId,
	currentAssetAmount xdr.Int64,
	currentBestAmount xdr.Int64,
) (xdr.Int64, error) {
	if len(offers) == 0 {
		return 0, errEmptyOffers
	}

	if currentAssetAmount == 0 {
		return 0, errAssetAmountIsZero
	}

	totalConsumed := xdr.Int64(0)
	for i := 0; i < len(offers); i++ {
		if ignoreOffersFrom != nil && ignoreOffersFrom.Equals(offers[i].SellerId) {
			continue
		}

		buyingUnitsFromOffer, sellingUnitsFromOffer, err := price.ConvertToBuyingUnits(
			int64(offers[i].Amount),
			int64(currentAssetAmount),
			int64(offers[i].Price.N),
			int64(offers[i].Price.D),
		)
		if err == price.ErrOverflow {
			// skip paths which would result in overflow errors
			// but still continue the path finding search
			return -1, nil
		} else if err != nil {
			return -1, err
		}

		totalConsumed += xdr.Int64(buyingUnitsFromOffer)

		// For sell-state, we are aiming to *minimize* the amount of the source
		// assets we need to get to the destination, so if we exceed the best
		// amount, it's time to bail.
		//
		// FIXME: Evaluate if this can work, and if it's actually performant.
		// if totalConsumed >= currentBestAmount && currentBestAmount > 0 {
		// 	return currentBestAmount, nil
		// }

		currentAssetAmount -= xdr.Int64(sellingUnitsFromOffer)

		if currentAssetAmount == 0 {
			return totalConsumed, nil
		}
		if currentAssetAmount < 0 {
			return -1, errSoldTooMuch
		}
	}

	return -1, nil
}

func consumeOffersForBuyingAsset(
	offers []xdr.OfferEntry,
	currentAssetAmount xdr.Int64,
) (xdr.Int64, error) {
	if len(offers) == 0 {
		return 0, errEmptyOffers
	}

	if currentAssetAmount == 0 {
		return 0, errAssetAmountIsZero
	}

	totalConsumed := xdr.Int64(0)
	for i := 0; i < len(offers); i++ {
		n := int64(offers[i].Price.N)
		d := int64(offers[i].Price.D)

		// check if we can spend all of currentAssetAmount on the current offer
		// otherwise consume entire offer and move on to the next one
		amountSold, err := price.MulFractionRoundDown(int64(currentAssetAmount), d, n)
		if err == nil {
			if amountSold == 0 {
				// not enough of the buying asset to consume the offer
				return -1, nil
			}
			if amountSold < 0 {
				return -1, errSoldTooMuch
			}

			amountSoldXDR := xdr.Int64(amountSold)
			if amountSoldXDR <= offers[i].Amount {
				totalConsumed += amountSoldXDR
				return totalConsumed, nil
			}
		} else if err != price.ErrOverflow {
			return -1, err
		}

		buyingUnitsFromOffer, sellingUnitsFromOffer, err := price.ConvertToBuyingUnits(
			int64(offers[i].Amount),
			int64(offers[i].Amount),
			n,
			d,
		)
		if err == price.ErrOverflow {
			// skip paths which would result in overflow errors
			// but still continue the path finding search
			return -1, nil
		} else if err != nil {
			return -1, err
		}

		totalConsumed += xdr.Int64(sellingUnitsFromOffer)
		currentAssetAmount -= xdr.Int64(buyingUnitsFromOffer)

		if currentAssetAmount == 0 {
			return totalConsumed, nil
		}
		if currentAssetAmount < 0 {
			return -1, errSoldTooMuch
		}
	}

	return -1, nil
}

func processVenues(
	state searchState,
	currentAsset xdr.Asset,
	currentAssetAmount xdr.Int64,
	venues Venues,
) (xdr.Asset, xdr.Int64, error) {
	var nextAsset xdr.Asset

	if currentAssetAmount == 0 {
		return nextAsset, 0, errAssetAmountIsZero
	}

	// We evaluate the pool venue (if any) before offers, because pool exchange
	// rates can only be evaluated with an amount.
	poolAmount := xdr.Int64(0)
	if pool := venues.pool; state.considerPools() && pool.Body.ConstantProduct != nil {
		amount, err := state.consumePool(pool, currentAsset, currentAssetAmount)
		if err == nil {
			nextAsset = getOtherAsset(currentAsset, pool)
			poolAmount = amount
		}
		// It's only a true error if the offers fail later, too
	}

	if poolAmount == 0 && len(venues.offers) == 0 {
		return nextAsset, -1, nil // not really an error
	}

	// This will return the pool amount if the LP performs better.
	offerAsset, nextAssetAmount, err := state.consumeOffers(
		currentAssetAmount, poolAmount, venues.offers)

	// Only error out the offers if the LP trade didn't happen.
	if err != nil && poolAmount == 0 {
		return nextAsset, 0, err
	} else if err == nil {
		nextAsset = offerAsset
	}

	return nextAsset, nextAssetAmount, nil
}
