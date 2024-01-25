package exporter

import (
	"bytes"
	"compress/gzip"
	"fmt"

	"github.com/stellar/go/support/errors"
)

const (
	fileSuffix = ".xdr.gzip"
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

func Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		return nil, errors.Wrapf(err, "failed to write compressed data")
	}
	if err := w.Close(); err != nil {
		return nil, errors.Wrapf(err, "failed to close writer")
	}
	return buf.Bytes(), nil
}
