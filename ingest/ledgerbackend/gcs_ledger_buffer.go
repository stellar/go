package ledgerbackend

import (
	"bytes"
	"context"
	"io"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stellar/go/support/collections/heap"
	"github.com/stellar/go/support/compressxdr"
	"github.com/stellar/go/support/datastore"
	"github.com/stellar/go/support/log"
	"google.golang.org/api/googleapi"
)

type ledgerBufferGCS struct {
	config              GCSBackendConfig
	dataStore           datastore.DataStore
	taskQueue           chan uint32 // buffer next gcs object read
	ledgerQueue         chan []byte // order corrected lcm batches
	ledgerPriorityQueue *heap.Heap[ledgerBatchObject]
	priorityQueueLock   sync.Mutex
	done                chan struct{}

	// keep track of the ledgers to be processed and the next ordering
	// the ledgers should be buffered
	currentLedger  uint32
	nextTaskLedger uint32
	ledgerRange    Range

	// passed through from GCSBackend to control lifetime of ledgerBufferGCS instance
	context context.Context
	cancel  context.CancelCauseFunc
	decoder compressxdr.XDRDecoder
}

func (gcsb *GCSBackend) newLedgerBuffer(ledgerRange Range) (*ledgerBufferGCS, error) {
	less := func(a, b ledgerBatchObject) bool {
		return a.startLedger < b.startLedger
	}
	pq := heap.New(less, int(gcsb.config.BufferConfig.BufferSize))

	done := make(chan struct{})

	ledgerBuffer := &ledgerBufferGCS{
		config:              gcsb.config,
		dataStore:           gcsb.dataStore,
		taskQueue:           make(chan uint32, gcsb.config.BufferConfig.BufferSize),
		ledgerQueue:         make(chan []byte, gcsb.config.BufferConfig.BufferSize),
		ledgerPriorityQueue: pq,
		done:                done,
		currentLedger:       ledgerRange.from,
		nextTaskLedger:      ledgerRange.from,
		ledgerRange:         ledgerRange,
		context:             gcsb.context,
		cancel:              gcsb.cancel,
		decoder:             gcsb.decoder,
	}

	// Workers to read LCM files
	for i := uint32(0); i < gcsb.config.BufferConfig.NumWorkers; i++ {
		go ledgerBuffer.worker()
	}

	return ledgerBuffer, nil
}

func (lb *ledgerBufferGCS) pushTaskQueue() {
	for {
		// In bounded mode, don't queue past the end ledger
		if lb.nextTaskLedger > lb.ledgerRange.to && lb.ledgerRange.bounded {
			return
		}
		select {
		case lb.taskQueue <- lb.nextTaskLedger:
			lb.nextTaskLedger += lb.config.LedgerBatchConfig.LedgersPerFile
		default:
			return
		}
	}
}

func (lb *ledgerBufferGCS) worker() {
	for {
		select {
		case <-lb.done:
			log.Error("abort: getFromLedgerQueue blocked")
			return
		case <-lb.context.Done():
			log.Error(lb.context.Err())
			return
		case sequence := <-lb.taskQueue:
			retryCount := uint32(0)
			for retryCount <= lb.config.BufferConfig.RetryLimit {
				ledgerObject, err := lb.getLedgerGCSObject(sequence)
				if err != nil {
					if e, ok := err.(*googleapi.Error); ok {
						// ledgerObject not found and unbounded
						if e.Code == 404 && !lb.ledgerRange.bounded {
							time.Sleep(lb.config.BufferConfig.RetryWait * time.Second)
							continue
						}
					}
					if retryCount == lb.config.BufferConfig.RetryLimit {
						err = errors.New("maximum retries exceeded for gcs object reads")
						lb.cancel(err)
					}
					retryCount++
					time.Sleep(lb.config.BufferConfig.RetryWait * time.Second)
				}

				// Add to priority queue and continue to next task
				lb.storeObject(ledgerObject, sequence)
				break
			}
		}
	}
}

func (lb *ledgerBufferGCS) getLedgerGCSObject(sequence uint32) ([]byte, error) {
	objectKey := lb.config.LedgerBatchConfig.GetObjectKeyFromSequenceNumber(sequence)

	reader, err := lb.dataStore.GetFile(context.Background(), objectKey)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting file: %s", objectKey)
	}

	defer reader.Close()

	objectBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.Wrapf(err, "failed reading file: %s", objectKey)
	}

	return objectBytes, nil
}

func (lb *ledgerBufferGCS) storeObject(ledgerObject []byte, sequence uint32) {
	lb.priorityQueueLock.Lock()
	defer lb.priorityQueueLock.Unlock()

	lb.ledgerPriorityQueue.Push(ledgerBatchObject{
		payload:     ledgerObject,
		startLedger: int(sequence),
	})

	// Check if the nextLedger is the next item in the priority queue
	for lb.ledgerPriorityQueue.Len() > 0 && lb.currentLedger == uint32(lb.ledgerPriorityQueue.Peek().startLedger) {
		item := lb.ledgerPriorityQueue.Pop()
		lb.ledgerQueue <- item.payload
		lb.currentLedger += lb.config.LedgerBatchConfig.LedgersPerFile
	}
}

func (lb *ledgerBufferGCS) getFromLedgerQueue() ([]byte, error) {
	for {
		select {
		case <-lb.context.Done():
			log.Info("Stopping getFromLedgerQueue due to context cancellation")
			close(lb.done)
			return nil, lb.context.Err()
		case compressedBinary := <-lb.ledgerQueue:
			// Add next task to the TaskQueue
			lb.pushTaskQueue()

			reader := bytes.NewReader(compressedBinary)
			lcmBinary, err := lb.decoder.Unzip(reader)
			if err != nil {
				return nil, err
			}

			return lcmBinary, nil
		}
	}
}
