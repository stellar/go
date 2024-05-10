package compressxdr

import (
	"bytes"
	"testing"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/require"
)

func createTestLedgerCloseMetaBatch(startSeq, endSeq uint32, count int) xdr.LedgerCloseMetaBatch {
	var ledgerCloseMetas []xdr.LedgerCloseMeta
	for i := 0; i < count; i++ {
		//ledgerCloseMetas = append(ledgerCloseMetas, datastore.CreateLedgerCloseMeta(startSeq+uint32(i)))
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
	encoder := NewXDREncoder(DefaultCompressor, testData)

	var buf bytes.Buffer
	_, err := encoder.WriteTo(&buf)
	require.NoError(t, err)

	// Decode the encoded data
	lcmBatch := xdr.LedgerCloseMetaBatch{}
	decoder := NewXDRDecoder(DefaultCompressor, &lcmBatch)

	_, err = decoder.ReadFrom(&buf)
	require.NoError(t, err)

	// Check if the decoded data matches the original test data
	decodedData := lcmBatch
	require.Equal(t, testData.StartSequence, decodedData.StartSequence)
	require.Equal(t, testData.EndSequence, decodedData.EndSequence)
	require.Equal(t, len(testData.LedgerCloseMetas), len(decodedData.LedgerCloseMetas))
	for i := range testData.LedgerCloseMetas {
		require.Equal(t, testData.LedgerCloseMetas[i], decodedData.LedgerCloseMetas[i])
	}
}
