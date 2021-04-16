package history

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/stellar/go/xdr"
)

// MockQTrustLines is a mock implementation of the QOffers interface
type MockQTrustLines struct {
	mock.Mock
}

func (m *MockQTrustLines) NewTrustLinesBatchInsertBuilder(maxBatchSize int) TrustLinesBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(TrustLinesBatchInsertBuilder)
}

func (m *MockQTrustLines) GetTrustLinesByKeys(ctx context.Context, keys []xdr.LedgerKeyTrustLine) ([]TrustLine, error) {
	a := m.Called(ctx, keys)
	return a.Get(0).([]TrustLine), a.Error(1)
}

func (m *MockQTrustLines) InsertTrustLine(ctx context.Context, entry xdr.LedgerEntry) (int64, error) {
	a := m.Called(ctx, entry)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQTrustLines) UpdateTrustLine(ctx context.Context, entry xdr.LedgerEntry) (int64, error) {
	a := m.Called(ctx, entry)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQTrustLines) UpsertTrustLines(ctx context.Context, trustLines []xdr.LedgerEntry) error {
	a := m.Called(ctx, trustLines)
	return a.Error(0)
}

func (m *MockQTrustLines) RemoveTrustLine(ctx context.Context, key xdr.LedgerKeyTrustLine) (int64, error) {
	a := m.Called(ctx, key)
	return a.Get(0).(int64), a.Error(1)
}
