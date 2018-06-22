package txsub

import (
	"context"
	"testing"
	"time"

	"github.com/stellar/go/build"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/txsub/sequence"
	"github.com/stretchr/testify/assert"
)

type TestStructSystem struct {
	Ctx       context.Context
	Submitter *MockSubmitter
	Results   *MockResultProvider
	Sequences *MockSequenceProvider
	System    *System
	R         Result
	NoResults Result
	SuccessTx Result
	BadSeq    SubmissionResult
}

func setupTestSystem() *TestStructSystem {
	ts := &TestStructSystem{}

	ts.Ctx = NewTestContext()
	ts.Submitter = &MockSubmitter{}
	ts.Results = &MockResultProvider{}
	ts.Sequences = &MockSequenceProvider{}

	ts.System = &System{
		Pending:           NewDefaultSubmissionList(),
		Submitter:         ts.Submitter,
		Results:           ts.Results,
		Sequences:         ts.Sequences,
		SubmissionQueue:   sequence.NewManager(),
		NetworkPassphrase: build.TestNetwork.Passphrase,
	}

	ts.NoResults = Result{Err: ErrNoResults}
	ts.SuccessTx = Result{
		Hash:           "2374e99349b9ef7dba9a5db3339b78fda8f34777b1af33ba468ad5c0df946d4d",
		LedgerSequence: 2,
		EnvelopeXDR:    "AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAABAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAAAO5rKAAAAAAAAAAABVvwF9wAAAECDzqvkQBQoNAJifPRXDoLhvtycT3lFPCQ51gkdsFHaBNWw05S/VhW0Xgkr0CBPE4NaFV2Kmcs3ZwLmib4TRrML",
		ResultXDR:      "I3Tpk0m57326ml2zM5t4/ajzR3exrzO6RorVwN+UbU0AAAAAAAAAZAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAA==",
	}

	ts.BadSeq = SubmissionResult{
		Err: ErrBadSequence,
	}

	ts.Sequences.Results = map[string]uint64{
		"GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H": 0,
	}

	return ts
}

func TestTxsub_Submit(t *testing.T) {
	t.Run("returns the result provided by the ResultProvider", func(t *testing.T) {
		ts := setupTestSystem()
		ts.Results.Results = []Result{ts.SuccessTx}
		r := <-ts.System.Submit(ts.Ctx, ts.SuccessTx.EnvelopeXDR)

		assert.Nil(t, r.Err)
		assert.Equal(t, ts.SuccessTx.Hash, r.Hash)
		assert.False(t, ts.Submitter.WasSubmittedTo)

	})

	t.Run("returns the error from submission if no result is found by hash and the submitter returns an error", func(t *testing.T) {
		ts := setupTestSystem()
		ts.Submitter.R.Err = errors.New("busted for some reason")
		r := <-ts.System.Submit(ts.Ctx, ts.SuccessTx.EnvelopeXDR)

		assert.NotNil(t, r.Err)
		assert.True(t, ts.Submitter.WasSubmittedTo)
		assert.EqualValues(t, 0, ts.System.Metrics.SuccessfulSubmissionsMeter.Count())
		assert.EqualValues(t, 1, ts.System.Metrics.FailedSubmissionsMeter.Count())
		assert.EqualValues(t, 1, ts.System.Metrics.SubmissionTimer.Count())
	})

	t.Run("if the error is bad_seq and the result at the transaction's sequence number is for the same hash, return result", func(t *testing.T) {
		ts := setupTestSystem()
		ts.Submitter.R = ts.BadSeq
		ts.Results.Results = []Result{ts.NoResults, ts.SuccessTx}

		r := <-ts.System.Submit(ts.Ctx, ts.SuccessTx.EnvelopeXDR)

		assert.Nil(t, r.Err)
		assert.Equal(t, ts.SuccessTx.Hash, r.Hash)
		assert.True(t, ts.Submitter.WasSubmittedTo)
	})

	t.Run("if error is bad_seq and no result is found, return error", func(t *testing.T) {
		ts := setupTestSystem()
		ts.Submitter.R = ts.BadSeq
		r := <-ts.System.Submit(ts.Ctx, ts.SuccessTx.EnvelopeXDR)

		assert.NotNil(t, r.Err)
		assert.True(t, ts.Submitter.WasSubmittedTo)
	})

	t.Run("if no result found and no error submitting, add to open transaction list", func(t *testing.T) {
		ts := setupTestSystem()
		_ = ts.System.Submit(ts.Ctx, ts.SuccessTx.EnvelopeXDR)
		pending := ts.System.Pending.Pending(ts.Ctx)
		assert.Equal(t, len(pending), 1)
		assert.Equal(t, pending[0], ts.SuccessTx.Hash)
		assert.EqualValues(t, 1, ts.System.Metrics.SuccessfulSubmissionsMeter.Count())
		assert.EqualValues(t, 0, ts.System.Metrics.FailedSubmissionsMeter.Count())
		assert.EqualValues(t, 1, ts.System.Metrics.SubmissionTimer.Count())
	})
}

func TestTxsub_Tick(t *testing.T) {
	ts := setupTestSystem()

	t.Run(" no-ops if there are no open submissions", func(t *testing.T) {
		ts.System.Tick(ts.Ctx)
	})

	t.Run("finishes any available transactions", func(t *testing.T) {
		l := make(chan Result, 1)
		ts.System.Pending.Add(ts.Ctx, ts.SuccessTx.Hash, l)
		ts.System.Tick(ts.Ctx)
		assert.Equal(t, 0, len(l))
		assert.Equal(t, 1, len(ts.System.Pending.Pending(ts.Ctx)))

		ts.Results.Results = []Result{ts.SuccessTx}
		ts.System.Tick(ts.Ctx)

		assert.Equal(t, 1, len(l))
		assert.Equal(t, 0, len(ts.System.Pending.Pending(ts.Ctx)))
	})

	t.Run("removes old submissions that have timed out", func(t *testing.T) {
		l := make(chan Result, 1)
		ts.System.SubmissionTimeout = 100 * time.Millisecond
		ts.System.Pending.Add(ts.Ctx, ts.SuccessTx.Hash, l)
		<-time.After(101 * time.Millisecond)
		ts.System.Tick(ts.Ctx)

		assert.Equal(t, 0, len(ts.System.Pending.Pending(ts.Ctx)))
		assert.Equal(t, 1, len(l))
		<-l
		select {
		case _, stillOpen := <-l:
			assert.False(t, stillOpen)
		default:
			panic("could not read from listener")
		}
	})
}
