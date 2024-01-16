package main

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
	objectKey     string
	startSequence uint32
	endSequence   uint32
	// Actual binary data
	data xdr.LedgerCloseMetaBatch
}

// AddLedgerCloseMeta adds a ledger
func (f *LedgerCloseMetaObject) AddLedgerCloseMeta(ledgerCloseMeta xdr.LedgerCloseMeta) {
	if f.startSequence == 0 {
		f.data.StartSequence = xdr.Uint32(ledgerCloseMeta.LedgerSequence())
	}
	f.data.LedgerCloseMetas = append(f.data.LedgerCloseMetas, ledgerCloseMeta)
	f.data.EndSequence = xdr.Uint32(ledgerCloseMeta.LedgerSequence())
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
func (e *ExporterConfig) getObjectKey(ledgerSeq uint32) string {
	var objectKey string

	if e.FilesPerPartition != 0 {
		partitionSize := e.LedgersPerFile * e.FilesPerPartition
		partitionStart := (ledgerSeq / partitionSize) * partitionSize
		partitionEnd := partitionStart + partitionSize - 1
		objectKey = fmt.Sprintf("%v-%v/", partitionStart, partitionEnd)
	}

	// TODO: 0 ledgersPerFile is invalid, throw an error and move this check to config validation.
	if e.LedgersPerFile != 0 {
		fileStart := (ledgerSeq / e.LedgersPerFile) * e.LedgersPerFile
		fileEnd := fileStart + e.LedgersPerFile - 1
		objectKey += fmt.Sprintf("%v", fileStart)

		// Multiple ledgers per file
		if fileStart != fileEnd {
			objectKey += fmt.Sprintf("-%v", fileEnd)
		}
	}
	return objectKey
}

// ExportManager manages the creation and handling of export objects.
type ExportManager interface {
	GetExportObjectsChannel() chan *LedgerCloseMetaObject
	Run(ctx context.Context, startLedger uint32, endLedger uint32) error
	AddLedgerCloseMeta(ledgerCloseMeta xdr.LedgerCloseMeta)
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
func (e *exportManager) AddLedgerCloseMeta(ledgerCloseMeta xdr.LedgerCloseMeta) {
	ledgerSeq := ledgerCloseMeta.LedgerSequence()

	// Determine the object key for the given ledger sequence
	objectKey := e.config.getObjectKey(ledgerSeq)
	ledgerCloseMetaObject, exists := e.objectMap[objectKey]

	if !exists {
		// Create a new LedgerCloseMetaObject and add it to the map.
		ledgerCloseMetaObject = &LedgerCloseMetaObject{
			objectKey:     objectKey,
			startSequence: ledgerSeq,
			endSequence:   ledgerSeq + e.config.LedgersPerFile - 1,
		}

		// Special case: Adjust the end ledger sequence for the first batch.
		// Since the start ledger is 2 instead of 0, we want to ensure that the end ledger sequence
		// does not exceed LedgersPerFile.
		// For example, if LedgersPerFile is 64, the file name for the first batch should be 0-63, not 2-66.
		if ledgerSeq < e.config.LedgersPerFile {
			ledgerCloseMetaObject.endSequence = e.config.LedgersPerFile - 1
		}

		e.objectMap[objectKey] = ledgerCloseMetaObject
	}

	// Add ledger to the LedgerCloseMetaObject
	ledgerCloseMetaObject.AddLedgerCloseMeta(ledgerCloseMeta)

	//logger.Logf("ledger Seq: %d object: %s ledgercount: %d ledgersperfile: %d", ledgerSeq,

	if ledgerSeq >= ledgerCloseMetaObject.endSequence {
		// Current export object is full, send it for upload
		// This is a blocking call!
		e.exportObjectCh <- ledgerCloseMetaObject
		// Remove it from the map
		delete(e.objectMap, objectKey)
	}
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
			logger.Info("ExportManager stopped due to context cancellation.")
			return nil
		default:
			ledgerCloseMeta, err := e.backend.GetLedger(ctx, nextLedger)
			if err != nil {
				return errors.Wrap(err, "ExportManager encountered an error while fetching ledger from the backend")
			}
			//time.Sleep(time.Duration(1) * time.Second)
			e.AddLedgerCloseMeta(ledgerCloseMeta)
			nextLedger++
		}
	}

	return nil
}
