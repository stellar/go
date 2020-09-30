package history

import (
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

type MockOffersBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockOffersBatchInsertBuilder) Add(entry xdr.LedgerEntry) error {
	a := m.Called(entry)
	return a.Error(0)
}

func (m *MockOffersBatchInsertBuilder) Exec() error {
	a := m.Called()
	return a.Error(0)
}
