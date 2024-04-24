package ledgerbackend

import (
	"compress/gzip"
	"context"
	"io"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stellar/go/support/datastore"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/ordered"
	"github.com/stellar/go/xdr"
)

// Ensure GCSBackend implements LedgerBackend
var _ LedgerBackend = (*GCSBackend)(nil)

type LCMCache struct {
	mu  sync.Mutex
	lcm map[uint32]xdr.LedgerCloseMeta
}

type LCMFileConfig struct {
	StorageURL        string
	FileSuffix        string
	LedgersPerFile    uint32
	FilesPerPartition uint32
	parallelReaders   uint32
}

// GCSBackend is a ledger backend that reads from a cloud storage service.
// The cloud storage service contains files generated from the ledgerExporter.
type GCSBackend struct {
	lcmDataStore      datastore.DataStore
	storageURL        string
	fileSuffix        string
	ledgersPerFile    uint32
	filesPerPartition uint32

	// cancel is the CancelFunc for context which controls the lifetime of a GCSBackend instance.
	// Once it is invoked GCSBackend will not be able to stream ledgers from GCSBackend or
	// spawn new instances of Stellar Core.
	cancel context.CancelFunc

	// gcsBackendLock protects access to gcsBackendRunner. When the read lock
	// is acquired gcsBackendRunner can be accessed. When the write lock is acquired
	// gcsBackendRunner can be updated.
	gcsBackendLock sync.RWMutex

	// lcmCache keeps that ledger close meta in-memory.
	lcmCache *LCMCache

	prepared        *Range // non-nil if any range is prepared
	closed          bool   // False until the core is closed
	parallelReaders uint32 // Number of parallel GCS readers
	context         context.Context
	nextLedger      uint32  // next ledger expected, error w/ restart if not seen
	lastLedger      *uint32 // end of current segment if offline, nil if online
}

// Return a new GCSBackend instance.
func NewGCSBackend(ctx context.Context, fileConfig LCMFileConfig) (*GCSBackend, error) {
	lcmDataStore, err := datastore.NewDataStore(ctx, fileConfig.StorageURL)
	if err != nil {
		return nil, err
	}

	if ctx == nil {
		ctx = context.Background()
	}

	lcmCache := &LCMCache{lcm: make(map[uint32]xdr.LedgerCloseMeta)}

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	// Need at least 1 reader
	parallelReaders := fileConfig.parallelReaders
	if parallelReaders == 0 {
		parallelReaders = 1
	}

	cloudStorageBackend := &GCSBackend{
		lcmDataStore:      lcmDataStore,
		storageURL:        fileConfig.StorageURL,
		fileSuffix:        fileConfig.FileSuffix,
		ledgersPerFile:    fileConfig.LedgersPerFile,
		filesPerPartition: fileConfig.FilesPerPartition,
		cancel:            cancel,
		lcmCache:          lcmCache,
		parallelReaders:   fileConfig.parallelReaders,
		context:           ctx,
	}

	return cloudStorageBackend, nil
}

// GetLatestLedgerSequence returns the most recent ledger sequence number in the cloud storage bucket.
func (gcsb *GCSBackend) GetLatestLedgerSequence(ctx context.Context) (uint32, error) {
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

	// Block until the requested sequence is available
	for {
		select {
		case <-ctx.Done():
			return xdr.LedgerCloseMeta{}, ctx.Err()
		default:
			lcm, ok := gcsb.lcmCache.lcm[sequence]
			if !ok {
				continue
			}
			// Delete to free space for unbounded mode lcm retrieval
			delete(gcsb.lcmCache.lcm, sequence)
			return lcm, nil
		}
	}
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

	// lastLedger is only set when ledgerRange is bounded
	lastLedger := uint32(0)
	if gcsb.lastLedger != nil {
		lastLedger = *gcsb.lastLedger
	}

	if gcsb.prepared == nil {
		return false
	}

	// Unbounded mode only checks for the starting ledger
	if lastLedger == 0 {
		_, ok := gcsb.lcmCache.lcm[ledgerRange.from]
		return ok
	}

	// From now on: lastLedger != 0 so current range is bounded
	if ledgerRange.bounded {
		_, ok := gcsb.lcmCache.lcm[ledgerRange.from]
		return ok && lastLedger >= ledgerRange.to
	}

	// Requested range is unbounded but current one is bounded
	return false
}

// Close closes existing GCSBackend process, streaming sessions and removes all
// temporary files. Note, once a GCSBackend instance is closed it can no longer be used and
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
		fileNameTrimExt := strings.TrimSuffix(fileName, gcsb.fileSuffix)
		fileNameTrimPath := strings.TrimPrefix(fileNameTrimExt, directory+"/")
		ledgerSequence, err := strconv.ParseUint(fileNameTrimPath, 10, 32)
		if err != nil {
			return uint32(0), errors.Wrapf(err, "failed converting filename to uint32 %s", fileName)
		}

		latestLedgerSequence = ordered.Max(latestLedgerSequence, uint32(ledgerSequence))
	}

	return latestLedgerSequence, nil
}

