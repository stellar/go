package history

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockQAccountFilterWhitelist is a mock implementation of the QAccountFilterWhitelist interface
type MockQFilter struct {
	mock.Mock
}

func (m *MockQFilter) GetAllFilters(ctx context.Context) ([]FilterConfig, error) {
	a := m.Called(ctx)
	return a.Get(0).([]FilterConfig), a.Error(1)
}

func (m *MockQFilter) GetFilterByName(ctx context.Context, name string) (FilterConfig, error) {
	a := m.Called(ctx, name)
	return a.Get(0).(FilterConfig), a.Error(1)
}

func (m *MockQFilter) SetFilterConfig(ctx context.Context, config FilterConfig) error {
	a := m.Called(ctx, config)
	return a.Error(0)
}
