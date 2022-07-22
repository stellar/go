package archive

import (
	"context"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

type MockLedgerTransactionReader struct {
	mock.Mock
}

func (m *MockLedgerTransactionReader) Read() (LedgerTransaction, error) {
	args := m.Called()
	return args.Get(0).(LedgerTransaction), args.Error(1)
}

type MockArchive struct {
	mock.Mock
}

func (m *MockArchive) GetLedger(ctx context.Context, sequence uint32) (xdr.LedgerCloseMeta, error) {
	args := m.Called(ctx, sequence)
	return args.Get(0).(xdr.LedgerCloseMeta), args.Error(1)
}

func (m *MockArchive) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockArchive) NewLedgerTransactionReaderFromLedgerCloseMeta(networkPassphrase string, ledgerCloseMeta xdr.LedgerCloseMeta) (LedgerTransactionReader, error) {
	args := m.Called(networkPassphrase, ledgerCloseMeta)
	return args.Get(0).(LedgerTransactionReader), args.Error(1)
}

func (m *MockArchive) GetTransactionParticipants(transaction LedgerTransaction) (map[string]struct{}, error) {
	args := m.Called(transaction)
	return args.Get(0).(map[string]struct{}), args.Error(1)
}

func (m *MockArchive) GetOperationParticipants(transaction LedgerTransaction, operation xdr.Operation, opIndex int) (map[string]struct{}, error) {
	args := m.Called(transaction, operation, opIndex)
	return args.Get(0).(map[string]struct{}), args.Error(1)
}
