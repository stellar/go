package history

import (
	"github.com/stretchr/testify/mock"

	"github.com/stellar/go/ingest"
)

type MockTransactionsBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockTransactionsBatchInsertBuilder) Add(transaction ingest.LedgerTransaction, sequence uint32) error {
	a := m.Called(transaction, sequence)
	return a.Error(0)
}

func (m *MockTransactionsBatchInsertBuilder) Exec() error {
	a := m.Called()
	return a.Error(0)
}
