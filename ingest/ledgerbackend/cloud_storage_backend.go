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

// Ensure CloudStorageBackend implements LedgerBackend
var _ LedgerBackend = (*CloudStorageBackend)(nil)

type CloudStorageBackendConfig struct {
	LedgerBatchConfig datastore.LedgerBatchConfig
	CompressionType   string
	DataStore         datastore.DataStore
	ResumableManager  datastore.ResumableManager
	BufferSize        uint32
	NumWorkers        uint32
	RetryLimit        uint32
	RetryWait         time.Duration
}

// CloudStorageBackend is a ledger backend that reads from a cloud storage service.
// The cloud storage service contains files generated from the ledgerExporter.
type CloudStorageBackend struct {
	config CloudStorageBackendConfig

	context context.Context
	// cancel is the CancelCauseFunc for context which controls the lifetime of a CloudStorageBackend instance.
	// Once it is invoked CloudStorageBackend will not be able to stream ledgers from CloudStorageBackend.
	cancel        context.CancelCauseFunc
	csBackendLock sync.RWMutex

	// ledgerBuffer is the buffer for LedgerCloseMeta data read in parallel.
	ledgerBuffer *ledgerBuffer

	dataStore         datastore.DataStore
	prepared          *Range // non-nil if any range is prepared
	closed            bool   // False until the core is closed
	ledgerMetaArchive *datastore.LedgerMetaArchive
	decoder           compressxdr.XDRDecoder
	nextLedger        uint32
	lastLedger        uint32
}

// Return a new CloudStorageBackend instance.
func NewCloudStorageBackend(ctx context.Context, config CloudStorageBackendConfig) (*CloudStorageBackend, error) {
	if config.BufferSize == 0 {
		return nil, errors.New("buffer size must be > 0")
	}

	if config.NumWorkers > config.BufferSize {
		return nil, errors.New("number of workers must be <= BufferSize")
	}

	if config.DataStore == nil {
		return nil, errors.New("no DataStore provided")
	}

	if config.ResumableManager == nil {
		return nil, errors.New("no ResumableManager provided")
	}

	if config.LedgerBatchConfig.LedgersPerFile <= 0 {
		return nil, errors.New("ledgersPerFile must be > 0")
	}

	if config.LedgerBatchConfig.FileSuffix == "" {
		return nil, errors.New("no file suffix provided in LedgerBatchConfig")
	}

	if config.CompressionType == "" {
		return nil, errors.New("no compression type provided in config")
	}

	ctx, cancel := context.WithCancelCause(ctx)

	ledgerMetaArchive := datastore.NewLedgerMetaArchive("", 0, 0)
	decoder, err := compressxdr.NewXDRDecoder(config.CompressionType, nil)
	if err != nil {
		return nil, err
	}

	csBackend := &CloudStorageBackend{
		config:            config,
		context:           ctx,
		cancel:            cancel,
		dataStore:         config.DataStore,
		ledgerMetaArchive: ledgerMetaArchive,
		decoder:           decoder,
	}

	return csBackend, nil
}

// GetLatestLedgerSequence returns the most recent ledger sequence number in the cloud storage bucket.
func (csb *CloudStorageBackend) GetLatestLedgerSequence(ctx context.Context) (uint32, error) {
	csb.csBackendLock.RLock()
	defer csb.csBackendLock.RUnlock()

	if csb.closed {
		return 0, errors.New("CloudStorageBackend is closed; cannot GetLatestLedgerSequence")
	}

	if csb.prepared == nil {
		return 0, errors.New("CloudStorageBackend must be prepared, call PrepareRange first")
	}

	latestSeq, err := csb.ledgerBuffer.getLatestLedgerSequence()
	if err != nil {
		return 0, err
	}

	return latestSeq, nil
}

