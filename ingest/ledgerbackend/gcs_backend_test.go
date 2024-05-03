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
		NumWorkers: 5,
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

func createTestLedgerCloseMetaBatch(startSeq, endSeq uint32, count int) xdr.LedgerCloseMetaBatch {
	var ledgerCloseMetas []xdr.LedgerCloseMeta
	for i := 0; i < count; i++ {
		ledgerCloseMetas = append(ledgerCloseMetas, datastore.CreateLedgerCloseMeta(startSeq+uint32(i)))
	}
	return xdr.LedgerCloseMetaBatch{
		StartSequence:    xdr.Uint32(startSeq),
		EndSequence:      xdr.Uint32(endSeq),
		LedgerCloseMetas: ledgerCloseMetas,
	}
}

func createLCMBatchReader(start, end uint32, count int) io.ReadCloser {
	testData := createTestLedgerCloseMetaBatch(start, end, count)
	encoder, _ := compressxdr.NewXDREncoder(compressxdr.GZIP, testData)
	var buf bytes.Buffer
	encoder.WriteTo(&buf)
	capturedBuf := buf.Bytes()
	reader1 := bytes.NewReader(capturedBuf)
	return io.NopCloser(reader1)
}

func TestGCSGetLedger_SingleLedgerPerFile(t *testing.T) {
	startLedger := uint32(3)
	endLedger := uint32(5)
	lcmArray := createLCMForTesting(startLedger, endLedger)
	gcsb := createGCSBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(startLedger, endLedger)

	readCloser1 := createLCMBatchReader(uint32(3), uint32(3), 1)
	readCloser2 := createLCMBatchReader(uint32(4), uint32(4), 1)
	readCloser3 := createLCMBatchReader(uint32(5), uint32(5), 1)

	mockDataStore := new(datastore.MockDataStore)
	gcsb.dataStore = mockDataStore
	mockDataStore.On("GetFile", ctx, "0-63999/3.xdr.gz").Return(readCloser1, nil)
	mockDataStore.On("GetFile", ctx, "0-63999/4.xdr.gz").Return(readCloser2, nil)
	mockDataStore.On("GetFile", ctx, "0-63999/5.xdr.gz").Return(readCloser3, nil)

	gcsb.PrepareRange(ctx, ledgerRange)

	lcm, err := gcsb.GetLedger(ctx, uint32(3))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[0], lcm)
	// Skip sequence 4; Test non consecutive GetLedger
	lcm, err = gcsb.GetLedger(ctx, uint32(5))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[2], lcm)
}

func TestGCSGetLedger_MultipleLedgerPerFile(t *testing.T) {
	startLedger := uint32(2)
	endLedger := uint32(5)
	lcmArray := createLCMForTesting(startLedger, endLedger)
	gcsb := createGCSBackendForTesting()
	ctx := context.Background()
	gcsb.config.LedgerBatchConfig.LedgersPerFile = uint32(2)
	ledgerRange := BoundedRange(startLedger, endLedger)

	readCloser1 := createLCMBatchReader(uint32(2), uint32(3), 2)
	readCloser2 := createLCMBatchReader(uint32(4), uint32(5), 2)

	mockDataStore := new(datastore.MockDataStore)
	gcsb.dataStore = mockDataStore
	mockDataStore.On("GetFile", ctx, "0-127999/2-3.xdr.gz").Return(readCloser1, nil)
	mockDataStore.On("GetFile", ctx, "0-127999/4-5.xdr.gz").Return(readCloser2, nil)

	gcsb.PrepareRange(ctx, ledgerRange)

	lcm, err := gcsb.GetLedger(ctx, uint32(2))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[0], lcm)

	lcm, err = gcsb.GetLedger(ctx, uint32(3))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[1], lcm)

	lcm, err = gcsb.GetLedger(ctx, uint32(4))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[2], lcm)
}

func TestGCSGetLedger_ErrorPreceedingLedger(t *testing.T) {
	startLedger := uint32(3)
	endLedger := uint32(5)
	lcmArray := createLCMForTesting(startLedger, endLedger)
	gcsb := createGCSBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(startLedger, endLedger)

	readCloser1 := createLCMBatchReader(uint32(3), uint32(3), 1)
	readCloser2 := createLCMBatchReader(uint32(4), uint32(4), 1)
	readCloser3 := createLCMBatchReader(uint32(5), uint32(5), 1)

	mockDataStore := new(datastore.MockDataStore)
	gcsb.dataStore = mockDataStore
	mockDataStore.On("GetFile", ctx, "0-63999/3.xdr.gz").Return(readCloser1, nil)
	mockDataStore.On("GetFile", ctx, "0-63999/4.xdr.gz").Return(readCloser2, nil)
	mockDataStore.On("GetFile", ctx, "0-63999/5.xdr.gz").Return(readCloser3, nil)

	gcsb.PrepareRange(ctx, ledgerRange)

	lcm, err := gcsb.GetLedger(ctx, uint32(5))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[2], lcm)

	_, err = gcsb.GetLedger(ctx, uint32(4))
	assert.Error(t, err, "requested sequence preceeds current LedgerCloseMetaBatch")
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

	err := gcsb.PrepareRange(ctx, ledgerRange)
	assert.NoError(t, err)
	assert.NotNil(t, gcsb.prepared)
}

func TestGCSPrepareRange_AlreadyPrepared(t *testing.T) {
	gcsb := createGCSBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(2, 3)
	gcsb.prepared = &ledgerRange

	err := gcsb.PrepareRange(ctx, ledgerRange)
	assert.NoError(t, err)
}

func TestGCSIsPrepared_Bounded(t *testing.T) {
	gcsb := createGCSBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(3, 4)
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
