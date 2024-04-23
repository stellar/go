package ledgerbackend

import (
	"compress/gzip"
	"context"
	"io"
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/stellar/go/support/datastore"
	"github.com/stellar/go/support/ordered"
	"github.com/stellar/go/xdr"
)

// Ensure GCSBackend implements LedgerBackend
var _ LedgerBackend = (*GCSBackend)(nil)

type LCMFileConfig struct {
	StorageURL        string
	FileSuffix        string
	LedgersPerFile    uint32
	FilesPerPartition uint32
}

// GCSBackend is a ledger backend that reads from a cloud storage service.
// The cloud storage service contains files generated from the ledgerExporter.
type GCSBackend struct {
	lcmDataStore      datastore.DataStore
	storageURL        string
	fileSuffix        string
	ledgersPerFile    uint32
	filesPerPartition uint32
}

// Return a new GCSBackend instance.
func NewGCSBackend(ctx context.Context, fileConfig LCMFileConfig) (*GCSBackend, error) {
	lcmDataStore, err := datastore.NewDataStore(ctx, fileConfig.StorageURL)
	if err != nil {
		return nil, err
	}

	cloudStorageBackend := &GCSBackend{
		lcmDataStore:      lcmDataStore,
		storageURL:        fileConfig.StorageURL,
		fileSuffix:        fileConfig.FileSuffix,
		ledgersPerFile:    fileConfig.LedgersPerFile,
		filesPerPartition: fileConfig.FilesPerPartition,
	}

	return cloudStorageBackend, nil
}

// GetLatestLedgerSequence returns the most recent ledger sequence number in the cloud storage bucket.
func (gcsb *GCSBackend) GetLatestLedgerSequence(ctx context.Context) (uint32, error) {
	// Get the latest parition directory from the bucket
	directories, err := gcsb.lcmDataStore.ListDirectoryNames(ctx)
	if err != nil {
		return 0, errors.Wrapf(err, "failed getting list of directory names")
	}

	latestDirectory, err := gcsb.GetLatestDirectory(directories)
	if err != nil {
		return 0, errors.Wrapf(err, "failed getting latest directory")
	}

	// Search through the latest partition to find the latest file which would be the latestLedgerSequence
	fileNames, err := gcsb.lcmDataStore.ListFileNames(ctx, latestDirectory)
	if err != nil {
		return 0, errors.Wrapf(err, "failed getting filenames in dir %s", latestDirectory)
	}

	latestLedgerSequence, err := gcsb.GetLatestFileNameLedgerSequence(fileNames, latestDirectory)
	if err != nil {
		return 0, errors.Wrapf(err, "failed converting filename to ledger sequence")
	}

	return latestLedgerSequence, nil
}

// GetLedger returns the LedgerCloseMeta for the specified ledger sequence number
func (gcsb *GCSBackend) GetLedger(ctx context.Context, sequence uint32) (xdr.LedgerCloseMeta, error) {
	var ledgerCloseMetaBatch xdr.LedgerCloseMetaBatch

	objectKey, err := datastore.GetObjectKeyFromSequenceNumber(sequence, gcsb.ledgersPerFile, gcsb.filesPerPartition, gcsb.fileSuffix)
	if err != nil {
		return xdr.LedgerCloseMeta{}, errors.Wrapf(err, "failed to get object key for ledger %d", sequence)
	}

	reader, err := gcsb.lcmDataStore.GetFile(ctx, objectKey)
	if err != nil {
		return xdr.LedgerCloseMeta{}, errors.Wrapf(err, "failed getting file: %s", objectKey)
	}

	defer reader.Close()

	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return xdr.LedgerCloseMeta{}, errors.Wrapf(err, "failed getting file: %s", objectKey)
	}

	defer gzipReader.Close()

	objectBytes, err := io.ReadAll(gzipReader)
	if err != nil {
		return xdr.LedgerCloseMeta{}, errors.Wrapf(err, "failed reading file: %s", objectKey)
	}

	err = ledgerCloseMetaBatch.UnmarshalBinary(objectBytes)
	if err != nil {
		return xdr.LedgerCloseMeta{}, errors.Wrapf(err, "failed unmarshalling file: %s", objectKey)
	}

	startSequence := uint32(ledgerCloseMetaBatch.StartSequence)
	if startSequence > sequence {
		return xdr.LedgerCloseMeta{}, errors.Errorf("start sequence: %d; greater than sequence to get: %d", startSequence, sequence)
	}

	ledgerCloseMetasIndex := sequence - startSequence
	ledgerCloseMeta := ledgerCloseMetaBatch.LedgerCloseMetas[ledgerCloseMetasIndex]

	return ledgerCloseMeta, nil
}

// PrepareRange checks if the starting and ending (if bounded) ledgers exist.
func (gcsb *GCSBackend) PrepareRange(ctx context.Context, ledgerRange Range) error {
	_, err := gcsb.GetLedger(ctx, ledgerRange.from)
	if err != nil {
		return errors.Wrapf(err, "error getting ledger %d", ledgerRange.from)
	}

	if ledgerRange.bounded {
		_, err := gcsb.GetLedger(ctx, ledgerRange.to)
		if err != nil {
			return errors.Wrapf(err, "error getting ending ledger %d", ledgerRange.to)
		}
	}

	return nil
}

// IsPrepared is a no-op for GCSBackend.
func (gcsb *GCSBackend) IsPrepared(ctx context.Context, ledgerRange Range) (bool, error) {
	return true, nil
}

// Close is a no-op for GCSBackend.
func (gcsb *GCSBackend) Close() error {
	return nil
}

func (gcsb *GCSBackend) GetLatestDirectory(directories []string) (string, error) {
	var latestDirectory string
	largestDirectoryLedger := 0

	for _, dir := range directories {
		// dir follows the format of "ledgers/<network>/<start>-<end>"
		// Need to split the dir string to retrieve the <end> ledger value to get the latest directory
		dirTruncSlash := strings.TrimSuffix(dir, "/")
		_, dirName := path.Split(dirTruncSlash)
		parts := strings.Split(dirName, "-")

		if len(parts) == 2 {
			upper, err := strconv.Atoi(parts[1])
			if err != nil {
				return "", errors.Wrapf(err, "failed getting latest directory %s", dir)
			}

			if upper > largestDirectoryLedger {
				latestDirectory = dir
				largestDirectoryLedger = upper
			}
		}
	}

	return latestDirectory, nil
}

func (gcsb *GCSBackend) GetLatestFileNameLedgerSequence(fileNames []string, directory string) (uint32, error) {
	latestLedgerSequence := uint32(0)

	for _, fileName := range fileNames {
		// fileName follows the format of "ledgers/<network>/<start>-<end>/<ledger_sequence>.<fileSuffix>"
		// Trim the file down to just the <ledger_sequence>
		fileNameTrimExt := strings.TrimSuffix(fileName, gcsb.fileSuffix)
		fileNameTrimPath := strings.TrimPrefix(fileNameTrimExt, directory+"/")
		ledgerSequence, err := strconv.ParseUint(fileNameTrimPath, 10, 32)
		if err != nil {
			return uint32(0), errors.Wrapf(err, "failed converting filename to uint32 %s", fileName)
		}

		latestLedgerSequence = ordered.Max(latestLedgerSequence, uint32(ledgerSequence))
	}

	return latestLedgerSequence, nil
}
