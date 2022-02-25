package ingest

import (
	"context"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest/processors"
	"github.com/stretchr/testify/mock"
)

type mockFilters struct {
	mock.Mock
}

func (m *mockFilters) GetFilters(filterQ history.QFilter, ctx context.Context) []processors.LedgerTransactionFilterer {
	a := m.Called(filterQ, ctx)
	return a.Get(0).([]processors.LedgerTransactionFilterer)
}
