package history

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockLiquidityPoolBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockLiquidityPoolBatchInsertBuilder) Add(liquidityPool LiquidityPool) error {
	a := m.Called(liquidityPool)
	return a.Error(0)
}

func (m *MockLiquidityPoolBatchInsertBuilder) Exec(ctx context.Context) error {
	a := m.Called(ctx)
	return a.Error(0)
}
