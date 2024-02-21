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
	GetMetaArchiveChannel() chan *LedgerMetaArchive
	Run(ctx context.Context, startLedger uint32, endLedger uint32) error
	AddLedgerCloseMeta(ledgerCloseMeta xdr.LedgerCloseMeta) error
}

type exportManager struct {
	config         ExporterConfig
	backend        ledgerbackend.LedgerBackend
	metaArchiveMap map[string]*LedgerMetaArchive
	metaArchiveCh  chan *LedgerMetaArchive
}

// NewExportManager creates a new ExportManager with the provided configuration.
func NewExportManager(config ExporterConfig, backend ledgerbackend.LedgerBackend) ExportManager {
	return &exportManager{
		config:         config,
		backend:        backend,
		metaArchiveMap: make(map[string]*LedgerMetaArchive),
		metaArchiveCh:  make(chan *LedgerMetaArchive, 1),
	}
}

// GetMetaArchiveChannel returns a channel that receives LedgerMetaArchive objects.
func (e *exportManager) GetMetaArchiveChannel() chan *LedgerMetaArchive {
	return e.metaArchiveCh
}

// AddLedgerCloseMeta adds ledger metadata to the current export object
func (e *exportManager) AddLedgerCloseMeta(ledgerCloseMeta xdr.LedgerCloseMeta) error {
	ledgerSeq := ledgerCloseMeta.LedgerSequence()

	// Determine the object key for the given ledger sequence
	objectKey, err := GetObjectKeyFromSequenceNumber(e.config, ledgerSeq)
	if err != nil {
		return errors.Wrapf(err, "failed to get object key for ledger %d", ledgerSeq)
	}
	metaArchive, exists := e.metaArchiveMap[objectKey]

	if !exists {
		endSeq := ledgerSeq + e.config.LedgersPerFile - 1
		if ledgerSeq < e.config.LedgersPerFile {
			// Special case: Adjust the end ledger sequence for the first batch.
			// Since the start ledger is 2 instead of 0, we want to ensure that the end ledger sequence
			// does not exceed LedgersPerFile.
			// For example, if LedgersPerFile is 64, the file name for the first batch should be 0-63, not 2-66.
			endSeq = e.config.LedgersPerFile - 1
		}

		// Create a new LedgerMetaArchive and add it to the map.
		metaArchive = NewLedgerMetaArchive(objectKey, ledgerSeq, endSeq)
		e.metaArchiveMap[objectKey] = metaArchive
	}

	err = metaArchive.AddLedger(ledgerCloseMeta)
	if err != nil {
		return errors.Wrapf(err, "failed to add ledger %d", ledgerSeq)
	}

	if ledgerSeq >= metaArchive.GetEndLedgerSequence() {
		// Current archive is full, send it for upload
		e.metaArchiveCh <- metaArchive
		// Remove it from the map
		delete(e.metaArchiveMap, objectKey)
	}
	return nil
}

// Run iterates over the specified range of ledgers, retrieves ledger data
// from the backend, and processes the corresponding ledger close metadata.
// The process continues until the ending ledger number is reached or a cancellation
// signal is received.
func (e *exportManager) Run(ctx context.Context, startLedger, endLedger uint32) error {

	// Close the object channel
	defer close(e.metaArchiveCh)

	for nextLedger := startLedger; endLedger < 1 || nextLedger <= endLedger; {
		select {
		case <-ctx.Done():
			logger.Info("ExportManager stopped")
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
	logger.Infof("ExportManager has completed exporting the ledger range [%d - %d]", startLedger, endLedger)
	return nil
}
