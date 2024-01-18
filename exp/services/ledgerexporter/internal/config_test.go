package exporter

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInvalidEndLedgerBoundedMode(t *testing.T) {
	config := &Config{
		StartLedger: 512,
		EndLedger:   2,
		ExporterConfig: ExporterConfig{
			LedgersPerFile: 1,
		},
	}
	assert.Error(t, validateAndAdjustLedgerRange(config), "invalid end ledger value, must be >= start ledger")
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
			assert.NoError(t, validateAndAdjustLedgerRange(tt.config))
			assert.EqualValues(t, tt.expected.StartLedger, tt.config.StartLedger)
			assert.EqualValues(t, tt.expected.EndLedger, tt.config.EndLedger)
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
			config:   &Config{StartLedger: 0, EndLedger: 0, ExporterConfig: ExporterConfig{LedgersPerFile: 1}},
			expected: &Config{StartLedger: 2, EndLedger: 0, ExporterConfig: ExporterConfig{LedgersPerFile: 1}},
		},
		{
			name:     "No change, 1 ledger per file",
			config:   &Config{StartLedger: 2, EndLedger: 0, ExporterConfig: ExporterConfig{LedgersPerFile: 1}},
			expected: &Config{StartLedger: 2, EndLedger: 0, ExporterConfig: ExporterConfig{LedgersPerFile: 1}},
		},
		{
			name:     "Round down start ledger, 15 ledgers per file ",
			config:   &Config{StartLedger: 4, EndLedger: 0, ExporterConfig: ExporterConfig{LedgersPerFile: 15}},
			expected: &Config{StartLedger: 2, EndLedger: 0, ExporterConfig: ExporterConfig{LedgersPerFile: 15}},
		},
		{
			name:     "Round down start ledger, 64 ledgers per file ",
			config:   &Config{StartLedger: 400, EndLedger: 0, ExporterConfig: ExporterConfig{LedgersPerFile: 64}},
			expected: &Config{StartLedger: 384, EndLedger: 0, ExporterConfig: ExporterConfig{LedgersPerFile: 64}},
		},
		{
			name:     "No change, 64 ledger per file",
			config:   &Config{StartLedger: 64, EndLedger: 0, ExporterConfig: ExporterConfig{LedgersPerFile: 64}},
			expected: &Config{StartLedger: 64, EndLedger: 0, ExporterConfig: ExporterConfig{LedgersPerFile: 64}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NoError(t, validateAndAdjustLedgerRange(tt.config))
			assert.EqualValues(t, tt.expected.StartLedger, tt.config.StartLedger)
			assert.EqualValues(t, tt.expected.EndLedger, tt.config.EndLedger)
		})
	}
}
