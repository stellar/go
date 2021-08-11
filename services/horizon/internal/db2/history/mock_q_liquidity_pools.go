package history

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/stellar/go/xdr"
)

// MockQLiquidityPools is a mock implementation of the QAccounts interface
type MockQLiquidityPools struct {
	mock.Mock
}

func (m *MockQLiquidityPools) NewLiquidityPoolsBatchInsertBuilder(maxBatchSize int) LiquidityPoolsBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(LiquidityPoolsBatchInsertBuilder)
}

func (m *MockQLiquidityPools) CountLiquidityPools(ctx context.Context) (int, error) {
	a := m.Called(ctx)
	return a.Get(0).(int), a.Error(1)
}

func (m *MockQLiquidityPools) GetLiquidityPoolsByID(ctx context.Context, ids []xdr.PoolId) ([]LiquidityPool, error) {
	a := m.Called(ctx, ids)
	return a.Get(0).([]LiquidityPool), a.Error(1)
}

func (m *MockQLiquidityPools) UpdateLiquidityPool(ctx context.Context, entry xdr.LedgerEntry) (int64, error) {
	a := m.Called(ctx, entry)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQLiquidityPools) RemoveLiquidityPool(ctx context.Context, cBalance xdr.LiquidityPoolEntry) (int64, error) {
	a := m.Called(ctx, cBalance)
	return a.Get(0).(int64), a.Error(1)
}

type MockLiquidityPoolsBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockLiquidityPoolsBatchInsertBuilder) Add(ctx context.Context, entry *xdr.LedgerEntry) error {
	a := m.Called(ctx, entry)
	return a.Error(0)
}

func (m *MockLiquidityPoolsBatchInsertBuilder) Exec(ctx context.Context) error {
	a := m.Called(ctx)
	return a.Error(0)
}
