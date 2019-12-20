package history

import (
	"github.com/stretchr/testify/mock"
)

// MockQParticipants is a mock implementation of the QParticipants interface
type MockQParticipants struct {
	mock.Mock
}

func (m *MockQParticipants) CheckExpParticipants(seq int32) (bool, error) {
	a := m.Called(seq)
	return a.Get(0).(bool), a.Error(1)
}

func (m *MockQParticipants) CreateExpAccounts(addresses []string) (map[string]int64, error) {
	a := m.Called(addresses)
	return a.Get(0).(map[string]int64), a.Error(1)
}

func (m *MockQParticipants) NewTransactionParticipantsBatchInsertBuilder(maxBatchSize int) TransactionParticipantsBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(TransactionParticipantsBatchInsertBuilder)
}

// NewOperationParticipantBatchInsertBuilder mock
func (m *MockQParticipants) NewOperationParticipantBatchInsertBuilder(maxBatchSize int) OperationParticipantBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(OperationParticipantBatchInsertBuilder)
}

// MockTransactionParticipantsBatchInsertBuilder is a mock implementation of the
// TransactionParticipantsBatchInsertBuilder interface
type MockTransactionParticipantsBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockTransactionParticipantsBatchInsertBuilder) Add(transactionID, accountID int64) error {
	a := m.Called(transactionID, accountID)
	return a.Error(0)
}

func (m *MockTransactionParticipantsBatchInsertBuilder) Exec() error {
	a := m.Called()
	return a.Error(0)
}
