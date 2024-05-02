package ledgerbackend

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/support/collections/heap"
	"github.com/stellar/go/support/compressxdr"
	"github.com/stellar/go/support/datastore"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"

	"google.golang.org/api/googleapi"
)

// Ensure GCSBackend implements LedgerBackend
var _ LedgerBackend = (*GCSBackend)(nil)

type LedgerBatchObject struct {
	Payload     []byte
	StartLedger int // Ledger sequence used as the priority for the priorityqueue.
}

type BufferConfig struct {
	BufferSize uint32
	NumWorkers uint32
	RetryLimit uint32
	RetryWait  time.Duration
}

type GCSBackendConfig struct {
	BufferConfig      BufferConfig
	DataStoreConfig   datastore.DataStoreConfig
	LedgerBatchConfig datastore.LedgerBatchConfig
	StorageUrl        string
	Network           string
	CompressionType   string
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
	prepared          *Range // non-nil if any range is prepared
	closed            bool   // False until the core is closed
	ledgerMetaArchive *datastore.LedgerMetaArchive
	decoder           compressxdr.XDRDecoder
}

type ledgerBufferGCS struct {
	config              GCSBackendConfig
	dataStore           datastore.DataStore
	taskQueue           chan uint32 // buffer next gcs object read
	ledgerQueue         chan []byte // order corrected lcm batches
	ledgerPriorityQueue *heap.Heap[LedgerBatchObject]
	priorityQueueLock   sync.Mutex
	count               uint32 // buffer count
	limit               uint32 // buffer max
	done                chan struct{}

	// keep track of the ledgers to be processed and the next ordering
	// the ledgers should be buffered
	currentLedger         uint32
	nextTaskLedger        uint32
	nextLedgerQueueLedger uint32
	ledgerRange           Range

	// passed through from GCSBackend to control lifetime of ledgerBufferGCS instance
	context context.Context
	cancel  context.CancelCauseFunc
	decoder compressxdr.XDRDecoder
}

func (gcsb *GCSBackend) NewLedgerBuffer(ledgerRange Range) (*ledgerBufferGCS, error) {
	less := func(a, b LedgerBatchObject) bool {
		return a.StartLedger < b.StartLedger
	}
	pq := heap.New(less, int(gcsb.config.BufferConfig.BufferSize))

	done := make(chan struct{})

	ledgerBuffer := &ledgerBufferGCS{
		config:                gcsb.config,
		dataStore:             gcsb.dataStore,
		taskQueue:             make(chan uint32, gcsb.config.BufferConfig.BufferSize),
		ledgerQueue:           make(chan []byte, gcsb.config.BufferConfig.BufferSize),
		ledgerPriorityQueue:   pq,
		count:                 0,
		limit:                 gcsb.config.BufferConfig.BufferSize,
		done:                  done,
		currentLedger:         ledgerRange.from,
		nextTaskLedger:        ledgerRange.from,
		nextLedgerQueueLedger: ledgerRange.from,
		ledgerRange:           ledgerRange,
		context:               gcsb.context,
		cancel:                gcsb.cancel,
		decoder:               gcsb.decoder,
	}

	// Workers to read LCM files
	for i := uint32(0); i < gcsb.config.BufferConfig.NumWorkers; i++ {
		go ledgerBuffer.worker()
	}

	return ledgerBuffer, nil
}

func (lb *ledgerBufferGCS) pushTaskQueue() {
	for lb.count <= lb.limit {
		// In bounded mode, don't queue past the end ledger
		if lb.ledgerRange.to < lb.nextTaskLedger && lb.ledgerRange.bounded {
			return
		}
		lb.taskQueue <- lb.nextTaskLedger
		lb.nextTaskLedger += lb.config.LedgerBatchConfig.LedgersPerFile
		lb.count++
	}
}

func (lb *ledgerBufferGCS) worker() {
	for {
		select {
		case <-lb.done:
			log.Error("abort: getFromLedgerQueue blocked")
			return
		case <-lb.context.Done():
			log.Error(lb.context.Err())
			return
		case sequence := <-lb.taskQueue:
			retryCount := uint32(0)
			for retryCount <= lb.config.BufferConfig.RetryLimit {
				ledgerObject, err := lb.getLedgerGCSObject(sequence)
				if err != nil {
					if e, ok := err.(*googleapi.Error); ok {
						// ledgerObject not found and unbounded
						if e.Code == 404 && !lb.ledgerRange.bounded {
							time.Sleep(lb.config.BufferConfig.RetryWait * time.Second)
							continue
						}
					}
					if retryCount == lb.config.BufferConfig.RetryLimit {
						err = errors.New("maximum retries exceeded for gcs object reads")
						lb.cancel(err)
					}
					retryCount++
					time.Sleep(lb.config.BufferConfig.RetryWait * time.Second)
				}

				// Add to priority queue and continue to next task
				lb.storeObject(ledgerObject, sequence)
				break
			}
		}
	}
}

