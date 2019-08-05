package simplepath

import (
	"github.com/go-errors/errors"
	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/services/horizon/internal/paths"
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
// using the experimental in memory orderbook
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
func (finder InMemoryFinder) Find(q paths.Query, maxLength uint) ([]paths.Path, error) {
	if finder.graph.IsEmpty() {
		return nil, ErrEmptyInMemoryOrderBook
	}

	if maxLength == 0 {
		maxLength = MaxInMemoryPathLength
	}
	if maxLength > MaxInMemoryPathLength {
		return nil, errors.New("invalid value of maxLength")
	}

	orderbookPaths, err := finder.graph.FindPaths(
		int(maxLength),
		q.DestinationAsset,
		q.DestinationAmount,
		q.SourceAccount,
		q.SourceAssets,
		q.SourceAssetBalances,
		maxAssetsPerPath,
	)
	results := make([]paths.Path, len(orderbookPaths))
	for i, path := range orderbookPaths {
		results[i] = paths.Path{
			Path:        path.InteriorNodes,
			Source:      path.SourceAsset,
			Destination: path.DestinationAsset,
			Cost:        path.SourceAmount,
		}
	}
	return results, err
}
