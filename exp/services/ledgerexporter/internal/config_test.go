package ledgerexporter

import (
	"context"
	"fmt"
	"testing"

	"github.com/stellar/go/historyarchive"
	"github.com/stretchr/testify/require"
)

func TestNewConfigResumeEnabled(t *testing.T) {
	ctx := context.Background()

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.On("GetRootHAS").Return(historyarchive.HistoryArchiveState{CurrentLedger: 5}, nil).Once()

	config, err := NewConfig(ctx,
		Flags{StartLedger: 1, EndLedger: 2, ConfigFilePath: "test/test.toml", Resume: true})
	config.ValidateAndSetLedgerRange(ctx, mockArchive)
	require.NoError(t, err)
	require.Equal(t, config.DataStoreConfig.Type, "ABC")
	require.Equal(t, config.LedgerBatchConfig.FilesPerPartition, uint32(1))
	require.Equal(t, config.LedgerBatchConfig.LedgersPerFile, uint32(3))
	require.True(t, config.Resume)
	url, ok := config.DataStoreConfig.Params["destination_bucket_path"]
	require.True(t, ok)
	require.Equal(t, url, "your-bucket-name/subpath")
}

func TestNewConfigResumeDisabled(t *testing.T) {
	ctx := context.Background()

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.On("GetRootHAS").Return(historyarchive.HistoryArchiveState{CurrentLedger: 5}, nil).Once()

	// resume disabled by default
	config, err := NewConfig(ctx,
		Flags{StartLedger: 1, EndLedger: 2, ConfigFilePath: "test/test.toml"})
	require.NoError(t, err)
	require.False(t, config.Resume)
}

func TestInvalidTomlConfig(t *testing.T) {
	ctx := context.Background()

	_, err := NewConfig(ctx,
		Flags{StartLedger: 1, EndLedger: 2, ConfigFilePath: "test/no_network.toml", Resume: true})
	require.ErrorContains(t, err, "Invalid TOML config")
}

