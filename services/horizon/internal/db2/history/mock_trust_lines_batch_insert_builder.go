package history

import (
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

type MockTrustLinesBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockTrustLinesBatchInsertBuilder) Add(entry xdr.LedgerEntry) error {
	a := m.Called(entry)
	return a.Error(0)
}

func (m *MockTrustLinesBatchInsertBuilder) Exec() error {
	a := m.Called()
	return a.Error(0)
}
