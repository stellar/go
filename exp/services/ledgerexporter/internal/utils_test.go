package ledgerexporter

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/require"
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
				require.Error(t, err)
				require.Empty(t, key)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedKey, key)
			}
		})
	}
}

func createTestLedgerCloseMetaBatch(startSeq, endSeq uint32, count int) xdr.LedgerCloseMetaBatch {
	var ledgerCloseMetas []xdr.LedgerCloseMeta
	for i := 0; i < count; i++ {
		ledgerCloseMetas = append(ledgerCloseMetas, createLedgerCloseMeta(startSeq+uint32(i)))
	}
	return xdr.LedgerCloseMetaBatch{
		StartSequence:    xdr.Uint32(startSeq),
		EndSequence:      xdr.Uint32(endSeq),
		LedgerCloseMetas: ledgerCloseMetas,
	}
}

func TestEncodeDecodeLedgerCloseMetaBatch(t *testing.T) {
	testData := createTestLedgerCloseMetaBatch(1000, 1005, 6)

	// Encode the test data
	var encoder XDRGzipEncoder
	encoder.XdrPayload = testData

	var buf bytes.Buffer
	_, err := encoder.WriteTo(&buf)
	require.NoError(t, err)

	// Decode the encoded data
	var decoder XDRGzipDecoder
	decoder.XdrPayload = &xdr.LedgerCloseMetaBatch{}

	_, err = decoder.ReadFrom(&buf)
	require.NoError(t, err)

	// Check if the decoded data matches the original test data
	decodedData := decoder.XdrPayload.(*xdr.LedgerCloseMetaBatch)
	require.Equal(t, testData.StartSequence, decodedData.StartSequence)
	require.Equal(t, testData.EndSequence, decodedData.EndSequence)
	require.Equal(t, len(testData.LedgerCloseMetas), len(decodedData.LedgerCloseMetas))
	for i := range testData.LedgerCloseMetas {
		require.Equal(t, testData.LedgerCloseMetas[i], decodedData.LedgerCloseMetas[i])
	}
}
