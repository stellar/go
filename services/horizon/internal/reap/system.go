package reap

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	logpkg "github.com/stellar/go/support/log"
	"github.com/stellar/go/toid"
)

var log = logpkg.DefaultLogger.WithField("service", "reaper")

// DeleteUnretainedHistory removes all data associated with unretained ledgers.
func (r *Reaper) DeleteUnretainedHistory(ctx context.Context) error {
	// RetentionCount of 0 indicates "keep all history"
	if r.config.RetentionCount == 0 {
		return nil
	}

	if !r.lock.TryLock() {
		log.Infof("reap already in progress")
		return nil
	}
	defer r.lock.Unlock()

	reapLockQ := &history.Q{r.historyQ.Clone()}
	if err := reapLockQ.Begin(ctx); err != nil {
		return errors.Wrap(err, "error while acquiring reaper lock transaction")
	}
	defer func() {
		if err := reapLockQ.Rollback(); err != nil {
			log.WithField("error", err).Error("failed to release reaper lock")
		}
	}()
	if acquired, err := reapLockQ.TryReaperLock(ctx); err != nil {
		return errors.Wrap(err, "error while acquiring reaper lock")
	} else if !acquired {
		log.Info("reap already in progress on another node")
		return nil
	}

	latest, err := r.historyQ.GetLatestHistoryLedger(ctx)
	if err != nil {
		return errors.Wrap(err, "error fetching latest history ledger")
	}
	var oldest uint32
	err = r.historyQ.ElderLedger(ctx, &oldest)
	if err != nil {
		return errors.Wrap(err, "error fetching elder ledger")
	}

	targetElder := latest - r.config.RetentionCount + 1
	if latest <= r.config.RetentionCount || targetElder < oldest {
		log.
			WithField("latest", latest).
			WithField("oldest", oldest).
			WithField("retention_count", r.config.RetentionCount).
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

// RegisterMetrics registers the prometheus metrics
func (s *Reaper) RegisterMetrics(registry *prometheus.Registry) {
	registry.MustRegister(s.deleteBatchDuration, s.rowsDeleted)
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

func (r *Reaper) clearBefore(ctx context.Context, startSeq, endSeq uint32) error {
	batchSize := r.config.ReapBatchSize
	if batchSize <= 0 {
		return fmt.Errorf("invalid batch size for reaping (%d)", batchSize)
	}

	log.WithField("start_ledger", startSeq).
		WithField("end_ledger", endSeq).
		WithField("batch_size", batchSize).
		Info("deleting history outside retention window")

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
			next, ok, err := r.historyQ.GetNextLedgerSequence(ctx, batchStartSeq)
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

func (r *Reaper) deleteBatch(ctx context.Context, batchStartSeq, batchEndSeq uint32) (int64, error) {
	batchStart, batchEnd, err := toid.LedgerRangeInclusive(int32(batchStartSeq), int32(batchEndSeq))
	if err != nil {
		return 0, err
	}

	startTime := time.Now()
	err = r.historyQ.Begin(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "Error in begin")
	}
	defer r.historyQ.Rollback()

	count, err := r.historyQ.DeleteRangeAll(ctx, batchStart, batchEnd)
	if err != nil {
		return 0, errors.Wrap(err, "Error in DeleteRangeAll")
	}

	err = r.historyQ.Commit()
	if err != nil {
		return 0, errors.Wrap(err, "Error in commit")
	}

	elapsedSeconds := time.Since(startTime).Seconds()
	log.WithField("start_ledger", batchStartSeq).
		WithField("end_ledger", batchEndSeq).
		WithField("rows_deleted", strconv.FormatInt(count, 10)).
		WithField("duration", elapsedSeconds).
		Info("successfully deleted batch")

	r.rowsDeleted.Observe(float64(count))
	r.deleteBatchDuration.Observe(elapsedSeconds)
	return count, nil
}
