package history

import (
	"context"
	"github.com/stretchr/testify/mock"

	"github.com/stellar/go/xdr"
)

// MockQAccounts is a mock implementation of the QAccounts interface
type MockQAccounts struct {
	mock.Mock
}

func (m *MockQAccounts) GetAccountsByIDs(ctx context.Context, ids []string) ([]AccountEntry, error) {
	a := m.Called(ctx, ids)
	return a.Get(0).([]AccountEntry), a.Error(1)
}

func (m *MockQAccounts) NewAccountsBatchInsertBuilder(maxBatchSize int) AccountsBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(AccountsBatchInsertBuilder)
}

func (m *MockQAccounts) UpsertAccounts(ctx context.Context, accounts []xdr.LedgerEntry) error {
	a := m.Called(ctx, accounts)
	return a.Error(0)
}

func (m *MockQAccounts) RemoveAccount(ctx context.Context, accountID string) (int64, error) {
	a := m.Called(ctx, accountID)
	return a.Get(0).(int64), a.Error(1)
}