func (lb *ledgerBufferGCS) getLedgerGCSObject(sequence uint32) ([]byte, error) {
	objectKey := lb.config.LedgerBatchConfig.GetObjectKeyFromSequenceNumber(sequence)

	reader, err := lb.dataStore.GetFile(context.Background(), objectKey)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting file: %s", objectKey)
	}

	defer reader.Close()

	objectBytes, err := lb.decoder.Unzip(reader)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unzipping file: %s", objectKey)
	}

	return objectBytes, nil
}

func (lb *ledgerBufferGCS) storeObject(ledgerObject []byte, sequence uint32) {
	lb.priorityQueueLock.Lock()
	defer lb.priorityQueueLock.Unlock()

	lb.ledgerPriorityQueue.Push(LedgerBatchObject{
		Payload:     ledgerObject,
		StartLedger: int(sequence),
	})

	// Check if the nextLedger is the next item in the priority queue
	for lb.ledgerPriorityQueue.Len() > 0 && lb.currentLedger == uint32(lb.ledgerPriorityQueue.Peek().StartLedger) {
		item := lb.ledgerPriorityQueue.Pop()
		lb.ledgerQueue <- item.Payload
		lb.nextLedgerQueueLedger++
	}
}

func (lb *ledgerBufferGCS) getFromLedgerQueue() ([]byte, error) {
	for {
		select {
		case <-lb.context.Done():
			log.Info("Stopping getFromLedgerQueue due to context cancellation")
			close(lb.done)
			return nil, lb.context.Err()
		case lcmBinary := <-lb.ledgerQueue:
			lb.currentLedger++
			// Decrement ledger buffer counter
			lb.count--
			// Add next task to the TaskQueue
			lb.pushTaskQueue()

			return lcmBinary, nil
		}
	}
}

// Return a new GCSBackend instance.
func NewGCSBackend(ctx context.Context, config GCSBackendConfig) (*GCSBackend, error) {
	// Check/set minimum config values
	if config.StorageUrl == "" {
		return nil, errors.New("storageURL is not set")
	}

	if config.LedgerBatchConfig.FileSuffix == "" {
		return nil, errors.New("ledgerBatchConfig.FileSuffix is not set")
	}

	if config.LedgerBatchConfig.LedgersPerFile == 0 {
		config.LedgerBatchConfig.LedgersPerFile = 1
	}

	if config.LedgerBatchConfig.FilesPerPartition == 0 {
		config.LedgerBatchConfig.FilesPerPartition = 1
	}

	// Check/set minimum config values
	if config.BufferConfig.BufferSize == 0 {
		config.BufferConfig.BufferSize = 1
	}

	if config.BufferConfig.NumWorkers == 0 {
		config.BufferConfig.NumWorkers = 1
	}

	ctx, cancel := context.WithCancelCause(ctx)

	dataStore, err := datastore.NewDataStore(ctx, config.DataStoreConfig, config.Network)
	if err != nil {
		return nil, err
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
		dataStore:         dataStore,
		ledgerMetaArchive: ledgerMetaArchive,
		decoder:           decoder,
	}

	return gcsBackend, nil
}

// GetLatestLedgerSequence returns the most recent ledger sequence number in the cloud storage bucket.
func (gcsb *GCSBackend) GetLatestLedgerSequence(ctx context.Context) (uint32, error) {
	var err error
	var archive historyarchive.ArchiveInterface

	if archive, err = datastore.CreateHistoryArchiveFromNetworkName(ctx, gcsb.config.Network); err != nil {
		return 0, err
	}

	resumableManager := datastore.NewResumableManager(gcsb.dataStore, gcsb.config.Network, gcsb.config.LedgerBatchConfig, archive)
	// Start at 2 to skip the genesis ledger
	absentLedger, ok, err := resumableManager.FindStart(ctx, 2, 0)
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

// GetLedger returns the LedgerCloseMeta for the specified ledger sequence number
func (gcsb *GCSBackend) GetLedger(ctx context.Context, sequence uint32) (xdr.LedgerCloseMeta, error) {
	gcsb.gcsBackendLock.RLock()
	defer gcsb.gcsBackendLock.RUnlock()

	if gcsb.closed {
		return xdr.LedgerCloseMeta{}, errors.New("gcsBackend is closed")
	}

	if gcsb.prepared == nil {
		return xdr.LedgerCloseMeta{}, errors.New("session is not prepared, call PrepareRange first")
	}

	err := gcsb.getSequenceInBatch(sequence)
	if err != nil {
		return xdr.LedgerCloseMeta{}, err
	}

	ledgerCloseMeta, err := gcsb.ledgerMetaArchive.GetLedger(sequence)
	if err != nil {
		return xdr.LedgerCloseMeta{}, err
	}

	return *ledgerCloseMeta, nil
}

// PrepareRange checks if the starting and ending (if bounded) ledgers exist.
func (gcsb *GCSBackend) PrepareRange(ctx context.Context, ledgerRange Range) error {
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
	gcsb.ledgerBuffer, err = gcsb.NewLedgerBuffer(ledgerRange)
	if err != nil {
		return false, err
	}

	// Start the ledgerBuffer
	gcsb.ledgerBuffer.pushTaskQueue()

	return false, nil
}
