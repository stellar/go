package orderbook

import (
	"context"
	"fmt"

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

	processVenues(
		asset xdr.Asset,
		assetAmount xdr.Int64,
		venues Venues,
	) (xdr.Asset, xdr.Int64, error)

	consumeOffers(
		currentAssetAmount xdr.Int64,
		offers []xdr.OfferEntry,
	) (xdr.Asset, xdr.Int64, error)
}

func contains(list []string, want string) bool {
	for i := 0; i < len(list); i++ {
		if list[i] == want {
			return true
		}
	}
	return false
}

func dfs(
	ctx context.Context,
	state searchState,
	maxPathLength int,
	visitedAssets []xdr.Asset,
	visitedAssetStrings []string,
	remainingTerminalNodes int,
	currentAssetString string,
	currentAsset xdr.Asset,
	currentAssetAmount xdr.Int64,
) error {
	// exit early if the context was cancelled
	if err := ctx.Err(); err != nil {
		return err
	}

	updatedVisitedAssets := append(visitedAssets, currentAsset)
	updatedVisitedStrings := append(visitedAssetStrings, currentAssetString)

	if state.isTerminalNode(currentAssetString, currentAssetAmount) {
		state.appendToPaths(
			updatedVisitedAssets,
			currentAssetString,
			currentAssetAmount,
		)
		remainingTerminalNodes--
	}

	// abort search if we've visited all destination nodes or if we've exceeded
	// maxPathLength
	if remainingTerminalNodes == 0 || len(updatedVisitedStrings) > maxPathLength {
		return nil
	}

	fmt.Println("Evaluating neighbors of", getCode(currentAsset))
	for nextAssetString, venues := range state.venues(currentAssetString) {
		fmt.Println("  evaluating", nextAssetString)
		if contains(visitedAssetStrings, nextAssetString) {
			fmt.Println("    skipping")
			continue
		}

		// Notice that we do evaluate LPs for assets we may have already
		// visited, because the amounts change depending on what we're
		// offering to the LP, and asking for X of asset Y is not the same
		// asking for X of asset Z in a Y<-->Z pool.
		//
		// TODO: I only *think* this is true, and should add a test case
		//       that demonstrates this.
		nextAsset, nextAssetAmount, err := state.processVenues(
			currentAsset, currentAssetAmount, venues)
		if err != nil {
			return err
		}

		if nextAssetAmount <= 0 { // avoid unnecessary extra recursion
			continue
		}

		fmt.Printf("To get %s for %d %s -> %d\n",
			getCode(nextAsset),
			currentAssetAmount, getCode(currentAsset),
			nextAssetAmount)

		if err := dfs(
			ctx,
			state,
			maxPathLength,
			updatedVisitedAssets,
			updatedVisitedStrings,
			remainingTerminalNodes,
			nextAssetString,
			nextAsset,
			nextAssetAmount,
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
	result := make(map[string]Venues,
		// we're overshooting by up to 2x, but it's still a reasonable size hint
		len(state.graph.edgesForSellingAsset)+len(state.graph.liquidityPoolsForAsset))

	for nextAsset, offers := range state.graph.edgesForSellingAsset[currentAsset] {
		result[nextAsset] = Venues{offers: offers}
	}

	// FIXME: I really don't like this whole "check or set" approach; lookups
	//        are suboptimal and the code itself isn't clean. This needs a
	//        refactor either way (deeper integration into the graph, I think),
	//        so it'll get resolved eventually.
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

func (state *sellingGraphSearchState) shouldIgnoreOffer(offer xdr.OfferEntry) bool {
	return state.ignoreOffersFrom != nil &&
		state.ignoreOffersFrom.Equals(offer.SellerId)
}

func (state *sellingGraphSearchState) processVenues(
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
	expectedPoolInput := xdr.Int64(0)
	if pool := venues.pool; pool.Body.ConstantProduct != nil {
		otherAsset := getOtherAsset(currentAsset, pool)

		amount, err := makeTrade(pool, currentAsset,
			tradeTypeExpectation, currentAssetAmount)

		if err == nil { // TODO: Should we differentiate errors?
			expectedPoolInput = amount
			nextAsset = otherAsset
		}
		fmt.Println("Evaluating pool", err)
	}

	fmt.Println("Pool needs", expectedPoolInput)

	offers := venues.offers
	if expectedPoolInput == 0 && len(offers) == 0 {
		return nextAsset, 0, errNoVenues
	}

	fmt.Println(offers)

	nextAssetAmount := expectedPoolInput
	offerAsset, offerAmount, err := state.consumeOffers(
		currentAssetAmount,
		offers,
	)

	if offerAmount <= 0 || err != nil {
		// We should only error out the offers if the LP trade didn't happen.
		if expectedPoolInput == 0 {
			return nextAsset, 0, err
		}
		// Otherwise, evaluate the offers against the pool trade.
		// TODO: Move this into consumeOffers.
	} else {
		if offerAmount < expectedPoolInput || expectedPoolInput == 0 { // offers outperform pool!
			nextAsset = offerAsset
			nextAssetAmount = offerAmount
		}
	}

	return nextAsset, nextAssetAmount, nil
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

// TODO: Behaves like the former `edges()` for now.
func (state *buyingGraphSearchState) venues(currentAsset string) map[string]Venues {
	edges := state.graph.edgesForBuyingAsset[currentAsset]
	result := make(map[string]Venues, len(edges))
	for nextAsset, offers := range edges {
		result[nextAsset] = Venues{offers: offers}
	}
	return result
}

func (state *buyingGraphSearchState) processVenues(
	currentAsset xdr.Asset,
	currentAssetAmount xdr.Int64,
	venues Venues,
) (xdr.Asset, xdr.Int64, error) {
	return state.consumeOffers(currentAssetAmount, venues.offers)
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
