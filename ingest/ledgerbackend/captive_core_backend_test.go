package ledgerbackend

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
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

func (m *stellarCoreRunnerMock) runFrom(from uint32, hash string) error {
	a := m.Called(from, hash)
	return a.Error(0)
}

func (m *stellarCoreRunnerMock) getMetaPipe() io.Reader {
	a := m.Called()
	return a.Get(0).(io.Reader)
}

func (m *stellarCoreRunnerMock) getProcessExitChan() <-chan struct{} {
	a := m.Called()
	return a.Get(0).(chan struct{})
}

func (m *stellarCoreRunnerMock) getProcessExitError() error {
	a := m.Called()
	return a.Error(0)
}

func (m *stellarCoreRunnerMock) close() error {
	a := m.Called()
	return a.Error(0)
}

func (m *stellarCoreRunnerMock) setLogger(*log.Entry) {}

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
	configPath := "/etc/stellar-core.cfg"
	networkPassphrase := network.PublicNetworkPassphrase
	historyURLs := []string{"http://history.stellar.org/prd/core-live/core_live_001"}

	captiveStellarCore, err := NewCaptive(
		executablePath,
		configPath,
		networkPassphrase,
		historyURLs,
	)

	assert.NoError(t, err)
	assert.Equal(t, executablePath, captiveStellarCore.executablePath)
	assert.Equal(t, configPath, captiveStellarCore.configPath)
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
	mockRunner.On("getProcessExitChan").Return(make(chan struct{}))
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
		stellarCoreRunnerFactory: func(configPath string) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
	}

	err := captiveBackend.PrepareRange(BoundedRange(100, 200))
	assert.NoError(t, err)
	mockRunner.On("close").Return(nil).Once()
	err = captiveBackend.Close()
	assert.NoError(t, err)
}

func TestCaptivePrepareRangeCrash(t *testing.T) {
	var buf bytes.Buffer

	ch := make(chan struct{})
	close(ch)
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(100), uint32(200)).Return(nil).Once()
	mockRunner.On("getProcessExitChan").Return(ch)
	mockRunner.On("getProcessExitError").Return(errors.New("exit code -1"))
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
		stellarCoreRunnerFactory: func(configPath string) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
	}

	err := captiveBackend.PrepareRange(BoundedRange(100, 200))
	assert.Error(t, err)
	assert.EqualError(t, err, "stellar-core process exited with an error: exit code -1")
}

func TestCaptivePrepareRangeTerminated(t *testing.T) {
	var buf bytes.Buffer

	ch := make(chan struct{})
	close(ch)
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(100), uint32(200)).Return(nil).Once()
	mockRunner.On("getProcessExitChan").Return(ch)
	mockRunner.On("getProcessExitError").Return(nil)
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
		stellarCoreRunnerFactory: func(configPath string) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
	}

	err := captiveBackend.PrepareRange(BoundedRange(100, 200))
	assert.Error(t, err)
	assert.EqualError(t, err, "stellar-core process exited unexpectedly without an error")
}

func TestCaptivePrepareRange_ErrClosingSession(t *testing.T) {
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("close").Return(fmt.Errorf("transient error"))

	captiveBackend := CaptiveStellarCore{
		networkPassphrase: network.PublicNetworkPassphrase,
		nextLedger:        300,
		stellarCoreRunner: mockRunner,
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
	}

	err := captiveBackend.PrepareRange(BoundedRange(100, 200))
	assert.EqualError(t, err, "opening subprocess: sequence: 100 is greater than max available in history archives: 64")
}

func TestCaptivePrepareRange_ToIsAheadOfRootHAS(t *testing.T) {
	var buf bytes.Buffer
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(100), uint32(192)).Return(nil).Once()
	mockRunner.On("getProcessExitChan").Return(make(chan struct{}))
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
		stellarCoreRunnerFactory: func(configPath string) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
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
		stellarCoreRunnerFactory: func(configPath string) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
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
	}

	err := captiveBackend.PrepareRange(UnboundedRange(193))
	assert.EqualError(t, err, "opening subprocess: trying to start online mode too far (latest checkpoint=64), only two checkpoints in the future allowed")
}

