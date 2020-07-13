package ledgerbackend

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"testing"

	"github.com/stellar/go/network"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
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

func (m *stellarCoreRunnerMock) getProcessExitChan() <-chan error {
	a := m.Called()
	return a.Get(0).(chan error)
}

func (m *stellarCoreRunnerMock) close() error {
	a := m.Called()
	return a.Error(0)
}

func buildLedgerCloseMeta(sequence uint32) xdr.LedgerCloseMeta {
	opResults := []xdr.OperationResult{}
	opMeta := []xdr.OperationMeta{}

	tmpHash, _ := hex.DecodeString("cde54da3901f5b9c0331d24fbb06ac9c5c5de76de9fb2d4a7b86c09e46f11d8c")
	var hash [32]byte
	copy(hash[:], tmpHash)

	source := xdr.MustAddress("GAEJJMDDCRYF752PKIJICUVL7MROJBNXDV2ZB455T7BAFHU2LCLSE2LW")
	return xdr.LedgerCloseMeta{
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

}

func writeLedgerHeader(w io.Writer, sequence uint32) {
	err := xdr.MarshalFramed(w, buildLedgerCloseMeta(sequence))
	if err != nil {
		panic(err)
	}
}

func TestCaptiveNew(t *testing.T) {
	executablePath := "/etc/stellar-core"
	networkPassphrase := network.PublicNetworkPassphrase
	historyURLs := []string{"http://history.stellar.org/prd/core-live/core_live_001"}

	captiveStellarCore, err := NewCaptive(
		executablePath,
		networkPassphrase,
		historyURLs,
	)

	assert.NoError(t, err)
	assert.Equal(t, executablePath, captiveStellarCore.executablePath)
	assert.Equal(t, networkPassphrase, captiveStellarCore.networkPassphrase)
	assert.Equal(t, historyURLs, captiveStellarCore.historyURLs)
	assert.Equal(t, uint32(0), captiveStellarCore.nextLedger)
	assert.NotNil(t, captiveStellarCore.archive)
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
	mockRunner.On("getProcessExitChan").Return(make(chan error))
	mockRunner.On("getMetaPipe").Return(&buf)
	mockRunner.On("close").Return(nil).Once()

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
		stellarCoreRunner: mockRunner,
	}

	err := captiveBackend.PrepareRange(BoundedRange(100, 200))
	assert.NoError(t, err)
	err = captiveBackend.Close()
	assert.NoError(t, err)
}

func TestCaptivePrepareRangeCrash(t *testing.T) {
	var buf bytes.Buffer

	ch := make(chan error, 1) // we use buffered channel in tests only
	ch <- errors.New("exit code -1")
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(100), uint32(200)).Return(nil).Once()
	mockRunner.On("getProcessExitChan").Return(ch)
	mockRunner.On("getMetaPipe").Return(&buf)
	mockRunner.On("close").Return(nil).Once()

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
		stellarCoreRunner: mockRunner,
	}

	err := captiveBackend.PrepareRange(BoundedRange(100, 200))
	assert.Error(t, err)
	assert.EqualError(t, err, "stellar-core process exited with an error: exit code -1")
}

func TestCaptivePrepareRangeTerminated(t *testing.T) {
	var buf bytes.Buffer

	ch := make(chan error, 1) // we use buffered channel in tests only
	ch <- nil
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(100), uint32(200)).Return(nil).Once()
	mockRunner.On("getProcessExitChan").Return(ch)
	mockRunner.On("getMetaPipe").Return(&buf)
	mockRunner.On("close").Return(nil).Once()

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
		stellarCoreRunner: mockRunner,
	}

	err := captiveBackend.PrepareRange(BoundedRange(100, 200))
	assert.Error(t, err)
	assert.EqualError(t, err, "stellar-core process exited without an error unexpectedly")
}

func TestCaptivePrepareRange_ErrClosingSession(t *testing.T) {
	var buf bytes.Buffer
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(100), uint32(200)).Return(nil).Once()
	mockRunner.On("getMetaPipe").Return(&buf)
	mockRunner.On("close").Return(fmt.Errorf("transient error"))

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
		stellarCoreRunner: mockRunner,
		nextLedger:        1,
	}

	err := captiveBackend.PrepareRange(BoundedRange(100, 200))
	assert.EqualError(t, err, "opening subprocess: error closing existing session: error closing stellar-core subprocess: transient error")
}

func TestCaptivePrepareRange_ErrGettingRootHAS(t *testing.T) {
	var buf bytes.Buffer
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(100), uint32(200)).Return(nil).Once()
	mockRunner.On("getMetaPipe").Return(&buf)
	mockRunner.On("close").Return(nil)

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{}, errors.New("transient error"))

	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
		stellarCoreRunner: mockRunner,
	}

	err := captiveBackend.PrepareRange(BoundedRange(100, 200))
	assert.EqualError(t, err, "opening subprocess: error getting latest checkpoint sequence: error getting root HAS: transient error")
}

