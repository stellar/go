package archive

import (
	"context"

	"github.com/stellar/go/support/collections/set"
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

func (m *MockArchive) GetTransactionParticipants(tx LedgerTransaction) (set.Set[string], error) {
	args := m.Called(tx)
	return args.Get(0).(set.Set[string]), args.Error(1)
}

func (m *MockArchive) GetOperationParticipants(tx LedgerTransaction, op xdr.Operation, opIndex int) (set.Set[string], error) {
	args := m.Called(tx, op, opIndex)
	return args.Get(0).(set.Set[string]), args.Error(1)
}
