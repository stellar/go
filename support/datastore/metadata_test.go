package datastore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetaDataToMap(t *testing.T) {

	var tests = []struct {
		name     string
		metaData MetaData
		expected map[string]string
	}{
		{
			name: "testToMap",
			metaData: MetaData{
				StartLedger:          100,
				EndLedger:            200,
				StartLedgerCloseTime: 123456789,
				EndLedgerCloseTime:   987654321,
				ProtocolVersion:      "3",
				CoreVersion:          "v1.2.3",
				Network:              "testnet",
				CompressionType:      "gzip",
				Version:              "1.0.0",
			},
			expected: map[string]string{
				"x-lexie-start-ledger":            "100",
				"x-lexie-end-ledger":              "200",
				"x-lexie-start-ledger-close-time": "123456789",
				"x-lexie-end-ledger-close-time":   "987654321",
				"x-lexie-protocol-version":        "3",
				"x-lexie-core-version":            "v1.2.3",
				"x-lexie-network":                 "testnet",
				"x-lexie-compression-type":        "gzip",
				"x-lexie-version":                 "1.0.0",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := MetaData{
				StartLedger:          tt.metaData.StartLedger,
				EndLedger:            tt.metaData.EndLedger,
				StartLedgerCloseTime: tt.metaData.StartLedgerCloseTime,
				EndLedgerCloseTime:   tt.metaData.EndLedgerCloseTime,
				ProtocolVersion:      tt.metaData.ProtocolVersion,
				CoreVersion:          tt.metaData.CoreVersion,
				Network:              tt.metaData.Network,
				CompressionType:      tt.metaData.CompressionType,
				Version:              tt.metaData.Version,
			}
			got := m.ToMap()
			require.Equal(t, got, tt.expected)
		})
	}
}

func TestNewMetaDataFromMap(t *testing.T) {
	data := map[string]string{
		"x-lexie-start-ledger":            "100",
		"x-lexie-end-ledger":              "200",
		"x-lexie-start-ledger-close-time": "123456789",
		"x-lexie-end-ledger-close-time":   "987654321",
		"x-lexie-protocol-version":        "3",
		"x-lexie-core-version":            "v1.2.3",
		"x-lexie-network":                 "testnet",
		"x-lexie-compression-type":        "gzip",
		"x-lexie-version":                 "1.0.0",
	}

	expected := MetaData{
		StartLedger:          100,
		EndLedger:            200,
		StartLedgerCloseTime: 123456789,
		EndLedgerCloseTime:   987654321,
		ProtocolVersion:      "3",
		CoreVersion:          "v1.2.3",
		Network:              "testnet",
		CompressionType:      "gzip",
		Version:              "1.0.0",
	}

	got, err := NewMetaDataFromMap(data)
	require.NoError(t, err)
	require.Equal(t, got, expected)
}