func TestCaptivePrepareRange_FromIsAheadOfRootHAS(t *testing.T) {
	var buf bytes.Buffer
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(100), uint32(200)).Return(nil).Once()
	mockRunner.On("getMetaPipe").Return(&buf)
	mockRunner.On("close").Return(nil)

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(64),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
		stellarCoreRunner: mockRunner,
	}

	err := captiveBackend.PrepareRange(BoundedRange(100, 200))
	assert.EqualError(t, err, "opening subprocess: sequence: 100 is greater than max available in history archives: 64")
}

func TestCaptivePrepareRange_ToIsAheadOfRootHAS(t *testing.T) {
	var buf bytes.Buffer
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(100), uint32(192)).Return(nil).Once()
	mockRunner.On("getProcessExitChan").Return(make(chan error))
	mockRunner.On("getMetaPipe").Return(&buf)
	mockRunner.On("close").Return(nil)

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(192),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
		stellarCoreRunner: mockRunner,
	}

	err := captiveBackend.PrepareRange(BoundedRange(100, 200))
	assert.NoError(t, err)
}

func TestCaptivePrepareRange_ErrCatchup(t *testing.T) {
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(100), uint32(192)).Return(errors.New("transient error")).Once()
	mockRunner.On("close").Return(nil)

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(192),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
		stellarCoreRunner: mockRunner,
	}

	err := captiveBackend.PrepareRange(BoundedRange(100, 200))
	assert.Error(t, err)
	assert.EqualError(t, err, "opening subprocess: error running stellar-core: transient error")
}

func TestCaptivePrepareRangeUnboundedRange_ErrGettingRootHAS(t *testing.T) {
	var buf bytes.Buffer
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(100), uint32(192)).Return(nil).Once()
	mockRunner.On("getMetaPipe").Return(&buf)
	mockRunner.On("close").Return(nil)

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{}, errors.New("transient error"))

	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
		stellarCoreRunner: mockRunner,
	}

	err := captiveBackend.PrepareRange(UnboundedRange(100))
	assert.EqualError(t, err, "opening subprocess: error getting latest checkpoint sequence: error getting root HAS: transient error")
}

func TestCaptivePrepareRangeUnboundedRange_FromIsTooFarAheadOfLatestHAS(t *testing.T) {
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("close").Return(nil)

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(64),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
		stellarCoreRunner: mockRunner,
	}

	err := captiveBackend.PrepareRange(UnboundedRange(193))
	assert.EqualError(t, err, "opening subprocess: trying to start online mode too far (latest checkpoint=64), only two checkpoints in the future allowed")
}

func TestCaptivePrepareRangeUnboundedRange_ErrRunFrom(t *testing.T) {
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("runFrom", uint32(65)).Return(errors.New("transient error")).Once()
	mockRunner.On("close").Return(nil)

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(64),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
		stellarCoreRunner: mockRunner,
	}

	err := captiveBackend.PrepareRange(UnboundedRange(65))
	assert.EqualError(t, err, "opening subprocess: error running stellar-core: transient error")
}

func TestCaptivePrepareRangeUnboundedRange_ErrClosingExistingSession(t *testing.T) {
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("close").Return(errors.New("transient error"))

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(64),
		}, nil)

	last := uint32(63)
	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
		stellarCoreRunner: mockRunner,
		nextLedger:        63,
		lastLedger:        &last,
	}

	err := captiveBackend.PrepareRange(UnboundedRange(64))
	assert.EqualError(t, err, "opening subprocess: error closing existing session: error closing stellar-core subprocess: transient error")
}
func TestCaptivePrepareRangeUnboundedRange_ReuseSession(t *testing.T) {
	var buf bytes.Buffer

	for i := 60; i <= 65; i++ {
		writeLedgerHeader(&buf, uint32(i))
	}

	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("runFrom", uint32(65)).Return(nil).Once()
	mockRunner.On("getMetaPipe").Return(&buf)
	mockRunner.On("close").Return(nil)

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(129),
		}, nil)
	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
		stellarCoreRunner: mockRunner,
	}

	err := captiveBackend.PrepareRange(UnboundedRange(65))
	assert.NoError(t, err)

	captiveBackend.nextLedger = 64
	err = captiveBackend.PrepareRange(UnboundedRange(65))
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
	mockRunner.On("getProcessExitChan").Return(make(chan error))
	mockRunner.On("close").Return(nil).Once()

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
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
func TestCaptiveGetLedger(t *testing.T) {
	tt := assert.New(t)
	var buf bytes.Buffer
	for i := 64; i <= 66; i++ {
		writeLedgerHeader(&buf, uint32(i))
	}

	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(65), uint32(66)).Return(nil)
	mockRunner.On("getMetaPipe").Return(&buf)
	mockRunner.On("close").Return(nil)

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
		stellarCoreRunner: mockRunner,
	}

	// requires PrepareRange
	_, _, err := captiveBackend.GetLedger(64)
	tt.EqualError(err, "session is closed, call PrepareRange first")

	err = captiveBackend.PrepareRange(BoundedRange(65, 66))
	assert.NoError(t, err)

	// reads value from buffer
	found, meta, err := captiveBackend.GetLedger(64)
	tt.NoError(err)
	tt.True(found)
	tt.Equal(xdr.Uint32(64), meta.V0.LedgerHeader.Header.LedgerSeq)

	// advance to next sequence number
	tt.Equal(uint32(65), captiveBackend.nextLedger)

	// reads value from cachedMeta
	_, cachedMeta, err := captiveBackend.GetLedger(64)
	tt.NoError(err)
	tt.Equal(meta, cachedMeta)

	// next sequence number didn't get consumed
	tt.Equal(uint32(65), captiveBackend.nextLedger)

	_, _, err = captiveBackend.GetLedger(66)
	tt.NoError(err)

	// closes after last ledger is consumed
	tt.True(captiveBackend.isClosed())

	// we should be able to call last ledger even after get ledger is closed
	_, _, err = captiveBackend.GetLedger(66)
	tt.NoError(err)
}

