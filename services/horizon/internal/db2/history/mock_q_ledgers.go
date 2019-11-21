package history

import (
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

type MockQLedgers struct {
	mock.Mock
}

func (m *MockQLedgers) InsertExpLedger(
	ledger xdr.LedgerHeaderHistoryEntry,
	successTxsCount int,
	failedTxsCount int,
	opCount int,
) (int64, error) {
	a := m.Called(ledger, successTxsCount, failedTxsCount, opCount)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQLedgers) CheckExpLeger(seq int32) (bool, error) {
	a := m.Called(seq)
	return a.Get(0).(bool), a.Error(1)
}
