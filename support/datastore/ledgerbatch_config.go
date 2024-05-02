package datastore

import (
	"fmt"
)

type LedgerBatchConfig struct {
	LedgersPerFile    uint32 `toml:"ledgers_per_file"`
	FilesPerPartition uint32 `toml:"files_per_partition"`
	FileSuffix        string `toml:"file_suffix"`
}

func (ec LedgerBatchConfig) GetSequenceNumberStartBoundary(ledgerSeq uint32) uint32 {
	if ec.LedgersPerFile == 0 {
		return 0
	}
	return (ledgerSeq / ec.LedgersPerFile) * ec.LedgersPerFile
}

func (ec LedgerBatchConfig) GetSequenceNumberEndBoundary(ledgerSeq uint32) uint32 {
	return ec.GetSequenceNumberStartBoundary(ledgerSeq) + ec.LedgersPerFile - 1
}

// GetObjectKeyFromSequenceNumber generates the object key name from the ledger sequence number based on configuration.
func (ec LedgerBatchConfig) GetObjectKeyFromSequenceNumber(ledgerSeq uint32) string {
	var objectKey string

	if ec.FilesPerPartition > 1 {
		partitionSize := ec.LedgersPerFile * ec.FilesPerPartition
		partitionStart := (ledgerSeq / partitionSize) * partitionSize
		partitionEnd := partitionStart + partitionSize - 1
		objectKey = fmt.Sprintf("%d-%d/", partitionStart, partitionEnd)
	}

	fileStart := ec.GetSequenceNumberStartBoundary(ledgerSeq)
	fileEnd := ec.GetSequenceNumberEndBoundary(ledgerSeq)
	objectKey += fmt.Sprintf("%d", fileStart)

	// Multiple ledgers per file
	if fileStart != fileEnd {
		objectKey += fmt.Sprintf("-%d", fileEnd)
	}
	objectKey += ec.FileSuffix

	return objectKey
}
