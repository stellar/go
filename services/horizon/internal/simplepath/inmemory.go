package simplepath

import (
	"github.com/go-errors/errors"
	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/services/horizon/internal/paths"
	"github.com/stellar/go/xdr"
)

const (
	maxAssetsPerPath = 5
	// MaxInMemoryPathLength is the maximum path length which can be queried by the InMemoryFinder
	MaxInMemoryPathLength = 5
)

var (
	// ErrEmptyInMemoryOrderBook indicates that the in memory order book is not yet populated
	ErrEmptyInMemoryOrderBook = errors.New("Empty orderbook")
)

// InMemoryFinder is an implementation of the path finding interface
// using the in memory orderbook
type InMemoryFinder struct {
	graph *orderbook.OrderBookGraph
}

// NewInMemoryFinder constructs a new InMemoryFinder instance
func NewInMemoryFinder(graph *orderbook.OrderBookGraph) InMemoryFinder {
	return InMemoryFinder{
		graph: graph,
	}
}

// Find implements the path payments finder interface
func (finder InMemoryFinder) Find(q paths.Query, maxLength uint) ([]paths.Path, uint32, error) {
	if finder.graph.IsEmpty() {
		return nil, 0, ErrEmptyInMemoryOrderBook
	}

	if maxLength == 0 {
		maxLength = MaxInMemoryPathLength
	}
	if maxLength > MaxInMemoryPathLength {
		return nil, 0, errors.New("invalid value of maxLength")
	}

	orderbookPaths, lastLedger, err := finder.graph.FindPaths(
		int(maxLength),
		q.DestinationAsset,
		q.DestinationAmount,
		q.SourceAccount,
		q.SourceAssets,
		q.SourceAssetBalances,
		q.ValidateSourceBalance,
		maxAssetsPerPath,
	)
	results := make([]paths.Path, len(orderbookPaths))
	for i, path := range orderbookPaths {
		results[i] = paths.Path{
			Path:              path.InteriorNodes,
			Source:            path.SourceAsset,
			SourceAmount:      path.SourceAmount,
			Destination:       path.DestinationAsset,
			DestinationAmount: path.DestinationAmount,
		}
	}
	return results, lastLedger, err
}

// FindFixedPaths returns a list of payment paths where the source and destination
// assets are fixed. All returned payment paths will start by spending `amountToSpend`
// of `sourceAsset` and will end with some positive balance of `destinationAsset`.
// `sourceAccountID` is optional. if `sourceAccountID` is provided then no offers
// created by `sourceAccountID` will be considered when evaluating payment paths
func (finder InMemoryFinder) FindFixedPaths(
	sourceAsset xdr.Asset,
	amountToSpend xdr.Int64,
	destinationAssets []xdr.Asset,
	maxLength uint,
) ([]paths.Path, uint32, error) {
	if finder.graph.IsEmpty() {
		return nil, 0, ErrEmptyInMemoryOrderBook
	}

	if maxLength == 0 {
		maxLength = MaxInMemoryPathLength
	}
	if maxLength > MaxInMemoryPathLength {
		return nil, 0, errors.New("invalid value of maxLength")
	}

	orderbookPaths, lastLedger, err := finder.graph.FindFixedPaths(
		int(maxLength),
		sourceAsset,
		amountToSpend,
		destinationAssets,
		maxAssetsPerPath,
	)
	results := make([]paths.Path, len(orderbookPaths))
	for i, path := range orderbookPaths {
		results[i] = paths.Path{
			Path:              path.InteriorNodes,
			Source:            path.SourceAsset,
			SourceAmount:      path.SourceAmount,
			Destination:       path.DestinationAsset,
			DestinationAmount: path.DestinationAmount,
		}
	}
	return results, lastLedger, err
}
