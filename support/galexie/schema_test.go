package galexie

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/stellar/go/support/datastore"
)

func TestGetObjectKeyFromSequenceNumber(t *testing.T) {
	testCases := []struct {
		filesPerPartition uint32
		ledgerSeq         uint32
		ledgersPerFile    uint32
		expectedKey       string
	}{
		{0, 5, 1, "FFFFFFFA--5.xdr.zst"},
		{0, 5, 10, "FFFFFFFF--0-9.xdr.zst"},
		{2, 10, 100, "FFFFFFFF--0-199/FFFFFFFF--0-99.xdr.zst"},
		{2, 150, 50, "FFFFFF9B--100-199/FFFFFF69--150-199.xdr.zst"},
		{2, 300, 200, "FFFFFFFF--0-399/FFFFFF37--200-399.xdr.zst"},
		{2, 1, 1, "FFFFFFFF--0-1/FFFFFFFE--1.xdr.zst"},
		{4, 10, 100, "FFFFFFFF--0-399/FFFFFFFF--0-99.xdr.zst"},
		{4, 250, 50, "FFFFFF37--200-399/FFFFFF05--250-299.xdr.zst"},
		{1, 300, 200, "FFFFFF37--200-399.xdr.zst"},
		{1, 1, 1, "FFFFFFFE--1.xdr.zst"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("LedgerSeq-%d-LedgersPerFile-%d", tc.ledgerSeq, tc.ledgersPerFile), func(t *testing.T) {
			config := Schema{FilesPerPartition: tc.filesPerPartition, LedgersPerFile: tc.ledgersPerFile}
			key := config.GetObjectKeyFromSequenceNumber(tc.ledgerSeq)
			require.Equal(t, tc.expectedKey, key)
		})
	}
}

func TestGetObjectKeyFromSequenceNumber_ObjectKeyDescOrder(t *testing.T) {
	config := Schema{
		LedgersPerFile:    1,
		FilesPerPartition: 10,
	}
	sequenceCount := 10000
	sequenceMap := make(map[uint32]string)
	keys := make([]uint32, len(sequenceMap))
	count := 0

	// Add 0 and MaxUint32 as edge cases
	sequenceMap[0] = config.GetObjectKeyFromSequenceNumber(0)
	keys = append(keys, 0)
	sequenceMap[math.MaxUint32] = config.GetObjectKeyFromSequenceNumber(math.MaxUint32)
	keys = append(keys, math.MaxUint32)

	for {
		if count >= sequenceCount {
			break
		}
		randSequence := rand.Uint32()
		if _, ok := sequenceMap[randSequence]; ok {
			continue
		}
		sequenceMap[randSequence] = config.GetObjectKeyFromSequenceNumber(randSequence)
		keys = append(keys, randSequence)
		count++
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	prev := sequenceMap[keys[0]]
	for i := 1; i < sequenceCount; i++ {
		curr := sequenceMap[keys[i]]
		if prev <= curr {
			t.Error("sequences not in lexicographic order")
		}
		prev = curr
	}
}

func TestGetLedgerFileExtension(t *testing.T) {
	type tc struct {
		name        string
		files       []string
		listErr     error
		ExpectedExt string
		ExpectedErr error
	}

	cases := []tc{
		{
			name:        "returns zst when first matching schema file is .xdr.zst",
			files:       []string{".config.json", "misc/ignore.txt", "ledger/FFFFFFFF--0.xdr.zst"},
			ExpectedExt: "zst",
			ExpectedErr: nil,
		},
		{
			name:        "returns zstd when first matching schema file is .xdr.zstd",
			files:       []string{".config.json", "something.bin", "ledger/FFFFFFFE--0-999.xdr.zstd"},
			ExpectedExt: "zstd",
			ExpectedErr: nil,
		},
		{
			name:        "ignores manifest and non-matching files and returns ErrNoLedgerFiles",
			files:       []string{".config.json", "random/2025-08-01-0001.zst", "ABCDEF--bad.xdr.gz"},
			ExpectedExt: "",
			ExpectedErr: ErrNoLedgerFiles,
		},
		{
			name:        "no files returns ErrNoLedgerFiles",
			files:       []string{},
			ExpectedExt: "",
			ExpectedErr: ErrNoLedgerFiles,
		},
		{
			name:        "propagates underlying list error",
			files:       []string{},
			listErr:     fmt.Errorf("boom"),
			ExpectedExt: "",
			ExpectedErr: fmt.Errorf("failed to list ledger files: boom"),
		},
		{
			name:        "works with nested paths including partition dir",
			files:       []string{"part/DEADBEEF--0-999/DEADBEEF--0-999.xdr.zst"},
			ExpectedExt: "zst",
			ExpectedErr: nil,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ds := new(datastore.MockDataStore)
			ds.On("ListFilePaths", mock.Anything, "", 0).Return(tt.files, tt.listErr).Once()

			ext, err := GetLedgerFileExtension(context.Background(), ds)
			require.Equal(t, tt.ExpectedExt, ext)

			if tt.ExpectedErr != nil {
				if tt.listErr != nil {
					require.ErrorContains(t, err, tt.ExpectedErr.Error())
				} else {
					require.ErrorIs(t, err, tt.ExpectedErr)
				}
			} else {
				require.NoError(t, err)
			}
			ds.AssertExpectations(t)
		})
	}
}

