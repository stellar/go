package ledgerbackend

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/stellar/go/support/compressxdr"
	"github.com/stellar/go/support/datastore"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func createGCSBackendConfigForTesting() GCSBackendConfig {
	bufferConfig := BufferConfig{
		BufferSize: 100,
		NumWorkers: 1,
		RetryLimit: 3,
		RetryWait:  1,
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

	dataStore := new(datastore.MockDataStore)

	resumableManager := new(datastore.MockResumableManager)

	return GCSBackendConfig{
		BufferConfig:      bufferConfig,
		DataStoreConfig:   dataStoreConfig,
		LedgerBatchConfig: ledgerBatchConfig,
		Network:           "testnet",
		CompressionType:   compressxdr.GZIP,
		DataStore:         dataStore,
		ResumableManager:  resumableManager,
	}
}

func createGCSBackendForTesting() GCSBackend {
	config := createGCSBackendConfigForTesting()
	ctx := context.Background()
	ledgerMetaArchive := datastore.NewLedgerMetaArchive("", 0, 0)
	decoder, _ := compressxdr.NewXDRDecoder(config.CompressionType, nil)

	return GCSBackend{
		config:            config,
		context:           ctx,
		dataStore:         config.DataStore,
		resumableManager:  config.ResumableManager,
		ledgerMetaArchive: ledgerMetaArchive,
		decoder:           decoder,
	}
}

func createGCSLedgerBufferForTesting(ledgerRange Range) *ledgerBufferGCS {
	gcsb := createGCSBackendForTesting()
	ledgerBuffer, _ := gcsb.NewLedgerBuffer(ledgerRange)
	return ledgerBuffer
}

func createReadCloserForTesting() io.ReadCloser {
	var capturedBuf []byte
	reader := bytes.NewReader(capturedBuf)
	return io.NopCloser(reader)
}

func TestNewGCSBackend(t *testing.T) {
	ctx := context.Background()
	config := createGCSBackendConfigForTesting()
	config.LedgerBatchConfig = datastore.LedgerBatchConfig{}
	config.BufferConfig = BufferConfig{}

	gcsb, err := NewGCSBackend(ctx, config)
	assert.NoError(t, err)

	assert.Equal(t, gcsb.dataStore, config.DataStore)
	assert.Equal(t, gcsb.resumableManager, config.ResumableManager)
	assert.Equal(t, ".xdr.gz", gcsb.config.LedgerBatchConfig.FileSuffix)
	assert.Equal(t, uint32(1), gcsb.config.LedgerBatchConfig.LedgersPerFile)
	assert.Equal(t, uint32(64000), gcsb.config.LedgerBatchConfig.FilesPerPartition)
	assert.Equal(t, uint32(1000), gcsb.config.BufferConfig.BufferSize)
	assert.Equal(t, uint32(5), gcsb.config.BufferConfig.NumWorkers)
	assert.Equal(t, uint32(3), gcsb.config.BufferConfig.RetryLimit)
	assert.Equal(t, time.Duration(5), gcsb.config.BufferConfig.RetryWait)
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

func TestGCSGetLatestLedgerSequence(t *testing.T) {
	ctx := context.Background()
	gcsb := createGCSBackendForTesting()
	resumableManager := new(datastore.MockResumableManager)
	gcsb.resumableManager = resumableManager

	resumableManager.On("FindStart", ctx, uint32(2), uint32(0)).Return(uint32(6), true, nil)

	seq, err := gcsb.GetLatestLedgerSequence(ctx)
	assert.NoError(t, err)

	assert.Equal(t, uint32(5), seq)
}

func createLCMForTesting(start, end uint32) []xdr.LedgerCloseMeta {
	var lcmArray []xdr.LedgerCloseMeta
	for i := start; i <= end; i++ {
		lcmArray = append(lcmArray, datastore.CreateLedgerCloseMeta(i))
	}

	return lcmArray
}

func createLCMBatchBinaryForTesting(lcm xdr.LedgerCloseMeta, start uint32, end uint32) []byte {
	lcmBatch := xdr.LedgerCloseMetaBatch{
		StartSequence: xdr.Uint32(start),
		EndSequence:   xdr.Uint32(end),
		LedgerCloseMetas: []xdr.LedgerCloseMeta{
			lcm,
		},
	}
	lcmBatchBinary, _ := lcmBatch.MarshalBinary()
	return lcmBatchBinary
}

func TestGCSGetLedger(t *testing.T) {
	startLedger := uint32(3)
	endLedger := uint32(5)
	lcmArray := createLCMForTesting(startLedger, endLedger)

	gcsb := createGCSBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(startLedger, endLedger)
	readCloser1 := createReadCloserForTesting()
	readCloser2 := createReadCloserForTesting()
	readCloser3 := createReadCloserForTesting()

	mockDataStore := new(datastore.MockDataStore)
	gcsb.dataStore = mockDataStore
	mockDataStore.On("GetFile", ctx, "0-63999/3.xdr.gz").Return(readCloser1, nil)
	mockDataStore.On("GetFile", ctx, "0-63999/4.xdr.gz").Return(readCloser2, nil)
	mockDataStore.On("GetFile", ctx, "0-63999/5.xdr.gz").Return(readCloser3, nil)

	objectBytes1 := createLCMBatchBinaryForTesting(lcmArray[0], uint32(3), uint32(3))
	objectBytes2 := createLCMBatchBinaryForTesting(lcmArray[1], uint32(4), uint32(4))
	objectBytes3 := createLCMBatchBinaryForTesting(lcmArray[2], uint32(5), uint32(5))

	mockDecoder := new(compressxdr.MockXDRDecoder)
	gcsb.decoder = mockDecoder
	mockDecoder.On("Unzip", readCloser1).Return(objectBytes1, nil)
	mockDecoder.On("Unzip", readCloser2).Return(objectBytes2, nil)
	mockDecoder.On("Unzip", readCloser3).Return(objectBytes3, nil)

	gcsb.PrepareRange(ctx, ledgerRange)

	lcm, err := gcsb.GetLedger(ctx, uint32(3))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[0], lcm)

	lcm, err = gcsb.GetLedger(ctx, uint32(4))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[1], lcm)
}

