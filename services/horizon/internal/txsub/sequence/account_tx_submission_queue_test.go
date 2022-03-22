//lint:file-ignore U1001 Ignore all unused code, staticcheck doesn't understand testify/suite

package sequence

import (
	"context"
	"testing"
	"time"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type QueueTestSuite struct {
	suite.Suite
	ctx   context.Context
	queue *AccountTxSubmissionQueue
}

func (suite *QueueTestSuite) SetupTest() {
	suite.ctx = test.Context()
	suite.queue = NewAccountTxSubmissionQueue()
}

//Push adds the provided channel on to the priority queue
func (suite *QueueTestSuite) TestQueue_Push() {
	ctx := test.Context()
	_ = ctx

	assert.Equal(suite.T(), 0, suite.queue.Size())

	suite.queue.Push(6, nil)
	assert.Equal(suite.T(), 1, suite.queue.Size())
	entry := suite.queue.queue[0]
	assert.Equal(suite.T(), uint64(5), entry.MinAccSeqNum)
	assert.Equal(suite.T(), uint64(5), entry.MinAccSeqNum)

	min := uint64(1)
	suite.queue.Push(5, &min)
	assert.Equal(suite.T(), 2, suite.queue.Size())
	entry = suite.queue.queue[0]
	assert.Equal(suite.T(), uint64(1), entry.MinAccSeqNum)
	assert.Equal(suite.T(), uint64(4), entry.MaxAccSeqNum)

	suite.queue.Push(5, nil)
	assert.Equal(suite.T(), 3, suite.queue.Size())
	entry = suite.queue.queue[0]
	assert.Equal(suite.T(), uint64(4), entry.MinAccSeqNum)
	assert.Equal(suite.T(), uint64(4), entry.MaxAccSeqNum)

	suite.queue.Push(4, nil)
	assert.Equal(suite.T(), 4, suite.queue.Size())
	entry = suite.queue.queue[0]
	assert.Equal(suite.T(), uint64(3), entry.MinAccSeqNum)
	assert.Equal(suite.T(), uint64(3), entry.MaxAccSeqNum)
}

// Tests the NotifyLastAccountSequence method
func (suite *QueueTestSuite) TestQueue_NotifyLastAccountSequence() {
	// NotifyLastAccountSequence removes sequences that are submittable or in the past
	lowMin := uint64(1)
	results := []<-chan error{
		suite.queue.Push(1, nil),
		suite.queue.Push(2, nil),
		suite.queue.Push(3, nil),
		suite.queue.Push(4, nil),
		suite.queue.Push(4, &lowMin),
	}

	suite.queue.NotifyLastAccountSequence(2)

	// the update above signifies that 2 is the accounts current sequence,
	// meaning that 3 is submittable, and so only 4 (Min/MaxAccSeqNum=3) should remain
	assert.Equal(suite.T(), 1, suite.queue.Size())
	entry := suite.queue.queue[0]
	assert.Equal(suite.T(), uint64(3), entry.MinAccSeqNum)
	assert.Equal(suite.T(), uint64(3), entry.MaxAccSeqNum)

	suite.queue.NotifyLastAccountSequence(4)
	assert.Equal(suite.T(), 0, suite.queue.Size())

	assert.Equal(suite.T(), ErrBadSequence, <-results[0])
	assert.Equal(suite.T(), ErrBadSequence, <-results[1])
	assert.Equal(suite.T(), nil, <-results[2])
	assert.Equal(suite.T(), ErrBadSequence, <-results[3])
	assert.Equal(suite.T(), nil, <-results[4])

	// NotifyLastAccountSequence clears the queue if the head has not been released within the time limit
	suite.queue.timeout = 1 * time.Millisecond
	result := suite.queue.Push(2, nil)
	<-time.After(10 * time.Millisecond)
	suite.queue.NotifyLastAccountSequence(0)

	assert.Equal(suite.T(), 0, suite.queue.Size())
	assert.Equal(suite.T(), ErrBadSequence, <-result)
}

func TestQueueTestSuite(t *testing.T) {
	suite.Run(t, new(QueueTestSuite))
}
