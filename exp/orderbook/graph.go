package orderbook

import (
	"context"
	"sort"
	"sync"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

var (
	errOfferNotPresent     = errors.New("offer is not present in the order book graph")
	errEmptyOffers         = errors.New("offers is empty")
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
	// buyingAsset corresponds to offer.Buying.String() from an xdr.OfferEntry
	buyingAsset int32
	// sellingAsset corresponds to offer.Selling.String() from an xdr.OfferEntry
	sellingAsset int32
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
	RemoveLiquidityPool(pool xdr.LiquidityPoolEntry) OBGraph
	Clear()
}

// OrderBookGraph is an in-memory graph representation of all the offers in the
// Stellar ledger.
type OrderBookGraph struct {
	// idToAssetString maps an int32 asset id to its string representation.
	// Every asset on the OrderBookGraph has an int32 id which indexes into idToAssetString.
	// The asset integer ids are largely contiguous. When an asset is completely removed
	// from the OrderBookGraph the integer id for that asset will be assigned to the next
	// asset which is added to the OrderBookGraph.
	idToAssetString []string
	// assetStringToID maps an asset string to its int32 id.
	assetStringToID map[string]int32
	// vacantIDs is a list of int32 asset ids which can be mapped to new assets.
	// When a new asset is added to the OrderBookGraph we first check if there are
	// any available vacantIDs, if so, we will assign the new asset to one of the vacantIDs.
	// Otherwise, we will add a new entry to idToAssetString for the new asset.
	vacantIDs []int32

	// venuesForBuyingAsset maps an asset to all of its buying opportunities,
	// which may be offers (sorted by price) or a liquidity pools.
	venuesForBuyingAsset []edgeSet
	// venuesForSellingAsset maps an asset to all of its *selling* opportunities,
	// which may be offers (sorted by price) or a liquidity pools.
	venuesForSellingAsset []edgeSet
	// liquidityPools associates a particular asset pair (in "asset order", see
	// xdr.Asset.LessThan) with a liquidity pool.
	liquidityPools map[tradingPair]xdr.LiquidityPoolEntry
	// tradingPairForOffer maps an offer ID to the assets which are being
	// exchanged in the given offer. It's mostly used privately in order to
	// associate specific offers with their respective edges in the graph.
	tradingPairForOffer map[xdr.Int64]tradingPair

	// batchedUpdates is internal batch of updates to this graph. Users can
	// create multiple batches using `Batch()` method but sometimes only one
	// batch is enough.
	batchedUpdates *orderBookBatchedUpdates
	lock           sync.RWMutex
	// the orderbook graph is accurate up to lastLedger
	lastLedger uint32
}

var _ OBGraph = (*OrderBookGraph)(nil)

// NewOrderBookGraph constructs an empty OrderBookGraph
func NewOrderBookGraph() *OrderBookGraph {
	graph := &OrderBookGraph{}
	graph.Clear()
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

// RemoveLiquidityPool will queue an operation to remove any liquidity pool (if
// any) that matches the given pool, based exclusively on the pool ID.
//
// You need to run Apply() to apply all enqueued operations.
func (graph *OrderBookGraph) RemoveLiquidityPool(pool xdr.LiquidityPoolEntry) OBGraph {
	graph.batchedUpdates.removeLiquidityPool(pool)
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
	for _, edges := range graph.venuesForSellingAsset {
		for _, edge := range edges {
			offers = append(offers, edge.value.offers...)
		}
	}

	return offers
}

// LiquidityPools returns a list of unique liquidity pools contained in the
// order book graph
func (graph *OrderBookGraph) LiquidityPools() []xdr.LiquidityPoolEntry {
	graph.lock.RLock()
	defer graph.lock.RUnlock()

	pools := make([]xdr.LiquidityPoolEntry, 0, len(graph.liquidityPools))
	for _, pool := range graph.liquidityPools {
		pools = append(pools, pool)
	}

	return pools
}

// Clear removes all offers from the graph.
func (graph *OrderBookGraph) Clear() {
	graph.lock.Lock()
	defer graph.lock.Unlock()

	graph.assetStringToID = map[string]int32{}
	graph.idToAssetString = []string{}
	graph.vacantIDs = []int32{}
	graph.venuesForSellingAsset = []edgeSet{}
	graph.venuesForBuyingAsset = []edgeSet{}
	graph.tradingPairForOffer = map[xdr.Int64]tradingPair{}
	graph.liquidityPools = map[tradingPair]xdr.LiquidityPoolEntry{}
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

func (graph *OrderBookGraph) getOrCreateAssetID(asset xdr.Asset) int32 {
	assetString := asset.String()
	id, ok := graph.assetStringToID[assetString]
	if ok {
		return id
	}
	// before creating a new int32 asset id we will try to use
	// a vacant id so that we can plug any empty cells in the
	// idToAssetString array.
	if len(graph.vacantIDs) > 0 {
		id = graph.vacantIDs[len(graph.vacantIDs)-1]
		graph.vacantIDs = graph.vacantIDs[:len(graph.vacantIDs)-1]
		graph.idToAssetString[id] = assetString
	} else {
		// idToAssetString never decreases in length unless we call graph.Clear()
		id = int32(len(graph.idToAssetString))
		// we assign id to asset
		graph.idToAssetString = append(graph.idToAssetString, assetString)
		graph.venuesForBuyingAsset = append(graph.venuesForBuyingAsset, nil)
		graph.venuesForSellingAsset = append(graph.venuesForSellingAsset, nil)
	}

	graph.assetStringToID[assetString] = id
	return id
}

func (graph *OrderBookGraph) maybeDeleteAsset(asset int32) {
	buyingEdgesEmpty := len(graph.venuesForBuyingAsset[asset]) == 0
	sellingEdgesEmpty := len(graph.venuesForSellingAsset[asset]) == 0

	if buyingEdgesEmpty && sellingEdgesEmpty {
		delete(graph.assetStringToID, graph.idToAssetString[asset])
		// When removing an asset we do not resize the idToAssetString array.
		// Instead, we allow the cell occupied by the id to be empty.
		// The next time we will add an asset to the graph we will allocate the
		// id to the new asset.
		graph.idToAssetString[asset] = ""
		graph.vacantIDs = append(graph.vacantIDs, asset)
	}
}

// addOffer inserts a given offer into the order book graph
func (graph *OrderBookGraph) addOffer(offer xdr.OfferEntry) error {
	// If necessary, replace any existing offer with a new one.
	if _, contains := graph.tradingPairForOffer[offer.OfferId]; contains {
		if err := graph.removeOffer(offer.OfferId); err != nil {
			return errors.Wrap(err, "could not update offer in order book graph")
		}
	}

	buying := graph.getOrCreateAssetID(offer.Buying)
	selling := graph.getOrCreateAssetID(offer.Selling)

	graph.tradingPairForOffer[offer.OfferId] = tradingPair{
		buyingAsset: buying, sellingAsset: selling,
	}

	graph.venuesForSellingAsset[selling] = graph.venuesForSellingAsset[selling].addOffer(buying, offer)
	graph.venuesForBuyingAsset[buying] = graph.venuesForBuyingAsset[buying].addOffer(selling, offer)

	return nil
}

func (graph *OrderBookGraph) poolFromEntry(poolXDR xdr.LiquidityPoolEntry) liquidityPool {
	aXDR, bXDR := getPoolAssets(poolXDR)
	assetA, assetB := graph.getOrCreateAssetID(aXDR), graph.getOrCreateAssetID(bXDR)
	return liquidityPool{
		LiquidityPoolEntry: poolXDR,
		assetA:             assetA,
		assetB:             assetB,
	}
}

// addPool sets the given pool as the venue for the given trading pair.
func (graph *OrderBookGraph) addPool(poolEntry xdr.LiquidityPoolEntry) {
	// Liquidity pools have no concept of a "buying" or "selling" asset,
	// so we create venues in both directions.
	pool := graph.poolFromEntry(poolEntry)
	graph.liquidityPools[tradingPair{
		buyingAsset:  pool.assetA,
		sellingAsset: pool.assetB,
	}] = pool.LiquidityPoolEntry

	for _, table := range [][]edgeSet{
		graph.venuesForBuyingAsset,
		graph.venuesForSellingAsset,
	} {
		table[pool.assetA] = table[pool.assetA].addPool(pool.assetB, pool)
		table[pool.assetB] = table[pool.assetB].addPool(pool.assetA, pool)
	}
}

// removeOffer deletes a given offer from the order book graph
func (graph *OrderBookGraph) removeOffer(offerID xdr.Int64) error {
	pair, ok := graph.tradingPairForOffer[offerID]
	if !ok {
		return errOfferNotPresent
	}
	delete(graph.tradingPairForOffer, offerID)

	if set, ok := graph.venuesForSellingAsset[pair.sellingAsset].removeOffer(pair.buyingAsset, offerID); !ok {
		return errOfferNotPresent
	} else {
		graph.venuesForSellingAsset[pair.sellingAsset] = set
	}

	if set, ok := graph.venuesForBuyingAsset[pair.buyingAsset].removeOffer(pair.sellingAsset, offerID); !ok {
		return errOfferNotPresent
	} else {
		graph.venuesForBuyingAsset[pair.buyingAsset] = set
	}

	graph.maybeDeleteAsset(pair.buyingAsset)
	graph.maybeDeleteAsset(pair.sellingAsset)
	return nil
}

// removePool unsets the pool matching the given asset pair, if it exists.
func (graph *OrderBookGraph) removePool(poolXDR xdr.LiquidityPoolEntry) {
	aXDR, bXDR := getPoolAssets(poolXDR)
	assetA, assetB := graph.getOrCreateAssetID(aXDR), graph.getOrCreateAssetID(bXDR)

	for _, asset := range []int32{assetA, assetB} {
		otherAsset := assetB
		if asset == assetB {
			otherAsset = assetA
		}

		for _, table := range [][]edgeSet{
			graph.venuesForBuyingAsset,
			graph.venuesForSellingAsset,
		} {
			table[asset] = table[asset].removePool(otherAsset)
		}
	}

	delete(graph.liquidityPools, tradingPair{assetA, assetB})
	graph.maybeDeleteAsset(assetA)
	graph.maybeDeleteAsset(assetB)
}

// IsEmpty returns true if the orderbook graph is not populated
func (graph *OrderBookGraph) IsEmpty() bool {
	graph.lock.RLock()
	defer graph.lock.RUnlock()

	return len(graph.liquidityPools) == 0 && len(graph.tradingPairForOffer) == 0
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
	includePools bool,
) ([]Path, uint32, error) {
	destinationAssetString := destinationAsset.String()
	sourceAssetsMap := make(map[int32]xdr.Int64, len(sourceAssets))
	for i, sourceAsset := range sourceAssets {
		sourceAssetString := sourceAsset.String()
		sourceAssetsMap[graph.assetStringToID[sourceAssetString]] = sourceAssetBalances[i]
	}

	searchState := &sellingGraphSearchState{
		graph:                  graph,
		destinationAssetString: destinationAssetString,
		destinationAssetAmount: destinationAmount,
		ignoreOffersFrom:       sourceAccountID,
		targetAssets:           sourceAssetsMap,
		validateSourceBalance:  validateSourceBalance,
		paths:                  []Path{},
		includePools:           includePools,
	}
	graph.lock.RLock()
	err := search(
		ctx,
		searchState,
		maxPathLength,
		graph.assetStringToID[destinationAssetString],
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
	includePools bool,
) ([]Path, uint32, error) {
	target := map[int32]bool{}
	for _, destinationAsset := range destinationAssets {
		destinationAssetString := destinationAsset.String()
		target[graph.assetStringToID[destinationAssetString]] = true
	}

	searchState := &buyingGraphSearchState{
		graph:             graph,
		sourceAssetString: sourceAsset.String(),
		sourceAssetAmount: amountToSpend,
		targetAssets:      target,
		paths:             []Path{},
		includePools:      includePools,
	}
	graph.lock.RLock()
	err := search(
		ctx,
		searchState,
		maxPathLength,
		graph.assetStringToID[sourceAsset.String()],
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
	if allPaths[i].SourceAsset == allPaths[j].SourceAsset {
		if allPaths[i].SourceAmount == allPaths[j].SourceAmount {
			return len(allPaths[i].InteriorNodes) < len(allPaths[j].InteriorNodes)
		}
		return allPaths[i].SourceAmount < allPaths[j].SourceAmount
	}
	return allPaths[i].SourceAsset < allPaths[j].SourceAsset
}

// compareDestinationAsset will group payment paths by `DestinationAsset`. Paths
// which deliver a higher `DestinationAmount` will appear earlier in the
// sorting. If there are multiple paths which deliver the same
// `DestinationAmount`, then shorter payment paths will be prioritized.
func compareDestinationAsset(allPaths []Path, i, j int) bool {
	if allPaths[i].DestinationAsset == allPaths[j].DestinationAsset {
		if allPaths[i].DestinationAmount == allPaths[j].DestinationAmount {
			return len(allPaths[i].InteriorNodes) < len(allPaths[j].InteriorNodes)
		}
		return allPaths[i].DestinationAmount > allPaths[j].DestinationAmount
	}
	return allPaths[i].DestinationAsset < allPaths[j].DestinationAsset
}

func sourceAssetEquals(p, otherPath Path) bool {
	return p.SourceAsset == otherPath.SourceAsset
}

func destinationAssetEquals(p, otherPath Path) bool {
	return p.DestinationAsset == otherPath.DestinationAsset
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
