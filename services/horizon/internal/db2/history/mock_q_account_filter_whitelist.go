package history

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockQAccountFilterWhitelist is a mock implementation of the QAccountFilterWhitelist interface
type MockQAccountFilterWhitelist struct {
	mock.Mock
}

func (m *MockQAccountFilterWhitelist) GetAccountFilterWhitelist(ctx context.Context) ([]string, error) {
	a := m.Called(ctx)
	return a.Get(0).([]string), a.Error(1)
}

func (m *MockQAccountFilterWhitelist) UpsertAccountFilterWhitelist(ctx context.Context, accountIDs []string) error {
	a := m.Called(ctx, accountIDs)
	return a.Error(0)
}

func (m *MockQAccountFilterWhitelist) RemoveFromAccountFilterWhitelist(ctx context.Context, accountIDs []string) (int64, error) {
	a := m.Called(ctx, accountIDs)
	return a.Get(0).(int64), a.Error(1)
}
