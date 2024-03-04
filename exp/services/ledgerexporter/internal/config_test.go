package ledgerexporter

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateStartAndEndLedger(t *testing.T) {
	const latestNetworkLedger = 20000

	config := &Config{
		ExporterConfig: ExporterConfig{
			LedgersPerFile: 1,
		},
	}
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.StartLedger = tt.startLedger
			config.EndLedger = tt.endLedger
			if tt.errMsg != "" {
				require.Equal(t, tt.errMsg, config.validateAndSetLedgerRange(latestNetworkLedger).Error())
			} else {
				require.NoError(t, config.validateAndSetLedgerRange(latestNetworkLedger))
			}
		})
	}
}

func TestAdjustLedgerRangeBoundedMode(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected *Config
	}{
		{
			name:     "Min start ledger 2",
			config:   &Config{StartLedger: 0, EndLedger: 10, ExporterConfig: ExporterConfig{LedgersPerFile: 1}},
			expected: &Config{StartLedger: 2, EndLedger: 10, ExporterConfig: ExporterConfig{LedgersPerFile: 1}},
		},
		{
			name:     "No change, 1 ledger per file",
			config:   &Config{StartLedger: 2, EndLedger: 2, ExporterConfig: ExporterConfig{LedgersPerFile: 1}},
			expected: &Config{StartLedger: 2, EndLedger: 2, ExporterConfig: ExporterConfig{LedgersPerFile: 1}},
		},
		{
			name:     "Min start ledger2, round up end ledger, 10 ledgers per file",
			config:   &Config{StartLedger: 0, EndLedger: 1, ExporterConfig: ExporterConfig{LedgersPerFile: 10}},
			expected: &Config{StartLedger: 2, EndLedger: 10, ExporterConfig: ExporterConfig{LedgersPerFile: 10}},
		},
		{
			name:     "Round down start ledger and round up end ledger, 15 ledgers per file ",
			config:   &Config{StartLedger: 4, EndLedger: 10, ExporterConfig: ExporterConfig{LedgersPerFile: 15}},
			expected: &Config{StartLedger: 2, EndLedger: 15, ExporterConfig: ExporterConfig{LedgersPerFile: 15}},
		},
		{
			name:     "Round down start ledger and round up end ledger, 64 ledgers per file ",
			config:   &Config{StartLedger: 400, EndLedger: 500, ExporterConfig: ExporterConfig{LedgersPerFile: 64}},
			expected: &Config{StartLedger: 384, EndLedger: 512, ExporterConfig: ExporterConfig{LedgersPerFile: 64}},
		},
		{
			name:     "No change, 64 ledger per file",
			config:   &Config{StartLedger: 64, EndLedger: 128, ExporterConfig: ExporterConfig{LedgersPerFile: 64}},
			expected: &Config{StartLedger: 64, EndLedger: 128, ExporterConfig: ExporterConfig{LedgersPerFile: 64}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.adjustLedgerRange()
			require.EqualValues(t, tt.expected.StartLedger, tt.config.StartLedger)
			require.EqualValues(t, tt.expected.EndLedger, tt.config.EndLedger)
		})
	}
}

func TestAdjustLedgerRangeUnBoundedMode(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected *Config
	}{
		{
			name:     "Min start ledger 2",
			config:   &Config{StartLedger: 0, ExporterConfig: ExporterConfig{LedgersPerFile: 1}},
			expected: &Config{StartLedger: 2, ExporterConfig: ExporterConfig{LedgersPerFile: 1}},
		},
		{
			name:     "No change, 1 ledger per file",
			config:   &Config{StartLedger: 2, ExporterConfig: ExporterConfig{LedgersPerFile: 1}},
			expected: &Config{StartLedger: 2, ExporterConfig: ExporterConfig{LedgersPerFile: 1}},
		},
		{
			name:     "Round down start ledger, 15 ledgers per file ",
			config:   &Config{StartLedger: 4, ExporterConfig: ExporterConfig{LedgersPerFile: 15}},
			expected: &Config{StartLedger: 2, ExporterConfig: ExporterConfig{LedgersPerFile: 15}},
		},
		{
			name:     "Round down start ledger, 64 ledgers per file ",
			config:   &Config{StartLedger: 400, ExporterConfig: ExporterConfig{LedgersPerFile: 64}},
			expected: &Config{StartLedger: 384, ExporterConfig: ExporterConfig{LedgersPerFile: 64}},
		},
		{
			name:     "No change, 64 ledger per file",
			config:   &Config{StartLedger: 64, ExporterConfig: ExporterConfig{LedgersPerFile: 64}},
			expected: &Config{StartLedger: 64, ExporterConfig: ExporterConfig{LedgersPerFile: 64}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.adjustLedgerRange()
			require.EqualValues(t, int(tt.expected.StartLedger), int(tt.config.StartLedger))
			require.EqualValues(t, int(tt.expected.EndLedger), int(tt.config.EndLedger))
		})
	}
}
