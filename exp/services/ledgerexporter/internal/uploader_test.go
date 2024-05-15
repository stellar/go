package ledgerexporter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/stellar/go/support/compressxdr"
	"github.com/stellar/go/support/datastore"
	"github.com/stellar/go/support/errors"
)

func TestUploaderSuite(t *testing.T) {
	suite.Run(t, new(UploaderSuite))
}

// UploaderSuite is a test suite for the Uploader.
type UploaderSuite struct {
	suite.Suite
	ctx           context.Context
	mockDataStore datastore.MockDataStore
}

func (s *UploaderSuite) SetupTest() {
	s.ctx = context.Background()
	s.mockDataStore = datastore.MockDataStore{}
}

func (s *UploaderSuite) TestUpload() {
	s.testUpload(false)
	s.testUpload(true)
}

func (s *UploaderSuite) testUpload(putOkReturnVal bool) {
	key, start, end := "test-1-100", uint32(1), uint32(100)
	archive := newLedgerMetaArchive(key, start, end)

	var capturedBuf bytes.Buffer
	var capturedKey string
	s.mockDataStore.On("PutFileIfNotExists", mock.Anything, key, mock.Anything).
		Run(func(args mock.Arguments) {
			capturedKey = args.Get(1).(string)
			_, err := args.Get(2).(io.WriterTo).WriteTo(&capturedBuf)
			require.NoError(s.T(), err)
		}).Return(putOkReturnVal, nil).Once()

	registry := prometheus.NewRegistry()
	queue := NewUploadQueue(1, registry)
	dataUploader := NewUploader(&s.mockDataStore, queue, registry)
	require.NoError(s.T(), dataUploader.Upload(context.Background(), archive))

	expectedCompressedLength := capturedBuf.Len()
	var decodedArchive datastore.LedgerMetaArchive
	xdrDecoder := compressxdr.NewXDRDecoder(compressxdr.DefaultCompressor, &decodedArchive.Data)

	decoder := xdrDecoder
	_, err := decoder.ReadFrom(&capturedBuf)
	require.NoError(s.T(), err)

	// require that the decoded data matches the original test data
	require.Equal(s.T(), key, capturedKey)
	require.Equal(s.T(), archive.Data, decodedArchive.Data)

	alreadyExists := !putOkReturnVal
	metric, err := dataUploader.uploadDurationMetric.MetricVec.GetMetricWith(prometheus.Labels{
		"ledgers":        "100",
		"already_exists": strconv.FormatBool(alreadyExists),
	})
	require.NoError(s.T(), err)
	require.Equal(
		s.T(),
		uint64(1),
		getMetricValue(metric).GetSummary().GetSampleCount(),
	)
	require.Positive(s.T(), getMetricValue(metric).GetSummary().GetSampleSum())
	metric, err = dataUploader.uploadDurationMetric.MetricVec.GetMetricWith(prometheus.Labels{
		"ledgers":        "100",
		"already_exists": strconv.FormatBool(!alreadyExists),
	})
	require.NoError(s.T(), err)
	require.Equal(
		s.T(),
		uint64(0),
		getMetricValue(metric).GetSummary().GetSampleCount(),
	)

	metric, err = dataUploader.objectSizeMetrics.MetricVec.GetMetricWith(prometheus.Labels{
		"ledgers":        "100",
		"compression":    decoder.Compressor.Name(),
		"already_exists": strconv.FormatBool(alreadyExists),
	})
	require.NoError(s.T(), err)
	require.Equal(
		s.T(),
		uint64(1),
		getMetricValue(metric).GetSummary().GetSampleCount(),
	)
	require.Equal(
		s.T(),
		float64(expectedCompressedLength),
		getMetricValue(metric).GetSummary().GetSampleSum(),
	)
	metric, err = dataUploader.objectSizeMetrics.MetricVec.GetMetricWith(prometheus.Labels{
		"ledgers":        "100",
		"compression":    decoder.Compressor.Name(),
		"already_exists": strconv.FormatBool(!alreadyExists),
	})
	require.NoError(s.T(), err)
	require.Equal(
		s.T(),
		uint64(0),
		getMetricValue(metric).GetSummary().GetSampleCount(),
	)

	metric, err = dataUploader.objectSizeMetrics.MetricVec.GetMetricWith(prometheus.Labels{
		"ledgers":        "100",
		"compression":    "none",
		"already_exists": strconv.FormatBool(alreadyExists),
	})
	require.NoError(s.T(), err)
	require.Equal(
		s.T(),
		uint64(1),
		getMetricValue(metric).GetSummary().GetSampleCount(),
	)
	uncompressedPayload, err := decodedArchive.Data.MarshalBinary()
	require.NoError(s.T(), err)
	require.Equal(
		s.T(),
		float64(len(uncompressedPayload)),
		getMetricValue(metric).GetSummary().GetSampleSum(),
	)
	metric, err = dataUploader.objectSizeMetrics.MetricVec.GetMetricWith(prometheus.Labels{
		"ledgers":        "100",
		"compression":    "none",
		"already_exists": strconv.FormatBool(!alreadyExists),
	})
	require.NoError(s.T(), err)
	require.Equal(
		s.T(),
		uint64(0),
		getMetricValue(metric).GetSummary().GetSampleCount(),
	)
}

