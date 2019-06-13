package ingestadapters

import (
	"testing"

	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockDatabaseBackend struct {
	mock.Mock
}

func (m *MockDatabaseBackend) GetLatestLedgerSequence() (uint32, error) {
	args := m.Called()
	return args.Get(0).(uint32), args.Error(1)
}

func (m *MockDatabaseBackend) GetLedger(sequence uint32) (bool, ledgerbackend.LedgerCloseMeta, error) {
	args := m.Called()
	return args.Bool(0), args.Get(1).(ledgerbackend.LedgerCloseMeta), args.Error(2)
}

func (m *MockDatabaseBackend) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestGetLatestLedgerSequenceHappyPath(t *testing.T) {
	mockBackend := new(MockDatabaseBackend)
	mockBackend.On("GetLatestLedgerSequence").Return(uint32(1), nil)

	lba := LedgerBackendAdapter{Backend: mockBackend}
	seq, err := lba.GetLatestLedgerSequence()

	if assert.NoError(t, err) {
		assert.Equal(t, uint32(1), seq, "latest seqnum returned from backend")
	}
}

func TestGetLatestLedgerSequenceBackendRequired(t *testing.T) {
	lba := LedgerBackendAdapter{}
	_, err := lba.GetLatestLedgerSequence()

	assert.EqualError(t, err, "missing LedgerBackendAdapter.Backend")
}

func TestGetLedgerBackendRequired(t *testing.T) {
	lba := LedgerBackendAdapter{}
	_, err := lba.GetLedger(uint32(1))

	assert.EqualError(t, err, "missing LedgerBackendAdapter.Backend")
}

func TestCloseBackendRequired(t *testing.T) {
	lba := LedgerBackendAdapter{}
	err := lba.Close()

	assert.EqualError(t, err, "missing LedgerBackendAdapter.Backend")
}
