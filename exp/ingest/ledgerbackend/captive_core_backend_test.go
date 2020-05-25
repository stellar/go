package ledgerbackend

import (
	"bytes"
	"io"
	"testing"

	"github.com/stellar/go/network"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TODO: test frame decoding
// TODO: test from static base64-encoded data

type stellarCoreRunnerMock struct {
	mock.Mock
}

func (m *stellarCoreRunnerMock) run(from, to uint32) error {
	a := m.Called(from, to)
	return a.Error(0)
}

func (m *stellarCoreRunnerMock) getMetaPipe() io.Reader {
	a := m.Called()
	return a.Get(0).(io.Reader)
}

func (m *stellarCoreRunnerMock) close() error {
	a := m.Called()
	return a.Error(0)
}

func writeLedgerHeader(w io.Writer, sequence uint32) error {
	ledgerCloseMeta := xdr.LedgerCloseMeta{
		V: 0,
		V0: &xdr.LedgerCloseMetaV0{
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Header: xdr.LedgerHeader{
					LedgerSeq: xdr.Uint32(sequence),
				},
			},
		},
	}

	return xdr.MarshalFramed(w, ledgerCloseMeta)
}

func TestPrepareRange(t *testing.T) {
	var buf bytes.Buffer

	// Core will actually start with the last checkpoint before the from ledger
	// and then rewind to the `from` ledger.
	for i := 64; i <= 99; i++ {
		writeLedgerHeader(&buf, uint32(i))
	}

	mockRunner := &stellarCoreRunnerMock{}
	// We prepare [from-1, to] range because it's not possible to rewind the reader
	// and there is no other way to check if stellar-core has built the state without
	// reading actual ledger.
	mockRunner.On("run", uint32(99), uint32(200)).Return(nil).Once()
	mockRunner.On("getMetaPipe").Return(&buf)
	mockRunner.On("close").Return(nil).Once()

	captiveBackend := captiveStellarCore{
		networkPassphrase: network.PublicNetworkPassphrase,
		historyURLs:       []string{"http://history.stellar.org/prd/core-live/core_live_001"},
		stellarCoreRunner: mockRunner,
	}

	err := captiveBackend.PrepareRange(100, 200)
	assert.NoError(t, err)
}
