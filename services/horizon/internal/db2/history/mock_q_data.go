package history

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/stellar/go/xdr"
)

// MockQData is a mock implementation of the QAccounts interface
type MockQData struct {
	mock.Mock
}

func (m *MockQData) GetAccountDataByKeys(ctx context.Context, keys []xdr.LedgerKeyData) ([]Data, error) {
	a := m.Called(ctx)
	return a.Get(0).([]Data), a.Error(1)
}

func (m *MockQData) CountAccountsData(ctx context.Context) (int, error) {
	a := m.Called(ctx)
	return a.Get(0).(int), a.Error(1)
}

func (m *MockQData) NewAccountDataBatchInsertBuilder(maxBatchSize int) AccountDataBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(AccountDataBatchInsertBuilder)
}

func (m *MockQData) InsertAccountData(ctx context.Context, entry xdr.LedgerEntry) (int64, error) {
	a := m.Called(ctx, entry)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQData) UpdateAccountData(ctx context.Context, entry xdr.LedgerEntry) (int64, error) {
	a := m.Called(ctx, entry)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQData) RemoveAccountData(ctx context.Context, key xdr.LedgerKeyData) (int64, error) {
	a := m.Called(ctx, key)
	return a.Get(0).(int64), a.Error(1)
}
