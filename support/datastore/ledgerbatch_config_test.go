package datastore

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetObjectKeyFromSequenceNumber(t *testing.T) {
	testCases := []struct {
		filesPerPartition uint32
		ledgerSeq         uint32
		ledgersPerFile    uint32
		fileSuffix        string
		expectedKey       string
	}{
		{0, 5, 1, ".xdr.gz", "5.xdr.gz"},
		{0, 5, 10, ".xdr.gz", "0-9.xdr.gz"},
		{2, 10, 100, ".xdr.gz", "0-199/0-99.xdr.gz"},
		{2, 150, 50, ".xdr.gz", "100-199/150-199.xdr.gz"},
		{2, 300, 200, ".xdr.gz", "0-399/200-399.xdr.gz"},
		{2, 1, 1, ".xdr.gz", "0-1/1.xdr.gz"},
		{4, 10, 100, ".xdr.gz", "0-399/0-99.xdr.gz"},
		{4, 250, 50, ".xdr.gz", "200-399/250-299.xdr.gz"},
		{1, 300, 200, ".xdr.gz", "200-399.xdr.gz"},
		{1, 1, 1, ".xdr.gz", "1.xdr.gz"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("LedgerSeq-%d-LedgersPerFile-%d", tc.ledgerSeq, tc.ledgersPerFile), func(t *testing.T) {
			config := LedgerBatchConfig{FilesPerPartition: tc.filesPerPartition, LedgersPerFile: tc.ledgersPerFile, FileSuffix: tc.fileSuffix}
			key := config.GetObjectKeyFromSequenceNumber(tc.ledgerSeq)
			require.Equal(t, tc.expectedKey, key)
		})
	}
}