func TestValidate(t *testing.T) {
	orig := loadSchema
	defer func() { loadSchema = orig }()

	type tc struct {
		name        string
		local       Schema
		remote      Schema
		remoteErr   error
		expectedErr string
		expectedExt string
	}

	cases := []tc{
		{
			name:      "os.ErrNotExist returns nil",
			local:     Schema{LedgersPerFile: 64, FilesPerPartition: 1},
			remoteErr: fmt.Errorf("error fetching file: %w", os.ErrNotExist),
		},
		{
			name:        "mismatch LedgersPerFile returns error",
			local:       Schema{LedgersPerFile: 64, FilesPerPartition: 1},
			remote:      Schema{LedgersPerFile: 100, FilesPerPartition: 1},
			expectedErr: "does not match provided schema",
		},
		{
			name:        "mismatch FilesPerPartition returns error",
			local:       Schema{LedgersPerFile: 64, FilesPerPartition: 5},
			remote:      Schema{LedgersPerFile: 64, FilesPerPartition: 1},
			expectedErr: "does not match provided schema",
		},
		{
			name:        "updates extension when local empty",
			local:       Schema{LedgersPerFile: 64, FilesPerPartition: 1, FileExtension: ""},
			remote:      Schema{LedgersPerFile: 64, FilesPerPartition: 1, FileExtension: "zstd"},
			expectedExt: "zstd",
		},
		{
			name:        "does not update extension when remote empty",
			local:       Schema{LedgersPerFile: 64, FilesPerPartition: 1, FileExtension: "zstd"},
			remote:      Schema{LedgersPerFile: 64, FilesPerPartition: 1, FileExtension: ""},
			expectedExt: "zstd",
		},
		{
			name:        "extension mismatch returns error when remote not empty",
			local:       Schema{LedgersPerFile: 64, FilesPerPartition: 1, FileExtension: "zstd"},
			remote:      Schema{LedgersPerFile: 64, FilesPerPartition: 1, FileExtension: "zst"},
			expectedErr: "does not match provided schema",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			loadSchema = func(ctx context.Context, ds datastore.DataStore) (Schema, error) {
				if tt.remoteErr != nil {
					return Schema{}, tt.remoteErr
				}
				return tt.remote, nil
			}

			cfg := tt.local
			err := cfg.Validate(context.Background(), nil)

			if tt.expectedErr != "" {
				require.ErrorContains(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedExt, cfg.FileExtension)
			}
		})
	}
}

func TestLoadSchema(t *testing.T) {
	origExt := getLedgerFileExtension
	origManifest := getManifest
	defer func() {
		getLedgerFileExtension = origExt
		getManifest = origManifest
	}()

	type tc struct {
		name              string
		ext               string
		extErr            error
		manifest          Manifest
		manifestErr       error
		want              Schema
		expectErrContains string
	}

	cases := []tc{
		{
			name:     "success",
			ext:      "zst",
			manifest: Manifest{LedgersPerBatch: 100, BatchesPerPartition: 2},
			want:     Schema{LedgersPerFile: 100, FilesPerPartition: 2, FileExtension: "zst"},
		},
		{
			name:     "no ledger files -> empty extension",
			extErr:   ErrNoLedgerFiles,
			manifest: Manifest{LedgersPerBatch: 64, BatchesPerPartition: 4},
			want:     Schema{LedgersPerFile: 64, FilesPerPartition: 4, FileExtension: ""},
		},
		{
			name:              "extension error",
			extErr:            fmt.Errorf("list err"),
			manifest:          Manifest{},
			expectErrContains: "unable to determine ledger file extension from data store",
		},
		{
			name:              "manifest error",
			ext:               "zstd",
			manifestErr:       fmt.Errorf("read err"),
			expectErrContains: "failed to read manifest",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			getLedgerFileExtension = func(ctx context.Context, ds datastore.DataStore) (string, error) {
				return tt.ext, tt.extErr
			}
			getManifest = func(ctx context.Context, ds datastore.DataStore) (Manifest, error) {
				return tt.manifest, tt.manifestErr
			}

			schema, err := LoadSchema(context.Background(), nil)

			if tt.expectErrContains != "" {
				require.ErrorContains(t, err, tt.expectErrContains)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, schema)
		})
	}
}
