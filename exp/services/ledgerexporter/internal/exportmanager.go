package ledgerexporter

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stellar/go/ingest/ledgerbackend"
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
	AddLedgerCloseMeta(ctx context.Context, ledgerCloseMeta xdr.LedgerCloseMeta) error
}

type exportManager struct {
	config             ExporterConfig
	ledgerBackend      ledgerbackend.LedgerBackend
	currentMetaArchive *LedgerMetaArchive
	metaArchiveCh      chan *LedgerMetaArchive
}

// NewExportManager creates a new ExportManager with the provided configuration.
func NewExportManager(config ExporterConfig, backend ledgerbackend.LedgerBackend) ExportManager {
	return &exportManager{
		config:        config,
		ledgerBackend: backend,
		metaArchiveCh: make(chan *LedgerMetaArchive, 1),
	}
}

// GetMetaArchiveChannel returns a channel that receives LedgerMetaArchive objects.
func (e *exportManager) GetMetaArchiveChannel() chan *LedgerMetaArchive {
	return e.metaArchiveCh
}

// AddLedgerCloseMeta adds ledger metadata to the current export object
func (e *exportManager) AddLedgerCloseMeta(ctx context.Context, ledgerCloseMeta xdr.LedgerCloseMeta) error {
	ledgerSeq := ledgerCloseMeta.LedgerSequence()

	// Determine the object key for the given ledger sequence
	objectKey, err := GetObjectKeyFromSequenceNumber(e.config, ledgerSeq)
	if err != nil {
		return errors.Wrapf(err, "failed to get object key for ledger %d", ledgerSeq)
	}
	if e.currentMetaArchive != nil && e.currentMetaArchive.GetObjectKey() != objectKey {
		return errors.New("Current meta archive object key mismatch")
	}
	if e.currentMetaArchive == nil {
		endSeq := ledgerSeq + e.config.LedgersPerFile - 1
		if ledgerSeq < e.config.LedgersPerFile {
			// Special case: Adjust the end ledger sequence for the first batch.
			// Since the start ledger is 2 instead of 0, we want to ensure that the end ledger sequence
			// does not exceed LedgersPerFile.
			// For example, if LedgersPerFile is 64, the file name for the first batch should be 0-63, not 2-66.
			endSeq = e.config.LedgersPerFile - 1
		}

		// Create a new LedgerMetaArchive and add it to the map.
		e.currentMetaArchive = NewLedgerMetaArchive(objectKey, ledgerSeq, endSeq)
	}

	err = e.currentMetaArchive.AddLedger(ledgerCloseMeta)
	if err != nil {
		return errors.Wrapf(err, "failed to add ledger %d", ledgerSeq)
	}

	if ledgerSeq >= e.currentMetaArchive.GetEndLedgerSequence() {
		// Current archive is full, send it for upload
		select {
		case e.metaArchiveCh <- e.currentMetaArchive:
			e.currentMetaArchive = nil
		case <-ctx.Done():
			return ctx.Err()
		}
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

	for nextLedger := startLedger; endLedger < 1 || nextLedger <= endLedger; nextLedger++ {
		select {
		case <-ctx.Done():
			logger.Info("Stopping ExportManager due to context cancellation")
			return ctx.Err()
		default:
			ledgerCloseMeta, err := e.ledgerBackend.GetLedger(ctx, nextLedger)
			if err != nil {
				return errors.Wrapf(err, "failed to retrieve ledger %d from the ledger backend", nextLedger)
			}
			err = e.AddLedgerCloseMeta(ctx, ledgerCloseMeta)
			if err != nil {
				return errors.Wrapf(err, "failed to add ledgerCloseMeta for ledger %d", nextLedger)
			}
		}
	}
	logger.Infof("ExportManager successfully exported ledgers from %d to %d", startLedger, endLedger)
	return nil
}
