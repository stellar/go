package ledgerbackend

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// TODO: test frame decoding
// TODO: test from static base64-encoded data

type stellarCoreRunnerMock struct {
	mock.Mock
}

func (m *stellarCoreRunnerMock) context() context.Context {
	a := m.Called()
	return a.Get(0).(context.Context)
}

func (m *stellarCoreRunnerMock) catchup(from, to uint32) error {
	a := m.Called(from, to)
	return a.Error(0)
}

func (m *stellarCoreRunnerMock) runFrom(from uint32, hash string) error {
	a := m.Called(from, hash)
	return a.Error(0)
}

func (m *stellarCoreRunnerMock) getMetaPipe() <-chan metaResult {
	a := m.Called()
	return a.Get(0).(<-chan metaResult)
}

func (m *stellarCoreRunnerMock) getProcessExitError() (bool, error) {
	a := m.Called()
	return a.Bool(0), a.Error(1)
}

func (m *stellarCoreRunnerMock) close() error {
	a := m.Called()
	return a.Error(0)
}

func buildLedgerCloseMeta(header testLedgerHeader) xdr.LedgerCloseMeta {
	opResults := []xdr.OperationResult{}
	opMeta := []xdr.OperationMeta{}

	tmpHash, _ := hex.DecodeString("cde54da3901f5b9c0331d24fbb06ac9c5c5de76de9fb2d4a7b86c09e46f11d8c")
	var hash [32]byte
	copy(hash[:], tmpHash)

	var ledgerHash [32]byte
	if header.hash != "" {
		tmpHash, err := hex.DecodeString(header.hash)
		if err != nil {
			panic(err)
		}
		copy(ledgerHash[:], tmpHash)
	}

	var previousLedgerHash [32]byte
	if header.hash != "" {
		tmpHash, err := hex.DecodeString(header.previousLedgerHash)
		if err != nil {
			panic(err)
		}
		copy(previousLedgerHash[:], tmpHash)
	}

	source := xdr.MustAddress("GAEJJMDDCRYF752PKIJICUVL7MROJBNXDV2ZB455T7BAFHU2LCLSE2LW")
	return xdr.LedgerCloseMeta{
		V: 0,
		V0: &xdr.LedgerCloseMetaV0{
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Hash: ledgerHash,
				Header: xdr.LedgerHeader{
					LedgerSeq:          xdr.Uint32(header.sequence),
					PreviousLedgerHash: previousLedgerHash,
				},
			},
			TxSet: xdr.TransactionSet{
				Txs: []xdr.TransactionEnvelope{
					{
						Type: xdr.EnvelopeTypeEnvelopeTypeTx,
						V1: &xdr.TransactionV1Envelope{
							Tx: xdr.Transaction{
								SourceAccount: source.ToMuxedAccount(),
								Fee:           xdr.Uint32(header.sequence),
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
							FeeCharged: xdr.Int64(header.sequence),
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

type testLedgerHeader struct {
	sequence           uint32
	hash               string
	previousLedgerHash string
}

func TestCaptiveNew(t *testing.T) {
	executablePath := "/etc/stellar-core"
	configPath := "/etc/stellar-core.cfg"
	networkPassphrase := network.PublicNetworkPassphrase
	historyURLs := []string{"http://history.stellar.org/prd/core-live/core_live_001"}

	captiveStellarCore, err := NewCaptive(
		CaptiveCoreConfig{
			BinaryPath:         executablePath,
			ConfigAppendPath:   configPath,
			NetworkPassphrase:  networkPassphrase,
			HistoryArchiveURLs: historyURLs,
		},
	)

	assert.NoError(t, err)
	assert.Equal(t, uint32(0), captiveStellarCore.nextLedger)
	assert.NotNil(t, captiveStellarCore.archive)
}

func TestCaptivePrepareRange(t *testing.T) {
	metaChan := make(chan metaResult, 100)

	// Core will actually start with the last checkpoint before the from ledger
	// and then rewind to the `from` ledger.
	for i := 64; i <= 100; i++ {
		meta := buildLedgerCloseMeta(testLedgerHeader{sequence: uint32(i)})
		metaChan <- metaResult{
			LedgerCloseMeta: &meta,
		}
	}

	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(100), uint32(200)).Return(nil).Once()
	mockRunner.On("getMetaPipe").Return((<-chan metaResult)(metaChan))
	mockRunner.On("context").Return(context.Background())

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)

	cancelCalled := false
	captiveBackend := CaptiveStellarCore{
		archive: mockArchive,
		stellarCoreRunnerFactory: func(_ stellarCoreRunnerMode) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
		checkpointManager: historyarchive.NewCheckpointManager(64),
		cancel: context.CancelFunc(func() {
			cancelCalled = true
		}),
	}

	err := captiveBackend.PrepareRange(BoundedRange(100, 200))
	assert.NoError(t, err)
	mockRunner.On("close").Return(nil).Once()
	err = captiveBackend.Close()
	assert.NoError(t, err)
	assert.True(t, cancelCalled)
	mockRunner.AssertExpectations(t)
	mockArchive.AssertExpectations(t)
}

func TestCaptivePrepareRangeCrash(t *testing.T) {
	metaChan := make(chan metaResult)
	close(metaChan)
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(100), uint32(200)).Return(nil).Once()
	mockRunner.On("getProcessExitError").Return(true, errors.New("exit code -1"))
	mockRunner.On("getMetaPipe").Return((<-chan metaResult)(metaChan))
	mockRunner.On("close").Return(nil).Once()
	mockRunner.On("context").Return(context.Background())

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive: mockArchive,
		stellarCoreRunnerFactory: func(_ stellarCoreRunnerMode) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
		checkpointManager: historyarchive.NewCheckpointManager(64),
	}

	err := captiveBackend.PrepareRange(BoundedRange(100, 200))
	assert.EqualError(t, err, "Error fast-forwarding to 100: stellar core exited unexpectedly: exit code -1")
	mockRunner.AssertExpectations(t)
	mockArchive.AssertExpectations(t)
}

func TestCaptivePrepareRangeTerminated(t *testing.T) {
	metaChan := make(chan metaResult, 100)

	// Core will actually start with the last checkpoint before the from ledger
	// and then rewind to the `from` ledger.
	for i := 64; i <= 100; i++ {
		meta := buildLedgerCloseMeta(testLedgerHeader{sequence: uint32(i)})
		metaChan <- metaResult{
			LedgerCloseMeta: &meta,
		}
	}
	close(metaChan)
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(100), uint32(200)).Return(nil).Once()
	mockRunner.On("getMetaPipe").Return((<-chan metaResult)(metaChan))
	mockRunner.On("context").Return(context.Background())

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive: mockArchive,
		stellarCoreRunnerFactory: func(_ stellarCoreRunnerMode) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
		checkpointManager: historyarchive.NewCheckpointManager(64),
	}

	err := captiveBackend.PrepareRange(BoundedRange(100, 200))
	assert.NoError(t, err)
	mockRunner.AssertExpectations(t)
	mockArchive.AssertExpectations(t)
}

func TestCaptivePrepareRange_ErrClosingSession(t *testing.T) {
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("close").Return(fmt.Errorf("transient error"))
	mockRunner.On("context").Return(context.Background())

	captiveBackend := CaptiveStellarCore{
		nextLedger:        300,
		stellarCoreRunner: mockRunner,
	}

	err := captiveBackend.PrepareRange(BoundedRange(100, 200))
	assert.EqualError(t, err, "error starting prepare range: error closing existing session: transient error")

	err = captiveBackend.PrepareRange(UnboundedRange(64))
	assert.EqualError(t, err, "error starting prepare range: error closing existing session: transient error")

	mockRunner.AssertExpectations(t)
}

func TestCaptivePrepareRange_ErrGettingRootHAS(t *testing.T) {
	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{}, errors.New("transient error"))

	captiveBackend := CaptiveStellarCore{
		archive: mockArchive,
	}

	err := captiveBackend.PrepareRange(BoundedRange(100, 200))
	assert.EqualError(t, err, "error starting prepare range: opening subprocess: error getting latest checkpoint sequence: error getting root HAS: transient error")

	err = captiveBackend.PrepareRange(UnboundedRange(100))
	assert.EqualError(t, err, "error starting prepare range: opening subprocess: error getting latest checkpoint sequence: error getting root HAS: transient error")

	mockArchive.AssertExpectations(t)
}

