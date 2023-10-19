package history

import (
	"context"
	"github.com/stellar/go/support/db"

	"github.com/stretchr/testify/mock"
)

type MockClaimableBalanceClaimantBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockClaimableBalanceClaimantBatchInsertBuilder) Add(claimableBalanceClaimant ClaimableBalanceClaimant) error {
	a := m.Called(claimableBalanceClaimant)
	return a.Error(0)
}

func (m *MockClaimableBalanceClaimantBatchInsertBuilder) Exec(ctx context.Context, session db.SessionInterface) error {
	a := m.Called(ctx, session)
	return a.Error(0)
}

func (m *MockClaimableBalanceClaimantBatchInsertBuilder) Reset() error {
	a := m.Called()
	return a.Error(0)
}
