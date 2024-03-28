package ledgerexporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResumability(t *testing.T) {

	tests := []struct {
		name           string
		startLedger    uint32
		endLedger      uint32
		exporterConfig ExporterConfig
		resumeResponse uint32
		networkName    string
	}{
		{
			name:           "End ledger same as start, data store has it",
			startLedger:    4,
			endLedger:      4,
			resumeResponse: 10,
			exporterConfig: ExporterConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
			},
			networkName: "test",
		},
		{
			name:           "End ledger same as start, data store does not have it",
			startLedger:    14,
			endLedger:      14,
			resumeResponse: 10,
			exporterConfig: ExporterConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
			},
			networkName: "test",
		},
		{
			name:           "Data store is beyond boundary aligned start ledger",
			startLedger:    20,
			endLedger:      50,
			resumeResponse: 40,
			exporterConfig: ExporterConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
			},
			networkName: "test",
		},
		{
			name:           "Data store is beyond non boundary aligned start ledger",
			startLedger:    55,
			endLedger:      85,
			resumeResponse: 80,
			exporterConfig: ExporterConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
			},
			networkName: "test",
		},
		{
			name:           "Data store is beyond start and end ledger",
			startLedger:    255,
			endLedger:      275,
			resumeResponse: 280,
			exporterConfig: ExporterConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
			},
			networkName: "test",
		},
		{
			name:           "Data store is not beyond start ledger",
			startLedger:    95,
			endLedger:      125,
			resumeResponse: 90,
			exporterConfig: ExporterConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
			},
			networkName: "test",
		},
		{
			name:           "No start ledger provided",
			startLedger:    0,
			endLedger:      10,
			resumeResponse: 0,
			exporterConfig: ExporterConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
			},
			networkName: "test",
		},
		{
			name:           "No end ledger provided, data store not beyond start",
			startLedger:    145,
			endLedger:      0,
			resumeResponse: 140,
			exporterConfig: ExporterConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
			},
			networkName: "test2",
		},
		{
			name:           "No end ledger provided, data store is beyond start",
			startLedger:    345,
			endLedger:      0,
			resumeResponse: 350,
			exporterConfig: ExporterConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
			},
			networkName: "test3",
		},
		{
			name:           "No end ledger provided, data store is beyond start and network latest",
			startLedger:    405,
			endLedger:      0,
			resumeResponse: 460,
			exporterConfig: ExporterConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
			},
			networkName: "test4",
		},
		{
			name:           "No end ledger provided, start is beyond network latest",
			startLedger:    505,
			endLedger:      0,
			resumeResponse: 0,
			exporterConfig: ExporterConfig{
				FilesPerPartition: uint32(1),
				LedgersPerFile:    uint32(10),
			},
			networkName: "test5",
		},
	}

	ctx := context.Background()

	mockNetworkManager := &MockNetworkManager{}
	mockNetworkManager.On("GetLatestLedgerSequenceFromHistoryArchives", ctx, "test").Return(uint32(1000), nil)
	mockNetworkManager.On("GetLatestLedgerSequenceFromHistoryArchives", ctx, "test2").Return(uint32(180), nil)
	mockNetworkManager.On("GetLatestLedgerSequenceFromHistoryArchives", ctx, "test3").Return(uint32(380), nil)
	mockNetworkManager.On("GetLatestLedgerSequenceFromHistoryArchives", ctx, "test4").Return(uint32(450), nil)
	mockNetworkManager.On("GetLatestLedgerSequenceFromHistoryArchives", ctx, "test5").Return(uint32(500), nil)

	mockDataStore := &MockDataStore{}

	//"End ledger same as start, data store has it"
	mockDataStore.On("Exists", ctx, "0-9.xdr.gz").Return(true, nil).Once()

	//"End ledger same as start, data store does not have it"
	mockDataStore.On("Exists", ctx, "10-19.xdr.gz").Return(false, nil).Once()

	//"Data store is beyond boundary aligned start ledger"
	mockDataStore.On("Exists", ctx, "30-39.xdr.gz").Return(true, nil).Once()
	mockDataStore.On("Exists", ctx, "40-49.xdr.gz").Return(false, nil).Once()

	//"Data store is beyond non boundary aligned start ledger"
	mockDataStore.On("Exists", ctx, "70-79.xdr.gz").Return(true, nil).Once()
	mockDataStore.On("Exists", ctx, "80-89.xdr.gz").Return(false, nil).Once()

	//"Data store is beyond start and end ledger"
	mockDataStore.On("Exists", ctx, "260-269.xdr.gz").Return(true, nil).Once()
	mockDataStore.On("Exists", ctx, "270-279.xdr.gz").Return(true, nil).Once()

	//"Data store is not beyond start ledger"
	mockDataStore.On("Exists", ctx, "110-119.xdr.gz").Return(false, nil).Once()
	mockDataStore.On("Exists", ctx, "100-109.xdr.gz").Return(false, nil).Once()
	mockDataStore.On("Exists", ctx, "90-99.xdr.gz").Return(false, nil).Once()

	//"No end ledger provided, data store not beyond start" uses latest from network="test2"
	mockDataStore.On("Exists", ctx, "160-169.xdr.gz").Return(false, nil).Once()
	mockDataStore.On("Exists", ctx, "150-159.xdr.gz").Return(false, nil).Once()
	mockDataStore.On("Exists", ctx, "140-149.xdr.gz").Return(false, nil).Once()

	//"No end ledger provided, data store is beyond start" uses latest from network="test3"
	mockDataStore.On("Exists", ctx, "360-369.xdr.gz").Return(false, nil).Once()
	mockDataStore.On("Exists", ctx, "350-359.xdr.gz").Return(false, nil).Once()
	mockDataStore.On("Exists", ctx, "340-349.xdr.gz").Return(true, nil).Once()

	//"No end ledger provided, data store is beyond start and network latest" uses latest from network="test4"
	mockDataStore.On("Exists", ctx, "420-429.xdr.gz").Return(true, nil).Once()
	mockDataStore.On("Exists", ctx, "430-439.xdr.gz").Return(true, nil).Once()
	mockDataStore.On("Exists", ctx, "440-449.xdr.gz").Return(true, nil).Once()
	mockDataStore.On("Exists", ctx, "450-459.xdr.gz").Return(true, nil).Once()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resumableManager := NewResumableManager(mockDataStore, tt.exporterConfig, mockNetworkManager, tt.networkName)
			response := resumableManager.FindStartBoundary(ctx, tt.startLedger, tt.endLedger)
			require.Equal(t, tt.resumeResponse, response)
		})
	}

	mockDataStore.AssertExpectations(t)
}
