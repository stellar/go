package index

import types "github.com/stellar/go/exp/lighthorizon/index/types"

// TODO: Use a more standardized filesystem-style backend, so we can re-use
// code
type Backend interface {
	Flush(map[string]types.NamedIndices) error
	FlushAccounts([]string) error
	Read(account string) (types.NamedIndices, error)
	ReadAccounts() ([]string, error)
	FlushTransactions(map[string]*types.TrieIndex) error
	ReadTransactions(prefix string) (*types.TrieIndex, error)
}
