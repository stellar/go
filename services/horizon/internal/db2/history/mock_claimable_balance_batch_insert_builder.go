package history

import (
	"context"

	"github.com/stellar/go/support/db"
	"github.com/stretchr/testify/mock"
)

type MockClaimableBalanceBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockClaimableBalanceBatchInsertBuilder) Add(claimableBalance ClaimableBalance) error {
	a := m.Called(claimableBalance)
	return a.Error(0)
}

func (m *MockClaimableBalanceBatchInsertBuilder) Exec(ctx context.Context, session db.SessionInterface) error {
	a := m.Called(ctx, session)
	return a.Error(0)
}

func (m *MockClaimableBalanceBatchInsertBuilder) Reset() {
	m.Called()
}
