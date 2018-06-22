package txsub

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type TestStructOpenSubmissionList struct {
	Ctx       context.Context
	List      OpenSubmissionList
	RealList  *submissionList
	Hashes    []string
	Listeners []chan Result
	R         Result
}

func setupTestOpenSubmissionList() *TestStructOpenSubmissionList {
	ts := &TestStructOpenSubmissionList{}
	ts.Ctx = NewTestContext()

	ts.List = NewDefaultSubmissionList()
	ts.RealList = ts.List.(*submissionList)
	ts.Hashes = []string{
		"0000000000000000000000000000000000000000000000000000000000000000",
		"0000000000000000000000000000000000000000000000000000000000000001",
	}

	ts.Listeners = []chan Result{
		make(chan Result, 1),
		make(chan Result, 1),
	}

	return ts
}

func setupTestOpenSubmissionListFinish() *TestStructOpenSubmissionList {
	ts := setupTestOpenSubmissionList()
	ts.List.Add(ts.Ctx, ts.Hashes[0], ts.Listeners[0])
	ts.List.Add(ts.Ctx, ts.Hashes[0], ts.Listeners[1])
	ts.R = Result{
		Hash: ts.Hashes[0],
	}
	ts.List.Finish(ts.Ctx, ts.R)

	return ts
}

func TestSubmissionList_Add(t *testing.T) {
	t.Run("adds an entry to the submission list when a new hash is used", func(t *testing.T) {
		ts := setupTestOpenSubmissionList()
		ts.List.Add(ts.Ctx, ts.Hashes[0], ts.Listeners[0])
		sub := ts.RealList.submissions[ts.Hashes[0]]
		assert.Equal(t, sub.Hash, ts.Hashes[0])
		assert.WithinDuration(t, sub.SubmittedAt, time.Now(), 1*time.Second)

		// drop the send side of the channel by casting to listener
		var l Listener = ts.Listeners[0]
		assert.Equal(t, sub.Listeners[0], l)
	})
	t.Run("adds an listener to an existing entry when a hash is used with a new listener", func(t *testing.T) {
		ts := setupTestOpenSubmissionList()
		ts.List.Add(ts.Ctx, ts.Hashes[0], ts.Listeners[0])
		sub := ts.RealList.submissions[ts.Hashes[0]]
		st := sub.SubmittedAt
		<-time.After(20 * time.Millisecond)
		ts.List.Add(ts.Ctx, ts.Hashes[0], ts.Listeners[1])

		// increases the size of the listener
		assert.Equal(t, 2, len(sub.Listeners))
		// doesn't update the submitted at time
		assert.True(t, st == sub.SubmittedAt)
	})
	t.Run("panics when the listener is not buffered", func(t *testing.T) {
		ts := setupTestOpenSubmissionList()
		assert.Panics(t, func() { ts.List.Add(ts.Ctx, ts.Hashes[0], make(Listener)) })
	})
	t.Run("errors when the provided hash is not 64-bytes", func(t *testing.T) {
		ts := setupTestOpenSubmissionList()
		err := ts.List.Add(ts.Ctx, "123", ts.Listeners[0])
		assert.NotNil(t, err)
	})
}

func TestSubmissionList_Clean(t *testing.T) {
	ts := setupTestOpenSubmissionList()
	ts.List.Add(ts.Ctx, ts.Hashes[0], ts.Listeners[0])
	<-time.After(200 * time.Millisecond)
	ts.List.Add(ts.Ctx, ts.Hashes[1], ts.Listeners[1])
	left, err := ts.List.Clean(ts.Ctx, 200*time.Millisecond)

	assert.Nil(t, err)
	assert.Equal(t, left, 1)

	t.Run("removes submissions older than the maxAge provided", func(t *testing.T) {
		_, ok := ts.RealList.submissions[ts.Hashes[0]]
		assert.False(t, ok)
	})

	t.Run("leaves submissions that are younger than the maxAge provided", func(t *testing.T) {
		_, ok := ts.RealList.submissions[ts.Hashes[1]]
		assert.True(t, ok)
	})

	t.Run("closes any cleaned listeners", func(t *testing.T) {
		assert.Equal(t, len(ts.Listeners[0]), 1)
		<-ts.Listeners[0]
		select {
		case _, stillOpen := <-ts.Listeners[0]:
			assert.False(t, stillOpen)
		default:
			panic("cleaned listener is still open")
		}
	})
}

func TestSubmissionList_Finish(t *testing.T) {

	t.Run("writes to every listener", func(t *testing.T) {
		ts := setupTestOpenSubmissionListFinish()
		r1, ok1 := <-ts.Listeners[0]
		assert.EqualValues(t, ts.R, r1)
		assert.True(t, ok1)

		r2, ok2 := <-ts.Listeners[1]
		assert.EqualValues(t, ts.R, r2)
		assert.True(t, ok2)
	})

	t.Run("removes the entry", func(t *testing.T) {
		ts := setupTestOpenSubmissionListFinish()
		_, ok := ts.RealList.submissions[ts.Hashes[0]]
		assert.False(t, ok)
	})

	t.Run("closes every listener", func(t *testing.T) {
		ts := setupTestOpenSubmissionListFinish()
		_, _ = <-ts.Listeners[0]
		_, more := <-ts.Listeners[0]
		assert.False(t, more)

		_, _ = <-ts.Listeners[1]
		_, more = <-ts.Listeners[1]
		assert.False(t, more)
	})

	t.Run("works when the noone is waiting for the result", func(t *testing.T) {
		ts := setupTestOpenSubmissionListFinish()
		err := ts.List.Finish(ts.Ctx, ts.R)
		assert.Nil(t, err)
	})
}

func TestSubmissionList_Pending(t *testing.T) {
	ts := setupTestOpenSubmissionList()
	assert.Equal(t, len(ts.List.Pending(ts.Ctx)), 0)
	ts.List.Add(ts.Ctx, ts.Hashes[0], ts.Listeners[0])
	assert.Equal(t, len(ts.List.Pending(ts.Ctx)), 1)
	ts.List.Add(ts.Ctx, ts.Hashes[1], ts.Listeners[1])
	assert.Equal(t, len(ts.List.Pending(ts.Ctx)), 2)
}
