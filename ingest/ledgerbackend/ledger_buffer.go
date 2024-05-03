package ledgerbackend

import (
	"bytes"
	"context"
	"io"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stellar/go/support/collections/heap"
	"github.com/stellar/go/support/compressxdr"
	"github.com/stellar/go/support/datastore"
	"github.com/stellar/go/support/log"
)

type ledgerBatchObject struct {
	payload     []byte
	startLedger int // Ledger sequence used as the priority for the priorityqueue.
}

type ledgerBuffer struct {
	config              CloudStorageBackendConfig
	dataStore           datastore.DataStore
	taskQueue           chan uint32 // buffer next object read
	ledgerQueue         chan []byte // order corrected lcm batches
	ledgerPriorityQueue *heap.Heap[ledgerBatchObject]
	priorityQueueLock   sync.Mutex
	done                chan struct{}

	// keep track of the ledgers to be processed and the next ordering
	// the ledgers should be buffered
	currentLedger  uint32
	nextTaskLedger uint32
	ledgerRange    Range

	// passed through from CloudStorageBackend to control lifetime of ledgerBuffer instance
	context context.Context
	cancel  context.CancelCauseFunc
	decoder compressxdr.XDRDecoder
}

func (csb *CloudStorageBackend) newLedgerBuffer(ledgerRange Range) (*ledgerBuffer, error) {
	less := func(a, b ledgerBatchObject) bool {
		return a.startLedger < b.startLedger
	}
	pq := heap.New(less, int(csb.config.BufferSize))

	done := make(chan struct{})

	ledgerBuffer := &ledgerBuffer{
		config:              csb.config,
		dataStore:           csb.dataStore,
		taskQueue:           make(chan uint32, csb.config.BufferSize),
		ledgerQueue:         make(chan []byte, csb.config.BufferSize),
		ledgerPriorityQueue: pq,
		done:                done,
		currentLedger:       ledgerRange.from,
		nextTaskLedger:      ledgerRange.from,
		ledgerRange:         ledgerRange,
		context:             csb.context,
		cancel:              csb.cancel,
		decoder:             csb.decoder,
	}

	// Workers to read LCM files
	for i := uint32(0); i < csb.config.NumWorkers; i++ {
		go ledgerBuffer.worker()
	}

	// Start the ledgerBuffer
	for i := 0; i <= int(csb.config.BufferSize); i++ {
		if csb.ledgerBuffer.nextTaskLedger > ledgerRange.to && ledgerRange.bounded {
			break
		}
		csb.ledgerBuffer.pushTaskQueue()
	}

	return ledgerBuffer, nil
}

func (lb *ledgerBuffer) pushTaskQueue() {
	// In bounded mode, don't queue past the end ledger
	if lb.nextTaskLedger > lb.ledgerRange.to && lb.ledgerRange.bounded {
		return
	}
	lb.taskQueue <- lb.nextTaskLedger
	lb.nextTaskLedger += lb.config.LedgerBatchConfig.LedgersPerFile
}

func (lb *ledgerBuffer) worker() {
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
			for retryCount <= lb.config.RetryLimit {
				ledgerObject, err := lb.getLedgerObject(sequence)
				if err != nil {
					if err == os.ErrNotExist {
						// ledgerObject not found and unbounded
						if !lb.ledgerRange.bounded {
							time.Sleep(lb.config.RetryWait * time.Second)
							continue
						}
						lb.cancel(err)
						return
					}
					if retryCount == lb.config.RetryLimit {
						err = errors.New("maximum retries exceeded for object reads")
						lb.cancel(err)
						return
					}
					retryCount++
					time.Sleep(lb.config.RetryWait * time.Second)
				}

				// Add to priority queue and continue to next task
				lb.storeObject(ledgerObject, sequence)
				break
			}
		}
	}
}

func (lb *ledgerBuffer) getLedgerObject(sequence uint32) ([]byte, error) {
	objectKey := lb.config.LedgerBatchConfig.GetObjectKeyFromSequenceNumber(sequence)

	ok, err := lb.dataStore.Exists(context.Background(), objectKey)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, os.ErrNotExist
	}

	reader, err := lb.dataStore.GetFile(context.Background(), objectKey)
	if err != nil {
		return nil, err
	}

	defer reader.Close()

	objectBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.Wrapf(err, "failed reading file: %s", objectKey)
	}

	return objectBytes, nil
}

func (lb *ledgerBuffer) storeObject(ledgerObject []byte, sequence uint32) {
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

func (lb *ledgerBuffer) getFromLedgerQueue() ([]byte, error) {
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
