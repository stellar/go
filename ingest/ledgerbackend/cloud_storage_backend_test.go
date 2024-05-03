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

func createCloudStorageBackendConfigForTesting() CloudStorageBackendConfig {
	param := make(map[string]string)
	param["destination_bucket_path"] = "testURL"

	ledgerBatchConfig := datastore.LedgerBatchConfig{
		LedgersPerFile:    1,
		FilesPerPartition: 64000,
		FileSuffix:        ".xdr.gz",
	}

	dataStore := new(datastore.MockDataStore)

	resumableManager := new(datastore.MockResumableManager)

	return CloudStorageBackendConfig{
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

func createCloudStorageBackendForTesting() CloudStorageBackend {
	config := createCloudStorageBackendConfigForTesting()
	ctx := context.Background()
	ledgerMetaArchive := datastore.NewLedgerMetaArchive("", 0, 0)
	decoder, _ := compressxdr.NewXDRDecoder(config.CompressionType, nil)

	return CloudStorageBackend{
		config:            config,
		context:           ctx,
		dataStore:         config.DataStore,
		resumableManager:  config.ResumableManager,
		ledgerMetaArchive: ledgerMetaArchive,
		decoder:           decoder,
	}
}

func TestNewCloudStorageBackend(t *testing.T) {
	ctx := context.Background()
	config := createCloudStorageBackendConfigForTesting()

	csb, err := NewCloudStorageBackend(ctx, config)
	assert.NoError(t, err)

	assert.Equal(t, csb.dataStore, config.DataStore)
	assert.Equal(t, csb.resumableManager, config.ResumableManager)
	assert.Equal(t, ".xdr.gz", csb.config.LedgerBatchConfig.FileSuffix)
	assert.Equal(t, uint32(1), csb.config.LedgerBatchConfig.LedgersPerFile)
	assert.Equal(t, uint32(64000), csb.config.LedgerBatchConfig.FilesPerPartition)
	assert.Equal(t, uint32(100), csb.config.BufferSize)
	assert.Equal(t, uint32(5), csb.config.NumWorkers)
	assert.Equal(t, uint32(3), csb.config.RetryLimit)
	assert.Equal(t, time.Duration(1), csb.config.RetryWait)
}

func TestGCSNewLedgerBuffer(t *testing.T) {
	csb := createCloudStorageBackendForTesting()
	ledgerRange := BoundedRange(2, 3)

	ledgerBuffer, err := csb.newLedgerBuffer(ledgerRange)
	assert.NoError(t, err)

	assert.Equal(t, uint32(2), ledgerBuffer.currentLedger)
	assert.Equal(t, uint32(2), ledgerBuffer.nextTaskLedger)
	assert.Equal(t, ledgerRange, ledgerBuffer.ledgerRange)
}

func TestCloudStorageGetLatestLedgerSequence(t *testing.T) {
	ctx := context.Background()
	csb := createCloudStorageBackendForTesting()
	resumableManager := new(datastore.MockResumableManager)
	csb.resumableManager = resumableManager

	resumableManager.On("FindStart", ctx, uint32(2), uint32(0)).Return(uint32(6), true, nil)

	seq, err := csb.GetLatestLedgerSequence(ctx)
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

func TestCloudStorageGetLedger_SingleLedgerPerFile(t *testing.T) {
	startLedger := uint32(3)
	endLedger := uint32(5)
	lcmArray := createLCMForTesting(startLedger, endLedger)
	csb := createCloudStorageBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(startLedger, endLedger)

	readCloser1 := createLCMBatchReader(uint32(3), uint32(3), 1)
	readCloser2 := createLCMBatchReader(uint32(4), uint32(4), 1)
	readCloser3 := createLCMBatchReader(uint32(5), uint32(5), 1)

	mockDataStore := new(datastore.MockDataStore)
	csb.dataStore = mockDataStore
	mockDataStore.On("GetFile", ctx, "0-63999/3.xdr.gz").Return(readCloser1, nil)
	mockDataStore.On("GetFile", ctx, "0-63999/4.xdr.gz").Return(readCloser2, nil)
	mockDataStore.On("GetFile", ctx, "0-63999/5.xdr.gz").Return(readCloser3, nil)

	csb.PrepareRange(ctx, ledgerRange)

	lcm, err := csb.GetLedger(ctx, uint32(3))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[0], lcm)
	// Skip sequence 4; Test non consecutive GetLedger
	lcm, err = csb.GetLedger(ctx, uint32(5))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[2], lcm)
}

func TestCloudStorageGetLedger_MultipleLedgerPerFile(t *testing.T) {
	startLedger := uint32(2)
	endLedger := uint32(5)
	lcmArray := createLCMForTesting(startLedger, endLedger)
	csb := createCloudStorageBackendForTesting()
	ctx := context.Background()
	csb.config.LedgerBatchConfig.LedgersPerFile = uint32(2)
	ledgerRange := BoundedRange(startLedger, endLedger)

	readCloser1 := createLCMBatchReader(uint32(2), uint32(3), 2)
	readCloser2 := createLCMBatchReader(uint32(4), uint32(5), 2)

	mockDataStore := new(datastore.MockDataStore)
	csb.dataStore = mockDataStore
	mockDataStore.On("GetFile", ctx, "0-127999/2-3.xdr.gz").Return(readCloser1, nil)
	mockDataStore.On("GetFile", ctx, "0-127999/4-5.xdr.gz").Return(readCloser2, nil)

	csb.PrepareRange(ctx, ledgerRange)

	lcm, err := csb.GetLedger(ctx, uint32(2))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[0], lcm)

	lcm, err = csb.GetLedger(ctx, uint32(3))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[1], lcm)

	lcm, err = csb.GetLedger(ctx, uint32(4))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[2], lcm)
}

