package ledgerbackend

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/stellar/go/support/compressxdr"
	"github.com/stellar/go/support/datastore"
	"github.com/stellar/go/xdr"
)

// Ensure GCSBackend implements LedgerBackend
var _ LedgerBackend = (*GCSBackend)(nil)

// default config values
var defaultFileSuffix = ".xdr.gz"
var defaultLedgersPerFile = uint32(1)
var defaultFilesPerPartition = uint32(64000)
var defaultBufferSize = uint32(1000)
var defaultNumWorkers = uint32(5)
var defaultRetryLimit = uint32(3)
var defaultRetryWait = time.Duration(5)

type ledgerBatchObject struct {
	payload     []byte
	startLedger int // Ledger sequence used as the priority for the priorityqueue.
}

type BufferConfig struct {
	BufferSize uint32
	NumWorkers uint32
	RetryLimit uint32
	RetryWait  time.Duration
}

type GCSBackendConfig struct {
	BufferConfig      BufferConfig
	LedgerBatchConfig datastore.LedgerBatchConfig
	CompressionType   string
	DataStore         datastore.DataStore
	ResumableManager  datastore.ResumableManager
}

// GCSBackend is a ledger backend that reads from a cloud storage service.
// The cloud storage service contains files generated from the ledgerExporter.
type GCSBackend struct {
	config GCSBackendConfig

	context context.Context
	// cancel is the CancelCauseFunc for context which controls the lifetime of a GCSBackend instance.
	// Once it is invoked GCSBackend will not be able to stream ledgers from GCSBackend.
	cancel context.CancelCauseFunc

	// gcsBackendLock protects access to gcsBackendRunner. When the read lock
	// is acquired gcsBackendRunner can be accessed. When the write lock is acquired
	// gcsBackendRunner can be updated.
	gcsBackendLock sync.RWMutex

	// ledgerBuffer is the buffer for LedgerCloseMeta data read in parallel.
	ledgerBuffer *ledgerBufferGCS

	dataStore         datastore.DataStore
	resumableManager  datastore.ResumableManager
	prepared          *Range // non-nil if any range is prepared
	closed            bool   // False until the core is closed
	ledgerMetaArchive *datastore.LedgerMetaArchive
	decoder           compressxdr.XDRDecoder
	nextLedger        uint32
}

// Return a new GCSBackend instance.
func NewGCSBackend(ctx context.Context, config GCSBackendConfig) (*GCSBackend, error) {
	// Check/set minimum config values for LedgerBatchConfig
	if config.LedgerBatchConfig.FileSuffix == "" {
		config.LedgerBatchConfig.FileSuffix = defaultFileSuffix
	}

	if config.LedgerBatchConfig.LedgersPerFile == 0 {
		config.LedgerBatchConfig.LedgersPerFile = defaultLedgersPerFile
	}

	if config.LedgerBatchConfig.FilesPerPartition == 0 {
		config.LedgerBatchConfig.FilesPerPartition = defaultFilesPerPartition
	}

	// Check/set minimum config values for BufferConfig
	if config.BufferConfig.BufferSize == 0 {
		config.BufferConfig.BufferSize = defaultBufferSize
	}

	if config.BufferConfig.NumWorkers == 0 {
		config.BufferConfig.NumWorkers = defaultNumWorkers
	}

	if config.BufferConfig.RetryLimit == 0 {
		config.BufferConfig.RetryLimit = defaultRetryLimit
	}

	if config.BufferConfig.RetryWait == 0 {
		config.BufferConfig.RetryWait = defaultRetryWait
	}

	ctx, cancel := context.WithCancelCause(ctx)

	if config.DataStore == nil {
		return nil, errors.New("no DataStore provided")
	}

	if config.ResumableManager == nil {
		return nil, errors.New("no ResumableManager provided")
	}

	ledgerMetaArchive := datastore.NewLedgerMetaArchive("", 0, 0)
	decoder, err := compressxdr.NewXDRDecoder(config.CompressionType, nil)
	if err != nil {
		return nil, err
	}

	gcsBackend := &GCSBackend{
		config:            config,
		context:           ctx,
		cancel:            cancel,
		dataStore:         config.DataStore,
		resumableManager:  config.ResumableManager,
		ledgerMetaArchive: ledgerMetaArchive,
		decoder:           decoder,
	}

	return gcsBackend, nil
}

// GetLatestLedgerSequence returns the most recent ledger sequence number in the cloud storage bucket.
func (gcsb *GCSBackend) GetLatestLedgerSequence(ctx context.Context) (uint32, error) {
	var err error

	if gcsb.closed {
		return 0, errors.New("gcsBackend is closed; cannot GetLatestLedgerSequence")
	}

	// Start at 2 to skip the genesis ledger
	absentLedger, ok, err := gcsb.resumableManager.FindStart(ctx, uint32(2), uint32(0))
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, errors.New("findStart returned sequence beyond latest history archive ledger")
	}

	// Subtract one to get the oldest existing ledger seq closest to genesis
	return absentLedger - 1, nil
}

// getSequenceInBatch checks if the requested sequence is in the cached batch.
// Otherwise will continuously load in the next LedgerCloseMetaBatch until found.
func (gcsb *GCSBackend) getSequenceInBatch(sequence uint32) error {
	for {
		// Sequence inside the current cached LedgerCloseMetaBatch
		if sequence >= gcsb.ledgerMetaArchive.GetStartLedgerSequence() && sequence <= gcsb.ledgerMetaArchive.GetEndLedgerSequence() {
			return nil
		}

		// Sequence is before the current LedgerCloseMetaBatch
		// Does not support retrieving LedgerCloseMeta before the current cached batch
		if sequence < gcsb.ledgerMetaArchive.GetStartLedgerSequence() {
			return errors.New("requested sequence preceeds current LedgerCloseMetaBatch")
		}

		// Sequence is beyond the current LedgerCloseMetaBatch
		lcmBatchBinary, err := gcsb.ledgerBuffer.getFromLedgerQueue()
		if err != nil {
			return errors.Wrap(err, "failed getting next ledger batch from queue")
		}

		// Turn binary into xdr
		err = gcsb.ledgerMetaArchive.Data.UnmarshalBinary(lcmBatchBinary)
		if err != nil {
			return errors.Wrap(err, "failed unmarshalling lcmBatchBinary")
		}
	}
}