func TestCaptivePrepareRange_FromIsAheadOfRootHAS(t *testing.T) {
	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(64),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive: mockArchive,
	}

	err := captiveBackend.PrepareRange(BoundedRange(100, 200))
	assert.EqualError(t, err, "error starting prepare range: opening subprocess: sequence: 100 is greater than max available in history archives: 64")

	err = captiveBackend.PrepareRange(UnboundedRange(100))
	assert.EqualError(t, err, "error starting prepare range: opening subprocess: trying to start online mode too far (latest checkpoint=64), only two checkpoints in the future allowed")

	mockArchive.AssertExpectations(t)
}

func TestCaptivePrepareRange_ToIsAheadOfRootHAS(t *testing.T) {
	metaChan := make(chan metaResult, 100)

	// Core will actually start with the last checkpoint before the from ledger
	// and then rewind to the `from` ledger.
	for i := 64; i <= 100; i++ {
		meta := buildLedgerCloseMeta(testLedgerHeader{sequence: uint32(i)})
		metaChan <- metaResult{
			LedgerCloseMeta: &meta,
		}
	}

	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(100), uint32(192)).Return(nil).Once()
	mockRunner.On("getMetaPipe").Return((<-chan metaResult)(metaChan))
	mockRunner.On("context").Return(context.Background())

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(192),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive: mockArchive,
		stellarCoreRunnerFactory: func(_ stellarCoreRunnerMode) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
		checkpointManager: historyarchive.NewCheckpointManager(64),
	}

	err := captiveBackend.PrepareRange(BoundedRange(100, 200))
	assert.NoError(t, err)

	mockArchive.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
}

