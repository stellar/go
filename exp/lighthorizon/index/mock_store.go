package index

import (
	types "github.com/stellar/go/exp/lighthorizon/index/types"
	"github.com/stretchr/testify/mock"
)

type MockStore struct {
	mock.Mock
}

func (m *MockStore) NextActive(account, index string, afterCheckpoint uint32) (uint32, error) {
	args := m.Called(account, index, afterCheckpoint)
	return args.Get(0).(uint32), args.Error(1)
}

func (m *MockStore) TransactionTOID(hash [32]byte) (int64, error) {
	args := m.Called(hash)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockStore) AddTransactionToIndexes(txnTOID int64, hash [32]byte) error {
	args := m.Called(txnTOID, hash)
	return args.Error(0)
}

func (m *MockStore) AddParticipantsToIndexes(checkpoint uint32, index string, participants []string) error {
	args := m.Called(checkpoint, index, participants)
	return args.Error(0)
}

func (m *MockStore) AddParticipantsToIndexesNoBackend(checkpoint uint32, index string, participants []string) error {
	args := m.Called(checkpoint, index, participants)
	return args.Error(0)
}

func (m *MockStore) AddParticipantToIndexesNoBackend(participant string, indexes types.NamedIndices) {
	m.Called(participant, indexes)
}

func (m *MockStore) Flush() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockStore) FlushAccounts() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockStore) ClearMemory(arg bool) {
	m.Called(arg)
}

func (m *MockStore) Read(account string) (types.NamedIndices, error) {
	args := m.Called(account)
	return args.Get(0).(types.NamedIndices), args.Error(1)
}

func (m *MockStore) ReadAccounts() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockStore) ReadTransactions(prefix string) (*types.TrieIndex, error) {
	args := m.Called(prefix)
	return args.Get(0).(*types.TrieIndex), args.Error(1)
}

func (m *MockStore) MergeTransactions(prefix string, other *types.TrieIndex) error {
	args := m.Called(prefix, other)
	return args.Error(0)
}
