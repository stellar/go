package sequence

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Push adds the provided channel on to the priority queue
func TestQueue_Push(t *testing.T) {
	queue := NewQueue()
	assert.Equal(t, 0, queue.Size())

	queue.Push(2)
	assert.Equal(t, 1, queue.Size())
	_, s := queue.head()
	assert.EqualValues(t, 2, s)

	queue.Push(1)
	assert.Equal(t, 2, queue.Size())
	_, s = queue.head()
	assert.EqualValues(t, 1, s)
}

// Update clears the queue if the head has not been released within the time limit
func TestQueue_Timeout(t *testing.T) {
	queue := NewQueue()
	queue.timeout = 1 * time.Millisecond
	result := queue.Push(2)
	<-time.After(10 * time.Millisecond)
	queue.Update(0)

	assert.Equal(t, 0, queue.Size())
	assert.Equal(t, ErrBadSequence, <-result)
}

// Update removes sequences that are submittable or in the past
func TestQueue_Update(t *testing.T) {
	queue := NewQueue()
	results := []<-chan error{
		queue.Push(1),
		queue.Push(2),
		queue.Push(3),
		queue.Push(4),
	}

	queue.Update(2)

	// the update above signifies that 2 is the accounts current sequence,
	// meaning that 3 is submittable, and so only 4 should still be queued
	assert.Equal(t, 1, queue.Size())
	_, s := queue.head()
	assert.EqualValues(t, 4, s)

	queue.Update(4)
	assert.Equal(t, 0, queue.Size())

	assert.Equal(t, ErrBadSequence, <-results[0])
	assert.Equal(t, ErrBadSequence, <-results[1])
	assert.Equal(t, nil, <-results[2])
	assert.Equal(t, ErrBadSequence, <-results[3])
}
