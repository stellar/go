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
	ingestVersion int,
) (int64, error) {
	a := m.Called(ledger, successTxsCount, failedTxsCount, opCount, ingestVersion)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQLedgers) CheckExpLedger(seq int32) (bool, error) {
	a := m.Called(seq)
	return a.Get(0).(bool), a.Error(1)
}