func TestCaptivePrepareRange_ErrCatchup(t *testing.T) {
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(100), uint32(192)).Return(errors.New("transient error")).Once()
	mockRunner.On("close").Return(nil).Once()

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(192),
		}, nil)

	cancelCalled := false
	captiveBackend := CaptiveStellarCore{
		archive: mockArchive,
		stellarCoreRunnerFactory: func(_ stellarCoreRunnerMode) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
		cancel: context.CancelFunc(func() {
			cancelCalled = true
		}),
	}

	err := captiveBackend.PrepareRange(BoundedRange(100, 200))
	assert.EqualError(t, err, "error starting prepare range: opening subprocess: error running stellar-core: transient error")

	// make sure we can Close without errors
	assert.NoError(t, captiveBackend.Close())
	assert.True(t, cancelCalled)

	mockArchive.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
}

func TestCaptivePrepareRangeUnboundedRange_ErrRunFrom(t *testing.T) {
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("runFrom", uint32(126), "0000000000000000000000000000000000000000000000000000000000000000").Return(errors.New("transient error")).Once()
	mockRunner.On("close").Return(nil).Once()

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(127),
		}, nil)

	mockArchive.
		On("GetLedgerHeader", uint32(127)).
		Return(xdr.LedgerHeaderHistoryEntry{}, nil)

	cancelCalled := false
	captiveBackend := CaptiveStellarCore{
		archive: mockArchive,
		stellarCoreRunnerFactory: func(_ stellarCoreRunnerMode) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
		checkpointManager: historyarchive.NewCheckpointManager(64),
		cancel: context.CancelFunc(func() {
			cancelCalled = true
		}),
	}

	err := captiveBackend.PrepareRange(UnboundedRange(128))
	assert.EqualError(t, err, "error starting prepare range: opening subprocess: error running stellar-core: transient error")

	// make sure we can Close without errors
	assert.NoError(t, captiveBackend.Close())
	assert.True(t, cancelCalled)

	mockArchive.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
}

func TestCaptivePrepareRangeUnboundedRange_ReuseSession(t *testing.T) {
	metaChan := make(chan metaResult, 100)

	// Core will actually start with the last checkpoint before the from ledger
	// and then rewind to the `from` ledger.
	for i := 2; i <= 65; i++ {
		meta := buildLedgerCloseMeta(testLedgerHeader{sequence: uint32(i)})
		metaChan <- metaResult{
			LedgerCloseMeta: &meta,
		}
	}

	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("runFrom", uint32(62), "0000000000000000000000000000000000000000000000000000000000000000").Return(nil).Once()
	mockRunner.On("getMetaPipe").Return((<-chan metaResult)(metaChan))
	mockRunner.On("context").Return(context.Background())

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
		archive: mockArchive,
		stellarCoreRunnerFactory: func(_ stellarCoreRunnerMode) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
		checkpointManager: historyarchive.NewCheckpointManager(64),
	}

	err := captiveBackend.PrepareRange(UnboundedRange(65))
	assert.NoError(t, err)

	captiveBackend.nextLedger = 64
	err = captiveBackend.PrepareRange(UnboundedRange(65))
	assert.NoError(t, err)

	mockArchive.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
}

func TestGetLatestLedgerSequence(t *testing.T) {
	metaChan := make(chan metaResult, 300)

	// Core will actually start with the last checkpoint before the from ledger
	// and then rewind to the `from` ledger.
	for i := 2; i <= 200; i++ {
		meta := buildLedgerCloseMeta(testLedgerHeader{sequence: uint32(i)})
		metaChan <- metaResult{
			LedgerCloseMeta: &meta,
		}
	}

	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("runFrom", uint32(62), "0000000000000000000000000000000000000000000000000000000000000000").Return(nil).Once()
	mockRunner.On("getMetaPipe").Return((<-chan metaResult)(metaChan))
	mockRunner.On("context").Return(context.Background())

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
		archive: mockArchive,
		stellarCoreRunnerFactory: func(_ stellarCoreRunnerMode) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
		checkpointManager: historyarchive.NewCheckpointManager(64),
	}

	err := captiveBackend.PrepareRange(UnboundedRange(64))
	assert.NoError(t, err)

	latest, err := captiveBackend.GetLatestLedgerSequence()
	assert.NoError(t, err)
	assert.Equal(t, uint32(200), latest)

	mockArchive.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
}

