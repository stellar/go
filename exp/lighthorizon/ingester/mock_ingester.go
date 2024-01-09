package ingester

import (
	"context"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

type MockIngester struct {
	mock.Mock
}

func (m *MockIngester) NewLedgerTransactionReader(
	ledgerCloseMeta xdr.SerializedLedgerCloseMeta,
) (LedgerTransactionReader, error) {
	args := m.Called(ledgerCloseMeta)
	return args.Get(0).(LedgerTransactionReader), args.Error(1)
}

func (m *MockIngester) GetLatestLedgerSequence(ctx context.Context) (uint32, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint32), args.Error(1)
}

func (m *MockIngester) GetLedger(ctx context.Context, sequence uint32) (xdr.SerializedLedgerCloseMeta, error) {
	args := m.Called(ctx, sequence)
	return args.Get(0).(xdr.SerializedLedgerCloseMeta), args.Error(1)
}

func (m *MockIngester) PrepareRange(ctx context.Context, r historyarchive.Range) error {
	args := m.Called(ctx, r)
	return args.Error(0)
}

type MockLedgerTransactionReader struct {
	mock.Mock
}

func (m *MockLedgerTransactionReader) Read() (LedgerTransaction, error) {
	args := m.Called()
	return args.Get(0).(LedgerTransaction), args.Error(1)
}
