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
)

// TODO: test frame decoding
// TODO: test from static base64-encoded data

type stellarCoreRunnerMock struct {
	mock.Mock
}

func (m *stellarCoreRunnerMock) catchup(from, to uint32) error {
	a := m.Called(from, to)
	return a.Error(0)
}

func (m *stellarCoreRunnerMock) runFrom(from uint32) error {
	a := m.Called(from)
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

func writeLedgerHeader(w io.Writer, sequence uint32) {
	opResults := []xdr.OperationResult{}
	opMeta := []xdr.OperationMeta{}

	tmpHash, _ := hex.DecodeString("cde54da3901f5b9c0331d24fbb06ac9c5c5de76de9fb2d4a7b86c09e46f11d8c")
	var hash [32]byte
	copy(hash[:], tmpHash)

	source := xdr.MustAddress("GAEJJMDDCRYF752PKIJICUVL7MROJBNXDV2ZB455T7BAFHU2LCLSE2LW")
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
								SourceAccount: source.ToMuxedAccount(),
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

	err := xdr.MarshalFramed(w, ledgerCloseMeta)
	if err != nil {
		panic(err)
	}
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
		writeLedgerHeader(&buf, uint32(i))
	}

	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(100), uint32(200)).Return(nil).Once()
	mockRunner.On("getMetaPipe").Return(&buf)
	mockRunner.On("close").Return(nil).Once()

	captiveBackend := CaptiveStellarCore{
		networkPassphrase: network.PublicNetworkPassphrase,
		historyURLs:       []string{"http://history.stellar.org/prd/core-live/core_live_001"},
		stellarCoreRunner: mockRunner,
	}

	err := captiveBackend.PrepareRange(BoundedRange(100, 200))
	assert.NoError(t, err)
	err = captiveBackend.Close()
	assert.NoError(t, err)
}

func TestGetLatestLedgerSequence(t *testing.T) {
	var buf bytes.Buffer

	for i := 64; i <= 99; i++ {
		writeLedgerHeader(&buf, uint32(i))
	}

	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("runFrom", uint32(64)).Return(nil).Once()
	mockRunner.On("getMetaPipe").Return(&buf)
	mockRunner.On("close").Return(nil).Once()

	captiveBackend := CaptiveStellarCore{
		networkPassphrase: network.PublicNetworkPassphrase,
		historyURLs:       []string{"http://history.stellar.org/prd/core-live/core_live_001"},
		stellarCoreRunner: mockRunner,
	}

	err := captiveBackend.PrepareRange(UnboundedRange(64))
	assert.NoError(t, err)

	// To prevent flaky test runs wait for channel to fill.
	waitForBufferToFill(&captiveBackend)

	latest, err := captiveBackend.GetLatestLedgerSequence()
	assert.NoError(t, err)
	// readAheadBufferSize is 2 so 2 ledgers are buffered: 64 and 65
	assert.Equal(t, uint32(65), latest)

	exists, _, err := captiveBackend.GetLedger(64)
	assert.NoError(t, err)
	assert.True(t, exists)

	waitForBufferToFill(&captiveBackend)

	latest, err = captiveBackend.GetLatestLedgerSequence()
	assert.NoError(t, err)
	// readAheadBufferSize is 2 so 2 ledgers are buffered: 65 and 66
	assert.Equal(t, uint32(66), latest)

	err = captiveBackend.Close()
	assert.NoError(t, err)
}

func waitForBufferToFill(captiveBackend *CaptiveStellarCore) {
	for {
		if len(captiveBackend.metaC) == readAheadBufferSize {
			break
		}
	}
}

func TestGetLedgerBoundsCheck(t *testing.T) {
	var buf bytes.Buffer
	writeLedgerHeader(&buf, 128)
	writeLedgerHeader(&buf, 129)
	writeLedgerHeader(&buf, 130)

	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(128), uint32(130)).Return(nil).Once()
	mockRunner.On("getMetaPipe").Return(&buf)
	mockRunner.On("close").Return(nil).Once()

	captiveBackend := CaptiveStellarCore{
		networkPassphrase: network.PublicNetworkPassphrase,
		historyURLs:       []string{"http://history.stellar.org/prd/core-live/core_live_001"},
		stellarCoreRunner: mockRunner,
	}

	err := captiveBackend.PrepareRange(BoundedRange(128, 130))
	assert.NoError(t, err)

	exists, meta, err := captiveBackend.GetLedger(128)
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, uint32(128), meta.LedgerSequence())

	prev := meta
	exists, meta, err = captiveBackend.GetLedger(128)
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, prev, meta)

	_, _, err = captiveBackend.GetLedger(64)
	assert.EqualError(t, err, "requested ledger 64 is behind the captive core stream (expected=129)")

	err = captiveBackend.Close()
	assert.NoError(t, err)
	mockRunner.AssertExpectations(t)

	buf.Reset()
	writeLedgerHeader(&buf, 64)
	writeLedgerHeader(&buf, 65)
	writeLedgerHeader(&buf, 66)
	mockRunner.On("catchup", uint32(64), uint32(66)).Return(nil).Once()
	mockRunner.On("getMetaPipe").Return(&buf)
	mockRunner.On("close").Return(nil).Once()

	err = captiveBackend.PrepareRange(BoundedRange(64, 66))
	assert.NoError(t, err)

	exists, meta, err = captiveBackend.GetLedger(64)
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, uint32(64), meta.LedgerSequence())

	err = captiveBackend.Close()
	assert.NoError(t, err)
	mockRunner.AssertExpectations(t)
}