// getBatchForSequence checks if the requested sequence is in the cached batch.
// Otherwise will continuously load in the next LedgerCloseMetaBatch until found.
func (csb *CloudStorageBackend) getBatchForSequence(sequence uint32) error {
	for {
		// Sequence inside the current cached LedgerCloseMetaBatch
		if sequence >= csb.ledgerMetaArchive.GetStartLedgerSequence() && sequence <= csb.ledgerMetaArchive.GetEndLedgerSequence() {
			return nil
		}

		// Sequence is before the current LedgerCloseMetaBatch
		// Does not support retrieving LedgerCloseMeta before the current cached batch
		if sequence < csb.ledgerMetaArchive.GetStartLedgerSequence() {
			return errors.New("requested sequence preceeds current LedgerCloseMetaBatch")
		}

		// Sequence is beyond the current LedgerCloseMetaBatch
		lcmBatchBinary, err := csb.ledgerBuffer.getFromLedgerQueue()
		if err != nil {
			return errors.Wrap(err, "failed getting next ledger batch from queue")
		}

		// Turn binary into xdr
		err = csb.ledgerMetaArchive.Data.UnmarshalBinary(lcmBatchBinary)
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
func (csb *CloudStorageBackend) nextExpectedSequence() uint32 {
	if csb.nextLedger == 0 && csb.prepared != nil {
		return csb.prepared.from
	}
	return csb.nextLedger
}

// GetLedger returns the LedgerCloseMeta for the specified ledger sequence number
func (csb *CloudStorageBackend) GetLedger(ctx context.Context, sequence uint32) (xdr.LedgerCloseMeta, error) {
	csb.csBackendLock.RLock()
	defer csb.csBackendLock.RUnlock()

	if csb.closed {
		return xdr.LedgerCloseMeta{}, errors.New("CloudStorageBackend is closed; cannot GetLedger")
	}

	if csb.prepared == nil {
		return xdr.LedgerCloseMeta{}, errors.New("session is not prepared, call PrepareRange first")
	}

	if sequence < csb.ledgerBuffer.ledgerRange.from {
		return xdr.LedgerCloseMeta{}, errors.New("requested sequence preceeds current LedgerRange")
	}

	if csb.ledgerBuffer.ledgerRange.bounded {
		if sequence > csb.ledgerBuffer.ledgerRange.to {
			return xdr.LedgerCloseMeta{}, errors.New("requested sequence beyond current LedgerRange")
		}
	}

	if sequence > csb.nextExpectedSequence() {
		return xdr.LedgerCloseMeta{}, errors.New("requested sequence is not the lastLedger nor the next available ledger")
	}

	err := csb.getBatchForSequence(sequence)
	if err != nil {
		return xdr.LedgerCloseMeta{}, err
	}

	ledgerCloseMeta, err := csb.ledgerMetaArchive.GetLedger(sequence)
	if err != nil {
		return xdr.LedgerCloseMeta{}, err
	}
	csb.lastLedger = csb.nextLedger
	csb.nextLedger++

	return ledgerCloseMeta, nil
}

// PrepareRange checks if the starting and ending (if bounded) ledgers exist.
func (csb *CloudStorageBackend) PrepareRange(ctx context.Context, ledgerRange Range) error {
	csb.csBackendLock.Lock()
	defer csb.csBackendLock.Unlock()

	if csb.closed {
		return errors.New("CloudStorageBackend is closed; cannot PrepareRange")
	}

	if alreadyPrepared, err := csb.startPreparingRange(ledgerRange); err != nil {
		return errors.Wrap(err, "error starting prepare range")
	} else if alreadyPrepared {
		return nil
	}

	csb.prepared = &ledgerRange

	return nil
}

// IsPrepared returns true if a given ledgerRange is prepared.
func (csb *CloudStorageBackend) IsPrepared(ctx context.Context, ledgerRange Range) (bool, error) {
	csb.csBackendLock.RLock()
	defer csb.csBackendLock.RUnlock()

	if csb.closed {
		return false, errors.New("CloudStorageBackend is closed; cannot IsPrepared")
	}

	return csb.isPrepared(ledgerRange), nil
}

func (csb *CloudStorageBackend) isPrepared(ledgerRange Range) bool {
	if csb.closed {
		return false
	}

	if csb.prepared == nil {
		return false
	}

	if csb.ledgerBuffer.ledgerRange.from > ledgerRange.from {
		return false
	}

	if csb.ledgerBuffer.ledgerRange.bounded && !ledgerRange.bounded {
		return false
	}

	if !csb.ledgerBuffer.ledgerRange.bounded && !ledgerRange.bounded {
		return true
	}

	if !csb.ledgerBuffer.ledgerRange.bounded && ledgerRange.bounded {
		return true
	}

	if csb.ledgerBuffer.ledgerRange.to >= ledgerRange.to {
		return true
	}

	return false
}

// Close closes existing CloudStorageBackend processes.
// Note, once a CloudStorageBackend instance is closed it can no longer be used and
// all subsequent calls to PrepareRange(), GetLedger(), etc will fail.
// Close is thread-safe and can be called from another go routine.
func (csb *CloudStorageBackend) Close() error {
	csb.csBackendLock.RLock()
	defer csb.csBackendLock.RUnlock()

	csb.closed = true

	// after the CloudStorageBackend context is Done all subsequent calls to PrepareRange() will fail
	csb.context.Done()

	return nil
}

// startPreparingRange prepares the ledger range by setting the range in the ledgerBuffer
func (csb *CloudStorageBackend) startPreparingRange(ledgerRange Range) (bool, error) {
	if csb.isPrepared(ledgerRange) {
		return true, nil
	}

	var err error
	csb.ledgerBuffer, err = csb.newLedgerBuffer(ledgerRange)
	if err != nil {
		return false, err
	}

	csb.nextLedger = ledgerRange.from

	return false, nil
}
