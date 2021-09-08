package history

import (
	"context"

	"github.com/stretchr/testify/mock"
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

func (m *MockQLiquidityPools) GetLiquidityPoolsByID(ctx context.Context, poolIDs []string) ([]LiquidityPool, error) {
	a := m.Called(ctx, poolIDs)
	return a.Get(0).([]LiquidityPool), a.Error(1)
}

func (m *MockQLiquidityPools) UpdateLiquidityPool(ctx context.Context, lp LiquidityPool) (int64, error) {
	a := m.Called(ctx, lp)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQLiquidityPools) RemoveLiquidityPool(ctx context.Context, liquidityPoolID string, sequence uint32) (int64, error) {
	a := m.Called(ctx, liquidityPoolID, sequence)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQLiquidityPools) FindLiquidityPoolByID(ctx context.Context, liquidityPoolID string) (LiquidityPool, error) {
	a := m.Called(ctx, liquidityPoolID)
	return a.Get(0).(LiquidityPool), a.Error(1)
}

func (m *MockQLiquidityPools) GetUpdatedLiquidityPools(ctx context.Context, sequence uint32) ([]LiquidityPool, error) {
	a := m.Called(ctx, sequence)
	return a.Get(0).([]LiquidityPool), a.Error(1)
}

func (m *MockQLiquidityPools) CompactLiquidityPools(ctx context.Context, cutOffSequence uint32) (int64, error) {
	a := m.Called(ctx, cutOffSequence)
	return a.Get(0).(int64), a.Error(1)
}

type MockLiquidityPoolsBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockLiquidityPoolsBatchInsertBuilder) Add(ctx context.Context, lp LiquidityPool) error {
	a := m.Called(ctx, lp)
	return a.Error(0)
}

func (m *MockLiquidityPoolsBatchInsertBuilder) Exec(ctx context.Context) error {
	a := m.Called(ctx)
	return a.Error(0)
}
