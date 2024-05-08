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
		{0, 5, 1, ".xdr.gz", "FFFFFFFA--5.xdr.gz"},
		{0, 5, 10, ".xdr.gz", "FFFFFFFF--0-9.xdr.gz"},
		{2, 10, 100, ".xdr.gz", "FFFFFFFF--0-199/FFFFFFFF--0-99.xdr.gz"},
		{2, 150, 50, ".xdr.gz", "FFFFFF9B--100-199/FFFFFF69--150-199.xdr.gz"},
		{2, 300, 200, ".xdr.gz", "FFFFFFFF--0-399/FFFFFF37--200-399.xdr.gz"},
		{2, 1, 1, ".xdr.gz", "FFFFFFFF--0-1/FFFFFFFE--1.xdr.gz"},
		{4, 10, 100, ".xdr.gz", "FFFFFFFF--0-399/FFFFFFFF--0-99.xdr.gz"},
		{4, 250, 50, ".xdr.gz", "FFFFFF37--200-399/FFFFFF05--250-299.xdr.gz"},
		{1, 300, 200, ".xdr.gz", "FFFFFF37--200-399.xdr.gz"},
		{1, 1, 1, ".xdr.gz", "FFFFFFFE--1.xdr.gz"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("LedgerSeq-%d-LedgersPerFile-%d", tc.ledgerSeq, tc.ledgersPerFile), func(t *testing.T) {
			config := LedgerBatchConfig{FilesPerPartition: tc.filesPerPartition, LedgersPerFile: tc.ledgersPerFile, FileSuffix: tc.fileSuffix}
			key := config.GetObjectKeyFromSequenceNumber(tc.ledgerSeq)
			require.Equal(t, tc.expectedKey, key)
		})
	}
}
