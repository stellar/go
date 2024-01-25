package exporter

import (
	"context"

	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type ExporterConfig struct {
	LedgersPerFile    uint32 `toml:"ledgers_per_file"`
	FilesPerPartition uint32 `toml:"files_per_partition"`
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
	objectKey, err := GetObjectKeyFromSequenceNumber(e.config, ledgerSeq)
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
