package history

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/stellar/go/xdr"
)

// MockQClaimableBalances is a mock implementation of the QAccounts interface
type MockQClaimableBalances struct {
	mock.Mock
}

func (m *MockQClaimableBalances) NewClaimableBalancesBatchInsertBuilder(maxBatchSize int) ClaimableBalancesBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(ClaimableBalancesBatchInsertBuilder)
}

func (m *MockQClaimableBalances) CountClaimableBalances(ctx context.Context) (int, error) {
	a := m.Called(ctx)
	return a.Get(0).(int), a.Error(1)
}

func (m *MockQClaimableBalances) GetClaimableBalancesByID(ctx context.Context, ids []xdr.ClaimableBalanceId) ([]ClaimableBalance, error) {
	a := m.Called(ctx, ids)
	return a.Get(0).([]ClaimableBalance), a.Error(1)
}

func (m *MockQClaimableBalances) UpdateClaimableBalance(ctx context.Context, entry xdr.LedgerEntry) (int64, error) {
	a := m.Called(ctx, entry)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQClaimableBalances) RemoveClaimableBalance(ctx context.Context, cBalance xdr.ClaimableBalanceEntry) (int64, error) {
	a := m.Called(ctx, cBalance)
	return a.Get(0).(int64), a.Error(1)
}

type MockClaimableBalancesBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockClaimableBalancesBatchInsertBuilder) Add(ctx context.Context, entry *xdr.LedgerEntry) error {
	a := m.Called(ctx, entry)
	return a.Error(0)
}

func (m *MockClaimableBalancesBatchInsertBuilder) Exec(ctx context.Context) error {
	a := m.Called(ctx)
	return a.Error(0)
}
