package ledgerexporter

import (
	"context"
	"io"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// Uploader is responsible for uploading data to a storage destination.
type Uploader struct {
	dataStore            DataStore
	queue                UploadQueue
	uploadDurationMetric *prometheus.SummaryVec
	objectSizeMetrics    *prometheus.SummaryVec
}

// NewUploader constructs a new Uploader instance
func NewUploader(
	destination DataStore,
	queue UploadQueue,
	prometheusRegistry *prometheus.Registry,
) Uploader {
	uploadDurationMetric := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: "ledger_exporter", Subsystem: "uploader", Name: "upload_duration_seconds",
			Help:       "duration for uploading a ledger batch, sliding window = 10m",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"ledgers"},
	)
	objectSizeMetrics := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: "ledger_exporter", Subsystem: "uploader", Name: "object_size_bytes",
			Help:       "size of a ledger batch in bytes, sliding window = 10m",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"ledgers", "compression"},
	)
	prometheusRegistry.MustRegister(uploadDurationMetric)
	return Uploader{
		dataStore:            destination,
		queue:                queue,
		uploadDurationMetric: uploadDurationMetric,
		objectSizeMetrics:    objectSizeMetrics,
	}
}

type writerRecorder struct {
	io.Writer
	count int
}

func (r *writerRecorder) Write(p []byte) (int, error) {
	total, err := r.Writer.Write(p)
	r.count += total
	return total, err
}

type writerToRecorder struct {
	io.WriterTo
	compression       string
	ledgers           string
	objectSizeMetrics *prometheus.SummaryVec
}

func (r writerToRecorder) WriteTo(w io.Writer) (int64, error) {
	wrapper := &writerRecorder{
		Writer: w,
	}
	totalUncompressed, err := r.WriterTo.WriteTo(wrapper)
	if err != nil {
		return totalUncompressed, err
	}
	r.objectSizeMetrics.With(prometheus.Labels{
		"compression": "none",
		"ledgers":     r.ledgers,
	}).Observe(float64(totalUncompressed))
	r.objectSizeMetrics.With(prometheus.Labels{
		"compression": r.compression,
		"ledgers":     r.ledgers,
	}).Observe(float64(wrapper.count))
	return totalUncompressed, nil
}

// Upload uploads the serialized binary data of ledger TxMeta to the specified destination.
func (u Uploader) Upload(ctx context.Context, metaArchive *LedgerMetaArchive) error {
	logger.Infof("Uploading: %s", metaArchive.GetObjectKey())
	startTime := time.Now()
	numLedgers := strconv.FormatUint(uint64(metaArchive.GetLedgerCount()), 10)

	writerTo := writerToRecorder{
		WriterTo:          &XDRGzipEncoder{XdrPayload: &metaArchive.data},
		compression:       "gzip",
		ledgers:           numLedgers,
		objectSizeMetrics: u.objectSizeMetrics,
	}
	err := u.dataStore.PutFileIfNotExists(ctx, metaArchive.GetObjectKey(), writerTo)
	if err != nil {
		return errors.Wrapf(err, "error uploading %s", metaArchive.GetObjectKey())
	}

	u.uploadDurationMetric.With(prometheus.Labels{"ledgers": numLedgers}).Observe(time.Since(startTime).Seconds())
	return nil
}

// TODO: make it configurable
var uploaderShutdownWaitTime = 10 * time.Second

// Run starts the uploader, continuously listening for LedgerMetaArchive objects to upload.
func (u Uploader) Run(ctx context.Context) error {
	uploadCtx, cancel := context.WithCancel(context.Background())
	go func() {
		<-ctx.Done()
		logger.Info("Context done, waiting for remaining uploads to complete...")
		// wait for a few seconds to upload remaining objects from metaArchiveCh
		<-time.After(uploaderShutdownWaitTime)
		logger.Info("Timeout reached, canceling remaining uploads...")
		cancel()
	}()

	for {
		metaObject, ok, err := u.queue.Dequeue(uploadCtx)
		if err != nil {
			return err
		}
		if !ok {
			logger.Info("Meta archive channel closed, stopping uploader")
			return nil
		}

		// Upload the received LedgerMetaArchive.
		if err = u.Upload(uploadCtx, metaObject); err != nil {
			return err
		}
		logger.Infof("Uploaded %s successfully", metaObject.objectKey)
	}
}
