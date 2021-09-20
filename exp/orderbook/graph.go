package orderbook

import (
	"context"
	"math"
	"math/big"
	"sort"
	"sync"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

var (
	errOfferNotPresent     = errors.New("offer is not present in the order book graph")
	errEmptyOffers         = errors.New("offers is empty")
	errNoVenues            = errors.New("no liquidity pool or offers")
	errAssetAmountIsZero   = errors.New("current asset amount is 0")
	errSoldTooMuch         = errors.New("sold more than current balance")
	errBatchAlreadyApplied = errors.New("cannot apply batched updates more than once")
	errUnexpectedLedger    = errors.New("cannot apply unexpected ledger")
)

type sortByType string

const (
	sortBySourceAsset      sortByType = "source"
	sortByDestinationAsset sortByType = "destination"
)

// trading pair represents two assets that can be exchanged if an order is fulfilled
type tradingPair struct {
	// buyingAsset is obtained by calling offer.Buying.String() where offer is an xdr.OfferEntry
	buyingAsset string
	// sellingAsset is obtained by calling offer.Selling.String() where offer is an xdr.OfferEntry
	sellingAsset string
}

// OBGraph is an interface for orderbook graphs
type OBGraph interface {
	AddOffers(offer ...xdr.OfferEntry)
	AddLiquidityPools(liquidityPool ...xdr.LiquidityPoolEntry)
	Apply(ledger uint32) error
	Discard()
	Offers() []xdr.OfferEntry
	LiquidityPools() []xdr.LiquidityPoolEntry
	RemoveOffer(xdr.Int64) OBGraph
	RemoveLiquidityPool(params xdr.LiquidityPoolConstantProductParameters) OBGraph
	Clear()
}

// OrderBookGraph is an in-memory graph representation of all the offers in the
// Stellar ledger.
type OrderBookGraph struct {
	// edgesForSellingAsset maps an asset to all offers which sell that asset
	// note that each key in the map is obtained by calling
	// offer.Selling.String() where offer is an xdr.OfferEntry
	edgesForSellingAsset map[string]edgeSet
	// edgesForBuyingAsset maps an asset to all offers which buy that asset note
	// that each key in the map is obtained by calling offer.Buying.String()
	// where offer is an xdr.OfferEntry
	edgesForBuyingAsset map[string]edgeSet
	// tradingPairForOffer maps an offer id to the assets which are being
	// exchanged in the given offer
	tradingPairForOffer map[xdr.Int64]tradingPair
	// liquidityPools maps an asset string to any liquidity pools which contain
	// that asset in their reserves. Note that you can make trades for either
	// asset in a pool, and this will have duplicate pool entries for each of
	// the two assets (for efficient lookups).
	liquidityPoolsForAsset map[string][]xdr.LiquidityPoolEntry
	// batchedUpdates is internal batch of updates to this graph. Users can
	// create multiple batches using `Batch()` method but sometimes only one
	// batch is enough.
	batchedUpdates *orderBookBatchedUpdates
	lock           sync.RWMutex
	// the orderbook graph is accurate up to lastLedger
	lastLedger uint32
}

var _ OBGraph = (*OrderBookGraph)(nil)

// NewOrderBookGraph constructs a new OrderBookGraph
func NewOrderBookGraph() *OrderBookGraph {
	graph := &OrderBookGraph{
		edgesForSellingAsset:   map[string]edgeSet{},
		edgesForBuyingAsset:    map[string]edgeSet{},
		tradingPairForOffer:    map[xdr.Int64]tradingPair{},
		liquidityPoolsForAsset: map[string][]xdr.LiquidityPoolEntry{},
	}

	graph.batchedUpdates = graph.batch()
	return graph
}

// AddOffers will queue an operation to add the given offer(s) to the order book
// in the internal batch.
//
// You need to run Apply() to apply all enqueued operations.
func (graph *OrderBookGraph) AddOffers(offers ...xdr.OfferEntry) {
	for _, offer := range offers {
		graph.batchedUpdates.addOffer(offer)
	}
}

// AddLiquidityPools will queue an operation to add the given liquidity pool(s)
// to the order book graph in the internal batch.
//
// You need to run Apply() to apply all enqueued operations.
func (graph *OrderBookGraph) AddLiquidityPools(pools ...xdr.LiquidityPoolEntry) {
	for _, lp := range pools {
		graph.batchedUpdates.addLiquidityPool(lp)
	}
}

// RemoveOffer will queue an operation to remove the given offer from the order
// book in the internal batch.
//
// You need to run Apply() to apply all enqueued operations.
func (graph *OrderBookGraph) RemoveOffer(offerID xdr.Int64) OBGraph {
	graph.batchedUpdates.removeOffer(offerID)
	return graph
}

// RemoveLiquidityPool will queue an operation to remove any liquidity pool that
// has both of the given assets in the internal batch.
//
// You need to run Apply() to apply all enqueued operations.
func (graph *OrderBookGraph) RemoveLiquidityPool(params xdr.LiquidityPoolConstantProductParameters) OBGraph {
	graph.batchedUpdates.removeLiquidityPool(tradingPair{
		params.AssetA.String(),
		params.AssetB.String(),
	})
	return graph
}

// Discard removes all operations which have been queued but not yet applied to the OrderBookGraph
func (graph *OrderBookGraph) Discard() {
	graph.batchedUpdates = graph.batch()
}

// Apply will attempt to apply all the updates in the internal batch to the order book.
// When Apply is successful, a new empty, instance of internal batch will be created.
func (graph *OrderBookGraph) Apply(ledger uint32) error {
	err := graph.batchedUpdates.apply(ledger)
	if err != nil {
		return err
	}
	graph.batchedUpdates = graph.batch()
	return nil
}

// Offers returns a list of offers contained in the order book
func (graph *OrderBookGraph) Offers() []xdr.OfferEntry {
	graph.lock.RLock()
	defer graph.lock.RUnlock()

	var offers []xdr.OfferEntry
	for _, edges := range graph.edgesForSellingAsset {
		for _, offersForEdge := range edges {
			offers = append(offers, offersForEdge...)
		}
	}

	return offers
}

// LiquidityPools returns a list of unique liquidity pools contained in the
// order book graph
func (graph *OrderBookGraph) LiquidityPools() []xdr.LiquidityPoolEntry {
	graph.lock.RLock()
	defer graph.lock.RUnlock()

	// Since we double-store each pool for each of the two assets, we'll need to
	// put some extra effort in to only return one of them here.
	entries := NewIdSet(len(graph.liquidityPoolsForAsset))
	allPools := make([]xdr.LiquidityPoolEntry, 0, len(graph.liquidityPoolsForAsset))

	for _, pools := range graph.liquidityPoolsForAsset {
		for _, pool := range pools {
			if entries.Contains(pool.LiquidityPoolId) {
				continue
			}

			entries.Add(pool.LiquidityPoolId)
			allPools = append(allPools, pool)
		}
	}

	return allPools
}

// Clear removes all offers from the graph.
func (graph *OrderBookGraph) Clear() {
	graph.lock.Lock()
	defer graph.lock.Unlock()

	graph.edgesForSellingAsset = map[string]edgeSet{}
	graph.edgesForBuyingAsset = map[string]edgeSet{}
	graph.tradingPairForOffer = map[xdr.Int64]tradingPair{}
	graph.liquidityPoolsForAsset = map[string][]xdr.LiquidityPoolEntry{}
	graph.batchedUpdates = graph.batch()
	graph.lastLedger = 0
}

// Batch creates a new batch of order book updates which can be applied
// on this graph
func (graph *OrderBookGraph) batch() *orderBookBatchedUpdates {
	return &orderBookBatchedUpdates{
		operations: []orderBookOperation{},
		committed:  false,
		orderbook:  graph,
	}
}

// add inserts a given offer into the order book graph
func (graph *OrderBookGraph) add(offer xdr.OfferEntry) error {
	if _, contains := graph.tradingPairForOffer[offer.OfferId]; contains {
		if err := graph.remove(offer.OfferId); err != nil {
			return errors.Wrap(err, "could not update offer in order book graph")
		}
	}

	sellingAsset := offer.Selling.String()
	buyingAsset := offer.Buying.String()
	graph.tradingPairForOffer[offer.OfferId] = tradingPair{
		buyingAsset:  buyingAsset,
		sellingAsset: sellingAsset,
	}
	if set, ok := graph.edgesForSellingAsset[sellingAsset]; !ok {
		graph.edgesForSellingAsset[sellingAsset] = edgeSet{}
		graph.edgesForSellingAsset[sellingAsset].add(buyingAsset, offer)
	} else {
		set.add(buyingAsset, offer)
	}

	if set, ok := graph.edgesForBuyingAsset[buyingAsset]; !ok {
		graph.edgesForBuyingAsset[buyingAsset] = edgeSet{}
		graph.edgesForBuyingAsset[buyingAsset].add(sellingAsset, offer)
	} else {
		set.add(sellingAsset, offer)
	}

	// if venues, ok := graph.venuesForSellingAsset[sellingAsset]; ok {
	// 	venues.offers = append(venues.offers, offer)
	// } else {
	// 	graph.venuesForBuyingAsset[sellingAsset] = Venues{
	// 		offers: []xdr.OfferEntry{offer},
	// 	}
	// }

	// if venues, ok := graph.venuesForBuyingAsset[buyingAsset]; ok {
	// 	venues.offers = append(venues.offers, offer)
	// } else {
	// 	graph.venuesForBuyingAsset[buyingAsset] = Venues{
	// 		offers: []xdr.OfferEntry{offer},
	// 	}
	// }

	return nil
}

// remove deletes a given offer from the order book graph
func (graph *OrderBookGraph) remove(offerID xdr.Int64) error {
	pair, ok := graph.tradingPairForOffer[offerID]
	if !ok {
		return errOfferNotPresent
	}

	delete(graph.tradingPairForOffer, offerID)

	if set, ok := graph.edgesForSellingAsset[pair.sellingAsset]; !ok {
		return errOfferNotPresent
	} else if !set.remove(offerID, pair.buyingAsset) {
		return errOfferNotPresent
	} else if len(set) == 0 {
		delete(graph.edgesForSellingAsset, pair.sellingAsset)
	}

	if set, ok := graph.edgesForBuyingAsset[pair.buyingAsset]; !ok {
		return errOfferNotPresent
	} else if !set.remove(offerID, pair.sellingAsset) {
		return errOfferNotPresent
	} else if len(set) == 0 {
		delete(graph.edgesForBuyingAsset, pair.buyingAsset)
	}

	return nil
}

// IsEmpty returns true if the orderbook graph is not populated
func (graph *OrderBookGraph) IsEmpty() bool {
	graph.lock.RLock()
	defer graph.lock.RUnlock()

	return len(graph.edgesForSellingAsset) == 0
}

// FindPaths returns a list of payment paths originating from a source account
// and ending with a given destinaton asset and amount.
func (graph *OrderBookGraph) FindPaths(
	ctx context.Context,
	maxPathLength int,
	destinationAsset xdr.Asset,
	destinationAmount xdr.Int64,
	sourceAccountID *xdr.AccountId,
	sourceAssets []xdr.Asset,
	sourceAssetBalances []xdr.Int64,
	validateSourceBalance bool,
	maxAssetsPerPath int,
) ([]Path, uint32, error) {
	destinationAssetString := destinationAsset.String()
	sourceAssetsMap := make(map[string]xdr.Int64, len(sourceAssets))
	for i, sourceAsset := range sourceAssets {
		sourceAssetString := sourceAsset.String()
		sourceAssetsMap[sourceAssetString] = sourceAssetBalances[i]
	}

	searchState := &sellingGraphSearchState{
		graph:                  graph,
		destinationAsset:       destinationAsset,
		destinationAssetAmount: destinationAmount,
		ignoreOffersFrom:       sourceAccountID,
		targetAssets:           sourceAssetsMap,
		validateSourceBalance:  validateSourceBalance,
		paths:                  []Path{},
	}
	graph.lock.RLock()
	err := dfs(
		ctx,
		searchState,
		maxPathLength,
		[]xdr.Asset{},
		[]string{},
		len(sourceAssets),
		destinationAssetString,
		destinationAsset,
		destinationAmount,
	)
	lastLedger := graph.lastLedger
	graph.lock.RUnlock()
	if err != nil {
		return nil, lastLedger, errors.Wrap(err, "could not determine paths")
	}

	paths, err := sortAndFilterPaths(
		searchState.paths,
		maxAssetsPerPath,
		sortBySourceAsset,
	)
	return paths, lastLedger, err
}

// FindFixedPaths returns a list of payment paths where the source and
// destination assets are fixed.
//
// All returned payment paths will start by spending `amountToSpend` of
// `sourceAsset` and will end with some positive balance of `destinationAsset`.
//
// `sourceAccountID` is optional, but if it's provided, then no offers created
// by `sourceAccountID` will be considered when evaluating payment paths.
func (graph *OrderBookGraph) FindFixedPaths(
	ctx context.Context,
	maxPathLength int,
	sourceAsset xdr.Asset,
	amountToSpend xdr.Int64,
	destinationAssets []xdr.Asset,
	maxAssetsPerPath int,
) ([]Path, uint32, error) {
	target := map[string]bool{}
	for _, destinationAsset := range destinationAssets {
		destinationAssetString := destinationAsset.String()
		target[destinationAssetString] = true
	}

	searchState := &buyingGraphSearchState{
		graph:             graph,
		sourceAsset:       sourceAsset,
		sourceAssetAmount: amountToSpend,
		targetAssets:      target,
		paths:             []Path{},
	}
	graph.lock.RLock()
	err := dfs(
		ctx,
		searchState,
		maxPathLength,
		[]xdr.Asset{},
		[]string{},
		len(destinationAssets),
		sourceAsset.String(),
		sourceAsset,
		amountToSpend,
	)
	lastLedger := graph.lastLedger
	graph.lock.RUnlock()
	if err != nil {
		return nil, lastLedger, errors.Wrap(err, "could not determine paths")
	}

	sort.Slice(searchState.paths, func(i, j int) bool {
		return searchState.paths[i].DestinationAmount > searchState.paths[j].DestinationAmount
	})

	paths, err := sortAndFilterPaths(
		searchState.paths,
		maxAssetsPerPath,
		sortByDestinationAsset,
	)
	return paths, lastLedger, err
}

// compareSourceAsset will group payment paths by `SourceAsset`
// paths which spend less `SourceAmount` will appear earlier in the sorting
// if there are multiple paths which spend the same `SourceAmount` then shorter payment paths
// will be prioritized
func compareSourceAsset(allPaths []Path, i, j int) bool {
	if allPaths[i].SourceAsset.Equals(allPaths[j].SourceAsset) {
		if allPaths[i].SourceAmount == allPaths[j].SourceAmount {
			return len(allPaths[i].InteriorNodes) < len(allPaths[j].InteriorNodes)
		}
		return allPaths[i].SourceAmount < allPaths[j].SourceAmount
	}
	return allPaths[i].SourceAssetString() < allPaths[j].SourceAssetString()
}

// compareDestinationAsset will group payment paths by `DestinationAsset`
// paths which deliver a higher `DestinationAmount` will appear earlier in the sorting
// if there are multiple paths which deliver the same `DestinationAmount` then shorter payment paths
// will be prioritized
func compareDestinationAsset(allPaths []Path, i, j int) bool {
	if allPaths[i].DestinationAsset.Equals(allPaths[j].DestinationAsset) {
		if allPaths[i].DestinationAmount == allPaths[j].DestinationAmount {
			return len(allPaths[i].InteriorNodes) < len(allPaths[j].InteriorNodes)
		}
		return allPaths[i].DestinationAmount > allPaths[j].DestinationAmount
	}
	return allPaths[i].DestinationAssetString() < allPaths[j].DestinationAssetString()
}

func sourceAssetEquals(p, otherPath Path) bool {
	return p.SourceAsset.Equals(otherPath.SourceAsset)
}

func destinationAssetEquals(p, otherPath Path) bool {
	return p.DestinationAsset.Equals(otherPath.DestinationAsset)
}

// sortAndFilterPaths sorts the given list of paths using `comparePaths`
// also, we limit the number of paths with the same asset to `maxPathsPerAsset`
func sortAndFilterPaths(
	allPaths []Path,
	maxPathsPerAsset int,
	sortType sortByType,
) ([]Path, error) {
	var comparePaths func([]Path, int, int) bool
	var assetsEqual func(Path, Path) bool

	switch sortType {
	case sortBySourceAsset:
		comparePaths = compareSourceAsset
		assetsEqual = sourceAssetEquals
	case sortByDestinationAsset:
		comparePaths = compareDestinationAsset
		assetsEqual = destinationAssetEquals
	default:
		return nil, errors.New("invalid sort by type")
	}

	sort.Slice(allPaths, func(i, j int) bool {
		return comparePaths(allPaths, i, j)
	})

	filtered := []Path{}
	countForAsset := 0
	for _, entry := range allPaths {
		if len(filtered) == 0 || !assetsEqual(filtered[len(filtered)-1], entry) {
			countForAsset = 1
			filtered = append(filtered, entry)
		} else if countForAsset < maxPathsPerAsset {
			countForAsset++
			filtered = append(filtered, entry)
		}
	}

	return filtered, nil
}

const (
	tradeTypeDeposit     = iota // deposit into pool, what's the payout?
	tradeTypeExpectation = iota // expect payout, what to deposit?
)

// makeTrade simulates execution of an exchange with a liquidity pool.
//
// There are two different exchanges that can be simulated:
//
// 1. You know how much you can *give* to the pool, and are curious about the
// resulting payout. We call this a "deposit", and you should pass
// tradeTypeDeposit.
//
// 2. You know how much you'd like to *receive* from the pool, and want to know
// how much to deposit to achieve this. We call this an "expectation", and you
// should pass tradeTypeExpectation.
//
// In (1), this returns the amount that would be paid out by the pool for
// depositing `amount` of `asset` alongside the new state of the liquidity pool
// after this exchange.
//
// In (2), this returns the amount of `asset` necessary to give to the pool in
// order to get `amount` of the other asset in return.
//
// In both cases, an error can be returned. They occur because of invalid
// assets, pool overflows, etc.
//
// Warning: If you pass an asset that is NOT one of the pool reserves, the
// behavior of this function is undefined (for performance).
//
// Refer to https://github.com/stellar/stellar-protocol/blob/master/core/cap-0038.md#pathpaymentstrictsendop-and-pathpaymentstrictreceiveop
// and the calculation functions (below) for details on the exchange algorithm.
func makeTrade(
	pool xdr.LiquidityPoolEntry,
	asset xdr.Asset,
	tradeType int,
	amount xdr.Int64,
) (xdr.Int64, error) {
	details, ok := pool.Body.GetConstantProduct()
	if !ok {
		return 0, errors.New("Unsupported liquidity pool: must be ConstantProduct")
	}

	if amount <= 0 {
		return 0, errors.New("Exchange amount must be positive")
	}

	// determine which asset `amount` corresponds to
	X, Y := details.ReserveA, details.ReserveB
	if !details.Params.AssetA.Equals(asset) {
		X, Y = details.ReserveB, details.ReserveA
	}

	ok = false
	var result xdr.Int64
	switch tradeType {
	case tradeTypeDeposit:
		result, ok = calculatePoolPayout(X, Y, amount, details.Params.Fee)

	case tradeTypeExpectation:
		result, ok = calculatePoolExpectation(X, Y, amount, details.Params.Fee)
	}

	if !ok {
		return 0, errors.New("Liquidity pool overflows from this exchange")
	}
	return result, nil
}

// calculatePoolPayout calculates the amount disbursed from the pool for an
// amount received. From CAP-38:
//
//      y = floor[(1 - F) Yx / (X + x - Fx)]
//
// It returns false if the calculation overflows.
func calculatePoolPayout(reserveA, reserveB, received xdr.Int64, feeBips xdr.Int32) (xdr.Int64, bool) {
	X, Y := big.NewInt(int64(reserveA)), big.NewInt(int64(reserveB))
	F, x := big.NewInt(int64(feeBips)), big.NewInt(int64(received))

	// would this deposit overflow the reserve?
	if math.MaxInt64-received < reserveA {
		return 0, false
	}

	// We do all of the math in bips, so it's all upscaled by this value.
	maxBips := big.NewInt(10000)
	f := new(big.Int).Sub(maxBips, F) // upscaled 1 - F

	// right half: X + (1 - F)x
	denom := X.Mul(X, maxBips).Add(X, new(big.Int).Mul(x, f))
	if denom.Cmp(big.NewInt(0)) == 0 { // avoid div-by-zero panic
		return 0, false
	}

	// left half, a: (1 - F) Yx
	numer := Y.Mul(Y, x).Mul(Y, f)

	// divide & check overflow
	result := numer.Div(numer, denom)

	return xdr.Int64(result.Int64()), result.IsInt64()
}

// calculatePoolExpectation determines how much you would need to put into a
// pool to get a certain amount disbursed.
//
//      x = ceil[Xy / (Y - y) / (1 - F)]
//
// It returns false if the calculation overflows.
func calculatePoolExpectation(
	reserveA, reserveB, disbursed xdr.Int64, feeBips xdr.Int32,
) (xdr.Int64, bool) {
	X, Y := big.NewInt(int64(reserveA)), big.NewInt(int64(reserveB))
	F, y := big.NewInt(int64(feeBips)), big.NewInt(int64(disbursed))

	// sanity check: disbursing shouldn't underflow the reserve
	if reserveA-disbursed <= 0 {
		return 0, false
	}

	// We do all of the math in bips, so it's all upscaled by this value.
	maxBips := big.NewInt(10000)
	f := new(big.Int).Sub(maxBips, F) // upscaled 1 - F

	denom := Y.Sub(Y, y).Mul(Y, f)     // right half:
	if denom.Cmp(big.NewInt(0)) == 0 { // avoid div-by-zero panic
		return 0, false
	}

	numer := X.Mul(X, y).Mul(X, maxBips)

	result, rem := new(big.Int), new(big.Int)
	result.DivMod(numer, denom, rem)

	// hacky way to ceil(): if there's a remainder, add 1
	if rem.Cmp(big.NewInt(0)) > 0 {
		result.Add(result, big.NewInt(1))
	}

	return xdr.Int64(result.Int64()), result.IsInt64()
}

// getOtherAsset returns the other asset in the liquidity pool. Note that
// doesn't check to make sure the passed in `asset` is actually part of the
// pool; behavior in that case is undefined.
func getOtherAsset(asset xdr.Asset, pool xdr.LiquidityPoolEntry) xdr.Asset {
	cp := pool.Body.MustConstantProduct()
	if cp.Params.AssetA.Equals(asset) {
		return cp.Params.AssetB
	}
	return cp.Params.AssetA
}

func getCode(asset xdr.Asset) string {
	code := asset.GetCode()
	if code == "" {
		return "XLM"
	}
	return code
}
