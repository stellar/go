package exporter

import (
	"fmt"

	"github.com/stellar/go/support/errors"
)

const (
	fileSuffix = ".xdr"
)

// GetObjectKeyFromSequenceNumber generates the file name based on the ledger sequence.
func GetObjectKeyFromSequenceNumber(config ExporterConfig, ledgerSeq uint32) (string, error) {
	var objectKey string

	if config.FilesPerPartition > 1 {
		partitionSize := config.LedgersPerFile * config.FilesPerPartition
		partitionStart := (ledgerSeq / partitionSize) * partitionSize
		partitionEnd := partitionStart + partitionSize - 1
		objectKey = fmt.Sprintf("%v-%v/", partitionStart, partitionEnd)
	}

	if config.LedgersPerFile < 1 {
		return "", errors.New("Ledgers per file must be at least 1")
	}

	fileStart := (ledgerSeq / config.LedgersPerFile) * config.LedgersPerFile
	fileEnd := fileStart + config.LedgersPerFile - 1
	objectKey += fmt.Sprintf("%v", fileStart)

	// Multiple ledgers per file
	if fileStart != fileEnd {
		objectKey += fmt.Sprintf("-%v", fileEnd)
	}
	objectKey += fileSuffix

	return objectKey, nil
}
