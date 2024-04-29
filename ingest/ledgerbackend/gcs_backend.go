package ledgerbackend

import (
	"compress/gzip"
	"container/heap"
	"context"
	"io"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stellar/go/support/datastore"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/priorityqueue"
	"github.com/stellar/go/xdr"
	"google.golang.org/api/googleapi"
)

// Ensure GCSBackend implements LedgerBackend
var _ LedgerBackend = (*GCSBackend)(nil)

type LCMFileConfig struct {
	StorageURL        string
	FileSuffix        string
	LedgersPerFile    uint32 `default:"1"`
	FilesPerPartition uint32 `default:"64000"`
}

type BufferConfig struct {
	BufferSize uint32        `default:"1000"`
	NumWorkers uint32        `default:"5"`
	RetryLimit uint32        `default:"3"`
	RetryWait  time.Duration `default:"5"`
}

type GCSBackendConfig struct {
	LcmFileConfig LCMFileConfig
	BufferConfig  BufferConfig
}

// GCSBackend is a ledger backend that reads from a cloud storage service.
// The cloud storage service contains files generated from the ledgerExporter.
type GCSBackend struct {
	config GCSBackendConfig

	// cancel is the CancelFunc for context which controls the lifetime of a GCSBackend instance.
	// Once it is invoked GCSBackend will not be able to stream ledgers from GCSBackend.
	cancel context.CancelFunc

	// gcsBackendLock protects access to gcsBackendRunner. When the read lock
	// is acquired gcsBackendRunner can be accessed. When the write lock is acquired
	// gcsBackendRunner can be updated.
	gcsBackendLock sync.RWMutex

	// ledgerBuffer is the buffer for LedgerCloseMeta data read in parallel.
	ledgerBuffer *ledgerBufferGCS

	prepared *Range // non-nil if any range is prepared
	closed   bool   // False until the core is closed
}

type ledgerBufferGCS struct {
	config                GCSBackendConfig
	lcmDataStore          datastore.DataStore
	taskQueue             chan uint32
	ledgerQueue           chan []byte
	ledgerPriorityQueue   priorityqueue.PriorityQueue
	priorityQueueLock     sync.Mutex
	count                 uint32
	limit                 uint32
	currentLedger         uint32
	nextTaskLedger        uint32
	nextLedgerQueueLedger uint32
	cancel                context.CancelFunc
	bounded               bool
	ledgerRange           *Range
}

func NewLedgerBuffer(ctx context.Context, config GCSBackendConfig) (*ledgerBufferGCS, error) {
	var cancel context.CancelFunc

	lcmDataStore, err := datastore.NewDataStore(ctx, config.LcmFileConfig.StorageURL)
	if err != nil {
		return nil, err
	}

	pq := make(priorityqueue.PriorityQueue, config.BufferConfig.BufferSize)
	heap.Init(&pq)

	ledgerBuffer := &ledgerBufferGCS{
		lcmDataStore:        lcmDataStore,
		taskQueue:           make(chan uint32, config.BufferConfig.BufferSize),
		ledgerQueue:         make(chan []byte, config.BufferConfig.BufferSize),
		ledgerPriorityQueue: pq,
		count:               0,
		limit:               config.BufferConfig.BufferSize,
		cancel:              cancel,
	}

	// Workers to read LCM files
	for i := uint32(0); i < config.BufferConfig.NumWorkers; i++ {
		go ledgerBuffer.worker()
	}

	// goroutine to correctly LCM files
	go ledgerBuffer.reorderLedgers()

	return ledgerBuffer, nil
}

func (lb *ledgerBufferGCS) pushTaskQueue() {
	for lb.count <= lb.limit {
		lb.taskQueue <- lb.nextTaskLedger
		lb.nextTaskLedger++
		lb.count++
	}
}

func (lb *ledgerBufferGCS) worker() {
	for sequence := range lb.taskQueue {
		retryCount := uint32(0)
		for retryCount <= lb.config.BufferConfig.RetryLimit {
			ledgerObject, err := lb.getLedgerGCSObject(sequence)
			if err != nil {
				if e, ok := err.(*googleapi.Error); ok {
					// ledgerObject not found and unbounded
					if e.Code == 404 && !lb.bounded {
						time.Sleep(lb.config.BufferConfig.RetryWait * time.Second)
						continue
					}
				}
				retryCount++
				time.Sleep(lb.config.BufferConfig.RetryWait * time.Second)
			}

			// Add to priority queue and continue to next task
			lb.priorityQueueLock.Lock()
			item := &priorityqueue.Item{
				Value:    ledgerObject,
				Priority: int(sequence),
			}
			heap.Push(&lb.ledgerPriorityQueue, item)
			lb.ledgerPriorityQueue.Update(item, item.Value, int(sequence))
			lb.priorityQueueLock.Unlock()
			break
		}
		// Add abort case for max retries
	}
}