func TestCaptivePrepareRangeUnboundedRange_ErrRunFrom(t *testing.T) {
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("runFrom", uint32(126), "0000000000000000000000000000000000000000000000000000000000000000").Return(errors.New("transient error")).Once()
	mockRunner.On("close").Return(nil)

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(127),
		}, nil)

	mockArchive.
		On("GetLedgerHeader", uint32(127)).
		Return(xdr.LedgerHeaderHistoryEntry{}, nil)

	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
		configPath:        "foo",
		stellarCoreRunnerFactory: func(configPath string) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
	}

	err := captiveBackend.PrepareRange(UnboundedRange(128))
	assert.EqualError(t, err, "opening subprocess: error running stellar-core: transient error")
}

func TestCaptivePrepareRangeUnboundedRange_ErrClosingExistingSession(t *testing.T) {
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("close").Return(errors.New("transient error"))

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(127),
		}, nil)

	last := uint32(63)
	captiveBackend := CaptiveStellarCore{
		networkPassphrase: network.PublicNetworkPassphrase,
		nextLedger:        63,
		lastLedger:        &last,
		stellarCoreRunner: mockRunner,
	}

	err := captiveBackend.PrepareRange(UnboundedRange(64))
	assert.EqualError(t, err, "opening subprocess: error closing existing session: error closing stellar-core subprocess: transient error")
}
func TestCaptivePrepareRangeUnboundedRange_ReuseSession(t *testing.T) {
	var buf bytes.Buffer

	for i := 2; i <= 65; i++ {
		writeLedgerHeader(&buf, uint32(i))
	}

	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("runFrom", uint32(62), "0000000000000000000000000000000000000000000000000000000000000000").Return(nil).Once()
	mockRunner.On("runFrom", uint32(63), "0000000000000000000000000000000000000000000000000000000000000000").Return(nil).Once()
	mockRunner.On("getMetaPipe").Return(&buf)
	mockRunner.On("getProcessExitChan").Return(make(chan struct{}))
	mockRunner.On("close").Return(nil)

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(129),
		}, nil)
	mockArchive.
		On("GetLedgerHeader", uint32(63)).
		Return(xdr.LedgerHeaderHistoryEntry{}, nil)

	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
		configPath:        "foo",
		stellarCoreRunnerFactory: func(configPath string) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
	}

	err := captiveBackend.PrepareRange(UnboundedRange(65))
	assert.NoError(t, err)

	captiveBackend.nextLedger = 64
	err = captiveBackend.PrepareRange(UnboundedRange(65))
	assert.NoError(t, err)
}

func TestGetLatestLedgerSequence(t *testing.T) {
	var buf bytes.Buffer

	for i := 2; i <= 200; i++ {
		writeLedgerHeader(&buf, uint32(i))
	}

	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("runFrom", uint32(62), "0000000000000000000000000000000000000000000000000000000000000000").Return(nil).Once()
	mockRunner.On("getMetaPipe").Return(&buf)
	mockRunner.On("getProcessExitChan").Return(make(chan struct{}))
	mockRunner.On("close").Return(nil).Once()

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)

	mockArchive.
		On("GetLedgerHeader", uint32(63)).
		Return(xdr.LedgerHeaderHistoryEntry{}, nil)

	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		networkPassphrase: network.PublicNetworkPassphrase,
		configPath:        "foo",
		stellarCoreRunnerFactory: func(configPath string) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
	}

	err := captiveBackend.PrepareRange(UnboundedRange(64))
	assert.NoError(t, err)

	// To prevent flaky test runs wait for channel to fill.
	waitForBufferToFill(&captiveBackend)

	latest, err := captiveBackend.GetLatestLedgerSequence()
	assert.NoError(t, err)
	// This should be last read ledger + ledgerReadAheadBufferSize.
	assert.Equal(t, uint32(64+ledgerReadAheadBufferSize), latest)

	exists, _, err := captiveBackend.GetLedger(64)
	assert.NoError(t, err)
	assert.True(t, exists)

	waitForBufferToFill(&captiveBackend)

	latest, err = captiveBackend.GetLatestLedgerSequence()
	assert.NoError(t, err)
	// This should be last read ledger + ledgerReadAheadBufferSize.
	assert.Equal(t, uint32(64+ledgerReadAheadBufferSize), latest)

	mockRunner.On("close").Return(nil).Once()
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
	mockRunner.On("getProcessExitChan").Return(make(chan struct{}))
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
		stellarCoreRunnerFactory: func(configPath string) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
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
	mockRunner.On("getProcessExitChan").Return(make(chan struct{}))
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
		stellarCoreRunnerFactory: func(configPath string) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
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
	mockRunner.On("getProcessExitChan").Return(make(chan struct{}))
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
		stellarCoreRunnerFactory: func(configPath string) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
	}

	err := captiveBackend.PrepareRange(BoundedRange(65, 66))
	assert.NoError(t, err)

	// try reading from an empty buffer
	_, _, err = captiveBackend.GetLedger(64)
	tt.EqualError(err, "error reading frame length: unmarshalling XDR frame header: xdr:DecodeUint: EOF while decoding 4 bytes - read: '[]'")

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
	mockRunner.On("getProcessExitChan").Return(make(chan struct{}))
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
		stellarCoreRunnerFactory: func(configPath string) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
	}

	err := captiveBackend.PrepareRange(BoundedRange(65, 66))
	assert.NoError(t, err)

	_, _, err = captiveBackend.GetLedger(66)
	tt.EqualError(err, "error closing session: error closing stellar-core subprocess: transient error")
}