// startPreparingRange prepares the ledger range
// Bounded ranges will load the full range to lcmCache
// Unbounded ranges will continuously read new LCM to lcmCache
func (gcsb *GCSBackend) startPreparingRange(ctx context.Context, ledgerRange Range) (bool, error) {
	gcsb.gcsBackendLock.Lock()
	defer gcsb.gcsBackendLock.Unlock()

	if gcsb.isPrepared(ledgerRange) {
		return true, nil
	}

	// Set the starting ledger
	gcsb.nextLedger = ledgerRange.from

	if ledgerRange.bounded {
		gcsb.getGCSObjectsParallel(ctx, ledgerRange)
		return false, nil
	}

	// If unbounded, continously get new ledgers
	go gcsb.getNewLedgerObjects(ctx)

	return false, nil
}

// getGCSObjectsParallel loads the LCM from the ledgerRange to lcmCache in parallel
func (gcsb *GCSBackend) getGCSObjectsParallel(ctx context.Context, ledgerRange Range) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, gcsb.parallelReaders)

	interval := (ledgerRange.to - ledgerRange.from) / gcsb.parallelReaders

	for i := uint32(0); i < gcsb.parallelReaders; i++ {
		wg.Add(1)
		sem <- struct{}{} // Acquire a slot in the semaphore

		// Get the subrange of ledgers to process
		from := ledgerRange.from + (i * interval)
		to := ledgerRange.from + ((i + 1) * interval) - 1
		// The last reader should run to the final ledger
		if i+1 == gcsb.parallelReaders {
			to = ledgerRange.to
		}
		subLedgerRange := BoundedRange(from, to)

		wg.Add(1)
		sem <- struct{}{}
		go gcsb.getGCSObjects(ctx, subLedgerRange, &wg, sem)
	}

	wg.Wait()
	gcsb.lastLedger = &ledgerRange.to
}

// getGCSObjects loads the LCM to lcmCache
func (gcsb *GCSBackend) getGCSObjects(ctx context.Context, ledgerRange Range, wg *sync.WaitGroup, sem chan struct{}) {
	defer wg.Done()
	defer func() { <-sem }()

	select {
	case <-ctx.Done(): // Check if the context was cancelled
		return
	default:
		for i := ledgerRange.from; i <= ledgerRange.to; i++ {
			lcm := gcsb.getLedgerGCSObject(i)

			// Store lcm in-memory
			gcsb.lcmCache.mu.Lock()
			gcsb.lcmCache.lcm[i] = lcm
			gcsb.lcmCache.mu.Unlock()
		}
	}
}

// getLedgerGCSObject gets the LCM for a given ledger sequence
func (gcsb *GCSBackend) getLedgerGCSObject(sequence uint32) xdr.LedgerCloseMeta {
	var ledgerCloseMetaBatch xdr.LedgerCloseMetaBatch

	objectKey, err := datastore.GetObjectKeyFromSequenceNumber(sequence, gcsb.ledgersPerFile, gcsb.filesPerPartition, gcsb.fileSuffix)
	if err != nil {
		log.Fatalf("failed to get object key for ledger %d; %s", sequence, err)
	}

	reader, err := gcsb.lcmDataStore.GetFile(context.Background(), objectKey)
	if err != nil {
		log.Fatalf("failed getting file: %s; %s", objectKey, err)
	}

	defer reader.Close()

	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		log.Fatalf("failed getting file: %s; %s", objectKey, err)
	}

	defer gzipReader.Close()

	objectBytes, err := io.ReadAll(gzipReader)
	if err != nil {
		log.Fatalf("failed reading file: %s; %s", objectKey, err)
	}

	err = ledgerCloseMetaBatch.UnmarshalBinary(objectBytes)
	if err != nil {
		log.Fatalf("failed unmarshalling file: %s; %s", objectKey, err)
	}

	startSequence := uint32(ledgerCloseMetaBatch.StartSequence)
	if startSequence > sequence {
		log.Fatalf("start sequence: %d; greater than sequence to get: %d; %s", startSequence, sequence, err)
	}

	ledgerCloseMetasIndex := sequence - startSequence
	ledgerCloseMeta := ledgerCloseMetaBatch.LedgerCloseMetas[ledgerCloseMetasIndex]

	return ledgerCloseMeta
}

// getNewLedgerObjects polls GCS and buffers new LCM
func (gcsb *GCSBackend) getNewLedgerObjects(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return // Exit if the context is canceled
		case <-time.After(1):
			// Buffer lcmCache if LCMs exist
			for len(gcsb.lcmCache.lcm) < 1000 {
				// Check if LCM exists; otherwise wait and poll again
				objectKey, _ := datastore.GetObjectKeyFromSequenceNumber(gcsb.nextLedger, gcsb.ledgersPerFile, gcsb.filesPerPartition, gcsb.fileSuffix)
				exists, _ := gcsb.lcmDataStore.Exists(context.Background(), objectKey)
				if !exists {
					break
				}
				// Get LCM and add to lcmCache
				lcm := gcsb.getLedgerGCSObject(gcsb.nextLedger)
				gcsb.lcmCache.lcm[gcsb.nextLedger] = lcm
				gcsb.nextLedger += 1
			}
		}
	}
}