func (lb *ledgerBufferGCS) getLedgerGCSObject(sequence uint32) ([]byte, error) {
	var ledgerCloseMetaBatch xdr.LedgerCloseMetaBatch

	objectKey, err := datastore.GetObjectKeyFromSequenceNumber(
		sequence,
		lb.config.LcmFileConfig.LedgersPerFile,
		lb.config.LcmFileConfig.FilesPerPartition,
		lb.config.LcmFileConfig.FileSuffix)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get object key for ledger %d", sequence)
	}

	reader, err := lb.lcmDataStore.GetFile(context.Background(), objectKey)
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
	for lb.currentLedger == uint32(lb.ledgerPriorityQueue[0].Priority) {
		item := heap.Pop(&lb.ledgerPriorityQueue).(*priorityqueue.Item)
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
func NewGCSBackend(ctx context.Context, config GCSBackendConfig) (*GCSBackend, error) {
	// Check/set minimum config values
	if config.LcmFileConfig.StorageURL == "" {
		return nil, errors.New("fileConfig.storageURL is not set")
	}

	if config.LcmFileConfig.FileSuffix == "" {
		return nil, errors.New("fileConfig.FileSuffix is not set")
	}

	if config.LcmFileConfig.LedgersPerFile == 0 {
		config.LcmFileConfig.LedgersPerFile = 1
	}

	if config.LcmFileConfig.FilesPerPartition == 0 {
		config.LcmFileConfig.FilesPerPartition = 1
	}

	// Check/set minimum config values
	if config.BufferConfig.BufferSize == 0 {
		config.BufferConfig.BufferSize = 1
	}

	if config.BufferConfig.NumWorkers == 0 {
		config.BufferConfig.NumWorkers = 1
	}

	var cancel context.CancelFunc

	ledgerBuffer, err := NewLedgerBuffer(ctx, config)
	if err != nil {
		return nil, err
	}

	gcsBackend := &GCSBackend{
		config:       config,
		cancel:       cancel,
		ledgerBuffer: ledgerBuffer,
	}

	return gcsBackend, nil
}

// GetLatestLedgerSequence returns the most recent ledger sequence number in the cloud storage bucket.
func (gcsb *GCSBackend) GetLatestLedgerSequence(ctx context.Context) (uint32, error) {
	/* TODO: replace with binary search code
	gcsb.gcsBackendLock.RLock()
	defer gcsb.gcsBackendLock.RUnlock()

	// Get the latest parition directory from the bucket
	directories, err := gcsb.lcmDataStore.ListDirectoryNames(ctx)
	if err != nil {
		return 0, errors.Wrapf(err, "failed getting list of directory names")
	}

	latestDirectory, err := gcsb.GetLatestDirectory(directories)
	if err != nil {
		return 0, errors.Wrapf(err, "failed getting latest directory")
	}

	// Search through the latest partition to find the latest file which would be the latestLedgerSequence
	fileNames, err := gcsb.lcmDataStore.ListFileNames(ctx, latestDirectory)
	if err != nil {
		return 0, errors.Wrapf(err, "failed getting filenames in dir %s", latestDirectory)
	}

	latestLedgerSequence, err := gcsb.GetLatestFileNameLedgerSequence(fileNames, latestDirectory)
	if err != nil {
		return 0, errors.Wrapf(err, "failed converting filename to ledger sequence")
	}

	return latestLedgerSequence, nil
	*/
	return 0, nil
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

	if gcsb.ledgerBuffer.ledgerRange == nil {
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

// TODO: remove when binary search is merged
/*
// GetLatestDirectory returns the latest directory from an array of directories
func (gcsb *GCSBackend) GetLatestDirectory(directories []string) (string, error) {
	var latestDirectory string
	largestDirectoryLedger := 0

	for _, dir := range directories {
		// dir follows the format of "ledgers/<network>/<start>-<end>"
		// Need to split the dir string to retrieve the <end> ledger value to get the latest directory
		dirTruncSlash := strings.TrimSuffix(dir, "/")
		_, dirName := path.Split(dirTruncSlash)
		parts := strings.Split(dirName, "-")

		if len(parts) == 2 {
			upper, err := strconv.Atoi(parts[1])
			if err != nil {
				return "", errors.Wrapf(err, "failed getting latest directory %s", dir)
			}

			if upper > largestDirectoryLedger {
				latestDirectory = dir
				largestDirectoryLedger = upper
			}
		}
	}

	return latestDirectory, nil
}

// GetLatestFileNameLedgerSequence returns the lastest ledger sequence in a directory
func (gcsb *GCSBackend) GetLatestFileNameLedgerSequence(fileNames []string, directory string) (uint32, error) {
	latestLedgerSequence := uint32(0)

	for _, fileName := range fileNames {
		// fileName follows the format of "ledgers/<network>/<start>-<end>/<ledger_sequence>.<fileSuffix>"
		// Trim the file down to just the <ledger_sequence>
		fileNameTrimExt := strings.TrimSuffix(fileName, gcsb.lcmFileConfig.FileSuffix)
		fileNameTrimPath := strings.TrimPrefix(fileNameTrimExt, directory+"/")
		ledgerSequence, err := strconv.ParseUint(fileNameTrimPath, 10, 32)
		if err != nil {
			return uint32(0), errors.Wrapf(err, "failed converting filename to uint32 %s", fileName)
		}

		latestLedgerSequence = ordered.Max(latestLedgerSequence, uint32(ledgerSequence))
	}

	return latestLedgerSequence, nil
}
*/

// startPreparingRange prepares the ledger range by setting the range in the ledgerBuffer
func (gcsb *GCSBackend) startPreparingRange(ledgerRange Range) (bool, error) {
	gcsb.gcsBackendLock.Lock()
	defer gcsb.gcsBackendLock.Unlock()

	if gcsb.isPrepared(ledgerRange) {
		return true, nil
	}

	// Set the ledgerRange in ledgerBuffer
	gcsb.ledgerBuffer.ledgerRange = &ledgerRange
	gcsb.ledgerBuffer.currentLedger = ledgerRange.from
	gcsb.ledgerBuffer.nextTaskLedger = ledgerRange.from
	gcsb.ledgerBuffer.nextLedgerQueueLedger = ledgerRange.from
	gcsb.ledgerBuffer.bounded = ledgerRange.bounded

	// Start the ledgerBuffer
	gcsb.ledgerBuffer.pushTaskQueue()

	return false, nil
}
