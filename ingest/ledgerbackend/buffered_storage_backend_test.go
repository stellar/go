package ledgerbackend

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stellar/go/support/compressxdr"
	"github.com/stellar/go/support/datastore"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var partitionSize = uint32(64000)
var ledgerPerFileCount = uint32(1)

func createBufferedStorageBackendConfigForTesting() BufferedStorageBackendConfig {
	param := make(map[string]string)
	param["destination_bucket_path"] = "testURL"

	ledgerBatchConfig := datastore.LedgerBatchConfig{
		LedgersPerFile:    1,
		FilesPerPartition: 64000,
		FileSuffix:        ".xdr.gz",
	}

	dataStore := new(datastore.MockDataStore)

	return BufferedStorageBackendConfig{
		LedgerBatchConfig: ledgerBatchConfig,
		CompressionType:   compressxdr.GZIP,
		DataStore:         dataStore,
		BufferSize:        100,
		NumWorkers:        5,
		RetryLimit:        3,
		RetryWait:         1,
	}
}

func createBufferedStorageBackendForTesting() BufferedStorageBackend {
	config := createBufferedStorageBackendConfigForTesting()
	ledgerMetaArchive := datastore.NewLedgerMetaArchive("", 0, 0)
	decoder, _ := compressxdr.NewXDRDecoder(config.CompressionType, nil)

	return BufferedStorageBackend{
		config:            config,
		dataStore:         config.DataStore,
		ledgerMetaArchive: ledgerMetaArchive,
		decoder:           decoder,
	}
}

func createMockdataStore(start, end, partitionSize, count uint32) *datastore.MockDataStore {
	mockDataStore := new(datastore.MockDataStore)
	partition := count*partitionSize - 1
	for i := start; i <= end; i = i + count {
		var objectName string
		var readCloser io.ReadCloser
		if count > 1 {
			endFileSeq := i + count - 1
			readCloser = createLCMBatchReader(i, endFileSeq, count)
			objectName = fmt.Sprintf("0-%d/%d-%d.xdr.gz", partition, i, endFileSeq)
		} else {
			readCloser = createLCMBatchReader(i, i, count)
			objectName = fmt.Sprintf("0-%d/%d.xdr.gz", partition, i)
		}
		mockDataStore.On("GetFile", mock.Anything, objectName).Return(readCloser, nil)
	}

	return mockDataStore
}

func createLCMForTesting(start, end uint32) []xdr.LedgerCloseMeta {
	var lcmArray []xdr.LedgerCloseMeta
	for i := start; i <= end; i++ {
		lcmArray = append(lcmArray, datastore.CreateLedgerCloseMeta(i))
	}

	return lcmArray
}

func createTestLedgerCloseMetaBatch(startSeq, endSeq, count uint32) xdr.LedgerCloseMetaBatch {
	var ledgerCloseMetas []xdr.LedgerCloseMeta
	for i := uint32(0); i < count; i++ {
		ledgerCloseMetas = append(ledgerCloseMetas, datastore.CreateLedgerCloseMeta(startSeq+uint32(i)))
	}
	return xdr.LedgerCloseMetaBatch{
		StartSequence:    xdr.Uint32(startSeq),
		EndSequence:      xdr.Uint32(endSeq),
		LedgerCloseMetas: ledgerCloseMetas,
	}
}