func TestCaptiveGetLedger(t *testing.T) {
	tt := assert.New(t)
	metaChan := make(chan metaResult, 300)

	for i := 64; i <= 66; i++ {
		meta := buildLedgerCloseMeta(testLedgerHeader{sequence: uint32(i)})
		metaChan <- metaResult{
			LedgerCloseMeta: &meta,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(65), uint32(66)).Return(nil)
	mockRunner.On("getMetaPipe").Return((<-chan metaResult)(metaChan))
	mockRunner.On("context").Return(ctx)

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive: mockArchive,
		stellarCoreRunnerFactory: func(_ stellarCoreRunnerMode) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
		checkpointManager: historyarchive.NewCheckpointManager(64),
	}

	// requires PrepareRange
	_, _, err := captiveBackend.GetLedger(64)
	tt.EqualError(err, "session is closed, call PrepareRange first")

	err = captiveBackend.PrepareRange(BoundedRange(65, 66))
	assert.NoError(t, err)

	_, _, err = captiveBackend.GetLedger(64)
	tt.Error(err, "requested ledger 64 is behind the captive core stream (expected=66)")

	// reads value from buffer
	found, meta, err := captiveBackend.GetLedger(65)
	tt.NoError(err)
	tt.True(found)
	tt.Equal(xdr.Uint32(65), meta.V0.LedgerHeader.Header.LedgerSeq)

	// reads value from cachedMeta
	_, cachedMeta, err := captiveBackend.GetLedger(65)
	tt.NoError(err)
	tt.Equal(meta, cachedMeta)

	// next sequence number didn't get consumed
	tt.Equal(uint32(66), captiveBackend.nextLedger)

	mockRunner.On("close").Return(nil).Run(func(args mock.Arguments) {
		cancel()
	}).Once()

	_, _, err = captiveBackend.GetLedger(66)
	tt.NoError(err)

	// closes after last ledger is consumed
	tt.True(captiveBackend.isClosed())

	// we should be able to call last ledger even after get ledger is closed
	_, _, err = captiveBackend.GetLedger(66)
	tt.NoError(err)

	mockArchive.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
}

// TestCaptiveGetLedgerCacheLatestLedger test the following case:
// 1. Prepare Unbounded range.
// 2. GetLedger that is still not in the buffer.
// 3. Get latest ledger in the buffer using GetLedger.
//
// Before 3d97762 this test failed because cachedMeta was only updated when
// the ledger with a requested sequence was reached while streaming meta.
func TestCaptiveGetLedgerCacheLatestLedger(t *testing.T) {
	tt := assert.New(t)
	metaChan := make(chan metaResult, 300)

	for i := 2; i <= 67; i++ {
		meta := buildLedgerCloseMeta(testLedgerHeader{sequence: uint32(i)})
		metaChan <- metaResult{
			LedgerCloseMeta: &meta,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("runFrom", uint32(62), "0101010100000000000000000000000000000000000000000000000000000000").Return(nil).Once()
	mockRunner.On("getMetaPipe").Return((<-chan metaResult)(metaChan))
	mockRunner.On("context").Return(ctx)

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)

	mockArchive.
		On("GetLedgerHeader", uint32(63)).
		Return(xdr.LedgerHeaderHistoryEntry{
			Header: xdr.LedgerHeader{
				PreviousLedgerHash: xdr.Hash{1, 1, 1, 1},
			},
		}, nil).Once()

	captiveBackend := CaptiveStellarCore{
		archive: mockArchive,
		stellarCoreRunnerFactory: func(_ stellarCoreRunnerMode) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
		checkpointManager: historyarchive.NewCheckpointManager(64),
	}

	err := captiveBackend.PrepareRange(UnboundedRange(66))
	assert.NoError(t, err)

	found, _, err := captiveBackend.GetLedger(68)
	tt.NoError(err)
	tt.False(found)
	tt.Equal(uint32(67), captiveBackend.cachedMeta.LedgerSequence())
	tt.Equal(uint32(68), captiveBackend.nextLedger)

	found, meta, err := captiveBackend.GetLedger(67)
	tt.NoError(err)
	tt.True(found)
	tt.Equal(uint32(67), meta.LedgerSequence())

	mockArchive.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
}

func TestCaptiveGetLedger_NextLedgerIsDifferentToLedgerFromBuffer(t *testing.T) {
	metaChan := make(chan metaResult, 100)

	for i := 64; i <= 65; i++ {
		meta := buildLedgerCloseMeta(testLedgerHeader{sequence: uint32(i)})
		metaChan <- metaResult{
			LedgerCloseMeta: &meta,
		}
	}
	{
		meta := buildLedgerCloseMeta(testLedgerHeader{sequence: uint32(68)})
		metaChan <- metaResult{
			LedgerCloseMeta: &meta,
		}
	}

	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(65), uint32(66)).Return(nil)
	mockRunner.On("getMetaPipe").Return((<-chan metaResult)(metaChan))
	mockRunner.On("context").Return(context.Background())
	mockRunner.On("close").Return(nil)

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive: mockArchive,
		stellarCoreRunnerFactory: func(_ stellarCoreRunnerMode) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
		checkpointManager: historyarchive.NewCheckpointManager(64),
	}

	err := captiveBackend.PrepareRange(BoundedRange(65, 66))
	assert.NoError(t, err)

	_, _, err = captiveBackend.GetLedger(66)
	assert.EqualError(t, err, "unexpected ledger sequence (expected=66 actual=68)")

	mockArchive.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
}

