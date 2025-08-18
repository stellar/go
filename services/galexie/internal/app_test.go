package galexie

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/stellar/go/support/compressxdr"
	"github.com/stellar/go/support/datastore"
)

func TestApplyResumeHasStartError(t *testing.T) {
	ctx := context.Background()
	app := &App{}
	app.config = &Config{StartLedger: 10, EndLedger: 19, Mode: Append}
	mockResumableManager := &datastore.MockResumableManager{}
	mockResumableManager.On("FindStart", ctx, uint32(10), uint32(19)).Return(uint32(0), false, errors.New("start error")).Once()

	err := app.applyResumability(ctx, mockResumableManager)
	require.ErrorContains(t, err, "start error")
	mockResumableManager.AssertExpectations(t)
}

func TestApplyResumeDatastoreComplete(t *testing.T) {
	ctx := context.Background()
	app := &App{}
	app.config = &Config{StartLedger: 10, EndLedger: 19, Mode: Append}
	mockResumableManager := &datastore.MockResumableManager{}
	mockResumableManager.On("FindStart", ctx, uint32(10), uint32(19)).Return(uint32(0), false, nil).Once()

	var alreadyExported *DataAlreadyExportedError
	err := app.applyResumability(ctx, mockResumableManager)
	require.ErrorAs(t, err, &alreadyExported)
	mockResumableManager.AssertExpectations(t)
}

func TestApplyResumeInvalidDataStoreLedgersPerFileBoundary(t *testing.T) {
	ctx := context.Background()
	app := &App{}
	app.config = &Config{
		StartLedger:     3,
		EndLedger:       9,
		Mode:            Append,
		DataStoreConfig: datastore.DataStoreConfig{Schema: datastore.DataStoreSchema{LedgersPerFile: 10, FilesPerPartition: 50}},
	}
	mockResumableManager := &datastore.MockResumableManager{}
	// simulate the datastore has inconsistent data,
	// with last ledger not aligned to starting boundary
	mockResumableManager.On("FindStart", ctx, uint32(3), uint32(9)).Return(uint32(6), true, nil).Once()

	var invalidStore *InvalidDataStoreError
	err := app.applyResumability(ctx, mockResumableManager)
	require.ErrorAs(t, err, &invalidStore)
	mockResumableManager.AssertExpectations(t)
}

func TestApplyResumeWithPartialRemoteDataPresent(t *testing.T) {
	ctx := context.Background()
	app := &App{}
	app.config = &Config{
		StartLedger:     10,
		EndLedger:       99,
		Mode:            Append,
		DataStoreConfig: datastore.DataStoreConfig{Schema: datastore.DataStoreSchema{LedgersPerFile: 10, FilesPerPartition: 50}},
	}
	mockResumableManager := &datastore.MockResumableManager{}
	// simulates a data store that had ledger files populated up to seq=49, so the first absent ledger would be 50
	mockResumableManager.On("FindStart", ctx, uint32(10), uint32(99)).Return(uint32(50), true, nil).Once()

	err := app.applyResumability(ctx, mockResumableManager)
	require.NoError(t, err)
	require.Equal(t, app.config.StartLedger, uint32(50))
	mockResumableManager.AssertExpectations(t)
}

func TestApplyResumeWithNoRemoteDataPresent(t *testing.T) {
	ctx := context.Background()
	app := &App{}
	app.config = &Config{
		StartLedger:     10,
		EndLedger:       99,
		Mode:            Append,
		DataStoreConfig: datastore.DataStoreConfig{Schema: datastore.DataStoreSchema{LedgersPerFile: 10, FilesPerPartition: 50}},
	}
	mockResumableManager := &datastore.MockResumableManager{}
	// simulates a data store that had no data in the requested range
	mockResumableManager.On("FindStart", ctx, uint32(10), uint32(99)).Return(uint32(2), true, nil).Once()

	err := app.applyResumability(ctx, mockResumableManager)
	require.NoError(t, err)
	require.Equal(t, app.config.StartLedger, uint32(2))
	mockResumableManager.AssertExpectations(t)
}

func TestApplyResumeWithNoRemoteDataAndRequestFromGenesis(t *testing.T) {
	// app will coerce config.StartLedger values less than 2 to a min of 2 before applying resumability FindStart
	// app will validate the response from FindStart to ensure datastore is ledgers-per-file aligned
	// config.StartLedger=2 is a special genesis case that shouldn't trigger ledgers-per-file validation error
	ctx := context.Background()
	app := &App{}
	app.config = &Config{
		StartLedger:     2,
		EndLedger:       99,
		Mode:            Append,
		DataStoreConfig: datastore.DataStoreConfig{Schema: datastore.DataStoreSchema{LedgersPerFile: 10, FilesPerPartition: 50}},
	}
	mockResumableManager := &datastore.MockResumableManager{}
	// simulates a data store that had no data in the requested range
	mockResumableManager.On("FindStart", ctx, uint32(2), uint32(99)).Return(uint32(2), true, nil).Once()

	err := app.applyResumability(ctx, mockResumableManager)
	require.NoError(t, err)
	require.Equal(t, app.config.StartLedger, uint32(2))
	mockResumableManager.AssertExpectations(t)
}

func TestValidateExistingFileExtension(t *testing.T) {
	var someOtherError = errors.New("a different error")

	testCases := []struct {
		name         string
		getExtReturn string
		files        []string
		getExtError  error
		expectedErr  error
	}{
		{
			name:        "no files, no error",
			files:       []string{},
			expectedErr: nil,
		},
		{
			name:        "only manifest, no error",
			files:       []string{".config.json"},
			expectedErr: nil,
		},
		{
			name:        "valid schema filename with default extension, no error",
			files:       []string{".config.json", "ledger/FFFFFFFF--0.xdr." + compressxdr.DefaultCompressor.Name()},
			expectedErr: nil,
		},
		{
			name:  "valid schema filename with non-default extension returns error",
			files: []string{".config.json", "ledger/FFFFFFFE--0-999.xdr.zstd"},
			expectedErr: fmt.Errorf("detected older incompatible ledger files in the data store (extension %q). "+
				"Galexie v23.0+ requires starting with an empty datastore", "zstd"),
		},
		{
			name:        "underlying GetLedgerFileExtension error, returns wrapped error",
			getExtError: someOtherError,
			expectedErr: fmt.Errorf("unable to determine ledger file extension from data store: failed to list ledger files: %w", someOtherError),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ds := new(datastore.MockDataStore)
			ds.On("ListFilePaths", context.Background(), "", 0).Return(tc.files, tc.getExtError)

			actualErr := validateExistingFileExtension(context.Background(), ds)

			if tc.expectedErr != nil {
				require.ErrorContains(t, actualErr, tc.expectedErr.Error())
			} else {
				require.NoError(t, actualErr)
			}
			ds.AssertExpectations(t)
		})
	}
}