func createLCMBatchReader(start, end, count uint32) io.ReadCloser {
	testData := createTestLedgerCloseMetaBatch(start, end, count)
	encoder, _ := compressxdr.NewXDREncoder(compressxdr.GZIP, testData)
	var buf bytes.Buffer
	encoder.WriteTo(&buf)
	capturedBuf := buf.Bytes()
	reader := bytes.NewReader(capturedBuf)
	return io.NopCloser(reader)
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

func TestNewLedgerBuffer(t *testing.T) {
	startLedger := uint32(2)
	endLedger := uint32(3)
	bsb := createBufferedStorageBackendForTesting()
	ledgerRange := BoundedRange(startLedger, endLedger)
	mockDataStore := createMockdataStore(startLedger, endLedger, partitionSize, ledgerPerFileCount)
	bsb.dataStore = mockDataStore

	ledgerBuffer, err := bsb.newLedgerBuffer(ledgerRange)
	assert.Eventually(t, func() bool { return len(ledgerBuffer.ledgerQueue) == 2 }, time.Second*5, time.Millisecond*50)
	assert.Eventually(t, func() bool { return len(ledgerBuffer.taskQueue) == 0 }, time.Second*5, time.Millisecond*50)
	assert.Eventually(t, func() bool { return ledgerBuffer.ledgerPriorityQueue.Len() == 0 }, time.Second*5, time.Millisecond*50)
	assert.NoError(t, err)

	// values should be the ledger following ledgerRange.to
	assert.Equal(t, uint32(4), ledgerBuffer.currentLedger)
	assert.Equal(t, uint32(4), ledgerBuffer.nextTaskLedger)
	assert.Equal(t, ledgerRange, ledgerBuffer.ledgerRange)

	mockDataStore.AssertExpectations(t)
}

func TestBSBGetLatestLedgerSequence(t *testing.T) {
	startLedger := uint32(3)
	endLedger := uint32(5)
	ctx := context.Background()
	bsb := createBufferedStorageBackendForTesting()
	ledgerRange := BoundedRange(startLedger, endLedger)
	mockDataStore := createMockdataStore(startLedger, endLedger, partitionSize, ledgerPerFileCount)
	bsb.dataStore = mockDataStore

	assert.NoError(t, bsb.PrepareRange(ctx, ledgerRange))
	assert.Eventually(t, func() bool { return len(bsb.ledgerBuffer.ledgerQueue) == 3 }, time.Second*5, time.Millisecond*50)

	latestSeq, err := bsb.GetLatestLedgerSequence(ctx)
	assert.NoError(t, err)

	assert.Equal(t, uint32(5), latestSeq)

	mockDataStore.AssertExpectations(t)
}

func TestBSBGetLedger_SingleLedgerPerFile(t *testing.T) {
	startLedger := uint32(3)
	endLedger := uint32(5)
	ctx := context.Background()
	lcmArray := createLCMForTesting(startLedger, endLedger)
	bsb := createBufferedStorageBackendForTesting()
	ledgerRange := BoundedRange(startLedger, endLedger)

	mockDataStore := createMockdataStore(startLedger, endLedger, partitionSize, ledgerPerFileCount)
	bsb.dataStore = mockDataStore

	assert.NoError(t, bsb.PrepareRange(ctx, ledgerRange))
	assert.Eventually(t, func() bool { return len(bsb.ledgerBuffer.ledgerQueue) == 3 }, time.Second*5, time.Millisecond*50)

	lcm, err := bsb.GetLedger(ctx, uint32(3))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[0], lcm)
	lcm, err = bsb.GetLedger(ctx, uint32(4))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[1], lcm)
	lcm, err = bsb.GetLedger(ctx, uint32(5))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[2], lcm)

	mockDataStore.AssertExpectations(t)
}

func TestCloudStorageGetLedger_MultipleLedgerPerFile(t *testing.T) {
	startLedger := uint32(2)
	endLedger := uint32(5)
	lcmArray := createLCMForTesting(startLedger, endLedger)
	bsb := createBufferedStorageBackendForTesting()
	ctx := context.Background()
	bsb.config.LedgerBatchConfig.LedgersPerFile = uint32(2)
	ledgerRange := BoundedRange(startLedger, endLedger)

	mockDataStore := createMockdataStore(startLedger, endLedger, partitionSize, 2)
	bsb.dataStore = mockDataStore

	assert.NoError(t, bsb.PrepareRange(ctx, ledgerRange))
	assert.Eventually(t, func() bool { return len(bsb.ledgerBuffer.ledgerQueue) == 2 }, time.Second*5, time.Millisecond*50)

	lcm, err := bsb.GetLedger(ctx, uint32(2))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[0], lcm)
	lcm, err = bsb.GetLedger(ctx, uint32(3))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[1], lcm)
	lcm, err = bsb.GetLedger(ctx, uint32(4))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[2], lcm)

	mockDataStore.AssertExpectations(t)
}

