package ledgerbackend

import "github.com/stretchr/testify/mock"

var _ LedgerStore = (*MockLedgerStore)(nil)

type MockLedgerStore struct {
	mock.Mock
}

func (m *MockLedgerStore) LastLedger(seq uint32) (Ledger, bool, error) {
	args := m.Called(seq)
	return args.Get(0).(Ledger), args.Get(1).(bool), args.Error(2)
}
