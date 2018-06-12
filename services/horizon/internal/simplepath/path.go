package simplepath

import (
	"bytes"
	"fmt"

	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/paths"
	"github.com/stellar/go/xdr"
)

// pathNode implements the paths.Path interface and represents a path
// as a linked list pointing from source to destination.
type pathNode struct {
	Asset     xdr.Asset
	Tail      *pathNode
	Q         *core.Q
	costCache map[xdr.Int64]xdr.Int64
}

// check interface compatibility
var _ paths.Path = &pathNode{}

func (p *pathNode) String() string {
	if p == nil {
		return ""
	}

	var out bytes.Buffer
	fmt.Fprintf(&out, "%v", p.Asset)

	cur := p.Tail

	for cur != nil {
		fmt.Fprintf(&out, " -> %v", cur.Asset)
		cur = cur.Tail
	}

	return out.String()
}

// Destination implements paths.Path.Destination interface method
func (p *pathNode) Destination() xdr.Asset {
	cur := p
	for cur.Tail != nil {
		cur = cur.Tail
	}
	return cur.Asset
}

// Source implements paths.Path.Source interface method
func (p *pathNode) Source() xdr.Asset {
	// the destination for path is the head of the linked list
	return p.Asset
}

// Path implements paths.Path.Path interface method
func (p *pathNode) Path() []xdr.Asset {
	path := p.Flatten()

	if len(path) < 2 {
		return nil
	}

	// return the flattened slice without the first and last elements
	// which are the source and the destination assets
	return path[1 : len(path)-1]
}

func (p *pathNode) init() {
	p.costCache = make(map[xdr.Int64]xdr.Int64)
}

// Cost implements the paths.Path.Cost interface method
func (p *pathNode) Cost(amount xdr.Int64) (xdr.Int64, error) {
	if p.costCache == nil {
		p.init()
	}

	result := amount
	var err error
	cur := p
	for cur.Tail != nil {
		ob := cur.OrderBook()
		result, err = ob.CostToConsumeLiquidity(result)
		if err != nil {
			return result, err
		}
		cur = cur.Tail
	}

	p.costCache[amount] = result
	return result, nil
}

// CachedCost impl, returns a cached version for the provided amount or nil if not cached
func (p *pathNode) CachedCost(amount xdr.Int64) *xdr.Int64 {
	if p.costCache == nil {
		return nil
	}

	if cached, ok := p.costCache[amount]; ok {
		return &cached
	}
	return nil
}

// Depth returns the length of the list
func (p *pathNode) Depth() int {
	depth := 0
	cur := p
	for cur != nil {
		cur = cur.Tail
		depth++
	}
	return depth
}

// Flatten walks the list and returns a slice of assets
func (p *pathNode) Flatten() []xdr.Asset {
	result := []xdr.Asset{}
	cur := p
	for cur != nil {
		result = append(result, cur.Asset)
		cur = cur.Tail
	}
	return result
}

func (p *pathNode) OrderBook() *orderBook {
	if p.Tail == nil {
		return nil
	}

	return &orderBook{
		Selling: p.Tail.Asset, // offer is selling this asset
		Buying:  p.Asset,      // offer is buying this asset
		Q:       p.Q,
	}
}
