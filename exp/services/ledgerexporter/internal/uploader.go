package ledgerexporter

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"

	"github.com/stellar/go/support/compressxdr"
	"github.com/stellar/go/support/datastore"
)

// Uploader is responsible for uploading data to a storage destination.
type Uploader struct {
	dataStore            datastore.DataStore
	queue                *UploadQueue
	uploadDurationMetric *prometheus.SummaryVec
	objectSizeMetrics    *prometheus.SummaryVec
	latestLedgerMetric   *prometheus.GaugeVec
}

// NewUploader constructs a new Uploader instance
func NewUploader(
	destination datastore.DataStore,
	queue *UploadQueue,
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
	latestLedgerMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "ledger_exporter", Subsystem: "uploader", Name: "latest_ledger",
		Help: "sequence number of the latest ledger uploaded by a worker",
	}, []string{"worker"})
	prometheusRegistry.MustRegister(uploadDurationMetric, objectSizeMetrics, latestLedgerMetric)
	return Uploader{
		dataStore:            destination,
		queue:                queue,
		uploadDurationMetric: uploadDurationMetric,
		objectSizeMetrics:    objectSizeMetrics,
		latestLedgerMetric:   latestLedgerMetric,
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
func (u Uploader) Upload(ctx context.Context, metaArchive *datastore.LedgerMetaArchive) error {
	startTime := time.Now()
	numLedgers := strconv.FormatUint(uint64(metaArchive.GetLedgerCount()), 10)

	xdrEncoder := compressxdr.NewXDREncoder(compressxdr.DefaultCompressor, &metaArchive.Data)

	writerTo := &writerToRecorder{
		WriterTo: xdrEncoder,
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
		"compression":    xdrEncoder.Compressor.Name(),
		"ledgers":        numLedgers,
		"already_exists": alreadyExists,
	}).Observe(float64(writerTo.totalCompressed))
	return nil
}

// TODO: make it configurable
var uploaderShutdownWaitTime = 10 * time.Second

// Run starts the uploader, continuously listening for LedgerMetaArchive objects to upload.
func (u Uploader) Run(ctx context.Context, numWorkers int) error {
	if numWorkers <= 0 {
		return fmt.Errorf("workers %v must be positive", numWorkers)
	}

	// groupCtx will cancel if ctx (the parent context) is canceled or
	// one of the upload workers returns an error
	group, groupCtx := errgroup.WithContext(ctx)
	// uploadCtx will cancel once groupCtx is cancelled but after a delay
	// in order to give time to complete remaining uploads
	uploadCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-uploadCtx.Done():
			// if uploadCtx is cancelled that means we have exited Run()
			// and therefore all the workers have been terminated so
			// there is no need to wait because there are no remaining
			// uploads
			return
		case <-groupCtx.Done():
			logger.Info("Received shutdown signal, waiting for remaining uploads to complete...")
		}

		select {
		case <-time.After(uploaderShutdownWaitTime):
			// wait for some time to upload remaining objects from
			// the upload queue
			logger.Info("Timeout reached, canceling remaining uploads...")
			cancel()
		case <-uploadCtx.Done():
			// if uploadCtx is cancelled that means we have exited Run()
			// and therefore all the workers have been terminated so
			// there is no need to wait because there are no remaining
			// uploads
			return
		}
	}()

	for i := 0; i < numWorkers; i++ {
		worker := i
		group.Go(func() error {
			for {
				metaObject, ok, err := u.queue.Dequeue(uploadCtx)
				if err != nil {
					return err
				}
				if !ok {
					logger.WithField("worker", worker).Info("Upload queue is closed, stopping upload worker")
					return nil
				}

				logger.WithField("worker", worker).Infof("Uploading: %s", metaObject.GetObjectKey())

				// Upload the received LedgerMetaArchive.
				if err = u.Upload(uploadCtx, metaObject); err != nil {
					return err
				}
				u.queue.Done(metaObject)
				u.latestLedgerMetric.With(prometheus.Labels{
					"worker": strconv.Itoa(worker),
				}).Set(float64(metaObject.GetEndLedgerSequence()))
				logger.WithField("worker", worker).Infof("Uploaded %s successfully", metaObject.ObjectKey)
			}
			return nil
		})
	}

	return group.Wait()
}
