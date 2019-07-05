package orderbook

import (
	"sort"
	"sync"

	"github.com/stellar/go/price"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

var (
	errOfferNotPresent             = errors.New("offer is not present in the order book graph")
	errEmptyOffers                 = errors.New("offers is empty")
	errAssetAmountIsZero           = errors.New("current asset amount is 0")
	errOfferPriceDenominatorIsZero = errors.New("denominator of offer price is 0")
	errBatchAlreadyApplied         = errors.New("cannot apply batched updates more than once")
)

// trading pair represents two assets that can be exchanged if an order is fulfilled
type tradingPair struct {
	// buyingAsset is obtained by calling offer.Buying.String() where offer is an xdr.OfferEntry
	buyingAsset string
	// sellingAsset is obtained by calling offer.Selling.String() where offer is an xdr.OfferEntry
	sellingAsset string
}

// OrderBookGraph is an in memory graph representation of all the offers in the stellar ledger
type OrderBookGraph struct {
	// edgesForSellingAsset maps an asset to all offers which sell that asset
	// note that each key in the map is obtained by calling offer.Selling.String()
	// where offer is an xdr.OfferEntry
	edgesForSellingAsset map[string]edgeSet
	// tradingPairForOffer maps an offer id to the assets which are being exchanged
	// in the given offer
	tradingPairForOffer map[xdr.Int64]tradingPair
	// batchedUpdates is internal batch of updates to this graph. Users can
	// create multiple batches using `Batch()` method but sometimes only one
	// batch is enough.
	batchedUpdates *orderBookBatchedUpdates
	lock           sync.RWMutex
}

// NewOrderBookGraph constructs a new OrderBookGraph
func NewOrderBookGraph() *OrderBookGraph {
	graph := &OrderBookGraph{
		edgesForSellingAsset: map[string]edgeSet{},
		tradingPairForOffer:  map[xdr.Int64]tradingPair{},
	}

	graph.batchedUpdates = graph.batch()
	return graph
}

// AddOffer will queue an operation to add the given offer to the order book in
// the internal batch.
// You need to run Apply() to apply all enqueued operations.
func (graph *OrderBookGraph) AddOffer(offer xdr.OfferEntry) *OrderBookGraph {
	graph.batchedUpdates.addOffer(offer)
	return graph
}

// RemoveOffer will queue an operation to remove the given offer from the order book in
// the internal batch.
// You need to run Apply() to apply all enqueued operations.
func (graph *OrderBookGraph) RemoveOffer(offerID xdr.Int64) *OrderBookGraph {
	graph.batchedUpdates.removeOffer(offerID)
	return graph
}

// Apply will attempt to apply all the updates in the internal batch to the order book.
// When Apply is successful, a new empty, instance of internal batch will be created.
func (graph *OrderBookGraph) Apply() error {
	err := graph.batchedUpdates.apply()
	if err != nil {
		return err
	}
	graph.batchedUpdates = graph.batch()
	return nil
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
	graph.tradingPairForOffer[offer.OfferId] = tradingPair{
		buyingAsset:  offer.Buying.String(),
		sellingAsset: sellingAsset,
	}
	if set, ok := graph.edgesForSellingAsset[sellingAsset]; !ok {
		graph.edgesForSellingAsset[sellingAsset] = edgeSet{}
		graph.edgesForSellingAsset[sellingAsset].add(offer)
	} else {
		set.add(offer)
	}

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

	return nil
}

// Path represents a payment path from a source asset to some destination asset
type Path struct {
	SourceAmount xdr.Int64
	SourceAsset  xdr.Asset
	// sourceAssetString is included as an optimization to improve the performance
	// of sorting paths by avoiding serializing assets to strings repeatedly
	sourceAssetString string
	InteriorNodes     []xdr.Asset
	DestinationAsset  xdr.Asset
	DestinationAmount xdr.Int64
}

// findPaths performs a DFS of maxPathLength depth
// the DFS maintains the following invariants:
// no node is repeated
// no offers are consumed from the `ignoreOffersFrom` account
// each payment path must originate with an asset in `targetAssets`
// also, the required source asset amount cannot exceed the balance in `targetAssets`
func (graph *OrderBookGraph) findPaths(
	maxPathLength int,
	visited map[string]bool,
	visitedList []xdr.Asset,
	currentAssetString string,
	currentAsset xdr.Asset,
	currentAssetAmount xdr.Int64,
	destinationAsset xdr.Asset,
	destinationAssetAmount xdr.Int64,
	ignoreOffersFrom xdr.AccountId,
	targetAssets map[string]xdr.Int64,
	paths []Path,
) ([]Path, error) {
	if currentAssetAmount <= 0 {
		return paths, nil
	}
	if visited[currentAssetString] {
		return paths, nil
	}
	if len(visitedList) > maxPathLength {
		return paths, nil
	}
	visited[currentAssetString] = true
	defer func() {
		visited[currentAssetString] = false
	}()

	updatedVisitedList := append(visitedList, currentAsset)
	if targetAssetBalance, ok := targetAssets[currentAssetString]; ok && targetAssetBalance >= currentAssetAmount {
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

		paths = append(paths, Path{
			sourceAssetString: currentAssetString,
			SourceAmount:      currentAssetAmount,
			SourceAsset:       currentAsset,
			InteriorNodes:     interiorNodes,
			DestinationAsset:  destinationAsset,
			DestinationAmount: destinationAssetAmount,
		})
	}

	edges, ok := graph.edgesForSellingAsset[currentAssetString]
	if !ok {
		return paths, nil
	}

	for nextAssetString, offers := range edges {
		if len(offers) == 0 {
			continue
		}
		nextAssetAmount, err := consumeOffers(offers, ignoreOffersFrom, currentAssetAmount)
		if err != nil {
			return nil, err
		}
		if nextAssetAmount <= 0 {
			continue
		}

		nextAsset := offers[0].Buying
		paths, err = graph.findPaths(
			maxPathLength,
			visited,
			updatedVisitedList,
			nextAssetString,
			nextAsset,
			nextAssetAmount,
			destinationAsset,
			destinationAssetAmount,
			ignoreOffersFrom,
			targetAssets,
			paths,
		)
		if err != nil {
			return nil, err
		}
	}

	return paths, nil
}

// FindPaths returns a list of payment paths originating from a source account
// and ending with a given destinaton asset and amount.
func (graph *OrderBookGraph) FindPaths(
	maxPathLength int,
	destinationAsset xdr.Asset,
	destinationAmount xdr.Int64,
	sourceAccountID xdr.AccountId,
	sourceAssets []xdr.Asset,
	sourceAssetBalances []xdr.Int64,
	maxAssetsPerPath int,
) ([]Path, error) {
	destinationAssetString := destinationAsset.String()
	sourceAssetsMap := map[string]xdr.Int64{}
	for i, sourceAsset := range sourceAssets {
		sourceAssetString := sourceAsset.String()
		sourceAssetsMap[sourceAssetString] = sourceAssetBalances[i]
	}

	graph.lock.RLock()
	allPaths, err := graph.findPaths(
		maxPathLength,
		map[string]bool{},
		[]xdr.Asset{},
		destinationAssetString,
		destinationAsset,
		destinationAmount,
		destinationAsset,
		destinationAmount,
		sourceAccountID,
		sourceAssetsMap,
		[]Path{},
	)
	graph.lock.RUnlock()
	if err != nil {
		return nil, errors.Wrap(err, "could not determine paths")
	}

	return sortAndFilterPaths(
		allPaths,
		maxAssetsPerPath,
	), nil
}

func consumeOffers(
	offers []xdr.OfferEntry,
	ignoreOffersFrom xdr.AccountId,
	currentAssetAmount xdr.Int64,
) (xdr.Int64, error) {
	totalConsumed := xdr.Int64(0)

	if len(offers) == 0 {
		return totalConsumed, errEmptyOffers
	}

	if currentAssetAmount == 0 {
		return totalConsumed, errAssetAmountIsZero
	}

	for _, offer := range offers {
		if offer.SellerId.Equals(ignoreOffersFrom) {
			continue
		}
		if offer.Price.D == 0 {
			return -1, errOfferPriceDenominatorIsZero
		}

		buyingUnitsFromOffer, sellingUnitsFromOffer, err := price.ConvertToBuyingUnits(
			int64(offer.Amount),
			int64(currentAssetAmount),
			int64(offer.Price.N),
			int64(offer.Price.D),
		)
		if err != nil {
			return -1, errors.Wrap(err, "could not determine buying units")
		}

		totalConsumed += xdr.Int64(buyingUnitsFromOffer)
		currentAssetAmount -= xdr.Int64(sellingUnitsFromOffer)
	}

	if currentAssetAmount <= 0 {
		return totalConsumed, nil
	}
	return -1, nil
}

// sortAndFilterPaths sorts the given list of paths by
// source asset, source asset amount, and path length
// also, we limit the number of paths with the same source asset to maxPathsPerAsset
func sortAndFilterPaths(
	allPaths []Path,
	maxPathsPerAsset int,
) []Path {
	sort.Slice(allPaths, func(i, j int) bool {
		if allPaths[i].SourceAsset.Equals(allPaths[j].SourceAsset) {
			if allPaths[i].SourceAmount == allPaths[j].SourceAmount {
				return len(allPaths[i].InteriorNodes) < len(allPaths[j].InteriorNodes)
			}
			return allPaths[i].SourceAmount < allPaths[j].SourceAmount
		}
		return allPaths[i].sourceAssetString < allPaths[j].sourceAssetString
	})

	filtered := []Path{}
	countForAsset := 0
	for _, entry := range allPaths {
		if len(filtered) == 0 || !filtered[len(filtered)-1].SourceAsset.Equals(entry.SourceAsset) {
			countForAsset = 1
			filtered = append(filtered, entry)
		} else if countForAsset < maxPathsPerAsset {
			countForAsset++
			filtered = append(filtered, entry)
		}
	}

	return filtered
}