func TestCaptiveStellarCore_PrepareRangeAfterClose(t *testing.T) {
	executablePath := "/etc/stellar-core"
	networkPassphrase := network.PublicNetworkPassphrase
	historyURLs := []string{"http://localhost"}

	captiveStellarCore, err := NewCaptive(
		CaptiveCoreConfig{
			BinaryPath:         executablePath,
			NetworkPassphrase:  networkPassphrase,
			HistoryArchiveURLs: historyURLs,
		},
	)
	assert.NoError(t, err)

	assert.NoError(t, captiveStellarCore.Close())

	assert.EqualError(
		t,
		captiveStellarCore.PrepareRange(BoundedRange(65, 66)),
		"error starting prepare range: opening subprocess: error getting latest checkpoint sequence: "+
			"error getting root HAS: Get \"http://localhost/.well-known/stellar-history.json\": context canceled",
	)

	// even if the request to fetch the latest checkpoint succeeds, we should fail at creating the subprocess
	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)
	captiveStellarCore.archive = mockArchive
	assert.EqualError(
		t,
		captiveStellarCore.PrepareRange(BoundedRange(65, 66)),
		"error starting prepare range: opening subprocess: error running stellar-core: context canceled",
	)
	mockArchive.AssertExpectations(t)
}

func TestCaptiveGetLedger_ErrReadingMetaResult(t *testing.T) {
	tt := assert.New(t)
	metaChan := make(chan metaResult, 100)

	for i := 64; i <= 65; i++ {
		meta := buildLedgerCloseMeta(testLedgerHeader{sequence: uint32(i)})
		metaChan <- metaResult{
			LedgerCloseMeta: &meta,
		}
	}
	metaChan <- metaResult{
		err: fmt.Errorf("unmarshalling error"),
	}

	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(65), uint32(66)).Return(nil)
	mockRunner.On("getMetaPipe").Return((<-chan metaResult)(metaChan))
	ctx, cancel := context.WithCancel(context.Background())
	mockRunner.On("context").Return(ctx)
	mockRunner.On("getProcessExitError").Return(false, nil)
	mockRunner.On("close").Return(nil).Run(func(args mock.Arguments) {
		cancel()
	}).Once()

	// even if the request to fetch the latest checkpoint succeeds, we should fail at creating the subprocess
	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive: mockArchive,
		stellarCoreRunnerFactory: func(_ stellarCoreRunnerMode) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
		checkpointManager: historyarchive.NewCheckpointManager(64),
	}

	err := captiveBackend.PrepareRange(BoundedRange(65, 66))
	assert.NoError(t, err)

	found, meta, err := captiveBackend.GetLedger(65)
	tt.NoError(err)
	tt.True(found)
	tt.Equal(xdr.Uint32(65), meta.V0.LedgerHeader.Header.LedgerSeq)

	// try reading from an empty buffer
	_, _, err = captiveBackend.GetLedger(66)
	tt.EqualError(err, "unmarshalling error")

	// closes if there is an error getting ledger
	tt.True(captiveBackend.isClosed())

	mockArchive.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
}

func TestCaptiveGetLedger_ErrClosingAfterLastLedger(t *testing.T) {
	tt := assert.New(t)
	metaChan := make(chan metaResult, 100)

	for i := 64; i <= 66; i++ {
		meta := buildLedgerCloseMeta(testLedgerHeader{sequence: uint32(i)})
		metaChan <- metaResult{
			LedgerCloseMeta: &meta,
		}
	}

	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(65), uint32(66)).Return(nil)
	mockRunner.On("getMetaPipe").Return((<-chan metaResult)(metaChan))
	mockRunner.On("context").Return(context.Background())
	mockRunner.On("close").Return(fmt.Errorf("transient error")).Once()

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive: mockArchive,
		stellarCoreRunnerFactory: func(_ stellarCoreRunnerMode) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
		checkpointManager: historyarchive.NewCheckpointManager(64),
	}

	err := captiveBackend.PrepareRange(BoundedRange(65, 66))
	assert.NoError(t, err)

	_, _, err = captiveBackend.GetLedger(66)
	tt.EqualError(err, "error closing session: transient error")

	mockArchive.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
}

