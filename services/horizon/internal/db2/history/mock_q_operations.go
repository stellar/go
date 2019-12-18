package history

import "github.com/stretchr/testify/mock"

// MockQOperations is a mock implementation of the QOperations interface
type MockQOperations struct {
	mock.Mock
}

// NewOperationBatchInsertBuilder mock
func (m *MockQOperations) NewOperationBatchInsertBuilder(maxBatchSize int) OperationBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(OperationBatchInsertBuilder)
}

// CheckExpOperations mock
func (m *MockQOperations) CheckExpOperations(seq int32) (bool, error) {
	a := m.Called(seq)
	return a.Get(0).(bool), a.Error(1)
}
