package ingest

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	logpkg "github.com/stellar/go/support/log"
	"github.com/stellar/go/toid"
)

// Reaper represents the history reaping subsystem of horizon.
type Reaper struct {
	historyQ  *history.Q
	reapLockQ *history.Q
	config    ReapConfig
	logger    *logpkg.Entry

	deleteBatchDuration prometheus.Summary
	rowsDeleted         prometheus.Summary

	lock sync.Mutex
}

type ReapConfig struct {
	RetentionCount uint32
	ReapBatchSize  uint32
}

// NewReaper creates a new Reaper instance
func NewReaper(config ReapConfig, dbSession db.SessionInterface) *Reaper {
	r := &Reaper{
		historyQ:  &history.Q{dbSession.Clone()},
		reapLockQ: &history.Q{dbSession.Clone()},
		config:    config,
		deleteBatchDuration: prometheus.NewSummary(prometheus.SummaryOpts{
			Namespace: "horizon", Subsystem: "reap", Name: "delete_batch_duration",
			Help:       "reap batch duration in seconds, sliding window = 10m",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}),
		rowsDeleted: prometheus.NewSummary(prometheus.SummaryOpts{
			Namespace: "horizon", Subsystem: "reap", Name: "rows_deleted",
			Help:       "rows deleted during reap batch , sliding window = 10m",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}),
		logger: log.WithField("subservice", "reaper"),
	}

	return r
}

// DeleteUnretainedHistory removes all data associated with unretained ledgers.
func (r *Reaper) DeleteUnretainedHistory(ctx context.Context) error {
	// RetentionCount of 0 indicates "keep all history"
	if r.config.RetentionCount == 0 {
		return nil
	}

	if !r.lock.TryLock() {
		r.logger.Infof("reap already in progress")
		return nil
	}
	defer r.lock.Unlock()

	if err := r.reapLockQ.Begin(ctx); err != nil {
		return errors.Wrap(err, "error while acquiring reaper lock transaction")
	}
	defer func() {
		if err := r.reapLockQ.Rollback(); err != nil {
			r.logger.WithField("error", err).Error("failed to release reaper lock")
		}
	}()
	if acquired, err := r.reapLockQ.TryReaperLock(ctx); err != nil {
		return errors.Wrap(err, "error while acquiring reaper lock")
	} else if !acquired {
		r.logger.Info("reap already in progress on another node")
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
		r.logger.
			WithField("latest", latest).
			WithField("oldest", oldest).
			WithField("retention_count", r.config.RetentionCount).
			Info("not enough history to reap")
		return nil
	}

	if err = r.clearBefore(ctx, oldest, targetElder); err != nil {
		return err
	}

	r.logger.
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

	r.logger.WithField("start_ledger", startSeq).
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
	r.logger.WithField("start_ledger", batchStartSeq).
		WithField("end_ledger", batchEndSeq).
		WithField("rows_deleted", strconv.FormatInt(count, 10)).
		WithField("duration", elapsedSeconds).
		Info("successfully deleted batch")

	r.rowsDeleted.Observe(float64(count))
	r.deleteBatchDuration.Observe(elapsedSeconds)
	return count, nil
}