func TestCaptiveAfterClose(t *testing.T) {
	metaChan := make(chan metaResult, 100)

	for i := 64; i <= 66; i++ {
		meta := buildLedgerCloseMeta(testLedgerHeader{sequence: uint32(i)})
		metaChan <- metaResult{
			LedgerCloseMeta: &meta,
		}
	}

	mockRunner := &stellarCoreRunnerMock{}
	ctx, cancel := context.WithCancel(context.Background())
	mockRunner.On("catchup", uint32(65), uint32(66)).Return(nil)
	mockRunner.On("getMetaPipe").Return((<-chan metaResult)(metaChan))
	mockRunner.On("context").Return(ctx)
	mockRunner.On("close").Return(nil).Once()

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive: mockArchive,
		stellarCoreRunnerFactory: func(_ stellarCoreRunnerMode) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
		checkpointManager: historyarchive.NewCheckpointManager(64),
		cancel:            cancel,
	}

	boundedRange := BoundedRange(65, 66)
	err := captiveBackend.PrepareRange(boundedRange)
	assert.NoError(t, err)

	assert.NoError(t, captiveBackend.Close())

	_, _, err = captiveBackend.GetLedger(boundedRange.to)
	assert.EqualError(t, err, "session is closed, call PrepareRange first")

	var prepared bool
	prepared, err = captiveBackend.IsPrepared(boundedRange)
	assert.False(t, prepared)
	assert.NoError(t, err)

	_, err = captiveBackend.GetLatestLedgerSequence()
	assert.EqualError(t, err, "stellar-core must be opened to return latest available sequence")

	mockArchive.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
}

func TestGetLedgerBoundsCheck(t *testing.T) {
	metaChan := make(chan metaResult, 100)

	for i := 128; i <= 130; i++ {
		meta := buildLedgerCloseMeta(testLedgerHeader{sequence: uint32(i)})
		metaChan <- metaResult{
			LedgerCloseMeta: &meta,
		}
	}

	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("catchup", uint32(128), uint32(130)).Return(nil).Once()
	mockRunner.On("getMetaPipe").Return((<-chan metaResult)(metaChan))
	mockRunner.On("context").Return(context.Background())

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(200),
		}, nil)

	captiveBackend := CaptiveStellarCore{
		archive: mockArchive,
		stellarCoreRunnerFactory: func(_ stellarCoreRunnerMode) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
		checkpointManager: historyarchive.NewCheckpointManager(64),
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

	mockArchive.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
}

func TestCaptiveGetLedgerTerminatedUnexpectedly(t *testing.T) {
	ledger64 := buildLedgerCloseMeta(testLedgerHeader{sequence: uint32(64)})

	for _, testCase := range []struct {
		name               string
		ctx                context.Context
		ledgers            []metaResult
		processExited      bool
		processExitedError error
		expectedError      string
	}{
		{
			"stellar core exited unexpectedly without error",
			context.Background(),
			[]metaResult{{LedgerCloseMeta: &ledger64}, {err: fmt.Errorf("transient error")}},
			true,
			nil,
			"stellar core exited unexpectedly",
		},
		{
			"stellar core exited unexpectedly with an error",
			context.Background(),
			[]metaResult{{LedgerCloseMeta: &ledger64}, {err: fmt.Errorf("transient error")}},
			true,
			fmt.Errorf("signal kill"),
			"stellar core exited unexpectedly: signal kill",
		},
		{
			"stellar core exited unexpectedly without error and closed channel",
			context.Background(),
			[]metaResult{{LedgerCloseMeta: &ledger64}},
			true,
			nil,
			"stellar core exited unexpectedly",
		},
		{
			"stellar core exited unexpectedly with an error and closed channel",
			context.Background(),
			[]metaResult{{LedgerCloseMeta: &ledger64}},
			true,
			fmt.Errorf("signal kill"),
			"stellar core exited unexpectedly: signal kill",
		},
		{
			"meta pipe closed unexpectedly",
			context.Background(),
			[]metaResult{{LedgerCloseMeta: &ledger64}},
			false,
			nil,
			"meta pipe closed unexpectedly",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			metaChan := make(chan metaResult, 100)

			for _, result := range testCase.ledgers {
				metaChan <- result
			}
			close(metaChan)

			mockRunner := &stellarCoreRunnerMock{}
			mockRunner.On("catchup", uint32(64), uint32(100)).Return(nil).Once()
			mockRunner.On("getMetaPipe").Return((<-chan metaResult)(metaChan))
			mockRunner.On("context").Return(context.Background())
			mockRunner.On("getProcessExitError").Return(testCase.processExited, testCase.processExitedError)
			mockRunner.On("close").Return(nil).Once()

			mockArchive := &historyarchive.MockArchive{}
			mockArchive.
				On("GetRootHAS").
				Return(historyarchive.HistoryArchiveState{
					CurrentLedger: uint32(200),
				}, nil)

			captiveBackend := CaptiveStellarCore{
				archive: mockArchive,
				stellarCoreRunnerFactory: func(_ stellarCoreRunnerMode) (stellarCoreRunnerInterface, error) {
					return mockRunner, nil
				},
				checkpointManager: historyarchive.NewCheckpointManager(64),
			}

			err := captiveBackend.PrepareRange(BoundedRange(64, 100))
			assert.NoError(t, err)

			exists, meta, err := captiveBackend.GetLedger(64)
			assert.NoError(t, err)
			assert.True(t, exists)
			assert.Equal(t, uint32(64), meta.LedgerSequence())

			_, _, err = captiveBackend.GetLedger(65)
			assert.EqualError(t, err, testCase.expectedError)

			mockArchive.AssertExpectations(t)
			mockRunner.AssertExpectations(t)
		})
	}
}

