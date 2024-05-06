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

func createBufferedStorageBackendConfigForTesting() BufferedStorageBackendConfig {
	param := make(map[string]string)
	param["destination_bucket_path"] = "testURL"

	ledgerBatchConfig := datastore.LedgerBatchConfig{
		LedgersPerFile:    1,
		FilesPerPartition: 64000,
		FileSuffix:        ".xdr.gz",
	}

	dataStore := new(datastore.MockDataStore)

	resumableManager := new(datastore.MockResumableManager)

	return BufferedStorageBackendConfig{
		LedgerBatchConfig: ledgerBatchConfig,
		CompressionType:   compressxdr.GZIP,
		DataStore:         dataStore,
		ResumableManager:  resumableManager,
		BufferSize:        100,
		NumWorkers:        5,
		RetryLimit:        3,
		RetryWait:         1,
	}
}

func createBufferedStorageBackendForTesting() BufferedStorageBackend {
	config := createBufferedStorageBackendConfigForTesting()
	ctx := context.Background()
	ledgerMetaArchive := datastore.NewLedgerMetaArchive("", 0, 0)
	decoder, _ := compressxdr.NewXDRDecoder(config.CompressionType, nil)

	return BufferedStorageBackend{
		config:            config,
		context:           ctx,
		dataStore:         config.DataStore,
		ledgerMetaArchive: ledgerMetaArchive,
		decoder:           decoder,
	}
}

func TestNewBufferedStorageBackend(t *testing.T) {
	ctx := context.Background()
	config := createBufferedStorageBackendConfigForTesting()

	bsb, err := NewBufferedStorageBackend(ctx, config)
	assert.NoError(t, err)

	assert.Equal(t, bsb.dataStore, config.DataStore)
	assert.Equal(t, ".xdr.gz", bsb.config.LedgerBatchConfig.FileSuffix)
	assert.Equal(t, uint32(1), bsb.config.LedgerBatchConfig.LedgersPerFile)
	assert.Equal(t, uint32(64000), bsb.config.LedgerBatchConfig.FilesPerPartition)
	assert.Equal(t, uint32(100), bsb.config.BufferSize)
	assert.Equal(t, uint32(5), bsb.config.NumWorkers)
	assert.Equal(t, uint32(3), bsb.config.RetryLimit)
	assert.Equal(t, time.Duration(1), bsb.config.RetryWait)
}

func TestGCSNewLedgerBuffer(t *testing.T) {
	bsb := createBufferedStorageBackendForTesting()
	ledgerRange := BoundedRange(2, 3)

	ledgerBuffer, err := bsb.newLedgerBuffer(ledgerRange)
	assert.NoError(t, err)

	assert.Equal(t, uint32(2), ledgerBuffer.currentLedger)
	assert.Equal(t, uint32(4), ledgerBuffer.nextTaskLedger)
	assert.Equal(t, ledgerRange, ledgerBuffer.ledgerRange)
}

func TestCloudStorageGetLatestLedgerSequence(t *testing.T) {
	startLedger := uint32(3)
	endLedger := uint32(5)
	bsb := createBufferedStorageBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(startLedger, endLedger)

	readCloser1 := createLCMBatchReader(uint32(3), uint32(3), 1)
	readCloser2 := createLCMBatchReader(uint32(4), uint32(4), 1)
	readCloser3 := createLCMBatchReader(uint32(5), uint32(5), 1)

	mockDataStore := new(datastore.MockDataStore)
	bsb.dataStore = mockDataStore
	mockDataStore.On("GetFile", ctx, "0-63999/3.xdr.gz").Return(readCloser1, nil)
	mockDataStore.On("GetFile", ctx, "0-63999/4.xdr.gz").Return(readCloser2, nil)
	mockDataStore.On("GetFile", ctx, "0-63999/5.xdr.gz").Return(readCloser3, nil)

	bsb.PrepareRange(ctx, ledgerRange)
	latestSeq, err := bsb.GetLatestLedgerSequence(ctx)
	assert.NoError(t, err)

	assert.Equal(t, uint32(5), latestSeq)
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

func TestCloudStorageGetLedger_SingleLedgerPerFile(t *testing.T) {
	startLedger := uint32(3)
	endLedger := uint32(5)
	lcmArray := createLCMForTesting(startLedger, endLedger)
	bsb := createBufferedStorageBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(startLedger, endLedger)

	readCloser1 := createLCMBatchReader(uint32(3), uint32(3), 1)
	readCloser2 := createLCMBatchReader(uint32(4), uint32(4), 1)
	readCloser3 := createLCMBatchReader(uint32(5), uint32(5), 1)

	mockDataStore := new(datastore.MockDataStore)
	bsb.dataStore = mockDataStore
	mockDataStore.On("GetFile", ctx, "0-63999/3.xdr.gz").Return(readCloser1, nil)
	mockDataStore.On("GetFile", ctx, "0-63999/4.xdr.gz").Return(readCloser2, nil)
	mockDataStore.On("GetFile", ctx, "0-63999/5.xdr.gz").Return(readCloser3, nil)

	bsb.PrepareRange(ctx, ledgerRange)

	lcm, err := bsb.GetLedger(ctx, uint32(3))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[0], lcm)
	lcm, err = bsb.GetLedger(ctx, uint32(4))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[1], lcm)
	lcm, err = bsb.GetLedger(ctx, uint32(5))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[2], lcm)
}