func TestGCSGetLedger_ErrorPreceedingLedger(t *testing.T) {
	startLedger := uint32(3)
	endLedger := uint32(5)
	lcmArray := createLCMForTesting(startLedger, endLedger)
	csb := createCloudStorageBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(startLedger, endLedger)

	readCloser1 := createLCMBatchReader(uint32(3), uint32(3), 1)
	readCloser2 := createLCMBatchReader(uint32(4), uint32(4), 1)
	readCloser3 := createLCMBatchReader(uint32(5), uint32(5), 1)

	mockDataStore := new(datastore.MockDataStore)
	csb.dataStore = mockDataStore
	mockDataStore.On("GetFile", ctx, "0-63999/3.xdr.gz").Return(readCloser1, nil)
	mockDataStore.On("GetFile", ctx, "0-63999/4.xdr.gz").Return(readCloser2, nil)
	mockDataStore.On("GetFile", ctx, "0-63999/5.xdr.gz").Return(readCloser3, nil)

	csb.PrepareRange(ctx, ledgerRange)

	lcm, err := csb.GetLedger(ctx, uint32(5))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[2], lcm)

	_, err = csb.GetLedger(ctx, uint32(4))
	assert.Error(t, err, "requested sequence preceeds current LedgerCloseMetaBatch")
}

func TestGCSGetLedger_NotPrepared(t *testing.T) {
	csb := createCloudStorageBackendForTesting()
	ctx := context.Background()

	_, err := csb.GetLedger(ctx, uint32(3))
	assert.Error(t, err, "session is not prepared, call PrepareRange first")
}

func TestGCSGetLedger_SequenceNotInBatch(t *testing.T) {
	csb := createCloudStorageBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(3, 5)

	csb.PrepareRange(ctx, ledgerRange)

	_, err := csb.GetLedger(ctx, uint32(2))
	assert.Error(t, err, "requested sequence preceeds current LedgerRange")

	_, err = csb.GetLedger(ctx, uint32(6))
	assert.Error(t, err, "requested sequence beyond current LedgerRange")
}

func TestGCSPrepareRange(t *testing.T) {
	csb := createCloudStorageBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(2, 3)

	err := csb.PrepareRange(ctx, ledgerRange)
	assert.NoError(t, err)
	assert.NotNil(t, csb.prepared)

	// check alreadyPrepared
	err = csb.PrepareRange(ctx, ledgerRange)
	assert.NoError(t, err)
	assert.NotNil(t, csb.prepared)
}

func TestGCSIsPrepared_Bounded(t *testing.T) {
	csb := createCloudStorageBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(3, 4)
	csb.PrepareRange(ctx, ledgerRange)

	ok, err := csb.IsPrepared(ctx, ledgerRange)
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = csb.IsPrepared(ctx, BoundedRange(2, 4))
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = csb.IsPrepared(ctx, UnboundedRange(3))
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = csb.IsPrepared(ctx, UnboundedRange(2))
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestGCSIsPrepared_Unbounded(t *testing.T) {
	csb := createCloudStorageBackendForTesting()
	ctx := context.Background()
	ledgerRange := UnboundedRange(3)
	csb.PrepareRange(ctx, ledgerRange)

	ok, err := csb.IsPrepared(ctx, ledgerRange)
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = csb.IsPrepared(ctx, BoundedRange(3, 4))
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = csb.IsPrepared(ctx, BoundedRange(2, 4))
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = csb.IsPrepared(ctx, UnboundedRange(4))
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = csb.IsPrepared(ctx, UnboundedRange(2))
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestGCSClose(t *testing.T) {
	csb := createCloudStorageBackendForTesting()
	ctx := context.Background()
	ledgerRange := BoundedRange(3, 5)
	csb.PrepareRange(ctx, ledgerRange)

	err := csb.Close()
	assert.NoError(t, err)
	assert.Equal(t, true, csb.closed)

	_, err = csb.GetLatestLedgerSequence(ctx)
	assert.Error(t, err, "gcsBackend is closed; cannot GetLatestLedgerSequence")

	_, err = csb.GetLedger(ctx, 3)
	assert.Error(t, err, "gcsBackend is closed; cannot GetLedger")

	err = csb.PrepareRange(ctx, ledgerRange)
	assert.Error(t, err, "gcsBackend is closed; cannot PrepareRange")

	_, err = csb.IsPrepared(ctx, ledgerRange)
	assert.Error(t, err, "gcsBackend is closed; cannot IsPrepared")
}
