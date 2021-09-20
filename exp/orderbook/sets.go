package orderbook

import "github.com/stellar/go/xdr"

// IdSet provides a set-like container of liquidity pool IDs.
type IdSet struct {
	m map[xdr.PoolId]struct{}
}

func NewIdSet(sizeHint int) *IdSet {
	return &IdSet{m: make(map[xdr.PoolId]struct{}, sizeHint)}
}

func (s *IdSet) Contains(elem xdr.PoolId) bool {
	_, ok := s.m[elem]
	return ok
}

func (s *IdSet) Add(elem xdr.PoolId) {
	s.m[elem] = struct{}{}
}

func (s *IdSet) Clone() *IdSet {
	newSet := NewIdSet(len(s.m))
	for key, value := range s.m {
		newSet.m[key] = value
	}
	return newSet
}