// nextExpectedSequence returns nextLedger (if currently set) or start of
// prepared range. Otherwise it returns 0.
// This is done because `nextLedger` is 0 between the moment Stellar-Core is
// started and streaming the first ledger (in such case we return first ledger
// in requested range).
func (gcsb *GCSBackend) nextExpectedSequence() uint32 {
	if gcsb.nextLedger == 0 && gcsb.prepared != nil {
		return gcsb.prepared.from
	}
	return gcsb.nextLedger
}

// GetLedger returns the LedgerCloseMeta for the specified ledger sequence number
func (gcsb *GCSBackend) GetLedger(ctx context.Context, sequence uint32) (xdr.LedgerCloseMeta, error) {
	gcsb.gcsBackendLock.RLock()
	defer gcsb.gcsBackendLock.RUnlock()

	if gcsb.closed {
		return xdr.LedgerCloseMeta{}, errors.New("gcsBackend is closed; cannot GetLedger")
	}

	if gcsb.prepared == nil {
		return xdr.LedgerCloseMeta{}, errors.New("session is not prepared, call PrepareRange first")
	}

	if sequence < gcsb.ledgerBuffer.ledgerRange.from {
		return xdr.LedgerCloseMeta{}, errors.New("requested sequence preceeds current LedgerRange")
	}

	if gcsb.ledgerBuffer.ledgerRange.bounded {
		if sequence > gcsb.ledgerBuffer.ledgerRange.to {
			return xdr.LedgerCloseMeta{}, errors.New("requested sequence beyond current LedgerRange")
		}
	}

	if gcsb.nextExpectedSequence() != sequence {
		return xdr.LedgerCloseMeta{}, errors.New("requested sequence is not the next available ledger")
	}

	err := gcsb.getSequenceInBatch(sequence)
	if err != nil {
		return xdr.LedgerCloseMeta{}, err
	}

	ledgerCloseMeta, err := gcsb.ledgerMetaArchive.GetLedger(sequence)
	if err != nil {
		return xdr.LedgerCloseMeta{}, err
	}
	gcsb.nextLedger++

	return ledgerCloseMeta, nil
}

// PrepareRange checks if the starting and ending (if bounded) ledgers exist.
func (gcsb *GCSBackend) PrepareRange(ctx context.Context, ledgerRange Range) error {
	if gcsb.closed {
		return errors.New("gcsBackend is closed; cannot PrepareRange")
	}

	if alreadyPrepared, err := gcsb.startPreparingRange(ledgerRange); err != nil {
		return errors.Wrap(err, "error starting prepare range")
	} else if alreadyPrepared {
		return nil
	}

	gcsb.prepared = &ledgerRange

	return nil
}

// IsPrepared returns true if a given ledgerRange is prepared.
func (gcsb *GCSBackend) IsPrepared(ctx context.Context, ledgerRange Range) (bool, error) {
	gcsb.gcsBackendLock.RLock()
	defer gcsb.gcsBackendLock.RUnlock()

	if gcsb.closed {
		return false, errors.New("gcsBackend is closed; cannot IsPrepared")
	}

	return gcsb.isPrepared(ledgerRange), nil
}

func (gcsb *GCSBackend) isPrepared(ledgerRange Range) bool {
	if gcsb.closed {
		return false
	}

	if gcsb.prepared == nil {
		return false
	}

	if gcsb.ledgerBuffer.ledgerRange.from > ledgerRange.from {
		return false
	}

	if gcsb.ledgerBuffer.ledgerRange.bounded && !ledgerRange.bounded {
		return false
	}

	if !gcsb.ledgerBuffer.ledgerRange.bounded && !ledgerRange.bounded {
		return true
	}

	if !gcsb.ledgerBuffer.ledgerRange.bounded && ledgerRange.bounded {
		return true
	}

	if gcsb.ledgerBuffer.ledgerRange.to >= ledgerRange.to {
		return true
	}

	return false
}

// Close closes existing GCSBackend processes.
// Note, once a GCSBackend instance is closed it can no longer be used and
// all subsequent calls to PrepareRange(), GetLedger(), etc will fail.
// Close is thread-safe and can be called from another go routine.
func (gcsb *GCSBackend) Close() error {
	gcsb.gcsBackendLock.RLock()
	defer gcsb.gcsBackendLock.RUnlock()

	gcsb.closed = true

	// after the GCSBackend context is Done all subsequent calls to PrepareRange() will fail
	gcsb.context.Done()

	return nil
}

// startPreparingRange prepares the ledger range by setting the range in the ledgerBuffer
func (gcsb *GCSBackend) startPreparingRange(ledgerRange Range) (bool, error) {
	gcsb.gcsBackendLock.Lock()
	defer gcsb.gcsBackendLock.Unlock()

	if gcsb.isPrepared(ledgerRange) {
		return true, nil
	}

	var err error
	gcsb.ledgerBuffer, err = gcsb.newLedgerBuffer(ledgerRange)
	if err != nil {
		return false, err
	}

	// Start the ledgerBuffer
	gcsb.ledgerBuffer.pushTaskQueue()
	gcsb.nextLedger = ledgerRange.from

	return false, nil
}
