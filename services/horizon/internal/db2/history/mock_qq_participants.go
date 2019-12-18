package history

import "github.com/stretchr/testify/mock"

// MockQQParticipants is a mock implementation of the QOperations interface
type MockQQParticipants struct {
	mock.Mock
}

// NewOperationParticipantBatchInsertBuilder mock
func (m *MockQQParticipants) NewOperationParticipantBatchInsertBuilder(maxBatchSize int) OperationParticipantBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(OperationParticipantBatchInsertBuilder)
}

// CheckExpOperationParticipants mock
func (m *MockQQParticipants) CheckExpOperationParticipants(seq int32) (bool, error) {
	a := m.Called(seq)
	return a.Get(0).(bool), a.Error(1)
}