func waitForBufferToFill(captiveBackend *CaptiveStellarCore) {
	for {
		if len(captiveBackend.ledgerBuffer.c) == ledgerReadAheadBufferSize {
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
	mockRunner.On("getProcessExitChan").Return(make(chan struct{}))
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
		stellarCoreRunnerFactory: func(configPath string) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
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

	ch := make(chan struct{})
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(64), uint32(100)).Return(nil).Once()
	mockRunner.On("getProcessExitChan").Return(ch)
	mockRunner.On("getProcessExitError").Return(nil)
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
		stellarCoreRunnerFactory: func(configPath string) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
	}

	go writeLedgerHeader(writer, 64)
	err := captiveBackend.PrepareRange(BoundedRange(64, 100))
	assert.NoError(t, err)
	for {
		// Wait for ledger to appear in the buffer
		if len(captiveBackend.ledgerBuffer.c) == 1 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	exists, meta, err := captiveBackend.GetLedger(64)
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, uint32(64), meta.LedgerSequence())

	close(ch)
	writer.Close()

	_, _, err = captiveBackend.GetLedger(65)
	assert.Error(t, err)
	assert.EqualError(t, err, "stellar-core process exited unexpectedly without an error")
}

func TestCaptiveRunFromParams(t *testing.T) {
	var tests = []struct {
		from           uint32
		runFrom        uint32
		ledgerArchives uint32
		nextLedger     uint32
	}{
		// Before and including 1st checkpoint:
		{2, 2, 3, 2},
		{3, 2, 3, 2},
		{3, 2, 3, 2},
		{4, 2, 3, 2},
		{62, 2, 3, 2},
		{63, 2, 3, 2},

		// Starting from 64 we go normal path: between 1st and 2nd checkpoint:
		{64, 62, 63, 2},
		{65, 62, 63, 2},
		{66, 62, 63, 2},
		{126, 62, 63, 2},

		// between 2nd and 3rd checkpoint... and so on.
		{127, 126, 127, 64},
		{128, 126, 127, 64},
		{129, 126, 127, 64},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("from_%d", tc.from), func(t *testing.T) {
			tt := assert.New(t)
			mockRunner := &stellarCoreRunnerMock{}
			mockArchive := &historyarchive.MockArchive{}
			mockArchive.
				On("GetLedgerHeader", uint32(tc.ledgerArchives)).
				Return(xdr.LedgerHeaderHistoryEntry{
					Header: xdr.LedgerHeader{
						PreviousLedgerHash: xdr.Hash{1, 1, 1, 1},
					},
				}, nil)

			captiveBackend := CaptiveStellarCore{
				archive:           mockArchive,
				networkPassphrase: network.PublicNetworkPassphrase,
				stellarCoreRunner: mockRunner,
			}

			runFrom, ledgerHash, nextLedger, err := captiveBackend.runFromParams(tc.from)
			tt.NoError(err)
			tt.Equal(tc.runFrom, runFrom, "runFrom")
			if tc.from <= 63 {
				tt.Empty(ledgerHash)
			} else {
				tt.Equal("0101010100000000000000000000000000000000000000000000000000000000", ledgerHash)
			}
			tt.Equal(tc.nextLedger, nextLedger, "nextLedger")
		})
	}
}
