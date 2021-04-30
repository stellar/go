package history

import (
	"context"

	"github.com/stellar/go/xdr"

	"github.com/stretchr/testify/mock"
)

type MockQTrades struct {
	mock.Mock
}

func (m *MockQTrades) CreateAccounts(ctx context.Context, addresses []string, maxBatchSize int) (map[string]int64, error) {
	a := m.Called(ctx, addresses, maxBatchSize)
	return a.Get(0).(map[string]int64), a.Error(1)
}

func (m *MockQTrades) CreateAssets(ctx context.Context, assets []xdr.Asset, maxBatchSize int) (map[string]Asset, error) {
	a := m.Called(ctx, assets, maxBatchSize)
	return a.Get(0).(map[string]Asset), a.Error(1)
}

func (m *MockQTrades) NewTradeBatchInsertBuilder(maxBatchSize int) TradeBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(TradeBatchInsertBuilder)
}

type MockTradeBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockTradeBatchInsertBuilder) Add(ctx context.Context, entries ...InsertTrade) error {
	a := m.Called(ctx, entries)
	return a.Error(0)
}

func (m *MockTradeBatchInsertBuilder) Exec(ctx context.Context) error {
	a := m.Called(ctx)
	return a.Error(0)
}
