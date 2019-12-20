package history

import (
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stretchr/testify/mock"
)

// MockOperationsBatchInsertBuilder OperationsBatchInsertBuilder mock
type MockOperationsBatchInsertBuilder struct {
	mock.Mock
}

// Add mock
func (m *MockOperationsBatchInsertBuilder) Add(transaction io.LedgerTransaction, sequence uint32) error {
	a := m.Called(transaction, sequence)
	return a.Error(0)
}

// Exec mock
func (m *MockOperationsBatchInsertBuilder) Exec() error {
	a := m.Called()
	return a.Error(0)
}
