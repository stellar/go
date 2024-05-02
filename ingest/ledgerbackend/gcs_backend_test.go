package ledgerbackend

import (
	"context"
	"testing"

	"github.com/stellar/go/support/compressxdr"
	"github.com/stellar/go/support/datastore"
	"github.com/stretchr/testify/assert"
)

func createGCSBackendConfigForTesting() gcsBackendConfig {
	bufferConfig := BufferConfig{
		BufferSize: 1000,
		NumWorkers: 5,
		RetryLimit: 3,
		RetryWait:  5,
	}

	param := make(map[string]string)
	param["destination_bucket_path"] = "testURL"
	dataStoreConfig := datastore.DataStoreConfig{
		Type:   "GCS",
		Params: param,
	}

	ledgerBatchConfig := datastore.LedgerBatchConfig{
		LedgersPerFile:    1,
		FilesPerPartition: 64000,
		FileSuffix:        ".xdr.gz",
	}

	return gcsBackendConfig{
		bufferConfig:      bufferConfig,
		dataStoreConfig:   dataStoreConfig,
		ledgerBatchConfig: ledgerBatchConfig,
		storageUrl:        "testURL",
		network:           "testnet",
		compressionType:   compressxdr.GZIP,
	}
}

func createGCSBackendForTesting() GCSBackend {
	config := createGCSBackendConfigForTesting()
	ctx := context.Background()
	mockDataStore := new(datastore.MockDataStore)
	ledgerMetaArchive := datastore.NewLedgerMetaArchive("", 0, 0)
	decoder, _ := compressxdr.NewXDRDecoder(config.compressionType, nil)

	return GCSBackend{
		config:            config,
		context:           ctx,
		dataStore:         mockDataStore,
		ledgerMetaArchive: ledgerMetaArchive,
		decoder:           decoder,
	}
}

func createLedgerBufferGCSForTesting(ledgerRange Range) *ledgerBufferGCS {
	gcsb := createGCSBackendForTesting()
	ledgerBuffer, _ := gcsb.NewLedgerBuffer(ledgerRange)
	return ledgerBuffer
}

func TestGCSNewLedgerBuffer(t *testing.T) {
	gcsb := createGCSBackendForTesting()
	ledgerRange := BoundedRange(2, 3)

	ledgerBuffer, err := gcsb.NewLedgerBuffer(ledgerRange)
	assert.NoError(t, err)
	assert.Equal(t, uint32(2), ledgerBuffer.currentLedger)
	assert.Equal(t, uint32(2), ledgerBuffer.nextTaskLedger)
	assert.Equal(t, uint32(2), ledgerBuffer.nextLedgerQueueLedger)
	assert.Equal(t, ledgerRange, ledgerBuffer.ledgerRange)
}

func TestGCSPrepareRange(t *testing.T) {
	gcsb := createGCSBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(2, 3)
	gcsb.ledgerBuffer = createLedgerBufferGCSForTesting(ledgerRange)

	err := gcsb.PrepareRange(ctx, ledgerRange)
	assert.NoError(t, err)
	assert.NotNil(t, gcsb.prepared)
}

func TestGCSPrepareRange_AlreadyPrepared(t *testing.T) {
	gcsb := createGCSBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(2, 3)
	gcsb.ledgerBuffer = createLedgerBufferGCSForTesting(ledgerRange)
	gcsb.prepared = &ledgerRange

	err := gcsb.PrepareRange(ctx, ledgerRange)
	assert.NoError(t, err)
}

func TestGCSIsPrepared_Bounded(t *testing.T) {
	gcsb := createGCSBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(3, 4)
	gcsb.ledgerBuffer = createLedgerBufferGCSForTesting(ledgerRange)
	gcsb.PrepareRange(ctx, ledgerRange)

	ok, err := gcsb.IsPrepared(ctx, ledgerRange)
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = gcsb.IsPrepared(ctx, BoundedRange(2, 4))
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = gcsb.IsPrepared(ctx, UnboundedRange(3))
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = gcsb.IsPrepared(ctx, UnboundedRange(2))
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestGCSIsPrepared_Unbounded(t *testing.T) {
	gcsb := createGCSBackendForTesting()
	ctx := context.Background()
	ledgerRange := UnboundedRange(3)
	gcsb.ledgerBuffer = createLedgerBufferGCSForTesting(ledgerRange)
	gcsb.PrepareRange(ctx, ledgerRange)

	ok, err := gcsb.IsPrepared(ctx, ledgerRange)
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = gcsb.IsPrepared(ctx, BoundedRange(3, 4))
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = gcsb.IsPrepared(ctx, BoundedRange(2, 4))
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = gcsb.IsPrepared(ctx, UnboundedRange(4))
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = gcsb.IsPrepared(ctx, UnboundedRange(2))
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestGCSClose(t *testing.T) {
	gcsb := createGCSBackendForTesting()
	ctx := context.Background()
	ledgerRange := UnboundedRange(3)
	gcsb.ledgerBuffer = createLedgerBufferGCSForTesting(ledgerRange)
	gcsb.PrepareRange(ctx, ledgerRange)

	err := gcsb.Close()
	assert.NoError(t, err)
	assert.Equal(t, true, gcsb.closed)
}

func TestGCSGetLedger(t *testing.T) {
	expectedLCM := datastore.CreateLedgerCloseMeta(3)

	gcsb := createGCSBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(3, 3)
	gcsb.ledgerBuffer = createLedgerBufferGCSForTesting(ledgerRange)
	gcsb.PrepareRange(ctx, ledgerRange)

	mockDataStore := new(datastore.MockDataStore)
	gcsb.dataStore = mockDataStore
	gcsb.ledgerBuffer.dataStore = mockDataStore
	mockDataStore.On("GetFile", ctx, "0-63999/3.xdr.gz")

	lcm, err := gcsb.GetLedger(ctx, 2)
	assert.NoError(t, err)
	assert.Equal(t, expectedLCM, lcm)
}
