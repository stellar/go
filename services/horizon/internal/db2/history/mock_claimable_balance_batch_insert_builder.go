package history

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockClaimableBalanceBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockClaimableBalanceBatchInsertBuilder) Add(claimableBalance ClaimableBalance) error {
	a := m.Called(claimableBalance)
	return a.Error(0)
}

func (m *MockClaimableBalanceBatchInsertBuilder) Exec(ctx context.Context) error {
	a := m.Called(ctx)
	return a.Error(0)
}