func TestBSBGetLedger_ErrorPreceedingLedger(t *testing.T) {
	startLedger := uint32(3)
	endLedger := uint32(5)
	ctx := context.Background()
	lcmArray := createLCMForTesting(startLedger, endLedger)
	bsb := createBufferedStorageBackendForTesting()
	ledgerRange := BoundedRange(startLedger, endLedger)

	mockDataStore := createMockdataStore(startLedger, endLedger, partitionSize, ledgerPerFileCount)
	bsb.dataStore = mockDataStore

	assert.NoError(t, bsb.PrepareRange(ctx, ledgerRange))
	assert.Eventually(t, func() bool { return len(bsb.ledgerBuffer.ledgerQueue) == 3 }, time.Second*5, time.Millisecond*50)

	lcm, err := bsb.GetLedger(ctx, uint32(3))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[0], lcm)

	_, err = bsb.GetLedger(ctx, uint32(2))
	assert.Error(t, err, "requested sequence preceeds current LedgerCloseMetaBatch")

	mockDataStore.AssertExpectations(t)
}

func TestBSBGetLedger_NotPrepared(t *testing.T) {
	bsb := createBufferedStorageBackendForTesting()
	ctx := context.Background()

	_, err := bsb.GetLedger(ctx, uint32(3))
	assert.Error(t, err, "session is not prepared, call PrepareRange first")
}

func TestBSBGetLedger_SequenceNotInBatch(t *testing.T) {
	startLedger := uint32(3)
	endLedger := uint32(5)
	ctx := context.Background()
	bsb := createBufferedStorageBackendForTesting()
	ledgerRange := BoundedRange(startLedger, endLedger)

	mockDataStore := createMockdataStore(startLedger, endLedger, partitionSize, ledgerPerFileCount)
	bsb.dataStore = mockDataStore

	assert.NoError(t, bsb.PrepareRange(ctx, ledgerRange))
	assert.Eventually(t, func() bool { return len(bsb.ledgerBuffer.ledgerQueue) == 3 }, time.Second*5, time.Millisecond*50)

	_, err := bsb.GetLedger(ctx, uint32(2))
	assert.Error(t, err, "requested sequence preceeds current LedgerRange")

	_, err = bsb.GetLedger(ctx, uint32(6))
	assert.Error(t, err, "requested sequence beyond current LedgerRange")
}

func TestBSBPrepareRange(t *testing.T) {
	startLedger := uint32(2)
	endLedger := uint32(3)
	ctx := context.Background()
	bsb := createBufferedStorageBackendForTesting()
	ledgerRange := BoundedRange(startLedger, endLedger)

	mockDataStore := createMockdataStore(startLedger, endLedger, partitionSize, ledgerPerFileCount)
	bsb.dataStore = mockDataStore

	assert.NoError(t, bsb.PrepareRange(ctx, ledgerRange))
	assert.Eventually(t, func() bool { return len(bsb.ledgerBuffer.ledgerQueue) == 2 }, time.Second*5, time.Millisecond*50)

	assert.NotNil(t, bsb.prepared)

	// check alreadyPrepared
	err := bsb.PrepareRange(ctx, ledgerRange)
	assert.NoError(t, err)
	assert.NotNil(t, bsb.prepared)
}