func TestCloudStorageGetLedger_MultipleLedgerPerFile(t *testing.T) {
	startLedger := uint32(2)
	endLedger := uint32(5)
	lcmArray := createLCMForTesting(startLedger, endLedger)
	bsb := createBufferedStorageBackendForTesting()
	ctx := context.Background()
	bsb.config.LedgerBatchConfig.LedgersPerFile = uint32(2)
	ledgerRange := BoundedRange(startLedger, endLedger)

	readCloser1 := createLCMBatchReader(uint32(2), uint32(3), 2)
	readCloser2 := createLCMBatchReader(uint32(4), uint32(5), 2)

	mockDataStore := new(datastore.MockDataStore)
	bsb.dataStore = mockDataStore
	mockDataStore.On("GetFile", ctx, "0-127999/2-3.xdr.gz").Return(readCloser1, nil)
	mockDataStore.On("GetFile", ctx, "0-127999/4-5.xdr.gz").Return(readCloser2, nil)

	bsb.PrepareRange(ctx, ledgerRange)

	lcm, err := bsb.GetLedger(ctx, uint32(2))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[0], lcm)

	lcm, err = bsb.GetLedger(ctx, uint32(3))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[1], lcm)

	lcm, err = bsb.GetLedger(ctx, uint32(4))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[2], lcm)
}

func TestGCSGetLedger_ErrorPreceedingLedger(t *testing.T) {
	startLedger := uint32(3)
	endLedger := uint32(5)
	lcmArray := createLCMForTesting(startLedger, endLedger)
	bsb := createBufferedStorageBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(startLedger, endLedger)

	readCloser1 := createLCMBatchReader(uint32(3), uint32(3), 1)
	readCloser2 := createLCMBatchReader(uint32(4), uint32(4), 1)
	readCloser3 := createLCMBatchReader(uint32(5), uint32(5), 1)

	mockDataStore := new(datastore.MockDataStore)
	bsb.dataStore = mockDataStore
	mockDataStore.On("GetFile", ctx, "0-63999/3.xdr.gz").Return(readCloser1, nil)
	mockDataStore.On("GetFile", ctx, "0-63999/4.xdr.gz").Return(readCloser2, nil)
	mockDataStore.On("GetFile", ctx, "0-63999/5.xdr.gz").Return(readCloser3, nil)

	bsb.PrepareRange(ctx, ledgerRange)

	lcm, err := bsb.GetLedger(ctx, uint32(3))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[0], lcm)

	_, err = bsb.GetLedger(ctx, uint32(2))
	assert.Error(t, err, "requested sequence preceeds current LedgerCloseMetaBatch")
}

func TestGCSGetLedger_NotPrepared(t *testing.T) {
	bsb := createBufferedStorageBackendForTesting()
	ctx := context.Background()

	_, err := bsb.GetLedger(ctx, uint32(3))
	assert.Error(t, err, "session is not prepared, call PrepareRange first")
}

func TestGCSGetLedger_SequenceNotInBatch(t *testing.T) {
	bsb := createBufferedStorageBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(3, 5)

	bsb.PrepareRange(ctx, ledgerRange)

	_, err := bsb.GetLedger(ctx, uint32(2))
	assert.Error(t, err, "requested sequence preceeds current LedgerRange")

	_, err = bsb.GetLedger(ctx, uint32(6))
	assert.Error(t, err, "requested sequence beyond current LedgerRange")
}

func TestGCSPrepareRange(t *testing.T) {
	bsb := createBufferedStorageBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(2, 3)

	err := bsb.PrepareRange(ctx, ledgerRange)
	assert.NoError(t, err)
	assert.NotNil(t, bsb.prepared)

	// check alreadyPrepared
	err = bsb.PrepareRange(ctx, ledgerRange)
	assert.NoError(t, err)
	assert.NotNil(t, bsb.prepared)
}

func TestGCSIsPrepared_Bounded(t *testing.T) {
	bsb := createBufferedStorageBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(3, 4)
	bsb.PrepareRange(ctx, ledgerRange)

	ok, err := bsb.IsPrepared(ctx, ledgerRange)
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = bsb.IsPrepared(ctx, BoundedRange(2, 4))
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = bsb.IsPrepared(ctx, UnboundedRange(3))
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = bsb.IsPrepared(ctx, UnboundedRange(2))
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestGCSIsPrepared_Unbounded(t *testing.T) {
	bsb := createBufferedStorageBackendForTesting()
	ctx := context.Background()
	ledgerRange := UnboundedRange(3)
	bsb.PrepareRange(ctx, ledgerRange)

	ok, err := bsb.IsPrepared(ctx, ledgerRange)
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = bsb.IsPrepared(ctx, BoundedRange(3, 4))
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = bsb.IsPrepared(ctx, BoundedRange(2, 4))
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = bsb.IsPrepared(ctx, UnboundedRange(4))
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = bsb.IsPrepared(ctx, UnboundedRange(2))
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestGCSClose(t *testing.T) {
	bsb := createBufferedStorageBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(3, 5)
	bsb.PrepareRange(ctx, ledgerRange)

	err := bsb.Close()
	assert.NoError(t, err)
	assert.Equal(t, true, bsb.closed)

	_, err = bsb.GetLatestLedgerSequence(ctx)
	assert.Error(t, err, "gbsbackend is closed; cannot GetLatestLedgerSequence")

	_, err = bsb.GetLedger(ctx, 3)
	assert.Error(t, err, "gbsbackend is closed; cannot GetLedger")

	err = bsb.PrepareRange(ctx, ledgerRange)
	assert.Error(t, err, "gbsbackend is closed; cannot PrepareRange")

	_, err = bsb.IsPrepared(ctx, ledgerRange)
	assert.Error(t, err, "gbsbackend is closed; cannot IsPrepared")
}
