package history

import (
	"github.com/stretchr/testify/mock"
)

// MockOperationsBatchInsertBuilder OperationsBatchInsertBuilder mock
type MockOperationsBatchInsertBuilder struct {
	mock.Mock
}

// Add mock
func (m *MockOperationsBatchInsertBuilder) Add(operation TransactionOperation) error {
	a := m.Called(operation)
	return a.Error(0)
}

// Exec mock
func (m *MockOperationsBatchInsertBuilder) Exec() error {
	a := m.Called()
	return a.Error(0)
}
