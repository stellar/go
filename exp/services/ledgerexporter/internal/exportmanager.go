package exporter

import (
	"context"
	"fmt"

	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// LedgerCloseMetaObject represents a file with metadata and binary data.
type LedgerCloseMetaObject struct {
	// file name
	objectKey string
	// Actual binary data
	data xdr.LedgerCloseMetaBatch
}

func NewLedgerCloseMetaObject(key string, startSeq uint32, endSeq uint32) *LedgerCloseMetaObject {
	return &LedgerCloseMetaObject{
		objectKey: key,
		data: xdr.LedgerCloseMetaBatch{
			StartSequence: xdr.Uint32(startSeq),
			EndSequence:   xdr.Uint32(endSeq),
		},
	}
}

func (f *LedgerCloseMetaObject) GetLastLedgerCloseMetaSequence() (uint32, error) {
	if len(f.data.LedgerCloseMetas) == 0 {
		return 0, errors.New("LedgerCloseMetas is empty")
	}

	return f.data.LedgerCloseMetas[len(f.data.LedgerCloseMetas)-1].LedgerSequence(), nil
}

// AddLedgerCloseMeta adds a ledger
func (f *LedgerCloseMetaObject) AddLedgerCloseMeta(ledgerCloseMeta xdr.LedgerCloseMeta) error {
	lastSequence, err := f.GetLastLedgerCloseMetaSequence()
	if err == nil {
		if ledgerCloseMeta.LedgerSequence() != lastSequence+1 {
			return fmt.Errorf("ledgers must be added sequentially. Sequence number: %d, "+
				"expected sequence number: %d", ledgerCloseMeta.LedgerSequence(), lastSequence+1)
		}
	}

	f.data.LedgerCloseMetas = append(f.data.LedgerCloseMetas, ledgerCloseMeta)
	return nil
}

// LedgerCount returns the number of ledgers added so far
func (f *LedgerCloseMetaObject) LedgerCount() uint32 {
	return uint32(len(f.data.LedgerCloseMetas))
}

type ExporterConfig struct {
	LedgersPerFile    uint32 `toml:"ledgers_per_file"`
	FilesPerPartition uint32 `toml:"files_per_partition"`
}

// getObjectKey generates the file name based on the ledger sequence.
func (e *ExporterConfig) getObjectKey(ledgerSeq uint32) (string, error) {
	var objectKey string

	if e.FilesPerPartition > 1 {
		partitionSize := e.LedgersPerFile * e.FilesPerPartition
		partitionStart := (ledgerSeq / partitionSize) * partitionSize
		partitionEnd := partitionStart + partitionSize - 1
		objectKey = fmt.Sprintf("%v-%v/", partitionStart, partitionEnd)
	}

	if e.LedgersPerFile < 1 {
		return "", errors.New("Ledgers per file must be at least 1")
	}

	fileStart := (ledgerSeq / e.LedgersPerFile) * e.LedgersPerFile
	fileEnd := fileStart + e.LedgersPerFile - 1
	objectKey += fmt.Sprintf("%v", fileStart)

	// Multiple ledgers per file
	if fileStart != fileEnd {
		objectKey += fmt.Sprintf("-%v", fileEnd)
	}

	return objectKey, nil
}

// ExportManager manages the creation and handling of export objects.
type ExportManager interface {
	GetExportObjectsChannel() chan *LedgerCloseMetaObject
	Run(ctx context.Context, startLedger uint32, endLedger uint32) error
	AddLedgerCloseMeta(ledgerCloseMeta xdr.LedgerCloseMeta) error
}

type exportManager struct {
	config         ExporterConfig
	backend        ledgerbackend.LedgerBackend
	objectMap      map[string]*LedgerCloseMetaObject
	exportObjectCh chan *LedgerCloseMetaObject
}

// NewExportManager creates a new ExportManager with the provided configuration.
func NewExportManager(config ExporterConfig, backend ledgerbackend.LedgerBackend) ExportManager {
	return &exportManager{
		config:         config,
		backend:        backend,
		objectMap:      make(map[string]*LedgerCloseMetaObject),
		exportObjectCh: make(chan *LedgerCloseMetaObject, 1),
	}
}

func (e *exportManager) GetExportObjectsChannel() chan *LedgerCloseMetaObject {
	return e.exportObjectCh
}

// AddLedgerCloseMeta adds ledger metadata to the current export object
func (e *exportManager) AddLedgerCloseMeta(ledgerCloseMeta xdr.LedgerCloseMeta) error {
	ledgerSeq := ledgerCloseMeta.LedgerSequence()

	// Determine the object key for the given ledger sequence
	objectKey, err := e.config.getObjectKey(ledgerSeq)
	if err != nil {
		return errors.Wrapf(err, "failed to get object key for ledger %d", ledgerSeq)
	}
	ledgerCloseMetaObject, exists := e.objectMap[objectKey]

	if !exists {
		// Create a new LedgerCloseMetaObject and add it to the map.
		ledgerCloseMetaObject = NewLedgerCloseMetaObject(objectKey, ledgerSeq,
			ledgerSeq+e.config.LedgersPerFile-1)

		// Special case: Adjust the end ledger sequence for the first batch.
		// Since the start ledger is 2 instead of 0, we want to ensure that the end ledger sequence
		// does not exceed LedgersPerFile.
		// For example, if LedgersPerFile is 64, the file name for the first batch should be 0-63, not 2-66.
		if ledgerSeq < e.config.LedgersPerFile {
			ledgerCloseMetaObject.data.EndSequence = xdr.Uint32(e.config.LedgersPerFile - 1)
		}

		e.objectMap[objectKey] = ledgerCloseMetaObject
	}

	// Add ledger to the LedgerCloseMetaObject
	if err := ledgerCloseMetaObject.AddLedgerCloseMeta(ledgerCloseMeta); err != nil {
		return errors.Wrapf(err, "failed to add ledger to LedgerCloseMetaObject")
	}

	//logger.Logf("ledger Seq: %d object: %s ledgercount: %d ledgersperfile: %d", ledgerSeq,

	if ledgerSeq >= uint32(ledgerCloseMetaObject.data.EndSequence) {
		// Current export object is full, send it for upload
		// This is a blocking call!
		e.exportObjectCh <- ledgerCloseMetaObject
		// Remove it from the map
		delete(e.objectMap, objectKey)
	}
	return nil
}

// Run iterates over the specified range of ledgers, retrieves ledger data
// from the backend, and processes the corresponding ledger close metadata.
// The process continues until the ending ledger number is reached or a cancellation
// signal is received.
func (e *exportManager) Run(ctx context.Context, startLedger, endLedger uint32) error {

	// Close the object channel
	defer close(e.exportObjectCh)

	for nextLedger := startLedger; endLedger < 1 || nextLedger <= endLedger; {
		select {
		case <-ctx.Done():
			logger.Info("ExportManager stopping..")
			return ctx.Err()
		default:
			ledgerCloseMeta, err := e.backend.GetLedger(ctx, nextLedger)
			if err != nil {
				return errors.Wrap(err, "ExportManager failed to fetch ledger from backend")
			}
			//time.Sleep(time.Duration(1) * time.Second)
			err = e.AddLedgerCloseMeta(ledgerCloseMeta)
			if err != nil {
				return errors.Wrapf(err, "failed to add ledger %d", nextLedger)
			}
			nextLedger++
		}
	}

	return nil
}
