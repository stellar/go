package datastore

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stellar/go/historyarchive"
	"github.com/stretchr/testify/require"
)

func TestResumability(t *testing.T) {

	tests := []struct {
		name              string
		startLedger       uint32
		endLedger         uint32
		ledgerBatchConfig LedgerBatchConfig
		absentLedger      uint32
		findStartOk       bool
		networkName       string
		latestLedger      uint32
		errorSnippet      string
		archiveError      error
	}{
		{
			name:         "archive error when resolving network latest",
			startLedger:  4,
			endLedger:    0,
			absentLedger: 0,
			findStartOk:  false,
			ledgerBatchConfig: LedgerBatchConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
				FileSuffix:        ".xdr.gz",
			},
			networkName:  "test",
			errorSnippet: "archive error",
			archiveError: errors.New("archive error"),
		},
		{
			name:         "End ledger same as start, data store has it",
			startLedger:  4,
			endLedger:    4,
			absentLedger: 0,
			findStartOk:  false,
			ledgerBatchConfig: LedgerBatchConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
				FileSuffix:        ".xdr.gz",
			},
			networkName: "test",
		},
		{
			name:         "End ledger same as start, data store does not have it",
			startLedger:  14,
			endLedger:    14,
			absentLedger: 14,
			findStartOk:  true,
			ledgerBatchConfig: LedgerBatchConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
				FileSuffix:        ".xdr.gz",
			},
			networkName: "test",
		},
		{
			name:         "binary search encounters an error during datastore retrieval",
			startLedger:  24,
			endLedger:    24,
			absentLedger: 0,
			findStartOk:  false,
			ledgerBatchConfig: LedgerBatchConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
				FileSuffix:        ".xdr.gz",
			},
			networkName:  "test",
			errorSnippet: "datastore error happened",
		},
		{
			name:         "Data store is beyond boundary aligned start ledger",
			startLedger:  20,
			endLedger:    50,
			absentLedger: 40,
			findStartOk:  true,
			ledgerBatchConfig: LedgerBatchConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
				FileSuffix:        ".xdr.gz",
			},
			networkName: "test",
		},
		{
			name:         "Data store is beyond non boundary aligned start ledger",
			startLedger:  55,
			endLedger:    85,
			absentLedger: 80,
			findStartOk:  true,
			ledgerBatchConfig: LedgerBatchConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
				FileSuffix:        ".xdr.gz",
			},
			networkName: "test",
		},
		{
			name:         "Data store is beyond start and end ledger",
			startLedger:  255,
			endLedger:    275,
			absentLedger: 0,
			findStartOk:  false,
			ledgerBatchConfig: LedgerBatchConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
				FileSuffix:        ".xdr.gz",
			},
			networkName: "test",
		},
		{
			name:         "Data store is not beyond start ledger",
			startLedger:  95,
			endLedger:    125,
			absentLedger: 95,
			findStartOk:  true,
			ledgerBatchConfig: LedgerBatchConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
				FileSuffix:        ".xdr.gz",
			},
			networkName: "test",
		},
		{
			name:         "No start ledger provided",
			startLedger:  0,
			endLedger:    10,
			absentLedger: 0,
			findStartOk:  false,
			ledgerBatchConfig: LedgerBatchConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
				FileSuffix:        ".xdr.gz",
			},
			networkName:  "test",
			errorSnippet: "Invalid start value",
		},
		{
			name:         "No end ledger provided, data store not beyond start",
			startLedger:  1145,
			endLedger:    0,
			absentLedger: 1145,
			findStartOk:  true,
			ledgerBatchConfig: LedgerBatchConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
				FileSuffix:        ".xdr.gz",
			},
			networkName:  "test2",
			latestLedger: uint32(2000),
		},
		{
			name:         "No end ledger provided, data store is beyond start",
			startLedger:  2145,
			endLedger:    0,
			absentLedger: 2250,
			findStartOk:  true,
			ledgerBatchConfig: LedgerBatchConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
				FileSuffix:        ".xdr.gz",
			},
			networkName:  "test3",
			latestLedger: uint32(3000),
		},
		{
			name:         "No end ledger provided, data store is beyond start and archive network latest, and partially into checkpoint frequency padding",
			startLedger:  3145,
			endLedger:    0,
			absentLedger: 4070,
			findStartOk:  true,
			ledgerBatchConfig: LedgerBatchConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
				FileSuffix:        ".xdr.gz",
			},
			networkName:  "test4",
			latestLedger: uint32(4000),
		},
		{
			name:         "No end ledger provided, start is beyond archive network latest and checkpoint frequency padding",
			startLedger:  5129,
			endLedger:    0,
			absentLedger: 0,
			findStartOk:  false,
			ledgerBatchConfig: LedgerBatchConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
				FileSuffix:        ".xdr.gz",
			},
			networkName:  "test5",
			latestLedger: uint32(5000),
			errorSnippet: "Invalid start value of 5129, it is greater than network's latest ledger of 5128",
		},
	}

	ctx := context.Background()

	mockDataStore := &MockDataStore{}

	//"End ledger same as start, data store has it"
	mockDataStore.On("Exists", ctx, "FFFFFFFF--0-9.xdr.gz").Return(true, nil).Once()

	//"End ledger same as start, data store does not have it"
	mockDataStore.On("Exists", ctx, "FFFFFFF5--10-19.xdr.gz").Return(false, nil).Once()

	//"binary search encounters an error during datastore retrieval",
	mockDataStore.On("Exists", ctx, "FFFFFFEB--20-29.xdr.gz").Return(false, errors.New("datastore error happened")).Once()

	//"Data store is beyond boundary aligned start ledger"
	mockDataStore.On("Exists", ctx, "FFFFFFE1--30-39.xdr.gz").Return(true, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFFFD7--40-49.xdr.gz").Return(false, nil).Once()

	//"Data store is beyond non boundary aligned start ledger"
	mockDataStore.On("Exists", ctx, "FFFFFFB9--70-79.xdr.gz").Return(true, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFFFAF--80-89.xdr.gz").Return(false, nil).Once()

	//"Data store is beyond start and end ledger"
	mockDataStore.On("Exists", ctx, "FFFFFEFB--260-269.xdr.gz").Return(true, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFFEF1--270-279.xdr.gz").Return(true, nil).Once()

	//"Data store is not beyond start ledger"
	mockDataStore.On("Exists", ctx, "FFFFFF91--110-119.xdr.gz").Return(false, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFFF9B--100-109.xdr.gz").Return(false, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFFFA5--90-99.xdr.gz").Return(false, nil).Once()

	//"No end ledger provided, data store not beyond start" uses latest from network="test2"
	mockDataStore.On("Exists", ctx, "FFFFF9A1--1630-1639.xdr.gz").Return(false, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFFA91--1390-1399.xdr.gz").Return(false, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFFB13--1260-1269.xdr.gz").Return(false, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFFB4F--1200-1209.xdr.gz").Return(false, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFFB77--1160-1169.xdr.gz").Return(false, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFFB6D--1170-1179.xdr.gz").Return(false, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFFB81--1150-1159.xdr.gz").Return(false, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFFB8B--1140-1149.xdr.gz").Return(false, nil).Once()

	//"No end ledger provided, data store is beyond start" uses latest from network="test3"
	mockDataStore.On("Exists", ctx, "FFFFF5B9--2630-2639.xdr.gz").Return(false, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFF6A9--2390-2399.xdr.gz").Return(false, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFF72B--2260-2269.xdr.gz").Return(false, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFF735--2250-2259.xdr.gz").Return(false, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFF73F--2240-2249.xdr.gz").Return(true, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFF749--2230-2239.xdr.gz").Return(true, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFF767--2200-2209.xdr.gz").Return(true, nil).Once()

	//"No end ledger provided, data store is beyond start and archive network latest, and partially into checkpoint frequency padding" uses latest from network="test4"
	mockDataStore.On("Exists", ctx, "FFFFF1D1--3630-3639.xdr.gz").Return(true, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFF0D7--3880-3889.xdr.gz").Return(true, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFF05F--4000-4009.xdr.gz").Return(true, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFF023--4060-4069.xdr.gz").Return(true, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFF005--4090-4099.xdr.gz").Return(false, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFF00F--4080-4089.xdr.gz").Return(false, nil).Once()
	mockDataStore.On("Exists", ctx, "FFFFF019--4070-4079.xdr.gz").Return(false, nil).Once()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockArchive := &historyarchive.MockArchive{}
			mockArchive.On("GetRootHAS").Return(historyarchive.HistoryArchiveState{CurrentLedger: tt.latestLedger}, tt.archiveError).Once()

			resumableManager := NewResumableManager(mockDataStore, tt.networkName, tt.ledgerBatchConfig, mockArchive)
			absentLedger, ok, err := resumableManager.FindStart(ctx, tt.startLedger, tt.endLedger)
			if tt.errorSnippet != "" {
				require.ErrorContains(t, err, tt.errorSnippet)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.absentLedger, absentLedger)
			require.Equal(t, tt.findStartOk, ok)
			if tt.endLedger == 0 {
				// archives are only expected to be called when end = 0
				mockArchive.AssertExpectations(t)
			}
		})
	}

	mockDataStore.AssertExpectations(t)
}
