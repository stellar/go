package io

import (
	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

type MockLedgerReadCloser struct {
	mock.Mock
}

func (m *MockLedgerReadCloser) GetSequence() uint32 {
	args := m.Called()
	return args.Get(0).(uint32)
}

func (m *MockLedgerReadCloser) GetHeader() xdr.LedgerHeaderHistoryEntry {
	args := m.Called()
	return args.Get(0).(xdr.LedgerHeaderHistoryEntry)
}

func (m *MockLedgerReadCloser) Read() (LedgerTransaction, error) {
	args := m.Called()
	return args.Get(0).(LedgerTransaction), args.Error(1)
}

func (m *MockLedgerReadCloser) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockLedgerReadCloser) Init(sequence uint32, backend ledgerbackend.LedgerBackend) error {
	args := m.Called()
	return args.Error(0)
}
