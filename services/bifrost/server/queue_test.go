package server

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stellar/go/services/bifrost/common"
	"github.com/stellar/go/services/bifrost/queue"
	"github.com/stellar/go/services/bifrost/stellar"
	"github.com/stellar/go/support/errors"
	"github.com/stretchr/testify/mock"
)

func TestPollTransactionQueueShouldRetryOnErrors(t *testing.T) {
	queueMock := &QueueMock{}
	server := Server{
		TransactionsQueue:          queueMock,
		StellarAccountConfigurator: &stellar.AccountConfigurator{},
		log: common.CreateLogger("test server"),
	}
	defaultQueueRetryDelay = time.Nanosecond
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Millisecond)
	defer cancel()

	retried := make(chan struct{}, 1)
	queueMock.Mock.On("IsEmpty").
		Return(false, errors.New("test, please ignore")).
		Once()

	queueMock.Mock.On("IsEmpty").
		Return(false, errors.New("test, please ignore")).
		Run(func(_ mock.Arguments) {
			retried <- struct{}{}
			cancel()
		})

	// when
	go server.poolTransactionsQueue(ctx)

	// then
	select {
	case <-retried:
	case <-ctx.Done():
		t.Fatal("timeout before stub got called")
		return
	}

	queueMock.Mock.AssertExpectations(t)
}

func TestPollTransactionQueueShouldNotSleepWhenQueueHasElements(t *testing.T) {
	var counter uint64 = 0
	stub := func(func(queue.Transaction) error) error {
		atomic.AddUint64(&counter, 1)
		return nil
	}
	server := Server{
		TransactionsQueue:          queuedTransactionStub(stub),
		StellarAccountConfigurator: &stellar.AccountConfigurator{},
		log: common.CreateLogger("test server"),
	}
	defaultQueueRetryDelay = time.Second
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// when
	go server.poolTransactionsQueue(ctx)

	// then
	for {
		select {
		case <-ctx.Done():
			t.Fatal("timeout before stub got called")
		default:
			if atomic.LoadUint64(&counter) > 1 {
				return
			}
		}
	}
}

func TestPollTransactionQueueShouldExitWhenCtxClosed(t *testing.T) {
	queueMock := &QueueMock{}
	server := Server{
		TransactionsQueue:          queueMock,
		StellarAccountConfigurator: &stellar.AccountConfigurator{},
		log: common.CreateLogger("test server"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	cancel()

	// when
	server.poolTransactionsQueue(ctx)

	// then no call IsEmpty
	queueMock.Mock.AssertNumberOfCalls(t, "IsEmpty", 0)
}

func TestPollTransactionQueueShouldNotBlockWhileProcessing(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Done()

	var counter uint64 = 0
	stub := func(func(queue.Transaction) error) error {
		atomic.AddUint64(&counter, 1)
		wg.Wait() // block until test completed
		return nil
	}
	server := Server{
		TransactionsQueue:          queuedTransactionStub(stub),
		StellarAccountConfigurator: &stellar.AccountConfigurator{},
		log: common.CreateLogger("test server"),
	}
	defaultQueueRetryDelay = time.Second
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// when
	go server.poolTransactionsQueue(ctx)

	// then test should not timeout
	for {
		select {
		case <-ctx.Done():
			t.Fatal("timeout before stub got called")
		default:
			if atomic.LoadUint64(&counter) > 1 {
				return
			}
		}
	}
}

// queuedTransactionStub is a test helper type to stub the `WithQueuedTransaction method` only
type queuedTransactionStub func(func(queue.Transaction) error) error

func (s queuedTransactionStub) QueueAdd(_ queue.Transaction) error {
	return errors.New("not supported")
}
func (s queuedTransactionStub) WithQueuedTransaction(f func(queue.Transaction) error) error {
	return s(f)
}
func (s queuedTransactionStub) IsEmpty() (bool, error) {
	return false, nil
}

// QueueMock can be used to mock any method of the Queue interface
type QueueMock struct {
	mock.Mock
}

func (m *QueueMock) QueueAdd(t queue.Transaction) error {
	a := m.Called(t)
	return a.Error(0)
}

func (m *QueueMock) WithQueuedTransaction(f func(queue.Transaction) error) error {
	a := m.Called(f)
	return a.Error(0)
}

func (m *QueueMock) IsEmpty() (bool, error) {
	a := m.Called()
	return a.Get(0).(bool), a.Error(1)

}
