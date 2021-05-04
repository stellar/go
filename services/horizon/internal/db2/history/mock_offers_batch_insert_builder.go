package history

import (
	"context"
	"github.com/stretchr/testify/mock"
)

type MockOffersBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockOffersBatchInsertBuilder) Add(ctx context.Context, row Offer) error {
	a := m.Called(ctx, row)
	return a.Error(0)
}

func (m *MockOffersBatchInsertBuilder) Exec(ctx context.Context) error {
	a := m.Called(ctx)
	return a.Error(0)
}
