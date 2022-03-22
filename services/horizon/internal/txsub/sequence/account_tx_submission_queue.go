package sequence

import (
	"container/heap"
	"time"
)

// AccountTxSubmissionQueue manages the submission queue for a single source account. The
// transaction system uses Push to enqueue submissions for given sequence
// numbers.
//
// AccountTxSubmissionQueue maintains a priority queue of pending submissions, and when updated
// (via the NotifyLastAccountSequence() method) with the current sequence number of the account
// being managed, queued submissions that can be acted upon will be unblocked.
//
type AccountTxSubmissionQueue struct {
	lastActiveAt            time.Time
	timeout                 time.Duration
	lastSeenAccountSequence uint64
	queue                   pqueue
}

// NewAccountTxSubmissionQueue creates a new *AccountTxSubmissionQueue
func NewAccountTxSubmissionQueue() *AccountTxSubmissionQueue {
	result := &AccountTxSubmissionQueue{
		lastActiveAt: time.Now(),
		timeout:      10 * time.Second,
		queue:        nil,
	}

	heap.Init(&result.queue)

	return result
}

// Size returns the count of currently buffered submissions in the queue.
func (q *AccountTxSubmissionQueue) Size() int {
	return len(q.queue)
}

// Push enqueues the intent to submit a transaction at the provided sequence
// number and returns a channel that will emit when it is safe for the client
// to do so.
//
// Push does not perform any triggering (which
// occurs in NotifyLastAccountSequence(), even if the current sequence number for this queue is
// the same as the provided sequence, to keep internal complexity much lower.
// Given that, the recommended usage pattern is:
//
// 1. Push the submission onto the queue
// 2. Load the current sequence number for the source account from the DB
// 3. Call NotifyLastAccountSequence() with the result from step 2 to trigger the submission if
//		possible
func (q *AccountTxSubmissionQueue) Push(sequence uint64, minSeqNum *uint64) <-chan error {
	ch := make(chan error, 1)
	// From CAP 19: If minSeqNum is nil, the tx is only valid when sourceAccount's sequence number is seqNum - 1.
	// Otherwise, valid when sourceAccount's sequence number n satisfies minSeqNum <= n < tx.seqNum.
	effectiveMinSeqNum := sequence - 1
	if minSeqNum != nil {
		effectiveMinSeqNum = *minSeqNum
	}
	heap.Push(&q.queue, item{
		MinAccSeqNum: effectiveMinSeqNum,
		MaxAccSeqNum: sequence - 1,
		Chan:         ch,
	})
	return ch
}

// NotifyLastAccountSequence notifies the queue that the provided sequence number is the latest
// seen value for the account that this queue manages submissions for.
//
// This function is monotonic... calling it with a sequence number lower than
// the latest seen sequence number is a noop.
func (q *AccountTxSubmissionQueue) NotifyLastAccountSequence(sequence uint64) {
	if q.lastSeenAccountSequence <= sequence {
		q.lastSeenAccountSequence = sequence
	}

	wasChanged := false

	// We need to traverse the full queue (ordered by MaxAccSeqNum)
	// in case there is a transaction with a submittable MinSeqNum we can use later on.
	for i := 0; i < q.Size(); {
		tx := q.queue[i]

		removeWithErr := func(err error) {
			tx.Chan <- err
			close(tx.Chan)
			wasChanged = true
			heap.Remove(&q.queue, i)
		}

		if q.lastSeenAccountSequence > tx.MaxAccSeqNum {
			// The transaction and account sequences will never match
			removeWithErr(ErrBadSequence)
		} else if q.lastSeenAccountSequence >= tx.MinAccSeqNum {
			// within range, ready to submit!
			removeWithErr(nil)
		} else {
			// we only need to increment the heap index when we don't remove
			// an item
			i++
		}
	}

	// if we modified the queue, bump the timeout for this queue
	if wasChanged {
		q.lastActiveAt = time.Now()
		return
	}

	// if the queue wasn't changed, see if it is too old, clear
	// it and make room for other's
	if time.Since(q.lastActiveAt) > q.timeout {
		for q.Size() > 0 {
			entry := q.queue.Pop().(item)
			entry.Chan <- ErrBadSequence
			close(entry.Chan)
		}
	}
}

// item is a element of the priority queue
type item struct {
	MinAccSeqNum uint64 // minimum account sequence required to send the transaction
	MaxAccSeqNum uint64 // maximum account sequence required to send the transaction
	Chan         chan error
}

// pqueue is a priority queue used by AccountTxSubmissionQueue to manage buffered submissions.  It
// implements heap.Interface.
type pqueue []item

func (pq pqueue) Len() int { return len(pq) }

func (pq pqueue) Less(i, j int) bool {
	// To maximize tx submission opportunity, order transactions by the account sequence
	// which would result from successful submission (MaxAccSeqNum+1) but,
	// if those are the same, by higher minimum sequence since there is less margin to send those
	// (a smaller interval).
	if pq[i].MaxAccSeqNum != pq[j].MaxAccSeqNum {
		return pq[i].MaxAccSeqNum < pq[j].MaxAccSeqNum
	}
	return pq[i].MinAccSeqNum > pq[j].MinAccSeqNum
}

func (pq pqueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *pqueue) Push(x interface{}) {
	*pq = append(*pq, x.(item))
}

func (pq *pqueue) Pop() interface{} {
	old := *pq
	n := len(old)
	result := old[n-1]
	*pq = old[0 : n-1]
	return result
}
