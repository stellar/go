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
	pool   xdr.LiquidityPoolEntry
}

type searchState interface {
	isTerminalNode(
		currentAsset string,
		currentAssetAmount xdr.Int64,
	) bool

	appendToPaths(
		updatedVisitedList []xdr.Asset,
		currentAsset string,
		currentAssetAmount xdr.Int64,
	)

	// Returns all "venues" (trading opportunities) for a particular asset.
	//
	// The result is grouped by the next asset hop, mapping to a list of offers
	// and a liquidity pool, if one exists for that trading pair.
	venues(currentAsset string) map[string]Venues

	consumeOffers(
		currentAssetAmount xdr.Int64,
		offers []xdr.OfferEntry,
	) (xdr.Asset, xdr.Int64, error)
}

func dfs(
	ctx context.Context,
	state searchState,
	maxPathLength int,
	visited []xdr.Asset,
	visitedPools []xdr.PoolId,
	remainingTerminalNodes int,
	currentAssetString string,
	currentAsset xdr.Asset,
	currentAssetAmount xdr.Int64,
) error {
	// exit early if the context was cancelled
	if err := ctx.Err(); err != nil {
		return err
	}
	if currentAssetAmount <= 0 {
		return nil
	}
	for _, asset := range visited {
		if asset.Equals(currentAsset) {
			return nil
		}
	}

	updatedVisitedList := append(visited, currentAsset)
	if state.isTerminalNode(currentAssetString, currentAssetAmount) {
		state.appendToPaths(
			updatedVisitedList,
			currentAssetString,
			currentAssetAmount,
		)
		remainingTerminalNodes--
	}

	// abort search if we've visited all destination nodes or if we've exceeded
	// maxPathLength
	if remainingTerminalNodes == 0 || len(updatedVisitedList) > maxPathLength {
		return nil
	}

	updatedVisitedPools := make([]xdr.PoolId, len(visitedPools))
	copy(updatedVisitedPools, visitedPools)

	for nextAssetString, venues := range state.venues(currentAssetString) {
		bestExchangeRate := xdr.Int64(0)
		var bestAsset xdr.Asset

		// For each asset, we first evaluate the pool (if any), then offers,
		// because pool exchange rates can only be evaluated with an amount.
		if pool := venues.pool; pool.Body.ConstantProduct != nil {
			found := false
			for _, seenPool := range updatedVisitedPools {
				if seenPool == pool.LiquidityPoolId {
					found = true
				}
			}

			if !found {
				updatedVisitedPools = append(updatedVisitedPools,
					pool.LiquidityPoolId)

				amount, err := makeTrade(pool, currentAsset,
					tradeTypeExpectation, currentAssetAmount)
				otherAsset := getOtherAsset(currentAsset, pool)

				if err == nil { // TODO: Should we differentiate errors?
					bestExchangeRate = amount
					bestAsset = otherAsset
				}
			}
		}

		if offers := venues.offers; len(offers) > 0 {
			nextAsset, nextAssetAmount, err := state.consumeOffers(
				currentAssetAmount,
				offers,
			)

			// We should only error out if the LP trade didn't happen.
			if err != nil {
				if bestExchangeRate == 0 {
					return err
				}

				continue
			}

			// TODO: Move this check into consumeOffers to optimize it.
			//
			// TODO: Should we prefer offers or LPs if the exchange is
			//       equivalent? My gut says offers, because (a) there's no fee
			//       and (b) we reduce the orderbook size.
			if nextAssetAmount <= bestExchangeRate || bestExchangeRate == 0 {
				bestExchangeRate = nextAssetAmount
				bestAsset = nextAsset
			}
		}

		if err := dfs(
			ctx,
			state,
			maxPathLength,
			updatedVisitedList,
			updatedVisitedPools,
			remainingTerminalNodes,
			nextAssetString,
			bestAsset,
			bestExchangeRate,
		); err != nil {
			return err
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
}

func (state *sellingGraphSearchState) isTerminalNode(
	currentAsset string,
	currentAssetAmount xdr.Int64,
) bool {
	targetAssetBalance, ok := state.targetAssets[currentAsset]
	return ok && (!state.validateSourceBalance || targetAssetBalance >= currentAssetAmount)
}

func (state *sellingGraphSearchState) appendToPaths(
	updatedVisitedList []xdr.Asset,
	currentAsset string,
	currentAssetAmount xdr.Int64,
) {
	var interiorNodes []xdr.Asset
	if len(updatedVisitedList) > 2 {
		// reverse updatedVisitedList and skip the first and last elements
		interiorNodes = make([]xdr.Asset, len(updatedVisitedList)-2)
		position := 0
		for i := len(updatedVisitedList) - 2; i >= 1; i-- {
			interiorNodes[position] = updatedVisitedList[i]
			position++
		}
	} else {
		interiorNodes = []xdr.Asset{}
	}

	state.paths = append(state.paths, Path{
		sourceAssetString: currentAsset,
		SourceAmount:      currentAssetAmount,
		SourceAsset:       updatedVisitedList[len(updatedVisitedList)-1],
		InteriorNodes:     interiorNodes,
		DestinationAsset:  state.destinationAsset,
		DestinationAmount: state.destinationAssetAmount,
	})
}

func (state *sellingGraphSearchState) venues(currentAsset string) map[string]Venues {
	result := map[string]Venues{}

	for nextAsset, offers := range state.graph.edgesForSellingAsset[currentAsset] {
		if opp, ok := result[nextAsset]; ok {
			opp.offers = offers
		} else {
			result[nextAsset] = Venues{offers: offers}
		}
	}

	for _, pool := range state.graph.liquidityPoolsForAsset[currentAsset] {
		params := pool.Body.MustConstantProduct().Params
		otherAsset := params.AssetA.String()
		if otherAsset == currentAsset {
			otherAsset = params.AssetB.String()
		}

		if opp, ok := result[otherAsset]; ok {
			opp.pool = pool
		} else {
			result[otherAsset] = Venues{pool: pool}
		}
	}

	return result
}

func (state *sellingGraphSearchState) consumeOffers(
	currentAssetAmount xdr.Int64,
	offers []xdr.OfferEntry,
) (xdr.Asset, xdr.Int64, error) {
	var nextAsset xdr.Asset
	nextAmount, err := consumeOffersForSellingAsset(offers, state.ignoreOffersFrom, currentAssetAmount)
	if err == nil {
		nextAsset = offers[0].Buying
	}

	return nextAsset, nextAmount, err
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
}

func (state *buyingGraphSearchState) isTerminalNode(
	currentAsset string,
	currentAssetAmount xdr.Int64,
) bool {
	return state.targetAssets[currentAsset]
}

func (state *buyingGraphSearchState) appendToPaths(
	updatedVisitedList []xdr.Asset,
	currentAsset string,
	currentAssetAmount xdr.Int64,
) {
	var interiorNodes []xdr.Asset
	if len(updatedVisitedList) > 2 {
		// skip the first and last elements
		interiorNodes = make([]xdr.Asset, len(updatedVisitedList)-2)
		copy(interiorNodes, updatedVisitedList[1:len(updatedVisitedList)-1])
	} else {
		interiorNodes = []xdr.Asset{}
	}

	state.paths = append(state.paths, Path{
		SourceAmount:           state.sourceAssetAmount,
		SourceAsset:            state.sourceAsset,
		InteriorNodes:          interiorNodes,
		DestinationAsset:       updatedVisitedList[len(updatedVisitedList)-1],
		DestinationAmount:      currentAssetAmount,
		destinationAssetString: currentAsset,
	})
}

// TODO
func (state *buyingGraphSearchState) venues(currentAsset string) map[string]Venues {
	result := map[string]Venues{}
	for nextAsset, offers := range state.graph.edgesForBuyingAsset[currentAsset] {
		result[nextAsset] = Venues{offers: offers}
	}
	return result
}

func (state *buyingGraphSearchState) consumeOffers(
	currentAssetAmount xdr.Int64,
	offers []xdr.OfferEntry,
) (xdr.Asset, xdr.Int64, error) {
	var nextAsset xdr.Asset
	nextAmount, err := consumeOffersForBuyingAsset(offers, currentAssetAmount)
	if err == nil {
		nextAsset = offers[0].Selling
	}

	return nextAsset, nextAmount, err
}

func consumeOffersForSellingAsset(
	offers []xdr.OfferEntry,
	ignoreOffersFrom *xdr.AccountId,
	currentAssetAmount xdr.Int64,
) (xdr.Int64, error) {
	totalConsumed := xdr.Int64(0)

	if len(offers) == 0 {
		return totalConsumed, errEmptyOffers
	}

	if currentAssetAmount == 0 {
		return totalConsumed, errAssetAmountIsZero
	}

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
	totalConsumed := xdr.Int64(0)

	if len(offers) == 0 {
		return totalConsumed, errEmptyOffers
	}

	if currentAssetAmount == 0 {
		return totalConsumed, errAssetAmountIsZero
	}

	for i := 0; i < len(offers); i++ {
		n := int64(offers[i].Price.N)
		d := int64(offers[i].Price.D)

		// check if we can spend all of currentAssetAmount on the current offer
		// otherwise consume entire offer and move on to the next one
		amountSold, err := price.MulFractionRoundDown(int64(currentAssetAmount), d, n)
		if err == nil {
			amountSoldXDR := xdr.Int64(amountSold)
			if amountSoldXDR == 0 {
				// we do not have enough of the buying asset to consume the offer
				return -1, nil
			}
			if amountSoldXDR < 0 {
				return -1, errSoldTooMuch
			}
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
