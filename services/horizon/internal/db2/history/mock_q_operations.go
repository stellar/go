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

// NewOperationParticipantBatchInsertBuilder mock
func (m *MockQOperations) NewOperationParticipantBatchInsertBuilder(maxBatchSize int) OperationParticipantBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(OperationParticipantBatchInsertBuilder)
}

// CheckExpOperations mock
func (m *MockQOperations) CheckExpOperations(seq int32) (bool, error) {
	a := m.Called(seq)
	return a.Get(0).(bool), a.Error(1)
}

// CheckExpOperationParticipants mock
func (m *MockQOperations) CheckExpOperationParticipants(seq int32) (bool, error) {
	a := m.Called(seq)
	return a.Get(0).(bool), a.Error(1)
}
