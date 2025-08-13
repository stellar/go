package galexie

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/stellar/go/support/compressxdr"
	"github.com/stellar/go/support/datastore"
)

// Function indirection variables to simplify unit testing.
// Tests can override these to mock internal calls.
var (
	loadSchema             = LoadSchema
	getLedgerFileExtension = GetLedgerFileExtension
	getManifest            = GetManifest
)

// ledgerFilenameRe is the regular expression that matches filenames produced by
// Schema.GetObjectKeyFromSequenceNumber (base name only).
// Examples:
//
//	FFFFFFFF--0.xdr.zstd
//	FFFFFFFE--0-999.xdr.zst
var ledgerFilenameRe = regexp.MustCompile(`^[0-9A-F]{8}--[0-9]+(?:-[0-9]+)?\.xdr\.[A-Za-z0-9._-]+$`)

type Schema struct {
	LedgersPerFile    uint32 `toml:"ledgers_per_file"`
	FilesPerPartition uint32 `toml:"files_per_partition"`
	FileExtension     string // Optional – for backward (zstd) compatibility only
}

func (ec Schema) GetSequenceNumberStartBoundary(ledgerSeq uint32) uint32 {
	if ec.LedgersPerFile == 0 {
		return 0
	}
	return (ledgerSeq / ec.LedgersPerFile) * ec.LedgersPerFile
}

func (ec Schema) GetSequenceNumberEndBoundary(ledgerSeq uint32) uint32 {
	return ec.GetSequenceNumberStartBoundary(ledgerSeq) + ec.LedgersPerFile - 1
}

// GetObjectKeyFromSequenceNumber generates the object key name from the ledger sequence number based on configuration.
func (ec Schema) GetObjectKeyFromSequenceNumber(ledgerSeq uint32) string {
	var objectKey string

	if ec.FilesPerPartition > 1 {
		partitionSize := ec.LedgersPerFile * ec.FilesPerPartition
		partitionStart := (ledgerSeq / partitionSize) * partitionSize
		partitionEnd := partitionStart + partitionSize - 1

		objectKey = fmt.Sprintf("%08X--%d-%d/", math.MaxUint32-partitionStart, partitionStart, partitionEnd)
	}

	fileStart := ec.GetSequenceNumberStartBoundary(ledgerSeq)
	fileEnd := ec.GetSequenceNumberEndBoundary(ledgerSeq)
	objectKey += fmt.Sprintf("%08X--%d", math.MaxUint32-fileStart, fileStart)

	// Multiple ledgers per file
	if fileStart != fileEnd {
		objectKey += fmt.Sprintf("-%d", fileEnd)
	}

	if ec.FileExtension == "" {
		ec.FileExtension = compressxdr.DefaultCompressor.Name()
	}

	objectKey += fmt.Sprintf(".xdr.%s", ec.FileExtension)

	return objectKey
}

func (ec *Schema) Validate(ctx context.Context, dataStore datastore.DataStore) error {
	remote, err := loadSchema(ctx, dataStore)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to retrieve datastore schema: %w", err)
	} else if remote.LedgersPerFile != ec.LedgersPerFile {
		return fmt.Errorf("datastore schema %v does not match provided schema %v", remote, *ec)
	} else if remote.FilesPerPartition != ec.FilesPerPartition {
		return fmt.Errorf("datastore schema %v does not match provided schema %v", remote, *ec)
	} else if ec.FileExtension == "" {
		ec.FileExtension = remote.FileExtension
	} else if remote.FileExtension != "" && ec.FileExtension != remote.FileExtension {
		return fmt.Errorf("datastore schema %v does not match provided schema %v", remote, *ec)
	}
	return nil
}

// LoadSchema reads the manifest from the given DataStore and returns its schema configuration.
func LoadSchema(ctx context.Context, dataStore datastore.DataStore) (Schema, error) {
	fileExt, err := getLedgerFileExtension(ctx, dataStore)
	if err != nil && !errors.Is(err, ErrNoLedgerFiles) {
		return Schema{}, fmt.Errorf("unable to determine ledger file extension from data store: %w", err)
	}

	manifest, err := getManifest(ctx, dataStore)
	if err != nil {
		return Schema{}, fmt.Errorf("failed to read manifest: %w", err)
	}

	return Schema{
		LedgersPerFile:    manifest.LedgersPerBatch,
		FilesPerPartition: manifest.BatchesPerPartition,
		FileExtension:     fileExt,
	}, nil
}

var ErrNoLedgerFiles = errors.New("no ledger files found")

func GetLedgerFileExtension(ctx context.Context, dataStore datastore.DataStore) (string, error) {
	files, err := dataStore.ListFilePaths(ctx, "", 0)
	if err != nil {
		return "", fmt.Errorf("failed to list ledger files: %w", err)
	}

	// Note: The file may be inside a partition directory; we only check the base name here.
	for _, file := range files {
		base := filepath.Base(file)
		if ledgerFilenameRe.MatchString(base) {
			// Extract the extension after the last dot using filepath.Ext
			// e.g., "...xdr.zst" -> ".zst" -> "zst"
			ext := strings.TrimPrefix(filepath.Ext(base), ".")
			if ext != "" {
				return ext, nil
			}
		}
	}

	return "", ErrNoLedgerFiles
}
