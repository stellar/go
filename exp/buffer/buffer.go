package buffer

import (
	"errors"
	"sync"
)

var ErrBatchFull = errors.New("batch full")

type PaymentBufferNewBatchIDFunc func() string

type PaymentBufferSubmitBatchFunc func(batchID string, amount int64, payments []PaymentParams)

type PaymentBufferConfig struct {
	MaxBatchSize    int
	NewBatchIDFunc  PaymentBufferNewBatchIDFunc
	SubmitBatchFunc PaymentBufferSubmitBatchFunc
}

type PaymentBuffer struct {
	maxBatchSize     int
	mu               sync.Mutex
	newBatchIDFunc   PaymentBufferNewBatchIDFunc
	submitBatchFunc  PaymentBufferSubmitBatchFunc
	batchID          string
	batch            []PaymentParams
	batchTotalAmount int64
	batchReady       chan struct{}
	waitFinished     chan struct{}
}

func NewPaymentBuffer(c PaymentBufferConfig) *PaymentBuffer {
	b := &PaymentBuffer{
		maxBatchSize:    c.MaxBatchSize,
		newBatchIDFunc:  c.NewBatchIDFunc,
		submitBatchFunc: c.SubmitBatchFunc,
		batchReady:      make(chan struct{}, 1),
		waitFinished:    make(chan struct{}),
	}
	b.resetBatch()
	go b.flushLoop()
	return b
}

type PaymentParams struct {
	Amount int64
	Memo   int64
}

func (b *PaymentBuffer) Payment(p PaymentParams) (batchID string, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	// TODO: Return error if payment would overflow batch total amount.
	if len(b.batch) == b.maxBatchSize {
		return "", ErrBatchFull
	}
	b.batch = append(b.batch, p)
	b.batchTotalAmount += p.Amount
	batchID = b.batchID
	select {
	case b.batchReady <- struct{}{}:
	default:
	}
	return
}

func (b *PaymentBuffer) Done() {
	close(b.batchReady)
}

func (b *PaymentBuffer) Wait() {
	<-b.waitFinished
}

func (b *PaymentBuffer) flushLoop() {
	defer close(b.waitFinished)
	for {
		select {
		case _, open := <-b.batchReady:
			if !open {
				return
			}
			b.flush()
		default:
			select {
			case _, open := <-b.batchReady:
				if !open {
					return
				}
				b.flush()
			case b.waitFinished <- struct{}{}:
			}
		}
	}
}

func (b *PaymentBuffer) flush() {
	var batchID string
	var batch []PaymentParams
	var batchTotalAmount int64
	func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		batchID = b.batchID
		batch = b.batch
		batchTotalAmount = b.batchTotalAmount
		b.resetBatch()
	}()
	if len(batch) == 0 {
		return
	}
	b.submitBatchFunc(batchID, batchTotalAmount, batch)
}

func (b *PaymentBuffer) resetBatch() {
	b.batchID = b.newBatchIDFunc()
	b.batch = nil
	b.batchTotalAmount = 0
}