func newLedgerMetaArchive(key string, start uint32, end uint32) *datastore.LedgerMetaArchive {
	archive := datastore.NewLedgerMetaArchive(key, start, end)
	for i := start; i <= end; i++ {
		_ = archive.AddLedger(datastore.CreateLedgerCloseMeta(i))
	}
	return archive
}

func (s *UploaderSuite) TestUploadPutError() {
	s.testUploadPutError(true)
	s.testUploadPutError(false)
}

func (s *UploaderSuite) testUploadPutError(putOkReturnVal bool) {
	key, start, end := "test-1-100", uint32(1), uint32(100)
	archive := newLedgerMetaArchive(key, start, end)

	s.mockDataStore.On("PutFileIfNotExists", context.Background(), key,
		mock.Anything).Return(putOkReturnVal, errors.New("error in PutFileIfNotExists"))

	registry := prometheus.NewRegistry()
	queue := NewUploadQueue(1, registry)
	dataUploader := NewUploader(&s.mockDataStore, queue, registry)
	err := dataUploader.Upload(context.Background(), archive)
	require.Equal(s.T(), fmt.Sprintf("error uploading %s: error in PutFileIfNotExists", key), err.Error())

	for _, alreadyExists := range []string{"true", "false"} {
		metric, err := dataUploader.uploadDurationMetric.MetricVec.GetMetricWith(prometheus.Labels{
			"ledgers":        "100",
			"already_exists": alreadyExists,
		})
		require.NoError(s.T(), err)
		require.Equal(
			s.T(),
			uint64(0),
			getMetricValue(metric).GetSummary().GetSampleCount(),
		)

		for _, compression := range []string{compressxdr.DefaultCompressor.Name(), "none"} {
			metric, err = dataUploader.objectSizeMetrics.MetricVec.GetMetricWith(prometheus.Labels{
				"ledgers":        "100",
				"compression":    compression,
				"already_exists": alreadyExists,
			})
			require.NoError(s.T(), err)
			require.Equal(
				s.T(),
				uint64(0),
				getMetricValue(metric).GetSummary().GetSampleCount(),
			)
		}
	}
}

func (s *UploaderSuite) TestRunChannelClose() {
	s.mockDataStore.On("PutFileIfNotExists", mock.Anything,
		mock.Anything, mock.Anything).Return(true, nil)

	registry := prometheus.NewRegistry()
	queue := NewUploadQueue(1, registry)
	go func() {
		key, start, end := "test", uint32(1), uint32(100)
		for i := start; i <= end; i++ {
			s.Assert().NoError(queue.Enqueue(s.ctx, newLedgerMetaArchive(key, i, i)))
		}
		queue.Close()
	}()

	dataUploader := NewUploader(&s.mockDataStore, queue, registry)
	require.NoError(s.T(), dataUploader.Run(context.Background(), 1))
}

func (s *UploaderSuite) TestRunContextCancel() {
	s.mockDataStore.On("PutFileIfNotExists", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
	ctx, cancel := context.WithCancel(context.Background())
	registry := prometheus.NewRegistry()
	queue := NewUploadQueue(1, registry)
	s.Assert().NoError(queue.Enqueue(s.ctx, newLedgerMetaArchive("test", 1, 1)))

	go func() {
		<-time.After(time.Second * 2)
		cancel()
	}()

	dataUploader := NewUploader(&s.mockDataStore, queue, registry)
	require.EqualError(s.T(), dataUploader.Run(ctx, 1), "context canceled")
}

func (s *UploaderSuite) TestRunUploadError() {
	registry := prometheus.NewRegistry()
	queue := NewUploadQueue(1, registry)

	s.Assert().NoError(queue.Enqueue(s.ctx, newLedgerMetaArchive("test", 1, 1)))
	s.mockDataStore.On("PutFileIfNotExists", mock.Anything, "test",
		mock.Anything).Return(false, errors.New("Put error"))

	dataUploader := NewUploader(&s.mockDataStore, queue, registry)
	err := dataUploader.Run(context.Background(), 1)
	require.Equal(s.T(), "error uploading test: Put error", err.Error())
}
