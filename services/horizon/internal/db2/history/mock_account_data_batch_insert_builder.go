package history

import (
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

type MockAccountDataBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockAccountDataBatchInsertBuilder) Add(entry xdr.LedgerEntry) error {
	a := m.Called(entry)
	return a.Error(0)
}

func (m *MockAccountDataBatchInsertBuilder) Exec() error {
	a := m.Called()
	return a.Error(0)
}
