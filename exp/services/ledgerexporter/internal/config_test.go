package ledgerexporter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/support/errors"
)

func TestNewConfigResumeEnabled(t *testing.T) {
	ctx := context.Background()

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.On("GetRootHAS").Return(historyarchive.HistoryArchiveState{CurrentLedger: 5}, nil).Once()

	config, err := NewConfig(Flags{StartLedger: 1, EndLedger: 2, ConfigFilePath: "test/test.toml", Resume: true})
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

	mockArchive := &historyarchive.MockArchive{}
	mockArchive.On("GetRootHAS").Return(historyarchive.HistoryArchiveState{CurrentLedger: 5}, nil).Once()

	// resume disabled by default
	config, err := NewConfig(Flags{StartLedger: 1, EndLedger: 2, ConfigFilePath: "test/test.toml"})
	require.NoError(t, err)
	require.False(t, config.Resume)
}

func TestInvalidTomlConfig(t *testing.T) {

	_, err := NewConfig(Flags{StartLedger: 1, EndLedger: 2, ConfigFilePath: "test/no_network.toml", Resume: true})
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
			config, err := NewConfig(Flags{StartLedger: tt.startLedger, EndLedger: tt.endLedger, ConfigFilePath: "test/validate_start_end.toml"})
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
			expectedEnd:   9,
		},
		{
			name:          "Round down start ledger and round up end ledger, 15 ledgers per file ",
			configFile:    "test/15perfile.toml",
			start:         4,
			end:           10,
			expectedStart: 2,
			expectedEnd:   14,
		},
		{
			name:          "Round down start ledger and round up end ledger, 64 ledgers per file ",
			configFile:    "test/64perfile.toml",
			start:         400,
			end:           500,
			expectedStart: 384,
			expectedEnd:   511,
		},
		{
			name:          "No change, 64 ledger per file",
			configFile:    "test/64perfile.toml",
			start:         64,
			end:           128,
			expectedStart: 64,
			expectedEnd:   191,
		},
	}

	ctx := context.Background()
	mockArchive := &historyarchive.MockArchive{}
	mockArchive.On("GetRootHAS").Return(historyarchive.HistoryArchiveState{CurrentLedger: 500}, nil).Times(len(tests))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewConfig(Flags{StartLedger: tt.start, EndLedger: tt.end, ConfigFilePath: tt.configFile})
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
			config, err := NewConfig(Flags{StartLedger: tt.start, EndLedger: tt.end, ConfigFilePath: tt.configFile})
			require.NoError(t, err)
			err = config.ValidateAndSetLedgerRange(ctx, mockArchive)
			require.NoError(t, err)
			require.EqualValues(t, tt.expectedStart, config.StartLedger)
			require.EqualValues(t, tt.expectedEnd, config.EndLedger)
		})
	}
}

var cmdOut = ""

func fakeExecCommand(command string, args ...string) *exec.Cmd {
	cs := append([]string{"-test.run=TestExecCmdHelperProcess", "--", command}, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = append(os.Environ(), "GO_EXEC_CMD_HELPER_PROCESS=1", "CMD_OUT="+cmdOut)
	return cmd
}

func TestExecCmdHelperProcess(t *testing.T) {
	if os.Getenv("GO_EXEC_CMD_HELPER_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, os.Getenv("CMD_OUT"))
	os.Exit(0)
}

func TestSetCoreVersionInfo(t *testing.T) {
	tests := []struct {
		name            string
		commandOutput   string
		expectedError   error
		expectedCoreVer string
	}{
		{
			name: "version found",
			commandOutput: "v20.2.0-2-g6e73c0a88\n" +
				"rust version: rustc 1.74.1 (a28077b28 2023-12-04)\n" +
				"soroban-env-host: \n" +
				"    curr:\n" +
				"       package version: 20.2.0\n" +
				"       git version: 1bfc0f2a2ee134efc1e1b0d5270281d0cba61c2e\n" +
				"       ledger protocol version: 20\n" +
				"       pre-release version: 0\n" +
				"       rs-stellar-xdr:\n" +
				"           package version: 20.1.0\n" +
				"           git version: 8b9d623ef40423a8462442b86997155f2c04d3a1\n" +
				"           base XDR git version: b96148cd4acc372cc9af17b909ffe4b12c43ecb6\n",
			expectedError:   nil,
			expectedCoreVer: "v20.2.0-2-g6e73c0a88",
		},
		{
			name:            "core version not found",
			commandOutput:   "",
			expectedError:   errors.New("stellar-core version not found"),
			expectedCoreVer: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{}

			cmdOut = tt.commandOutput
			execCommand = fakeExecCommand
			err := config.setCoreVersionInfo()

			if tt.expectedError != nil {
				require.EqualError(t, err, tt.expectedError.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedCoreVer, config.CoreVersion)
			}
		})
	}
}
