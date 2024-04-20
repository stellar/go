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
			Namespace: "ledger_exporter", Subsystem: "uploader", Name: "put_duration_seconds",
			Help:       "duration for uploading a ledger batch, sliding window = 10m",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"already_exists", "ledgers"},
	)
	objectSizeMetrics := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: "ledger_exporter", Subsystem: "uploader", Name: "object_size_bytes",
			Help:       "size of a ledger batch in bytes, sliding window = 10m",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"ledgers", "already_exists", "compression"},
	)
	prometheusRegistry.MustRegister(uploadDurationMetric, objectSizeMetrics)
	return Uploader{
		dataStore:            destination,
		queue:                queue,
		uploadDurationMetric: uploadDurationMetric,
		objectSizeMetrics:    objectSizeMetrics,
	}
}

type writerRecorder struct {
	io.Writer
	count *int64
}

func (r writerRecorder) Write(p []byte) (int, error) {
	total, err := r.Writer.Write(p)
	*r.count += int64(total)
	return total, err
}

type writerToRecorder struct {
	io.WriterTo
	totalCompressed   int64
	totalUncompressed int64
}

func (r *writerToRecorder) WriteTo(w io.Writer) (int64, error) {
	uncompressedCount, err := r.WriterTo.WriteTo(writerRecorder{
		Writer: w,
		count:  &r.totalCompressed,
	})
	r.totalUncompressed += uncompressedCount
	return uncompressedCount, err
}

// Upload uploads the serialized binary data of ledger TxMeta to the specified destination.
func (u Uploader) Upload(ctx context.Context, metaArchive *LedgerMetaArchive) error {
	logger.Infof("Uploading: %s", metaArchive.GetObjectKey())
	startTime := time.Now()
	numLedgers := strconv.FormatUint(uint64(metaArchive.GetLedgerCount()), 10)

	writerTo := &writerToRecorder{
		WriterTo: &XDRGzipEncoder{XdrPayload: &metaArchive.data},
	}
	ok, err := u.dataStore.PutFileIfNotExists(ctx, metaArchive.GetObjectKey(), writerTo)
	if err != nil {
		return errors.Wrapf(err, "error uploading %s", metaArchive.GetObjectKey())
	}
	alreadyExists := strconv.FormatBool(!ok)

	u.uploadDurationMetric.With(prometheus.Labels{
		"ledgers":        numLedgers,
		"already_exists": alreadyExists,
	}).Observe(time.Since(startTime).Seconds())
	u.objectSizeMetrics.With(prometheus.Labels{
		"compression":    "none",
		"ledgers":        numLedgers,
		"already_exists": alreadyExists,
	}).Observe(float64(writerTo.totalUncompressed))
	u.objectSizeMetrics.With(prometheus.Labels{
		"compression":    "gzip",
		"ledgers":        numLedgers,
		"already_exists": alreadyExists,
	}).Observe(float64(writerTo.totalCompressed))
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
