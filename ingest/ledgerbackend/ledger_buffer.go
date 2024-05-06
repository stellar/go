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
	// passed through from CloudStorageBackend to control lifetime of ledgerBuffer instance
	config    CloudStorageBackendConfig
	dataStore datastore.DataStore
	context   context.Context
	cancel    context.CancelCauseFunc
	decoder   compressxdr.XDRDecoder

	// The pipes and data structures below help establish the ledgerBuffer invariant which is
	// the number of tasks (both pending and in-flight) + len(ledgerQueue) + ledgerPriorityQueue.Len()
	// is always less than or equal to the config.BufferSize
	taskQueue           chan uint32                   // Buffer next object read
	ledgerQueue         chan []byte                   // Order corrected lcm batches
	ledgerPriorityQueue *heap.Heap[ledgerBatchObject] // Priority is set to the sequence number
	priorityQueueLock   sync.Mutex

	// done is used to signal the closure of the ledgerBuffer and workers
	done chan struct{}

	// keep track of the ledgers to be processed and the next ordering
	// the ledgers should be buffered
	currentLedger     uint32 // The current ledger that should be popped from ledgerPriorityQueue
	nextTaskLedger    uint32 // The next task ledger that should be added to taskQueue
	ledgerRange       Range
	currentLedgerLock sync.RWMutex
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

	// Start workers to read LCM files
	for i := uint32(0); i < csb.config.NumWorkers; i++ {
		go ledgerBuffer.worker()
	}

	// Upon initialization, the ledgerBuffer invariant is maintained because
	// we create csb.config.BufferSize tasks while the len(ledgerQueue) and ledgerPriorityQueue.Len() are 0.
	// Effectively, this is len(taskQueue) + len(ledgerQueue) + ledgerPriorityQueue.Len() == csb.config.BufferSize
	// which enforces a limit of max tasks (both pending and in-flight) to be equal to csb.config.BufferSize.
	// Note: when a task is in-flight it is no longer in the taskQueue
	// but for easier conceptualization, len(taskQueue) can be interpreted as both pending and in-flight tasks
	// where we assume the workers are empty and not processing any tasks.
	for i := 0; i <= int(csb.config.BufferSize); i++ {
		ledgerBuffer.pushTaskQueue()
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
			for retryCount := uint32(0); retryCount <= lb.config.RetryLimit; {
				ledgerObject, err := lb.downloadLedgerObject(sequence)
				if err != nil {
					if errors.Is(err, os.ErrNotExist) {
						// ledgerObject not found and unbounded
						if !lb.ledgerRange.bounded {
							time.Sleep(lb.config.RetryWait * time.Second)
							continue
						}
						lb.cancel(err)
						return
					}
					if retryCount == lb.config.RetryLimit {
						err = errors.Wrap(err, "maximum retries exceeded for object reads")
						lb.cancel(err)
						return
					}
					retryCount++
					time.Sleep(lb.config.RetryWait * time.Second)
				}

				// When we store an object we still maintain the ledger buffer invariant because
				// at this point the current task is finished and we add 1 ledger object to the priority queue.
				// Thus, the number of tasks decreases by 1 and the priority queue length increases by 1.
				// This keeps the overall total the same (<= BufferSize). As long as the the ledger buffer invariant
				// was maintained in the previous state, it is still maintained during this state transition.
				lb.storeObject(ledgerObject, sequence)
				break
			}
		}
	}
}

func (lb *ledgerBuffer) downloadLedgerObject(sequence uint32) ([]byte, error) {
	objectKey := lb.config.LedgerBatchConfig.GetObjectKeyFromSequenceNumber(sequence)

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

	lb.currentLedgerLock.Lock()
	defer lb.currentLedgerLock.Unlock()

	lb.ledgerPriorityQueue.Push(ledgerBatchObject{
		payload:     ledgerObject,
		startLedger: int(sequence),
	})

	// Check if the nextLedger is the next item in the ledgerPriorityQueue
	// The ledgerBuffer invariant is maintained here because items are transferred from the ledgerPriorityQueue to the ledgerQueue.
	// Thus the overall sum of ledgerPriorityQueue.Len() + len(lb.ledgerQueue) remains the same.
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
			// The ledger buffer invariant is maintained here because
			// we create an extra task when consuming one item from the ledger queue.
			// Thus len(ledgerQueue) decreases by 1 and the number of tasks increases by 1.
			// The overall sum below remains the same:
			// len(taskQueue) + len(ledgerQueue) + ledgerPriorityQueue.Len() == csb.config.BufferSize
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

func (lb *ledgerBuffer) getLatestLedgerSequence() (uint32, error) {
	lb.currentLedgerLock.Lock()
	defer lb.currentLedgerLock.Unlock()

	if lb.currentLedger == lb.ledgerRange.from {
		return 0, nil
	}

	// Subtract 1 to get the latest ledger in buffer
	return lb.currentLedger - 1, nil
}
