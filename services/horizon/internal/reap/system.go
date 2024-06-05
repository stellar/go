package reap

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	herrors "github.com/stellar/go/services/horizon/internal/errors"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/toid"
)

// DeleteUnretainedHistory removes all data associated with unretained ledgers.
func (r *System) DeleteUnretainedHistory(ctx context.Context) error {
	// RetentionCount of 0 indicates "keep all history"
	if r.RetentionCount == 0 {
		return nil
	}

	var (
		latest      = r.ledgerState.CurrentStatus()
		targetElder = (latest.HistoryLatest - int32(r.RetentionCount)) + 1
	)

	if targetElder < latest.HistoryElder {
		return nil
	}

	err := r.clearBefore(ctx, latest.HistoryElder, targetElder)
	if err != nil {
		return err
	}

	log.
		WithField("new_elder", targetElder).
		Info("reaper succeeded")

	return nil
}

// Run triggers the reaper system to update itself, deleted unretained history
// if it is the appropriate time.
func (r *System) Run() {
	for {
		select {
		case <-time.After(1 * time.Hour):
			r.runOnce(r.ctx)
		case <-r.ctx.Done():
			return
		}
	}
}

// RegisterMetrics registers the prometheus metrics
func (s *System) RegisterMetrics(registry *prometheus.Registry) {
	registry.MustRegister(s.deleteBatchDuration, s.rowsDeleted)
}

func (r *System) Shutdown() {
	r.cancel()
	if err := r.HistoryQ.Close(); err != nil {
		log.Errorf("reaper could not close db connection: %s", err)
	}
}

func (r *System) runOnce(ctx context.Context) {
	defer func() {
		if rec := recover(); rec != nil {
			err := herrors.FromPanic(rec)
			log.Errorf("reaper panicked: %s", err)
			herrors.ReportToSentry(err, nil)
		}
	}()

	err := r.DeleteUnretainedHistory(ctx)
	if err != nil {
		log.Errorf("reaper failed: %s", err)
	}
}

// Work backwards in 50k (by default, otherwise configurable via the CLI) ledger
// blocks to prevent using all the CPU.
//
// This runs every hour, so we need to make sure it doesn't run for longer than
// an hour.
//
// Current ledger at 2024-04-04s is 51,092,283, so 50k means 1021 batches. At 1
// batch/second, that seems like a reasonable balance between running under an
// hour, and slowing it down enough to leave some CPU for other processes.
var sleep = 1 * time.Second

func (r *System) clearBefore(ctx context.Context, startSeq, endSeq int32) error {
	batchSize := int32(r.RetentionBatch)
	if batchSize <= 0 {
		return fmt.Errorf("invalid batch size for reaping (%d)", batchSize)
	}

	log.WithField("start_ledger", startSeq).
		WithField("end_ledger", endSeq).
		WithField("batch_size", batchSize).
		Info("reaper: deleting history outside retention window")

	for batchEndSeq := endSeq - 1; batchEndSeq >= startSeq; batchEndSeq -= batchSize {
		batchStartSeq := batchEndSeq - batchSize
		if batchStartSeq < startSeq {
			batchStartSeq = startSeq
		}

		if err := r.deleteBatch(ctx, batchStartSeq, batchEndSeq); err != nil {
			return err
		}
		time.Sleep(sleep)
	}

	return nil
}

func (r *System) deleteBatch(ctx context.Context, batchStartSeq int32, batchEndSeq int32) error {
	batchStart, batchEnd, err := toid.LedgerRangeInclusive(batchStartSeq, batchEndSeq)
	if err != nil {
		return err
	}

	startTime := time.Now()
	err = r.HistoryQ.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "Error in begin")
	}
	defer r.HistoryQ.Rollback()

	count, err := r.HistoryQ.DeleteRangeAll(ctx, batchStart, batchEnd)
	if err != nil {
		return errors.Wrap(err, "Error in DeleteRangeAll")
	}

	err = r.HistoryQ.Commit()
	if err != nil {
		return errors.Wrap(err, "Error in commit")
	}

	elapsedSeconds := time.Since(startTime).Seconds()
	log.WithField("start_ledger", batchStartSeq).
		WithField("end_ledger", batchEndSeq).
		WithField("rows_deleted", strconv.FormatInt(count, 10)).
		WithField("duration", elapsedSeconds).
		Info("reaper: successfully deleted batch")

	r.rowsDeleted.Observe(float64(count))
	r.deleteBatchDuration.Observe(elapsedSeconds)
	return nil
}
