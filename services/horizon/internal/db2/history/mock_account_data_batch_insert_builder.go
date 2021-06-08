package history

import (
	"context"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

type MockAccountDataBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockAccountDataBatchInsertBuilder) Add(ctx context.Context, entry xdr.LedgerEntry) error {
	a := m.Called(ctx, entry)
	return a.Error(0)
}

func (m *MockAccountDataBatchInsertBuilder) Exec(ctx context.Context) error {
	a := m.Called(ctx)
	return a.Error(0)
}
