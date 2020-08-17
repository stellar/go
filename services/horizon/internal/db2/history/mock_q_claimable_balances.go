package history

import (
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

// MockQClaimableBalances is a mock implementation of the QAccounts interface
type MockQClaimableBalances struct {
	mock.Mock
}

func (m *MockQClaimableBalances) NewClaimableBalancesBatchInsertBuilder(maxBatchSize int) ClaimableBalancesBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(ClaimableBalancesBatchInsertBuilder)
}

func (m *MockQClaimableBalances) CountClaimableBalances() (int, error) {
	a := m.Called()
	return a.Get(0).(int), a.Error(1)
}

func (m *MockQClaimableBalances) GetClaimableBalancesByID(ids []xdr.ClaimableBalanceId) ([]ClaimableBalance, error) {
	a := m.Called(ids)
	return a.Get(0).([]ClaimableBalance), a.Error(1)
}

func (m *MockQClaimableBalances) UpdateClaimableBalance(entry xdr.LedgerEntry) (int64, error) {
	a := m.Called(entry)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQClaimableBalances) RemoveClaimableBalance(cBalance xdr.ClaimableBalanceEntry) (int64, error) {
	a := m.Called(cBalance)
	return a.Get(0).(int64), a.Error(1)
}

type MockClaimableBalancesBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockClaimableBalancesBatchInsertBuilder) Add(entry *xdr.LedgerEntry) error {
	a := m.Called(entry)
	return a.Error(0)
}

func (m *MockClaimableBalancesBatchInsertBuilder) Exec() error {
	a := m.Called()
	return a.Error(0)
}
