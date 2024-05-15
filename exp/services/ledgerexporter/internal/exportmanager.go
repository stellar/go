package ledgerexporter

import (
	"context"
	"strconv"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/support/datastore"
	"github.com/stellar/go/xdr"
)

type ExportManager struct {
	config             datastore.LedgerBatchConfig
	ledgerBackend      ledgerbackend.LedgerBackend
	currentMetaArchive *datastore.LedgerMetaArchive
	queue              *UploadQueue
	latestLedgerMetric *prometheus.GaugeVec
}

// NewExportManager creates a new ExportManager with the provided configuration.
func NewExportManager(config datastore.LedgerBatchConfig, backend ledgerbackend.LedgerBackend, queue *UploadQueue, prometheusRegistry *prometheus.Registry) (*ExportManager, error) {
	if config.LedgersPerFile < 1 {
		return nil, errors.Errorf("Invalid ledgers per file (%d): must be at least 1", config.LedgersPerFile)
	}

	latestLedgerMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "ledger_exporter", Subsystem: "export_manager", Name: "latest_ledger",
		Help: "sequence number of the latest ledger consumed by the export manager",
	}, []string{"start_ledger", "end_ledger"})
	prometheusRegistry.MustRegister(latestLedgerMetric)

	return &ExportManager{
		config:             config,
		ledgerBackend:      backend,
		queue:              queue,
		latestLedgerMetric: latestLedgerMetric,
	}, nil
}

// AddLedgerCloseMeta adds ledger metadata to the current export object
func (e *ExportManager) AddLedgerCloseMeta(ctx context.Context, ledgerCloseMeta xdr.LedgerCloseMeta) error {
	ledgerSeq := ledgerCloseMeta.LedgerSequence()

	// Determine the object key for the given ledger sequence
	objectKey := e.config.GetObjectKeyFromSequenceNumber(ledgerSeq)

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
		e.currentMetaArchive = datastore.NewLedgerMetaArchive(objectKey, ledgerSeq, endSeq)
	}

	if err := e.currentMetaArchive.AddLedger(ledgerCloseMeta); err != nil {
		return errors.Wrapf(err, "failed to add ledger %d", ledgerSeq)
	}

	if ledgerSeq >= e.currentMetaArchive.GetEndLedgerSequence() {
		// Current archive is full, send it for upload
		if err := e.queue.Enqueue(ctx, e.currentMetaArchive); err != nil {
			return err
		}
		e.currentMetaArchive = nil
	}
	return nil
}

// Run iterates over the specified range of ledgers, retrieves ledger data
// from the backend, and processes the corresponding ledger close metadata.
// The process continues until the ending ledger number is reached or a cancellation
// signal is received.
func (e *ExportManager) Run(ctx context.Context, startLedger, endLedger uint32) error {
	defer e.queue.Close()
	labels := prometheus.Labels{
		"start_ledger": strconv.FormatUint(uint64(startLedger), 10),
		"end_ledger":   strconv.FormatUint(uint64(endLedger), 10),
	}

	var ledgerRange ledgerbackend.Range
	if endLedger < 1 {
		ledgerRange = ledgerbackend.UnboundedRange(startLedger)
	} else {
		ledgerRange = ledgerbackend.BoundedRange(startLedger, endLedger)
	}
	if err := e.ledgerBackend.PrepareRange(ctx, ledgerRange); err != nil {
		return errors.Wrap(err, "Could not prepare captive core ledger backend")
	}

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
			e.latestLedgerMetric.With(labels).Set(float64(nextLedger))
			err = e.AddLedgerCloseMeta(ctx, ledgerCloseMeta)
			if err != nil {
				return errors.Wrapf(err, "failed to add ledgerCloseMeta for ledger %d", nextLedger)
			}
		}
	}
	logger.Infof("ExportManager successfully exported ledgers from %d to %d", startLedger, endLedger)
	return nil
}
