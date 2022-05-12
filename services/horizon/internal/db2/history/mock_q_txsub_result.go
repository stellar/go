package history

import (
	"context"
	"time"

	"github.com/stellar/go/ingest"
	"github.com/stretchr/testify/mock"
)

// MockQTxSubmissionResult is a mock implementation of the QTxSubmissionResult interface
type MockQTxSubmissionResult struct {
	mock.Mock
}

func (m *MockQTxSubmissionResult) GetTxSubmissionResult(ctx context.Context, hash string) (Transaction, error) {
	a := m.Called(ctx, hash)
	return a.Get(0).(Transaction), a.Error(1)
}

func (m *MockQTxSubmissionResult) SetTxSubmissionResults(ctx context.Context, transactions []ingest.LedgerTransaction, sequence uint32, ledgerClosetime time.Time) (int64, error) {
	a := m.Called(ctx, transactions, sequence)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQTxSubmissionResult) InitEmptyTxSubmissionResult(ctx context.Context, hash string, innerHash string) error {
	a := m.Called(ctx, hash, innerHash)
	return a.Error(0)
}

func (m *MockQTxSubmissionResult) DeleteTxSubmissionResultsOlderThan(ctx context.Context, howOldInSeconds uint64) (int64, error) {
	a := m.Called(ctx, howOldInSeconds)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQTxSubmissionResult) GetTxSubmissionResults(ctx context.Context, hashes []string) ([]Transaction, error) {
	a := m.Called(ctx, hashes)
	return a.Get(0).([]Transaction), a.Error(1)
}