func TestGCSGetLedger_NotPrepared(t *testing.T) {
	gcsb := createGCSBackendForTesting()
	ctx := context.Background()

	_, err := gcsb.GetLedger(ctx, uint32(3))
	assert.Error(t, err, "session is not prepared, call PrepareRange first")
}

func TestGCSGetLedger_SequenceNotInBatch(t *testing.T) {
	gcsb := createGCSBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(3, 5)
	gcsb.ledgerBuffer = createGCSLedgerBufferForTesting(ledgerRange)
	gcsb.PrepareRange(ctx, ledgerRange)
	gcsb.ledgerMetaArchive = datastore.NewLedgerMetaArchive("", 4, 0)

	_, err := gcsb.GetLedger(ctx, uint32(2))
	assert.Error(t, err, "requested sequence preceeds current LedgerRange")

	_, err = gcsb.GetLedger(ctx, uint32(6))
	assert.Error(t, err, "requested sequence beyond current LedgerRange")

	_, err = gcsb.GetLedger(ctx, uint32(3))
	assert.Error(t, err, "requested sequence preceeds current LedgerCloseMetaBatch")
}

func TestGCSPrepareRange(t *testing.T) {
	gcsb := createGCSBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(2, 3)
	gcsb.ledgerBuffer = createGCSLedgerBufferForTesting(ledgerRange)

	err := gcsb.PrepareRange(ctx, ledgerRange)
	assert.NoError(t, err)
	assert.NotNil(t, gcsb.prepared)
}

func TestGCSPrepareRange_AlreadyPrepared(t *testing.T) {
	gcsb := createGCSBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(2, 3)
	gcsb.ledgerBuffer = createGCSLedgerBufferForTesting(ledgerRange)
	gcsb.prepared = &ledgerRange

	err := gcsb.PrepareRange(ctx, ledgerRange)
	assert.NoError(t, err)
}

func TestGCSIsPrepared_Bounded(t *testing.T) {
	gcsb := createGCSBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(3, 4)
	gcsb.ledgerBuffer = createGCSLedgerBufferForTesting(ledgerRange)
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
	gcsb.ledgerBuffer = createGCSLedgerBufferForTesting(ledgerRange)
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
	gcsb.ledgerBuffer = createGCSLedgerBufferForTesting(ledgerRange)
	gcsb.PrepareRange(ctx, ledgerRange)

	err := gcsb.Close()
	assert.NoError(t, err)
	assert.Equal(t, true, gcsb.closed)

	_, err = gcsb.GetLatestLedgerSequence(ctx)
	assert.Error(t, err, "gcsBackend is closed; cannot GetLatestLedgerSequence")

	_, err = gcsb.GetLedger(ctx, 3)
	assert.Error(t, err, "gcsBackend is closed; cannot GetLedger")

	err = gcsb.PrepareRange(ctx, ledgerRange)
	assert.Error(t, err, "gcsBackend is closed; cannot PrepareRange")

	_, err = gcsb.IsPrepared(ctx, ledgerRange)
	assert.Error(t, err, "gcsBackend is closed; cannot IsPrepared")
}
