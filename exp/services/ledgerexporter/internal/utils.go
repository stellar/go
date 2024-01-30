package exporter

import (
	"bytes"
	"compress/gzip"
	"fmt"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/storage"
)

const (
	fileSuffix = ".xdr.gzip"
)

// GetObjectKeyFromSequenceNumber generates the file name based on the ledger sequence.
func GetObjectKeyFromSequenceNumber(config ExporterConfig, ledgerSeq uint32) (string, error) {
	var objectKey string

	if config.LedgersPerFile < 1 {
		return "", errors.Errorf("Invalid ledgers per file (%d): must be at least 1", config.LedgersPerFile)
	}

	if config.FilesPerPartition > 1 {
		partitionSize := config.LedgersPerFile * config.FilesPerPartition
		partitionStart := (ledgerSeq / partitionSize) * partitionSize
		partitionEnd := partitionStart + partitionSize - 1
		objectKey = fmt.Sprintf("%d-%d/", partitionStart, partitionEnd)
	}

	fileStart := (ledgerSeq / config.LedgersPerFile) * config.LedgersPerFile
	fileEnd := fileStart + config.LedgersPerFile - 1
	objectKey += fmt.Sprintf("%d", fileStart)

	// Multiple ledgers per file
	if fileStart != fileEnd {
		objectKey += fmt.Sprintf("-%d", fileEnd)
	}
	objectKey += fileSuffix

	return objectKey, nil
}

// GetLatestLedgerSequenceFromHistoryArchives returns the most recent ledger sequence (checkpoint ledger)
// number present in the history archives.
func GetLatestLedgerSequenceFromHistoryArchives(historyArchivesURLs []string) (uint32, error) {
	for _, historyArchiveURL := range historyArchivesURLs {
		ha, err := historyarchive.Connect(
			historyArchiveURL,
			historyarchive.ArchiveOptions{
				ConnectOptions: storage.ConnectOptions{
					UserAgent: "ledger-exporter",
				},
			},
		)
		if err != nil {
			logger.WithError(err).Warnf("Error connecting to history archive %s", historyArchiveURL)
			continue // Skip to next archive
		}

		has, err := ha.GetRootHAS()
		if err != nil {
			logger.WithError(err).Warnf("Error getting RootHAS for %s", historyArchiveURL)
			continue // Skip to next archive
		}

		return has.CurrentLedger, nil
	}

	return 0, errors.New("failed to retrieve the latest ledger sequence from any history archive")
}

// Compress compresses the given data using gzip compression.
func Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		return nil, errors.Wrap(err, "failed to write compressed data")
	}
	if err := w.Close(); err != nil {
		return nil, errors.Wrap(err, "failed to close writer")
	}
	return buf.Bytes(), nil
}
