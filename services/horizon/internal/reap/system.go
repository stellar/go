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

	latest, err := r.HistoryQ.GetLatestHistoryLedger(ctx)
	if err != nil {
		return errors.Wrap(err, "error fetching latest history ledger")
	}
	var oldest uint32
	err = r.HistoryQ.ElderLedger(ctx, &oldest)
	if err != nil {
		return errors.Wrap(err, "error fetching elder ledger")
	}

	targetElder := latest - r.RetentionCount + 1
	if latest <= r.RetentionCount || targetElder < oldest {
		log.
			WithField("latest", latest).
			WithField("oldest", oldest).
			WithField("retention_count", r.RetentionCount).
			Info("not enough history to reap")
		return nil
	}

	if err = r.clearBefore(ctx, oldest, targetElder); err != nil {
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

func (r *System) clearBefore(ctx context.Context, startSeq, endSeq uint32) error {
	batchSize := r.RetentionBatch
	if batchSize <= 0 {
		return fmt.Errorf("invalid batch size for reaping (%d)", batchSize)
	}

	log.WithField("start_ledger", startSeq).
		WithField("end_ledger", endSeq).
		WithField("batch_size", batchSize).
		Info("reaper: deleting history outside retention window")

	for batchStartSeq := startSeq; batchStartSeq < endSeq; {
		batchEndSeq := batchStartSeq + batchSize
		if batchEndSeq >= endSeq {
			batchEndSeq = endSeq - 1
		}

		count, err := r.deleteBatch(ctx, batchStartSeq, batchEndSeq)
		if err != nil {
			return err
		}
		if count == 0 {
			next, ok, err := r.HistoryQ.GetNextLedgerSequence(ctx, batchStartSeq)
			if err != nil {
				return errors.Wrapf(err, "could not find next ledger sequence after %d", batchStartSeq)
			}
			if !ok {
				break
			}
			batchStartSeq = next
		} else {
			batchStartSeq += batchSize
		}
		time.Sleep(sleep)
	}

	return nil
}

func (r *System) deleteBatch(ctx context.Context, batchStartSeq, batchEndSeq uint32) (int64, error) {
	batchStart, batchEnd, err := toid.LedgerRangeInclusive(int32(batchStartSeq), int32(batchEndSeq))
	if err != nil {
		return 0, err
	}

	startTime := time.Now()
	err = r.HistoryQ.Begin(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "Error in begin")
	}
	defer r.HistoryQ.Rollback()

	count, err := r.HistoryQ.DeleteRangeAll(ctx, batchStart, batchEnd)
	if err != nil {
		return 0, errors.Wrap(err, "Error in DeleteRangeAll")
	}

	err = r.HistoryQ.Commit()
	if err != nil {
		return 0, errors.Wrap(err, "Error in commit")
	}

	elapsedSeconds := time.Since(startTime).Seconds()
	log.WithField("start_ledger", batchStartSeq).
		WithField("end_ledger", batchEndSeq).
		WithField("rows_deleted", strconv.FormatInt(count, 10)).
		WithField("duration", elapsedSeconds).
		Info("reaper: successfully deleted batch")

	r.rowsDeleted.Observe(float64(count))
	r.deleteBatchDuration.Observe(elapsedSeconds)
	return count, nil
}