func TestCaptiveUseOfLedgerHashStore(t *testing.T) {
	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetLedgerHeader", uint32(255)).
		Return(xdr.LedgerHeaderHistoryEntry{
			Header: xdr.LedgerHeader{
				PreviousLedgerHash: xdr.Hash{1, 1, 1, 1},
			},
		}, nil)

	mockLedgerHashStore := &MockLedgerHashStore{}
	mockLedgerHashStore.On("GetLedgerHash", uint32(1022)).
		Return("", false, fmt.Errorf("transient error")).Once()
	mockLedgerHashStore.On("GetLedgerHash", uint32(254)).
		Return("", false, nil).Once()
	mockLedgerHashStore.On("GetLedgerHash", uint32(62)).
		Return("cde", true, nil).Once()
	mockLedgerHashStore.On("GetLedgerHash", uint32(126)).
		Return("ghi", true, nil).Once()
	mockLedgerHashStore.On("GetLedgerHash", uint32(2)).
		Return("mnb", true, nil).Once()

	captiveBackend := CaptiveStellarCore{
		archive:           mockArchive,
		ledgerHashStore:   mockLedgerHashStore,
		checkpointManager: historyarchive.NewCheckpointManager(64),
	}

	runFrom, ledgerHash, nextLedger, err := captiveBackend.runFromParams(24)
	assert.NoError(t, err)
	assert.Equal(t, uint32(2), runFrom)
	assert.Equal(t, "mnb", ledgerHash)
	assert.Equal(t, uint32(2), nextLedger)

	runFrom, ledgerHash, nextLedger, err = captiveBackend.runFromParams(86)
	assert.NoError(t, err)
	assert.Equal(t, uint32(62), runFrom)
	assert.Equal(t, "cde", ledgerHash)
	assert.Equal(t, uint32(2), nextLedger)

	runFrom, ledgerHash, nextLedger, err = captiveBackend.runFromParams(128)
	assert.NoError(t, err)
	assert.Equal(t, uint32(126), runFrom)
	assert.Equal(t, "ghi", ledgerHash)
	assert.Equal(t, uint32(64), nextLedger)

	runFrom, ledgerHash, nextLedger, err = captiveBackend.runFromParams(1050)
	assert.EqualError(t, err, "error trying to read ledger hash 1022: transient error")

	runFrom, ledgerHash, nextLedger, err = captiveBackend.runFromParams(300)
	assert.NoError(t, err)
	assert.Equal(t, uint32(254), runFrom, "runFrom")
	assert.Equal(t, "0101010100000000000000000000000000000000000000000000000000000000", ledgerHash)
	assert.Equal(t, uint32(192), nextLedger, "nextLedger")

	mockLedgerHashStore.AssertExpectations(t)
	mockArchive.AssertExpectations(t)
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
				checkpointManager: historyarchive.NewCheckpointManager(64),
			}

			runFrom, ledgerHash, nextLedger, err := captiveBackend.runFromParams(tc.from)
			tt.NoError(err)
			tt.Equal(tc.runFrom, runFrom, "runFrom")
			tt.Equal("0101010100000000000000000000000000000000000000000000000000000000", ledgerHash)
			tt.Equal(tc.nextLedger, nextLedger, "nextLedger")

			mockArchive.AssertExpectations(t)
		})
	}
}

