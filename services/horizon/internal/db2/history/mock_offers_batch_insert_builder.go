package history

import (
	"github.com/stretchr/testify/mock"
)

type MockOffersBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockOffersBatchInsertBuilder) Add(row Offer) error {
	a := m.Called(row)
	return a.Error(0)
}

func (m *MockOffersBatchInsertBuilder) Exec() error {
	a := m.Called()
	return a.Error(0)
}