func TestValidateStartAndEndLedger(t *testing.T) {
	const latestNetworkLedger = uint32(20000)

	tests := []struct {
		name        string
		startLedger uint32
		endLedger   uint32
		errMsg      string
	}{
		{
			name:        "End ledger same as latest ledger",
			startLedger: 512,
			endLedger:   512,
			errMsg:      "",
		},
		{
			name:        "End ledger greater than start ledger",
			startLedger: 512,
			endLedger:   600,
			errMsg:      "",
		},
		{
			name:        "No end ledger provided, unbounded mode",
			startLedger: 512,
			endLedger:   0,
			errMsg:      "",
		},
		{
			name:        "End ledger before start ledger",
			startLedger: 512,
			endLedger:   2,
			errMsg:      "invalid --end value, must be >= --start",
		},
		{
			name:        "End ledger exceeds latest ledger",
			startLedger: 512,
			endLedger:   latestNetworkLedger + 1,
			errMsg: fmt.Sprintf("--end %d exceeds latest network ledger %d",
				latestNetworkLedger+1, latestNetworkLedger),
		},
		{
			name:        "Start ledger 0",
			startLedger: 0,
			endLedger:   2,
			errMsg:      "",
		},
		{
			name:        "Start ledger exceeds latest ledger",
			startLedger: latestNetworkLedger + 1,
			endLedger:   0,
			errMsg: fmt.Sprintf("--start %d exceeds latest network ledger %d",
				latestNetworkLedger+1, latestNetworkLedger),
		},
	}

	ctx := context.Background()
	mockArchive := &historyarchive.MockArchive{}
	mockArchive.On("GetRootHAS").Return(historyarchive.HistoryArchiveState{CurrentLedger: latestNetworkLedger}, nil).Times(len(tests))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewConfig(ctx,
				Flags{StartLedger: tt.startLedger, EndLedger: tt.endLedger, ConfigFilePath: "test/validate_start_end.toml"})
			require.NoError(t, err)
			err = config.ValidateAndSetLedgerRange(ctx, mockArchive)
			if tt.errMsg != "" {
				require.Equal(t, tt.errMsg, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAdjustedLedgerRangeBoundedMode(t *testing.T) {
	tests := []struct {
		name          string
		configFile    string
		start         uint32
		end           uint32
		expectedStart uint32
		expectedEnd   uint32
	}{
		{
			name:          "Min start ledger 2",
			configFile:    "test/1perfile.toml",
			start:         0,
			end:           10,
			expectedStart: 2,
			expectedEnd:   10,
		},
		{
			name:          "No change, 1 ledger per file",
			configFile:    "test/1perfile.toml",
			start:         2,
			end:           2,
			expectedStart: 2,
			expectedEnd:   2,
		},
		{
			name:          "Min start ledger2, round up end ledger, 10 ledgers per file",
			configFile:    "test/10perfile.toml",
			start:         0,
			end:           1,
			expectedStart: 2,
			expectedEnd:   10,
		},
		{
			name:          "Round down start ledger and round up end ledger, 15 ledgers per file ",
			configFile:    "test/15perfile.toml",
			start:         4,
			end:           10,
			expectedStart: 2,
			expectedEnd:   15,
		},
		{
			name:          "Round down start ledger and round up end ledger, 64 ledgers per file ",
			configFile:    "test/64perfile.toml",
			start:         400,
			end:           500,
			expectedStart: 384,
			expectedEnd:   512,
		},
		{
			name:          "No change, 64 ledger per file",
			configFile:    "test/64perfile.toml",
			start:         64,
			end:           128,
			expectedStart: 64,
			expectedEnd:   128,
		},
	}

	ctx := context.Background()
	mockArchive := &historyarchive.MockArchive{}
	mockArchive.On("GetRootHAS").Return(historyarchive.HistoryArchiveState{CurrentLedger: 500}, nil).Times(len(tests))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewConfig(ctx,
				Flags{StartLedger: tt.start, EndLedger: tt.end, ConfigFilePath: tt.configFile})
			require.NoError(t, err)
			err = config.ValidateAndSetLedgerRange(ctx, mockArchive)
			require.NoError(t, err)
			require.EqualValues(t, tt.expectedStart, config.StartLedger)
			require.EqualValues(t, tt.expectedEnd, config.EndLedger)
		})
	}
}

func TestAdjustedLedgerRangeUnBoundedMode(t *testing.T) {
	tests := []struct {
		name          string
		configFile    string
		start         uint32
		end           uint32
		expectedStart uint32
		expectedEnd   uint32
	}{
		{
			name:          "Min start ledger 2",
			configFile:    "test/1perfile.toml",
			start:         0,
			end:           0,
			expectedStart: 2,
			expectedEnd:   0,
		},
		{
			name:          "No change, 1 ledger per file",
			configFile:    "test/1perfile.toml",
			start:         2,
			end:           0,
			expectedStart: 2,
			expectedEnd:   0,
		},
		{
			name:          "Round down start ledger, 15 ledgers per file ",
			configFile:    "test/15perfile.toml",
			start:         4,
			end:           0,
			expectedStart: 2,
			expectedEnd:   0,
		},
		{
			name:          "Round down start ledger, 64 ledgers per file ",
			configFile:    "test/64perfile.toml",
			start:         400,
			end:           0,
			expectedStart: 384,
			expectedEnd:   0,
		},
		{
			name:          "No change, 64 ledger per file",
			configFile:    "test/64perfile.toml",
			start:         64,
			end:           0,
			expectedStart: 64,
			expectedEnd:   0,
		},
	}

	ctx := context.Background()

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.On("GetRootHAS").Return(historyarchive.HistoryArchiveState{CurrentLedger: 500}, nil).Times(len(tests))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewConfig(ctx,
				Flags{StartLedger: tt.start, EndLedger: tt.end, ConfigFilePath: tt.configFile})
			require.NoError(t, err)
			err = config.ValidateAndSetLedgerRange(ctx, mockArchive)
			require.NoError(t, err)
			require.EqualValues(t, tt.expectedStart, config.StartLedger)
			require.EqualValues(t, tt.expectedEnd, config.EndLedger)
		})
	}
}