func TestCaptiveIsPrepared(t *testing.T) {
	var tests = []struct {
		nextLedger   uint32
		lastLedger   uint32
		cachedLedger uint32
		ledgerRange  Range
		result       bool
	}{
		{0, 0, 0, UnboundedRange(100), false},
		{100, 0, 0, UnboundedRange(101), true},
		{101, 0, 100, UnboundedRange(100), true},
		{100, 200, 0, UnboundedRange(100), false},

		{100, 200, 0, BoundedRange(100, 200), true},
		{100, 200, 0, BoundedRange(100, 201), false},
		{100, 201, 0, BoundedRange(100, 200), true},
		{101, 200, 100, BoundedRange(100, 200), true},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("next_%d_last_%d_cached_%d_range_%v", tc.nextLedger, tc.lastLedger, tc.cachedLedger, tc.ledgerRange), func(t *testing.T) {
			mockRunner := &stellarCoreRunnerMock{}
			mockRunner.On("context").Return(context.Background()).Maybe()

			captiveBackend := CaptiveStellarCore{
				stellarCoreRunner: mockRunner,
				nextLedger:        tc.nextLedger,
			}
			if tc.lastLedger > 0 {
				captiveBackend.lastLedger = &tc.lastLedger
			}
			if tc.cachedLedger > 0 {
				meta := buildLedgerCloseMeta(testLedgerHeader{
					sequence: tc.cachedLedger,
				})
				captiveBackend.cachedMeta = &meta
			}

			result := captiveBackend.isPrepared(tc.ledgerRange)
			assert.Equal(t, tc.result, result)
		})
	}
}

// TestCaptivePreviousLedgerCheck checks if previousLedgerHash is set in PrepareRange
// and then checked and updated in GetLedger.
func TestCaptivePreviousLedgerCheck(t *testing.T) {
	metaChan := make(chan metaResult, 200)

	h := 3
	for i := 192; i <= 300; i++ {
		meta := buildLedgerCloseMeta(testLedgerHeader{
			sequence:           uint32(i),
			hash:               fmt.Sprintf("%02x00000000000000000000000000000000000000000000000000000000000000", h),
			previousLedgerHash: fmt.Sprintf("%02x00000000000000000000000000000000000000000000000000000000000000", h-1),
		})
		metaChan <- metaResult{
			LedgerCloseMeta: &meta,
		}
		h++
	}

	{
		// Write invalid hash
		meta := buildLedgerCloseMeta(testLedgerHeader{
			sequence:           301,
			hash:               "0000000000000000000000000000000000000000000000000000000000000000",
			previousLedgerHash: "0000000000000000000000000000000000000000000000000000000000000000",
		})
		metaChan <- metaResult{
			LedgerCloseMeta: &meta,
		}

	}

	mockRunner := &stellarCoreRunnerMock{}
	mockRunner.On("runFrom", uint32(254), "0101010100000000000000000000000000000000000000000000000000000000").Return(nil).Once()
	mockRunner.On("getMetaPipe").Return((<-chan metaResult)(metaChan))
	mockRunner.On("context").Return(context.Background())
	mockRunner.On("close").Return(nil).Once()

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: uint32(255),
		}, nil)
	mockArchive.
		On("GetLedgerHeader", uint32(255)).
		Return(xdr.LedgerHeaderHistoryEntry{
			Header: xdr.LedgerHeader{
				PreviousLedgerHash: xdr.Hash{1, 1, 1, 1},
			},
		}, nil).Once()

	mockLedgerHashStore := &MockLedgerHashStore{}
	mockLedgerHashStore.On("GetLedgerHash", uint32(254)).
		Return("", false, nil).Once()
	mockLedgerHashStore.On("GetLedgerHash", uint32(191)).
		Return("0200000000000000000000000000000000000000000000000000000000000000", true, nil).Once()

	captiveBackend := CaptiveStellarCore{
		archive: mockArchive,
		stellarCoreRunnerFactory: func(_ stellarCoreRunnerMode) (stellarCoreRunnerInterface, error) {
			return mockRunner, nil
		},
		ledgerHashStore:   mockLedgerHashStore,
		checkpointManager: historyarchive.NewCheckpointManager(64),
	}

	err := captiveBackend.PrepareRange(UnboundedRange(300))
	assert.NoError(t, err)

	exists, meta, err := captiveBackend.GetLedger(300)
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.NotNil(t, captiveBackend.previousLedgerHash)
	assert.Equal(t, uint32(301), captiveBackend.nextLedger)
	assert.Equal(t, meta.LedgerHash().HexString(), *captiveBackend.previousLedgerHash)

	_, _, err = captiveBackend.GetLedger(301)
	assert.EqualError(t, err, "unexpected previous ledger hash for ledger 301 (expected=6f00000000000000000000000000000000000000000000000000000000000000 actual=0000000000000000000000000000000000000000000000000000000000000000)")

	mockRunner.AssertExpectations(t)
	mockArchive.AssertExpectations(t)
	mockLedgerHashStore.AssertExpectations(t)
}
