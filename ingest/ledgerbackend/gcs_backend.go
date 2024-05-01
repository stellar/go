package ledgerbackend

import (
	"compress/gzip"
	"context"
	"io"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/support/collections/heap"
	"github.com/stellar/go/support/datastore"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"

	"google.golang.org/api/googleapi"
)

// Ensure GCSBackend implements LedgerBackend
var _ LedgerBackend = (*GCSBackend)(nil)

// An Item is something we manage in a priority queue.
type Item struct {
	Value    []byte // Value of the item
	Priority int    // The priority of the item in the queue.
}

type LCMFileConfig struct {
	StorageURL        string
	FileSuffix        string
	LedgersPerFile    uint32
	FilesPerPartition uint32
}

type BufferConfig struct {
	BufferSize uint32
	NumWorkers uint32
	RetryLimit uint32
	RetryWait  time.Duration
}

type gcsBackendConfig struct {
	lcmFileConfig   LCMFileConfig
	bufferConfig    BufferConfig
	dataStoreConfig datastore.DataStoreConfig
	network         string
}

// GCSBackend is a ledger backend that reads from a cloud storage service.
// The cloud storage service contains files generated from the ledgerExporter.
type GCSBackend struct {
	config gcsBackendConfig

	// cancel is the CancelFunc for context which controls the lifetime of a GCSBackend instance.
	// Once it is invoked GCSBackend will not be able to stream ledgers from GCSBackend.
	cancel context.CancelFunc

	// gcsBackendLock protects access to gcsBackendRunner. When the read lock
	// is acquired gcsBackendRunner can be accessed. When the write lock is acquired
	// gcsBackendRunner can be updated.
	gcsBackendLock sync.RWMutex

	// ledgerBuffer is the buffer for LedgerCloseMeta data read in parallel.
	ledgerBuffer *ledgerBufferGCS

	dataStore         datastore.DataStore
	ledgerBatchConfig datastore.LedgerBatchConfig
	network           string
	prepared          *Range // non-nil if any range is prepared
	closed            bool   // False until the core is closed
}

type ledgerBufferGCS struct {
	config                gcsBackendConfig
	dataStore             datastore.DataStore
	taskQueue             chan uint32
	ledgerQueue           chan []byte
	ledgerPriorityQueue   *heap.Heap[Item]
	priorityQueueLock     sync.Mutex
	count                 uint32
	limit                 uint32
	cancel                context.CancelFunc
	currentLedger         uint32
	nextTaskLedger        uint32
	nextLedgerQueueLedger uint32
	ledgerRange           Range
}

func (gcsb *GCSBackend) NewLedgerBuffer(ctx context.Context, ledgerRange Range) (*ledgerBufferGCS, error) {
	var cancel context.CancelFunc
	less := func(a, b Item) bool {
		return a.Priority < b.Priority
	}
	pq := heap.New(less, int(gcsb.config.bufferConfig.BufferSize))

	ledgerBuffer := &ledgerBufferGCS{
		config:                gcsb.config,
		dataStore:             gcsb.dataStore,
		taskQueue:             make(chan uint32, gcsb.config.bufferConfig.BufferSize),
		ledgerQueue:           make(chan []byte, gcsb.config.bufferConfig.BufferSize),
		ledgerPriorityQueue:   pq,
		count:                 0,
		limit:                 gcsb.config.bufferConfig.BufferSize,
		cancel:                cancel,
		currentLedger:         ledgerRange.from,
		nextTaskLedger:        ledgerRange.from,
		nextLedgerQueueLedger: ledgerRange.from,
		ledgerRange:           ledgerRange,
	}

	// Workers to read LCM files
	for i := uint32(0); i < gcsb.config.bufferConfig.NumWorkers; i++ {
		go ledgerBuffer.worker()
	}

	// goroutine to correctly LCM files
	go ledgerBuffer.reorderLedgers()

	return ledgerBuffer, nil
}

func (lb *ledgerBufferGCS) pushTaskQueue() {
	for lb.count <= lb.limit {
		// In bounded mode, don't queue past the end ledger
		if lb.ledgerRange.to < lb.nextTaskLedger && lb.ledgerRange.bounded {
			return
		}
		lb.taskQueue <- lb.nextTaskLedger
		lb.nextTaskLedger++
		lb.count++
	}
}

func (lb *ledgerBufferGCS) worker() {
	for sequence := range lb.taskQueue {
		retryCount := uint32(0)
		for retryCount <= lb.config.bufferConfig.RetryLimit {
			ledgerObject, err := lb.getLedgerGCSObject(sequence)
			if err != nil {
				if e, ok := err.(*googleapi.Error); ok {
					// ledgerObject not found and unbounded
					if e.Code == 404 && !lb.ledgerRange.bounded {
						time.Sleep(lb.config.bufferConfig.RetryWait * time.Second)
						continue
					}
				}
				retryCount++
				time.Sleep(lb.config.bufferConfig.RetryWait * time.Second)
			}

			// Add to priority queue and continue to next task
			lb.priorityQueueLock.Lock()
			item := Item{
				Value:    ledgerObject,
				Priority: int(sequence),
			}
			lb.ledgerPriorityQueue.Push(item)
			lb.priorityQueueLock.Unlock()
			break
		}
		// Add abort case for max retries
	}
}

