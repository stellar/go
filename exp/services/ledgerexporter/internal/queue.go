package ledgerexporter

import (
	"context"
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stellar/go/support/collections/heap"
	"github.com/stellar/go/support/datastore"
)

type uploadTask struct {
	start     uint32
	batchSize uint32
}

// UploadQueue is a queue of LedgerMetaArchive objects which are scheduled for upload
type UploadQueue struct {
	metaArchiveCh chan *datastore.LedgerMetaArchive
	// window and start represents a sliding window of ledger files
	// that are allowed to be uploaded. Ledger files preceding start have already been
	// uploaded. Ledger files ahead of the window will not be uploaded until the window
	// advances to include those files.
	// For example if start is 5, number of ledgers per file is 1, and queue size is 10,
	// we can allow ledger files 5 to 14 to be uploaded and we will not upload ledger
	// file 15 until the upload for ledger file 5 is complete (at which point start
	// can advance to 6)
	window chan struct{}
	start  uint32
	// completed contains all finished tasks from the sliding window described above
	completed         *heap.Heap[uploadTask]
	completedLock     sync.Mutex
	queueLengthMetric prometheus.Gauge
	inProgressMetric  prometheus.Gauge
}

// NewUploadQueue constructs a new UploadQueue
func NewUploadQueue(size int, prometheusRegistry *prometheus.Registry) *UploadQueue {
	queueLengthMetric := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "ledger_exporter",
		Subsystem: "upload_queue",
		Name:      "length",
		Help:      "The number of objects queued for upload",
	})
	inProgressMetric := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "ledger_exporter",
		Subsystem: "upload_queue",
		Name:      "in_progress",
		Help:      "The number of object uploads which are in progress",
	})
	prometheusRegistry.MustRegister(queueLengthMetric, inProgressMetric)
	// initialize the sliding window so that there are `size` number
	// of tasks which are allowed to be uploaded
	window := make(chan struct{}, size)
	for i := 0; i < size; i++ {
		window <- struct{}{}
	}
	return &UploadQueue{
		completed: heap.New(func(a, b uploadTask) bool {
			return a.start < b.start
		}, size),
		metaArchiveCh:     make(chan *datastore.LedgerMetaArchive, size),
		window:            window,
		queueLengthMetric: queueLengthMetric,
		inProgressMetric:  inProgressMetric,
	}
}

// Enqueue will add an upload task to the queue. Enqueue may block if the queue is full.
// Enqueue must be called on consecutive LedgerMetaArchives.
// Enqueue is not thread safe.
func (u *UploadQueue) Enqueue(ctx context.Context, archive *datastore.LedgerMetaArchive) error {
	if archive.GetLedgerCount() == 0 {
		return fmt.Errorf("archive has 0 ledgers")
	}
	if startLedger := archive.GetStartLedgerSequence(); startLedger <= 0 {
		return fmt.Errorf("archive has invalid start: %v", startLedger)
	}
	if u.start == 0 {
		u.start = archive.GetStartLedgerSequence()
	}
	u.queueLengthMetric.Inc()
	select {
	case u.metaArchiveCh <- archive:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Dequeue will pop a task off the queue. Dequeue may block if the queue is empty.
// Done() must be called once the task is complete to allow additional tasks to be
// consumed in future calls to Dequeue().
// Dequeue is thread safe.
func (u *UploadQueue) Dequeue(ctx context.Context) (*datastore.LedgerMetaArchive, bool, error) {
	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()

	case metaObject, ok := <-u.metaArchiveCh:
		if !ok {
			return metaObject, ok, nil
		}

		// block until the sliding window allows us to proceed with the next task
		select {
		case <-ctx.Done():
			return nil, false, ctx.Err()
		case <-u.window:
		}
		u.queueLengthMetric.Dec()
		u.inProgressMetric.Inc()
		return metaObject, ok, nil
	}
}

// Done registers that the given task emitted by Dequeue() is complete.
// Done should not be called more than once on a given task.
// Done is thread safe.
func (u *UploadQueue) Done(metaObj *datastore.LedgerMetaArchive) {
	u.completedLock.Lock()
	defer u.completedLock.Unlock()
	u.inProgressMetric.Dec()
	u.completed.Push(uploadTask{
		start:     metaObj.GetStartLedgerSequence(),
		batchSize: metaObj.GetLedgerCount(),
	})

	// advance the sliding window if we have completed tasks at the
	// start boundary of the sliding window
	for u.completed.Len() > 0 && u.start == u.completed.Peek().start {
		item := u.completed.Pop()
		u.start += item.batchSize
		// Note that sending on the window channel should never block
		// because Done() is called no more than once on a task emitted by
		// Dequeue(). Dequeue() consumes an item from the window channel
		// and Done() sends the item back to the window channel.
		// There is no other way to consume / send on the window channel
		// other than by calling Dequeue() / Done().
		// Therefore, we can never exceed the capacity of the window channel
		// because for that to occur we would need to send (e.g. calling Done())
		// more times than we consume (e.g. calling Dequeue()).
		u.window <- struct{}{}
	}
}

// Close will close the queue. After the queue is closed Enqueue should
// no longer be called.
// Close is thread safe.
func (u *UploadQueue) Close() {
	close(u.metaArchiveCh)
}
