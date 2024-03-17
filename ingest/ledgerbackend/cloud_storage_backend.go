package ledgerbackend

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/stellar/go/support/datastore"
	"github.com/stellar/go/xdr"
)

// Suffix for TxMeta files
const (
	fileSuffix        = ".xdr.gz"
	ledgersPerFile    = 1
	filesPerPartition = 64000
)

// Ensure CloudStorageBackend implements LedgerBackend
var _ LedgerBackend = (*CloudStorageBackend)(nil)

// CloudStorageBackend is a ledger backend that reads from a cloud storage service.
// The cloud storage service contains files generated from the ledgerExporter.
type CloudStorageBackend struct {
	lcmDataStore datastore.DataStore
	storageURL   string
}

// Return a new CloudStorageBackend instance.
func NewCloudStorageBackend(ctx context.Context, storageURL string) (*CloudStorageBackend, error) {
	lcmDataStore, err := datastore.NewDataStore(ctx, storageURL)
	if err != nil {
		return nil, err
	}

	return &CloudStorageBackend{lcmDataStore: lcmDataStore, storageURL: storageURL}, nil
}

// GetLatestLedgerSequence returns the most recent ledger sequence number in the cloud storage bucket.
func (csb *CloudStorageBackend) GetLatestLedgerSequence(ctx context.Context) (uint32, error) {
	// TODO: Can probably copy the code that the ledger exporter will use for resumability
	// Otherwise can use the ListObject function in datastore and find the largest valued filename
	return uint32(0), nil
}

// GetLedger returns the LedgerCloseMeta for the specified ledger sequence number
func (csb *CloudStorageBackend) GetLedger(ctx context.Context, sequence uint32) (xdr.LedgerCloseMeta, error) {
	var ledgerCloseMetaBatch xdr.LedgerCloseMetaBatch

	objectKey, err := GetObjectKeyFromSequenceNumber(sequence, ledgersPerFile, filesPerPartition)
	if err != nil {
		return xdr.LedgerCloseMeta{}, errors.Wrapf(err, "failed to get object key for ledger %d", sequence)
	}

	reader, err := csb.lcmDataStore.GetFile(ctx, objectKey)
	if err != nil {
		return xdr.LedgerCloseMeta{}, errors.Wrapf(err, "failed getting file: %s", objectKey)
	}

	defer reader.Close()

	objectBytes, err := io.ReadAll(reader)
	if err != nil {
		return xdr.LedgerCloseMeta{}, errors.Wrapf(err, "failed reading file: %s", objectKey)
	}

	err = ledgerCloseMetaBatch.UnmarshalBinary(objectBytes)
	if err != nil {
		return xdr.LedgerCloseMeta{}, errors.Wrapf(err, "failed unmarshalling file: %s", objectKey)
	}

	ledgerCloseMetasIndex := sequence - uint32(ledgerCloseMetaBatch.StartSequence)
	ledgerCloseMeta := ledgerCloseMetaBatch.LedgerCloseMetas[ledgerCloseMetasIndex]

	return ledgerCloseMeta, nil
}

// PrepareRange checks if the starting and ending (if bounded) ledgers exist.
func (csb *CloudStorageBackend) PrepareRange(ctx context.Context, ledgerRange Range) error {
	_, err := csb.GetLedger(ctx, ledgerRange.from)
	if err != nil {
		return errors.Wrapf(err, "error getting ledger %d", ledgerRange.from)
	}

	if ledgerRange.bounded {
		_, err := csb.GetLedger(ctx, ledgerRange.to)
		if err != nil {
			return errors.Wrapf(err, "error getting ending ledger %d", ledgerRange.to)
		}
	}

	return nil
}

// IsPrepared is a no-op for CloudStorageBackend.
func (csb *CloudStorageBackend) IsPrepared(ctx context.Context, ledgerRange Range) (bool, error) {
	return true, nil
}

// Close is a no-op for CloudStorageBackend.
func (csb *CloudStorageBackend) Close() error {
	return nil
}

// TODO: Should this function also be modified and added to support/datastore?
// This function should be shared between ledger exporter and this legerbackend reader
func GetObjectKeyFromSequenceNumber(ledgerSeq uint32, ledgersPerFile uint32, filesPerPartition uint32) (string, error) {
	var objectKey string

	if ledgersPerFile < 1 {
		return "", errors.Errorf("Invalid ledgers per file (%d): must be at least 1", ledgersPerFile)
	}

	if filesPerPartition > 1 {
		partitionSize := ledgersPerFile * filesPerPartition
		partitionStart := (ledgerSeq / partitionSize) * partitionSize
		partitionEnd := partitionStart + partitionSize - 1
		objectKey = fmt.Sprintf("%d-%d/", partitionStart, partitionEnd)
	}

	fileStart := (ledgerSeq / ledgersPerFile) * ledgersPerFile
	fileEnd := fileStart + ledgersPerFile - 1
	objectKey += fmt.Sprintf("%d", fileStart)

	// Multiple ledgers per file
	if fileStart != fileEnd {
		objectKey += fmt.Sprintf("-%d", fileEnd)
	}
	objectKey += fileSuffix

	return objectKey, nil
}
