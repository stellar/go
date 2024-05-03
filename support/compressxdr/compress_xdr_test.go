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

func TestEncodeDecodeLedgerCloseMetaBatchGzip(t *testing.T) {
	testData := createTestLedgerCloseMetaBatch(1000, 1005, 6)

	// Encode the test data
	encoder, err := NewXDREncoder(GZIP, testData)
	require.NoError(t, err)

	var buf bytes.Buffer
	_, err = encoder.WriteTo(&buf)
	require.NoError(t, err)

	// Decode the encoded data
	lcmBatch := xdr.LedgerCloseMetaBatch{}
	decoder, err := NewXDRDecoder(GZIP, &lcmBatch)
	require.NoError(t, err)

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

func TestDecodeUnzipGzip(t *testing.T) {
	expectedBinary := []byte{0x0, 0x0, 0x0, 0x2, 0x0, 0x0, 0x0, 0x2, 0x0, 0x0, 0x0, 0x0}
	testData := createTestLedgerCloseMetaBatch(2, 2, 1)

	// Encode the test data
	encoder, err := NewXDREncoder(GZIP, testData)
	require.NoError(t, err)

	var buf bytes.Buffer
	_, err = encoder.WriteTo(&buf)
	require.NoError(t, err)

	// Decode the encoded data
	lcmBatch := xdr.LedgerCloseMetaBatch{}
	decoder, err := NewXDRDecoder(GZIP, &lcmBatch)
	require.NoError(t, err)

	binary, err := decoder.Unzip(&buf)
	require.NoError(t, err)

	require.Equal(t, expectedBinary, binary)
}