func TestCaptiveGetLedger_NextLedgerIsDifferentToLedgerFromBuffer(t *testing.T) {
	tt := assert.New(t)
	var buf bytes.Buffer
	for i := 64; i <= 65; i++ {
		writeLedgerHeader(&buf, uint32(i))
	}

	writeLedgerHeader(&buf, uint32(68))

	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(65), uint32(66)).Return(nil)
	mockRunner.On("getMetaPipe").Return(&buf)
	mockRunner.On("close").Return(nil)

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
		stellarCoreRunner: mockRunner,
	}

	err := captiveBackend.PrepareRange(BoundedRange(65, 66))
	assert.NoError(t, err)

	_, _, err = captiveBackend.GetLedger(66)
	tt.EqualError(err, "unexpected ledger (expected=66 actual=68)")
}
func TestCaptiveGetLedger_ErrReadingMetaResult(t *testing.T) {
	tt := assert.New(t)
	var buf bytes.Buffer

	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(65), uint32(66)).Return(nil)
	mockRunner.On("getProcessExitChan").Return(make(chan error))
	mockRunner.On("getMetaPipe").Return(&buf)
	mockRunner.On("close").Return(nil)

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
		stellarCoreRunner: mockRunner,
	}

	err := captiveBackend.PrepareRange(BoundedRange(65, 66))
	assert.NoError(t, err)

	// try reading from an empty buffer
	_, _, err = captiveBackend.GetLedger(64)
	tt.EqualError(err, "unmarshalling framed LedgerCloseMeta: unmarshalling XDR frame header: xdr:DecodeUint: EOF while decoding 4 bytes - read: '[]'")

	// closes if there is an error getting ledger
	tt.True(captiveBackend.isClosed())
}
func TestCaptiveGetLedger_ErrClosingAfterLastLedger(t *testing.T) {
	tt := assert.New(t)
	var buf bytes.Buffer
	for i := 64; i <= 66; i++ {
		writeLedgerHeader(&buf, uint32(i))
	}

	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(65), uint32(66)).Return(nil)
	mockRunner.On("getMetaPipe").Return(&buf)
	mockRunner.On("close").Return(fmt.Errorf("transient error"))

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
		stellarCoreRunner: mockRunner,
	}

	err := captiveBackend.PrepareRange(BoundedRange(65, 66))
	assert.NoError(t, err)

	_, _, err = captiveBackend.GetLedger(66)
	tt.EqualError(err, "error closing session: error closing stellar-core subprocess: transient error")
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

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
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

func TestCaptiveGetLedgerTerminated(t *testing.T) {
	reader, writer := io.Pipe()

	ch := make(chan error, 1) // we use buffered channel in tests only
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(64), uint32(100)).Return(nil).Once()
	mockRunner.On("getProcessExitChan").Return(ch)
	mockRunner.On("getMetaPipe").Return(reader)
	mockRunner.On("close").Return(nil).Once()

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
		stellarCoreRunner: mockRunner,
	}

	go writeLedgerHeader(writer, 64)
	err := captiveBackend.PrepareRange(BoundedRange(64, 100))
	assert.NoError(t, err)

	ch <- nil
	writer.Close()

	exists, meta, err := captiveBackend.GetLedger(64)
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, uint32(64), meta.LedgerSequence())

	_, _, err = captiveBackend.GetLedger(65)
	assert.Error(t, err)
	assert.EqualError(t, err, "stellar-core process exited without an error unexpectedly")
}
