package sequence

import (
	"sort"
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
	queue                   submissionQueue
}

// NewAccountTxSubmissionQueue creates a new *AccountTxSubmissionQueue
func NewAccountTxSubmissionQueue() *AccountTxSubmissionQueue {
	result := &AccountTxSubmissionQueue{
		lastActiveAt: time.Now(),
		timeout:      10 * time.Second,
		queue:        nil,
	}
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
	// From CAP 21: If minSeqNum is nil, the txToSubmit is only valid when sourceAccount's sequence number is seqNum - 1.
	// Otherwise, valid when sourceAccount's sequence number n satisfies minSeqNum <= n < txToSubmit.seqNum.
	effectiveMinSeqNum := sequence - 1
	if minSeqNum != nil {
		effectiveMinSeqNum = *minSeqNum
	}
	q.queue.insert(txToSubmit{
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
	for i := 0; i < len(q.queue); {
		tx := q.queue[i]

		removeWithErr := func(err error) {
			tx.Chan <- err
			close(tx.Chan)
			wasChanged = true
			q.queue.remove(i)
		}

		if q.lastSeenAccountSequence > tx.MaxAccSeqNum {
			// The transaction and account sequences will never match
			removeWithErr(ErrBadSequence)
		} else if q.lastSeenAccountSequence >= tx.MinAccSeqNum {
			// within range, ready to submit!
			removeWithErr(nil)
		} else {
			// we only need to increment the index when we don't remove
			// a transaction
			i++
		}
	}

	// if we modified the queue, bump the timeout for this queue
	if wasChanged {
		q.lastActiveAt = time.Now()
		return
	}

	// if the queue wasn't changed, see if it is too old, clear
	// it and make room for other submissions
	if time.Since(q.lastActiveAt) > q.timeout {
		for i := 0; i < len(q.queue); i++ {
			c := q.queue[i].Chan
			c <- ErrBadSequence
			close(c)
		}
		q.queue = nil
	}
}

// txToSubmit represents a transaction being tracked in submissionQueue
type txToSubmit struct {
	MinAccSeqNum uint64 // minimum account sequence required to send the transaction
	MaxAccSeqNum uint64 // maximum account sequence required to send the transaction
	Chan         chan error
}

// submissionQueue is a priority queue (implemented as a sorted slice),
// used to track the submission order of transactions.
// Lower indices have higher submission priority.
type submissionQueue []txToSubmit

func (sq *submissionQueue) insert(tx txToSubmit) {
	txHasLessPriorityThanIth := func(i int) bool {
		// To maximize transaction submission opportunity, we prioritize transactions by the account sequence
		// which would result from successful submission (i.e. MaxAccSeqNum+1) but,
		// if those are the same, by higher minimum sequence since there is less margin to send those
		// (a smaller interval).
		if tx.MaxAccSeqNum != (*sq)[i].MaxAccSeqNum {
			return tx.MaxAccSeqNum < (*sq)[i].MaxAccSeqNum
		}
		return tx.MinAccSeqNum > (*sq)[i].MinAccSeqNum
	}
	i := sort.Search(len(*sq), txHasLessPriorityThanIth)
	if len(*sq) == i {
		*sq = append(*sq, tx)
		return
	}
	*sq = append((*sq)[:i+1], (*sq)[i:]...)
	(*sq)[i] = tx
}

func (sq *submissionQueue) remove(i int) {
	*sq = append((*sq)[:i], (*sq)[i+1:]...)
}
