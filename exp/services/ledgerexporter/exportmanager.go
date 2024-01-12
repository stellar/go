package main

import (
	"context"
	"strconv"

	"github.com/stellar/go/ingest/ledgerbackend"
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
		objectKey = strconv.Itoa(int(partitionStart)) + "-" + strconv.Itoa(int(partitionEnd)) + "/"
	}

	// TODO: 0 ledgersPerFile is invalid, throw an error and move this check to config validation.
	if e.LedgersPerFile != 0 {
		fileStart := (ledgerSeq / e.LedgersPerFile) * e.LedgersPerFile
		fileEnd := fileStart + e.LedgersPerFile - 1

		// Single ledger per file
		if fileStart == fileEnd {
			objectKey += strconv.Itoa(int(fileStart))
		} else {
			objectKey += strconv.Itoa(int(fileStart)) + "-" + strconv.Itoa(int(fileEnd))
		}
	}
	return objectKey
}

// ExportManager manages the creation and handling of export objects.
type ExportManager struct {
	config                  ExporterConfig
	backend                 ledgerbackend.LedgerBackend
	objectMap               map[string]*LedgerCloseMetaObject
	LedgerCloseMetaObjectCh chan *LedgerCloseMetaObject
}

// NewExportManager creates a new ExportManager with the provided configuration.
func NewExportManager(config ExporterConfig, backend ledgerbackend.LedgerBackend,
	LedgerCloseMetaObjectCh chan *LedgerCloseMetaObject) *ExportManager {
	return &ExportManager{
		config:                  config,
		backend:                 backend,
		objectMap:               make(map[string]*LedgerCloseMetaObject),
		LedgerCloseMetaObjectCh: LedgerCloseMetaObjectCh,
	}
}

// AddLedgerCloseMeta adds ledger metadata to the current export object
func (e *ExportManager) AddLedgerCloseMeta(ledgerCloseMeta xdr.LedgerCloseMeta) error {
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
		// For example, if LedgersPerFile is 64, the first batch should be 0-63, not 2-66.
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
		e.LedgerCloseMetaObjectCh <- ledgerCloseMetaObject
		// Remove it from the map
		delete(e.objectMap, objectKey)
	}
	return nil
}

// Run iterates over the specified range of ledgers, retrieves ledger data
// from the backend, and processes the corresponding ledger close metadata.
// The process continues until the ending ledger number is reached or a cancellation
// signal is received.
func (e *ExportManager) Run(ctx context.Context, startLedger, endLedger uint32) {

	// Close the object channel to signal uploader that no more objects will be sent
	defer close(e.LedgerCloseMetaObjectCh)

	for nextLedger := startLedger; endLedger < 1 || nextLedger <= endLedger; {
		select {
		case <-ctx.Done():
			logger.Info("ExportManager stopped due to context cancellation.")
			return
		default:
			ledgerCloseMeta, err := e.backend.GetLedger(ctx, nextLedger)
			if err != nil {
				//Handle error
			}
			//time.Sleep(time.Duration(1) * time.Second)
			e.AddLedgerCloseMeta(ledgerCloseMeta)
			nextLedger++
		}
	}
}
