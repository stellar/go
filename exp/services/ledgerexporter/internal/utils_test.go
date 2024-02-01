package exporter

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetObjectKeyFromSequenceNumber(t *testing.T) {
	testCases := []struct {
		filesPerPartition uint32
		ledgerSeq         uint32
		ledgersPerFile    uint32
		expectedKey       string
		expectedError     bool
	}{
		{0, 5, 1, "5.xdr.gz", false},
		{0, 5, 10, "0-9.xdr.gz", false},
		{2, 5, 0, "", true},
		{2, 10, 100, "0-199/0-99.xdr.gz", false},
		{2, 150, 50, "100-199/150-199.xdr.gz", false},
		{2, 300, 200, "0-399/200-399.xdr.gz", false},
		{2, 1, 1, "0-1/1.xdr.gz", false},
		{4, 10, 100, "0-399/0-99.xdr.gz", false},
		{4, 250, 50, "200-399/250-299.xdr.gz", false},
		{1, 300, 200, "200-399.xdr.gz", false},
		{1, 1, 1, "1.xdr.gz", false},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("LedgerSeq-%d-LedgersPerFile-%d", tc.ledgerSeq, tc.ledgersPerFile), func(t *testing.T) {
			config := ExporterConfig{FilesPerPartition: tc.filesPerPartition, LedgersPerFile: tc.ledgersPerFile}
			key, err := GetObjectKeyFromSequenceNumber(config, tc.ledgerSeq)

			if tc.expectedError {
				assert.Error(t, err)
				assert.Empty(t, key)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedKey, key)
			}
		})
	}
}