func TestBSBIsPrepared_Bounded(t *testing.T) {
	startLedger := uint32(3)
	endLedger := uint32(5)
	ctx := context.Background()
	bsb := createBufferedStorageBackendForTesting()
	ledgerRange := BoundedRange(startLedger, endLedger)

	mockDataStore := createMockdataStore(startLedger, endLedger, partitionSize, ledgerPerFileCount)
	bsb.dataStore = mockDataStore

	assert.NoError(t, bsb.PrepareRange(ctx, ledgerRange))
	assert.Eventually(t, func() bool { return len(bsb.ledgerBuffer.ledgerQueue) == 3 }, time.Second*5, time.Millisecond*50)

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

func TestBSBIsPrepared_Unbounded(t *testing.T) {
	startLedger := uint32(3)
	endLedger := uint32(8)
	ctx := context.Background()
	bsb := createBufferedStorageBackendForTesting()
	bsb.config.NumWorkers = 2
	bsb.config.BufferSize = 5
	ledgerRange := UnboundedRange(3)
	mockDataStore := createMockdataStore(startLedger, endLedger, partitionSize, ledgerPerFileCount)
	bsb.dataStore = mockDataStore

	assert.NoError(t, bsb.PrepareRange(ctx, ledgerRange))
	assert.Eventually(t, func() bool { return len(bsb.ledgerBuffer.ledgerQueue) == 5 }, time.Second*5, time.Millisecond*50)

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

func TestBSBClose(t *testing.T) {
	startLedger := uint32(2)
	endLedger := uint32(3)
	ctx := context.Background()
	bsb := createBufferedStorageBackendForTesting()
	ledgerRange := BoundedRange(startLedger, endLedger)

	mockDataStore := createMockdataStore(startLedger, endLedger, partitionSize, ledgerPerFileCount)
	bsb.dataStore = mockDataStore

	assert.NoError(t, bsb.PrepareRange(ctx, ledgerRange))
	assert.Eventually(t, func() bool { return len(bsb.ledgerBuffer.ledgerQueue) == 2 }, time.Second*5, time.Millisecond*50)

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

	mockDataStore.AssertExpectations(t)
}

func TestLedgerBufferInvariant(t *testing.T) {
	startLedger := uint32(3)
	endLedger := uint32(6)
	ctx := context.Background()
	lcmArray := createLCMForTesting(startLedger, endLedger)
	bsb := createBufferedStorageBackendForTesting()
	bsb.config.NumWorkers = 2
	bsb.config.BufferSize = 2
	ledgerRange := BoundedRange(startLedger, endLedger)

	mockDataStore := createMockdataStore(startLedger, endLedger, partitionSize, ledgerPerFileCount)
	bsb.dataStore = mockDataStore

	assert.NoError(t, bsb.PrepareRange(ctx, ledgerRange))
	assert.Eventually(t, func() bool { return len(bsb.ledgerBuffer.ledgerQueue) == 2 }, time.Second*5, time.Millisecond*50)

	// Buffer should have hit the BufferSize limit
	assert.Equal(t, 2, len(bsb.ledgerBuffer.ledgerQueue))

	lcm, err := bsb.GetLedger(ctx, uint32(3))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[0], lcm)
	lcm, err = bsb.GetLedger(ctx, uint32(4))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[1], lcm)

	// Buffer should fill up with remaining ledgers
	assert.Eventually(t, func() bool { return len(bsb.ledgerBuffer.ledgerQueue) == 2 }, time.Second*5, time.Millisecond*50)
	assert.Equal(t, 2, len(bsb.ledgerBuffer.ledgerQueue))

	lcm, err = bsb.GetLedger(ctx, uint32(5))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[2], lcm)

	// Buffer should only have the final ledger
	assert.Eventually(t, func() bool { return len(bsb.ledgerBuffer.ledgerQueue) == 1 }, time.Second*5, time.Millisecond*50)
	assert.Equal(t, 1, len(bsb.ledgerBuffer.ledgerQueue))

	lcm, err = bsb.GetLedger(ctx, uint32(6))
	assert.NoError(t, err)
	assert.Equal(t, lcmArray[3], lcm)

	// Buffer should be empty
	assert.Equal(t, 0, len(bsb.ledgerBuffer.ledgerQueue))

	mockDataStore.AssertExpectations(t)
}
