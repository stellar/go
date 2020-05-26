package ledgerbackend

import (
	"bytes"
	"encoding/hex"
	"io"
	"testing"

	"github.com/stellar/go/network"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
	opResults := []xdr.OperationResult{}
	opMeta := []xdr.OperationMeta{}

	tmpHash, _ := hex.DecodeString("cde54da3901f5b9c0331d24fbb06ac9c5c5de76de9fb2d4a7b86c09e46f11d8c")
	var hash [32]byte
	copy(hash[:], tmpHash)

	ledgerCloseMeta := xdr.LedgerCloseMeta{
		V: 0,
		V0: &xdr.LedgerCloseMetaV0{
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Header: xdr.LedgerHeader{
					LedgerSeq: xdr.Uint32(sequence),
				},
			},
			TxSet: xdr.TransactionSet{
				Txs: []xdr.TransactionEnvelope{
					{
						Type: xdr.EnvelopeTypeEnvelopeTypeTx,
						V1: &xdr.TransactionV1Envelope{
							Tx: xdr.Transaction{
								SourceAccount: xdr.MustMuxedAccountAddress("GAEJJMDDCRYF752PKIJICUVL7MROJBNXDV2ZB455T7BAFHU2LCLSE2LW"),
								Fee:           xdr.Uint32(sequence),
							},
						},
					},
				},
			},
			TxProcessing: []xdr.TransactionResultMeta{
				{
					Result: xdr.TransactionResultPair{
						TransactionHash: xdr.Hash(hash),
						Result: xdr.TransactionResult{
							FeeCharged: xdr.Int64(sequence),
							Result: xdr.TransactionResultResult{
								Code:    xdr.TransactionResultCodeTxSuccess,
								Results: &opResults,
							},
						},
					},
					TxApplyProcessing: xdr.TransactionMeta{
						Operations: &opMeta,
					},
				},
			},
		},
	}

	return xdr.MarshalFramed(w, ledgerCloseMeta)
}

func TestCaptiveNew(t *testing.T) {
	executablePath := "/etc/stellar-core"
	networkPassphrase := network.PublicNetworkPassphrase
	historyURLs := []string{"http://history.stellar.org/prd/core-live/core_live_001"}

	captiveStellarCore := NewCaptive(
		executablePath,
		networkPassphrase,
		historyURLs,
	)

	assert.Equal(t, networkPassphrase, captiveStellarCore.networkPassphrase)
	assert.Equal(t, historyURLs, captiveStellarCore.historyURLs)
	assert.Equal(t, uint32(0), captiveStellarCore.nextLedger)

	assert.Equal(t, executablePath, captiveStellarCore.stellarCoreRunner.(*stellarCoreRunner).executablePath)
	assert.Equal(t, networkPassphrase, captiveStellarCore.stellarCoreRunner.(*stellarCoreRunner).networkPassphrase)
	assert.Equal(t, historyURLs, captiveStellarCore.stellarCoreRunner.(*stellarCoreRunner).historyURLs)
}

func TestCaptivePrepareRange(t *testing.T) {
	var buf bytes.Buffer

	// Core will actually start with the last checkpoint before the from ledger
	// and then rewind to the `from` ledger.
	for i := 64; i <= 99; i++ {
		err := writeLedgerHeader(&buf, uint32(i))
		require.NoError(t, err)
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
	err = captiveBackend.Close()
	assert.NoError(t, err)
}
