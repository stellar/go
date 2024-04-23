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
		expectedError     bool
	}{
		{0, 5, 1, ".xdr.gz", "5.xdr.gz", false},
		{0, 5, 10, ".xdr.gz", "0-9.xdr.gz", false},
		{2, 5, 0, ".xdr.gz", "", true},
		{2, 10, 100, ".xdr.gz", "0-199/0-99.xdr.gz", false},
		{2, 150, 50, ".xdr.gz", "100-199/150-199.xdr.gz", false},
		{2, 300, 200, ".xdr.gz", "0-399/200-399.xdr.gz", false},
		{2, 1, 1, ".xdr.gz", "0-1/1.xdr.gz", false},
		{4, 10, 100, ".xdr.gz", "0-399/0-99.xdr.gz", false},
		{4, 250, 50, ".xdr.gz", "200-399/250-299.xdr.gz", false},
		{1, 300, 200, ".xdr.gz", "200-399.xdr.gz", false},
		{1, 1, 1, ".xdr.gz", "1.xdr.gz", false},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("LedgerSeq-%d-LedgersPerFile-%d", tc.ledgerSeq, tc.ledgersPerFile), func(t *testing.T) {
			key, err := GetObjectKeyFromSequenceNumber(tc.ledgerSeq, tc.ledgersPerFile, tc.filesPerPartition, tc.fileSuffix)

			if tc.expectedError {
				require.Error(t, err)
				require.Empty(t, key)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedKey, key)
			}
		})
	}
}