func (lb *ledgerBufferGCS) getLedgerGCSObject(sequence uint32) ([]byte, error) {
	var ledgerCloseMetaBatch xdr.LedgerCloseMetaBatch

	config := datastore.LedgerBatchConfig{}
	objectKey := config.GetObjectKeyFromSequenceNumber(sequence)

	reader, err := lb.dataStore.GetFile(context.Background(), objectKey)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting file: %s", objectKey)
	}

	defer reader.Close()

	// Read file and unzip
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting file: %s", objectKey)
	}

	defer gzipReader.Close()

	objectBytes, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, errors.Wrapf(err, "failed reading file: %s", objectKey)
	}

	// Turn binary into xdr
	err = ledgerCloseMetaBatch.UnmarshalBinary(objectBytes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling file: %s", objectKey)
	}

	// Check if ledger sequence within the xdr.ledgerCloseMetaBatch
	startSequence := uint32(ledgerCloseMetaBatch.StartSequence)
	if startSequence > sequence {
		return nil, errors.Wrapf(err, "start sequence: %d; greater than sequence to get: %d", startSequence, sequence)
	}

	ledgerCloseMetasIndex := sequence - startSequence
	ledgerCloseMeta := ledgerCloseMetaBatch.LedgerCloseMetas[ledgerCloseMetasIndex]

	// Turn lcm back to binary to save memory in buffer
	lcmBinary, err := ledgerCloseMeta.MarshalBinary()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling lcm sequence: %d", sequence)
	}

	return lcmBinary, nil
}

func (lb *ledgerBufferGCS) reorderLedgers() {
	lb.priorityQueueLock.Lock()
	defer lb.priorityQueueLock.Unlock()

	// Nothing in priority queue
	if lb.ledgerPriorityQueue.Len() < 0 {
		return
	}

	// Check if the nextLedger is the next item in the priority queue
	for lb.currentLedger == uint32(lb.ledgerPriorityQueue.Peek().Priority) {
		item := lb.ledgerPriorityQueue.Pop()
		lb.ledgerQueue <- item.Value
		lb.nextLedgerQueueLedger++
	}
}

func (lb *ledgerBufferGCS) getFromLedgerQueue(ctx context.Context) ([]byte, error) {
	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping ExportManager due to context cancellation")
			return nil, ctx.Err()
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
func NewGCSBackend(ctx context.Context, config gcsBackendConfig) (*GCSBackend, error) {
	// Check/set minimum config values
	if config.lcmFileConfig.StorageURL == "" {
		return nil, errors.New("fileConfig.storageURL is not set")
	}

	if config.lcmFileConfig.FileSuffix == "" {
		return nil, errors.New("fileConfig.FileSuffix is not set")
	}

	if config.lcmFileConfig.LedgersPerFile == 0 {
		config.lcmFileConfig.LedgersPerFile = 1
	}

	if config.lcmFileConfig.FilesPerPartition == 0 {
		config.lcmFileConfig.FilesPerPartition = 1
	}

	// Check/set minimum config values
	if config.bufferConfig.BufferSize == 0 {
		config.bufferConfig.BufferSize = 1
	}

	if config.bufferConfig.NumWorkers == 0 {
		config.bufferConfig.NumWorkers = 1
	}

	var cancel context.CancelFunc

	dataStore, err := datastore.NewDataStore(ctx, config.dataStoreConfig, config.network)
	if err != nil {
		return nil, err
	}

	gcsBackend := &GCSBackend{
		config:    config,
		cancel:    cancel,
		dataStore: dataStore,
	}

	return gcsBackend, nil
}

// GetLatestLedgerSequence returns the most recent ledger sequence number in the cloud storage bucket.
func (gcsb *GCSBackend) GetLatestLedgerSequence(ctx context.Context) (uint32, error) {
	var err error
	var archive historyarchive.ArchiveInterface

	if archive, err = datastore.CreateHistoryArchiveFromNetworkName(ctx, gcsb.network); err != nil {
		return 0, err
	}
	resumableManager := datastore.NewResumableManager(gcsb.dataStore, gcsb.network, gcsb.ledgerBatchConfig, archive)
	absentLedger, ok, err := resumableManager.FindStart(ctx, 1, 0)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, errors.New("findStart returned sequence beyond latest history archive ledger")
	}

	// Subtract one to get the oldest existing ledger seq closest to genesis
	return absentLedger - 1, nil
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

	var lcmBinary []byte
	var err error
	for gcsb.ledgerBuffer.currentLedger <= sequence {
		lcmBinary, err = gcsb.ledgerBuffer.getFromLedgerQueue(ctx)
		if err != nil {
			return xdr.LedgerCloseMeta{}, errors.Wrapf(err, "could not get ledger sequence binary: %d", sequence)
		}
	}

	var lcm xdr.LedgerCloseMeta
	err = lcm.UnmarshalBinary(lcmBinary)
	if err != nil {
		return xdr.LedgerCloseMeta{}, err
	}

	return lcm, nil
}

// PrepareRange checks if the starting and ending (if bounded) ledgers exist.
func (gcsb *GCSBackend) PrepareRange(ctx context.Context, ledgerRange Range) error {
	if alreadyPrepared, err := gcsb.startPreparingRange(ctx, ledgerRange); err != nil {
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

	// after the GCSBackend context is canceled all subsequent calls to PrepareRange() will fail
	gcsb.cancel()

	return nil
}

// startPreparingRange prepares the ledger range by setting the range in the ledgerBuffer
func (gcsb *GCSBackend) startPreparingRange(ctx context.Context, ledgerRange Range) (bool, error) {
	gcsb.gcsBackendLock.Lock()
	defer gcsb.gcsBackendLock.Unlock()

	if gcsb.isPrepared(ledgerRange) {
		return true, nil
	}

	var err error
	gcsb.ledgerBuffer, err = gcsb.NewLedgerBuffer(ctx, ledgerRange)
	if err != nil {
		return false, err
	}

	// Start the ledgerBuffer
	gcsb.ledgerBuffer.pushTaskQueue()

	return false, nil
}
